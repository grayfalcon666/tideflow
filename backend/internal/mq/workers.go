package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	goredis "github.com/redis/go-redis/v9"

	"tideflow/infra/redis"
	"tideflow/internal/models"
	"tideflow/internal/repository"
)

type TimelineWorker struct {
	mq         *MQ
	rdb        *goredis.Client
	repo       *repository.Repository
	bigVThresh int
}

func NewTimelineWorker(mq *MQ, rdb *goredis.Client, repo *repository.Repository, bigVThresh int) *TimelineWorker {
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
		d.Nack(false, false)
		return
	}

	// 1. 写全局时间线
	w.rdb.ZAdd(ctx, redis.FeedGlobal(), goredis.Z{Score: float64(e.CreateTime), Member: e.VideoID})
	w.rdb.ZRemRangeByRank(ctx, redis.FeedGlobal(), 0, -1001)

	// 2. 写作者发件箱
	outboxKey := redis.Outbox(e.AuthorID)
	w.rdb.ZAdd(ctx, outboxKey, goredis.Z{Score: float64(e.CreateTime), Member: e.VideoID})
	w.rdb.ZRemRangeByRank(ctx, outboxKey, 0, -5001)

	// 3. 大V判定：查询作者粉丝数
	acc, err := w.repo.GetAccountByID(ctx, e.AuthorID)
	if err != nil {
		log.Printf("failed to get account %d: %v", e.AuthorID, err)
		d.Nack(false, true)
		return
	}
	if acc.FollowerCount >= w.bigVThresh {
		// 大V：仅写Outbox，不推送Inbox
		d.Ack(false)
		return
	}

	// 4. 普通博主：批量推送至粉丝Inbox
	followerIDs, err := w.repo.GetFollowerIDs(ctx, e.AuthorID)
	if err != nil {
		log.Printf("failed to get follower IDs for %d: %v", e.AuthorID, err)
		d.Nack(false, true)
		return
	}

	for _, fid := range followerIDs {
		inboxKey := redis.Inbox(fid)
		w.rdb.ZAdd(ctx, inboxKey, goredis.Z{Score: float64(e.CreateTime), Member: e.VideoID})
		w.rdb.ZRemRangeByRank(ctx, inboxKey, 0, -1001)
	}

	// 5. 删除作者的冷拉取缓存（可选）
	pattern := fmt.Sprintf("v1:feed:followcache:%d:*", e.AuthorID)
	iter := w.rdb.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		w.rdb.Del(ctx, iter.Val())
	}

	d.Ack(false)
}

type LikeWorker struct {
	mq         *MQ
	repo       *repository.Repository
	rdb        *goredis.Client
	bigVThresh int
}

func NewLikeWorker(mq *MQ, repo *repository.Repository, rdb *goredis.Client, bigVThresh int) *LikeWorker {
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
		d.Nack(false, false)
		return
	}

	// 更新 MySQL likes_count
	w.repo.UpdateVideo(ctx, e.VideoID, map[string]interface{}{
		"likes_count": w.repo.DB().Raw("SELECT likes_count + 1 FROM videos WHERE id = ?", e.VideoID),
	})

	// 更新 Redis 热度分
	now := time.Now()
	key := redis.HotVideo("1m", now.Format("200601021504"))
	w.rdb.ZIncrBy(ctx, key, 1, fmt.Sprintf("%d", e.VideoID))
	w.rdb.Expire(ctx, key, 2*time.Hour)

	// 删除视频实体缓存（Cache Aside）
	w.rdb.Del(ctx, redis.VideoEntity(e.VideoID))

	d.Ack(false)
}

func (w *LikeWorker) handleUnlike(ctx context.Context, d amqp.Delivery) {
	e, err := ParseLikeEvent(d)
	if err != nil {
		log.Printf("failed to parse unlike event: %v", err)
		d.Nack(false, false)
		return
	}

	// 更新 MySQL likes_count
	w.repo.UpdateVideo(ctx, e.VideoID, map[string]interface{}{
		"likes_count": w.repo.DB().Raw("SELECT likes_count - 1 FROM videos WHERE id = ?", e.VideoID),
	})

	// 更新 Redis 热度分
	now := time.Now()
	key := redis.HotVideo("1m", now.Format("200601021504"))
	w.rdb.ZIncrBy(ctx, key, -1, fmt.Sprintf("%d", e.VideoID))

	// 删除视频实体缓存
	w.rdb.Del(ctx, redis.VideoEntity(e.VideoID))

	d.Ack(false)
}

type CommentWorker struct {
	mq   *MQ
	rdb  *goredis.Client
	repo *repository.Repository
}

func NewCommentWorker(mq *MQ, rdb *goredis.Client, repo *repository.Repository) *CommentWorker {
	return &CommentWorker{mq: mq, rdb: rdb, repo: repo}
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
	e, err := ParseCommentEvent(d)
	if err != nil {
		log.Printf("failed to parse comment publish event: %v", err)
		d.Nack(false, false)
		return
	}

	// 更新 MySQL popularity（评论权重 5）
	w.repo.UpdateVideo(ctx, e.VideoID, map[string]interface{}{
		"popularity": w.repo.DB().Raw("SELECT popularity + 5 FROM videos WHERE id = ?", e.VideoID),
	})

	// 更新 Redis 热度分
	now := time.Now()
	key := redis.HotVideo("1m", now.Format("200601021504"))
	w.rdb.ZIncrBy(ctx, key, 5, fmt.Sprintf("%d", e.VideoID))
	w.rdb.Expire(ctx, key, 2*time.Hour)
	// 删除视频实体缓存
	w.rdb.Del(ctx, redis.VideoEntity(e.VideoID))

	d.Ack(false)
}

func (w *CommentWorker) handleCommentDelete(ctx context.Context, d amqp.Delivery) {
	e, err := ParseCommentEvent(d)
	if err != nil {
		log.Printf("failed to parse comment delete event: %v", err)
		d.Nack(false, false)
		return
	}

	// 更新 MySQL popularity（评论权重 5）
	w.repo.UpdateVideo(ctx, e.VideoID, map[string]interface{}{
		"popularity": w.repo.DB().Raw("SELECT popularity - 5 FROM videos WHERE id = ?", e.VideoID),
	})

	// 更新 Redis 热度分
	now := time.Now()
	key := redis.HotVideo("1m", now.Format("200601021504"))
	// 评论删除减少 popularity（评论权重 5）
	w.rdb.ZIncrBy(ctx, key, -5, fmt.Sprintf("%d", e.VideoID))

	// 删除视频实体缓存
	w.rdb.Del(ctx, redis.VideoEntity(e.VideoID))

	d.Ack(false)
}

type SocialWorker struct {
	mq         *MQ
	rdb        *goredis.Client
	repo       *repository.Repository
	bigVThresh int
}

func NewSocialWorker(mq *MQ, rdb *goredis.Client, repo *repository.Repository, bigVThresh int) *SocialWorker {
	return &SocialWorker{mq: mq, rdb: rdb, repo: repo, bigVThresh: bigVThresh}
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
	e, err := ParseSocialEvent(d)
	if err != nil {
		log.Printf("failed to parse social follow event: %v", err)
		return
	}

	// 更新博主粉丝计数（+1）
	w.repo.UpdateAccount(ctx, e.VloggerID, map[string]interface{}{
		"follower_count": w.repo.DB().Raw("SELECT follower_count + 1 FROM accounts WHERE id = ?", e.VloggerID),
	})

	// 删除粉丝的冷拉取缓存
	w.deleteFeedCache(ctx, e.FollowerID)

	// 如果博主之前是大V，降级后应删除其大V标记缓存
	w.rdb.Del(ctx, redis.BigVMark(e.VloggerID))
}

func (w *SocialWorker) handleUnfollow(ctx context.Context, d amqp.Delivery) {
	e, err := ParseSocialEvent(d)
	if err != nil {
		log.Printf("failed to parse social unfollow event: %v", err)
		return
	}

	// 更新博主粉丝计数（-1）
	w.repo.UpdateAccount(ctx, e.VloggerID, map[string]interface{}{
		"follower_count": w.repo.DB().Raw("SELECT follower_count - 1 FROM accounts WHERE id = ?", e.VloggerID),
	})

	// 删除粉丝的冷拉取缓存
	w.deleteFeedCache(ctx, e.FollowerID)
}

func (w *SocialWorker) deleteFeedCache(ctx context.Context, userID uint) {
	pattern := fmt.Sprintf("v1:feed:followcache:%d:*", userID)
	iter := w.rdb.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		w.rdb.Del(ctx, iter.Val())
	}
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
	repo *repository.Repository
}

func NewOutboxWorker(mq *MQ, repo *repository.Repository) *OutboxWorker {
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
	msgs, err := w.repo.GetPendingOutboxMsgs(ctx, 100)
	if err != nil || len(msgs) == 0 {
		return
	}
	for _, m := range msgs {
		if m.AuthorID == 0 || m.VideoID == 0 {
			// 防御：跳过字段不完整的旧遗留记录，直接标记为 processed
			w.repo.UpdateOutboxMsgStatus(ctx, m.ID, "processed")
			continue
		}
		event := VideoPublishEvent{
			EventID:    fmt.Sprintf("%d", m.ID),
			VideoID:    m.VideoID,
			AuthorID:   m.AuthorID,
			CreateTime: m.CreateTime.UnixMilli(),
			OccurredAt: time.Now().UnixMilli(),
		}
		if err := w.mq.Publish(ctx, "video.events", "video.publish", event); err != nil {
			log.Printf("failed to publish video.publish event for msg %d: %v", m.ID, err)
			continue
		}
		if err := w.repo.UpdateOutboxMsgStatus(ctx, m.ID, "processed"); err != nil {
			log.Printf("failed to update outbox msg %d status: %v", m.ID, err)
		}
	}
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
	repo  *repository.Repository
}

func NewNotificationWorker(mq *MQ, hub *SSEHub, repo *repository.Repository) *NotificationWorker {
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
	e, err := ParseLikeEvent(d)
	if err != nil {
		log.Printf("failed to parse like event: %v", err)
		d.Nack(false, false)
		return
	}

	// 获取视频作者（通知接收者）
	video, err := w.repo.GetVideoByID(ctx, e.VideoID)
	if err != nil {
		log.Printf("failed to get video %d: %v", e.VideoID, err)
		d.Ack(false)
		return
	}
	// 不要给自己点赞发送通知
	if video.AuthorID == e.UserID {
		d.Ack(false)
		return
	}

	// 创建通知
	content := fmt.Sprintf("用户 %d 点赞了你的视频", e.UserID)
	notif := &models.Notification{
		RecipientID: video.AuthorID,
		SenderID:    e.UserID,
		Type:        "like",
		TargetID:    e.VideoID,
		Content:     content,
		IsRead:      false,
		CreatedAt:   time.Now(),
	}
	if err := w.repo.CreateNotification(ctx, notif); err != nil {
		log.Printf("failed to create like notification: %v", err)
	}

	// SSE 推送
	msg, _ := json.Marshal(map[string]interface{}{
		"type":      "like",
		"user_id":   e.UserID,
		"video_id":  e.VideoID,
		"content":   content,
		"occurred_at": e.OccurredAt,
	})
	w.hub.Push(video.AuthorID, string(msg))

	d.Ack(false)
}

func (w *NotificationWorker) handleCommentNotification(ctx context.Context, d amqp.Delivery) {
	e, err := ParseCommentEvent(d)
	if err != nil {
		log.Printf("failed to parse comment event: %v", err)
		d.Nack(false, false)
		return
	}

	// 获取视频作者（通知接收者）
	video, err := w.repo.GetVideoByID(ctx, e.VideoID)
	if err != nil {
		log.Printf("failed to get video %d: %v", e.VideoID, err)
		d.Ack(false)
		return
	}
	// 不要给自己评论发送通知
	if video.AuthorID == e.AuthorID {
		d.Ack(false)
		return
	}

	// 创建通知
	content := fmt.Sprintf("用户 %s 评论了你的视频: %s", e.Username, e.Content)
	if len(content) > 255 {
		content = content[:252] + "..."
	}
	notif := &models.Notification{
		RecipientID: video.AuthorID,
		SenderID:    e.AuthorID,
		Type:        "comment",
		TargetID:    e.CommentID,
		Content:     content,
		IsRead:      false,
		CreatedAt:   time.Now(),
	}
	if err := w.repo.CreateNotification(ctx, notif); err != nil {
		log.Printf("failed to create comment notification: %v", err)
	}

	// SSE 推送
	msg, _ := json.Marshal(map[string]interface{}{
		"type":       "comment",
		"user_id":    e.AuthorID,
		"username":   e.Username,
		"video_id":   e.VideoID,
		"comment_id": e.CommentID,
		"content":    e.Content,
		"occurred_at": e.OccurredAt,
	})
	w.hub.Push(video.AuthorID, string(msg))

	d.Ack(false)
}

func (w *NotificationWorker) handleFollowNotification(ctx context.Context, d amqp.Delivery) {
	e, err := ParseSocialEvent(d)
	if err != nil {
		log.Printf("failed to parse social event: %v", err)
		d.Nack(false, false)
		return
	}

	// 创建通知
	content := fmt.Sprintf("用户 %d 关注了你", e.FollowerID)
	notif := &models.Notification{
		RecipientID: e.VloggerID,
		SenderID:    e.FollowerID,
		Type:        "follow",
		TargetID:    e.FollowerID,
		Content:     content,
		IsRead:      false,
		CreatedAt:   time.Now(),
	}
	if err := w.repo.CreateNotification(ctx, notif); err != nil {
		log.Printf("failed to create follow notification: %v", err)
	}

	// SSE 推送
	msg, _ := json.Marshal(map[string]interface{}{
		"type":       "follow",
		"user_id":    e.FollowerID,
		"vlogger_id": e.VloggerID,
		"content":    content,
		"occurred_at": e.OccurredAt,
	})
	w.hub.Push(e.VloggerID, string(msg))

	d.Ack(false)
}