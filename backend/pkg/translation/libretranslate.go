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
		return TranslateResponse{}, fmt.Errorf("libretranslate returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
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