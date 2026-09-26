package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// SparkyCacheTTL bounds how long a cached answer is reused. The cache key
// embeds SparkyPromptVersion, so prompt bumps invalidate automatically.
const SparkyCacheTTL = 24 * time.Hour

// sparkyCacheEntry is the Redis value for a cached Sparky answer.
type sparkyCacheEntry struct {
	Content  string `json:"content"`
	Provider string `json:"provider"`
}

// SparkyCacheKey derives the Redis key for a (context, query, languages)
// tuple. Context is part of the key, so "analyze the last message" asked in
// two different chats never collides.
func SparkyCacheKey(contextText, language, nativeLanguage, query string) string {
	sum := sha256.Sum256([]byte(SparkyPromptVersion + "\x00" + language + "\x00" + nativeLanguage + "\x00" + contextText + "\x00" + query))
	return "sparky:" + SparkyPromptVersion + ":" + hex.EncodeToString(sum[:])
}

// NewSparkyRedisCache returns load/store closures over Redis matching the
// SparkyQueueService cache hook signatures. Nil client yields nil closures
// (queue treats nil as "no cache").
func NewSparkyRedisCache(rdb *redis.Client) (
	load func(contextText, language, nativeLanguage, query string) (string, string, bool),
	store func(contextText, language, nativeLanguage, query, content, provider string),
) {
	if rdb == nil {
		return nil, nil
	}
	load = func(contextText, language, nativeLanguage, query string) (string, string, bool) {
		raw, err := rdb.Get(context.Background(), SparkyCacheKey(contextText, language, nativeLanguage, query)).Result()
		if err != nil || raw == "" {
			return "", "", false
		}
		var e sparkyCacheEntry
		if err := json.Unmarshal([]byte(raw), &e); err != nil || e.Content == "" {
			return "", "", false
		}
		return e.Content, e.Provider, true
	}
	store = func(contextText, language, nativeLanguage, query, content, provider string) {
		if content == "" {
			return
		}
		raw, err := json.Marshal(sparkyCacheEntry{Content: content, Provider: provider})
		if err != nil {
			return
		}
		_ = rdb.Set(context.Background(), SparkyCacheKey(contextText, language, nativeLanguage, query), raw, SparkyCacheTTL).Err()
	}
	return load, store
}
