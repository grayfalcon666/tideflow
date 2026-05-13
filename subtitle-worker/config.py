import os

from dotenv import load_dotenv

# Load .env from project root (override=False ensures Docker env vars take precedence)
load_dotenv(dotenv_path=os.path.join(os.path.dirname(__file__), "..", ".env"), override=False)

# RabbitMQ
RABBITMQ_URL = os.getenv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")

# MySQL (pymysql uses separate host/port/user/pass)
MYSQL_HOST = os.getenv("MYSQL_HOST", "localhost")
MYSQL_PORT = int(os.getenv("MYSQL_PORT", "3306"))
MYSQL_USER = os.getenv("MYSQL_USER", "root")
MYSQL_PASSWORD = os.getenv("MYSQL_PASSWORD", "password")
MYSQL_DATABASE = os.getenv("MYSQL_DATABASE", "tideflow")

# Redis — Go uses combined "host:port"; Python splits them
_raw_redis_host = os.getenv("REDIS_HOST", "localhost:6379")
if ":" in _raw_redis_host:
    REDIS_HOST, _port = _raw_redis_host.rsplit(":", 1)
    REDIS_PORT = int(os.getenv("REDIS_PORT", _port))
else:
    REDIS_HOST = _raw_redis_host
    REDIS_PORT = int(os.getenv("REDIS_PORT", "6379"))
REDIS_PASSWORD = os.getenv("REDIS_PASSWORD", "")
REDIS_DB = int(os.getenv("REDIS_DB", "0"))

# Whisper
WHISPER_MODEL = os.getenv("WHISPER_MODEL", "small")
WHISPER_DEVICE = os.getenv("WHISPER_DEVICE", "cpu")
WHISPER_COMPUTE_TYPE = os.getenv("WHISPER_COMPUTE_TYPE", "int8")

# Worker
SUBTITLE_CONCURRENCY = int(os.getenv("SUBTITLE_CONCURRENCY", "1"))
UPLOAD_DIR = os.getenv("UPLOAD_DIR", "./uploads")
ECDICT_PATH = os.getenv("ECDICT_PATH", "./ecdict.db")
PROCESSING_TIMEOUT = int(os.getenv("PROCESSING_TIMEOUT", "600"))
MAX_RETRY_COUNT = int(os.getenv("MAX_RETRY_COUNT", "3"))
