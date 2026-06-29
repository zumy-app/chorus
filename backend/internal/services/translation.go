package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/translate"
	"github.com/redis/go-redis/v9"
	"golang.org/x/text/language"
	"google.golang.org/api/option"
)

type TranslationService struct {
	client     *translate.Client
	redis      *redis.Client
	apiKey     string
	ctx        context.Context
}

func NewTranslationService(apiKey string, redis *redis.Client) *TranslationService {
	ctx := context.Background()
	var client *translate.Client
	
	if apiKey != "" {
		var err error
		client, err = translate.NewClient(ctx, option.WithAPIKey(apiKey))
		if err != nil {
			fmt.Printf("Failed to create translation client: %v\n", err)
			client = nil
		}
	}

	return &TranslationService{
		client: client,
		redis:  redis,
		apiKey: apiKey,
		ctx:    ctx,
	}
}

func (s *TranslationService) Translate(text, targetLang string) (string, error) {
	if s.client == nil {
		// Return mock translation if no API key is configured
		return s.mockTranslate(text, targetLang), nil
	}

	// Check cache first
	cacheKey := fmt.Sprintf("translation:%s:%s", targetLang, text)
	if s.redis != nil {
		cached, err := s.redis.Get(s.ctx, cacheKey).Result()
		if err == nil && cached != "" {
			return cached, nil
		}
	}

	// Translate using Google Translate API
	lang, err := language.Parse(targetLang)
	if err != nil {
		return "", err
	}

	translations, err := s.client.Translate(s.ctx, []string{text}, lang, nil)
	if err != nil {
		return "", err
	}

	if len(translations) == 0 {
		return "", errors.New("no translation returned")
	}

	result := translations[0].Text

	// Cache the result
	if s.redis != nil {
		s.redis.Set(s.ctx, cacheKey, result, 24*time.Hour)
	}

	return result, nil
}

func (s *TranslationService) TranslateMultiple(text string, targetLangs []string) (map[string]string, error) {
	translations := make(map[string]string)

	for _, lang := range targetLangs {
		translated, err := s.Translate(text, lang)
		if err != nil {
			// Continue with other languages if one fails
			continue
		}
		translations[lang] = translated
	}

	return translations, nil
}

func (s *TranslationService) DetectLanguage(text string) (string, error) {
	if s.client == nil {
		return "en", nil // Default to English
	}

	detections, err := s.client.DetectLanguage(s.ctx, []string{text})
	if err != nil {
		return "", err
	}

	if len(detections) == 0 || len(detections[0]) == 0 {
		return "", errors.New("no language detected")
	}

	return detections[0][0].Language.String(), nil
}

// Mock translation for development without API key
func (s *TranslationService) mockTranslate(text, targetLang string) string {
	// Comprehensive bidirectional mock dictionary
	mockTranslations := map[string]map[string]string{
		"es": {
			"Hello": "Hola",
			"Hi": "Hola",
			"How are you?": "¿Cómo estás?",
			"Good morning": "Buenos días",
			"Good day": "Buenos días",
			"Good afternoon": "Buenas tardes",
			"Good evening": "Buenas noches",
			"Good night": "Buenas noches",
			"Thank you": "Gracias",
			"Thanks": "Gracias",
			"Please": "Por favor",
			"Sorry": "Lo siento",
			"Excuse me": "Disculpe",
			"Yes": "Sí",
			"No": "No",
			"Friend": "Amigo",
			"How are you doing?": "¿Cómo te va?",
			"See you later": "Hasta luego",
			"Goodbye": "Adiós",
			"Welcome": "Bienvenido",
			"What's up?": "¿Qué tal?",
		},
		"en": {
			"Hola": "Hello",
			"¿Cómo estás?": "How are you?",
			"Buenos días": "Good morning",
			"Buenas tardes": "Good afternoon",
			"Buenas noches": "Good evening",
			"Gracias": "Thank you",
			"Por favor": "Please",
			"Lo siento": "Sorry",
			"Disculpe": "Excuse me",
			"Sí": "Yes",
			"No": "No",
			"Amigo": "Friend",
			"Amiga": "Friend",
			"¿Cómo te va?": "How are you doing?",
			"Hasta luego": "See you later",
			"Adiós": "Goodbye",
			"Bienvenido": "Welcome",
			"¿Qué tal?": "What's up?",
			"Buenas": "Good day",
			"Bueno": "Good",
			"Mucho gusto": "Nice to meet you",
			"De nada": "You're welcome",
		},
		"fr": {
			"Hello": "Bonjour",
			"How are you?": "Comment allez-vous?",
			"Good morning": "Bon matin",
			"Thank you": "Merci",
		},
		"de": {
			"Hello": "Hallo",
			"How are you?": "Wie geht es dir?",
			"Good morning": "Guten Morgen",
			"Thank you": "Danke",
		},
	}

	if langMap, ok := mockTranslations[targetLang]; ok {
		if translation, ok := langMap[text]; ok {
			return translation
		}
		// Also try case-insensitive match
		lowerText := strings.ToLower(text)
		for key, val := range langMap {
			if strings.ToLower(key) == lowerText {
				return val
			}
		}
	}

	// If no exact match, return a friendly mock
	return fmt.Sprintf("[%s] %s", targetLang, text)
}

// Batch translation for efficiency
type TranslationJob struct {
	MessageID   string
	Text        string
	TargetLangs []string
}

func (s *TranslationService) BatchTranslate(jobs []TranslationJob) (map[string]map[string]string, error) {
	results := make(map[string]map[string]string)

	for _, job := range jobs {
		translations, err := s.TranslateMultiple(job.Text, job.TargetLangs)
		if err != nil {
			continue
		}
		results[job.MessageID] = translations
	}

	return results, nil
}

// Cache helpers
func (s *TranslationService) CacheTranslation(text, lang, translation string) error {
	if s.redis == nil {
		return nil
	}

	cacheKey := fmt.Sprintf("translation:%s:%s", lang, text)
	return s.redis.Set(s.ctx, cacheKey, translation, 24*time.Hour).Err()
}

func (s *TranslationService) GetCachedTranslation(text, lang string) (string, error) {
	if s.redis == nil {
		return "", errors.New("redis not available")
	}

	cacheKey := fmt.Sprintf("translation:%s:%s", lang, text)
	return s.redis.Get(s.ctx, cacheKey).Result()
}

// Prefetch translations for common phrases
func (s *TranslationService) PrefetchCommonPhrases() {
	commonPhrases := []string{
		"Hello", "Hi", "Hey", "Good morning", "Good evening",
		"How are you?", "Thank you", "Thanks", "You're welcome",
		"Goodbye", "See you", "Please", "Sorry", "Excuse me",
	}

	targetLangs := []string{"es", "fr", "de", "it", "pt", "ja", "ko", "zh"}

	for _, phrase := range commonPhrases {
		for _, lang := range targetLangs {
			go s.Translate(phrase, lang)
		}
	}
}

// Store translation metadata in cache
type TranslationMetadata struct {
	OriginalText     string    `json:"originalText"`
	TargetLanguage   string    `json:"targetLanguage"`
	TranslatedText   string    `json:"translatedText"`
	DetectedLanguage string    `json:"detectedLanguage"`
	Timestamp        time.Time `json:"timestamp"`
}

func (s *TranslationService) StoreMetadata(meta TranslationMetadata) error {
	if s.redis == nil {
		return nil
	}

	key := fmt.Sprintf("translation_meta:%s:%s", meta.TargetLanguage, meta.OriginalText)
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	return s.redis.Set(s.ctx, key, data, 7*24*time.Hour).Err()
}
