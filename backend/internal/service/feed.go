package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"tideflow/internal/models"
	"tideflow/internal/repository"
	infraredis "tideflow/infra/redis"
)

type FeedService struct {
	repo    *repository.Repository
	cache   *infraredis.Cache
	rdb     *goredis.Client
	bigVThresh int
}

func NewFeedService(repo *repository.Repository, cache *infraredis.Cache, rdb *goredis.Client, bigVThresh int) *FeedService {
	return &FeedService{repo: repo, cache: cache, rdb: rdb, bigVThresh: bigVThresh}
}

func (s *FeedService) ListLatest(ctx context.Context, cursor string, limit int) ([]*models.Video, *string, bool, error) {
	var before time.Time
	if cursor != "" {
		ts, err := strconv.ParseInt(cursor, 10, 64)
		if err == nil {
			before = time.UnixMilli(ts)
		}
	}

	ids, err := s.rdb.ZRevRange(ctx, infraredis.FeedGlobal(), 0, int64(limit-1)).Result()
	if err != nil || len(ids) == 0 {
		videos, err := s.repo.GetLatestVideos(ctx, before, limit+1)
		return videos, nil, false, err
	}

	videoIDs := make([]uint, len(ids))
	for i, idStr := range ids {
		id, _ := strconv.ParseUint(idStr, 10, 64)
		videoIDs[i] = uint(id)
	}

	videos, err := s.cache.GetVideoByIDs(ctx, videoIDs)
	if err != nil || len(videos) == 0 {
		dbVideos, dbErr := s.repo.GetLatestVideos(ctx, before, limit+1)
		return dbVideos, nil, false, dbErr
	}

	hasMore := len(videos) > limit
	if hasMore {
		videos = videos[:limit]
	}

	var nextCursor *string
	if hasMore && len(videos) > 0 {
		nc := strconv.FormatInt(videos[len(videos)-1].CreateTime.UnixMilli(), 10)
		nextCursor = &nc
	}

	return videos, nextCursor, hasMore, nil
}

func (s *FeedService) ListPopular(ctx context.Context, offset, limit int, window string) ([]*models.Video, *string, bool, error) {
	now := time.Now()
	ts := now.Format("200601021504")

	mergeKey := infraredis.HotMerge(window, ts)

	count, _ := s.rdb.ZCard(ctx, mergeKey).Result()
	if count == 0 {
		keys := make([]string, 60)
		for i := 0; i < 60; i++ {
			t, _ := time.Parse("200601021504", ts)
			t = t.Add(-time.Duration(i) * time.Minute)
			keys[i] = infraredis.HotVideo(window, t.Format("200601021504"))
		}
		s.rdb.ZUnionStore(ctx, mergeKey, &goredis.ZStore{Keys: keys})
		s.rdb.Expire(ctx, mergeKey, 2*time.Minute)
	}

	idStrs, err := s.rdb.ZRevRange(ctx, mergeKey, int64(offset), int64(offset+limit-1)).Result()
	if err != nil || len(idStrs) == 0 {
		videos, err := s.repo.GetPopularVideos(ctx, offset, limit+1)
		if err != nil {
			return nil, nil, false, err
		}
		hasMore := len(videos) > limit
		if hasMore {
			videos = videos[:limit]
		}
		var next *string
		if hasMore {
			n := strconv.Itoa(offset + limit)
			next = &n
		}
		return videos, next, hasMore, nil
	}

	videoIDs := make([]uint, len(idStrs))
	for i, idStr := range idStrs {
		id, _ := strconv.ParseUint(idStr, 10, 64)
		videoIDs[i] = uint(id)
	}

	videos, err := s.cache.GetVideoByIDs(ctx, videoIDs)
	if err != nil {
		return nil, nil, false, err
	}

	next := strconv.Itoa(offset + limit)
	return videos, &next, true, nil
}

func (s *FeedService) ListByFollowing(ctx context.Context, userID uint, cursor string, limit int) ([]*models.Video, *string, bool, error) {
	followingIDs, err := s.repo.GetFollowingIDs(ctx, userID)
	if err != nil || len(followingIDs) == 0 {
		return nil, nil, false, nil
	}

	var before time.Time
	if cursor != "" {
		ts, err := strconv.ParseInt(cursor, 10, 64)
		if err == nil {
			before = time.UnixMilli(ts)
		}
	}

	// Split following into big-V and normal bloggers
	var bigVs, normals []uint
	for _, fid := range followingIDs {
		acc, _ := s.repo.GetAccountByID(ctx, fid)
		if acc != nil && acc.FollowerCount >= s.bigVThresh {
			bigVs = append(bigVs, fid)
		} else {
			normals = append(normals, fid)
		}
	}

	type videoScore struct {
		video *models.Video
		score float64
	}
	var all []videoScore

	// ── Hot path: Inbox + big-V Outbox (only when no cursor / first page) ──
	if before.IsZero() {
		// Inbox: pull up to 500 recent entries
		inboxKey := infraredis.Inbox(userID)
		inboxIDs, _ := s.rdb.ZRevRange(ctx, inboxKey, 0, 499).Result()
		for _, idStr := range inboxIDs {
			id, _ := strconv.ParseUint(idStr, 10, 64)
			videos, _ := s.cache.GetVideoByIDs(ctx, []uint{uint(id)})
			if len(videos) > 0 && videos[0] != nil {
				all = append(all, videoScore{videos[0], float64(videos[0].CreateTime.UnixMilli())})
			}
		}

		// Big-V outboxes: up to 200 each
		for _, bvID := range bigVs {
			ids, _ := s.rdb.ZRevRange(ctx, infraredis.Outbox(bvID), 0, 199).Result()
			for _, idStr := range ids {
				id, _ := strconv.ParseUint(idStr, 10, 64)
				videos, _ := s.cache.GetVideoByIDs(ctx, []uint{uint(id)})
				if len(videos) > 0 && videos[0] != nil {
					all = append(all, videoScore{videos[0], float64(videos[0].CreateTime.UnixMilli())})
				}
			}
		}
	}

	// ── Cold path: check / build follow cache ──
	cacheKey := infraredis.FeedCache(userID, cursor, limit)
	cached, _ := s.rdb.ZRange(ctx, cacheKey, 0, -1).Result()
	if len(cached) == 0 && len(normals) > 0 {
		// Cache miss — rebuild via singleflight-protected cold pull
		cacheKeyForBuild := infraredis.FeedCache(userID, cursor, limit)
		s.cache.RebuildFollowCache(ctx, userID, before, limit, normals,
			func(ctx context.Context, authorID uint, before time.Time, limit int) ([]string, error) {
				key := infraredis.Outbox(authorID)
				var ids []string
				var err error
				if before.IsZero() {
					ids, err = s.rdb.ZRange(ctx, key, 0, int64(limit-1)).Result()
				} else {
					ids, err = s.rdb.ZRevRangeByScore(ctx, key, &goredis.ZRangeBy{
						Min:   "-inf",
						Max:   fmt.Sprintf("%d", before.UnixMilli()),
						Count: int64(limit),
					}).Result()
				}
				if err != nil || len(ids) == 0 {
					// Fallback: pull from DB
					dbVideos, dbErr := s.repo.GetVideosByAuthor(ctx, authorID, before, limit)
					if dbErr != nil || len(dbVideos) == 0 {
						return nil, dbErr
					}
					ids = make([]string, len(dbVideos))
					for i, v := range dbVideos {
						ids[i] = fmt.Sprintf("%d", v.ID)
					}
				}
				return ids, err
			},
		)
		// Re-read after rebuild
		cached, _ = s.rdb.ZRange(ctx, cacheKeyForBuild, 0, -1).Result()
	}

	// Merge cold cache entries
	for _, idStr := range cached {
		id, _ := strconv.ParseUint(idStr, 10, 64)
		videos, _ := s.cache.GetVideoByIDs(ctx, []uint{uint(id)})
		if len(videos) > 0 && videos[0] != nil {
			all = append(all, videoScore{videos[0], float64(videos[0].CreateTime.UnixMilli())})
		}
	}

	// ── Sort and dedup ──
	for i := 0; i < len(all)-1; i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].score > all[i].score {
				all[i], all[j] = all[j], all[i]
			}
		}
	}
	seen := make(map[uint]bool)
	deduped := make([]videoScore, 0, len(all))
	for _, vs := range all {
		if !seen[vs.video.ID] {
			seen[vs.video.ID] = true
			deduped = append(deduped, vs)
		}
	}
	if len(deduped) > limit {
		deduped = deduped[:limit]
	}

	videos := make([]*models.Video, len(deduped))
	for i, vs := range deduped {
		videos[i] = vs.video
	}

	var nextCursor *string
	if len(videos) == limit && videos[limit-1] != nil {
		nc := strconv.FormatInt(videos[limit-1].CreateTime.UnixMilli(), 10)
		nextCursor = &nc
	}

	return videos, nextCursor, len(videos) == limit, nil
}

func (s *FeedService) ListByTag(ctx context.Context, tag string, cursor string, limit int) ([]*models.Video, *string, bool, error) {
	var tagModel models.Tag
	s.repo.DB().Where("name = ?", tag).First(&tagModel)

	var videoTags []models.VideoTag
	s.repo.DB().Where("tag_id = ?", tagModel.ID).Find(&videoTags)

	if len(videoTags) == 0 {
		return []*models.Video{}, nil, false, nil
	}

	videoIDs := make([]uint, len(videoTags))
	for i, vt := range videoTags {
		videoIDs[i] = vt.VideoID
	}

	videos, err := s.cache.GetVideoByIDs(ctx, videoIDs)
	if err != nil || len(videos) == 0 {
		dbVideos, dbErr := s.repo.GetVideosByIDs(ctx, videoIDs)
		return dbVideos, nil, false, dbErr
	}

	hasMore := len(videos) > limit
	if hasMore {
		videos = videos[:limit]
	}

	var nextCursor *string
	if hasMore && len(videos) > 0 {
		nc := strconv.FormatInt(videos[len(videos)-1].CreateTime.UnixMilli(), 10)
		nextCursor = &nc
	}

	return videos, nextCursor, hasMore, nil
}