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

// OllamaProvider translates text using a local Ollama instance.
// It uses the /api/chat endpoint with the OpenAI-compatible message format.
type OllamaProvider struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewOllamaProvider creates a new Ollama translation provider.
//
//   - baseURL: The Ollama server URL (e.g. "http://localhost:11434").
//   - model:   The model name (e.g. "qwen2.5:3b").
func NewOllamaProvider(baseURL, model string) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "qwen2.5:1.5b-instruct"
	}
	return &OllamaProvider{
		baseURL:    strings.TrimRight(baseURL, "/"),
		model:      model,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the provider name for logging/metrics.
func (p *OllamaProvider) Name() string {
	return fmt.Sprintf("ollama(%s)", p.model)
}

// ollamaChatMessage represents a message in the Ollama chat format.
type ollamaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ollamaChatRequest is the request body for Ollama's /api/chat endpoint.
type ollamaChatRequest struct {
	Model    string              `json:"model"`
	Messages []ollamaChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
	Options  map[string]any      `json:"options,omitempty"`
}

// ollamaChatResponse is the response from Ollama's /api/chat endpoint.
type ollamaChatResponse struct {
	Message ollamaChatMessage `json:"message"`
}

// Translate translates text using the local Ollama instance.
func (p *OllamaProvider) Translate(ctx context.Context, req TranslateRequest) (TranslateResponse, error) {
	if p.baseURL == "" {
		return TranslateResponse{}, fmt.Errorf("%w: Ollama URL is empty", ErrNotConfigured)
	}

	langName := languageCodeToName(req.TargetLang)
	if langName == "" {
		langName = req.TargetLang
	}

	prompt := fmt.Sprintf("Translate the following text to %s. Return ONLY the translated text, preserving all original formatting.\n\n%s", langName, req.Text)

	chatReq := ollamaChatRequest{
		Model: p.model,
		Messages: []ollamaChatMessage{
			{Role: "system", Content: "You are a precise translation engine. Return only the translated text with no preamble."},
			{Role: "user", Content: prompt},
		},
		Stream: false,
		Options: map[string]any{
			"temperature": 0.1,
			"top_p":       0.9,
			"num_predict": 128,
		},
	}

	body, err := json.Marshal(chatReq)
	if err != nil {
		return TranslateResponse{}, fmt.Errorf("ollama marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return TranslateResponse{}, fmt.Errorf("ollama create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return TranslateResponse{}, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return TranslateResponse{}, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var chatResp ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return TranslateResponse{}, fmt.Errorf("ollama decode response: %w", err)
	}

	translated := strings.TrimSpace(chatResp.Message.Content)
	translated = stripQuotes(translated)

	if translated == "" {
		return TranslateResponse{}, fmt.Errorf("%w: Ollama returned empty content", ErrEmptyResponse)
	}

	return TranslateResponse{
		TranslatedText: translated,
		Provider:       p.Name(),
	}, nil
}