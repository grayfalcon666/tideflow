package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Create a temp .env file
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	envContent := `
DSN=user:pass@tcp(localhost:3306)/testdb?charset=utf8mb4
REDIS_HOST=127.0.0.1:6379
REDIS_PASSWORD=secret
REDIS_DB=1
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
JWT_SECRET=test-secret
JWT_ACCESS_EXPIRY=12h
JWT_REFRESH_EXPIRY=72h
SERVER_PORT=9090
UPLOAD_DIR=/tmp/uploads
BIG_V_THRESHOLD=5000
LOGIN_RATE_LIMIT=20
REGISTER_RATE_LIMIT=10
`
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("failed to write temp env: %v", err)
	}

	cfg, err := Load(envPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// DB
	if cfg.DB.DSN != "user:pass@tcp(localhost:3306)/testdb?charset=utf8mb4" {
		t.Errorf("DSN = %q, want correct DSN", cfg.DB.DSN)
	}

	// Redis
	if cfg.Redis.Host != "127.0.0.1:6379" {
		t.Errorf("Redis.Host = %q, want 127.0.0.1:6379", cfg.Redis.Host)
	}
	if cfg.Redis.Password != "secret" {
		t.Errorf("Redis.Password = %q, want secret", cfg.Redis.Password)
	}
	if cfg.Redis.DB != 1 {
		t.Errorf("Redis.DB = %d, want 1", cfg.Redis.DB)
	}

	// RabbitMQ
	if cfg.RabbitMQ.URL != "amqp://guest:guest@localhost:5672/" {
		t.Errorf("RabbitMQ.URL = %q", cfg.RabbitMQ.URL)
	}

	// JWT
	if cfg.JWT.Secret != "test-secret" {
		t.Errorf("JWT.Secret = %q, want test-secret", cfg.JWT.Secret)
	}
	if cfg.JWT.AccessExpiry != 12*time.Hour {
		t.Errorf("JWT.AccessExpiry = %v, want 12h", cfg.JWT.AccessExpiry)
	}
	if cfg.JWT.RefreshExpiry != 72*time.Hour {
		t.Errorf("JWT.RefreshExpiry = %v, want 72h", cfg.JWT.RefreshExpiry)
	}

	// Server
	if cfg.Server.Port != "9090" {
		t.Errorf("Server.Port = %q, want 9090", cfg.Server.Port)
	}

	// Upload
	if cfg.Upload.Dir != "/tmp/uploads" {
		t.Errorf("Upload.Dir = %q, want /tmp/uploads", cfg.Upload.Dir)
	}

	// BigV
	if cfg.BigVThreshold != 5000 {
		t.Errorf("BigVThreshold = %d, want 5000", cfg.BigVThreshold)
	}

	// Rate limits
	if cfg.RateLimits.LoginRateLimit != 20 {
		t.Errorf("LoginRateLimit = %d, want 20", cfg.RateLimits.LoginRateLimit)
	}
	if cfg.RateLimits.LikeRateLimit != 30 {
		t.Errorf("LikeRateLimit = %d, want 30", cfg.RateLimits.LikeRateLimit)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	// Missing file should fail
	_, err := Load("/nonexistent/path/.env")
	if err == nil {
		t.Error("Load() should fail for nonexistent file")
	}
}

func TestLoadConfigZeroExpiry(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	// Empty expiry should default
	envContent := `
DSN=test:pass@tcp(localhost)/test
REDIS_HOST=127.0.0.1:6379
JWT_SECRET=secret
JWT_ACCESS_EXPIRY=
JWT_REFRESH_EXPIRY=
SERVER_PORT=8080
`
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("failed to write temp env: %v", err)
	}

	cfg, err := Load(envPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	// Should fall back to defaults
	if cfg.JWT.AccessExpiry != 24*time.Hour {
		t.Errorf("AccessExpiry = %v, want 24h fallback", cfg.JWT.AccessExpiry)
	}
	if cfg.JWT.RefreshExpiry != 168*time.Hour {
		t.Errorf("RefreshExpiry = %v, want 168h fallback", cfg.JWT.RefreshExpiry)
	}
}
