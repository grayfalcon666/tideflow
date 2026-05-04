package service

import (
	"testing"
)

func TestUserProfileStruct(t *testing.T) {
	up := &UserProfile{
		ID:            1,
		Username:      "alice",
		AvatarURL:     "http://example.com/alice.jpg",
		Bio:           "Hello world",
		FollowerCount: 1000,
		IsBigV:        true,
	}

	if up.ID != 1 {
		t.Errorf("UserProfile.ID = %d, want 1", up.ID)
	}
	if up.Username != "alice" {
		t.Errorf("UserProfile.Username = %q, want alice", up.Username)
	}
	if up.FollowerCount != 1000 {
		t.Errorf("UserProfile.FollowerCount = %d, want 1000", up.FollowerCount)
	}
	if !up.IsBigV {
		t.Error("UserProfile.IsBigV = false, want true")
	}
}

func TestUserProfileIsBigVThreshold(t *testing.T) {
	tests := []struct {
		followerCount int
		isBigV        bool
	}{
		{0, false},
		{999, false},
		{1000, true},
		{5000, true},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			up := &UserProfile{FollowerCount: tt.followerCount, IsBigV: tt.followerCount >= 1000}
			if up.IsBigV != tt.isBigV {
				t.Errorf("FollowerCount=%d: IsBigV=%v, want %v", tt.followerCount, up.IsBigV, tt.isBigV)
			}
		})
	}
}

func TestAuthServiceErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrUserExists", ErrUserExists},
		{"ErrInvalidPassword", ErrInvalidPassword},
		{"ErrInvalidToken", ErrInvalidToken},
		{"ErrTokenExpired", ErrTokenExpired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
			if tt.err.Error() == "" {
				t.Errorf("%s.Error() should not be empty", tt.name)
			}
		})
	}
}
