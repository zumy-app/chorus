package translation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// EngineProvider translates text using the legacy llama.cpp translator engine.
// It uses the OpenAI-compatible /v1/chat/completions endpoint.
type EngineProvider struct {
	baseURL    string
	httpClient *http.Client
}

// NewEngineProvider creates a new translator engine provider.
//
//   - baseURL: The translator engine URL (e.g. "http://localhost:5002").
func NewEngineProvider(baseURL string) *EngineProvider {
	if baseURL == "" {
		baseURL = "http://localhost:5002"
	}
	return &EngineProvider{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

// Name returns the provider name for logging/metrics.
func (p *EngineProvider) Name() string {
	return "translator-engine"
}

// engineChatMessage represents a message in the OpenAI-compatible format.
type engineChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// engineChatRequest is the request body for /v1/chat/completions.
type engineChatRequest struct {
	Model       string              `json:"model"`
	Messages    []engineChatMessage `json:"messages"`
	Temperature float64             `json:"temperature"`
	MaxTokens   int                 `json:"max_tokens"`
}

// engineChatChoice represents a single choice in the response.
type engineChatChoice struct {
	Message engineChatMessage `json:"message"`
}

// engineChatResponse is the response from /v1/chat/completions.
type engineChatResponse struct {
	Choices []engineChatChoice `json:"choices"`
}

// Translate translates text using the translator engine.
func (p *EngineProvider) Translate(ctx context.Context, req TranslateRequest) (TranslateResponse, error) {
	if p.baseURL == "" {
		return TranslateResponse{}, fmt.Errorf("%w: translator engine URL is empty", ErrNotConfigured)
	}

	langName := languageCodeToName(req.TargetLang)
	if langName == "" {
		langName = req.TargetLang
	}

	sourceInfo := ""
	if req.SourceLang != "" && req.SourceLang != "auto" {
		sourceName := languageCodeToName(req.SourceLang)
		sourceInfo = fmt.Sprintf(" from %s", sourceName)
	}

	userMsg := fmt.Sprintf("Translate the following text%s to %s. Return ONLY the translated text.\n\n%s",
		sourceInfo, langName, req.Text)

	chatReq := engineChatRequest{
		Model: "default",
		Messages: []engineChatMessage{
			{Role: "system", Content: "You are a translation engine. Translate the text exactly. Return only the translated text."},
			{Role: "user", Content: userMsg},
		},
		Temperature: 0.1,
		MaxTokens:   512,
	}

	body, err := json.Marshal(chatReq)
	if err != nil {
		return TranslateResponse{}, fmt.Errorf("engine marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return TranslateResponse{}, fmt.Errorf("engine create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return TranslateResponse{}, fmt.Errorf("engine request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return TranslateResponse{}, fmt.Errorf("engine returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var chatResp engineChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return TranslateResponse{}, fmt.Errorf("engine decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return TranslateResponse{}, fmt.Errorf("%w: no choices in engine response", ErrEmptyResponse)
	}

	translated := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	translated = stripQuotes(translated)

	if translated == "" {
		return TranslateResponse{}, fmt.Errorf("%w: engine returned empty content", ErrEmptyResponse)
	}

	return TranslateResponse{
		TranslatedText: translated,
		Provider:       p.Name(),
	}, nil
}