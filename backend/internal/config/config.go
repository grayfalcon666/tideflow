package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	DB            DBConfig
	Redis         RedisConfig
	RabbitMQ      RabbitMQConfig
	Elasticsearch ElasticsearchConfig
	JWT           JWTConfig
	Server         ServerConfig
	Upload         UploadConfig
	BigVThreshold  int
	VocabListsDir  string
	RateLimits     RateLimitsConfig
}

type DBConfig struct {
	DSN string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type RabbitMQConfig struct {
	URL string
}

type ElasticsearchConfig struct {
	Addresses []string
	Username  string
	Password  string
}

type JWTConfig struct {
	Secret         string
	AccessExpiry   time.Duration
	RefreshExpiry  time.Duration
}

type ServerConfig struct {
	Port string
}

type UploadConfig struct {
	Dir string
}

type RateLimitsConfig struct {
	LoginRateLimit    int
	RegisterRateLimit int
	LikeRateLimit     int
	CommentRateLimit  int
	SocialRateLimit   int
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("env")

	v.SetDefault("DSN", "root:password@tcp(localhost:3306)/tideflow?charset=utf8mb4&parseTime=True&loc=Local")
	v.SetDefault("REDIS_HOST", "localhost:6379")
	v.SetDefault("REDIS_PASSWORD", "")
	v.SetDefault("REDIS_DB", 0)
	v.SetDefault("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	v.SetDefault("JWT_SECRET", "your-secret-key-change-in-production")
	v.SetDefault("JWT_ACCESS_EXPIRY", "24h")
	v.SetDefault("JWT_REFRESH_EXPIRY", "168h")
	v.SetDefault("SERVER_PORT", "8080")
	v.SetDefault("UPLOAD_DIR", "./uploads")
	v.SetDefault("BIG_V_THRESHOLD", 10000)
	v.SetDefault("VOCAB_LISTS_DIR", "../resources/vocab_lists")
	v.SetDefault("ES_ADDRESSES", "http://localhost:9200")
	v.SetDefault("ES_USERNAME", "")
	v.SetDefault("ES_PASSWORD", "")
	v.SetDefault("LOGIN_RATE_LIMIT", 10)
	v.SetDefault("REGISTER_RATE_LIMIT", 5)
	v.SetDefault("LIKE_RATE_LIMIT", 30)
	v.SetDefault("COMMENT_RATE_LIMIT", 10)
	v.SetDefault("SOCIAL_RATE_LIMIT", 20)

	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := &Config{}

	cfg.DB.DSN = v.GetString("DSN")
	cfg.Redis.Host = v.GetString("REDIS_HOST")
	cfg.Redis.Port = v.GetInt("REDIS_PORT")
	if cfg.Redis.Port == 0 {
		cfg.Redis.Port = 6379
	}
	cfg.Redis.Password = v.GetString("REDIS_PASSWORD")
	cfg.Redis.DB = v.GetInt("REDIS_DB")
	cfg.RabbitMQ.URL = v.GetString("RABBITMQ_URL")

	esAddresses := v.GetString("ES_ADDRESSES")
	if esAddresses == "" {
		cfg.Elasticsearch.Addresses = []string{"http://localhost:9200"}
	} else {
		cfg.Elasticsearch.Addresses = []string{esAddresses}
	}
	cfg.Elasticsearch.Username = v.GetString("ES_USERNAME")
	cfg.Elasticsearch.Password = v.GetString("ES_PASSWORD")

	cfg.JWT.Secret = v.GetString("JWT_SECRET")
	cfg.Server.Port = v.GetString("SERVER_PORT")
	cfg.Upload.Dir = v.GetString("UPLOAD_DIR")
	cfg.BigVThreshold = v.GetInt("BIG_V_THRESHOLD")
	cfg.VocabListsDir = v.GetString("VOCAB_LISTS_DIR")

	cfg.JWT.AccessExpiry, _ = time.ParseDuration(v.GetString("JWT_ACCESS_EXPIRY"))
	if cfg.JWT.AccessExpiry == 0 {
		cfg.JWT.AccessExpiry = 24 * time.Hour
	}

	cfg.JWT.RefreshExpiry, _ = time.ParseDuration(v.GetString("JWT_REFRESH_EXPIRY"))
	if cfg.JWT.RefreshExpiry == 0 {
		cfg.JWT.RefreshExpiry = 168 * time.Hour
	}

	cfg.RateLimits.LoginRateLimit = v.GetInt("LOGIN_RATE_LIMIT")
	cfg.RateLimits.RegisterRateLimit = v.GetInt("REGISTER_RATE_LIMIT")
	cfg.RateLimits.LikeRateLimit = v.GetInt("LIKE_RATE_LIMIT")
	cfg.RateLimits.CommentRateLimit = v.GetInt("COMMENT_RATE_LIMIT")
	cfg.RateLimits.SocialRateLimit = v.GetInt("SOCIAL_RATE_LIMIT")

	return cfg, nil
}