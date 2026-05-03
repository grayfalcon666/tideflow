package service

import (
	"context"
	"strconv"
	"time"

	"tideflow/internal/models"
	"tideflow/internal/repository"
)

type InteractionService struct {
	repo *repository.Repository
}

func NewInteractionService(repo *repository.Repository) *InteractionService {
	return &InteractionService{repo: repo}
}

func (s *InteractionService) LikeVideo(ctx context.Context, userID, videoID uint) error {
	_, err := s.repo.GetLike(ctx, videoID, userID)
	if err == nil {
		return nil
	}

	like := &models.Like{VideoID: videoID, AccountID: userID, CreatedAt: time.Now()}
	if err := s.repo.CreateLike(ctx, like); err != nil {
		return err
	}

	s.repo.UpdateVideo(ctx, videoID, map[string]interface{}{
		"likes_count": s.repo.DB().Raw("SELECT likes_count + 1 FROM videos WHERE id = ?", videoID),
	})

	return nil
}

func (s *InteractionService) UnlikeVideo(ctx context.Context, userID, videoID uint) error {
	if err := s.repo.DeleteLike(ctx, videoID, userID); err != nil {
		return err
	}

	s.repo.UpdateVideo(ctx, videoID, map[string]interface{}{
		"likes_count": s.repo.DB().Raw("SELECT likes_count - 1 FROM videos WHERE id = ?", videoID),
	})

	return nil
}

func (s *InteractionService) IsLiked(ctx context.Context, userID, videoID uint) (bool, error) {
	_, err := s.repo.GetLike(ctx, videoID, userID)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (s *InteractionService) GetLikedVideos(ctx context.Context, userID uint, cursor string, limit int) ([]*models.Video, *string, bool, error) {
	var before time.Time
	if cursor != "" {
		ts, err := parseCursorInt64(cursor)
		if err == nil {
			before = time.UnixMilli(ts)
		}
	}

	likes, err := s.repo.GetLikesByAccount(ctx, userID, before, limit+1)
	if err != nil {
		return nil, nil, false, err
	}

	hasMore := len(likes) > limit
	if hasMore {
		likes = likes[:limit]
	}

	videoIDs := make([]uint, len(likes))
	for i, l := range likes {
		videoIDs[i] = l.VideoID
	}

	videos, err := s.repo.GetVideosByIDs(ctx, videoIDs)
	if err != nil {
		return nil, nil, false, err
	}

	var nextCursor *string
	if hasMore && len(likes) > 0 {
		nc := strconv.FormatInt(likes[len(likes)-1].CreatedAt.UnixMilli(), 10)
		nextCursor = &nc
	}

	return videos, nextCursor, hasMore, nil
}

func parseCursorInt64(s string) (int64, error) {
	var v int64
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, nil
		}
		v = v*10 + int64(s[i]-'0')
	}
	return v, nil
}

func (s *InteractionService) PublishComment(ctx context.Context, videoID, authorID uint, username, content string, parentID, rootID uint) (uint, error) {
	comment := &models.Comment{
		VideoID:   videoID,
		AuthorID:  authorID,
		Username:  username,
		Content:   content,
		ParentID:  parentID,
		RootID:    rootID,
		CreatedAt: time.Now(),
	}
	if err := s.repo.CreateComment(ctx, comment); err != nil {
		return 0, err
	}

	s.repo.UpdateVideo(ctx, videoID, map[string]interface{}{
		"popularity": s.repo.DB().Raw("SELECT popularity + 5 FROM videos WHERE id = ?", videoID),
	})

	return comment.ID, nil
}

func (s *InteractionService) DeleteComment(ctx context.Context, commentID, authorID uint) error {
	comment, err := s.repo.GetComment(ctx, commentID)
	if err != nil {
		return err
	}
	if comment.AuthorID != authorID {
		return err
	}
	return s.repo.SoftDeleteComment(ctx, commentID)
}

func (s *InteractionService) GetComments(ctx context.Context, videoID uint, rootID uint, cursor string, limit int) ([]*CommentWithReplies, *string, bool, error) {
	var before time.Time
	if cursor != "" {
		ts, err := parseCursorInt64(cursor)
		if err == nil {
			before = time.UnixMilli(ts)
		}
	}

	comments, err := s.repo.GetCommentsByVideo(ctx, videoID, rootID, before, limit+1)
	if err != nil {
		return nil, nil, false, err
	}

	hasMore := len(comments) > limit
	if hasMore {
		comments = comments[:limit]
	}

	result := make([]*CommentWithReplies, len(comments))
	for i, c := range comments {
		replyCount, _ := s.repo.CountReplies(ctx, c.ID)
		result[i] = &CommentWithReplies{
			ID:         c.ID,
			AuthorID:   c.AuthorID,
			Username:   c.Username,
			Content:    c.Content,
			ParentID:   c.ParentID,
			RootID:     c.RootID,
			CreatedAt:  c.CreatedAt,
			ReplyCount: int(replyCount),
		}
	}

	var nextCursor *string
	if hasMore && len(comments) > 0 {
		nc := strconv.FormatInt(comments[len(comments)-1].CreatedAt.UnixMilli(), 10)
		nextCursor = &nc
	}

	return result, nextCursor, hasMore, nil
}

type CommentWithReplies struct {
	ID         uint      `json:"id"`
	AuthorID   uint      `json:"author_id"`
	Username   string    `json:"username"`
	Content    string    `json:"content"`
	ParentID   uint      `json:"parent_id"`
	RootID     uint      `json:"root_id"`
	CreatedAt  time.Time `json:"created_at"`
	ReplyCount int       `json:"reply_count"`
}