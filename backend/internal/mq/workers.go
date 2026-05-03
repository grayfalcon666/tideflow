package mq

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	goredis "github.com/redis/go-redis/v9"

	"tideflow/infra/redis"
)

type TimelineWorker struct {
	mq         *MQ
	rdb        *goredis.Client
	repo       interface{}
	bigVThresh int
}

func NewTimelineWorker(mq *MQ, rdb *goredis.Client, repo interface{}, bigVThresh int) *TimelineWorker {
	return &TimelineWorker{mq: mq, rdb: rdb, repo: repo, bigVThresh: bigVThresh}
}

func (w *TimelineWorker) Start(ctx context.Context) error {
	msgs, err := w.mq.Consume("video.publish.queue")
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d, ok := <-msgs:
				if !ok {
					return
				}
				w.handleVideoPublish(ctx, d)
			}
		}
	}()

	return nil
}

func (w *TimelineWorker) handleVideoPublish(ctx context.Context, d amqp.Delivery) {
	e, err := ParseVideoPublishEvent(d)
	if err != nil {
		log.Printf("failed to parse video publish event: %v", err)
		return
	}

	w.rdb.ZAdd(ctx, redis.FeedGlobal(), goredis.Z{Score: float64(e.CreateTime), Member: e.VideoID})
	w.rdb.ZRemRangeByRank(ctx, redis.FeedGlobal(), 0, -1001)

	outboxKey := redis.Outbox(e.AuthorID)
	w.rdb.ZAdd(ctx, outboxKey, goredis.Z{Score: float64(e.CreateTime), Member: e.VideoID})
	w.rdb.ZRemRangeByRank(ctx, outboxKey, 0, -5001)
}

type LikeWorker struct {
	mq         *MQ
	repo       interface{}
	rdb        *goredis.Client
	bigVThresh int
}

func NewLikeWorker(mq *MQ, repo interface{}, rdb *goredis.Client, bigVThresh int) *LikeWorker {
	return &LikeWorker{mq: mq, repo: repo, rdb: rdb, bigVThresh: bigVThresh}
}

func (w *LikeWorker) Start(ctx context.Context) error {
	likeMsgs, _ := w.mq.Consume("like.like.queue")
	unlikeMsgs, _ := w.mq.Consume("like.unlike.queue")

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d, ok := <-likeMsgs:
				if !ok {
					return
				}
				w.handleLike(ctx, d)
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d, ok := <-unlikeMsgs:
				if !ok {
					return
				}
				w.handleUnlike(ctx, d)
			}
		}
	}()

	return nil
}

func (w *LikeWorker) handleLike(ctx context.Context, d amqp.Delivery) {
	e, err := ParseLikeEvent(d)
	if err != nil {
		log.Printf("failed to parse like event: %v", err)
		return
	}

	now := time.Now()
	key := redis.HotVideo("1m", now.Format("200601021504"))
	w.rdb.ZIncrBy(ctx, key, 1, fmt.Sprintf("%d", e.VideoID))
	w.rdb.Expire(ctx, key, 2*time.Hour)
}

func (w *LikeWorker) handleUnlike(ctx context.Context, d amqp.Delivery) {
	e, err := ParseLikeEvent(d)
	if err != nil {
		log.Printf("failed to parse unlike event: %v", err)
		return
	}

	now := time.Now()
	key := redis.HotVideo("1m", now.Format("200601021504"))
	w.rdb.ZIncrBy(ctx, key, -1, fmt.Sprintf("%d", e.VideoID))
}

type CommentWorker struct {
	mq *MQ
	rdb *goredis.Client
}

func NewCommentWorker(mq *MQ, rdb *goredis.Client) *CommentWorker {
	return &CommentWorker{mq: mq, rdb: rdb}
}

func (w *CommentWorker) Start(ctx context.Context) error {
	publishMsgs, _ := w.mq.Consume("comment.publish.queue")
	deleteMsgs, _ := w.mq.Consume("comment.delete.queue")

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d, ok := <-publishMsgs:
				if !ok {
					return
				}
				w.handleCommentPublish(ctx, d)
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d, ok := <-deleteMsgs:
				if !ok {
					return
				}
				w.handleCommentDelete(ctx, d)
			}
		}
	}()

	return nil
}

func (w *CommentWorker) handleCommentPublish(ctx context.Context, d amqp.Delivery) {
	now := time.Now()
	key := redis.HotVideo("1m", now.Format("200601021504"))
	e, err := ParseCommentEvent(d)
	if err == nil {
		w.rdb.ZIncrBy(ctx, key, 5, fmt.Sprintf("%d", e.VideoID))
		w.rdb.Expire(ctx, key, 2*time.Hour)
	}
}

func (w *CommentWorker) handleCommentDelete(ctx context.Context, d amqp.Delivery) {
}

type SocialWorker struct {
	mq         *MQ
	rdb        *goredis.Client
	bigVThresh int
}

func NewSocialWorker(mq *MQ, rdb *goredis.Client, bigVThresh int) *SocialWorker {
	return &SocialWorker{mq: mq, rdb: rdb, bigVThresh: bigVThresh}
}

func (w *SocialWorker) Start(ctx context.Context) error {
	followMsgs, _ := w.mq.Consume("social.follow.queue")
	unfollowMsgs, _ := w.mq.Consume("social.unfollow.queue")

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d, ok := <-followMsgs:
				if !ok {
					return
				}
				w.handleFollow(ctx, d)
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d, ok := <-unfollowMsgs:
				if !ok {
					return
				}
				w.handleUnfollow(ctx, d)
			}
		}
	}()

	return nil
}

func (w *SocialWorker) handleFollow(ctx context.Context, d amqp.Delivery) {
}

func (w *SocialWorker) handleUnfollow(ctx context.Context, d amqp.Delivery) {
}

type PopularityWorker struct {
	mq  *MQ
	rdb *goredis.Client
}

func NewPopularityWorker(mq *MQ, rdb *goredis.Client) *PopularityWorker {
	return &PopularityWorker{mq: mq, rdb: rdb}
}

func (w *PopularityWorker) Start(ctx context.Context) error {
	msgs, err := w.mq.Consume("popularity.update.queue")
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d, ok := <-msgs:
				if !ok {
					return
				}
				w.handlePopularityUpdate(ctx, d)
			}
		}
	}()

	return nil
}

func (w *PopularityWorker) handlePopularityUpdate(ctx context.Context, d amqp.Delivery) {
	e, err := ParsePopularityEvent(d)
	if err != nil {
		log.Printf("failed to parse popularity event: %v", err)
		return
	}

	now := time.Now()
	key := redis.HotVideo("1m", now.Format("200601021504"))
	w.rdb.ZIncrBy(ctx, key, float64(e.Change), fmt.Sprintf("%d", e.VideoID))
	w.rdb.Expire(ctx, key, 2*time.Hour)
}

type OutboxWorker struct {
	mq   *MQ
	repo interface{}
}

func NewOutboxWorker(mq *MQ, repo interface{}) *OutboxWorker {
	return &OutboxWorker{mq: mq, repo: repo}
}

func (w *OutboxWorker) Start(ctx context.Context) error {
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.processOutbox(ctx)
			}
		}
	}()
	return nil
}

func (w *OutboxWorker) processOutbox(ctx context.Context) {
}

type SSEHub struct {
	clients    map[uint]chan string
	broadcast  chan hubMessage
	register   chan *hubClient
	unregister chan uint
}

type hubClient struct {
	userID uint
	ch     chan string
}

type hubMessage struct {
	userID uint
	data   string
}

func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients:    make(map[uint]chan string),
		broadcast: make(chan hubMessage),
		register:  make(chan *hubClient),
		unregister: make(chan uint),
	}
}

func (h *SSEHub) Run() {
	for {
		select {
		case c := <-h.register:
			h.clients[c.userID] = c.ch
		case userID := <-h.unregister:
			if ch, ok := h.clients[userID]; ok {
				close(ch)
				delete(h.clients, userID)
			}
		case m := <-h.broadcast:
			if ch, ok := h.clients[m.userID]; ok {
				select {
				case ch <- m.data:
				default:
				}
			}
		}
	}
}

func (h *SSEHub) Push(userID uint, data string) {
	h.broadcast <- hubMessage{userID: userID, data: data}
}

func (h *SSEHub) Register(userID uint) chan string {
	ch := make(chan string, 256)
	h.register <- &hubClient{userID: userID, ch: ch}
	return ch
}

func (h *SSEHub) Unregister(userID uint) {
	h.unregister <- userID
}

type NotificationWorker struct {
	mq    *MQ
	hub   *SSEHub
	repo  interface{}
}

func NewNotificationWorker(mq *MQ, hub *SSEHub, repo interface{}) *NotificationWorker {
	return &NotificationWorker{mq: mq, hub: hub, repo: repo}
}

func (w *NotificationWorker) Start(ctx context.Context) error {
	likeMsgs, _ := w.mq.Consume("notification.like.queue")
	commentMsgs, _ := w.mq.Consume("notification.comment.queue")
	followMsgs, _ := w.mq.Consume("notification.follow.queue")

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d, ok := <-likeMsgs:
				if !ok {
					return
				}
				w.handleLikeNotification(ctx, d)
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d, ok := <-commentMsgs:
				if !ok {
					return
				}
				w.handleCommentNotification(ctx, d)
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d, ok := <-followMsgs:
				if !ok {
					return
				}
				w.handleFollowNotification(ctx, d)
			}
		}
	}()

	return nil
}

func (w *NotificationWorker) handleLikeNotification(ctx context.Context, d amqp.Delivery) {
	body := string(d.Body)
	if strings.Contains(body, "like") {
		w.hub.Push(0, body)
	}
}

func (w *NotificationWorker) handleCommentNotification(ctx context.Context, d amqp.Delivery) {
}

func (w *NotificationWorker) handleFollowNotification(ctx context.Context, d amqp.Delivery) {
}