package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Lock struct {
	rdb *redis.Client
}

func NewLock(rdb *redis.Client) *Lock {
	return &Lock{rdb: rdb}
}

func (l *Lock) Acquire(ctx context.Context, target string, token string, ttl time.Duration) (bool, error) {
	key := LockKey(target)
	ok, err := l.rdb.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	lua := `
if redis.call('GET', KEYS[1]) == ARGV[1] then
    return redis.call('DEL', KEYS[1])
else
    return 0
end
`
	_, err = l.rdb.Eval(ctx, lua, []string{key}, token).Result()
	return ok, err
}

func (l *Lock) Release(ctx context.Context, target string, token string) error {
	key := LockKey(target)
	lua := `
if redis.call('GET', KEYS[1]) == ARGV[1] then
    return redis.call('DEL', KEYS[1])
else
    return 0
end
`
	_, err := l.rdb.Eval(ctx, lua, []string{key}, token).Result()
	return err
}