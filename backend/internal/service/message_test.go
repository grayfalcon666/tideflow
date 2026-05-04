package service

import (
	"testing"
)

func TestConversationItemStruct(t *testing.T) {
	item := &ConversationItem{
		User: &UserBrief{
			ID:        1,
			Username:  "alice",
			AvatarURL: "http://example.com/avatar.jpg",
		},
		LastMessage: &LastMessage{
			Content:   "hello",
			IsRead:    false,
		},
	}

	if item.User.ID != 1 {
		t.Errorf("UserBrief ID = %d, want 1", item.User.ID)
	}
	if item.User.Username != "alice" {
		t.Errorf("UserBrief.Username = %q, want alice", item.User.Username)
	}
	if item.LastMessage.Content != "hello" {
		t.Errorf("LastMessage.Content = %q, want hello", item.LastMessage.Content)
	}
	if item.LastMessage.IsRead != false {
		t.Errorf("LastMessage.IsRead = %v, want false", item.LastMessage.IsRead)
	}
}

func TestUserBriefStruct(t *testing.T) {
	ub := &UserBrief{
		ID:        42,
		Username:  "bob",
		AvatarURL: "http://example.com/bob.jpg",
	}

	if ub.ID != 42 {
		t.Errorf("UserBrief.ID = %d, want 42", ub.ID)
	}
	if ub.Username != "bob" {
		t.Errorf("UserBrief.Username = %q, want bob", ub.Username)
	}
	if ub.AvatarURL != "http://example.com/bob.jpg" {
		t.Errorf("UserBrief.AvatarURL = %q, want http://example.com/bob.jpg", ub.AvatarURL)
	}
}

func TestLastMessageStruct(t *testing.T) {
	lm := &LastMessage{
		Content: "test message",
		IsRead:  true,
	}

	if lm.Content != "test message" {
		t.Errorf("LastMessage.Content = %q, want test message", lm.Content)
	}
	if lm.IsRead != true {
		t.Errorf("LastMessage.IsRead = %v, want true", lm.IsRead)
	}
}
