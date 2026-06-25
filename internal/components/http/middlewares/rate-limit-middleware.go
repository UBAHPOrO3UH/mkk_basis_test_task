package middlewares

import (
	"fmt"
	"mkk_basis/rest_api/internal/common"
	"mkk_basis/rest_api/internal/config"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimitBucket struct {
	count   int
	resetAt time.Time
}

type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*rateLimitBucket
}

var defaultRateLimiter = &rateLimiter{buckets: map[string]*rateLimitBucket{}}

func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := config.CurrentConfig.RateLimit
		if cfg == nil || !cfg.Enabled || cfg.RequestsPerMinute <= 0 {
			c.Next()
			return
		}

		allowed, remaining, resetAt := defaultRateLimiter.allow(rateLimitKey(c), cfg.RequestsPerMinute)
		c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.RequestsPerMinute))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))

		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, common.ErrorResponse(fmt.Errorf("rate limit exceeded: %d requests per minute", cfg.RequestsPerMinute)))
			return
		}

		c.Next()
	}
}

func (l *rateLimiter) allow(key string, limit int) (bool, int, time.Time) {
	now := time.Now()
	windowEnd := now.Truncate(time.Minute).Add(time.Minute)

	l.mu.Lock()
	defer l.mu.Unlock()

	bucket := l.buckets[key]
	if bucket == nil || !now.Before(bucket.resetAt) {
		bucket = &rateLimitBucket{resetAt: windowEnd}
		l.buckets[key] = bucket
	}

	l.cleanup(now)
	if bucket.count >= limit {
		return false, 0, bucket.resetAt
	}

	bucket.count++
	return true, limit - bucket.count, bucket.resetAt
}

func (l *rateLimiter) cleanup(now time.Time) {
	for key, bucket := range l.buckets {
		if now.After(bucket.resetAt.Add(time.Minute)) {
			delete(l.buckets, key)
		}
	}
}

func rateLimitKey(c *gin.Context) string {
	claims, ok := ClaimsFromContext(c.Request.Context())
	if ok {
		if userID, err := claims.UserID(); err == nil && userID != 0 {
			return fmt.Sprintf("user:%d", userID)
		}
	}

	return "ip:" + c.ClientIP()
}
