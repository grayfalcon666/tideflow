import pymysql
import redis

import config


def get_mysql_connection():
    """Create a new MySQL connection."""
    return pymysql.connect(
        host=config.MYSQL_HOST,
        port=config.MYSQL_PORT,
        user=config.MYSQL_USER,
        password=config.MYSQL_PASSWORD,
        database=config.MYSQL_DATABASE,
        charset="utf8mb4",
        cursorclass=pymysql.cursors.DictCursor,
    )


def get_redis_client():
    """Create a Redis client."""
    return redis.Redis(
        host=config.REDIS_HOST,
        port=config.REDIS_PORT,
        password=config.REDIS_PASSWORD or None,
        db=config.REDIS_DB,
        decode_responses=True,
    )


def update_video_status(cursor, video_id, subtitle_status, wordbank_status):
    """Update video subtitle and wordbank processing status."""
    cursor.execute(
        "UPDATE videos SET subtitle_status = %s, wordbank_status = %s WHERE id = %s",
        (subtitle_status, wordbank_status, video_id),
    )


def upsert_video_subtitle(cursor, video_id, subtitles_json, fmt, source):
    """Insert or update video subtitle record."""
    cursor.execute(
        """INSERT INTO video_subtitles (video_id, subtitles, format, source, version, created_at, updated_at)
           VALUES (%s, %s, %s, %s, 1, NOW(), NOW())
           ON DUPLICATE KEY UPDATE
             subtitles = VALUES(subtitles),
             format = VALUES(format),
             source = VALUES(source),
             version = version + 1,
             updated_at = NOW()""",
        (video_id, subtitles_json, fmt, source),
    )


def upsert_video_wordbank(cursor, video_id, words_json, size):
    """Insert or update video wordbank record."""
    cursor.execute(
        """INSERT INTO video_wordbank (video_id, words, size, version, created_at, updated_at)
           VALUES (%s, %s, %s, 1, NOW(), NOW())
           ON DUPLICATE KEY UPDATE
             words = VALUES(words),
             size = VALUES(size),
             version = version + 1,
             updated_at = NOW()""",
        (video_id, words_json, size),
    )


def create_notification(cursor, recipient_id, sender_id, ntype, content, target_id):
    """Create a notification for the video author."""
    cursor.execute(
        "INSERT INTO notifications (recipient_id, sender_id, type, target_id, content, is_read, created_at) "
        "VALUES (%s, %s, %s, %s, %s, false, NOW())",
        (recipient_id, sender_id, ntype, target_id, content),
    )


def get_video_info(cursor, video_id):
    """Get video author_id and play_url."""
    cursor.execute(
        "SELECT author_id, play_url, title FROM videos WHERE id = %s",
        (video_id,),
    )
    return cursor.fetchone()


def delete_redis_keys(r, video_id):
    """Delete subtitle and wordbank related Redis cache keys."""
    keys = [
        f"v1:video:entity:{video_id}",
        f"v1:video:subtitle:{video_id}",
        f"v1:video:wordbank:{video_id}",
    ]
    for key in keys:
        r.delete(key)
