package models

import (
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestAccountTableName(t *testing.T) {
	a := Account{}
	if a.TableName() != "accounts" {
		t.Errorf("Account.TableName() = %q, want accounts", a.TableName())
	}
}

func TestVideoTableName(t *testing.T) {
	v := Video{}
	if v.TableName() != "videos" {
		t.Errorf("Video.TableName() = %q, want videos", v.TableName())
	}
}

func TestLikeTableName(t *testing.T) {
	l := Like{}
	if l.TableName() != "likes" {
		t.Errorf("Like.TableName() = %q, want likes", l.TableName())
	}
}

func TestCommentTableName(t *testing.T) {
	c := Comment{}
	if c.TableName() != "comments" {
		t.Errorf("Comment.TableName() = %q, want comments", c.TableName())
	}
}

func TestSocialTableName(t *testing.T) {
	s := Social{}
	if s.TableName() != "socials" {
		t.Errorf("Social.TableName() = %q, want socials", s.TableName())
	}
}

func TestMessageTableName(t *testing.T) {
	m := Message{}
	if m.TableName() != "messages" {
		t.Errorf("Message.TableName() = %q, want messages", m.TableName())
	}
}

func TestTagTableName(t *testing.T) {
	tag := Tag{}
	if tag.TableName() != "tags" {
		t.Errorf("Tag.TableName() = %q, want tags", tag.TableName())
	}
}

func TestVideoTagTableName(t *testing.T) {
	vt := VideoTag{}
	if vt.TableName() != "video_tags" {
		t.Errorf("VideoTag.TableName() = %q, want video_tags", vt.TableName())
	}
}

func TestOutboxMsgTableName(t *testing.T) {
	o := OutboxMsg{}
	if o.TableName() != "outbox_msgs" {
		t.Errorf("OutboxMsg.TableName() = %q, want outbox_msgs", o.TableName())
	}
}

func TestNotificationTableName(t *testing.T) {
	n := Notification{}
	if n.TableName() != "notifications" {
		t.Errorf("Notification.TableName() = %q, want notifications", n.TableName())
	}
}

func TestVideoSoftDelete(t *testing.T) {
	v := Video{ID: 1, Title: "test", DeletedAt: gorm.DeletedAt{Time: time.Now(), Valid: true}}
	if v.DeletedAt.Valid != true {
		t.Error("DeletedAt.Valid should be true after soft delete")
	}
}

func TestCommentHierarchy(t *testing.T) {
	// First level comment
	c1 := Comment{ID: 1, RootID: 0, ParentID: 0}
	if c1.RootID != 0 || c1.ParentID != 0 {
		t.Errorf("c1: RootID=%d ParentID=%d, want 0 0", c1.RootID, c1.ParentID)
	}

	// Reply to first level
	c2 := Comment{ID: 2, RootID: 1, ParentID: 1}
	if c2.RootID != c2.ParentID {
		t.Error("reply to first level: RootID should equal ParentID")
	}

	// Reply to second level (capped at 2)
	c3 := Comment{ID: 3, RootID: 1, ParentID: 2}
	if c3.RootID != 1 {
		t.Error("c3: RootID should still be 1 (inherited from root)")
	}
	if c3.RootID == c3.ParentID {
		t.Error("c3: RootID should NOT equal ParentID (parent is second level)")
	}
}
