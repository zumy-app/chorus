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
// It uses the /api/generate endpoint for efficiency (no streaming, raw response).
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
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

// Name returns the provider name for logging/metrics.
func (p *OllamaProvider) Name() string {
	return fmt.Sprintf("ollama(%s)", p.model)
}

// ollamaGenerateRequest is the request body for Ollama's /api/generate endpoint.
type ollamaGenerateRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	Stream  bool           `json:"stream"`
	System  string         `json:"system,omitempty"`
	Options map[string]any `json:"options,omitempty"`
}

// ollamaGenerateResponse is the response from Ollama's /api/generate endpoint.
type ollamaGenerateResponse struct {
	Model     string `json:"model"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
	Error     string `json:"error,omitempty"`
}

// Translate translates text using the local Ollama instance via /api/generate.
func (p *OllamaProvider) Translate(ctx context.Context, req TranslateRequest) (TranslateResponse, error) {
	if p.baseURL == "" {
		return TranslateResponse{}, fmt.Errorf("%w: Ollama URL is empty", ErrNotConfigured)
	}

	langName := languageCodeToName(req.TargetLang)
	if langName == "" {
		langName = req.TargetLang
	}

	userPrompt := fmt.Sprintf("Translate ALL of the following text to %s. Do not skip any part. Return ONLY the complete translated text, preserving all original formatting.\n\n%s", langName, req.Text)

	genReq := ollamaGenerateRequest{
		Model:  p.model,
		Prompt: userPrompt,
		Stream: false,
		System: "You are a precise translation engine. Return only the translated text with no preamble.",
		Options: map[string]any{
			"temperature": 0.1,
			"top_p":       0.9,
			"num_predict": 512,
		},
	}

	body, err := json.Marshal(genReq)
	if err != nil {
		return TranslateResponse{}, fmt.Errorf("ollama marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return TranslateResponse{}, fmt.Errorf("ollama create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return TranslateResponse{}, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return TranslateResponse{}, fmt.Errorf("ollama read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return TranslateResponse{}, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var genResp ollamaGenerateResponse
	if err := json.Unmarshal(respBody, &genResp); err != nil {
		return TranslateResponse{}, fmt.Errorf("ollama decode response: %w", err)
	}

	if genResp.Error != "" {
		return TranslateResponse{}, fmt.Errorf("ollama error: %s", genResp.Error)
	}

	translated := strings.TrimSpace(genResp.Response)
	translated = stripQuotes(translated)

	if translated == "" {
		return TranslateResponse{}, fmt.Errorf("%w: Ollama returned empty content", ErrEmptyResponse)
	}

	return TranslateResponse{
		TranslatedText: translated,
		Provider:       p.Name(),
	}, nil
}
