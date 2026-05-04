package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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
	// 视频详情缓存 TTL，设计文档要求 5min
	detailTTL = 5 * time.Minute
	// 详情分布式锁 TTL
	detailLockTTL = 10 * time.Second
	// 关注流冷拉取缓存 TTL，设计文档要求 24h
	followCacheTTL = 24 * time.Hour
)

type Cache struct {
	local *cache.Cache
	rdb   *redis.Client
	sf    *singleflight.Group
	repo  *repository.Repository
	lock  *Lock
}

func NewCache(rdb *redis.Client) *Cache {
	return &Cache{
		local: cache.New(L1TTL, 2*L1TTL),
		rdb:   rdb,
		sf:    &singleflight.Group{},
	}
}

// SetLock sets the distributed lock (required for GetVideoDetail).
func (c *Cache) SetLock(lock *Lock) {
	c.lock = lock
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

// GetVideoDetail returns cached video detail. On cache miss, acquires a distributed
// lock and double-checks before falling back to the provided DB query function.
func (c *Cache) GetVideoDetail(ctx context.Context, id uint, dbQuery func(context.Context, uint) (*models.Video, error)) (*models.Video, error) {
	key := VideoDetail(id)

	// L1 本地缓存
	if v, ok := c.local.Get(key); ok {
		if video, ok := v.(*models.Video); ok {
			return video, nil
		}
	}

	// L2 Redis
	v, err := c.rdb.Get(ctx, key).Result()
	if err == nil {
		var video models.Video
		if json.Unmarshal([]byte(v), &video) == nil {
			c.local.Set(key, &video, L1TTL)
			return &video, nil
		}
	}

	// Cache miss — try to acquire lock to rebuild
	if c.lock == nil {
		return dbQuery(ctx, id)
	}

	token := fmt.Sprintf("%d", time.Now().UnixNano())
	acquired, err := c.lock.AcquireDetail(ctx, id, token, detailLockTTL)
	if err != nil {
		return dbQuery(ctx, id)
	}

	if !acquired {
		// Another goroutine is rebuilding; wait and re-check cache
		for i := 0; i < 10; i++ {
			time.Sleep(100 * time.Millisecond)
			v2, err2 := c.rdb.Get(ctx, key).Result()
			if err2 == nil {
				var video models.Video
				if json.Unmarshal([]byte(v2), &video) == nil {
					return &video, nil
				}
			}
		}
		return dbQuery(ctx, id)
	}

	// We hold the lock — double-check before querying DB
	v, err = c.rdb.Get(ctx, key).Result()
	if err == nil {
		var video models.Video
		if json.Unmarshal([]byte(v), &video) == nil {
			c.lock.ReleaseDetail(ctx, id, token)
			return &video, nil
		}
	}

	// Load from DB
	video, err := dbQuery(ctx, id)
	if err != nil || video == nil {
		c.lock.ReleaseDetail(ctx, id, token)
		return video, err
	}

	// Write back to cache
	data, _ := json.Marshal(video)
	c.rdb.Set(ctx, key, data, detailTTL)
	c.local.Set(key, video, L1TTL)
	c.lock.ReleaseDetail(ctx, id, token)
	return video, nil
}

// SetVideoDetail writes a video detail entry into the L2 cache (and L1 local cache).
func (c *Cache) SetVideoDetail(ctx context.Context, video *models.Video) error {
	key := VideoDetail(video.ID)
	data, err := json.Marshal(video)
	if err != nil {
		return err
	}
	c.local.Set(key, video, L1TTL)
	return c.rdb.Set(ctx, key, data, detailTTL).Err()
}

// InvalidateVideoDetail removes a video detail entry from cache.
func (c *Cache) InvalidateVideoDetail(id uint) {
	key := VideoDetail(id)
	c.local.Delete(key)
}

// RebuildFollowCache builds the cold follow cache for a user when they page past
// their Inbox boundary. It pulls from all normal bloggers' Outbox (or DB fallback)
// and writes the merged results to v1:feed:followcache:{uid}:before:{ts}:limit:{n}
// with a singleflight soft-label for stampede protection.
//
// Parameters:
//   - normalAuthors: list of non-big-V author IDs to pull Outbox from
//   - fetchOutbox: returns videoIDs from an author's Outbox older than `before` (Redis first, DB fallback)
func (c *Cache) RebuildFollowCache(ctx context.Context, uid uint, before time.Time, limit int, normalAuthors []uint, fetchOutbox func(ctx context.Context, authorID uint, before time.Time, limit int) ([]string, error)) error {
	// Stampede protection via soft-label
	labelKey := SFLabel(fmt.Sprintf("fallback:followcache:%d", uid))
	set, err := c.rdb.SetNX(ctx, labelKey, "1", sfLabelTTL).Result()
	if err != nil {
		return err
	}
	if !set {
		// Already being rebuilt by another goroutine
		return nil
	}
	defer c.rdb.Del(ctx, labelKey)

	cacheKey := FeedCache(uid, before.Format(time.RFC3339Nano), limit)

	var allScores []redis.Z
	for _, authorID := range normalAuthors {
		ids, err := fetchOutbox(ctx, authorID, before, limit)
		if err != nil || len(ids) == 0 {
			continue
		}
		for _, idStr := range ids {
			allScores = append(allScores, redis.Z{Score: float64(time.Now().UnixNano()), Member: idStr})
		}
	}

	if len(allScores) == 0 {
		return nil
	}

	// Sort descending by score and dedup by member (videoID)
	sort.Slice(allScores, func(i, j int) bool {
		return allScores[i].Score > allScores[j].Score
	})
	seen := make(map[string]bool)
	deduped := make([]redis.Z, 0, len(allScores))
	for _, z := range allScores {
		memberStr, _ := z.Member.(string)
		if !seen[memberStr] {
			seen[memberStr] = true
			deduped = append(deduped, z)
		}
	}
	if len(deduped) > limit {
		deduped = deduped[:limit]
	}

	// Write merged results to ZSET
	c.rdb.Del(ctx, cacheKey)
	if len(deduped) > 0 {
		c.rdb.ZAdd(ctx, cacheKey, deduped...)
	}
	c.rdb.Expire(ctx, cacheKey, followCacheTTL)
	return nil
}
