package models

import (
	"time"

	"gorm.io/gorm"
)

type Account struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Username      string         `gorm:"size:255;uniqueIndex;not null" json:"username"`
	Password      string         `gorm:"size:255;not null" json:"-"`
	Token         string         `gorm:"size:255" json:"-"`
	RefreshToken  string         `gorm:"size:255" json:"-"`
	AvatarURL     string         `gorm:"size:512" json:"avatar_url"`
	Bio           string         `gorm:"size:255" json:"bio"`
	FollowerCount int            `gorm:"not null;default:0" json:"follower_count"`
	LikesPublic   bool           `gorm:"not null;default:true" json:"likes_public"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

func (Account) TableName() string {
	return "accounts"
}

type Video struct {
	ID           uint           `gorm:"primaryKey;index:idx_videos_likes_count_id,priority:2;index:idx_videos_popularity_time_id,priority:3" json:"id"`
	AuthorID     uint           `gorm:"index:idx_videos_author_create,priority:1;not null" json:"author_id"`
	Username     string         `gorm:"size:255;not null" json:"username"`
	Title        string         `gorm:"size:255;not null" json:"title"`
	Description  string         `gorm:"size:255" json:"description"`
	PlayURL      string         `gorm:"size:255;not null" json:"play_url"`
	CoverURL     string         `gorm:"size:255;not null" json:"cover_url"`
	Duration     float64        `gorm:"type:float;default:0" json:"duration"`
	Width        int            `gorm:"type:int;default:0" json:"width"`
	Height       int            `gorm:"type:int;default:0" json:"height"`
	CreateTime   time.Time      `gorm:"index:idx_videos_author_create,priority:2;index:idx_videos_popularity_time_id,priority:2" json:"create_time"`
	UpdateTime   time.Time      `json:"update_time"`
	LikesCount   int64          `gorm:"not null;default:0;index:idx_videos_likes_count_id,priority:1" json:"likes_count"`
	CommentCount int64          `gorm:"not null;default:0" json:"comment_count"`
	ViewCount    int64          `gorm:"not null;default:0" json:"view_count"`
	Popularity   int64          `gorm:"not null;default:0;index:idx_videos_popularity_time_id,priority:1" json:"popularity"`
	SubtitleStatus string         `gorm:"size:20;not null;default:'none'" json:"subtitle_status"`
	WordbankStatus string         `gorm:"size:20;not null;default:'none'" json:"wordbank_status"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	// 虚拟字段（非持久化），由 feed service 填充
	AvatarURL string `gorm:"-" json:"avatar_url,omitempty"`
	IsBigV    bool   `gorm:"-" json:"is_big_v,omitempty"`
}

func (Video) TableName() string {
	return "videos"
}

type Like struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	VideoID   uint      `gorm:"not null;uniqueIndex:idx_like_video_account" json:"video_id"`
	AccountID uint      `gorm:"not null;uniqueIndex:idx_like_video_account" json:"account_id"`
	Status    int8      `gorm:"not null;default:1" json:"status"` // 1=点赞, 0=取消
	CreatedAt time.Time `json:"created_at"`
}

func (Like) TableName() string {
	return "likes"
}

type Comment struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	VideoID   uint           `gorm:"not null;index:idx_comments_video_root,priority:1" json:"video_id"`
	AuthorID  uint           `gorm:"index;not null" json:"author_id"`
	Username  string         `gorm:"size:255;index" json:"username"`
	RootID    uint           `gorm:"not null;default:0;index:idx_comments_video_root,priority:2" json:"root_id"`
	ParentID  uint           `gorm:"not null;default:0;index:idx_comments_parent_id,priority:1" json:"parent_id"`
	Content   string         `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time      `gorm:"index" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

func (Comment) TableName() string {
	return "comments"
}

type Social struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	FollowerID uint           `gorm:"column:follower_id;not null;uniqueIndex:idx_social_follower_vlogger,priority:1" json:"follower_id"`
	VloggerID  uint           `gorm:"column:vlogger_id;not null;uniqueIndex:idx_social_follower_vlogger,priority:2" json:"vlogger_id"`
	Status     int            `gorm:"column:status;not null;default:1" json:"status"` // 1=已关注, 0=未关注
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

func (Social) TableName() string {
	return "socials"
}

type Message struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	FromID    uint           `gorm:"index;not null" json:"from_id"`
	ToID      uint           `gorm:"index;not null" json:"to_id"`
	Content   string         `gorm:"type:text;not null" json:"content"`
	MsgType   string         `gorm:"size:20;default:text;not null" json:"msg_type"`
	IsRead    bool           `gorm:"default:false" json:"is_read"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

func (Message) TableName() string {
	return "messages"
}

type Tag struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Name string `gorm:"size:100;uniqueIndex;not null" json:"name"`
}

func (Tag) TableName() string {
	return "tags"
}

type VideoTag struct {
	ID     uint `gorm:"primaryKey" json:"id"`
	VideoID uint `gorm:"index;not null" json:"video_id"`
	TagID   uint `gorm:"index;not null" json:"tag_id"`
}

func (VideoTag) TableName() string {
	return "video_tags"
}

type OutboxMsg struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	VideoID   uint      `gorm:"index" json:"video_id"`
	AuthorID  uint      `gorm:"index" json:"author_id"`
	EventType string    `gorm:"size:50;not null" json:"event_type"`
	CreateTime time.Time `gorm:"not null" json:"create_time"`
	Status    string    `gorm:"size:50;index;default:pending" json:"status"`
}

func (OutboxMsg) TableName() string {
	return "outbox_msgs"
}

type Notification struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	RecipientID uint      `gorm:"index;not null" json:"recipient_id"`
	SenderID    uint      `gorm:"not null" json:"sender_id"`
	Type        string    `gorm:"size:50;not null" json:"type"`
	TargetID    uint      `json:"target_id"`
	Content     string    `gorm:"size:255" json:"content"`
	IsRead      bool      `gorm:"default:false" json:"is_read"`
	CreatedAt   time.Time `gorm:"not null" json:"created_at"`
}

func (Notification) TableName() string {
	return "notifications"
}

type Note struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	VideoID   uint           `gorm:"not null;index:idx_notes_video_timestamp,priority:1" json:"video_id"`
	AuthorID  uint           `gorm:"not null" json:"author_id"`
	Username  string         `gorm:"size:255;not null" json:"username"`
	Timestamp float64        `gorm:"type:float;not null;index:idx_notes_video_timestamp,priority:2" json:"timestamp"`
	Content   string         `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

func (Note) TableName() string {
	return "notes"
}

type VideoSubtitle struct {
	VideoID   uint      `gorm:"primaryKey" json:"video_id"`
	Subtitles string    `gorm:"type:json;not null" json:"subtitles"`
	Format    string    `gorm:"size:20;not null;default:'json'" json:"format"`
	Source    string    `gorm:"size:20;not null" json:"source"`
	Version   int       `gorm:"not null;default:1" json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (VideoSubtitle) TableName() string {
	return "video_subtitles"
}

type VideoWordbank struct {
	VideoID   uint      `gorm:"primaryKey" json:"video_id"`
	Words     string    `gorm:"type:json;not null" json:"words"`
	Size      int       `gorm:"not null;default:0" json:"size"`
	Version   int       `gorm:"not null;default:1" json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (VideoWordbank) TableName() string {
	return "video_wordbank"
}

type VocabList struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Name     string `gorm:"size:100;uniqueIndex;not null" json:"name"`
	Slug     string `gorm:"size:50;uniqueIndex;not null" json:"slug"`
	Language string `gorm:"size:20;not null;default:'en'" json:"language"`
	Total    int    `gorm:"not null;default:0" json:"total"`
}

func (VocabList) TableName() string {
	return "vocab_lists"
}

type VocabWord struct {
	ID     uint   `gorm:"primaryKey" json:"-"`
	ListID uint   `gorm:"index:idx_vocab_words_list_word,priority:1;not null" json:"list_id"`
	Word   string `gorm:"size:100;index:idx_vocab_words_list_word,priority:2;not null" json:"word"`
}

func (VocabWord) TableName() string {
	return "vocab_words"
}