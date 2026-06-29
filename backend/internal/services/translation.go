package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/translate"
	"github.com/redis/go-redis/v9"
	"golang.org/x/text/language"
	"google.golang.org/api/option"
)

type OllamaGenerateRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	Stream  bool   `json:"stream"`
}

type OllamaGenerateResponse struct {
	Model     string `json:"model"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
}

type TranslationService struct {
	client      *translate.Client
	redis       *redis.Client
	apiKey      string
	ollamaURL   string
	ollamaModel string
	httpClient  *http.Client
	ctx         context.Context
}

func NewTranslationService(apiKey, ollamaURL, ollamaModel string, redis *redis.Client) *TranslationService {
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
		client:      client,
		redis:       redis,
		apiKey:      apiKey,
		ollamaURL:   ollamaURL,
		ollamaModel: ollamaModel,
		httpClient:  &http.Client{Timeout: 60 * time.Second},
		ctx:         ctx,
	}
}

func (s *TranslationService) Translate(text, targetLang string) (string, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("translation:%s:%s", targetLang, text)
	if s.redis != nil {
		cached, err := s.redis.Get(s.ctx, cacheKey).Result()
		if err == nil && cached != "" {
			return cached, nil
		}
	}

	// Try Ollama first if URL is configured
	if s.ollamaURL != "" {
		result, err := s.translateWithOllama(text, targetLang)
		if err == nil && result != "" {
			// Cache the result
			if s.redis != nil {
				s.redis.Set(s.ctx, cacheKey, result, 24*time.Hour)
			}
			return result, nil
		}
		fmt.Printf("Ollama translation failed, falling back: %v\n", err)
	}

	// Fall back to Google Translate if configured
	if s.client != nil {
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

	// Final fallback: mock translation
	result := s.mockTranslate(text, targetLang)
	if s.redis != nil {
		s.redis.Set(s.ctx, cacheKey, result, 24*time.Hour)
	}
	return result, nil
}

// translateWithOllama calls the local Ollama instance to translate text.
func (s *TranslationService) translateWithOllama(text, targetLang string) (string, error) {
	langName := languageCodeToName(targetLang)
	if langName == "" {
		langName = targetLang
	}

	prompt := fmt.Sprintf(`Translate the following text to %s. Only return the translated text, nothing else.

Text: %s
Translation:`, langName, text)

	reqBody := OllamaGenerateRequest{
		Model:  s.ollamaModel,
		Prompt: prompt,
		Stream: false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(s.ctx, "POST", s.ollamaURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var ollamaResp OllamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode Ollama response: %w", err)
	}

	if ollamaResp.Response == "" {
		return "", errors.New("Ollama returned empty response")
	}

	// Clean up the response — trim whitespace and quotes
	result := strings.TrimSpace(ollamaResp.Response)
	result = strings.Trim(result, "\"'「」『』")

	return result, nil
}

// languageCodeToName maps ISO 639-1 codes to full language names for better Ollama prompt quality.
func languageCodeToName(code string) string {
	names := map[string]string{
		"af": "Afrikaans", "sq": "Albanian", "am": "Amharic", "ar": "Arabic",
		"hy": "Armenian", "az": "Azerbaijani", "eu": "Basque", "be": "Belarusian",
		"bn": "Bengali", "bs": "Bosnian", "bg": "Bulgarian", "ca": "Catalan",
		"ceb": "Cebuano", "ny": "Chichewa", "zh": "Chinese", "zh-CN": "Chinese (Simplified)",
		"zh-TW": "Chinese (Traditional)", "co": "Corsican", "hr": "Croatian", "cs": "Czech",
		"da": "Danish", "nl": "Dutch", "en": "English", "eo": "Esperanto",
		"et": "Estonian", "tl": "Filipino", "fi": "Finnish", "fr": "French",
		"fy": "Frisian", "gl": "Galician", "ka": "Georgian", "de": "German",
		"el": "Greek", "gu": "Gujarati", "ht": "Haitian Creole", "ha": "Hausa",
		"haw": "Hawaiian", "he": "Hebrew", "hi": "Hindi", "hmn": "Hmong",
		"hu": "Hungarian", "is": "Icelandic", "ig": "Igbo", "id": "Indonesian",
		"ga": "Irish", "it": "Italian", "ja": "Japanese", "jw": "Javanese",
		"kn": "Kannada", "kk": "Kazakh", "km": "Khmer", "rw": "Kinyarwanda",
		"ko": "Korean", "ku": "Kurdish", "ky": "Kyrgyz", "lo": "Lao",
		"la": "Latin", "lv": "Latvian", "lt": "Lithuanian", "lb": "Luxembourgish",
		"mk": "Macedonian", "mg": "Malagasy", "ms": "Malay", "ml": "Malayalam",
		"mt": "Maltese", "mi": "Maori", "mr": "Marathi", "mn": "Mongolian",
		"my": "Myanmar (Burmese)", "ne": "Nepali", "no": "Norwegian", "or": "Odia",
		"ps": "Pashto", "fa": "Persian", "pl": "Polish", "pt": "Portuguese",
		"pa": "Punjabi", "ro": "Romanian", "ru": "Russian", "sm": "Samoan",
		"gd": "Scots Gaelic", "sr": "Serbian", "st": "Sesotho", "sn": "Shona",
		"sd": "Sindhi", "si": "Sinhala", "sk": "Slovak", "sl": "Slovenian",
		"so": "Somali", "es": "Spanish", "su": "Sundanese", "sw": "Swahili",
		"sv": "Swedish", "tg": "Tajik", "ta": "Tamil", "tt": "Tatar",
		"te": "Telugu", "th": "Thai", "tr": "Turkish", "tk": "Turkmen",
		"uk": "Ukrainian", "ur": "Urdu", "ug": "Uyghur", "uz": "Uzbek",
		"vi": "Vietnamese", "cy": "Welsh", "xh": "Xhosa", "yi": "Yiddish",
		"yo": "Yoruba", "zu": "Zulu",
	}
	if name, ok := names[code]; ok {
		return name
	}
	return code
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
