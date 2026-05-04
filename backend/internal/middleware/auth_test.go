package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestExtractToken(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		queryToken string
		want       string
	}{
		{"empty", "", "", ""},
		{"bearer token", "Bearer abc123", "", "abc123"},
		{"raw token", "abc123", "", "abc123"},
		{"query param", "", "token=xyz", "xyz"},
		{"bearer overrides query", "Bearer tok1", "token=xyz", "tok1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				c.Request.Header.Set("Authorization", tt.authHeader)
			}
			if tt.queryToken != "" {
				c.Request, _ = http.NewRequest(http.MethodGet, "/?token=xyz", nil)
				if tt.authHeader != "" {
					c.Request.Header.Set("Authorization", tt.authHeader)
				}
			}

			got := extractToken(c)
			if got != tt.want {
				t.Errorf("extractToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(c *gin.Context)
		want   uint
	}{
		{"no user_id set", func(c *gin.Context) {}, 0},
		{"user_id set", func(c *gin.Context) { c.Set("user_id", uint(123)) }, 123},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setup(c)

			got := GetUserID(c)
			if got != tt.want {
				t.Errorf("GetUserID() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNewAuthMiddleware(t *testing.T) {
	// Nil auth service - just test construction
	m := NewAuthMiddleware(nil)
	if m == nil {
		t.Error("NewAuthMiddleware returned nil")
	}
}

func TestNewRateLimitMiddleware(t *testing.T) {
	// Nil limiter - just test construction
	m := NewRateLimitMiddleware(nil)
	if m == nil {
		t.Error("NewRateLimitMiddleware returned nil")
	}
}
