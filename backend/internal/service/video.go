package service

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"tideflow/internal/models"
	"tideflow/internal/repository"
	infraredis "tideflow/infra/redis"
)

var ErrVideoNotFound = errors.New("video not found")

type VideoService struct {
	repo     *repository.Repository
	cache    *infraredis.Cache
	bigVThresh int
	uploadDir string
}

func NewVideoService(repo *repository.Repository, cache *infraredis.Cache, bigVThresh int, uploadDir string) *VideoService {
	return &VideoService{repo: repo, cache: cache, bigVThresh: bigVThresh, uploadDir: uploadDir}
}

func (s *VideoService) UploadVideo(ctx context.Context, file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	filename := filepath.Join(s.uploadDir, "videos", generateFilename(file.Filename))
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return "", err
	}

	dst, err := os.Create(filename)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	return "/videos/" + filepath.Base(filename), nil
}

func (s *VideoService) UploadCover(ctx context.Context, file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	filename := filepath.Join(s.uploadDir, "covers", generateFilename(file.Filename))
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return "", err
	}

	dst, err := os.Create(filename)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	return "/covers/" + filepath.Base(filename), nil
}

func (s *VideoService) PublishVideo(ctx context.Context, authorID uint, username, title, description, playURL, coverURL string, tags []string) (uint, error) {
	video := &models.Video{
		AuthorID:   authorID,
		Username:   username,
		Title:      title,
		Description: description,
		PlayURL:    playURL,
		CoverURL:   coverURL,
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
	}
	if err := s.repo.CreateVideo(ctx, video); err != nil {
		return 0, err
	}

	for _, tagName := range tags {
		tag, err := s.repo.GetOrCreateTag(ctx, tagName)
		if err != nil {
			continue
		}
		s.repo.CreateVideoTag(ctx, video.ID, tag.ID)
	}

	outbox := &models.OutboxMsg{
		VideoID:    video.ID,
		EventType:  "video_publish",
		CreateTime: video.CreateTime,
		Status:     "pending",
	}
	s.repo.CreateOutboxMsg(ctx, outbox)

	return video.ID, nil
}

func (s *VideoService) GetVideoByID(ctx context.Context, id uint) (*models.Video, error) {
	video, err := s.repo.GetVideoByID(ctx, id)
	if err != nil {
		return nil, ErrVideoNotFound
	}
	return video, nil
}

func (s *VideoService) GetVideosByIDs(ctx context.Context, ids []uint) ([]*models.Video, error) {
	return s.repo.GetVideosByIDs(ctx, ids)
}

func (s *VideoService) UpdateVideo(ctx context.Context, id uint, authorID uint, updates map[string]interface{}) error {
	video, err := s.repo.GetVideoByID(ctx, id)
	if err != nil {
		return ErrVideoNotFound
	}
	if video.AuthorID != authorID {
		return errors.New("forbidden")
	}
	updates["update_time"] = time.Now()
	return s.repo.UpdateVideo(ctx, id, updates)
}

func (s *VideoService) DeleteVideo(ctx context.Context, id uint, authorID uint) error {
	video, err := s.repo.GetVideoByID(ctx, id)
	if err != nil {
		return ErrVideoNotFound
	}
	if video.AuthorID != authorID {
		return errors.New("forbidden")
	}
	return s.repo.SoftDeleteVideo(ctx, id)
}

func (s *VideoService) GetVideoTags(ctx context.Context, videoID uint) ([]string, error) {
	return s.repo.GetVideoTags(ctx, videoID)
}

func (s *VideoService) UpdatePopularity(ctx context.Context, videoID uint, change int64) error {
	return s.repo.UpdateVideo(ctx, videoID, map[string]interface{}{
		"popularity": s.repo.DB().Raw("SELECT popularity + ? FROM videos WHERE id = ?", change, videoID),
	})
}

func generateFilename(orig string) string {
	return time.Now().Format("20060102150405") + "_" + orig
}