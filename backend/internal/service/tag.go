package service

import (
	"context"

	"tideflow/internal/repository"
)

type TagService struct {
	repo *repository.Repository
}

func NewTagService(repo *repository.Repository) *TagService {
	return &TagService{repo: repo}
}

func (s *TagService) GetHotTags(ctx context.Context, limit int) ([]string, error) {
	tags, err := s.repo.GetHotTags(ctx, limit)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(tags))
	for i, t := range tags {
		names[i] = t.Name
	}
	return names, nil
}
