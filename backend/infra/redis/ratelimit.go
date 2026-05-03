package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	rdb *redis.Client
}

func NewRateLimiter(rdb *redis.Client) *RateLimiter {
	return &RateLimiter{rdb: rdb}
}

type limitResult struct {
	allowed bool
	current int64
}

func (rl *RateLimiter) Allow(ctx context.Context, action, subject string, limit int, window time.Duration) (bool, int64, error) {
	key := RateLimit(action, subject)

	lua := `
local current = redis.call('INCR', KEYS[1])
if current == 1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return current
`
	ttl := int(window.Seconds())
	current, err := rl.rdb.Eval(ctx, lua, []string{key}, ttl).Int64()
	if err != nil {
		return false, 0, err
	}
	return current <= int64(limit), current, nil
}