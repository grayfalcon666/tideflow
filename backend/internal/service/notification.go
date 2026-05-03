package service

import (
	"context"
	"strconv"
	"time"

	"tideflow/internal/models"
	"tideflow/internal/repository"
)

type NotificationService struct {
	repo *repository.Repository
}

func NewNotificationService(repo *repository.Repository) *NotificationService {
	return &NotificationService{repo: repo}
}

func (s *NotificationService) CreateNotification(ctx context.Context, recipientID, senderID uint, notifType string, targetID uint, content string) error {
	notif := &models.Notification{
		RecipientID: recipientID,
		SenderID:    senderID,
		Type:        notifType,
		TargetID:    targetID,
		Content:     content,
		IsRead:      false,
		CreatedAt:   time.Now(),
	}
	return s.repo.CreateNotification(ctx, notif)
}

func (s *NotificationService) GetNotifications(ctx context.Context, userID uint, cursor string, limit int) ([]*NotificationItem, *string, bool, error) {
	var before time.Time
	if cursor != "" {
		ts, err := strconv.ParseInt(cursor, 10, 64)
		if err == nil {
			before = time.UnixMilli(ts)
		}
	}

	notifs, err := s.repo.GetNotifications(ctx, userID, before, limit+1)
	if err != nil {
		return nil, nil, false, err
	}

	hasMore := len(notifs) > limit
	if hasMore {
		notifs = notifs[:limit]
	}

	if len(notifs) == 0 {
		return []*NotificationItem{}, nil, false, nil
	}

	senderIDs := make([]uint, len(notifs))
	for i, n := range notifs {
		senderIDs[i] = n.SenderID
	}

	senders, _ := s.repo.GetSendersByNotificationIDs(ctx, senderIDs)

	items := make([]*NotificationItem, len(notifs))
	for i, n := range notifs {
		sender := senders[n.SenderID]
		var senderBrief *NotificationUserBrief
		if sender != nil {
			senderBrief = &NotificationUserBrief{
				ID:        sender.ID,
				Username:  sender.Username,
				AvatarURL: sender.AvatarURL,
			}
		}
		items[i] = &NotificationItem{
			ID:      n.ID,
			Sender:  senderBrief,
			Type:    n.Type,
			TargetID: n.TargetID,
			Content: n.Content,
			IsRead:  n.IsRead,
			CreatedAt: n.CreatedAt,
		}
	}

	var nextCursor *string
	if hasMore && len(notifs) > 0 {
		nc := strconv.FormatInt(notifs[len(notifs)-1].CreatedAt.UnixMilli(), 10)
		nextCursor = &nc
	}

	return items, nextCursor, hasMore, nil
}

func (s *NotificationService) MarkRead(ctx context.Context, userID uint, ids []uint) error {
	return s.repo.MarkNotificationsRead(ctx, userID, ids)
}

func (s *NotificationService) CountUnread(ctx context.Context, userID uint) (int64, error) {
	return s.repo.CountUnreadNotifications(ctx, userID)
}

type NotificationItem struct {
	ID        uint       `json:"id"`
	Sender    *NotificationUserBrief `json:"sender"`
	Type      string     `json:"type"`
	TargetID  uint       `json:"target_id"`
	Content   string     `json:"content"`
	IsRead    bool       `json:"is_read"`
	CreatedAt time.Time  `json:"created_at"`
}

type NotificationUserBrief struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
}