package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"tideflow/infra/redis"
	"tideflow/internal/models"
	"tideflow/internal/repository"
)

var (
	ErrWordbankNotFound   = errors.New("wordbank not found")
	ErrWordbankFailed     = errors.New("wordbank generation failed")
	ErrWordbankProcessing = errors.New("wordbank still processing")
)

type WordbankService struct {
	repo  *repository.Repository
	cache *redis.Cache
}

func NewWordbankService(repo *repository.Repository, cache *redis.Cache) *WordbankService {
	return &WordbankService{repo: repo, cache: cache}
}

// WordbankResponse is the API response for GET /wordbank.
type WordbankResponse struct {
	VideoID uint            `json:"video_id"`
	Size    int             `json:"size"`
	Words   json.RawMessage `json:"words"`
}

// GetWordbank retrieves the wordbank for a video.
// Returns (response, nil) if ready, or an appropriate error if not.
func (s *WordbankService) GetWordbank(ctx context.Context, videoID uint) (*WordbankResponse, error) {
	// 1. Check Redis L2 cache
	key := redis.VideoWordbank(videoID)
	if cached, err := s.cache.GetRedis().Get(ctx, key).Result(); err == nil {
		var resp WordbankResponse
		if json.Unmarshal([]byte(cached), &resp) == nil {
			slog.Info("wordbank: L2(Redis) 命中", "video_id", videoID, "size", resp.Size)
			return &resp, nil
		}
	}

	slog.Info("wordbank: L2(Redis) 未命中，回源 MySQL", "video_id", videoID)

	// 2. L2 miss: check video status first (fast path, avoids heavy JSON read)
	video, err := s.repo.GetVideoByID(ctx, videoID)
	if err != nil {
		return nil, ErrWordbankNotFound
	}

	switch video.WordbankStatus {
	case "processing":
		return nil, ErrWordbankProcessing
	case "failed", "none":
		return nil, ErrWordbankFailed
	case "ready":
		// fall through to load from DB
	default:
		return nil, ErrWordbankFailed
	}

	// 3. Load from MySQL
	wb, err := s.repo.GetVideoWordbankByVideoID(ctx, videoID)
	if err != nil {
		return nil, ErrWordbankNotFound
	}

	resp := &WordbankResponse{
		VideoID: wb.VideoID,
		Size:    wb.Size,
		Words:   json.RawMessage(wb.Words),
	}

	// 4. Write back to Redis
	data, _ := json.Marshal(resp)
	s.cache.GetRedis().Set(ctx, key, data, time.Hour)
	slog.Info("wordbank: L3(MySQL) 命中，回写 L2(Redis)", "video_id", videoID, "size", resp.Size)

	return resp, nil
}

// ExportWordbank returns the raw wordbank JSON bytes for file download.
func (s *WordbankService) ExportWordbank(ctx context.Context, videoID uint) ([]byte, string, error) {
	resp, err := s.GetWordbank(ctx, videoID)
	if err != nil {
		return nil, "", err
	}

	// Get video title for filename
	video, err := s.repo.GetVideoByID(ctx, videoID)
	filename := fmt.Sprintf("wordbank_%d.json", videoID)
	if err == nil && video.Title != "" {
		filename = fmt.Sprintf("%s_wordbank.json", video.Title)
	}

	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return nil, "", err
	}
	return data, filename, nil
}

// GetVideoSubtitle retrieves subtitle for a video (for internal use).
func (s *WordbankService) GetVideoSubtitle(ctx context.Context, videoID uint) (*models.VideoSubtitle, error) {
	key := redis.VideoSubtitle(videoID)
	if cached, err := s.cache.GetRedis().Get(ctx, key).Result(); err == nil {
		var sub models.VideoSubtitle
		if json.Unmarshal([]byte(cached), &sub) == nil {
			return &sub, nil
		}
	}

	sub, err := s.repo.GetVideoSubtitleByVideoID(ctx, videoID)
	if err != nil {
		return nil, err
	}

	data, _ := json.Marshal(sub)
	s.cache.GetRedis().Set(ctx, key, data, time.Hour)
	return sub, nil
}
