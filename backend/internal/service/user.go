package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"tideflow/internal/mq"
	"tideflow/internal/repository"
)

var (
	ErrAlreadyFollowing = errors.New("already following")
	ErrNotFollowing    = errors.New("not following")
)

type UserService struct {
	repo        *repository.Repository
	bigVThresh  int
	mq          *mq.MQ
}

func NewUserService(repo *repository.Repository, bigVThresh int, mqInstance *mq.MQ) *UserService {
	return &UserService{repo: repo, bigVThresh: bigVThresh, mq: mqInstance}
}

type UserProfile struct {
	ID             uint   `json:"id"`
	Username       string `json:"username"`
	AvatarURL      string `json:"avatar_url"`
	Bio            string `json:"bio"`
	FollowerCount  int    `json:"follower_count"`
	FollowingCount int64   `json:"following_count,omitempty"`
	IsBigV         bool   `json:"is_big_v,omitempty"`
}

type FollowUserItem struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
	IsBigV    bool   `json:"is_big_v"`
}

type UserVideoItem struct {
	VideoID     uint   `json:"video_id"`
	Title       string `json:"title"`
	CoverURL    string `json:"cover_url"`
	CreateTime  int64  `json:"create_time"`
	LikesCount  int64  `json:"likes_count"`
	Popularity  int64  `json:"popularity"`
	IsLiked     bool   `json:"is_liked"`
}

func (s *UserService) GetUserByID(ctx context.Context, id uint) (*UserProfile, error) {
	acc, err := s.repo.GetAccountByID(ctx, id)
	if err != nil {
		return nil, err
	}
	followingCount, _ := s.repo.CountFollowing(ctx, id)
	return &UserProfile{
		ID:             acc.ID,
		Username:       acc.Username,
		AvatarURL:      acc.AvatarURL,
		Bio:            acc.Bio,
		FollowerCount:  acc.FollowerCount,
		FollowingCount: followingCount,
		IsBigV:         acc.FollowerCount >= s.bigVThresh,
	}, nil
}

func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*UserProfile, error) {
	acc, err := s.repo.GetAccountByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return &UserProfile{
		ID:            acc.ID,
		Username:      acc.Username,
		AvatarURL:     acc.AvatarURL,
		Bio:           acc.Bio,
		FollowerCount: acc.FollowerCount,
		IsBigV:        acc.FollowerCount >= s.bigVThresh,
	}, nil
}

func (s *UserService) UpdateUser(ctx context.Context, id uint, updates map[string]interface{}) error {
	return s.repo.UpdateAccount(ctx, id, updates)
}

func (s *UserService) UpdateUsername(ctx context.Context, id uint, username string) error {
	_, err := s.repo.GetAccountByUsername(ctx, username)
	if err == nil {
		return ErrUserExists
	}
	return s.repo.UpdateAccount(ctx, id, map[string]interface{}{"username": username})
}

func (s *UserService) IsBigV(ctx context.Context, userID uint) (bool, error) {
	acc, err := s.repo.GetAccountByID(ctx, userID)
	if err != nil {
		return false, err
	}
	return acc.FollowerCount >= s.bigVThresh, nil
}

func (s *UserService) Follow(ctx context.Context, followerID, vloggerID uint) error {
	// 不能关注自己
	if followerID == vloggerID {
		return fmt.Errorf("cannot follow yourself")
	}

	// 检查目标用户是否存在
	if _, err := s.repo.GetAccountByID(ctx, vloggerID); err != nil {
		return fmt.Errorf("user %d not found", vloggerID)
	}

	rows, err := s.repo.FollowOrCreate(ctx, followerID, vloggerID)
	if err != nil {
		return err
	}
	// rows > 0: 状态发生了改变（新关注 或 取消后重新关注），发MQ事件
	// rows == 0: 已关注且状态没变化，返回错误
	if rows == 0 {
		return ErrAlreadyFollowing
	}
	event := mq.SocialEvent{
		EventID:    fmt.Sprintf("%d-%d-%d", followerID, vloggerID, time.Now().UnixNano()),
		Action:     "follow",
		FollowerID: followerID,
		VloggerID:  vloggerID,
		OccurredAt: time.Now().UnixMilli(),
	}
	if err := s.mq.Publish(ctx, "social.events", "social.follow", event); err != nil {
		log.Printf("[Follow] failed to publish event: %v", err)
	}
	return nil
}

func (s *UserService) Unfollow(ctx context.Context, followerID, vloggerID uint) error {
	// 检查目标用户是否存在
	if _, err := s.repo.GetAccountByID(ctx, vloggerID); err != nil {
		return fmt.Errorf("user %d not found", vloggerID)
	}
	rows, err := s.repo.UnfollowStatus(ctx, followerID, vloggerID)
	if err != nil {
		return err
	}
	// rows=1: 真正取关，发MQ事件
	// rows=0: 本来就没关注，返回错误
	if rows == 0 {
		return ErrNotFollowing
	}
	event := mq.SocialEvent{
		EventID:    fmt.Sprintf("%d-%d-%d", followerID, vloggerID, time.Now().UnixNano()),
		Action:     "unfollow",
		FollowerID: followerID,
		VloggerID:  vloggerID,
		OccurredAt: time.Now().UnixMilli(),
	}
	if err := s.mq.Publish(ctx, "social.events", "social.unfollow", event); err != nil {
		log.Printf("[Unfollow] failed to publish event: %v", err)
	}
	return nil
}

func (s *UserService) GetFollowing(ctx context.Context, userID uint, cursor string, limit int) ([]*FollowUserItem, *string, bool, error) {
	var cur int64
	if cursor != "" {
		parsed, err := strconv.ParseInt(cursor, 10, 64)
		if err == nil {
			cur = parsed
		}
	}

	accounts, err := s.repo.GetFollowingWithCursor(ctx, userID, cur, limit+1)
	if err != nil {
		return nil, nil, false, err
	}

	hasMore := len(accounts) > limit
	if hasMore {
		accounts = accounts[:limit]
	}

	items := make([]*FollowUserItem, len(accounts))
	for i, acc := range accounts {
		items[i] = &FollowUserItem{
			ID:        acc.ID,
			Username:  acc.Username,
			AvatarURL: acc.AvatarURL,
			IsBigV:    acc.FollowerCount >= s.bigVThresh,
		}
	}

	var nextCursor *string
	if hasMore && len(accounts) > 0 {
		nc := strconv.FormatUint(uint64(accounts[len(accounts)-1].ID), 10)
		nextCursor = &nc
	}

	return items, nextCursor, hasMore, nil
}

func (s *UserService) GetFollowers(ctx context.Context, userID uint, cursor string, limit int) ([]*FollowUserItem, *string, bool, error) {
	var cur int64
	if cursor != "" {
		parsed, err := strconv.ParseInt(cursor, 10, 64)
		if err == nil {
			cur = parsed
		}
	}

	accounts, err := s.repo.GetFollowersWithCursor(ctx, userID, cur, limit+1)
	if err != nil {
		return nil, nil, false, err
	}

	hasMore := len(accounts) > limit
	if hasMore {
		accounts = accounts[:limit]
	}

	items := make([]*FollowUserItem, len(accounts))
	for i, acc := range accounts {
		items[i] = &FollowUserItem{
			ID:        acc.ID,
			Username:  acc.Username,
			AvatarURL: acc.AvatarURL,
			IsBigV:    acc.FollowerCount >= s.bigVThresh,
		}
	}

	var nextCursor *string
	if hasMore && len(accounts) > 0 {
		nc := strconv.FormatUint(uint64(accounts[len(accounts)-1].ID), 10)
		nextCursor = &nc
	}

	return items, nextCursor, hasMore, nil
}

func (s *UserService) GetSocialCounts(ctx context.Context, userID uint) (map[string]int64, error) {
	following, err := s.repo.CountFollowing(ctx, userID)
	if err != nil {
		return nil, err
	}
	followers, err := s.repo.CountFollowers(ctx, userID)
	if err != nil {
		return nil, err
	}
	return map[string]int64{
		"follower_count": followers,
		"following_count": following,
	}, nil
}

func (s *UserService) GetUserVideos(ctx context.Context, userID, requesterID uint, cursor string, limit int) ([]*UserVideoItem, *string, bool, error) {
	var before time.Time
	if cursor != "" {
		ts, err := strconv.ParseInt(cursor, 10, 64)
		if err == nil {
			before = time.UnixMilli(ts)
		}
	}

	videos, err := s.repo.GetVideosByAuthor(ctx, userID, before, limit+1)
	if err != nil {
		return nil, nil, false, err
	}

	hasMore := len(videos) > limit
	if hasMore {
		videos = videos[:limit]
	}

	// 批量查询 requester 对这些视频的点赞状态
	videoIDs := make([]uint, len(videos))
	for i, v := range videos {
		videoIDs[i] = v.ID
	}
	likedMap := make(map[uint]bool)
	if requesterID != 0 && len(videoIDs) > 0 {
		likedMap, _ = s.repo.GetLikesByAccountAndVideos(ctx, requesterID, videoIDs)
	}

	items := make([]*UserVideoItem, len(videos))
	for i, v := range videos {
		items[i] = &UserVideoItem{
			VideoID:    v.ID,
			Title:      v.Title,
			CoverURL:   v.CoverURL,
			CreateTime: v.CreateTime.UnixMilli(),
			LikesCount: v.LikesCount,
			Popularity: v.Popularity,
			IsLiked:   likedMap[v.ID],
		}
	}

	var nextCursor *string
	if hasMore && len(videos) > 0 {
		nc := strconv.FormatInt(videos[len(videos)-1].CreateTime.UnixMilli(), 10)
		nextCursor = &nc
	}

	return items, nextCursor, hasMore, nil
}