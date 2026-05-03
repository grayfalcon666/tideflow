package service

import (
	"context"
	"strconv"
	"time"

	"tideflow/internal/models"
	"tideflow/internal/repository"
)

type MessageService struct {
	repo *repository.Repository
}

func NewMessageService(repo *repository.Repository) *MessageService {
	return &MessageService{repo: repo}
}

func (s *MessageService) SendMessage(ctx context.Context, fromID, toID uint, content string) (uint, error) {
	msg := &models.Message{
		FromID:    fromID,
		ToID:      toID,
		Content:   content,
		IsRead:    false,
		CreatedAt: time.Now(),
	}
	if err := s.repo.CreateMessage(ctx, msg); err != nil {
		return 0, err
	}
	return msg.ID, nil
}

func (s *MessageService) GetConversations(ctx context.Context, userID uint) ([]*ConversationItem, error) {
	msgs, err := s.repo.GetConversations(ctx, userID)
	if err != nil {
		return nil, err
	}

	items := make([]*ConversationItem, 0, len(msgs))
	for _, msg := range msgs {
		var otherID uint
		if msg.FromID == userID {
			otherID = msg.ToID
		} else {
			otherID = msg.FromID
		}

		acc, err := s.repo.GetAccountByID(ctx, otherID)
		if err != nil {
			continue
		}

		items = append(items, &ConversationItem{
			User: &UserBrief{
				ID:         acc.ID,
				Username:   acc.Username,
				AvatarURL:  acc.AvatarURL,
			},
			LastMessage: &LastMessage{
				Content:   msg.Content,
				CreatedAt: msg.CreatedAt,
				IsRead:   msg.IsRead,
			},
		})
	}

	return items, nil
}

func (s *MessageService) GetMessages(ctx context.Context, userID, otherID uint, cursor string, limit int) ([]*models.Message, *string, bool, error) {
	var before time.Time
	if cursor != "" {
		ts, err := strconv.ParseInt(cursor, 10, 64)
		if err == nil {
			before = time.UnixMilli(ts)
		}
	}

	msgs, err := s.repo.GetMessages(ctx, userID, otherID, before, limit+1)
	if err != nil {
		return nil, nil, false, err
	}

	hasMore := len(msgs) > limit
	if hasMore {
		msgs = msgs[:limit]
	}

	var nextCursor *string
	if hasMore && len(msgs) > 0 {
		nc := strconv.FormatInt(msgs[len(msgs)-1].CreatedAt.UnixMilli(), 10)
		nextCursor = &nc
	}

	return msgs, nextCursor, hasMore, nil
}

func (s *MessageService) MarkRead(ctx context.Context, userID, otherID uint) error {
	return s.repo.MarkMessagesRead(ctx, otherID, userID)
}

type ConversationItem struct {
	User        *UserBrief   `json:"user"`
	LastMessage *LastMessage `json:"last_message"`
}

type UserBrief struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
}

type LastMessage struct {
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	IsRead    bool      `json:"is_read"`
}