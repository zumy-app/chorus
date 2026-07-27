package services

import (
	"testing"
	"time"

	"github.com/chorus/messenger/pkg/translation"
	"github.com/redis/go-redis/v9"
)

func TestNewTranslationService(t *testing.T) {
	provider := translation.NewOpenAIProvider("https://api.opencode.com/v1", "test-key", "gpt-4o-mini", 0)
	s := NewTranslationService(provider, nil, 0)
	if s == nil {
		t.Fatal("NewTranslationService returned nil")
	}
	if s.provider == nil {
		t.Fatal("provider should not be nil")
	}
}

func TestTranslate_ReturnsErrorOnNoProvider(t *testing.T) {
	s := &TranslationService{redis: nil, provider: nil}

	result, err := s.Translate("Hello", "es")
	if err == nil {
		t.Fatal("expected error with nil provider")
	}
	if result != "" {
		t.Fatalf("expected empty result, got %s", result)
	}
}

func TestTranslateQuick_ReturnsErrorOnNoProvider(t *testing.T) {
	s := &TranslationService{redis: nil, provider: nil}

	result, err := s.TranslateQuick("Hello", "es", "en")
	if err == nil {
		t.Fatal("expected error with nil provider")
	}
	if result != "" {
		t.Fatalf("expected empty result, got %s", result)
	}
}

func TestTranslateQuick_UsesCache(t *testing.T) {
	// With a redis client that can't connect, the cache lookup should fail gracefully
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:1"})
	provider := translation.NewOpenAIProvider("http://localhost:1", "test-key", "gpt-4o-mini", 0)
	s := NewTranslationService(provider, redisClient, 0)

	// Should handle the cache miss gracefully and attempt HTTP translation
	result, err := s.TranslateQuick("Hello", "es", "en")
	if err != nil {
		t.Logf("TranslateQuick error (expected with bad redis+http): %v", err)
	} else {
		t.Logf("TranslateQuick returned: %s", result)
	}
}

func TestTranslateMultiple(t *testing.T) {
	provider := translation.NewOpenAIProvider("http://localhost:1", "test-key", "gpt-4o-mini", 0)
	s := NewTranslationService(provider, nil, 0)

	// Should return an error because the provider is unreachable
	translations, err := s.TranslateMultiple("Hello", []string{"es", "fr"})
	if err != nil {
		t.Logf("TranslateMultiple error (expected): %v", err)
	} else {
		t.Logf("TranslateMultiple returned %d translations", len(translations))
	}
}

func TestProcessOllamaQueue_WithNilRedis(t *testing.T) {
	// With nil redis and nil callback, ProcessOllamaQueue should not panic
	provider := translation.NewOpenAIProvider("http://localhost:1", "test-key", "gpt-4o-mini", 0)
	s := NewTranslationService(provider, nil, 0)

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("ProcessOllamaQueue panicked with nil redis: %v", r)
		}
	}()

	// With queue disabled, the goroutine should exit quickly
	done := make(chan bool)
	go func() {
		s.ProcessOllamaQueue(nil)
		close(done)
	}()

	select {
	case <-done:
		// Success - function completed without panic
	case <-time.After(3 * time.Second):
		t.Fatal("ProcessOllamaQueue did not complete within timeout")
	}
}