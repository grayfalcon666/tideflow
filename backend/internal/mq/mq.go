package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"tideflow/internal/config"
)

type MQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	conf    *config.RabbitMQConfig
	mu      sync.Mutex
}

func NewMQ(conf *config.RabbitMQConfig) (*MQ, error) {
	conn, err := amqp.Dial(conf.URL)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	mq := &MQ{conn: conn, channel: ch, conf: conf}
	if err := mq.setupExchanges(); err != nil {
		return nil, err
	}

	return mq, nil
}

func (m *MQ) setupExchanges() error {
	exchanges := []string{
		"video.events",
		"like.events",
		"comment.events",
		"social.events",
		"video.popularity.events",
		"dlx.events",
	}

	for _, name := range exchanges {
		err := m.channel.ExchangeDeclare(name, "topic", true, false, false, false, nil)
		if err != nil {
			return err
		}
	}

	queues := []struct {
		name       string
		exchange   string
		routingKey string
	}{
		{"video.publish.queue", "video.events", "video.publish"},
		{"like.like.queue", "like.events", "like.like"},
		{"like.unlike.queue", "like.events", "like.unlike"},
		{"comment.publish.queue", "comment.events", "comment.publish"},
		{"comment.delete.queue", "comment.events", "comment.delete"},
		{"social.follow.queue", "social.events", "social.follow"},
		{"social.unfollow.queue", "social.events", "social.unfollow"},
		{"popularity.update.queue", "video.popularity.events", "video.popularity.update"},
		{"notification.like.queue", "like.events", "like.like"},
		{"notification.comment.queue", "comment.events", "comment.publish"},
		{"notification.follow.queue", "social.events", "social.follow"},
	}

	for _, q := range queues {
		_, err := m.channel.QueueDeclare(q.name, true, false, false, false, amqp.Table{
			"x-dead-letter-exchange": "dlx.events",
		})
		if err != nil {
			return err
		}

		err = m.channel.QueueBind(q.name, q.routingKey, q.exchange, false, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *MQ) Publish(ctx context.Context, exchange, routingKey string, body interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	return m.channel.PublishWithContext(ctx, exchange, routingKey, false, false, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         data,
		Timestamp:    time.Now(),
		MessageId:    fmt.Sprintf("%d", time.Now().UnixNano()),
	})
}

func (m *MQ) Consume(queue string) (<-chan amqp.Delivery, error) {
	return m.channel.Consume(queue, "", false, false, false, false, nil)
}

func (m *MQ) Close() {
	if m.channel != nil {
		m.channel.Close()
	}
	if m.conn != nil {
		m.conn.Close()
	}
}

type VideoPublishEvent struct {
	EventID    string `json:"event_id"`
	VideoID    uint   `json:"video_id"`
	AuthorID   uint   `json:"author_id"`
	CreateTime int64  `json:"create_time"`
	OccurredAt int64  `json:"occurred_at"`
}

type LikeEvent struct {
	EventID   string `json:"event_id"`
	Action    string `json:"action"`
	UserID    uint   `json:"user_id"`
	VideoID   uint   `json:"video_id"`
	OccurredAt int64 `json:"occurred_at"`
}

type CommentEvent struct {
	EventID   string `json:"event_id"`
	Action    string `json:"action"`
	CommentID uint   `json:"comment_id"`
	Username  string `json:"username"`
	VideoID   uint   `json:"video_id"`
	AuthorID  uint   `json:"author_id"`
	Content   string `json:"content"`
	OccurredAt int64 `json:"occurred_at"`
}

type SocialEvent struct {
	EventID    string `json:"event_id"`
	Action     string `json:"action"`
	FollowerID uint   `json:"follower_id"`
	VloggerID  uint   `json:"vlogger_id"`
	OccurredAt int64  `json:"occurred_at"`
}

type PopularityEvent struct {
	EventID   string `json:"event_id"`
	VideoID   uint   `json:"video_id"`
	Change    int64  `json:"change"`
	OccurredAt int64 `json:"occurred_at"`
}

func ParseVideoPublishEvent(d amqp.Delivery) (*VideoPublishEvent, error) {
	var e VideoPublishEvent
	if err := json.Unmarshal(d.Body, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

func ParseLikeEvent(d amqp.Delivery) (*LikeEvent, error) {
	var e LikeEvent
	if err := json.Unmarshal(d.Body, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

func ParseCommentEvent(d amqp.Delivery) (*CommentEvent, error) {
	var e CommentEvent
	if err := json.Unmarshal(d.Body, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

func ParseSocialEvent(d amqp.Delivery) (*SocialEvent, error) {
	var e SocialEvent
	if err := json.Unmarshal(d.Body, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

func ParsePopularityEvent(d amqp.Delivery) (*PopularityEvent, error) {
	var e PopularityEvent
	if err := json.Unmarshal(d.Body, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}