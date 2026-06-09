package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"tideflow/internal/models"
	"tideflow/internal/repository"
	redis "tideflow/infra/redis"
)

const historyLimitTTL = 30 * time.Minute
const historyCacheSize = 100

type HistoryService struct {
	repo *repository.Repository
	rdb  *goredis.Client
}

func NewHistoryService(repo *repository.Repository, rdb *goredis.Client) *HistoryService {
	return &HistoryService{repo: repo, rdb: rdb}
}

// RecordHistory records a video watch event with deduplication (upsert) and Redis cache update.
func (s *HistoryService) RecordHistory(ctx context.Context, userID, videoID uint) error {
	if userID == 0 || videoID == 0 {
		return nil
	}

	// 30-minute rate limit per user+video
	limitKey := redis.HistoryLimit(userID, videoID)
	set, err := s.rdb.SetNX(ctx, limitKey, "1", historyLimitTTL).Result()
	if err != nil {
		return err
	}
	if !set {
		return nil
	}

	// Write to DB (upsert)
	if err := s.repo.UpsertWatchHistory(ctx, userID, videoID); err != nil {
		return err
	}

	// Update Redis ZSET cache (incremental)
	cacheKey := redis.WatchHistory(userID)
	s.rdb.ZAdd(ctx, cacheKey, goredis.Z{Score: float64(time.Now().UnixMilli()), Member: videoID})
	s.rdb.ZRemRangeByRank(ctx, cacheKey, 0, -int64(historyCacheSize+1))

	return nil
}

// GetHistory returns paginated watch history. Reads from Redis cache first; falls back to DB and backfills cache.
func (s *HistoryService) GetHistory(ctx context.Context, userID uint, cursor string, limit int) ([]*models.Video, *string, bool, error) {
	cacheKey := redis.WatchHistory(userID)

	var before float64
	if cursor != "" {
		ts, err := parseHistoryCursor(cursor)
		if err == nil {
			before = float64(ts)
		}
	}

	// Try Redis cache first
	if s.rdb.Exists(ctx, cacheKey).Val() > 0 {
		return s.getHistoryFromCache(ctx, userID, cacheKey, before, limit)
	}

	// Cache miss: read from DB and backfill
	return s.getHistoryFromDBAndBackfill(ctx, userID, cacheKey, cursor, limit, before)
}

func (s *HistoryService) getHistoryFromCache(ctx context.Context, userID uint, cacheKey string, before float64, limit int) ([]*models.Video, *string, bool, error) {
	max := "+inf"
	if before > 0 {
		max = fmt.Sprintf("%d", int64(before)-1)
	}

	// ZREVRANGEBYSCORE with LIMIT
	results, err := s.rdb.ZRevRangeByScoreWithScores(ctx, cacheKey, &goredis.ZRangeBy{
		Min:   "-inf",
		Max:   max,
		Count: int64(limit + 1),
	}).Result()
	if err != nil {
		return nil, nil, false, err
	}

	hasMore := len(results) > limit
	if hasMore {
		results = results[:limit]
	}

	videoIDs := make([]uint, len(results))
	for i, r := range results {
		switch m := r.Member.(type) {
		case string:
			if id, err := strconv.ParseUint(m, 10, 64); err == nil {
				videoIDs[i] = uint(id)
			}
		case int64:
			videoIDs[i] = uint(m)
		case uint64:
			videoIDs[i] = uint(m)
		}
	}

	videos, err := s.repo.GetVideosByIDs(ctx, videoIDs)
	if err != nil {
		return nil, nil, false, err
	}

	videoMap := make(map[uint]*models.Video, len(videos))
	for _, v := range videos {
		videoMap[v.ID] = v
	}
	ordered := make([]*models.Video, 0, len(videoIDs))
	for _, id := range videoIDs {
		if v, ok := videoMap[id]; ok {
			ordered = append(ordered, v)
		}
	}

	var nextCursor *string
	if hasMore && len(results) > 0 {
		nc := strconv.FormatInt(int64(results[len(results)-1].Score), 10)
		nextCursor = &nc
	}

	return ordered, nextCursor, hasMore, nil
}

func (s *HistoryService) getHistoryFromDBAndBackfill(ctx context.Context, userID uint, cacheKey string, cursor string, limit int, before float64) ([]*models.Video, *string, bool, error) {
	var beforeTime time.Time
	if before > 0 {
		beforeTime = time.UnixMilli(int64(before))
	}

	records, err := s.repo.GetWatchHistory(ctx, userID, beforeTime, limit+1)
	if err != nil {
		return nil, nil, false, err
	}

	// Backfill Redis cache from DB (up to historyCacheSize records)
	if cursor == "" { // Only backfill on first page (no cursor)
		go func() {
			bgCtx := context.Background()
			allRecords, err := s.repo.GetWatchHistory(bgCtx, userID, time.Time{}, historyCacheSize)
			if err != nil || len(allRecords) == 0 {
				return
			}
			zs := make([]goredis.Z, len(allRecords))
			for i, r := range allRecords {
				zs[i] = goredis.Z{Score: float64(r.WatchedAt.UnixMilli()), Member: r.VideoID}
			}
			s.rdb.ZAdd(bgCtx, cacheKey, zs...)
		}()
	}

	hasMore := len(records) > limit
	if hasMore {
		records = records[:limit]
	}

	videoIDs := make([]uint, len(records))
	for i, r := range records {
		videoIDs[i] = r.VideoID
	}

	videos, err := s.repo.GetVideosByIDs(ctx, videoIDs)
	if err != nil {
		return nil, nil, false, err
	}

	videoMap := make(map[uint]*models.Video, len(videos))
	for _, v := range videos {
		videoMap[v.ID] = v
	}
	ordered := make([]*models.Video, 0, len(records))
	for _, r := range records {
		if v, ok := videoMap[r.VideoID]; ok {
			ordered = append(ordered, v)
		}
	}

	var nextCursor *string
	if hasMore && len(records) > 0 {
		nc := strconv.FormatInt(records[len(records)-1].WatchedAt.UnixMilli(), 10)
		nextCursor = &nc
	}

	return ordered, nextCursor, hasMore, nil
}

// DeleteHistory deletes a single watch history record from DB and Redis.
func (s *HistoryService) DeleteHistory(ctx context.Context, userID, videoID uint) error {
	if err := s.repo.DeleteWatchHistory(ctx, userID, videoID); err != nil {
		return err
	}
	s.rdb.ZRem(ctx, redis.WatchHistory(userID), videoID)
	s.rdb.Del(ctx, redis.HistoryLimit(userID, videoID)) // 清除限流，允许重新记录
	return nil
}

// ClearHistory deletes all watch history for a user from DB and Redis.
func (s *HistoryService) ClearHistory(ctx context.Context, userID uint) error {
	if err := s.repo.ClearWatchHistory(ctx, userID); err != nil {
		return err
	}
	s.rdb.Del(ctx, redis.WatchHistory(userID))
	return nil
}

func parseHistoryCursor(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}
	var v int64
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, fmt.Errorf("non-numeric character")
		}
		v = v*10 + int64(s[i]-'0')
	}
	return v, nil
}
