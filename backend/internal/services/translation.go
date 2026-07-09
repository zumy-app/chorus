package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

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

// OpenAI-compatible Chat Completion types (used with llama.cpp server)
type ChatCompletionRequest struct {
	Model       string         `json:"model"`
	Messages    []ChatMessage  `json:"messages"`
	Temperature float64        `json:"temperature"`
	MaxTokens   int            `json:"max_tokens"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	Choices []ChatChoice `json:"choices"`
}

type ChatChoice struct {
	Message  ChatResponseMessage `json:"message"`
	Index    int                  `json:"index"`
	FinishReason string           `json:"finish_reason"`
}

type ChatResponseMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type TranslationQueueJob struct {
	MessageID    string   `json:"messageId"`
	Text         string   `json:"text"`
	TargetLangs  []string `json:"targetLangs"`
}

type TranslationService struct {
	redis               *redis.Client
	ollamaURL           string
	ollamaModel         string
	translatorEngineURL string
	httpClient          *http.Client
	queueEnabled        bool
	ctx                 context.Context
}

func NewTranslationService(translatorEngineURL, ollamaURL, ollamaModel string, redis *redis.Client) *TranslationService {
	return &TranslationService{
		redis:               redis,
		ollamaURL:           ollamaURL,
		ollamaModel:         ollamaModel,
		translatorEngineURL: translatorEngineURL,
		httpClient:          &http.Client{Timeout: 120 * time.Second},
		queueEnabled:        true,
		ctx:                 context.Background(),
	}
}

func (s *TranslationService) Translate(text, targetLang string) (string, error) {
	return s.TranslateQuick(text, targetLang, "auto")
}

// TranslateQuick translates text using the llama.cpp translator engine.
// Uses the OpenAI-compatible /v1/chat/completions endpoint.
// Retries up to 3 times with backoff to handle container startup delay.
func (s *TranslationService) TranslateQuick(text, targetLang, sourceLang string) (string, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("translation:engine:%s:%s:%s", sourceLang, targetLang, text)
	if s.redis != nil {
		cached, err := s.redis.Get(s.ctx, cacheKey).Result()
		if err == nil && cached != "" {
			return cached, nil
		}
	}

	if s.translatorEngineURL == "" {
		return "", errors.New("translator engine URL not configured")
	}

	// Build the translation prompt
	// ALMA-7B / Madlad-400 format: "Translate from {source} to {target}: {text}"
	sourceName := languageCodeToName(sourceLang)
	targetName := languageCodeToName(targetLang)
	if sourceLang == "" || sourceLang == "auto" {
		sourceName = "English"
	}

	systemPrompt := "You are a professional translator. Translate the user's text accurately and naturally. Return ONLY the translated text, without any explanations, prefixes, or quotes."
	userPrompt := fmt.Sprintf("Translate from %s to %s: %s", sourceName, targetName, text)

	reqBody := ChatCompletionRequest{
		Model: "default",
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.1,
		MaxTokens:   512,
	}
	body, _ := json.Marshal(reqBody)

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}

		req, _ := http.NewRequestWithContext(s.ctx, "POST", s.translatorEngineURL+"/v1/chat/completions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := s.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("translator engine call (attempt %d): %w", attempt+1, err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			var chatResp ChatCompletionResponse
			if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
				resp.Body.Close()
				lastErr = fmt.Errorf("decoding response (attempt %d): %w", attempt+1, err)
				continue
			}
			resp.Body.Close()

			if len(chatResp.Choices) > 0 && chatResp.Choices[0].Message.Content != "" {
				translated := strings.TrimSpace(chatResp.Choices[0].Message.Content)
				// Strip surrounding quotes if the model added them
				if len(translated) >= 2 {
					if (translated[0] == '"' && translated[len(translated)-1] == '"') ||
						(translated[0] == '\'' && translated[len(translated)-1] == '\'') {
						translated = translated[1 : len(translated)-1]
					}
				}
				if s.redis != nil {
					s.redis.Set(s.ctx, cacheKey, translated, 24*time.Hour)
				}
				return translated, nil
			}
			lastErr = fmt.Errorf("translator engine returned empty response (attempt %d)", attempt+1)
		} else {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("translator engine status %d (attempt %d): %s", resp.StatusCode, attempt+1, string(respBody))
		}
	}
	return "", lastErr
}

func (s *TranslationService) TranslateWithOllama(text, targetLang string) (string, error) {
	if s.ollamaURL == "" {
		return "", errors.New("Ollama URL not configured")
	}
	langName := languageCodeToName(targetLang)
	prompt := fmt.Sprintf(
		"Translate the following text to %s. Return ONLY the translated text, preserving all original line breaks and formatting. Do not add any explanation or prefix.\n\n%s",
		langName, text,
	)
	reqBody := OllamaGenerateRequest{Model: s.ollamaModel, Prompt: prompt, Stream: false}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(s.ctx, "POST", s.ollamaURL+"/api/generate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama call: %w", err)
	}
	defer resp.Body.Close()
	var ollamaResp OllamaGenerateResponse
	json.NewDecoder(resp.Body).Decode(&ollamaResp)
	result := strings.TrimSpace(ollamaResp.Response)
	if len(result) >= 2 {
		if (result[0] == '"' && result[len(result)-1] == '"') ||
			(result[0] == '\'' && result[len(result)-1] == '\'') {
			result = result[1 : len(result)-1]
		}
	}
	if result == "" {
		return "", errors.New("ollama empty response")
	}
	if s.redis != nil {
		s.redis.Set(s.ctx, fmt.Sprintf("translation:ollama:%s:%s", targetLang, text), result, 24*time.Hour)
	}
	return result, nil
}

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

func (s *TranslationService) EnqueueOllamaTranslation(messageID, text string, targetLangs []string) error {
	if s.redis == nil {
		return errors.New("redis not available")
	}
	job := TranslationQueueJob{MessageID: messageID, Text: text, TargetLangs: targetLangs}
	data, _ := json.Marshal(job)
	return s.redis.LPush(s.ctx, "translation:ollama:queue", data).Err()
}

func (s *TranslationService) ProcessOllamaQueue(onComplete func(messageID string, translations map[string]string)) {
	if s.redis == nil {
		return
	}
	for {
		result, err := s.redis.BRPop(s.ctx, 30*time.Second, "translation:ollama:queue").Result()
		if err != nil {
			continue
		}
		if len(result) < 2 {
			continue
		}
		var job TranslationQueueJob
		if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
			continue
		}
		translations := make(map[string]string)
		for _, lang := range job.TargetLangs {
			trans, err := s.TranslateWithOllama(job.Text, lang)
			if err != nil {
				fmt.Printf("[OllamaQueue] failed %s: %v\n", lang, err)
				continue
			}
			translations[lang] = trans
		}
		if len(translations) > 0 {
			onComplete(job.MessageID, translations)
		}
	}
}



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