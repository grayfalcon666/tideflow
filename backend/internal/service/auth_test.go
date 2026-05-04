package service

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordHashing(t *testing.T) {
	password := "testpassword123"

	// Hash the password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	// Correct password should match
	err = bcrypt.CompareHashAndPassword(hash, []byte(password))
	if err != nil {
		t.Error("correct password should match hash")
	}

	// Wrong password should not match
	err = bcrypt.CompareHashAndPassword(hash, []byte("wrongpassword"))
	if err == nil {
		t.Error("wrong password should not match hash")
	}
}

func TestPasswordHashingDifferentPasswords(t *testing.T) {
	password1 := "password1"
	password2 := "password2"

	hash1, _ := bcrypt.GenerateFromPassword([]byte(password1), bcrypt.DefaultCost)
	hash2, _ := bcrypt.GenerateFromPassword([]byte(password2), bcrypt.DefaultCost)

	// Hashes should be different for different passwords
	if string(hash1) == string(hash2) {
		t.Error("different passwords should produce different hashes")
	}
}

func TestBigVThreshold(t *testing.T) {
	// Test BigV threshold logic: followers >= threshold = BigV
	threshold := 10000

	tests := []struct {
		followers int
		isBigV    bool
	}{
		{0, false},
		{9999, false},
		{10000, true},
		{20000, true},
	}

	for _, tt := range tests {
		isBigV := tt.followers >= threshold
		if isBigV != tt.isBigV {
			t.Errorf("followers=%d: isBigV=%v, want %v", tt.followers, isBigV, tt.isBigV)
		}
	}
}
