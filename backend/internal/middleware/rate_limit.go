package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimitEntry struct {
	count   int
	started time.Time
}

// IPRateLimiter limits a public endpoint independently for each client IP.
func IPRateLimiter(limit int, window time.Duration) gin.HandlerFunc {
	var mu sync.Mutex
	entries := map[string]rateLimitEntry{}
	return func(c *gin.Context) {
		host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err != nil {
			host = c.ClientIP()
		}
		now := time.Now()
		mu.Lock()
		entry := entries[host]
		if entry.started.IsZero() || now.Sub(entry.started) >= window {
			entry = rateLimitEntry{started: now}
		}
		entry.count++
		entries[host] = entry
		allowed := entry.count <= limit
		mu.Unlock()
		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests. Please try again later."})
			c.Abort()
			return
		}
		c.Next()
	}
}
