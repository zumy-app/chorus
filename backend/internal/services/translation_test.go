package services

import (
	"context"
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

func TestTranslateQuickResult_Lineage(t *testing.T) {
	provider := &fakeProvider{}
	s := NewTranslationService(provider, nil, 0)

	res, err := s.TranslateQuickResult("Hola", "es", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if res.Text != "ES:Hola" {
		t.Fatalf("expected 'ES:Hola', got %q", res.Text)
	}
	if res.Provider != "fake" {
		t.Fatalf("expected provider 'fake', got %q", res.Provider)
	}
	if res.CacheHit {
		t.Fatal("expected cacheHit false (no redis)")
	}
	if res.TargetLang != "es" || res.SourceLang != "en" {
		t.Fatalf("unexpected langs: %+v", res)
	}
	if provider.calls != 1 {
		t.Fatalf("expected 1 provider call, got %d", provider.calls)
	}
}

func TestTranslateQuickResult_NoProviderReturnsError(t *testing.T) {
	s := &TranslationService{redis: nil, provider: nil}
	if _, err := s.TranslateQuickResult("Hola", "es", "en"); err == nil {
		t.Fatal("expected error with nil provider")
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

// fakeProvider is an in-memory translation.Provider that records how many
// Translate calls it receives so tests can assert LLM-call-skipping behavior.
type fakeProvider struct {
	calls int
}

func (p *fakeProvider) Translate(_ context.Context, req translation.TranslateRequest) (translation.TranslateResponse, error) {
	p.calls++
	return translation.TranslateResponse{TranslatedText: "ES:" + req.Text, Provider: "fake"}, nil
}

func (p *fakeProvider) Name() string { return "fake" }

func knownResolver(words ...string) KnownWordsResolver {
	return func(userID string) (map[string]struct{}, error) {
		set := make(map[string]struct{}, len(words))
		for _, w := range words {
			set[NormalizeLearningTerm(w, "")] = struct{}{}
		}
		return set, nil
	}
}

func TestFilterKnownWords(t *testing.T) {
	s := NewTranslationService(&fakeProvider{}, nil, 0)
	s.SetKnownWordsResolver(knownResolver("hola", "hi"))

	known, unknown, err := s.FilterKnownWords("Hola, mi amigo. Hi!", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(known) != 2 || known[0] != "Hola" || known[1] != "Hi" {
		t.Fatalf("expected known ['Hola','Hi'], got %v", known)
	}
	if len(unknown) != 2 || unknown[0] != "mi" || unknown[1] != "amigo" {
		t.Fatalf("expected unknown ['mi','amigo'], got %v", unknown)
	}
}

func TestFilterKnownWords_DeDuplicatesByNormalizedForm(t *testing.T) {
	s := NewTranslationService(&fakeProvider{}, nil, 0)
	s.SetKnownWordsResolver(knownResolver("amigo"))

	known, _, err := s.FilterKnownWords("Amigo AMIGO amigo", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(known) != 1 {
		t.Fatalf("expected 1 deduped known word, got %v", known)
	}
}

func TestFilterKnownWords_NoResolverTreatsAllUnknown(t *testing.T) {
	s := NewTranslationService(&fakeProvider{}, nil, 0)

	known, unknown, err := s.FilterKnownWords("hola amigo", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(known) != 0 {
		t.Fatalf("expected no known words with nil resolver, got %v", known)
	}
	if len(unknown) != 2 {
		t.Fatalf("expected 2 unknown words, got %v", unknown)
	}
}

func TestTranslateWithLearnedFilter_SkipsLLMForAllKnown(t *testing.T) {
	provider := &fakeProvider{}
	s := NewTranslationService(provider, nil, 0)
	s.SetKnownWordsResolver(knownResolver("hola", "mundo"))

	res, err := s.TranslateWithLearnedFilter("Hola mundo", "es", "en", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.AllWordsKnown {
		t.Fatal("expected AllWordsKnown true")
	}
	if !res.SkippedLLM {
		t.Fatal("expected SkippedLLM true")
	}
	if res.Text != "Hola mundo" {
		t.Fatalf("expected original text, got %q", res.Text)
	}
	if provider.calls != 0 {
		t.Fatalf("expected 0 provider calls, got %d", provider.calls)
	}
}

func TestTranslateWithLearnedFilter_MixedSentenceKeepsTranslation(t *testing.T) {
	provider := &fakeProvider{}
	s := NewTranslationService(provider, nil, 0)
	s.SetKnownWordsResolver(knownResolver("hola"))

	res, err := s.TranslateWithLearnedFilter("Hola amigo", "es", "en", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Text != "ES:Hola amigo" {
		t.Fatalf("expected sentence translated, got %q", res.Text)
	}
	if provider.calls != 1 {
		t.Fatalf("expected 1 provider call, got %d", provider.calls)
	}
	// Known word surfaced but not re-translated individually.
	if len(res.Words) != 1 {
		t.Fatalf("expected 1 surfaced word, got %v", res.Words)
	}
	if res.Words[0].Word != "Hola" || !res.Words[0].Known || !res.Words[0].Skipped {
		t.Fatalf("unexpected surfaced word: %+v", res.Words[0])
	}
}

func TestTranslateWord(t *testing.T) {
	provider := &fakeProvider{}
	s := NewTranslationService(provider, nil, 0)

	got, err := s.TranslateWord("hola", "es", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "ES:hola" {
		t.Fatalf("expected 'ES:hola', got %q", got)
	}
	if provider.calls != 1 {
		t.Fatalf("expected 1 provider call, got %d", provider.calls)
	}
}

func TestTranslateWord_EmptyWordReturnsError(t *testing.T) {
	s := NewTranslationService(&fakeProvider{}, nil, 0)
	if _, err := s.TranslateWord("  ", "es", "en"); err == nil {
		t.Fatal("expected error for empty word")
	}
}

func TestTranslateWord_NoProviderReturnsError(t *testing.T) {
	s := &TranslationService{redis: nil, provider: nil}
	if _, err := s.TranslateWord("hola", "es", "en"); err == nil {
		t.Fatal("expected error with nil provider")
	}
}

func TestTranslateWithLearnedFilter_NoResolverTranslatesFully(t *testing.T) {
	provider := &fakeProvider{}
	s := NewTranslationService(provider, nil, 0)

	res, err := s.TranslateWithLearnedFilter("Hola amigo", "es", "en", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.SkippedLLM {
		t.Fatal("expected LLM call since no resolver")
	}
	if res.Text != "ES:Hola amigo" {
		t.Fatalf("expected sentence translated, got %q", res.Text)
	}
	if provider.calls != 1 {
		t.Fatalf("expected 1 provider call, got %d", provider.calls)
	}
}
