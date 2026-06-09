package service

import (
	"context"
	"log/slog"

	"tideflow/infra/es"
	"tideflow/internal/models"
	"tideflow/internal/repository"
	infraredis "tideflow/infra/redis"
)

type SearchService struct {
	esClient *es.Client
	cache    *infraredis.Cache
	repo     *repository.Repository
	bigVThresh int
}

func NewSearchService(esClient *es.Client, cache *infraredis.Cache, repo *repository.Repository, bigVThresh int) *SearchService {
	return &SearchService{esClient: esClient, cache: cache, repo: repo, bigVThresh: bigVThresh}
}

// SearchVideos performs an ES search and returns matched video IDs and total count.
func (s *SearchService) SearchVideos(ctx context.Context, query, sortBy, order string, from, size int) ([]uint, int64, error) {
	params := es.SearchParams{
		Query:  query,
		SortBy: sortBy,
		Order:  order,
		From:   from,
		Size:   size,
	}

	result, err := s.esClient.Search(ctx, params)
	if err != nil {
		slog.Warn("ES search failed", "query", query, "err", err)
		return nil, 0, err
	}

	slog.Info("ES search success", "query", query, "total", result.Total, "hits", len(result.VideoIDs))
	return result.VideoIDs, result.Total, nil
}

// EnrichVideoIDs fetches full video entities for a list of IDs using the three-tier cache.
func (s *SearchService) EnrichVideoIDs(ctx context.Context, ids []uint) ([]*models.Video, error) {
	videos, err := s.cache.GetVideoByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return videos, nil
}

// SearchUsers searches users by username (LIKE match), ordered by follower count.
func (s *SearchService) SearchUsers(ctx context.Context, query string, page, size int) ([]*models.Account, int64, error) {
	offset := (page - 1) * size
	return s.repo.SearchUsers(ctx, query, offset, size)
}

// EnrichWithAccountData fills in avatar_url and is_big_v for a list of videos.
func (s *SearchService) EnrichWithAccountData(ctx context.Context, videos []*models.Video) {
	if len(videos) == 0 {
		return
	}
	authorIDs := make(map[uint]bool)
	for _, v := range videos {
		if v != nil {
			authorIDs[v.AuthorID] = true
		}
	}
	ids := make([]uint, 0, len(authorIDs))
	for id := range authorIDs {
		ids = append(ids, id)
	}
	accounts, err := s.repo.GetAccountsByIDs(ctx, ids)
	if err != nil {
		return
	}
	accMap := make(map[uint]*models.Account)
	for _, acc := range accounts {
		accMap[acc.ID] = acc
	}
	for _, v := range videos {
		if v == nil {
			continue
		}
		if acc, ok := accMap[v.AuthorID]; ok {
			v.AvatarURL = acc.AvatarURL
			v.IsBigV = acc.FollowerCount >= s.bigVThresh
		}
	}
}