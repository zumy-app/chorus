package translation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// LanguageDetector is implemented by providers that can identify the language
// of a piece of text and return its ISO 639-1 code (e.g. "en", "es", "fr").
type LanguageDetector interface {
	DetectLanguage(ctx context.Context, text string) (string, error)
}

const detectLanguageSystemPrompt = "Identify the language of the following text. Reply with ONLY the two-letter ISO 639-1 language code (for example: en, es, fr, de, it, pt, hi, ar, zh). Do not add any explanation."

func detectLanguageUserPrompt(text string) string {
	return "Text:\n" + text
}

// normalizeDetectedCode cleans a model reply into an ISO 639-1 code, or
// returns "" if it cannot be interpreted. It tolerates stray quotes,
// backticks, whitespace, trailing words, and region variants (es-MX).
func normalizeDetectedCode(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, "`\"' \n\r\t")
	raw = strings.ToLower(raw)
	if i := strings.IndexAny(raw, " \n\r\t,;.:"); i > 0 {
		raw = raw[:i]
	}
	// Strip region subtag from variants like es-MX, pt-BR, zh-CN.
	if i := strings.Index(raw, "-"); i > 0 {
		raw = raw[:i]
	}
	raw = strings.TrimSpace(raw)
	if len(raw) != 2 || languageCodeToName(raw) == "" {
		return ""
	}
	return raw
}

// detectChatMessage/detectChatRequest/detectChatResponse are a minimal
// OpenAI-compatible chat payload used only for language detection. They are
// intentionally separate from each provider's translate structs.
type detectChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type detectChatRequest struct {
	Model       string             `json:"model"`
	Messages    []detectChatMessage `json:"messages"`
	Temperature float64            `json:"temperature"`
	MaxTokens   int                `json:"max_tokens"`
}

type detectChatChoice struct {
	Message detectChatMessage `json:"message"`
}

type detectChatResponse struct {
	Choices []detectChatChoice `json:"choices"`
}

// detectLanguageFromLLM asks any OpenAI-compatible chat endpoint to identify
// the language of text and returns the normalized ISO 639-1 code. endpoint is
// the full chat/completions URL; apiKey may be empty for unauthenticated
// local endpoints (translator engine, Ollama).
func detectLanguageFromLLM(ctx context.Context, client *http.Client, endpoint, apiKey, model, system, user string) (string, error) {
	if strings.TrimSpace(model) == "" {
		model = "default"
	}
	chatReq := detectChatRequest{
		Model: model,
		Messages: []detectChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Temperature: 0,
		MaxTokens:   8,
	}
	body, err := json.Marshal(chatReq)
	if err != nil {
		return "", fmt.Errorf("detect marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("detect create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("detect request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", NewHTTPStatusError("detect", resp.StatusCode, string(respBody))
	}
	var chatResp detectChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("detect decode response: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("%w: no choices in detect response", ErrEmptyResponse)
	}
	code := normalizeDetectedCode(chatResp.Choices[0].Message.Content)
	if code == "" {
		return "", fmt.Errorf("%w: detect response was not a language code", ErrEmptyResponse)
	}
	return code, nil
}
