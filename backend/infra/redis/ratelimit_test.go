package redis

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// TestAllowLogic tests the rate limit counter behavior without a real Redis.
func TestRateLimitKey(t *testing.T) {
	// Test key format
	key := RateLimit("login", "192.168.1.1")
	if key != "v1:ratelimit:login:192.168.1.1" {
		t.Errorf("RateLimit key = %q, want v1:ratelimit:login:192.168.1.1", key)
	}

	key2 := RateLimit("like_write", "100")
	if key2 != "v1:ratelimit:like_write:100" {
		t.Errorf("RateLimit key = %q, want v1:ratelimit:like_write:100", key2)
	}
}

// Integration test: requires real Redis via REDIS_ADDR env or docker.
func TestRateLimiterAllow(t *testing.T) {
	addr := getRedisAddr()
	if addr == "" {
		t.Skip("Redis not available, skipping integration test")
	}

	rdb := redis.NewClient(&redis.Options{Addr: addr})
	defer rdb.Close()

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not reachable at %s: %v", addr, err)
	}

	// Clean up before test
	rdb.Del(ctx, "v1:ratelimit:test:u1")

	rl := NewRateLimiter(rdb)

	// Allow up to limit
	for i := 1; i <= 5; i++ {
		allowed, current, err := rl.Allow(ctx, "test", "u1", 5, time.Second)
		if err != nil {
			t.Fatalf("Allow attempt %d: %v", i, err)
		}
		if !allowed {
			t.Errorf("attempt %d: should be allowed (current=%d, limit=5)", i, current)
		}
		if current != int64(i) {
			t.Errorf("attempt %d: current=%d, want %d", i, current, i)
		}
	}

	// 6th should be blocked
	allowed, current, err := rl.Allow(ctx, "test", "u1", 5, time.Second)
	if err != nil {
		t.Fatalf("Allow 6th: %v", err)
	}
	if allowed {
		t.Errorf("6th attempt: should be blocked (current=%d)", current)
	}

	// Cleanup
	rdb.Del(ctx, "v1:ratelimit:test:u1")
}

func getRedisAddr() string {
	return "127.0.0.1:6379"
}
