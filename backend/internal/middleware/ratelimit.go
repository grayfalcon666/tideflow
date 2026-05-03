package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"tideflow/infra/redis"
	"tideflow/pkg/response"
)

type RateLimitMiddleware struct {
	limiter *redis.RateLimiter
}

func NewRateLimitMiddleware(limiter *redis.RateLimiter) *RateLimitMiddleware {
	return &RateLimitMiddleware{limiter: limiter}
}

func (m *RateLimitMiddleware) LoginLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		allowed, _, err := m.limiter.Allow(c.Request.Context(), "account_login", ip, 10, time.Minute)
		if err != nil {
			c.Next()
			return
		}
		if !allowed {
			response.TooManyRequests(c)
			c.Abort()
			return
		}
		c.Next()
	}
}

func (m *RateLimitMiddleware) RegisterLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		allowed, _, err := m.limiter.Allow(c.Request.Context(), "account_register", ip, 5, time.Hour)
		if err != nil {
			c.Next()
			return
		}
		if !allowed {
			response.TooManyRequests(c)
			c.Abort()
			return
		}
		c.Next()
	}
}

func (m *RateLimitMiddleware) LikeLimit(userID uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed, _, err := m.limiter.Allow(c.Request.Context(), "like_write", string(rune(userID)), 30, time.Minute)
		if err != nil {
			c.Next()
			return
		}
		if !allowed {
			response.TooManyRequests(c)
			c.Abort()
			return
		}
		c.Next()
	}
}

func (m *RateLimitMiddleware) CommentLimit(userID uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed, _, err := m.limiter.Allow(c.Request.Context(), "comment_write", string(rune(userID)), 10, time.Minute)
		if err != nil {
			c.Next()
			return
		}
		if !allowed {
			response.TooManyRequests(c)
			c.Abort()
			return
		}
		c.Next()
	}
}

func (m *RateLimitMiddleware) SocialLimit(userID uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed, _, err := m.limiter.Allow(c.Request.Context(), "social_write", string(rune(userID)), 20, time.Minute)
		if err != nil {
			c.Next()
			return
		}
		if !allowed {
			response.TooManyRequests(c)
			c.Abort()
			return
		}
		c.Next()
	}
}