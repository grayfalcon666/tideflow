#!/usr/bin/env python3
"""
TideFlow SubtitleWorker — Main entry point.
Consumes video.publish events from RabbitMQ and generates:
  1. English subtitles (extracted or transcribed via faster-whisper)
  2. Video wordbank (NLP tokenized + dictionary enriched vocabulary)

Usage:
    python main.py
"""

import logging
import os
import signal
import sys

# Add current directory to path for imports
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import config
import dict_lookup
from worker import SubtitleWorker

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
logger = logging.getLogger("main")


def main():
    logger.info("Starting TideFlow SubtitleWorker")
    logger.info("Model: %s, Device: %s, Compute: %s, Concurrency: %d",
                config.WHISPER_MODEL, config.WHISPER_DEVICE,
                config.WHISPER_COMPUTE_TYPE, config.SUBTITLE_CONCURRENCY)

    # Pre-load dictionary at startup
    if os.path.exists(config.ECDICT_PATH):
        logger.info("Loading ECDICT from %s", config.ECDICT_PATH)
        dict_lookup.load_ecdict(config.ECDICT_PATH)
    else:
        logger.warning("ECDICT not found at %s, word enrichment disabled", config.ECDICT_PATH)

    worker = SubtitleWorker()

    # Handle graceful shutdown
    def shutdown(signum, frame):
        logger.info("Received signal %d, shutting down...", signum)
        worker.stop()

    signal.signal(signal.SIGINT, shutdown)
    signal.signal(signal.SIGTERM, shutdown)

    try:
        worker.run()
    except KeyboardInterrupt:
        worker.stop()
    except Exception as e:
        logger.error("Fatal error: %s", e, exc_info=True)
        sys.exit(1)

    logger.info("SubtitleWorker exited")


if __name__ == "__main__":
    main()
