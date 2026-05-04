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
	"tideflow/internal/repository"
)

const (
	// L1 本地缓存 TTL，设计文档要求 5s
	L1TTL = 5 * time.Second
	// L2 Redis 缓存 TTL，视频实体 1h
	L2TTL = time.Hour
	// 软标记 TTL，防止击穿标记的过期时间
	sfLabelTTL = 30 * time.Second
)

type Cache struct {
	local *cache.Cache
	rdb   *redis.Client
	sf    *singleflight.Group
	repo  *repository.Repository
}

func NewCache(rdb *redis.Client) *Cache {
	return &Cache{
		local: cache.New(L1TTL, 2*L1TTL),
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
			c.local.Set(key, &video, L1TTL)
			return &video, nil
		}
	}

	return nil, err
}

func (c *Cache) GetVideoByIDs(ctx context.Context, ids []uint) ([]*models.Video, error) {
	if len(ids) == 0 {
		return []*models.Video{}, nil
	}

	key := VideoEntity(0)
	results := make([]*models.Video, len(ids))
	var missed []int

	// L1 查询
	for i, id := range ids {
		if v, ok := c.local.Get(fmt.Sprintf("%s:%d", key, id)); ok {
			if video, ok := v.(*models.Video); ok {
				results[i] = video
				continue
			}
		}
		missed = append(missed, i)
	}

	if len(missed) == 0 {
		return results, nil
	}

	missedIDs := make([]uint, len(missed))
	for i, idx := range missed {
		missedIDs[i] = ids[idx]
	}

	// L2 MGet，带 singleflight 防击穿
	sfKey := fmt.Sprintf("videos:%v", missedIDs)
	ret, _, _ := c.sf.Do(sfKey, func() (interface{}, error) {
		// 写入 Redis 软标记，防止击穿
		sfLabelKey := SFLabel(fmt.Sprintf("entity:%v", missedIDs))
		c.rdb.SetNX(ctx, sfLabelKey, "1", sfLabelTTL)
		// L2 MGet
		vals, err := c.rdb.MGet(ctx, makeRedisKeys(missedIDs, VideoEntity)...).Result()
		if err != nil {
			return nil, err
		}
		// L2 命中的直接写 L1 本地缓存
		key := VideoEntity(0)
		for i, v := range vals {
			if v == nil {
				continue
			}
			var video models.Video
			if json.Unmarshal([]byte(v.(string)), &video) == nil {
				c.local.Set(fmt.Sprintf("%s:%d", key, missedIDs[i]), &video, L1TTL)
			}
		}
		return vals, nil
	})
	// 单飞结束后清理软标记
	defer func() {
		sfLabelKey := SFLabel(fmt.Sprintf("entity:%v", missedIDs))
		c.rdb.Del(context.Background(), sfLabelKey)
	}()

	vals, ok := ret.([]interface{})
	if !ok {
		return c.loadFromDB(ctx, missedIDs, results, missed)
	}
	for i, v := range vals {
		if v != nil {
			var video models.Video
			if json.Unmarshal([]byte(v.(string)), &video) == nil {
				results[missed[i]] = &video
			}
		}
	}

	// 处理 L2 未命中：从 DB 回填
	return c.loadFromDB(ctx, missedIDs, results, missed)
}

// loadFromDB 从 MySQL 回填未命中缓存
func (c *Cache) loadFromDB(ctx context.Context, missedIDs []uint, results []*models.Video, missed []int) ([]*models.Video, error) {
	// 找出仍然为 nil 的位置
	var stillMissedIdx []int
	var stillMissedIDs []uint
	for i, idx := range missed {
		if results[idx] == nil {
			stillMissedIdx = append(stillMissedIdx, idx)
			stillMissedIDs = append(stillMissedIDs, missedIDs[i])
		}
	}

	if len(stillMissedIDs) == 0 {
		return results, nil
	}

	// 聚合查询 DB（通过 Repository）
	dbVideos, err := c.repo.GetVideosByIDs(ctx, stillMissedIDs)
	if err != nil {
		return results, err
	}

	// 建立 id -> video 映射
	videoMap := make(map[uint]*models.Video)
	for _, v := range dbVideos {
		videoMap[v.ID] = v
	}

	key := VideoEntity(0)
	for i, idx := range stillMissedIdx {
		id := stillMissedIDs[i]
		if video, ok := videoMap[id]; ok {
			results[idx] = video
			// 回写 L2 Redis
			data, _ := json.Marshal(video)
			c.rdb.Set(ctx, VideoEntity(id), data, L2TTL)
			c.local.Set(fmt.Sprintf("%s:%d", key, id), video, L1TTL)
		}
	}

	// 清理软标记
	sfLabelKey := SFLabel(fmt.Sprintf("entity:%v", stillMissedIDs))
	c.rdb.Del(ctx, sfLabelKey)

	return results, nil
}

// SetRepo 设置 repository 引用（用于 DB 降级回填）
func (c *Cache) SetRepo(repo *repository.Repository) {
	c.repo = repo
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
	c.local.Set(key, video, L1TTL)
	return c.rdb.Set(ctx, key, data, L2TTL).Err()
}

func (c *Cache) InvalidateVideo(id uint) {
	key := VideoEntity(id)
	c.local.Delete(key)
}