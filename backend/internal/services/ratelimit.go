package services

import (
	"sync"
	"time"
)

type rateLimitEntry struct {
	count   int
	started time.Time
}

// FixedWindowLimiter is a concurrency-safe fixed-window rate counter keyed by
// an arbitrary string (client IP, user id, or connection id). Every key may
// make up to `limit` hits per `window`; the window resets on the first hit
// after it expires. It backs the HTTP rate-limit middleware (NFR-24) and the
// per-connection WebSocket message throttle.
type FixedWindowLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	entries map[string]rateLimitEntry
	maxKeys int
}

// NewFixedWindowLimiter returns a limiter allowing `limit` hits per `window`
// per key. maxKeys bounds the number of tracked keys; when that bound is
// reached, expired entries are evicted lazily on the next Allow call for a new
// key. A maxKeys of 0 (or negative) uses a sane default.
func NewFixedWindowLimiter(limit int, window time.Duration, maxKeys int) *FixedWindowLimiter {
	if maxKeys <= 0 {
		maxKeys = 10000
	}
	return &FixedWindowLimiter{
		limit:   limit,
		window:  window,
		entries: make(map[string]rateLimitEntry),
		maxKeys: maxKeys,
	}
}

// Allow records a hit for key and reports whether the key is still within its
// allowance for the current window. It is safe for concurrent use.
func (l *FixedWindowLimiter) Allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, exists := l.entries[key]
	if !exists || now.Sub(entry.started) >= l.window {
		if !exists && len(l.entries) >= l.maxKeys {
			l.pruneExpired(now)
		}
		entry = rateLimitEntry{started: now, count: 1}
	} else {
		entry.count++
	}
	l.entries[key] = entry
	return entry.count <= l.limit
}

// pruneExpired deletes keys whose window has fully elapsed. It assumes the
// caller holds the lock.
func (l *FixedWindowLimiter) pruneExpired(now time.Time) {
	for k, e := range l.entries {
		if now.Sub(e.started) >= l.window {
			delete(l.entries, k)
		}
	}
}
