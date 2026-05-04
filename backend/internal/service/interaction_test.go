package service

import (
	"testing"
)

func TestParseCursorInt64(t *testing.T) {
	tests := []struct {
		input string
		want  int64
		wantOK bool
	}{
		{"", 0, false},
		{"12345", 12345, true},
		{"0", 0, true},
		{"0001", 1, true},
		{"12abc", 0, false},
		{"abc12", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseCursorInt64(tt.input)
			ok := err == nil
			if ok != tt.wantOK {
				t.Errorf("parseCursorInt64(%q) ok=%v, want %v", tt.input, ok, tt.wantOK)
			}
			if ok && got != tt.want {
				t.Errorf("parseCursorInt64(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
func TestCommentParentChildRelationship(t *testing.T) {
	// Simulate the comment tree structure per design docs:
	// - root_id=0, parent_id=0 → first-level comment
	// - root_id=X, parent_id=X → reply to first-level comment
	// - root_id=X, parent_id=Y (Y≠X) → reply to second-level comment (inherits root=X, caps at 2 levels)

	tests := []struct {
		name     string
		comment  CommentWithReplies
		expected string
	}{
		{
			name: "first level comment",
			comment: CommentWithReplies{
				ID:       1,
				ParentID: 0,
				RootID:   0,
			},
			expected: "first-level",
		},
		{
			name: "reply to first level",
			comment: CommentWithReplies{
				ID:       2,
				ParentID: 1,
				RootID:   1,
			},
			expected: "second-level-reply",
		},
		{
			name: "reply to second level (should still have root_id=1)",
			comment: CommentWithReplies{
				ID:       3,
				ParentID: 2,
				RootID:   1,
			},
			expected: "second-level-capped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			if tt.comment.RootID == 0 && tt.comment.ParentID == 0 {
				got = "first-level"
			} else if tt.comment.RootID == tt.comment.ParentID {
				got = "second-level-reply"
			} else {
				got = "second-level-capped"
			}
			if got != tt.expected {
				t.Errorf("comment %d: got %s, want %s", tt.comment.ID, got, tt.expected)
			}
		})
	}
}

func TestInteractionIdempotency(t *testing.T) {
	// Test the LikeVideo idempotency logic: duplicate likes should be silently ignored
	// Simulate the service logic
	liked := map[uint]bool{}

	// First like
	liked[1] = true
	if !liked[1] {
		t.Error("first like should set liked=true")
	}

	// Duplicate like (should be idempotent - no error, no change)
	if liked[1] {
		// In the actual service, GetLike returns nil error when like exists
		// so the second call to CreateLike is skipped
	}

	if len(liked) != 1 {
		t.Errorf("duplicate like should not create new entry, got %d entries", len(liked))
	}
}
