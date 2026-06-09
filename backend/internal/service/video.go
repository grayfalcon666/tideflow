package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"tideflow/internal/models"
	"tideflow/internal/mq"
	"tideflow/internal/repository"
	"tideflow/pkg/media"
	infraredis "tideflow/infra/redis"
)

var ErrVideoNotFound = errors.New("video not found")

// UploadSession tracks a chunked upload in progress.
type UploadSession struct {
	UploadID       string `json:"upload_id"`
	Filename       string `json:"filename"`
	TotalChunks    int    `json:"total_chunks"`
	ChunkSize      int64  `json:"chunk_size"`
	FileSize       int64  `json:"file_size"`
	UploadedChunks []int  `json:"uploaded_chunks"`
	CreatedAt      int64  `json:"created_at"`
}

const (
	uploadSessionTTL  = 24 * time.Hour
	defaultChunkSize  = 5 * 1024 * 1024   // 5MB
	maxUploadFileSize = 500 * 1024 * 1024 // 500MB
)

type VideoService struct {
	repo       *repository.Repository
	cache      *infraredis.Cache
	bigVThresh int
	UploadDir  string
	mq         *mq.MQ
}

func NewVideoService(repo *repository.Repository, cache *infraredis.Cache, bigVThresh int, uploadDir string, mqInstance *mq.MQ) *VideoService {
	return &VideoService{repo: repo, cache: cache, bigVThresh: bigVThresh, UploadDir: uploadDir, mq: mqInstance}
}

func (s *VideoService) UploadVideo(ctx context.Context, file *multipart.FileHeader) (string, *media.VideoMeta, error) {
	src, err := file.Open()
	if err != nil {
		return "", nil, err
	}
	defer src.Close()

	filename := filepath.Join(s.UploadDir, "videos", generateFilename(file.Filename))
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return "", nil, err
	}

	dst, err := os.Create(filename)
	if err != nil {
		return "", nil, err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", nil, err
	}

	meta, err := media.ExtractVideoMeta(filename)
	if err != nil || meta == nil {
		meta = &media.VideoMeta{}
	}

	return "/videos/" + filepath.Base(filename), meta, nil
}

func (s *VideoService) UploadCover(ctx context.Context, file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	filename := filepath.Join(s.UploadDir, "covers", generateFilename(file.Filename))
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

func (s *VideoService) PublishVideo(ctx context.Context, authorID uint, username, title, description, playURL, coverURL string, tags []string, meta *media.VideoMeta) (uint, error) {
	video := &models.Video{
		AuthorID:    authorID,
		Username:    username,
		Title:       title,
		Description: description,
		PlayURL:     playURL,
		CoverURL:    coverURL,
		Duration:    meta.Duration,
		Width:       meta.Width,
		Height:      meta.Height,
		CreateTime:  time.Now(),
		UpdateTime:  time.Now(),
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
		AuthorID:   authorID,
		EventType:  "video_publish",
		CreateTime: video.CreateTime,
		Status:     "pending",
	}
	s.repo.CreateOutboxMsg(ctx, outbox)

	// Initialize Redis view count for the new video
	s.InitViewCount(ctx, video.ID)

	return video.ID, nil
}

func (s *VideoService) GetVideoByIDUnscoped(ctx context.Context, id uint) (*models.Video, error) {
	return s.repo.GetVideoByIDUnscoped(ctx, id)
}

func (s *VideoService) GetVideoByID(ctx context.Context, id uint) (*models.Video, error) {
	video, err := s.cache.GetVideoDetail(ctx, id, func(ctx context.Context, id uint) (*models.Video, error) {
		return s.repo.GetVideoByID(ctx, id)
	})
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
	if err := s.repo.UpdateVideo(ctx, id, updates); err != nil {
		return err
	}
	// 写入 outbox_msgs，触发 ES 搜索索引更新
	outbox := &models.OutboxMsg{
		VideoID:    id,
		AuthorID:   video.AuthorID,
		EventType:  "video_upsert",
		CreateTime: time.Now(),
		Status:     "pending",
	}
	s.repo.CreateOutboxMsg(ctx, outbox)
	return nil
}

func (s *VideoService) DeleteVideo(ctx context.Context, id uint, authorID uint) error {
	video, err := s.repo.GetVideoByID(ctx, id)
	if err != nil {
		return ErrVideoNotFound
	}
	if video.AuthorID != authorID {
		return errors.New("forbidden")
	}

	// 软删除关联评论
	if err := s.repo.SoftDeleteCommentsByVideo(ctx, id); err != nil {
		slog.Warn("soft delete comments failed", "video_id", id, "err", err)
	}

	// 软删除关联笔记
	if err := s.repo.SoftDeleteNotesByVideo(ctx, id); err != nil {
		slog.Warn("soft delete notes failed", "video_id", id, "err", err)
	}

	// 删除视频标签关联
	if err := s.repo.DeleteVideoTags(ctx, id); err != nil {
		slog.Warn("delete video tags failed", "video_id", id, "err", err)
	}

	// 删除物理视频文件
	if video.PlayURL != "" {
		videoPath := filepath.Join(s.UploadDir, "videos", filepath.Base(video.PlayURL))
		if err := os.Remove(videoPath); err != nil {
			if !os.IsNotExist(err) {
				slog.Warn("remove video file failed", "path", videoPath, "err", err)
			}
		} else {
			slog.Info("removed video file", "path", videoPath)
		}
	}

	// 删除物理封面文件
	if video.CoverURL != "" {
		coverPath := filepath.Join(s.UploadDir, "covers", filepath.Base(video.CoverURL))
		if err := os.Remove(coverPath); err != nil {
			if !os.IsNotExist(err) {
				slog.Warn("remove cover file failed", "path", coverPath, "err", err)
			}
		} else {
			slog.Info("removed cover file", "path", coverPath)
		}
	}

	// 软删除视频记录
	if err := s.repo.SoftDeleteVideo(ctx, id); err != nil {
		return err
	}

	// 写入 outbox_msgs，触发 ES 搜索索引删除
	outbox := &models.OutboxMsg{
		VideoID:    id,
		AuthorID:   authorID,
		EventType:  "video_delete",
		CreateTime: time.Now(),
		Status:     "pending",
	}
	s.repo.CreateOutboxMsg(ctx, outbox)

	// 删除后清理 L1/L2 缓存，防止已删除视频被拉取
	s.cache.InvalidateVideoDetail(id)
	return nil
}

func (s *VideoService) GetVideoTags(ctx context.Context, videoID uint) ([]string, error) {
	return s.repo.GetVideoTags(ctx, videoID)
}

func (s *VideoService) UpdatePopularity(ctx context.Context, videoID uint, change int64) error {
	return s.repo.UpdateVideo(ctx, videoID, map[string]interface{}{
		"popularity": s.repo.DB().Raw("SELECT popularity + ? FROM videos WHERE id = ?", change, videoID),
	})
}

func (s *VideoService) GetAccountByID(ctx context.Context, id uint) (*models.Account, error) {
	return s.repo.GetAccountByID(ctx, id)
}

const viewLimitTTL = 30 * time.Minute

func (s *VideoService) RecordView(ctx context.Context, videoID uint, userID uint) (bool, error) {
	rdb := s.cache.GetRedis()

	// 半小时限流：SETNX limit:view:{user_id}:{video_id}
	limitKey := infraredis.ViewLimit(userID, videoID)
	set, err := rdb.SetNX(ctx, limitKey, "1", viewLimitTTL).Result()
	if err != nil {
		return false, err
	}

	// 返回 0 说明半小时内已记录过，静默丢弃
	if !set {
		return false, nil
	}

	// 播放量 INCR（仅 Redis，不操作 DB）
	rdb.Incr(ctx, infraredis.ViewCount(videoID))

	// 发布热度增量事件（播放量权重 +1）
	if s.mq != nil {
		popEvent := mq.PopularityEvent{
			EventID:    fmt.Sprintf("%d-%d", videoID, time.Now().UnixNano()),
			VideoID:    videoID,
			Change:     1,
			OccurredAt: time.Now().UnixMilli(),
		}
		s.mq.Publish(ctx, "video.popularity.events", "video.popularity.update", popEvent)
	}

	return true, nil
}

// GetViewCount returns the view count for a video. Reads from Redis first; falls back to DB and backfills.
func (s *VideoService) GetViewCount(ctx context.Context, videoID uint) (int64, error) {
	rdb := s.cache.GetRedis()
	key := infraredis.ViewCount(videoID)

	val, err := rdb.Get(ctx, key).Int64()
	if err == nil {
		return val, nil
	}

	// Redis miss: read from DB and backfill
	video, err := s.repo.GetVideoByID(ctx, videoID)
	if err != nil {
		return 0, err
	}

	rdb.Set(ctx, key, video.ViewCount, 0)
	return video.ViewCount, nil
}

// InitViewCount initializes the Redis view count for a newly published video.
func (s *VideoService) InitViewCount(ctx context.Context, videoID uint) {
	rdb := s.cache.GetRedis()
	rdb.Set(ctx, infraredis.ViewCount(videoID), 0, 0)
}

func generateFilename(orig string) string {
	return time.Now().Format("20060102150405") + "_" + orig
}

// InitChunkedUpload creates an upload session and returns its upload_id.
func (s *VideoService) InitChunkedUpload(ctx context.Context, filename string, fileSize int64, chunkSize int64) (string, error) {
	if fileSize <= 0 || fileSize > maxUploadFileSize {
		return "", fmt.Errorf("invalid file size: %d", fileSize)
	}
	if chunkSize <= 0 {
		chunkSize = defaultChunkSize
	}

	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	uploadID := hex.EncodeToString(b)

	totalChunks := int((fileSize + chunkSize - 1) / chunkSize)

	chunksDir := filepath.Join(s.UploadDir, "chunks", uploadID)
	if err := os.MkdirAll(chunksDir, 0755); err != nil {
		return "", err
	}

	session := UploadSession{
		UploadID:       uploadID,
		Filename:       filename,
		TotalChunks:    totalChunks,
		ChunkSize:      chunkSize,
		FileSize:       fileSize,
		UploadedChunks: []int{},
		CreatedAt:      time.Now().Unix(),
	}

	data, err := json.Marshal(session)
	if err != nil {
		return "", err
	}

	rdb := s.cache.GetRedis()
	key := infraredis.UploadSession(uploadID)
	if err := rdb.Set(ctx, key, data, uploadSessionTTL).Err(); err != nil {
		return "", err
	}
	// Initialize empty chunk set for atomic SADD tracking.
	rdb.Expire(ctx, infraredis.UploadChunks(uploadID), uploadSessionTTL)

	return uploadID, nil
}

// UploadChunk writes a single chunk to disk and updates the session.
func (s *VideoService) UploadChunk(ctx context.Context, uploadID string, chunkIndex int, reader io.Reader) error {
	session, err := s.getUploadSession(ctx, uploadID)
	if err != nil {
		return fmt.Errorf("invalid upload session")
	}

	if chunkIndex < 0 || chunkIndex >= session.TotalChunks {
		return fmt.Errorf("invalid chunk index: %d", chunkIndex)
	}

	chunkPath := filepath.Join(s.UploadDir, "chunks", uploadID, fmt.Sprintf("chunk_%d", chunkIndex))
	dst, err := os.Create(chunkPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, reader); err != nil {
		os.Remove(chunkPath)
		return err
	}

	// Use SADD for atomic add — avoids read-modify-write race with concurrent chunks.
	rdb := s.cache.GetRedis()
	chunksKey := infraredis.UploadChunks(uploadID)
	if err := rdb.SAdd(ctx, chunksKey, chunkIndex).Err(); err != nil {
		return err
	}
	rdb.Expire(ctx, chunksKey, uploadSessionTTL)

	return nil
}

// GetUploadStatus returns the current upload session state.
func (s *VideoService) GetUploadStatus(ctx context.Context, uploadID string) (*UploadSession, error) {
	return s.getUploadSession(ctx, uploadID)
}

// CompleteChunkedUpload merges all chunks, extracts meta, and cleans up.
func (s *VideoService) CompleteChunkedUpload(ctx context.Context, uploadID string) (string, *media.VideoMeta, error) {
	session, err := s.getUploadSession(ctx, uploadID)
	if err != nil {
		return "", nil, fmt.Errorf("invalid upload session: %w", err)
	}

	chunksDir := filepath.Join(s.UploadDir, "chunks", uploadID)

	// Check actual uploaded chunk count from Redis Set for atomic accuracy.
	chunksKey := infraredis.UploadChunks(uploadID)
	rdb := s.cache.GetRedis()
	uploadedCount, _ := rdb.SCard(ctx, chunksKey).Result()
	if int(uploadedCount) != session.TotalChunks {
		missing := session.TotalChunks - int(uploadedCount)
		return "", nil, fmt.Errorf("missing %d chunks", missing)
	}

	finalFilename := generateFilename(session.Filename)
	finalPath := filepath.Join(s.UploadDir, "videos", finalFilename)
	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		return "", nil, err
	}

	dst, err := os.Create(finalPath)
	if err != nil {
		return "", nil, err
	}
	defer dst.Close()

	for i := 0; i < session.TotalChunks; i++ {
		chunkPath := filepath.Join(chunksDir, fmt.Sprintf("chunk_%d", i))
		src, err := os.Open(chunkPath)
		if err != nil {
			return "", nil, fmt.Errorf("failed to read chunk %d: %w", i, err)
		}
		if _, err := io.Copy(dst, src); err != nil {
			src.Close()
			return "", nil, fmt.Errorf("failed to merge chunk %d: %w", i, err)
		}
		src.Close()
	}

	meta, err := media.ExtractVideoMeta(finalPath)
	if err != nil || meta == nil {
		meta = &media.VideoMeta{}
	}

	os.RemoveAll(chunksDir)
	rdb.Del(ctx, infraredis.UploadSession(uploadID))
	rdb.Del(ctx, infraredis.UploadChunks(uploadID))

	return "/videos/" + filepath.Base(finalFilename), meta, nil
}

func (s *VideoService) getUploadSession(ctx context.Context, uploadID string) (*UploadSession, error) {
	rdb := s.cache.GetRedis()
	key := infraredis.UploadSession(uploadID)
	data, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var session UploadSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, err
	}
	return &session, nil
}