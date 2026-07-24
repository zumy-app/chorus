package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chorus/messenger/pkg/translation"
	"github.com/redis/go-redis/v9"
)

// OllamaGenerateRequest and OllamaGenerateResponse are kept here because
// GrammarService (in grammar.go) references them directly for its own Ollama calls.
// These are NOT used by the new TranslationService which uses the provider abstraction.
type OllamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaGenerateResponse struct {
	Model    string `json:"model"`
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

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
	redis    *redis.Client
	provider translation.Provider
	ctx      context.Context
}

// NewTranslationService creates a new TranslationService with the given provider.
//
// The provider is the translation backend to use (e.g. OpenAI, Ollama, etc.).
// Redis is optional; if nil, caching is disabled.
func NewTranslationService(provider translation.Provider, redis *redis.Client) *TranslationService {
	return &TranslationService{
		redis:    redis,
		provider: provider,
		ctx:      context.Background(),
	}
}

// Translate translates text to the target language, auto-detecting the source.
func (s *TranslationService) Translate(text, targetLang string) (string, error) {
	return s.TranslateQuick(text, targetLang, "auto")
}

// TranslateQuick translates text using the configured provider.
// Results are cached in Redis for 24 hours.
func (s *TranslationService) TranslateQuick(text, targetLang, sourceLang string) (string, error) {
	if s.provider == nil {
		return "", errors.New("translation provider not configured")
	}

	cacheKey := fmt.Sprintf("translation:%s:%s:%s:%s", s.provider.Name(), sourceLang, targetLang, text)
	if s.redis != nil {
		cached, err := s.redis.Get(s.ctx, cacheKey).Result()
		if err == nil && cached != "" {
			return cached, nil
		}
	}

	req := translation.TranslateRequest{
		Text:       text,
		SourceLang: sourceLang,
		TargetLang: targetLang,
	}

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	resp, err := s.provider.Translate(ctx, req)
	if err != nil {
		return "", fmt.Errorf("translation failed: %w", err)
	}

	result := strings.TrimSpace(resp.TranslatedText)
	if result == "" {
		return "", errors.New("translation returned empty result")
	}

	if s.redis != nil {
		s.redis.Set(s.ctx, cacheKey, result, 24*time.Hour)
	}

	return result, nil
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

// EnqueueOllamaTranslation is a legacy no-op kept for compatibility.
func (s *TranslationService) EnqueueOllamaTranslation(messageID, text string, targetLangs []string) error {
	return nil
}

// ProcessOllamaQueue is a legacy no-op kept for compatibility.
func (s *TranslationService) ProcessOllamaQueue(onComplete func(messageID string, translations map[string]string)) {
	return
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