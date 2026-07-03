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

	"github.com/redis/go-redis/v9"
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

type LibreTranslateRequest struct {
	Q      string `json:"q"`
	Source string `json:"source"`
	Target string `json:"target"`
	Format string `json:"format"`
}

type LibreTranslateResponse struct {
	TranslatedText string `json:"translatedText"`
}

type TranslationQueueJob struct {
	MessageID    string   `json:"messageId"`
	Text         string   `json:"text"`
	TargetLangs  []string `json:"targetLangs"`
}

type TranslationService struct {
	redis        *redis.Client
	ollamaURL    string
	ollamaModel  string
	libreURL     string
	httpClient   *http.Client
	queueEnabled bool
	ctx          context.Context
}

func NewTranslationService(libreURL, ollamaURL, ollamaModel string, redis *redis.Client) *TranslationService {
	return &TranslationService{
		redis:        redis,
		ollamaURL:    ollamaURL,
		ollamaModel:  ollamaModel,
		libreURL:     libreURL,
		httpClient:   &http.Client{Timeout: 120 * time.Second},
		queueEnabled: true,
		ctx:          context.Background(),
	}
}

func (s *TranslationService) Translate(text, targetLang string) (string, error) {
	return s.TranslateQuick(text, targetLang, "auto")
}

func (s *TranslationService) TranslateQuick(text, targetLang, sourceLang string) (string, error) {
	cacheKey := fmt.Sprintf("translation:libre:%s:%s", targetLang, text)
	if s.redis != nil {
		cached, err := s.redis.Get(s.ctx, cacheKey).Result()
		if err == nil && cached != "" {
			return cached, nil
		}
	}
	if s.libreURL == "" {
		return "", errors.New("LibreTranslate URL not configured")
	}
	reqBody := LibreTranslateRequest{
		Q:      text,
		Source: sourceLang,
		Target: targetLang,
		Format: "text",
	}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(s.ctx, "POST", s.libreURL+"/translate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("libre call: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("libre status %d: %s", resp.StatusCode, string(respBody))
	}
	var libreResp LibreTranslateResponse
	json.NewDecoder(resp.Body).Decode(&libreResp)
	if libreResp.TranslatedText == "" {
		return "", errors.New("libre empty response")
	}
	if s.redis != nil {
		s.redis.Set(s.ctx, cacheKey, libreResp.TranslatedText, 24*time.Hour)
	}
	return libreResp.TranslatedText, nil
}

func (s *TranslationService) TranslateWithOllama(text, targetLang string) (string, error) {
	if s.ollamaURL == "" {
		return "", errors.New("Ollama URL not configured")
	}
	langName := languageCodeToName(targetLang)
	// Explicitly instruct the model to preserve line breaks so multi-line messages
	// are not concatenated into a single line in the translation.
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
	// Only trim outer whitespace — do NOT use strings.Trim with quote chars,
	// which would also strip quotes that are part of the translated text.
	result := strings.TrimSpace(ollamaResp.Response)
	// Strip a single wrapping pair of quotes if the model wrapped the whole
	// translation in them (e.g. `"Bonjour le monde"`), but leave internal quotes alone.
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
	m := map[string]string{"en": "English", "es": "Spanish", "fr": "French", "de": "German", "it": "Italian", "pt": "Portuguese", "ja": "Japanese", "ko": "Korean", "zh": "Chinese", "ar": "Arabic", "nl": "Dutch", "pl": "Polish", "ru": "Russian", "sv": "Swedish"}
	if v, ok := m[code]; ok {
		return v
	}
	return code
}
