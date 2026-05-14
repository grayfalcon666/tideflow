import asyncio
import json
import logging
import os
import signal
import time

import pika

import config
import db as database
import dict_lookup
import nlp
import subtitle as subtitle_mod

logger = logging.getLogger(__name__)


class SubtitleWorker:
    def __init__(self):
        self.semaphore = asyncio.Semaphore(config.SUBTITLE_CONCURRENCY)
        self.connection = None
        self.channel = None
        self.running = False

    def connect_rabbitmq(self):
        """Establish connection to RabbitMQ and declare queue."""
        params = pika.URLParameters(config.RABBITMQ_URL)
        params.heartbeat = 60
        params.blocked_connection_timeout = 30

        self.connection = pika.BlockingConnection(params)
        self.channel = self.connection.channel()

        # Ensure the exchange and queue exist
        self.channel.exchange_declare(
            exchange="video.events",
            exchange_type="topic",
            durable=True,
        )
        self.channel.queue_declare(
            queue="subtitle.gen",
            durable=True,
            arguments={"x-dead-letter-exchange": "dlx.events"},
        )
        self.channel.queue_bind(
            queue="subtitle.gen",
            exchange="video.events",
            routing_key="video.publish",
        )

        # Set prefetch count for concurrency control
        self.channel.basic_qos(prefetch_count=config.SUBTITLE_CONCURRENCY)

        logger.info("Connected to RabbitMQ, consuming queue: subtitle.gen")

    def process_message(self, channel, method, properties, body):
        """Process a single message from the queue."""
        try:
            event = json.loads(body)
            video_id = event.get("video_id")
            author_id = event.get("author_id")
            create_time = event.get("create_time")
            logger.info("Processing video_id=%d author_id=%d", video_id, author_id)

            if not video_id:
                logger.warning("Event missing video_id, discarding")
                channel.basic_ack(delivery_tag=method.delivery_tag)
                return

            # Get retry count from x-death header (in properties, not method)
            retry_count = 0
            if properties and properties.headers and "x-death" in properties.headers:
                death_info = properties.headers["x-death"]
                if death_info:
                    retry_count = death_info[0].get("count", 0)

            success = self.process_video(video_id, author_id, retry_count)

            if success:
                channel.basic_ack(delivery_tag=method.delivery_tag)
            else:
                if retry_count >= config.MAX_RETRY_COUNT:
                    logger.error(
                        "Max retries (%d) exceeded for video_id=%d, discarding",
                        config.MAX_RETRY_COUNT, video_id,
                    )
                    channel.basic_ack(delivery_tag=method.delivery_tag)
                else:
                    logger.warning(
                        "Processing failed for video_id=%d, requeuing (retry %d/%d)",
                        video_id, retry_count + 1, config.MAX_RETRY_COUNT,
                    )
                    channel.basic_nack(
                        delivery_tag=method.delivery_tag, requeue=True,
                    )
        except Exception as e:
            logger.error("Failed to process message: %s", e, exc_info=True)
            channel.basic_nack(
                delivery_tag=method.delivery_tag, requeue=True,
            )

    def process_video(self, video_id, author_id, retry_count=0):
        """Execute the full subtitle extraction + wordbank generation pipeline."""
        conn = database.get_mysql_connection()
        r = database.get_redis_client()
        cursor = conn.cursor()

        try:
            # 1. Get video info
            video = database.get_video_info(cursor, video_id)
            if not video:
                logger.error("Video not found: id=%d", video_id)
                return True  # ACK — can't process non-existent video

            play_url = video.get("play_url", "")
            if not play_url:
                logger.warning("Video has no play_url: id=%d", video_id)
                return True  # ACK — nothing to process

            # Normalize path: /videos/xxx.mp4 -> /app/uploads/videos/xxx.mp4
            video_path = os.path.join(config.UPLOAD_DIR, play_url.lstrip("/"))

            if not os.path.exists(video_path):
                logger.error("Video file not found: %s", video_path)
                return True  # ACK — file missing, can't process

            # 2. Update status to processing
            database.update_video_status(cursor, video_id, "processing", "processing")
            conn.commit()

            # 3. Check for embedded subtitles
            has_embedded = subtitle_mod.check_embedded_subtitles(video_path)
            logger.info("Video %d: embedded subtitles = %s", video_id, has_embedded)

            if has_embedded:
                subtitles = subtitle_mod.extract_embedded_subtitles(video_path)
                source = "embedded"
                logger.info("Video %d: extracted %d embedded subtitle entries", video_id, len(subtitles))
            else:
                # Extract audio and transcribe
                wav_path = None
                try:
                    wav_path = subtitle_mod.extract_audio(video_path)
                    subtitles = subtitle_mod.transcribe_audio(
                        wav_path,
                        model_name=config.WHISPER_MODEL,
                        device=config.WHISPER_DEVICE,
                        compute_type=config.WHISPER_COMPUTE_TYPE,
                    )
                    source = "transcribed"
                finally:
                    if wav_path and os.path.exists(wav_path):
                        os.unlink(wav_path)

            # 4. If no speech detected, mark as failed
            if not subtitles:
                logger.warning("Video %d: no speech detected, marking as failed", video_id)
                database.update_video_status(cursor, video_id, "failed", "failed")
                conn.commit()
                _notify_author(cursor, video_id, author_id, "failed")
                conn.commit()
                return True  # ACK — clean failure, no retry

            # 5. Write video_subtitles
            subtitles_json = json.dumps(subtitles, ensure_ascii=False)
            database.upsert_video_subtitle(cursor, video_id, subtitles_json, "json", source)

            # 6. NLP pipeline
            word_list = nlp.build_word_list(subtitles)
            logger.info("Video %d: NLP extracted %d unique lemmas", video_id, len(word_list))

            # 7. Associate captions
            wordbank = dict_lookup.associate_captions(word_list, subtitles, max_per_word=3)

            # 8. Enrich with dictionary definitions
            wordbank = dict_lookup.enrich_wordbank(wordbank)

            # 9. Write video_wordbank
            words_json = json.dumps(wordbank, ensure_ascii=False)
            database.upsert_video_wordbank(cursor, video_id, words_json, len(wordbank))

            # 10. Update status to ready
            database.update_video_status(cursor, video_id, "ready", "ready")
            conn.commit()

            # 11. Delete Redis cache
            database.delete_redis_keys(r, video_id)

            # 12. Notify author
            video_title = video.get("title", "your video")
            _notify_author(cursor, video_id, author_id, "ready", video_title)
            conn.commit()

            logger.info("Video %d: wordbank ready, %d words", video_id, len(wordbank))
            return True

        except Exception as e:
            logger.error("Error processing video %d: %s", video_id, e, exc_info=True)
            conn.rollback()

            # Mark as failed on error
            try:
                database.update_video_status(cursor, video_id, "failed", "failed")
                database.delete_redis_keys(r, video_id)
                conn.commit()
            except Exception:
                pass

            return False  # NACK — will retry
        finally:
            cursor.close()
            conn.close()
            r.close()

    def run(self):
        """Main loop: consume messages from subtitle.gen queue with auto-reconnect."""
        self.running = True

        while self.running:
            try:
                self.connect_rabbitmq()

                for method, properties, body in self.channel.consume(
                    queue="subtitle.gen", auto_ack=False, inactivity_timeout=5,
                ):
                    if not self.running:
                        break

                    if method is None:
                        continue

                    # Process synchronously (pika BlockingConnection is not thread-safe)
                    self.process_message(self.channel, method, properties, body)

            except Exception as e:
                if not self.running:
                    break
                logger.error("Connection error: %s, reconnecting in 5s...", e)
                try:
                    if self.connection and self.connection.is_open:
                        self.connection.close()
                except Exception:
                    pass
                time.sleep(5)

        logger.info("SubtitleWorker run loop ended")

    def stop(self):
        """Gracefully stop the worker."""
        logger.info("Shutting down SubtitleWorker...")
        self.running = False
        if self.connection and self.connection.is_open:
            self.connection.close()
        logger.info("SubtitleWorker stopped")


def _notify_author(cursor, video_id, author_id, status, video_title=""):
    """Create a notification for the video author about wordbank status."""
    if not author_id:
        return
    if status == "ready":
        content = f'Your video "{video_title}" now has an English wordbank ready for learning'
    else:
        content = "Your video's English wordbank generation failed (no speech detected)"
    database.create_notification(
        cursor, author_id, author_id, "system", content, video_id,
    )
