package middleware

import (
	"net"
	"time"

	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// KeyProvider extracts a stable rate-limit key from a request (client IP or
// authenticated user id). It is invoked once per request inside the limiter.
type KeyProvider func(c *gin.Context) string

// IPKey returns the client IP (already split from the port) as the limit key.
func IPKey(c *gin.Context) string {
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.ClientIP()
	}
	return host
}

// UserKey returns the authenticated user id, falling back to the client IP so
// the limiter never silently lets traffic through when auth is unexpectedly
// missing.
func UserKey(c *gin.Context) string {
	if uid := c.GetString("userID"); uid != "" {
		return uid
	}
	return IPKey(c)
}

// RateLimiter limits requests per key returned by `key`. maxKeys bounds the
// number of distinct keys tracked concurrently (0 uses a sane default). It is
// the base for the IP- and user-scoped limiters.
func RateLimiter(limit int, window time.Duration, key KeyProvider, maxKeys int) gin.HandlerFunc {
	limiter := services.NewFixedWindowLimiter(limit, window, maxKeys)
	return func(c *gin.Context) {
		if !limiter.Allow(key(c)) {
			WriteError(c, ErrRateLimit("Too many requests. Please try again later."))
			return
		}
		c.Next()
	}
}

func RateLimiterRedis(redisClient *redis.Client, limit int, window time.Duration, key KeyProvider, prefix string) gin.HandlerFunc {
	var limiter services.RateLimiter
	if redisClient != nil {
		limiter = services.NewRedisRateLimiter(redisClient, limit, window, prefix)
	} else {
		limiter = services.NewFixedWindowLimiter(limit, window, 0)
	}
	return func(c *gin.Context) {
		if !limiter.Allow(key(c)) {
			WriteError(c, ErrRateLimit("Too many requests. Please try again later."))
			return
		}
		c.Next()
	}
}

// IPRateLimiter limits a public endpoint independently for each client IP.
func IPRateLimiter(limit int, window time.Duration) gin.HandlerFunc {
	return RateLimiter(limit, window, IPKey, 0)
}

// UserRateLimiter limits an authenticated endpoint per user id (falling back to
// the client IP). Use it on cost-heavy operations such as translation to resist
// abuse and cost exhaustion.
func UserRateLimiter(limit int, window time.Duration) gin.HandlerFunc {
	return RateLimiter(limit, window, UserKey, 0)
}
