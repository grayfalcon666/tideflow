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
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

func (Account) TableName() string {
	return "accounts"
}

type Video struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	AuthorID     uint           `gorm:"index;not null" json:"author_id"`
	Username     string         `gorm:"size:255;not null" json:"username"`
	Title        string         `gorm:"size:255;not null" json:"title"`
	Description  string         `gorm:"size:255" json:"description"`
	PlayURL      string         `gorm:"size:255;not null" json:"play_url"`
	CoverURL     string         `gorm:"size:255;not null" json:"cover_url"`
	CreateTime   time.Time      `gorm:"index" json:"create_time"`
	UpdateTime   time.Time      `json:"update_time"`
	LikesCount   int64          `gorm:"not null;default:0" json:"likes_count"`
	Popularity   int64          `gorm:"not null;default:0" json:"popularity"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

func (Video) TableName() string {
	return "videos"
}

type Like struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	VideoID   uint      `gorm:"not null;uniqueIndex:idx_like_video_account" json:"video_id"`
	AccountID uint      `gorm:"not null;uniqueIndex:idx_like_video_account" json:"account_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (Like) TableName() string {
	return "likes"
}

type Comment struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	VideoID   uint           `gorm:"index;not null" json:"video_id"`
	AuthorID  uint           `gorm:"index;not null" json:"author_id"`
	Username  string         `gorm:"size:255;index" json:"username"`
	RootID    uint           `gorm:"not null;default:0" json:"root_id"`
	ParentID  uint           `gorm:"not null;default:0" json:"parent_id"`
	Content   string         `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time      `gorm:"index" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

func (Comment) TableName() string {
	return "comments"
}

type Social struct {
	ID         uint `gorm:"primaryKey" json:"id"`
	FollowerID uint `gorm:"index;not null" json:"follower_id"`
	VloggerID  uint `gorm:"index;not null" json:"vlogger_id"`
}

func (Social) TableName() string {
	return "socials"
}

type Message struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	FromID    uint           `gorm:"index;not null" json:"from_id"`
	ToID      uint           `gorm:"index;not null" json:"to_id"`
	Content   string         `gorm:"type:text;not null" json:"content"`
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