package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/chorus/messenger/internal/observability"
	"github.com/chorus/messenger/pkg/translation"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TranslationQueueJob is a legacy type kept for compatibility.
type TranslationQueueJob struct {
	MessageID   string   `json:"messageId"`
	Text        string   `json:"text"`
	TargetLangs []string `json:"targetLangs"`
}

// TranslationService handles text translation using a pluggable provider.
// It caches results in Redis and delegates the actual translation to the
// configured Provider implementation (OpenAI, Ollama, or Translator Engine).
type TranslationService struct {
	redis        *redis.Client
	provider     translation.Provider
	ctx          context.Context
	chainTimeout time.Duration
	knownWords   KnownWordsResolver
}

// KnownWordsResolver returns the set of words a user has already learned
// (normalized/lowercased). It is wired to the vocabulary service so the
// translation pipeline can skip known words and avoid redundant LLM calls
// (FR-26 Learned-word / word-bank optimization). Returning nil disables the
// filtering.
type KnownWordsResolver func(userID string) (map[string]struct{}, error)

// WordTranslation is a per-word cache hit surfaced to callers so the UI can
// reuse known-word translations instead of re-translating them.
type WordTranslation struct {
	Word        string `json:"word"`
	Translation string `json:"translation,omitempty"`
	Known       bool   `json:"known"`
	Cached      bool   `json:"cached"`
	Skipped     bool   `json:"skipped,omitempty"`
}

// LearnedTranslateResult is the result of a learned-word-aware translation.
type LearnedTranslateResult struct {
	Text          string            `json:"text"`
	Words         []WordTranslation `json:"words"`
	AllWordsKnown bool              `json:"allWordsKnown"`
	SkippedLLM    bool              `json:"skippedLLM"`
}

// detectionTimeout bounds a single language-detection call so a slow provider
// cannot stall the message-send path waiting for it.
const detectionTimeout = 10 * time.Second

// TranslationPromptVersion is the current prompt/cache version for
// translations. Bump it (and its mirror in the translation cache key) to
// invalidate all cached translations after prompt/model changes (FR-30
// prompt-critique loop). It is persisted as translation_jobs.prompt_version so
// evals can be attributed to the exact prompt that produced them.
const TranslationPromptVersion = "v2"

// translationTracer emits FR-30/NFR-25 Phoenix spans from the translation path.
var translationTracer = observability.Tracer("chorus.translation")

// TranslatedResult carries the output plus the lineage that produced it
// (FR-30): which provider/model answered, how long it took, token usage, and
// whether the result came from the Redis cache.
type TranslatedResult struct {
	Text       string
	Provider   string
	Model      string
	Latency    time.Duration
	Tokens     int
	CacheHit   bool
	SourceLang string
	TargetLang string
}

// NewTranslationService creates a new TranslationService with the given provider.
//
// The provider is the translation backend to use (e.g. OpenAI, Ollama, etc.).
// Redis is optional; if nil, caching is disabled.
// chainTimeout is the total timeout for the entire chain across all providers.
// Use 0 to default to 120 seconds.
func NewTranslationService(provider translation.Provider, redis *redis.Client, chainTimeout time.Duration) *TranslationService {
	if chainTimeout <= 0 {
		chainTimeout = 120 * time.Second
	}
	return &TranslationService{
		redis:        redis,
		provider:     provider,
		ctx:          context.Background(),
		chainTimeout: chainTimeout,
	}
}

// SetKnownWordsResolver wires the vocabulary learned-word set into the
// translation pipeline so known words can skip redundant LLM calls (FR-26).
func (s *TranslationService) SetKnownWordsResolver(r KnownWordsResolver) {
	s.knownWords = r
}

// Translate translates text to the target language, auto-detecting the source.
func (s *TranslationService) Translate(text, targetLang string) (string, error) {
	return s.TranslateQuick(text, targetLang, "auto")
}

// DetectLanguage identifies the source language of a piece of text (ISO 639-1
// code) using the configured provider chain. Results are cached in Redis for
// 24 hours so repeated identical text is detected instantly. A short timeout
// keeps detection from stalling callers when the chain is slow.
func (s *TranslationService) DetectLanguage(text string) (string, error) {
	if s.provider == nil {
		return "", errors.New("translation provider not configured")
	}
	detector, ok := s.provider.(translation.LanguageDetector)
	if !ok {
		return "", errors.New("translation provider does not support language detection")
	}

	const cacheVersion = "v1"
	cacheKey := fmt.Sprintf("detected:%s:%s", cacheVersion, text)
	if s.redis != nil {
		cached, err := s.redis.Get(s.ctx, cacheKey).Result()
		if err == nil && cached != "" {
			return cached, nil
		}
	}

	ctx, cancel := context.WithTimeout(s.ctx, detectionTimeout)
	defer cancel()

	spanCtx, span := translationTracer.Start(ctx, "translation.detect_language", trace.WithAttributes(
		attribute.String("text", text),
	))
	code, err := detector.DetectLanguage(spanCtx, text)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "detect failed")
		span.End()
		return "", err
	}
	span.SetAttributes(attribute.String("detected_lang", code))
	span.End()
	if code == "" {
		return "", errors.New("language detection returned an empty code")
	}

	if s.redis != nil {
		s.redis.Set(s.ctx, cacheKey, code, 24*time.Hour)
	}
	return code, nil
}

// TranslateQuick translates text using the configured provider.
// Results are cached in Redis for 24 hours.
func (s *TranslationService) TranslateQuick(text, targetLang, sourceLang string) (string, error) {
	res, err := s.TranslateQuickResult(text, targetLang, sourceLang)
	if err != nil {
		return "", err
	}
	return res.Text, nil
}

// TranslateQuickResult is TranslateQuick plus FR-30 lineage: it returns the
// translated text together with the provider/model that answered, the measured
// latency, token usage, and whether the result came from the cache.
func (s *TranslationService) TranslateQuickResult(text, targetLang, sourceLang string) (*TranslatedResult, error) {
	if s.provider == nil {
		return nil, errors.New("translation provider not configured")
	}

	// Cache version — bump this to invalidate all cached translations after prompt/model changes.
	cacheKey := fmt.Sprintf("translation:%s:%s:%s:%s:%s", TranslationPromptVersion, s.provider.Name(), sourceLang, targetLang, text)
	if s.redis != nil {
		cached, err := s.redis.Get(s.ctx, cacheKey).Result()
		if err == nil && cached != "" {
			_, span := translationTracer.Start(s.ctx, "translation.translate",
				trace.WithAttributes(
					attribute.String("text", text),
					attribute.String("source_lang", sourceLang),
					attribute.String("target_lang", targetLang),
					attribute.String("provider", s.provider.Name()),
					attribute.Bool("cache_hit", true),
				),
			)
			span.End()
			// NFR-18: a Redis cache hit is a nominal-latency service call worth
			// tracking separately from real provider calls (cache_hit="hit").
			observability.ObserveTranslation(s.provider.Name(), true, "ok", 0, 0)
			return &TranslatedResult{
				Text:       cached,
				Provider:   s.provider.Name(),
				Model:      providerModel(s.provider),
				CacheHit:   true,
				SourceLang: sourceLang,
				TargetLang: targetLang,
			}, nil
		}
	}

	req := translation.TranslateRequest{
		Text:       text,
		SourceLang: sourceLang,
		TargetLang: targetLang,
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(s.ctx, s.chainTimeout)
	defer cancel()

	// FR-30/NFR-25: export a span to Phoenix (no-op when tracing is disabled).
	spanCtx, span := translationTracer.Start(ctx, "translation.translate",
		trace.WithAttributes(
			attribute.String("text", text),
			attribute.String("source_lang", sourceLang),
			attribute.String("target_lang", targetLang),
		),
	)
	resp, err := s.provider.Translate(spanCtx, req)
	latency := time.Since(start)
	if err != nil {
		observability.ObserveTranslation(s.provider.Name(), false, "error", latency, 0)
		span.RecordError(err)
		span.SetStatus(codes.Error, "translation failed")
		span.End()
		return nil, fmt.Errorf("translation failed: %w", err)
	}
	observability.ObserveTranslation(providerName(resp.Provider), false, "ok", latency, resp.Usage.Total())
	span.SetAttributes(
		attribute.String("provider", resp.Provider),
		attribute.Int("latency_ms", int(latency.Milliseconds())),
		attribute.Int("tokens", resp.Usage.Total()),
		attribute.Int("tokens_in", resp.Usage.PromptTokens),
		attribute.Int("tokens_out", resp.Usage.CompletionTokens),
	)
	span.End()

	result := strings.TrimSpace(resp.TranslatedText)
	if result == "" {
		return nil, errors.New("translation returned empty result")
	}

	if s.redis != nil {
		s.redis.Set(s.ctx, cacheKey, result, 24*time.Hour)
	}

	return &TranslatedResult{
		Text:       result,
		Provider:   resp.Provider,
		Model:      providerModel(s.provider),
		Latency:    latency,
		Tokens:     resp.Usage.Total(),
		CacheHit:   false,
		SourceLang: req.SourceLang,
		TargetLang: req.TargetLang,
	}, nil
}

// TranslateMultiple translates text into multiple target languages.
// Errors for individual languages are silently skipped.
func (s *TranslationService) TranslateMultiple(text string, targetLangs []string) (map[string]string, error) {
	translations := make(map[string]string)
	for _, lang := range targetLangs {
		trans, err := s.Translate(text, lang)
		if err != nil {
			continue
		}
		translations[lang] = trans
	}
	return translations, nil
}

// ---------------------------------------------------------------------------
// FR-26 Learned-word / word-bank optimization
// ---------------------------------------------------------------------------

const wordCacheVersion = "v1"

// wordCacheKey returns the Redis key for a per-word translation. Bumping
// wordCacheVersion invalidates all cached word translations.
func wordCacheKey(word, targetLang string) string {
	return fmt.Sprintf("translation:word:%s:%s:%s", wordCacheVersion, word, targetLang)
}

// splitWords tokenizes text into words, treating runs of non-letter/non-digit
// characters (excluding internal apostrophes and hyphens) as separators.
func splitWords(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '\'' && r != '-'
	})
}

// FilterKnownWords splits text into the words that userID already knows verses
// the unknown ones by consulting the wired known-words resolver (FR-26). When
// no resolver is configured (or userID is empty) every word is treated as
// unknown. Results are de-duplicated while preserving first-appearance order.
func (s *TranslationService) FilterKnownWords(text, userID string) (known, unknown []string, err error) {
	if s.knownWords == nil || strings.TrimSpace(userID) == "" {
		return nil, dedupeWords(splitWords(text)), nil
	}
	knownSet, err := s.knownWords(userID)
	if err != nil {
		return nil, nil, err
	}
	seenKnown := make(map[string]struct{})
	seenUnknown := make(map[string]struct{})
	for _, w := range splitWords(text) {
		norm := NormalizeLearningTerm(w, "")
		if _, ok := knownSet[norm]; ok {
			if _, dup := seenKnown[norm]; !dup {
				seenKnown[norm] = struct{}{}
				known = append(known, w)
			}
		} else {
			if _, dup := seenUnknown[norm]; !dup {
				seenUnknown[norm] = struct{}{}
				unknown = append(unknown, w)
			}
		}
	}
	return known, unknown, nil
}

// dedupeWords removes duplicate tokens while preserving order.
func dedupeWords(words []string) []string {
	seen := make(map[string]struct{}, len(words))
	out := make([]string, 0, len(words))
	for _, w := range words {
		norm := NormalizeLearningTerm(w, "")
		if _, dup := seen[norm]; dup {
			continue
		}
		seen[norm] = struct{}{}
		out = append(out, w)
	}
	return out
}

// TranslateWord translates a single word and caches the result per
// word+target language (translation:word:{word}:{lang}) so repeated words
// across messages/sentences avoid redundant LLM calls (FR-26).
func (s *TranslationService) TranslateWord(word, targetLang, sourceLang string) (string, error) {
	if s.provider == nil {
		return "", errors.New("translation provider not configured")
	}
	word = strings.TrimSpace(word)
	if word == "" {
		return "", errors.New("word is empty")
	}
	key := wordCacheKey(word, targetLang)
	if s.redis != nil {
		if cached, err := s.redis.Get(s.ctx, key).Result(); err == nil && cached != "" {
			return cached, nil
		}
	}
	ctx, cancel := context.WithTimeout(s.ctx, s.chainTimeout)
	defer cancel()
	resp, err := s.provider.Translate(ctx, translation.TranslateRequest{
		Text:       word,
		SourceLang: sourceLang,
		TargetLang: targetLang,
	})
	if err != nil {
		return "", fmt.Errorf("word translation failed: %w", err)
	}
	result := strings.TrimSpace(resp.TranslatedText)
	if result == "" {
		return "", errors.New("word translation returned empty result")
	}
	if s.redis != nil {
		s.redis.Set(s.ctx, key, result, 24*time.Hour)
	}
	return result, nil
}

// cachedWord reads a per-word translation from Redis without translating.
// It returns "" when the word translation is not cached (or Redis is nil).
func (s *TranslationService) cachedWord(word, targetLang string) string {
	if s.redis == nil {
		return ""
	}
	v, err := s.redis.Get(s.ctx, wordCacheKey(word, targetLang)).Result()
	if err != nil || v == "" {
		return ""
	}
	return v
}

// TranslateWithLearnedFilter is the FR-26 word-bank-aware translation path.
//   - If the entire text consists of words userID already knows, it skips the
//     LLM entirely (returning the original text) so known words like "hola"/"hi"
//     are never re-translated.
//   - For a mixed sentence it keeps the full-sentence translation (which needs
//     context) but also surfaces per-word cache hits for known words so callers
//     can reuse those translations rather than re-translating them individually.
func (s *TranslationService) TranslateWithLearnedFilter(text, targetLang, sourceLang, userID string) (LearnedTranslateResult, error) {
	var res LearnedTranslateResult
	known, unknown, err := s.FilterKnownWords(text, userID)
	if err != nil {
		return res, err
	}
	// Entire text is known -> no LLM call needed.
	if len(known) > 0 && len(unknown) == 0 {
		res.Text = text
		res.AllWordsKnown = true
		res.SkippedLLM = true
		seen := make(map[string]struct{}, len(known))
		for _, w := range known {
			norm := NormalizeLearningTerm(w, "")
			if _, dup := seen[norm]; dup {
				continue
			}
			seen[norm] = struct{}{}
			res.Words = append(res.Words, WordTranslation{Word: w, Known: true, Skipped: true})
		}
		return res, nil
	}
	// Mixed sentence: keep the full-sentence translation for context.
	if len(unknown) > 0 {
		translated, terr := s.TranslateQuick(text, targetLang, sourceLang)
		if terr != nil {
			return res, terr
		}
		res.Text = translated
	}
	// Surface per-word cache hits for known words (never trigger new LLM
	// translations for known words here).
	seen := make(map[string]struct{}, len(known))
	for _, w := range known {
		norm := NormalizeLearningTerm(w, "")
		if _, dup := seen[norm]; dup {
			continue
		}
		seen[norm] = struct{}{}
		wt := WordTranslation{Word: w, Known: true, Skipped: true}
		if tr := s.cachedWord(w, targetLang); tr != "" {
			wt.Translation = tr
			wt.Cached = true
		}
		res.Words = append(res.Words, wt)
	}
	return res, nil
}

// ProviderHealth reports the readiness of each provider in the configured
// chain. Used by the admin console to diagnose translation config issues
// (e.g. a cloud provider with a missing API key).
func (s *TranslationService) ProviderHealth() []translation.ProviderHealth {
	if s.provider == nil {
		return []translation.ProviderHealth{{Name: "none", Ready: false, Reason: "no translation provider configured"}}
	}
	if chain, ok := s.provider.(*translation.ChainProvider); ok {
		return chain.Healthy()
	}
	return []translation.ProviderHealth{{Name: s.provider.Name(), Ready: true}}
}

// EnqueueOllamaTranslation is a legacy no-op kept for compatibility.
func (s *TranslationService) EnqueueOllamaTranslation(messageID, text string, targetLangs []string) error {
	return nil
}

// ProcessOllamaQueue is a legacy no-op kept for compatibility.
func (s *TranslationService) ProcessOllamaQueue(onComplete func(messageID string, translations map[string]string)) {
	return
}

// providerModel extracts the configured model from a provider, when the
// provider exposes it. Used for FR-30 lineage (translation_jobs.model).
func providerModel(p translation.Provider) string {
	if m, ok := p.(interface{ ModelName() string }); ok {
		return m.ModelName()
	}
	return ""
}

// providerName returns the provider name reported by a result, falling back to
// the service's configured provider name so metrics never carry an empty label.
func providerName(reported string) string {
	if reported != "" {
		return reported
	}
	return "unknown"
}

// languageCodeToName is kept here for backward compatibility with grammar.go
// which references it from the services package.
func languageCodeToName(code string) string {
	m := map[string]string{
		"en": "English", "es": "Spanish", "fr": "French", "de": "German",
		"it": "Italian", "pt": "Portuguese", "ja": "Japanese", "ko": "Korean",
		"zh": "Chinese", "ar": "Arabic", "nl": "Dutch", "pl": "Polish",
		"ru": "Russian", "sv": "Swedish", "af": "Afrikaans", "bg": "Bulgarian",
		"bn": "Bengali", "bs": "Bosnian", "ca": "Catalan", "cs": "Czech",
		"cy": "Welsh", "da": "Danish", "el": "Greek", "et": "Estonian",
		"fa": "Persian", "fi": "Finnish", "ga": "Irish", "gl": "Galician",
		"gu": "Gujarati", "ha": "Hausa", "he": "Hebrew", "hi": "Hindi",
		"hr": "Croatian", "hu": "Hungarian", "id": "Indonesian", "ig": "Igbo",
		"is": "Icelandic", "kk": "Kazakh", "km": "Khmer", "kn": "Kannada",
		"ky": "Kyrgyz", "lo": "Lao", "lt": "Lithuanian", "lv": "Latvian",
		"mg": "Malagasy", "mk": "Macedonian", "ml": "Malayalam", "mn": "Mongolian",
		"mr": "Marathi", "ms": "Malay", "mt": "Maltese", "my": "Burmese",
		"ne": "Nepali", "no": "Norwegian", "pa": "Punjabi", "ps": "Pashto",
		"ro": "Romanian", "rw": "Kinyarwanda", "si": "Sinhala", "sk": "Slovak",
		"sl": "Slovenian", "so": "Somali", "sq": "Albanian", "sr": "Serbian",
		"sw": "Swahili", "ta": "Tamil", "te": "Telugu", "tg": "Tajik",
		"th": "Thai", "tk": "Turkmen", "tr": "Turkish", "uk": "Ukrainian",
		"ur": "Urdu", "uz": "Uzbek", "vi": "Vietnamese", "xh": "Xhosa",
		"yo": "Yoruba", "zu": "Zulu",
	}
	if v, ok := m[code]; ok {
		return v
	}
	return code
}

// Ensure json is used (import reference for grammar.go compatibility).
var _ = json.Marshal
