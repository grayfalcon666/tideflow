package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"tideflow/internal/models"
)

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) DB() *gorm.DB {
	return r.db
}

func (r *Repository) CreateAccount(ctx context.Context, acc *models.Account) error {
	return r.db.WithContext(ctx).Create(acc).Error
}

func (r *Repository) GetAccountByUsername(ctx context.Context, username string) (*models.Account, error) {
	var acc models.Account
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&acc).Error
	if err != nil {
		return nil, err
	}
	return &acc, nil
}

func (r *Repository) GetAccountByID(ctx context.Context, id uint) (*models.Account, error) {
	var acc models.Account
	err := r.db.WithContext(ctx).First(&acc, id).Error
	if err != nil {
		return nil, err
	}
	return &acc, nil
}

func (r *Repository) GetAccountsByIDs(ctx context.Context, ids []uint) ([]*models.Account, error) {
	var accounts []*models.Account
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&accounts).Error
	return accounts, err
}

func (r *Repository) UpdateAccount(ctx context.Context, id uint, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Account{}).Where("id = ?", id).Updates(updates).Error
}

func (r *Repository) CreateVideo(ctx context.Context, video *models.Video) error {
	return r.db.WithContext(ctx).Create(video).Error
}

func (r *Repository) GetVideoByID(ctx context.Context, id uint) (*models.Video, error) {
	var video models.Video
	err := r.db.WithContext(ctx).First(&video, id).Error
	if err != nil {
		return nil, err
	}
	return &video, nil
}

func (r *Repository) GetVideosByIDs(ctx context.Context, ids []uint) ([]*models.Video, error) {
	var videos []*models.Video
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&videos).Error
	return videos, err
}

func (r *Repository) GetVideosByAuthor(ctx context.Context, authorID uint, before time.Time, limit int) ([]*models.Video, error) {
	var videos []*models.Video
	query := r.db.WithContext(ctx).Where("author_id = ? AND deleted_at IS NULL", authorID)
	if !before.IsZero() {
		query = query.Where("create_time < ?", before)
	}
	err := query.Order("create_time DESC").Limit(limit).Find(&videos).Error
	return videos, err
}

func (r *Repository) GetLatestVideos(ctx context.Context, before time.Time, limit int) ([]*models.Video, error) {
	var videos []*models.Video
	query := r.db.WithContext(ctx).Where("deleted_at IS NULL")
	if !before.IsZero() {
		query = query.Where("create_time < ?", before)
	}
	err := query.Order("create_time DESC").Limit(limit).Find(&videos).Error
	return videos, err
}

func (r *Repository) GetPopularVideos(ctx context.Context, offset, limit int) ([]*models.Video, error) {
	var videos []*models.Video
	err := r.db.WithContext(ctx).Where("deleted_at IS NULL").
		Order("popularity DESC, id DESC").
		Offset(offset).Limit(limit).Find(&videos).Error
	return videos, err
}

func (r *Repository) UpdateVideo(ctx context.Context, id uint, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Video{}).Where("id = ?", id).Updates(updates).Error
}

func (r *Repository) SoftDeleteVideo(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&models.Video{}).Where("id = ?", id).Update("deleted_at", time.Now()).Error
}

func (r *Repository) IncrementLikesCount(ctx context.Context, id uint, delta int64) error {
	return r.db.WithContext(ctx).Exec(
		"UPDATE videos SET likes_count = likes_count + ? WHERE id = ? AND deleted_at IS NULL",
		delta, id,
	).Error
}

func (r *Repository) IncrementFollowerCount(ctx context.Context, id uint, delta int64) error {
	return r.db.WithContext(ctx).Exec(
		"UPDATE accounts SET follower_count = follower_count + ? WHERE id = ?",
		delta, id,
	).Error
}

func (r *Repository) IncrementVideoPopularity(ctx context.Context, id uint, delta int64) error {
	return r.db.WithContext(ctx).Exec(
		"UPDATE videos SET popularity = popularity + ? WHERE id = ? AND deleted_at IS NULL",
		delta, id,
	).Error
}

func (r *Repository) CreateLike(ctx context.Context, like *models.Like) error {
	return r.db.WithContext(ctx).Create(like).Error
}

func (r *Repository) DeleteLike(ctx context.Context, videoID, accountID uint) error {
	return r.db.WithContext(ctx).Where("video_id = ? AND account_id = ?", videoID, accountID).Delete(&models.Like{}).Error
}

func (r *Repository) GetLike(ctx context.Context, videoID, accountID uint) (*models.Like, error) {
	var like models.Like
	err := r.db.WithContext(ctx).Where("video_id = ? AND account_id = ?", videoID, accountID).First(&like).Error
	if err != nil {
		return nil, err
	}
	return &like, nil
}

func (r *Repository) GetLikesByAccount(ctx context.Context, accountID uint, before time.Time, limit int) ([]*models.Like, error) {
	var likes []*models.Like
	query := r.db.WithContext(ctx).Where("account_id = ?", accountID)
	if !before.IsZero() {
		query = query.Where("created_at < ?", before)
	}
	err := query.Order("created_at DESC").Limit(limit).Find(&likes).Error
	return likes, err
}

func (r *Repository) GetLikesByAccountAndVideos(ctx context.Context, accountID uint, videoIDs []uint) (map[uint]bool, error) {
	if len(videoIDs) == 0 || accountID == 0 {
		return make(map[uint]bool), nil
	}
	var likes []*models.Like
	err := r.db.WithContext(ctx).
		Where("account_id = ? AND video_id IN ?", accountID, videoIDs).
		Find(&likes).Error
	if err != nil {
		return nil, err
	}
	result := make(map[uint]bool, len(likes))
	for _, l := range likes {
		result[l.VideoID] = true
	}
	return result, nil
}

func (r *Repository) CreateComment(ctx context.Context, comment *models.Comment) error {
	return r.db.WithContext(ctx).Create(comment).Error
}

func (r *Repository) GetComment(ctx context.Context, id uint) (*models.Comment, error) {
	var comment models.Comment
	err := r.db.WithContext(ctx).First(&comment, id).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

func (r *Repository) GetCommentsByVideo(ctx context.Context, videoID uint, rootID uint, before time.Time, limit int) ([]*models.Comment, error) {
	var comments []*models.Comment
	query := r.db.WithContext(ctx).Where("video_id = ? AND deleted_at IS NULL", videoID)
	if rootID > 0 {
		query = query.Where("root_id = ?", rootID)
	} else {
		query = query.Where("root_id = 0")
	}
	if !before.IsZero() {
		query = query.Where("created_at < ?", before)
	}
	err := query.Order("created_at DESC").Limit(limit).Find(&comments).Error
	return comments, err
}

func (r *Repository) CountReplies(ctx context.Context, rootID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Comment{}).Where("root_id = ? AND deleted_at IS NULL", rootID).Count(&count).Error
	return count, err
}

func (r *Repository) SoftDeleteComment(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&models.Comment{}).Where("id = ?", id).Update("deleted_at", time.Now()).Error
}

func (r *Repository) CreateSocial(ctx context.Context, social *models.Social) error {
	return r.db.WithContext(ctx).Create(social).Error
}

func (r *Repository) GetSocial(ctx context.Context, followerID, vloggerID uint) (*models.Social, error) {
	var social models.Social
	err := r.db.WithContext(ctx).Where("follower_id = ? AND vlogger_id = ?", followerID, vloggerID).First(&social).Error
	if err != nil {
		return nil, err
	}
	return &social, nil
}

func (r *Repository) DeleteSocial(ctx context.Context, followerID, vloggerID uint) error {
	return r.db.WithContext(ctx).Where("follower_id = ? AND vlogger_id = ?", followerID, vloggerID).Delete(&models.Social{}).Error
}

func (r *Repository) GetFollowingIDs(ctx context.Context, followerID uint) ([]uint, error) {
	var socials []models.Social
	err := r.db.WithContext(ctx).Where("follower_id = ?", followerID).Find(&socials).Error
	if err != nil {
		return nil, err
	}
	ids := make([]uint, len(socials))
	for i, s := range socials {
		ids[i] = s.VloggerID
	}
	return ids, nil
}

func (r *Repository) GetFollowerIDs(ctx context.Context, vloggerID uint) ([]uint, error) {
	var socials []models.Social
	err := r.db.WithContext(ctx).Where("vlogger_id = ?", vloggerID).Find(&socials).Error
	if err != nil {
		return nil, err
	}
	ids := make([]uint, len(socials))
	for i, s := range socials {
		ids[i] = s.FollowerID
	}
	return ids, nil
}

func (r *Repository) GetFollowing(ctx context.Context, followerID uint, before time.Time, limit int) ([]*models.Account, error) {
	var accounts []*models.Account
	subQuery := r.db.WithContext(ctx).Table("socials").Select("vlogger_id").Where("follower_id = ?", followerID)
	query := r.db.WithContext(ctx).Table("accounts").Where("id IN ?", subQuery)
	if !before.IsZero() {
		query = query.Where("created_at < ?", before)
	}
	err := query.Limit(limit).Find(&accounts).Error
	return accounts, err
}

func (r *Repository) GetFollowers(ctx context.Context, vloggerID uint, before time.Time, limit int) ([]*models.Account, error) {
	var accounts []*models.Account
	subQuery := r.db.WithContext(ctx).Table("socials").Select("follower_id").Where("vlogger_id = ?", vloggerID)
	query := r.db.WithContext(ctx).Table("accounts").Where("id IN ?", subQuery)
	if !before.IsZero() {
		query = query.Where("created_at < ?", before)
	}
	err := query.Limit(limit).Find(&accounts).Error
	return accounts, err
}

func (r *Repository) CountFollowing(ctx context.Context, followerID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Social{}).Where("follower_id = ?", followerID).Count(&count).Error
	return count, err
}

func (r *Repository) CountFollowers(ctx context.Context, vloggerID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Social{}).Where("vlogger_id = ?", vloggerID).Count(&count).Error
	return count, err
}

func (r *Repository) CreateMessage(ctx context.Context, msg *models.Message) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

func (r *Repository) GetMessages(ctx context.Context, fromID, toID uint, before time.Time, limit int) ([]*models.Message, error) {
	var msgs []*models.Message
	query := r.db.WithContext(ctx).Where("deleted_at IS NULL AND ((from_id = ? AND to_id = ?) OR (from_id = ? AND to_id = ?))", fromID, toID, toID, fromID)
	if !before.IsZero() {
		query = query.Where("created_at < ?", before)
	}
	err := query.Order("created_at DESC").Limit(limit).Find(&msgs).Error
	return msgs, err
}

func (r *Repository) MarkMessagesRead(ctx context.Context, fromID, toID uint) error {
	return r.db.WithContext(ctx).Model(&models.Message{}).
		Where("from_id = ? AND to_id = ? AND is_read = ?", fromID, toID, false).
		Update("is_read", true).Error
}

func (r *Repository) GetConversations(ctx context.Context, userID uint) ([]*models.Message, error) {
	var msgs []*models.Message
	subQuery := r.db.WithContext(ctx).Table("messages").
		Select("MAX(id) as id").
		Where("deleted_at IS NULL AND (from_id = ? OR to_id = ?)", userID, userID).
		Group("LEAST(from_id, to_id), GREATEST(from_id, to_id)")
	err := r.db.WithContext(ctx).Where("id IN ?", subQuery).Order("created_at DESC").Find(&msgs).Error
	return msgs, err
}

func (r *Repository) GetOrCreateTag(ctx context.Context, name string) (*models.Tag, error) {
	var tag models.Tag
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&tag).Error
	if err == nil {
		return &tag, nil
	}
	tag = models.Tag{Name: name}
	err = r.db.WithContext(ctx).Create(&tag).Error
	if err != nil {
		var existing models.Tag
		if r.db.WithContext(ctx).Where("name = ?", name).First(&existing) == nil {
			return &existing, nil
		}
		return nil, err
	}
	return &tag, nil
}

func (r *Repository) GetVideoTags(ctx context.Context, videoID uint) ([]string, error) {
	var tags []string
	err := r.db.WithContext(ctx).Table("tags").Select("tags.name").
		Joins("JOIN video_tags ON tags.id = video_tags.tag_id").
		Where("video_tags.video_id = ?", videoID).Find(&tags).Error
	return tags, err
}

func (r *Repository) CreateVideoTag(ctx context.Context, videoID, tagID uint) error {
	vt := models.VideoTag{VideoID: videoID, TagID: tagID}
	return r.db.WithContext(ctx).Create(&vt).Error
}

func (r *Repository) CreateOutboxMsg(ctx context.Context, msg *models.OutboxMsg) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

func (r *Repository) GetPendingOutboxMsgs(ctx context.Context, limit int) ([]*models.OutboxMsg, error) {
	var msgs []*models.OutboxMsg
	err := r.db.WithContext(ctx).Where("status = ?", "pending").Limit(limit).Find(&msgs).Error
	return msgs, err
}

func (r *Repository) UpdateOutboxMsgStatus(ctx context.Context, id uint, status string) error {
	return r.db.WithContext(ctx).Model(&models.OutboxMsg{}).Where("id = ?", id).Update("status", status).Error
}

func (r *Repository) CreateNotification(ctx context.Context, notif *models.Notification) error {
	return r.db.WithContext(ctx).Create(notif).Error
}

func (r *Repository) GetNotifications(ctx context.Context, userID uint, before time.Time, limit int) ([]*models.Notification, error) {
	var notifs []*models.Notification
	query := r.db.WithContext(ctx).Where("recipient_id = ?", userID)
	if !before.IsZero() {
		query = query.Where("created_at < ?", before)
	}
	err := query.Order("created_at DESC").Limit(limit).Find(&notifs).Error
	return notifs, err
}

func (r *Repository) MarkNotificationsRead(ctx context.Context, userID uint, ids []uint) error {
	query := r.db.WithContext(ctx).Model(&models.Notification{}).Where("recipient_id = ?", userID)
	if len(ids) > 0 {
		query = query.Where("id IN ?", ids)
	}
	return query.Update("is_read", true).Error
}

func (r *Repository) CountUnreadNotifications(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Notification{}).Where("recipient_id = ? AND is_read = ?", userID, false).Count(&count).Error
	return count, err
}

func (r *Repository) GetSendersByNotificationIDs(ctx context.Context, ids []uint) (map[uint]*models.Account, error) {
	var notifs []*models.Notification
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&notifs).Error
	if err != nil {
		return nil, err
	}
	senderIDs := make([]uint, len(notifs))
	for i, n := range notifs {
		senderIDs[i] = n.SenderID
	}
	var accounts []*models.Account
	err = r.db.WithContext(ctx).Where("id IN ?", senderIDs).Find(&accounts).Error
	if err != nil {
		return nil, err
	}
	result := make(map[uint]*models.Account)
	for _, acc := range accounts {
		result[acc.ID] = acc
	}
	return result, nil
}

func (r *Repository) GetVideoAuthors(ctx context.Context, videoIDs []uint) (map[uint]*models.Account, error) {
	var videos []*models.Video
	err := r.db.WithContext(ctx).Where("id IN ?", videoIDs).Find(&videos).Error
	if err != nil {
		return nil, err
	}
	authorIDs := make([]uint, len(videos))
	for i, v := range videos {
		authorIDs[i] = v.AuthorID
	}
	var accounts []*models.Account
	err = r.db.WithContext(ctx).Where("id IN ?", authorIDs).Find(&accounts).Error
	if err != nil {
		return nil, err
	}
	result := make(map[uint]*models.Account)
	for _, acc := range accounts {
		result[acc.ID] = acc
	}
	return result, nil
}

func (r *Repository) GetFollowingWithCursor(ctx context.Context, followerID uint, cursor int64, limit int) ([]*models.Account, error) {
	var accounts []*models.Account

	var vloggerIDs []uint
	err := r.db.WithContext(ctx).Model(&models.Social{}).
		Where("follower_id = ?", followerID).
		Pluck("vlogger_id", &vloggerIDs).Error
	if err != nil {
		return nil, err
	}

	if len(vloggerIDs) == 0 {
		return accounts, nil
	}

	query := r.db.WithContext(ctx).Where("id IN ?", vloggerIDs)
	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}
	err = query.Order("id DESC").Limit(limit).Find(&accounts).Error
	return accounts, err
}

func (r *Repository) GetFollowersWithCursor(ctx context.Context, vloggerID uint, cursor int64, limit int) ([]*models.Account, error) {
	var accounts []*models.Account

	var followerIDs []uint
	err := r.db.WithContext(ctx).Model(&models.Social{}).
		Where("vlogger_id = ?", vloggerID).
		Pluck("follower_id", &followerIDs).Error
	if err != nil {
		return nil, err
	}

	if len(followerIDs) == 0 {
		return accounts, nil
	}

	query := r.db.WithContext(ctx).Where("id IN ?", followerIDs)
	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}
	err = query.Order("id DESC").Limit(limit).Find(&accounts).Error
	return accounts, err
}

func (r *Repository) GetHotTags(ctx context.Context, limit int) ([]models.Tag, error) {
	var tags []models.Tag
	err := r.db.WithContext(ctx).
		Table("tags").
		Select("tags.*, COUNT(video_tags.video_id) as video_count").
		Joins("LEFT JOIN video_tags ON tags.id = video_tags.tag_id").
		Group("tags.id").
		Order("video_count DESC").
		Limit(limit).
		Find(&tags).Error
	return tags, err
}