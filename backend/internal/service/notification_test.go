package service

import (
	"testing"
	"time"

	"tideflow/internal/models"
)

func TestNotificationItemStruct(t *testing.T) {
	now := time.Now()
	item := &NotificationItem{
		ID: 1,
		Sender: &NotificationUserBrief{
			ID:        10,
			Username:  "alice",
			AvatarURL: "http://example.com/alice.jpg",
		},
		Type:       "like",
		TargetID:   100,
		Content:    "liked your video",
		IsRead:     false,
		CreatedAt:  now,
	}

	if item.ID != 1 {
		t.Errorf("NotificationItem.ID = %d, want 1", item.ID)
	}
	if item.Sender.Username != "alice" {
		t.Errorf("NotificationUserBrief.Username = %q, want alice", item.Sender.Username)
	}
	if item.Type != "like" {
		t.Errorf("NotificationItem.Type = %q, want like", item.Type)
	}
	if item.IsRead != false {
		t.Errorf("NotificationItem.IsRead = %v, want false", item.IsRead)
	}
}

func TestNotificationUserBriefStruct(t *testing.T) {
	nub := &NotificationUserBrief{
		ID:        5,
		Username:  "bob",
		AvatarURL: "http://example.com/bob.jpg",
	}

	if nub.ID != 5 {
		t.Errorf("NotificationUserBrief.ID = %d, want 5", nub.ID)
	}
	if nub.Username != "bob" {
		t.Errorf("NotificationUserBrief.Username = %q, want bob", nub.Username)
	}
}

func TestNotificationModel(t *testing.T) {
	notif := &models.Notification{
		RecipientID: 1,
		SenderID:    2,
		Type:        "comment",
		TargetID:    10,
		Content:     "nice video",
		IsRead:      false,
		CreatedAt:   time.Now(),
	}

	if notif.RecipientID != 1 {
		t.Errorf("Notification.RecipientID = %d, want 1", notif.RecipientID)
	}
	if notif.Type != "comment" {
		t.Errorf("Notification.Type = %q, want comment", notif.Type)
	}
	if notif.IsRead != false {
		t.Errorf("Notification.IsRead = %v, want false", notif.IsRead)
	}
}

func TestNotificationItemZeroCase(t *testing.T) {
	// Empty notification item - zero values
	item := &NotificationItem{}
	if item.ID != 0 {
		t.Errorf("zero NotificationItem.ID = %d, want 0", item.ID)
	}
	if item.Sender != nil {
		t.Errorf("zero NotificationItem.Sender = %v, want nil", item.Sender)
	}
}
