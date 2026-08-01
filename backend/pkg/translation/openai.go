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

// OpenAIProvider translates text using any OpenAI-compatible API endpoint.
// This works with OpenRouter, OpenCode Go, OpenAI, Codex, and any other
// provider that implements the /v1/chat/completions interface.
type OpenAIProvider struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewOpenAIProvider creates a new OpenAI-compatible translation provider.
//
//   - baseURL:   The API base URL (e.g. "https://api.opencode.com/v1").
//   - apiKey:    The API key for authentication.
//   - model:     The model name (e.g. "gpt-4o-mini", "claude-3-haiku", etc.).
//   - timeoutSec: HTTP client timeout in seconds; <= 0 defaults to 30.
func NewOpenAIProvider(baseURL, apiKey, model string, timeoutSec int) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.opencode.com/v1"
	}
	if model == "" {
		model = "gpt-4o-mini"
	}
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	return &OpenAIProvider{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}
}

// Name returns the provider name for logging/metrics.
func (p *OpenAIProvider) Name() string {
	return fmt.Sprintf("openai(%s)", p.model)
}

// NeedsKey reports whether this provider requires an API key. Used by the
// chain's startup health check to flag misconfigured cloud providers.
func (p *OpenAIProvider) NeedsKey() bool {
	return p.apiKey == ""
}

// openAIChatMessage represents a message in the OpenAI chat format.
type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openAIChatRequest is the request body for /v1/chat/completions.
type openAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	Temperature float64             `json:"temperature"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
}

// openAIChatChoice represents a single choice in the response.
type openAIChatChoice struct {
	Message      openAIChatMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
}

// openAIChatResponse is the response from /v1/chat/completions.
type openAIChatResponse struct {
	Choices []openAIChatChoice `json:"choices"`
}

// Translate translates text using the OpenAI-compatible API.
func (p *OpenAIProvider) Translate(ctx context.Context, req TranslateRequest) (TranslateResponse, error) {
	if p.apiKey == "" {
		return TranslateResponse{}, fmt.Errorf("%w: OpenAI API key is empty", ErrNotConfigured)
	}

	start := time.Now()
	logutil.Debugf("[%s] Translate start: text_len=%d target=%s", p.Name(), len(req.Text), req.TargetLang)
	defer func() {
		logutil.Duration(p.Name(), start, "Translate")
	}()

	langName := languageCodeToName(req.TargetLang)
	if langName == "" {
		langName = req.TargetLang
	}

	systemMsg := "You are a precise translation engine. Translate ALL of the user's text completely. " +
		"Preserve all original formatting, line breaks, and punctuation. " +
		"Return ONLY the complete translated text with no preamble, explanation, or commentary."

	userMsg := fmt.Sprintf("Translate ALL of the following text to %s. Do not skip any part.\n\n%s", langName, req.Text)

	chatReq := openAIChatRequest{
		Model: p.model,
		Messages: []openAIChatMessage{
			{Role: "system", Content: systemMsg},
			{Role: "user", Content: userMsg},
		},
		Temperature: 0.1,
		MaxTokens:   4096,
	}

	body, err := json.Marshal(chatReq)
	if err != nil {
		return TranslateResponse{}, fmt.Errorf("openai marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return TranslateResponse{}, fmt.Errorf("openai create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return TranslateResponse{}, fmt.Errorf("openai request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return TranslateResponse{}, NewHTTPStatusError(p.Name(), resp.StatusCode, string(respBody))
	}

	var chatResp openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return TranslateResponse{}, fmt.Errorf("openai decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return TranslateResponse{}, fmt.Errorf("%w: no choices in response", ErrEmptyResponse)
	}

	if chatResp.Choices[0].FinishReason == "length" {
		return TranslateResponse{}, fmt.Errorf("%w: response truncated by max_tokens (finish_reason=length)", ErrEmptyResponse)
	}

	translated := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	// Strip any surrounding quotes the model might add
	translated = stripQuotes(translated)

	if translated == "" {
		return TranslateResponse{}, fmt.Errorf("%w: empty content in choice", ErrEmptyResponse)
	}

	return TranslateResponse{
		TranslatedText: translated,
		Provider:       p.Name(),
	}, nil
}

// stripQuotes removes matching surrounding quotes from a string.
func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') ||
			(s[0] == '\'' && s[len(s)-1] == '\'') ||
			(s[0] == '`' && s[len(s)-1] == '`') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// languageCodeToName maps ISO language codes to human-readable names.
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
