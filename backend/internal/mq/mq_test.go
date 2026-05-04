package mq

import (
	"encoding/json"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestParseVideoPublishEvent(t *testing.T) {
	e := VideoPublishEvent{
		EventID:    "123",
		VideoID:   456,
		AuthorID:  789,
		CreateTime: 1715000000000,
		OccurredAt: 1715000000001,
	}

	data, _ := json.Marshal(e)
	delivery := amqp.Delivery{Body: data}

	parsed, err := ParseVideoPublishEvent(delivery)
	if err != nil {
		t.Fatalf("ParseVideoPublishEvent failed: %v", err)
	}
	if parsed.EventID != "123" {
		t.Errorf("EventID = %q, want 123", parsed.EventID)
	}
	if parsed.VideoID != 456 {
		t.Errorf("VideoID = %d, want 456", parsed.VideoID)
	}
	if parsed.AuthorID != 789 {
		t.Errorf("AuthorID = %d, want 789", parsed.AuthorID)
	}
}

func TestParseVideoPublishEventInvalid(t *testing.T) {
	delivery := amqp.Delivery{Body: []byte("not json")}
	_, err := ParseVideoPublishEvent(delivery)
	if err == nil {
		t.Error("ParseVideoPublishEvent should fail on invalid JSON")
	}
}

func TestParseLikeEvent(t *testing.T) {
	e := LikeEvent{
		EventID:   "like-1",
		Action:    "like",
		UserID:    100,
		VideoID:   200,
		OccurredAt: 1715000000000,
	}

	data, _ := json.Marshal(e)
	delivery := amqp.Delivery{Body: data}

	parsed, err := ParseLikeEvent(delivery)
	if err != nil {
		t.Fatalf("ParseLikeEvent failed: %v", err)
	}
	if parsed.Action != "like" {
		t.Errorf("Action = %q, want like", parsed.Action)
	}
	if parsed.UserID != 100 {
		t.Errorf("UserID = %d, want 100", parsed.UserID)
	}
}

func TestParseCommentEvent(t *testing.T) {
	e := CommentEvent{
		EventID:   "c1",
		Action:    "publish",
		CommentID: 10,
		Username:  "alice",
		VideoID:   200,
		AuthorID:  100,
		Content:   "great video",
		OccurredAt: 1715000000000,
	}

	data, _ := json.Marshal(e)
	parsed, err := ParseCommentEvent(amqp.Delivery{Body: data})
	if err != nil {
		t.Fatalf("ParseCommentEvent failed: %v", err)
	}
	if parsed.Content != "great video" {
		t.Errorf("Content = %q, want great video", parsed.Content)
	}
	if parsed.Action != "publish" {
		t.Errorf("Action = %q, want publish", parsed.Action)
	}
}

func TestParseSocialEvent(t *testing.T) {
	e := SocialEvent{
		EventID:    "s1",
		Action:     "follow",
		FollowerID: 1,
		VloggerID:  2,
		OccurredAt: 1715000000000,
	}

	data, _ := json.Marshal(e)
	parsed, err := ParseSocialEvent(amqp.Delivery{Body: data})
	if err != nil {
		t.Fatalf("ParseSocialEvent failed: %v", err)
	}
	if parsed.Action != "follow" {
		t.Errorf("Action = %q, want follow", parsed.Action)
	}
	if parsed.VloggerID != 2 {
		t.Errorf("VloggerID = %d, want 2", parsed.VloggerID)
	}
}

func TestParsePopularityEvent(t *testing.T) {
	e := PopularityEvent{
		EventID:   "p1",
		VideoID:   300,
		Change:    5,
		OccurredAt: 1715000000000,
	}

	data, _ := json.Marshal(e)
	parsed, err := ParsePopularityEvent(amqp.Delivery{Body: data})
	if err != nil {
		t.Fatalf("ParsePopularityEvent failed: %v", err)
	}
	if parsed.Change != 5 {
		t.Errorf("Change = %d, want 5", parsed.Change)
	}
}

func TestVideoPublishEventStructure(t *testing.T) {
	// Verify field names match design docs
	e := VideoPublishEvent{
		EventID:    "1",
		VideoID:    1,
		AuthorID:   1,
		CreateTime: 1715000000000,
		OccurredAt: 1715000000001,
	}

	data, _ := json.Marshal(e)
	var m map[string]interface{}
	json.Unmarshal(data, &m)

	required := []string{"event_id", "video_id", "author_id", "create_time", "occurred_at"}
	for _, field := range required {
		if _, ok := m[field]; !ok {
			t.Errorf("missing field %q in VideoPublishEvent JSON", field)
		}
	}
}
