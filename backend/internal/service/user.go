package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"tideflow/internal/models"
	"tideflow/internal/mq"
	"tideflow/internal/repository"
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
	ID            uint   `json:"id"`
	Username      string `json:"username"`
	AvatarURL     string `json:"avatar_url"`
	Bio           string `json:"bio"`
	FollowerCount int    `json:"follower_count"`
	IsBigV        bool   `json:"is_big_v,omitempty"`
}

func (s *UserService) GetUserByID(ctx context.Context, id uint) (*UserProfile, error) {
	acc, err := s.repo.GetAccountByID(ctx, id)
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
	_, err := s.repo.GetSocial(ctx, followerID, vloggerID)
	if err == nil {
		return nil
	}

	social := &models.Social{FollowerID: followerID, VloggerID: vloggerID}
	if err := s.repo.CreateSocial(ctx, social); err != nil {
		return err
	}

	// 发布关注事件，SocialWorker 消费：更新粉丝计数 + 失效缓存
	event := mq.SocialEvent{
		EventID:    fmt.Sprintf("%d-%d-%d", followerID, vloggerID, time.Now().UnixNano()),
		Action:     "follow",
		FollowerID: followerID,
		VloggerID:  vloggerID,
		OccurredAt: time.Now().UnixMilli(),
	}
	s.mq.Publish(ctx, "social.events", "social.follow", event)

	return nil
}

func (s *UserService) Unfollow(ctx context.Context, followerID, vloggerID uint) error {
	if err := s.repo.DeleteSocial(ctx, followerID, vloggerID); err != nil {
		return err
	}

	// 发布取消关注事件，SocialWorker 消费：更新粉丝计数 + 失效缓存
	event := mq.SocialEvent{
		EventID:    fmt.Sprintf("%d-%d-%d", followerID, vloggerID, time.Now().UnixNano()),
		Action:     "unfollow",
		FollowerID: followerID,
		VloggerID:  vloggerID,
		OccurredAt: time.Now().UnixMilli(),
	}
	s.mq.Publish(ctx, "social.events", "social.unfollow", event)

	return nil
}

func (s *UserService) GetFollowing(ctx context.Context, userID uint, cursor string, limit int) ([]*UserProfile, *string, bool, error) {
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

	profiles := make([]*UserProfile, len(accounts))
	for i, acc := range accounts {
		profiles[i] = &UserProfile{
			ID:            acc.ID,
			Username:      acc.Username,
			AvatarURL:     acc.AvatarURL,
			Bio:           acc.Bio,
			FollowerCount: acc.FollowerCount,
			IsBigV:        acc.FollowerCount >= s.bigVThresh,
		}
	}

	var nextCursor *string
	if hasMore && len(accounts) > 0 {
		nc := strconv.FormatUint(uint64(accounts[len(accounts)-1].ID), 10)
		nextCursor = &nc
	}

	return profiles, nextCursor, hasMore, nil
}

func (s *UserService) GetFollowers(ctx context.Context, userID uint, cursor string, limit int) ([]*UserProfile, *string, bool, error) {
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

	profiles := make([]*UserProfile, len(accounts))
	for i, acc := range accounts {
		profiles[i] = &UserProfile{
			ID:            acc.ID,
			Username:      acc.Username,
			AvatarURL:     acc.AvatarURL,
			Bio:           acc.Bio,
			FollowerCount: acc.FollowerCount,
			IsBigV:        acc.FollowerCount >= s.bigVThresh,
		}
	}

	var nextCursor *string
	if hasMore && len(accounts) > 0 {
		nc := strconv.FormatUint(uint64(accounts[len(accounts)-1].ID), 10)
		nextCursor = &nc
	}

	return profiles, nextCursor, hasMore, nil
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

func (s *UserService) GetUserVideos(ctx context.Context, userID, requesterID uint, cursor string, limit int) ([]*models.Video, *string, bool, error) {
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

	var nextCursor *string
	if hasMore && len(videos) > 0 {
		nc := strconv.FormatInt(videos[len(videos)-1].CreateTime.UnixMilli(), 10)
		nextCursor = &nc
	}

	return videos, nextCursor, hasMore, nil
}