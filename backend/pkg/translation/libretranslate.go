package translation

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// LibreTranslateDefaultBaseURL is the default URL of the local LibreTranslate
// sidecar container ("libretranslate" resolves on the Docker bridge network).
const LibreTranslateDefaultBaseURL = "http://libretranslate:5000"

// LibreTranslateProvider translates text using a local LibreTranslate server
// (Argos / OPUS-MT backend). It implements the LibreTranslate REST API
// (POST /translate, form-encoded) exposed on port 5000 by the
// libretranslate container.
type LibreTranslateProvider struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewLibreTranslateProvider creates a new LibreTranslate provider.
//
//   - baseURL:    The LibreTranslate server URL (e.g. "http://libretranslate:5000").
//   - apiKey:     Optional API key; empty for an unauthenticated local server.
//   - timeoutSec: HTTP client timeout in seconds; <= 0 defaults to 10.
func NewLibreTranslateProvider(baseURL, apiKey string, timeoutSec int) *LibreTranslateProvider {
	if baseURL == "" {
		baseURL = LibreTranslateDefaultBaseURL
	}
	if timeoutSec <= 0 {
		timeoutSec = 10
	}
	return &LibreTranslateProvider{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}
}

// Name returns the provider name for logging/metrics.
func (p *LibreTranslateProvider) Name() string {
	return "libretranslate"
}

// Ping performs a lightweight readiness check against the LibreTranslate
// /languages endpoint. Used by the startup probe to warn when the offline
// translation sidecar is unreachable.
func (p *LibreTranslateProvider) Ping(ctx context.Context) error {
	if p.baseURL == "" {
		return fmt.Errorf("%w: LibreTranslate URL is empty", ErrNotConfigured)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/languages", nil)
	if err != nil {
		return fmt.Errorf("libretranslate ping create request: %w", err)
	}
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("libretranslate ping failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return NewHTTPStatusError(p.Name(), resp.StatusCode, "languages probe")
	}
	return nil
}

// libreTranslateResponse is the JSON payload returned by POST /translate.
type libreTranslateResponse struct {
	TranslatedText string `json:"translatedText"`
}

// Translate translates text via the LibreTranslate sidecar.
// Source "auto" is passed through — LibreTranslate detects languages natively.
func (p *LibreTranslateProvider) Translate(ctx context.Context, req TranslateRequest) (TranslateResponse, error) {
	if p.baseURL == "" {
		return TranslateResponse{}, fmt.Errorf("%w: LibreTranslate URL is empty", ErrNotConfigured)
	}

	source := strings.TrimSpace(req.SourceLang)
	if source == "" {
		source = "auto"
	}
	target := strings.TrimSpace(req.TargetLang)
	if target == "" {
		target = "en"
	}

	form := url.Values{}
	form.Set("q", req.Text)
	form.Set("source", source)
	form.Set("target", target)
	form.Set("format", "text")
	if p.apiKey != "" {
		form.Set("api_key", p.apiKey)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/translate", strings.NewReader(form.Encode()))
	if err != nil {
		return TranslateResponse{}, fmt.Errorf("libretranslate create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return TranslateResponse{}, fmt.Errorf("libretranslate request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return TranslateResponse{}, NewHTTPStatusError(p.Name(), resp.StatusCode, string(respBody))
	}

	var out libreTranslateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return TranslateResponse{}, fmt.Errorf("libretranslate decode response: %w", err)
	}

	translated := strings.TrimSpace(out.TranslatedText)
	if translated == "" {
		return TranslateResponse{}, fmt.Errorf("%w: LibreTranslate returned empty text", ErrEmptyResponse)
	}

	return TranslateResponse{
		TranslatedText: translated,
		Provider:       p.Name(),
	}, nil
}

// libreDetectResult is a single candidate from POST /detect.
type libreDetectResult struct {
	Language   string  `json:"language"`
	Confidence float64 `json:"confidence"`
}

// DetectLanguage identifies the language of text via LibreTranslate's native
// /detect endpoint, returning the highest-confidence ISO 639-1 code.
func (p *LibreTranslateProvider) DetectLanguage(ctx context.Context, text string) (string, error) {
	if p.baseURL == "" {
		return "", fmt.Errorf("%w: LibreTranslate URL is empty", ErrNotConfigured)
	}
	form := url.Values{}
	form.Set("q", text)
	if p.apiKey != "" {
		form.Set("api_key", p.apiKey)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/detect", strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("libretranslate detect create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("libretranslate detect request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", NewHTTPStatusError(p.Name(), resp.StatusCode, string(respBody))
	}

	var out []libreDetectResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("libretranslate detect decode response: %w", err)
	}
	best := ""
	bestConf := 0.0
	for _, r := range out {
		if r.Confidence > bestConf && languageCodeToName(strings.TrimSpace(r.Language)) != "" {
			best = strings.TrimSpace(r.Language)
			bestConf = r.Confidence
		}
	}
	if best == "" {
		return "", fmt.Errorf("%w: LibreTranslate could not detect a language", ErrEmptyResponse)
	}
	return best, nil
}
