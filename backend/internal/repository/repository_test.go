package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func newMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}
	return gormDB, mock
}

func TestGetAccountByUsername(t *testing.T) {
	gormDB, mock := newMockDB(t)
	repo := New(gormDB)

	// GORM adds LIMIT 1 and ORDER BY for First()
	rows := sqlmock.NewRows([]string{"id", "username", "password", "follower_count", "avatar_url", "bio", "token", "refresh_token", "created_at", "updated_at"}).
		AddRow(1, "alice", "hashedpass", 100, "", "", "", "", time.Now(), time.Now())
	mock.ExpectQuery("SELECT \\* FROM `accounts`").
		WillReturnRows(rows)

	acc, err := repo.GetAccountByUsername(context.Background(), "alice")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if acc.Username != "alice" {
		t.Errorf("username = %q, want alice", acc.Username)
	}
}

func TestGetAccountByID(t *testing.T) {
	gormDB, mock := newMockDB(t)
	repo := New(gormDB)

	rows := sqlmock.NewRows([]string{"id", "username", "password", "follower_count", "avatar_url", "bio", "token", "refresh_token", "created_at", "updated_at"}).
		AddRow(uint(1), "alice", "hashedpass", 100, "", "", "", "", time.Now(), time.Now())
	mock.ExpectQuery("SELECT \\* FROM `accounts`").
		WillReturnRows(rows)

	acc, err := repo.GetAccountByID(context.Background(), 1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if acc.FollowerCount != 100 {
		t.Errorf("follower_count = %d, want 100", acc.FollowerCount)
	}
}

func TestGetAccountByIDNotFound(t *testing.T) {
	gormDB, mock := newMockDB(t)
	repo := New(gormDB)

	mock.ExpectQuery("SELECT \\* FROM `accounts`").
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := repo.GetAccountByID(context.Background(), 999)
	if err == nil {
		t.Error("expected error for not found")
	}
}

func TestGetLike(t *testing.T) {
	gormDB, mock := newMockDB(t)
	repo := New(gormDB)

	rows := sqlmock.NewRows([]string{"id", "video_id", "account_id", "created_at"}).
		AddRow(uint(1), uint(10), uint(1), time.Now())
	// GORM First() adds ORDER BY id LIMIT 1
	mock.ExpectQuery("SELECT \\* FROM `likes`").
		WillReturnRows(rows)

	like, err := repo.GetLike(context.Background(), 10, 1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if like.VideoID != 10 {
		t.Errorf("video_id = %d, want 10", like.VideoID)
	}
}

func TestGetLikeNotFound(t *testing.T) {
	gormDB, mock := newMockDB(t)
	repo := New(gormDB)

	mock.ExpectQuery("SELECT \\* FROM `likes`").
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := repo.GetLike(context.Background(), 999, 1)
	if err == nil {
		t.Error("expected error for not found")
	}
}

func TestDeleteLike(t *testing.T) {
	gormDB, mock := newMockDB(t)
	repo := New(gormDB)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM `likes`").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.DeleteLike(context.Background(), 10, 1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetComment(t *testing.T) {
	gormDB, mock := newMockDB(t)
	repo := New(gormDB)

	rows := sqlmock.NewRows([]string{"id", "video_id", "author_id", "username", "root_id", "parent_id", "content", "created_at", "deleted_at"}).
		AddRow(uint(5), uint(1), uint(2), "alice", uint(0), uint(0), "nice video", time.Now(), nil)
	mock.ExpectQuery("SELECT \\* FROM `comments`").
		WillReturnRows(rows)

	comment, err := repo.GetComment(context.Background(), 5)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if comment.Content != "nice video" {
		t.Errorf("content = %q, want 'nice video'", comment.Content)
	}
}

func TestCountFollowing(t *testing.T) {
	gormDB, mock := newMockDB(t)
	repo := New(gormDB)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(50)
	// GORM adds backtick quoting: `socials`
	mock.ExpectQuery("SELECT count").
		WillReturnRows(rows)

	count, err := repo.CountFollowing(context.Background(), 1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if count != 50 {
		t.Errorf("count = %d, want 50", count)
	}
}

func TestCountFollowers(t *testing.T) {
	gormDB, mock := newMockDB(t)
	repo := New(gormDB)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(200)
	mock.ExpectQuery("SELECT count").
		WillReturnRows(rows)

	count, err := repo.CountFollowers(context.Background(), 2)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if count != 200 {
		t.Errorf("count = %d, want 200", count)
	}
}

func TestGetOrCreateTagExisting(t *testing.T) {
	gormDB, mock := newMockDB(t)
	repo := New(gormDB)

	rows := sqlmock.NewRows([]string{"id", "name"}).AddRow(uint(2), "music")
	mock.ExpectQuery("SELECT \\* FROM `tags`").
		WillReturnRows(rows)

	tag, err := repo.GetOrCreateTag(context.Background(), "music")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if tag.ID != 2 {
		t.Errorf("id = %d, want 2", tag.ID)
	}
}

func TestUpdateAccount(t *testing.T) {
	gormDB, mock := newMockDB(t)
	repo := New(gormDB)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `accounts`").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.UpdateAccount(context.Background(), 5, map[string]interface{}{"follower_count": 200})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSoftDeleteVideo(t *testing.T) {
	gormDB, mock := newMockDB(t)
	repo := New(gormDB)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `videos`").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.SoftDeleteVideo(context.Background(), 10)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCountUnreadNotifications(t *testing.T) {
	gormDB, mock := newMockDB(t)
	repo := New(gormDB)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(7)
	mock.ExpectQuery("SELECT count").
		WillReturnRows(rows)

	count, err := repo.CountUnreadNotifications(context.Background(), 1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if count != 7 {
		t.Errorf("count = %d, want 7", count)
	}
}

func TestGetConversations(t *testing.T) {
	gormDB, mock := newMockDB(t)
	repo := New(gormDB)

	// Subquery for latest message per conversation
	subRows := sqlmock.NewRows([]string{"id"}).AddRow(uint(10))
	mock.ExpectQuery("SELECT MAX").
		WillReturnRows(subRows)

	// Main query for the messages - GORM adds soft delete WHERE
	msgRows := sqlmock.NewRows([]string{"id", "from_id", "to_id", "content", "is_read", "created_at", "deleted_at"}).
		AddRow(uint(10), uint(2), uint(1), "hello", false, time.Now(), nil)
	mock.ExpectQuery("SELECT \\* FROM `messages`").
		WillReturnRows(msgRows)

	msgs, err := repo.GetConversations(context.Background(), 1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(msgs) != 1 {
		t.Errorf("len = %d, want 1", len(msgs))
	}
}

func TestGetLatestVideosNoCursor(t *testing.T) {
	gormDB, mock := newMockDB(t)
	repo := New(gormDB)

	// GORM: WHERE deleted_at IS NULL ORDER BY create_time DESC LIMIT
	rows := sqlmock.NewRows([]string{"id", "author_id", "username", "title", "description", "play_url", "cover_url", "create_time", "update_time", "likes_count", "popularity"}).
		AddRow(uint(1), uint(1), "alice", "Vid1", "", "", "", time.Now(), time.Now(), 0, 0).
		AddRow(uint(2), uint(1), "alice", "Vid2", "", "", "", time.Now(), time.Now(), 0, 0)
	mock.ExpectQuery("SELECT \\* FROM `videos`").
		WillReturnRows(rows)

	videos, err := repo.GetLatestVideos(context.Background(), time.Time{}, 3)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(videos) != 2 {
		t.Errorf("len = %d, want 2", len(videos))
	}
}

func TestGetVideoByID(t *testing.T) {
	gormDB, mock := newMockDB(t)
	repo := New(gormDB)

	rows := sqlmock.NewRows([]string{"id", "author_id", "username", "title", "description", "play_url", "cover_url", "create_time", "update_time", "likes_count", "popularity", "deleted_at"}).
		AddRow(uint(10), uint(1), "alice", "My Video", "desc", "/v/1.mp4", "/c/1.jpg", time.Now(), time.Now(), 10, 50, nil)
	mock.ExpectQuery("SELECT \\* FROM `videos`").
		WillReturnRows(rows)

	video, err := repo.GetVideoByID(context.Background(), 10)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if video.Title != "My Video" {
		t.Errorf("title = %q, want 'My Video'", video.Title)
	}
}

func TestGetVideoByIDNotFound(t *testing.T) {
	gormDB, mock := newMockDB(t)
	repo := New(gormDB)

	mock.ExpectQuery("SELECT \\* FROM `videos`").
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := repo.GetVideoByID(context.Background(), 999)
	if err == nil {
		t.Error("expected error for not found")
	}
}

func TestGetPopularVideos(t *testing.T) {
	gormDB, mock := newMockDB(t)
	repo := New(gormDB)

	rows := sqlmock.NewRows([]string{"id", "author_id", "username", "title", "description", "play_url", "cover_url", "create_time", "update_time", "likes_count", "popularity", "deleted_at"}).
		AddRow(uint(1), uint(1), "alice", "Vid1", "", "", "", time.Now(), time.Now(), 100, 500, nil).
		AddRow(uint(2), uint(2), "bob", "Vid2", "", "", "", time.Now(), time.Now(), 50, 300, nil)
	mock.ExpectQuery("SELECT \\* FROM `videos`").
		WillReturnRows(rows)

	videos, err := repo.GetPopularVideos(context.Background(), 0, 10)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(videos) != 2 {
		t.Errorf("len = %d, want 2", len(videos))
	}
}

func TestGetCommentsByVideoFirstLevel(t *testing.T) {
	gormDB, mock := newMockDB(t)
	repo := New(gormDB)

	// GORM adds soft delete clause
	rows := sqlmock.NewRows([]string{"id", "video_id", "author_id", "username", "root_id", "parent_id", "content", "created_at", "deleted_at"}).
		AddRow(uint(1), uint(1), uint(1), "alice", uint(0), uint(0), "top level", time.Now(), nil).
		AddRow(uint(2), uint(1), uint(2), "bob", uint(1), uint(1), "reply", time.Now(), nil)
	mock.ExpectQuery("SELECT \\* FROM `comments`").
		WillReturnRows(rows)

	comments, err := repo.GetCommentsByVideo(context.Background(), 1, 0, time.Time{}, 5)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(comments) != 2 {
		t.Errorf("len = %d, want 2", len(comments))
	}
}

func TestGetFollowingIDs(t *testing.T) {
	gormDB, mock := newMockDB(t)
	repo := New(gormDB)

	rows := sqlmock.NewRows([]string{"id", "follower_id", "vlogger_id"}).
		AddRow(uint(1), uint(1), uint(10)).
		AddRow(uint(2), uint(1), uint(20))
	mock.ExpectQuery("SELECT \\* FROM `socials`").
		WillReturnRows(rows)

	ids, err := repo.GetFollowingIDs(context.Background(), 1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 10 || ids[1] != 20 {
		t.Errorf("ids = %v, want [10 20]", ids)
	}
}

func TestGetFollowerIDs(t *testing.T) {
	gormDB, mock := newMockDB(t)
	repo := New(gormDB)

	rows := sqlmock.NewRows([]string{"id", "follower_id", "vlogger_id"}).
		AddRow(uint(1), uint(1), uint(5)).
		AddRow(uint(2), uint(2), uint(5))
	mock.ExpectQuery("SELECT \\* FROM `socials`").
		WillReturnRows(rows)

	ids, err := repo.GetFollowerIDs(context.Background(), 5)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
		t.Errorf("ids = %v, want [1 2]", ids)
	}
}

func TestGetVideoTags(t *testing.T) {
	gormDB, mock := newMockDB(t)
	repo := New(gormDB)

	rows := sqlmock.NewRows([]string{"name"}).AddRow("搞笑").AddRow("音乐")
	mock.ExpectQuery("SELECT tags.name FROM `tags`").
		WillReturnRows(rows)

	tags, err := repo.GetVideoTags(context.Background(), 1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(tags) != 2 || tags[0] != "搞笑" {
		t.Errorf("tags = %v, want [搞笑 音乐]", tags)
	}
}
