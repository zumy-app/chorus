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

	"github.com/chorus/messenger/pkg/logutil"
)

// defaultPullTimeout bounds a single Ollama /api/pull request. Pulling a model
// downloads it (qwen2.5:3b ≈ 1.9 GB) and can take several minutes on first run.
const defaultPullTimeout = 30 * time.Minute

// OllamaProvider translates text using a local Ollama instance.
// It uses the /api/generate endpoint for efficiency (no streaming, raw response).
type OllamaProvider struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewOllamaProvider creates a new Ollama translation provider.
//
//   - baseURL:   The Ollama server URL (e.g. "http://localhost:11434").
//   - model:     The model name (e.g. "qwen2.5:3b").
//   - timeoutSec: HTTP client timeout in seconds; <= 0 defaults to 120.
func NewOllamaProvider(baseURL, model string, timeoutSec int) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "qwen2.5:1.5b-instruct"
	}
	if timeoutSec <= 0 {
		timeoutSec = 120
	}
	return &OllamaProvider{
		baseURL:    strings.TrimRight(baseURL, "/"),
		model:      model,
		httpClient: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}
}

// Name returns the provider name for logging/metrics.
func (p *OllamaProvider) Name() string {
	return fmt.Sprintf("ollama(%s)", p.model)
}

// Ping performs a lightweight readiness check against the local Ollama server
// (/api/tags) and verifies the configured model is actually installed. Used by
// the startup probe.
func (p *OllamaProvider) Ping(ctx context.Context) error {
	installed, err := p.modelInstalled(ctx)
	if err != nil {
		return err
	}
	if !installed {
		return fmt.Errorf("ollama model %q is not installed (run: ollama pull %s)", p.model, p.model)
	}
	return nil
}

// modelInstalled reports whether the configured model is present in Ollama's
// model list (/api/tags).
func (p *OllamaProvider) modelInstalled(ctx context.Context) (bool, error) {
	if p.baseURL == "" {
		return false, fmt.Errorf("%w: Ollama URL is empty", ErrNotConfigured)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/api/tags", nil)
	if err != nil {
		return false, fmt.Errorf("ollama tags create request: %w", err)
	}
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return false, fmt.Errorf("ollama tags request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, NewHTTPStatusError(p.Name(), resp.StatusCode, "tags probe")
	}

	var tagsResp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return false, fmt.Errorf("ollama tags decode: %w", err)
	}
	return ollamaHasModel(tagsResp.Models, p.model), nil
}

// EnsureModel verifies the configured model is installed and pulls it if not,
// via the Ollama HTTP API (/api/pull). It returns nil once the model is
// available. This is what makes the offline local provider self-heal: a missing
// model is downloaded automatically instead of just warning.
func (p *OllamaProvider) EnsureModel(ctx context.Context) error {
	installed, err := p.modelInstalled(ctx)
	if err != nil {
		return err
	}
	if installed {
		return nil
	}

	logutil.Infof("[Ollama] model %q not installed — pulling via /api/pull (first run can take a few minutes)", p.model)

	pullCtx, cancel := context.WithTimeout(ctx, defaultPullTimeout)
	defer cancel()

	payload := fmt.Sprintf(`{"model":%q,"stream":false}`, p.model)
	httpReq, err := http.NewRequestWithContext(pullCtx, http.MethodPost, p.baseURL+"/api/pull", strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("ollama pull create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: defaultPullTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("ollama pull request failed: %w", err)
	}
	defer resp.Body.Close()

	var pullResp struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pullResp); err != nil {
		return fmt.Errorf("ollama pull decode response: %w", err)
	}
	if resp.StatusCode != http.StatusOK || pullResp.Error != "" {
		return fmt.Errorf("ollama pull failed (status %d): %s", resp.StatusCode, pullResp.Error)
	}

	installed, err = p.modelInstalled(ctx)
	if err != nil {
		return err
	}
	if !installed {
		return fmt.Errorf("ollama pull reported success but model %q is still not installed", p.model)
	}
	logutil.Infof("[Ollama] model %q installed successfully", p.model)
	return nil
}

// ollamaHasModel reports whether the tag list contains the given model name.
// Tag names can be a full match ("qwen2.5:3b") or an ollama digest reference.
func ollamaHasModel(models []struct {
	Name string `json:"name"`
}, want string) bool {
	for _, m := range models {
		if m.Name == want {
			return true
		}
	}
	return false
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
	Model    string `json:"model"`
	Response string `json:"response"`
	Done     bool   `json:"done"`
	Error    string `json:"error,omitempty"`
}

// Translate translates text using the local Ollama instance via /api/generate.
// If the configured model is missing, it automatically pulls it (EnsureModel)
// and retries once, so the offline provider self-heals even if the startup
// pull has not completed yet.
func (p *OllamaProvider) Translate(ctx context.Context, req TranslateRequest) (TranslateResponse, error) {
	resp, err := p.doGenerate(ctx, req)
	if err == nil || !IsModelNotFound(err) {
		return resp, err
	}

	logutil.Warnf("[Ollama] model %q missing — auto-pulling and retrying once", p.model)
	if pullErr := p.EnsureModel(ctx); pullErr != nil {
		return TranslateResponse{}, fmt.Errorf("ollama auto-pull failed: %v (original: %w)", pullErr, err)
	}
	return p.doGenerate(ctx, req)
}

// doGenerate performs a single translation request to /api/generate.
func (p *OllamaProvider) doGenerate(ctx context.Context, req TranslateRequest) (TranslateResponse, error) {
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
		return TranslateResponse{}, NewHTTPStatusError(p.Name(), resp.StatusCode, string(respBody))
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

// DetectLanguage identifies the language of text via Ollama's OpenAI-compatible
// chat endpoint (/v1/chat/completions), which Ollama exposes alongside its
// native /api/generate endpoint.
func (p *OllamaProvider) DetectLanguage(ctx context.Context, text string) (string, error) {
	if p.baseURL == "" {
		return "", fmt.Errorf("%w: Ollama URL is empty", ErrNotConfigured)
	}
	return detectLanguageFromLLM(ctx, p.httpClient, p.baseURL+"/v1/chat/completions",
		"", p.model, detectLanguageSystemPrompt, detectLanguageUserPrompt(text))
}
