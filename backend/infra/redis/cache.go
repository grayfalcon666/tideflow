package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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
	// 轮询重试间隔（每次等待时长）
	sfPollInterval = 100 * time.Millisecond
	// 轮询最多等待时间
	sfPollTimeout = 2 * time.Second
	// 关注流冷拉取缓存 TTL，设计文档要求 24h
	followCacheTTL = 24 * time.Hour
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

// SetRepo 设置 repository 引用（用于 DB 降级回填）
func (c *Cache) SetRepo(repo *repository.Repository) {
	c.repo = repo
}

// GetRedis returns the underlying redis client for low-level operations.
func (c *Cache) GetRedis() *redis.Client {
	return c.rdb
}

// ---------------------------------------------------------------
// 视频实体缓存（统一 L1 key 格式为 VideoEntity(id)）
// ---------------------------------------------------------------

func (c *Cache) GetVideoEntity(ctx context.Context, id uint) (*models.Video, error) {
	key := VideoEntity(id)

	// L1
	if v, ok := c.local.Get(key); ok {
		if video, ok := v.(*models.Video); ok {
			slog.Info("cache: L1命中", "key", key, "id", id)
			return video, nil
		}
	}

	// L2
	v, err := c.rdb.Get(ctx, key).Result()
	if err == nil {
		var video models.Video
		if json.Unmarshal([]byte(v), &video) == nil {
			c.local.Set(key, &video, L1TTL)
			slog.Info("cache: L2命中", "key", key, "id", id)
			return &video, nil
		}
	}

	// L2 miss，降级到 DB（健壮性补充）
	if c.repo != nil {
		dbVideo, dbErr := c.repo.GetVideoByID(ctx, id)
		if dbErr == nil && dbVideo != nil {
			// 回写缓存
			data, _ := json.Marshal(dbVideo)
			c.rdb.Set(ctx, key, data, L2TTL)
			c.local.Set(key, dbVideo, L1TTL)
			slog.Info("cache: 未命中，回源MySQL", "key", key, "id", id)
			return dbVideo, nil
		}
		return nil, dbErr
	}

	return nil, err
}

func (c *Cache) GetVideoByIDs(ctx context.Context, ids []uint) ([]*models.Video, error) {
	if len(ids) == 0 {
		return []*models.Video{}, nil
	}

	results := make([]*models.Video, len(ids))
	var missedIdx []int

	// L1 批量查询，统一使用 VideoEntity(id) 作为键
	l1HitCount := 0
	for i, id := range ids {
		if v, ok := c.local.Get(VideoEntity(id)); ok {
			if video, ok := v.(*models.Video); ok {
				results[i] = video
				l1HitCount++
				continue
			}
		}
		missedIdx = append(missedIdx, i)
	}
	slog.Info("cache: 批量L1查询", "total", len(ids), "l1_hits", l1HitCount)

	if len(missedIdx) == 0 {
		return results, nil
	}

	missedIDs := make([]uint, len(missedIdx))
	for i, idx := range missedIdx {
		missedIDs[i] = ids[idx]
	}

	// 软标记检查：如果已有其他实例正在重建，轮询等待
	sfLabelKey := SFLabel(fmt.Sprintf("entity:%v", missedIDs))
	if c.isRebuilding(ctx, sfLabelKey) {
		c.waitForRebuild(ctx, sfLabelKey)
		// 轮询结束后重新尝 L2 缓存
		vals2, err2 := c.rdb.MGet(ctx, makeRedisKeys(missedIDs, VideoEntity)...).Result()
		if err2 == nil {
			for i, v := range vals2 {
				if v == nil {
					continue
				}
				var video models.Video
				if json.Unmarshal([]byte(v.(string)), &video) == nil {
					c.local.Set(VideoEntity(missedIDs[i]), &video, L1TTL)
					results[missedIdx[i]] = &video
				}
			}
		}
		// 对仍未命中的 ID 继续查 DB
		stillMissedIdx := make([]int, 0)
		stillMissedIDs := make([]uint, 0)
		for i, idx := range missedIdx {
			if results[idx] == nil {
				stillMissedIdx = append(stillMissedIdx, idx)
				stillMissedIDs = append(stillMissedIDs, missedIDs[i])
			}
		}
		if len(stillMissedIDs) > 0 {
			return c.loadFromDB(ctx, stillMissedIDs, results, stillMissedIdx)
		}
		return results, nil
	}

	// singleflight：进程内去重
	sfKey := fmt.Sprintf("videos:%v", missedIDs)
	ret, _, _ := c.sf.Do(sfKey, func() (interface{}, error) {
		// 原子性设置软标记
		c.rdb.SetNX(ctx, sfLabelKey, "1", sfLabelTTL)
		// L2 MGet
		vals, err := c.rdb.MGet(ctx, makeRedisKeys(missedIDs, VideoEntity)...).Result()
		if err != nil {
			return nil, err
		}
		l2HitCount := 0
		// 回写 L1（统一 key）
		for i, v := range vals {
			if v == nil {
				continue
			}
			var video models.Video
			if json.Unmarshal([]byte(v.(string)), &video) == nil {
				c.local.Set(VideoEntity(missedIDs[i]), &video, L1TTL)
				// 标记 results 为命中
				results[missedIdx[i]] = &video
				l2HitCount++
			}
		}
		slog.Info("cache: 批量L2查询", "total", len(missedIDs), "l2_hits", l2HitCount)
		return vals, nil
	})

	// 清理软标记
	defer c.rdb.Del(context.Background(), sfLabelKey)

	if _, ok := ret.([]interface{}); ok {
		// 对 L2 未命中的 ID 继续查 DB
		stillMissedIdx := make([]int, 0)
		stillMissedIDs := make([]uint, 0)
		for i, idx := range missedIdx {
			if results[idx] == nil {
				stillMissedIdx = append(stillMissedIdx, idx)
				stillMissedIDs = append(stillMissedIDs, missedIDs[i])
			}
		}
		if len(stillMissedIDs) > 0 {
			return c.loadFromDB(ctx, stillMissedIDs, results, stillMissedIdx)
		}
		return results, nil
	}

	// singleflight 错误，降级 DB
	return c.loadFromDB(ctx, missedIDs, results, missedIdx)
}

func (c *Cache) SetVideoEntity(ctx context.Context, video *models.Video) error {
	if video == nil {
		return fmt.Errorf("video is nil")
	}
	key := VideoEntity(video.ID)
	data, err := json.Marshal(video)
	if err != nil {
		return err
	}
	c.local.Set(key, video, L1TTL)
	return c.rdb.Set(ctx, key, data, L2TTL).Err()
}

func (c *Cache) InvalidateVideo(id uint) {
	key := VideoEntity(id)
	c.local.Delete(key)
	c.rdb.Del(context.Background(), key) // 同时删除 L2
}

// GetVideoDetail 复用 GetVideoEntity 的单条查询，内部通过 singleflight 防止击穿。
func (c *Cache) GetVideoDetail(ctx context.Context, id uint, dbQuery func(context.Context, uint) (*models.Video, error)) (*models.Video, error) {
	videos, err := c.GetVideoByIDs(ctx, []uint{id})
	if err != nil {
		return nil, err
	}
	if len(videos) == 0 || videos[0] == nil {
		return dbQuery(ctx, id)
	}
	return videos[0], nil
}

// SetVideoDetail 兼容旧调用，统一写入 VideoEntity。
func (c *Cache) SetVideoDetail(ctx context.Context, video *models.Video) error {
	return c.SetVideoEntity(ctx, video)
}

// InvalidateVideoDetail 兼容旧调用，统一用 InvalidateVideo。
func (c *Cache) InvalidateVideoDetail(id uint) {
	c.InvalidateVideo(id)
}

// InvalidateSubtitle deletes subtitle cache from L2 Redis.
func (c *Cache) InvalidateSubtitle(videoID uint) {
	key := VideoSubtitle(videoID)
	c.rdb.Del(context.Background(), key)
}

// InvalidateWordbank deletes wordbank cache from L2 Redis.
func (c *Cache) InvalidateWordbank(videoID uint) {
	key := VideoWordbank(videoID)
	c.rdb.Del(context.Background(), key)
}

// InvalidateVideoSubtitleAndWordbank deletes video entity, subtitle, and wordbank caches.
func (c *Cache) InvalidateVideoSubtitleAndWordbank(videoID uint) {
	c.InvalidateVideo(videoID)
	c.InvalidateSubtitle(videoID)
	c.InvalidateWordbank(videoID)
}

// ---------------------------------------------------------------
// 学习习惯 — Bitmap 打卡
// ---------------------------------------------------------------

// SetHabitBitmap sets the habit bitmap for the given day of year.
// The key expires at the end of the current year.
func (c *Cache) SetHabitBitmap(ctx context.Context, key string, dayOfYear int) {
	now := time.Now()
	yearEnd := time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, time.UTC)
	ttl := yearEnd.Sub(now) + 24*time.Hour // extra day buffer
	c.rdb.SetBit(ctx, key, int64(dayOfYear-1), 1)
	c.rdb.Expire(ctx, key, ttl)
}

// GetHabitStats returns total check-in days and current streak from the bitmap.
func (c *Cache) GetHabitStats(ctx context.Context, key string) (totalDays int, currentStreak int) {
	countCmd := c.rdb.BitCount(ctx, key, nil)
	totalDays = int(countCmd.Val())

	// Calculate current streak by scanning backward from today
	today := time.Now()
	dayOfYear := today.YearDay()
	streak := 0
	for d := dayOfYear; d >= 1; d-- {
		bit, err := c.rdb.GetBit(ctx, key, int64(d-1)).Result()
		if err != nil || bit != 1 {
			break
		}
		streak++
	}
	return totalDays, streak
}

// GetTodayWords returns cached today's words from Redis.
func (c *Cache) GetTodayWords(ctx context.Context, key string) (string, error) {
	return c.rdb.Get(ctx, key).Result()
}

// SetTodayWords caches today's words with TTL until end of day.
func (c *Cache) SetTodayWords(ctx context.Context, key string, data string) {
	now := time.Now()
	eod := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	ttl := eod.Sub(now) + time.Minute
	c.rdb.Set(ctx, key, data, ttl)
}

// InvalidateTodayWords removes the today's words cache entry.
func (c *Cache) InvalidateTodayWords(ctx context.Context, key string) {
	c.rdb.Del(ctx, key)
}

// ---------------------------------------------------------------
// 冷拉取缓存重建
// ---------------------------------------------------------------

// RebuildFollowCache builds the cold follow cache for a user when they page past
// their Inbox boundary. fetchOutbox must return videoIDs and their publish timestamps
// (as unix seconds) for correct sorting.
func (c *Cache) RebuildFollowCache(ctx context.Context, uid uint, before time.Time, limit int, normalAuthors []uint, fetchOutbox func(ctx context.Context, authorID uint, before time.Time, limit int) ([]redis.Z, error)) error {
	// 软标记防多实例并发重建
	labelKey := SFLabel(fmt.Sprintf("fallback:followcache:%d", uid))
	set, err := c.rdb.SetNX(ctx, labelKey, "1", sfLabelTTL).Result()
	if err != nil {
		return err
	}
	if !set {
		// 已有其他实例在重建
		return nil
	}
	defer c.rdb.Del(ctx, labelKey)

	cacheKey := FeedCache(uid, before.Format(time.RFC3339Nano), limit)

	var allScores []redis.Z
	for _, authorID := range normalAuthors {
		scores, err := fetchOutbox(ctx, authorID, before, limit)
		if err != nil || len(scores) == 0 {
			continue
		}
		allScores = append(allScores, scores...)
	}

	if len(allScores) == 0 {
		return nil
	}

	// 按 Score（即发布时间戳）降序排序
	sort.Slice(allScores, func(i, j int) bool {
		return allScores[i].Score > allScores[j].Score
	})

	// 去重（按 member 即 videoID）
	seen := make(map[string]bool)
	deduped := make([]redis.Z, 0, len(allScores))
	for _, z := range allScores {
		memberStr, ok := z.Member.(string)
		if !ok {
			continue
		}
		if !seen[memberStr] {
			seen[memberStr] = true
			deduped = append(deduped, z)
		}
	}

	if len(deduped) > limit {
		deduped = deduped[:limit]
	}

	// 写入 ZSET，先清空再批量插入
	c.rdb.Del(ctx, cacheKey)
	if len(deduped) > 0 {
		c.rdb.ZAdd(ctx, cacheKey, deduped...)
	}
	c.rdb.Expire(ctx, cacheKey, followCacheTTL)
	return nil
}

// ---------------------------------------------------------------
// 内部辅助函数
// ---------------------------------------------------------------

// isRebuilding 检查 Redis 中是否存在指定的重建标记
func (c *Cache) isRebuilding(ctx context.Context, labelKey string) bool {
	exists, _ := c.rdb.Exists(ctx, labelKey).Result()
	return exists > 0
}

// waitForRebuild 轮询等待重建软标记消失，最长等待 sfPollTimeout。
// 设计文档要求：软标记 SETNX 短 TTL，完成即删除；其他请求轮询等待。
func (c *Cache) waitForRebuild(ctx context.Context, labelKey string) bool {
	deadline := time.Now().Add(sfPollTimeout)
	for time.Now().Before(deadline) {
		exists, err := c.rdb.Exists(ctx, labelKey).Result()
		if err != nil || exists == 0 {
			return true // 重建完成或出错
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(sfPollInterval):
		}
	}
	return false // 超时
}

// loadFromDB 从 MySQL 加载实体，并回写 L2 和 L1（统一 key）
func (c *Cache) loadFromDB(ctx context.Context, ids []uint, results []*models.Video, indices []int) ([]*models.Video, error) {
	if c.repo == nil {
		return results, fmt.Errorf("repository not set")
	}

	slog.Info("cache: 批量回源MySQL", "ids", ids)
	dbVideos, err := c.repo.GetVideosByIDs(ctx, ids)
	if err != nil {
		return results, err
	}

	videoMap := make(map[uint]*models.Video)
	for _, v := range dbVideos {
		videoMap[v.ID] = v
	}

	for i, idx := range indices {
		id := ids[i]
		if video, ok := videoMap[id]; ok {
			results[idx] = video
			// 回写 L2
			data, _ := json.Marshal(video)
			c.rdb.Set(ctx, VideoEntity(id), data, L2TTL)
			// 回写 L1（统一 key）
			c.local.Set(VideoEntity(id), video, L1TTL)
		}
	}

	// 清理可能残留的软标记
	sfLabelKey := SFLabel(fmt.Sprintf("entity:%v", ids))
	c.rdb.Del(ctx, sfLabelKey)
	return results, nil
}

func makeRedisKeys(ids []uint, fn func(uint) string) []string {
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = fn(id)
	}
	return keys
}
