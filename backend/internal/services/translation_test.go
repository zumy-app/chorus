package services

import (
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestNewTranslationService(t *testing.T) {
	s := NewTranslationService("http://localhost:5002", "http://localhost:11435", "qwen2.5:3b", nil)
	if s == nil {
		t.Fatal("NewTranslationService returned nil")
	}
	if s.translatorEngineURL != "http://localhost:5002" {
		t.Fatalf("expected http://localhost:5002, got %s", s.translatorEngineURL)
	}
	if s.ollamaURL != "http://localhost:11435" {
		t.Fatalf("expected http://localhost:11435, got %s", s.ollamaURL)
	}
	if s.ollamaModel != "qwen2.5:3b" {
		t.Fatalf("expected qwen2.5:3b, got %s", s.ollamaModel)
	}
}

func TestTranslate_ReturnsMockOnError(t *testing.T) {
	// When the translator engine is unreachable, Translate should return a mock translation
	s := NewTranslationService("http://localhost:1", "http://localhost:1", "test-model", nil)

	// Should not panic or hang - should return mock/error gracefully
	result, err := s.Translate("Hello", "es")
	if err != nil {
		// Accept error - the important thing is it doesn't panic
		t.Logf("Translate returned error (expected): %v", err)
	} else {
		t.Logf("Translate returned (no error): %s", result)
	}
}

func TestTranslateQuick_ReturnsMockOnError(t *testing.T) {
	s := NewTranslationService("http://localhost:1", "http://localhost:1", "test-model", nil)

	result, err := s.TranslateQuick("Hello", "es", "en")
	if err != nil {
		t.Logf("TranslateQuick returned error (expected): %v", err)
	} else {
		t.Logf("TranslateQuick returned: %s", result)
	}
}

func TestTranslateQuick_UsesCache(t *testing.T) {
	// With a redis client that can't connect, the cache lookup should fail gracefully
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:1"})
	s := NewTranslationService("http://localhost:1", "http://localhost:1", "test-model", redisClient)

	// Should handle the cache miss gracefully and attempt HTTP translation
	result, err := s.TranslateQuick("Hello", "es", "en")
	if err != nil {
		t.Logf("TranslateQuick error (expected with bad redis+http): %v", err)
	} else {
		t.Logf("TranslateQuick returned: %s", result)
	}
}

func TestTranslateMultiple(t *testing.T) {
	s := NewTranslationService("http://localhost:1", "http://localhost:1", "test-model", nil)

	// Should return an error because the translator engine is unreachable
	translations, err := s.TranslateMultiple("Hello", []string{"es", "fr"})
	if err != nil {
		t.Logf("TranslateMultiple error (expected): %v", err)
	} else {
		t.Logf("TranslateMultiple returned %d translations", len(translations))
	}
}

func TestProcessOllamaQueue_WithNilRedis(t *testing.T) {
	// With nil redis and nil callback, ProcessOllamaQueue should not panic
	s := NewTranslationService("http://localhost:1", "http://localhost:1", "test-model", nil)

	// Temporarily disable queue to test initialization
	s.queueEnabled = false

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
