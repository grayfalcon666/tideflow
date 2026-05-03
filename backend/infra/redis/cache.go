package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"

	"tideflow/internal/models"
)

type Cache struct {
	local *cache.Cache
	rdb   *redis.Client
	sf    *singleflight.Group
}

func NewCache(rdb *redis.Client) *Cache {
	return &Cache{
		local: cache.New(5*time.Second, 10*time.Second),
		rdb:   rdb,
		sf:    &singleflight.Group{},
	}
}

func (c *Cache) GetVideoEntity(ctx context.Context, id uint) (*models.Video, error) {
	key := VideoEntity(id)

	if v, ok := c.local.Get(key); ok {
		if video, ok := v.(*models.Video); ok {
			return video, nil
		}
	}

	v, err := c.rdb.Get(ctx, key).Result()
	if err == nil {
		var video models.Video
		if json.Unmarshal([]byte(v), &video) == nil {
			c.local.Set(key, &video, time.Hour)
			go func() {
				c.rdb.Set(context.Background(), key, v, time.Hour)
			}()
			return &video, nil
		}
	}

	return nil, err
}

func (c *Cache) GetVideoByIDs(ctx context.Context, ids []uint) ([]*models.Video, error) {
	key := VideoEntity(0)
	results := make([]*models.Video, len(ids))
	var missed []int

	for i, id := range ids {
		if v, ok := c.local.Get(fmt.Sprintf("%s:%d", key, id)); ok {
			if video, ok := v.(*models.Video); ok {
				results[i] = video
				continue
			}
		}
		missed = append(missed, i)
	}

	if len(missed) > 0 {
		missedIDs := make([]uint, len(missed))
		for i, idx := range missed {
			missedIDs[i] = ids[idx]
		}
		vals, err := c.rdb.MGet(ctx, makeRedisKeys(missedIDs, VideoEntity)...).Result()
		if err != nil {
			return nil, err
		}
		for i, v := range vals {
			if v != nil {
				var video models.Video
				if json.Unmarshal([]byte(v.(string)), &video) == nil {
					results[missed[i]] = &video
					c.local.Set(fmt.Sprintf("%s:%d", key, missedIDs[i]), &video, time.Hour)
				}
			}
		}
	}

	return results, nil
}

func makeRedisKeys(ids []uint, fn func(uint) string) []string {
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = fn(id)
	}
	return keys
}

func (c *Cache) SetVideoEntity(ctx context.Context, video *models.Video) error {
	data, err := json.Marshal(video)
	if err != nil {
		return err
	}
	key := VideoEntity(video.ID)
	c.local.Set(key, video, time.Hour)
	return c.rdb.Set(ctx, key, data, time.Hour).Err()
}

func (c *Cache) InvalidateVideo(id uint) {
	key := VideoEntity(id)
	c.local.Delete(key)
}