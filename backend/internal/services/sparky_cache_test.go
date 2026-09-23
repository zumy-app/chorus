package services

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newSparkyCacheTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(func() { mr.Close() })
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func TestSparkyCache_RoundTrip(t *testing.T) {
	load, store := NewSparkyRedisCache(newSparkyCacheTestRedis(t))

	if _, _, found := load("ctx", "es", "en", "q?"); found {
		t.Fatalf("expected miss on empty cache")
	}

	store("ctx", "es", "en", "q?", "answer!", "openrouter")

	content, provider, found := load("ctx", "es", "en", "q?")
	if !found {
		t.Fatalf("expected hit after store")
	}
	if content != "answer!" || provider != "openrouter" {
		t.Fatalf("unexpected cached value: %q %q", content, provider)
	}
}

func TestSparkyCache_KeysDifferByContextAndQuery(t *testing.T) {
	load, store := NewSparkyRedisCache(newSparkyCacheTestRedis(t))

	// Same question, different conversation context must not collide.
	store("ctx-A", "es", "en", "analyze?", "answer A", "ai")
	if _, _, found := load("ctx-B", "es", "en", "analyze?"); found {
		t.Fatalf("context collision: ctx-B hit ctx-A entry")
	}

	// Same context, different question must not collide.
	if _, _, found := load("ctx-A", "es", "en", "other?"); found {
		t.Fatalf("query collision: other? hit analyze? entry")
	}
}

func TestSparkyCache_EmptyContentNotStored(t *testing.T) {
	load, store := NewSparkyRedisCache(newSparkyCacheTestRedis(t))

	store("ctx", "es", "en", "q?", "", "ai")
	if _, _, found := load("ctx", "es", "en", "q?"); found {
		t.Fatalf("empty content must not be cached")
	}
}

func TestSparkyCache_NilClientYieldsNilHooks(t *testing.T) {
	load, store := NewSparkyRedisCache(nil)
	if load != nil || store != nil {
		t.Fatalf("expected nil hooks for nil client")
	}
}

func TestSparkyCacheKey_StableAndVersioned(t *testing.T) {
	a := SparkyCacheKey("ctx", "es", "en", "q?")
	b := SparkyCacheKey("ctx", "es", "en", "q?")
	if a != b {
		t.Fatalf("key not stable: %q vs %q", a, b)
	}
	if len(a) < len("sparky:"+SparkyPromptVersion+":") || a[:len("sparky:"+SparkyPromptVersion+":")] != "sparky:"+SparkyPromptVersion+":" {
		t.Fatalf("key missing version prefix: %q", a)
	}
}
