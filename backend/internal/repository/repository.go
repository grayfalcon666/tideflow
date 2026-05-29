package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"tideflow/internal/models"
)

type Repository struct {
	db *gorm.DB
}

// DB returns the underlying GORM DB instance.
func (r *Repository) DB() *gorm.DB {
	return r.db
}

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
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
		"UPDATE videos SET likes_count = GREATEST(likes_count + ?, 0) WHERE id = ? AND deleted_at IS NULL",
		delta, id,
	).Error
}

func (r *Repository) IncrementViewCount(ctx context.Context, id uint, delta int64) error {
	return r.db.WithContext(ctx).Exec(
		"UPDATE videos SET view_count = view_count + ? WHERE id = ? AND deleted_at IS NULL",
		delta, id,
	).Error
}

func (r *Repository) IncrementFollowerCount(ctx context.Context, id uint, delta int64) error {
	return r.db.WithContext(ctx).Exec(
		"UPDATE accounts SET follower_count = GREATEST(follower_count + ?, 0) WHERE id = ?",
		delta, id,
	).Error
}

func (r *Repository) IncrementVideoPopularity(ctx context.Context, id uint, delta int64) error {
	return r.db.WithContext(ctx).Exec(
		"UPDATE videos SET popularity = GREATEST(popularity + ?, 0) WHERE id = ? AND deleted_at IS NULL",
		delta, id,
	).Error
}

func (r *Repository) IncrementCommentCount(ctx context.Context, id uint, delta int64) error {
	return r.db.WithContext(ctx).Exec(
		"UPDATE videos SET comment_count = GREATEST(comment_count + ?, 0) WHERE id = ? AND deleted_at IS NULL",
		delta, id,
	).Error
}

// UpsertLike inserts a like record or reactivates a previously cancelled one (status 0→1).
// Returns (true, nil) if status was actually changed (inserted or 0→1), (false, nil) if already active.
func (r *Repository) UpsertLike(ctx context.Context, videoID, accountID uint) (bool, error) {
	result := r.db.WithContext(ctx).
		Exec(`INSERT INTO likes (video_id, account_id, status, created_at)
              VALUES (?, ?, 1, NOW())
              ON DUPLICATE KEY UPDATE
                status = IF(status = 0, 1, status),
                created_at = IF(status = 0, NOW(), created_at)`,
			videoID, accountID)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// UpdateLikeStatus updates the status of a like record.
// Returns (true, nil) if status was changed, (false, nil) if already at target status.
func (r *Repository) UpdateLikeStatus(ctx context.Context, videoID, accountID uint, status int8) (bool, error) {
	result := r.db.WithContext(ctx).
		Exec(`UPDATE likes SET status = ? WHERE video_id = ? AND account_id = ? AND status != ?`,
			status, videoID, accountID, status)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *Repository) GetLike(ctx context.Context, videoID, accountID uint) (*models.Like, error) {
	var like models.Like
	err := r.db.WithContext(ctx).Where("video_id = ? AND account_id = ? AND status = 1", videoID, accountID).First(&like).Error
	if err != nil {
		return nil, err
	}
	return &like, nil
}

func (r *Repository) GetLikesByAccount(ctx context.Context, accountID uint, before time.Time, limit int) ([]*models.Like, error) {
	var likes []*models.Like
	query := r.db.WithContext(ctx).Where("account_id = ? AND status = 1", accountID)
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
		Where("account_id = ? AND video_id IN ? AND status = 1", accountID, videoIDs).
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

func (r *Repository) GetRepliesByRootIDs(ctx context.Context, rootIDs []uint) ([]*models.Comment, error) {
	var replies []*models.Comment
	err := r.db.WithContext(ctx).
		Where("root_id IN ? AND deleted_at IS NULL", rootIDs).
		Order("created_at ASC").
		Find(&replies).Error
	return replies, err
}

func (r *Repository) SoftDeleteComment(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&models.Comment{}).Where("id = ?", id).Update("deleted_at", time.Now()).Error
}

func (r *Repository) SoftDeleteCommentsByVideo(ctx context.Context, videoID uint) error {
	return r.db.WithContext(ctx).Model(&models.Comment{}).Where("video_id = ?", videoID).Update("deleted_at", time.Now()).Error
}

// FollowOrCreate inserts or updates a follow relationship.
// Uses INSERT ... ON DUPLICATE KEY UPDATE: only updates status (and updated_at) when status was 0.
// Returns (affectedRows, error). affectedRows=1 means new insert or 0→1 change (发MQ).
// affectedRows=0 means already status=1 (不发MQ)。
func (r *Repository) FollowOrCreate(ctx context.Context, followerID, vloggerID uint) (int64, error) {
	res := r.db.WithContext(ctx).Exec(
		"INSERT INTO socials (follower_id, vlogger_id, status, created_at, updated_at) VALUES (?, ?, 1, NOW(), NOW()) "+
			"ON DUPLICATE KEY UPDATE "+
			"status = IF(status = 0, 1, status), "+
			"updated_at = IF(status = 0, NOW(), updated_at)",
		followerID, vloggerID,
	)
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

// UnfollowStatus sets status=0 for an active follow relationship.
// Returns (rows, error). rows=1 means actually unfollowed (发MQ)，=0 means was not following (不发MQ)。
func (r *Repository) UnfollowStatus(ctx context.Context, followerID, vloggerID uint) (int64, error) {
	res := r.db.WithContext(ctx).Exec(
		"UPDATE socials SET status = 0 WHERE follower_id = ? AND vlogger_id = ? AND status = 1",
		followerID, vloggerID,
	)
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

// GetActiveSocial returns the social record only if status=1 (active follow). Used by Worker for idempotency.
func (r *Repository) GetActiveSocial(ctx context.Context, followerID, vloggerID uint) (*models.Social, error) {
	var social models.Social
	err := r.db.WithContext(ctx).Where("follower_id = ? AND vlogger_id = ? AND status = 1", followerID, vloggerID).First(&social).Error
	if err != nil {
		return nil, err
	}
	return &social, nil
}

// GetSocial returns the social record regardless of status.
func (r *Repository) GetSocial(ctx context.Context, followerID, vloggerID uint) (*models.Social, error) {
	var social models.Social
	err := r.db.WithContext(ctx).Where("follower_id = ? AND vlogger_id = ?", followerID, vloggerID).First(&social).Error
	if err != nil {
		return nil, err
	}
	return &social, nil
}

// DeleteSocial is kept for migration compatibility but does not publish MQ events.
func (r *Repository) DeleteSocial(ctx context.Context, followerID, vloggerID uint) error {
	return r.db.WithContext(ctx).Where("follower_id = ? AND vlogger_id = ?", followerID, vloggerID).Delete(&models.Social{}).Error
}

func (r *Repository) GetFollowingIDs(ctx context.Context, followerID uint) ([]uint, error) {
	var socials []models.Social
	err := r.db.WithContext(ctx).Where("follower_id = ? AND status = 1", followerID).Find(&socials).Error
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
	err := r.db.WithContext(ctx).Where("vlogger_id = ? AND status = 1", vloggerID).Find(&socials).Error
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
	subQuery := r.db.WithContext(ctx).Table("socials").Select("vlogger_id").Where("follower_id = ? AND status = 1", followerID)
	query := r.db.WithContext(ctx).Table("accounts").Where("id IN ?", subQuery)
	if !before.IsZero() {
		query = query.Where("created_at < ?", before)
	}
	err := query.Limit(limit).Find(&accounts).Error
	return accounts, err
}

func (r *Repository) GetFollowers(ctx context.Context, vloggerID uint, before time.Time, limit int) ([]*models.Account, error) {
	var accounts []*models.Account
	subQuery := r.db.WithContext(ctx).Table("socials").Select("follower_id").Where("vlogger_id = ? AND status = 1", vloggerID)
	query := r.db.WithContext(ctx).Table("accounts").Where("id IN ?", subQuery)
	if !before.IsZero() {
		query = query.Where("created_at < ?", before)
	}
	err := query.Limit(limit).Find(&accounts).Error
	return accounts, err
}

func (r *Repository) CountFollowing(ctx context.Context, followerID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Social{}).Where("follower_id = ? AND status = 1", followerID).Count(&count).Error
	return count, err
}

func (r *Repository) CountFollowers(ctx context.Context, vloggerID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Social{}).Where("vlogger_id = ? AND status = 1", vloggerID).Count(&count).Error
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
	err := r.db.WithContext(ctx).Raw(`
		SELECT m.*
		FROM messages m
		INNER JOIN (
			SELECT MAX(id) as id
			FROM messages
			WHERE deleted_at IS NULL AND (from_id = ? OR to_id = ?)
			GROUP BY LEAST(from_id, to_id), GREATEST(from_id, to_id)
		) latest ON m.id = latest.id
		ORDER BY m.created_at DESC
	`, userID, userID).Scan(&msgs).Error
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

func (r *Repository) DeleteVideoTags(ctx context.Context, videoID uint) error {
	return r.db.WithContext(ctx).Where("video_id = ?", videoID).Delete(&models.VideoTag{}).Error
}

func (r *Repository) GetVideoByIDUnscoped(ctx context.Context, id uint) (*models.Video, error) {
	var video models.Video
	err := r.db.WithContext(ctx).Unscoped().First(&video, id).Error
	if err != nil {
		return nil, err
	}
	return &video, nil
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
		Where("vlogger_id = ? AND status = 1", vloggerID).
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

func (r *Repository) CreateNote(ctx context.Context, note *models.Note) error {
	return r.db.WithContext(ctx).Create(note).Error
}

func (r *Repository) GetNoteByID(ctx context.Context, id uint) (*models.Note, error) {
	var note models.Note
	err := r.db.WithContext(ctx).First(&note, id).Error
	if err != nil {
		return nil, err
	}
	return &note, nil
}

func (r *Repository) GetNotesByVideo(ctx context.Context, videoID uint, timestampCursor float64, limit int) ([]*models.Note, error) {
	var notes []*models.Note
	query := r.db.WithContext(ctx).Where("video_id = ? AND deleted_at IS NULL", videoID)
	if timestampCursor > 0 {
		query = query.Where("timestamp > ?", timestampCursor)
	}
	err := query.Order("timestamp ASC").Limit(limit).Find(&notes).Error
	return notes, err
}

func (r *Repository) SoftDeleteNote(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&models.Note{}).Where("id = ?", id).Update("deleted_at", time.Now()).Error
}

func (r *Repository) SoftDeleteNotesByVideo(ctx context.Context, videoID uint) error {
	return r.db.WithContext(ctx).Model(&models.Note{}).Where("video_id = ?", videoID).Update("deleted_at", time.Now()).Error
}

func (r *Repository) CountNotesByVideo(ctx context.Context, videoID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Note{}).Where("video_id = ? AND deleted_at IS NULL", videoID).Count(&count).Error
	return count, err
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

// GetVideoSubtitleByVideoID returns the subtitle record for a video.
func (r *Repository) GetVideoSubtitleByVideoID(ctx context.Context, videoID uint) (*models.VideoSubtitle, error) {
	var sub models.VideoSubtitle
	err := r.db.WithContext(ctx).First(&sub, videoID).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// UpsertVideoSubtitle inserts or updates a video subtitle record.
func (r *Repository) UpsertVideoSubtitle(ctx context.Context, sub *models.VideoSubtitle) error {
	return r.db.WithContext(ctx).
		Exec(`INSERT INTO video_subtitles (video_id, subtitles, format, source, version, created_at, updated_at)
		      VALUES (?, ?, ?, ?, 1, NOW(), NOW())
		      ON DUPLICATE KEY UPDATE
		        subtitles = VALUES(subtitles),
		        format = VALUES(format),
		        source = VALUES(source),
		        version = version + 1,
		        updated_at = NOW()`,
			sub.VideoID, sub.Subtitles, sub.Format, sub.Source).Error
}

// GetVideoWordbankByVideoID returns the wordbank record for a video.
func (r *Repository) GetVideoWordbankByVideoID(ctx context.Context, videoID uint) (*models.VideoWordbank, error) {
	var wb models.VideoWordbank
	err := r.db.WithContext(ctx).First(&wb, videoID).Error
	if err != nil {
		return nil, err
	}
	return &wb, nil
}

// UpsertVideoWordbank inserts or updates a video wordbank record.
func (r *Repository) UpsertVideoWordbank(ctx context.Context, wb *models.VideoWordbank) error {
	return r.db.WithContext(ctx).
		Exec(`INSERT INTO video_wordbank (video_id, words, size, version, created_at, updated_at)
		      VALUES (?, ?, ?, 1, NOW(), NOW())
		      ON DUPLICATE KEY UPDATE
		        words = VALUES(words),
		        size = VALUES(size),
		        version = version + 1,
		        updated_at = NOW()`,
			wb.VideoID, wb.Words, wb.Size).Error
}

// CreateVocabList inserts a new vocab list.
func (r *Repository) CreateVocabList(ctx context.Context, list *models.VocabList) error {
	return r.db.WithContext(ctx).Create(list).Error
}

// UpsertVocabList inserts or updates a vocab list (ON DUPLICATE KEY UPDATE).
func (r *Repository) UpsertVocabList(ctx context.Context, list *models.VocabList) error {
	return r.db.WithContext(ctx).
		Exec(`INSERT INTO vocab_lists (name, slug, language, total)
		      VALUES (?, ?, ?, ?)
		      ON DUPLICATE KEY UPDATE
		        name = VALUES(name),
		        total = VALUES(total)`,
			list.Name, list.Slug, list.Language, list.Total).Error
}

// GetAllVocabLists returns all vocab lists.
func (r *Repository) GetAllVocabLists(ctx context.Context) ([]*models.VocabList, error) {
	var lists []*models.VocabList
	err := r.db.WithContext(ctx).Find(&lists).Error
	return lists, err
}

// GetVocabWordsByListID returns all words for a given vocab list.
func (r *Repository) GetVocabWordsByListID(ctx context.Context, listID uint) ([]*models.VocabWord, error) {
	var words []*models.VocabWord
	err := r.db.WithContext(ctx).Where("list_id = ?", listID).Find(&words).Error
	return words, err
}

// GetAllVocabWords returns all vocab words.
func (r *Repository) GetAllVocabWords(ctx context.Context) ([]*models.VocabWord, error) {
	var words []*models.VocabWord
	err := r.db.WithContext(ctx).Find(&words).Error
	return words, err
}

// BatchUpsertVocabWords inserts vocab words in batches, replacing existing ones for the same list.
func (r *Repository) BatchUpsertVocabWords(ctx context.Context, words []*models.VocabWord) error {
	if len(words) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(words, 500).Error
}

// DeleteVocabWordsByListID deletes all words for a given vocab list.
func (r *Repository) DeleteVocabWordsByListID(ctx context.Context, listID uint) error {
	return r.db.WithContext(ctx).Where("list_id = ?", listID).Delete(&models.VocabWord{}).Error
}

// DeleteVocabList deletes a vocab list by ID.
func (r *Repository) DeleteVocabList(ctx context.Context, listID uint) error {
	return r.db.WithContext(ctx).Delete(&models.VocabList{}, listID).Error
}

// =================================================================
// UserWord — 用户词汇本
// =================================================================

// UpsertUserWord inserts or updates a single word status for a user.
func (r *Repository) UpsertUserWord(ctx context.Context, accountID uint, word string, status int8) error {
	return r.db.WithContext(ctx).Exec(
		`INSERT INTO user_words (account_id, word, status, updated_at) VALUES (?, ?, ?, NOW())
		 ON DUPLICATE KEY UPDATE status = VALUES(status), updated_at = NOW()`,
		accountID, word, status,
	).Error
}

// BatchUpsertUserWords bulk-upserts user word statuses within a transaction.
func (r *Repository) BatchUpsertUserWords(ctx context.Context, accountID uint, words []models.UserWord) error {
	if len(words) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range words {
			words[i].AccountID = accountID
		}
		return tx.CreateInBatches(words, 200).Error
	})
}

// UserWordStatus is a lightweight projection of user_words for review-interval filtering.
type UserWordStatus struct {
	Word      string
	Status    int8
	UpdatedAt time.Time
}

// GetUserWordStatuses returns all user word records with status>0 that are in the given word list.
// The caller (service layer) decides which words are within the review window.
func (r *Repository) GetUserWordStatuses(ctx context.Context, accountID uint, words []string) ([]UserWordStatus, error) {
	if len(words) == 0 {
		return nil, nil
	}
	var results []UserWordStatus
	err := r.db.WithContext(ctx).
		Model(&models.UserWord{}).
		Select("word, status, updated_at").
		Where("account_id = ? AND status > 0 AND word IN (?)", accountID, words).
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

// GetUserWordsToday returns all user words practiced today.
func (r *Repository) GetUserWordsToday(ctx context.Context, accountID uint) ([]UserWordStatus, error) {
	var results []UserWordStatus
	today := time.Now().UTC().Truncate(24 * time.Hour)
	err := r.db.WithContext(ctx).
		Model(&models.UserWord{}).
		Select("word, status, updated_at").
		Where("account_id = ? AND updated_at >= ?", accountID, today).
		Order("updated_at DESC").
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

// =================================================================
// UserDailyLearning — 每日学习流水
// =================================================================

// UpsertUserDailyLearning upserts the daily learning rollup for a user.
func (r *Repository) UpsertUserDailyLearning(ctx context.Context, accountID uint, date time.Time, wordsDelta, videosDelta int) error {
	return r.db.WithContext(ctx).Exec(
		`INSERT INTO user_daily_learnings (account_id, date, words_count, videos_count, updated_at)
		 VALUES (?, ?, ?, ?, NOW())
		 ON DUPLICATE KEY UPDATE words_count = words_count + ?, videos_count = videos_count + ?, updated_at = NOW()`,
		accountID, date, wordsDelta, videosDelta, wordsDelta, videosDelta,
	).Error
}

// GetUserDailyLearningByDate returns the daily learning record for a user on a specific date.
func (r *Repository) GetUserDailyLearningByDate(ctx context.Context, accountID uint, date time.Time) (*models.UserDailyLearning, error) {
	var record models.UserDailyLearning
	err := r.db.WithContext(ctx).
		Where("account_id = ? AND date = ?", accountID, date).
		First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

// GetUserWordStatusesAll returns all user word records (including status=0) for the given words.
func (r *Repository) GetUserWordStatusesAll(ctx context.Context, accountID uint, words []string) ([]UserWordStatus, error) {
	if len(words) == 0 {
		return nil, nil
	}
	var results []UserWordStatus
	err := r.db.WithContext(ctx).
		Model(&models.UserWord{}).
		Select("word, status, updated_at").
		Where("account_id = ? AND word IN (?)", accountID, words).
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

// GetUserDailyLearnings returns all daily learning records for a user in a given year.
func (r *Repository) GetUserDailyLearnings(ctx context.Context, accountID uint, year int) ([]*models.UserDailyLearning, error) {
	var records []*models.UserDailyLearning
	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	err := r.db.WithContext(ctx).
		Where("account_id = ? AND date >= ? AND date < ?", accountID, startDate, endDate).
		Order("date ASC").
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	return records, nil
}