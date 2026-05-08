package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"tideflow/internal/models"
	"tideflow/internal/mq"
	"tideflow/internal/repository"
)

var (
	ErrAlreadyLiked    = errors.New("already liked")
	ErrLikeNotExists   = errors.New("like not exists")
)

type InteractionService struct {
	repo *repository.Repository
	mq   *mq.MQ
}

func NewInteractionService(repo *repository.Repository, mqInstance *mq.MQ) *InteractionService {
	return &InteractionService{repo: repo, mq: mqInstance}
}

func (s *InteractionService) LikeVideo(ctx context.Context, userID, videoID uint) error {
	// 检查视频是否存在
	if _, err := s.repo.GetVideoByID(ctx, videoID); err != nil {
		return fmt.Errorf("video %d not found", videoID)
	}

	// ON DUPLICATE KEY UPDATE: 仅当真正插入新记录(status 0→1)时才发 MQ
	inserted, err := s.repo.UpsertLike(ctx, videoID, userID)
	if err != nil {
		return err
	}
	if !inserted {
		return ErrAlreadyLiked
	}

	// 发布点赞事件，LikeWorker 消费：更新 likes_count
	event := mq.LikeEvent{
		EventID:    fmt.Sprintf("%d-%d-%d", videoID, userID, time.Now().UnixNano()),
		Action:     "like",
		UserID:     userID,
		VideoID:    videoID,
		OccurredAt: time.Now().UnixMilli(),
	}
	s.mq.Publish(ctx, "like.events", "like.like", event)

	// 发布热度增量事件，由 PopularityWorker 统一更新 Redis 热门窗口
	popEvent := mq.PopularityEvent{
		EventID:   fmt.Sprintf("%d-%d", videoID, time.Now().UnixNano()),
		VideoID:   videoID,
		Change:    1,
		OccurredAt: time.Now().UnixMilli(),
	}
	s.mq.Publish(ctx, "video.popularity.events", "video.popularity.update", popEvent)

	return nil
}

func (s *InteractionService) UnlikeVideo(ctx context.Context, userID, videoID uint) error {
	// 检查视频是否存在
	if _, err := s.repo.GetVideoByID(ctx, videoID); err != nil {
		return fmt.Errorf("video %d not found", videoID)
	}
	// ON DUPLICATE KEY UPDATE: 仅当真正变更(status 1→0)时才发 MQ
	changed, err := s.repo.UpdateLikeStatus(ctx, videoID, userID, 0)
	if err != nil {
		return err
	}
	if !changed {
		return ErrLikeNotExists
	}

	// 发布取消点赞事件，LikeWorker 消费：更新 likes_count
	event := mq.LikeEvent{
		EventID:    fmt.Sprintf("%d-%d-%d", videoID, userID, time.Now().UnixNano()),
		Action:     "unlike",
		UserID:     userID,
		VideoID:    videoID,
		OccurredAt: time.Now().UnixMilli(),
	}
	s.mq.Publish(ctx, "like.events", "like.unlike", event)

	// 发布热度增量事件（负值），由 PopularityWorker 统一更新 Redis 热门窗口
	popEvent := mq.PopularityEvent{
		EventID:   fmt.Sprintf("%d-%d", videoID, time.Now().UnixNano()),
		VideoID:   videoID,
		Change:    -1,
		OccurredAt: time.Now().UnixMilli(),
	}
	s.mq.Publish(ctx, "video.popularity.events", "video.popularity.update", popEvent)

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

func (s *InteractionService) PublishComment(ctx context.Context, videoID, authorID uint, username, content string, parentID, rootID uint) (*models.Comment, error) {
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
		return nil, err
	}

	// 发布评论事件，CommentWorker 消费：写入 comments 表
	event := mq.CommentEvent{
		EventID:   fmt.Sprintf("%d-%d", comment.ID, time.Now().UnixNano()),
		Action:    "publish",
		CommentID: comment.ID,
		Username:  username,
		VideoID:   videoID,
		AuthorID:  authorID,
		Content:   content,
		OccurredAt: time.Now().UnixMilli(),
	}
	s.mq.Publish(ctx, "comment.events", "comment.publish", event)

	// 发布热度增量事件（评论权重 +5），由 PopularityWorker 统一更新 Redis 热门窗口
	popEvent := mq.PopularityEvent{
		EventID:   fmt.Sprintf("%d-%d", videoID, time.Now().UnixNano()),
		VideoID:   videoID,
		Change:    5,
		OccurredAt: time.Now().UnixMilli(),
	}
	s.mq.Publish(ctx, "video.popularity.events", "video.popularity.update", popEvent)

	return comment, nil
}

func (s *InteractionService) DeleteComment(ctx context.Context, commentID, authorID uint) error {
	comment, err := s.repo.GetComment(ctx, commentID)
	if err != nil {
		return err
	}
	if comment.AuthorID != authorID {
		return err
	}

	// 发布评论删除事件，CommentWorker 消费：软删除
	event := mq.CommentEvent{
		EventID:   fmt.Sprintf("%d-%d", commentID, time.Now().UnixNano()),
		Action:    "delete",
		CommentID: commentID,
		VideoID:   comment.VideoID,
		AuthorID:  authorID,
		OccurredAt: time.Now().UnixMilli(),
	}
	s.mq.Publish(ctx, "comment.events", "comment.delete", event)

	// 发布热度增量事件（评论权重 -5），由 PopularityWorker 统一更新 Redis 热门窗口
	popEvent := mq.PopularityEvent{
		EventID:   fmt.Sprintf("%d-%d", comment.VideoID, time.Now().UnixNano()),
		VideoID:   comment.VideoID,
		Change:    -5,
		OccurredAt: time.Now().UnixMilli(),
	}
	s.mq.Publish(ctx, "video.popularity.events", "video.popularity.update", popEvent)

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

	// root_id=0 时一次性加载所有根评论的子评论，嵌入到 Replies 字段
	if rootID == 0 && len(comments) > 0 {
		rootIDs := make([]uint, len(comments))
		for i, c := range comments {
			rootIDs[i] = c.ID
		}
		replies, _ := s.repo.GetRepliesByRootIDs(ctx, rootIDs)
		replyMap := make(map[uint][]*CommentWithReplies)
		for _, r := range replies {
			replyMap[r.RootID] = append(replyMap[r.RootID], &CommentWithReplies{
				ID:         r.ID,
				AuthorID:   r.AuthorID,
				Username:   r.Username,
				Content:    r.Content,
				ParentID:   r.ParentID,
				RootID:     r.RootID,
				CreatedAt:  r.CreatedAt,
				ReplyCount: 0,
			})
		}
		for _, c := range result {
			if repls, ok := replyMap[c.ID]; ok {
				c.Replies = repls
			}
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
	ID         uint                `json:"id"`
	AuthorID   uint                `json:"author_id"`
	Username   string              `json:"username"`
	Content    string              `json:"content"`
	ParentID   uint                `json:"parent_id"`
	RootID     uint                `json:"root_id"`
	CreatedAt  time.Time          `json:"created_at"`
	ReplyCount int                `json:"reply_count"`
	Replies    []*CommentWithReplies `json:"replies,omitempty"`
}