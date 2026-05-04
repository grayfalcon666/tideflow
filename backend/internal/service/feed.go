package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
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
		minutes := windowToMinutes(window)
		keys := make([]string, minutes)
		for i := 0; i < minutes; i++ {
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

	// ── 大V判定：优先读 Redis BigVMark，miss 时查 DB 并回写 ──────────────
	var bigVs, normals []uint
	for _, fid := range followingIDs {
		isBigV, _ := s.rdb.Exists(ctx, infraredis.BigVMark(fid)).Result()
		if isBigV > 0 {
			bigVs = append(bigVs, fid)
		} else {
			normals = append(normals, fid)
		}
	}

	// 对于 normals，补充 DB 查询确认（BigVMark miss 不代表不是大V）
	if len(normals) > 0 {
		bigVUpdates := make([]uint, 0)
		for _, fid := range normals {
			acc, err := s.repo.GetAccountByID(ctx, fid)
			if err == nil && acc != nil {
				if acc.FollowerCount >= s.bigVThresh {
					bigVs = append(bigVs, fid)
					bigVUpdates = append(bigVUpdates, fid)
				}
			}
		}
		// 回写 BigVMark（TTL 1h）
		for _, fid := range bigVUpdates {
			s.rdb.Set(ctx, infraredis.BigVMark(fid), "1", time.Hour)
		}
		// normals 重新计算（排除已被确认大V的）
		normalsMap := make(map[uint]bool)
		for _, fid := range normals {
			normalsMap[fid] = true
		}
		for _, fid := range bigVUpdates {
			delete(normalsMap, fid)
		}
		normals = normalsMapToSlice(normalsMap)
	}

	type videoScore struct {
		video *models.Video
		score float64
	}
	var all []videoScore

	// ── 热路径：Inbox + 并行大V Outbox（仅首页/无 cursor） ──────────────
	if before.IsZero() {
		// Inbox：取 500 条
		inboxIDs, _ := s.rdb.ZRevRange(ctx, infraredis.Inbox(userID), 0, 499).Result()
		for _, idStr := range inboxIDs {
			id, _ := strconv.ParseUint(idStr, 10, 64)
			videos, _ := s.cache.GetVideoByIDs(ctx, []uint{uint(id)})
			if len(videos) > 0 && videos[0] != nil {
				all = append(all, videoScore{videos[0], float64(videos[0].CreateTime.UnixMilli())})
			}
		}

		// 大V Outbox：并行拉取
		if len(bigVs) > 0 {
			var wg sync.WaitGroup
			var mu sync.Mutex
			for _, bvID := range bigVs {
				wg.Add(1)
				go func(bvID uint) {
					defer wg.Done()
					ids, _ := s.rdb.ZRevRange(ctx, infraredis.Outbox(bvID), 0, 199).Result()
					if len(ids) == 0 {
						return
					}
					videoIDs := make([]uint, 0, len(ids))
					for _, idStr := range ids {
						id, _ := strconv.ParseUint(idStr, 10, 64)
						videoIDs = append(videoIDs, uint(id))
					}
					videos, _ := s.cache.GetVideoByIDs(ctx, videoIDs)
					mu.Lock()
					for _, v := range videos {
						if v != nil {
							all = append(all, videoScore{v, float64(v.CreateTime.UnixMilli())})
						}
					}
					mu.Unlock()
				}(bvID)
			}
			wg.Wait()
		}
	}

	// ── 冷路径：查 follow cache，未命中则触发重建 ────────────────────────
	cacheKey := infraredis.FeedCache(userID, cursor, limit)
	cached, _ := s.rdb.ZRange(ctx, cacheKey, 0, -1).Result()
	if len(cached) == 0 && len(normals) > 0 {
		s.cache.RebuildFollowCache(ctx, userID, before, limit, normals,
			func(ctx context.Context, authorID uint, before time.Time, limit int) ([]goredis.Z, error) {
				key := infraredis.Outbox(authorID)
				var err error
				var zs []goredis.Z
				if before.IsZero() {
					zs, err = s.rdb.ZRangeWithScores(ctx, key, 0, int64(limit-1)).Result()
				} else {
					zs, err = s.rdb.ZRevRangeByScoreWithScores(ctx, key, &goredis.ZRangeBy{
						Min:   "-inf",
						Max:   fmt.Sprintf("%d", before.UnixMilli()),
						Count: int64(limit),
					}).Result()
				}
				if err != nil || len(zs) == 0 {
					dbVideos, dbErr := s.repo.GetVideosByAuthor(ctx, authorID, before, limit)
					if dbErr != nil || len(dbVideos) == 0 {
						return nil, dbErr
					}
					zs = make([]goredis.Z, len(dbVideos))
					for i, v := range dbVideos {
						zs[i] = goredis.Z{Score: float64(v.CreateTime.UnixMilli()), Member: fmt.Sprintf("%d", v.ID)}
					}
				}
				return zs, err
			},
		)
		cached, _ = s.rdb.ZRange(ctx, cacheKey, 0, -1).Result()
	}

	// 合并冷缓存数据
	for _, idStr := range cached {
		id, _ := strconv.ParseUint(idStr, 10, 64)
		videos, _ := s.cache.GetVideoByIDs(ctx, []uint{uint(id)})
		if len(videos) > 0 && videos[0] != nil {
			all = append(all, videoScore{videos[0], float64(videos[0].CreateTime.UnixMilli())})
		}
	}

	// ── 排序去重 ─────────────────────────────────────────────────────────
	sort.Slice(all, func(i, j int) bool {
		return all[i].score > all[j].score
	})
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

// windowToMinutes converts a window string to number of minutes to aggregate.
func windowToMinutes(window string) int {
	switch window {
	case "1m":
		return 1
	case "5m":
		return 5
	case "15m":
		return 15
	case "1h":
		return 60
	case "6h":
		return 360
	default:
		return 60
	}
}

func normalsMapToSlice(m map[uint]bool) []uint {
	res := make([]uint, 0, len(m))
	for k := range m {
		res = append(res, k)
	}
	return res
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