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

// NLLBDefaultBaseURL is the default URL of the local NLLB-200 CTranslate2
// sidecar container ("nllb-local" resolves on the Docker bridge network).
const NLLBDefaultBaseURL = "http://nllb-local:5001"

// NLLBProvider translates text using the local NLLB-200 CTranslate2 sidecar.
// It implements the OpenAI-compatible-free /translate JSON API exposed on
// port 5001 by the nllb-local container.
type NLLBProvider struct {
	baseURL    string
	httpClient *http.Client
}

// NewNLLBProvider creates a new NLLB sidecar provider.
//
//   - baseURL:    The sidecar URL (e.g. "http://nllb-local:5001").
//   - timeoutSec: HTTP client timeout in seconds; <= 0 defaults to 60.
func NewNLLBProvider(baseURL string, timeoutSec int) *NLLBProvider {
	if baseURL == "" {
		baseURL = NLLBDefaultBaseURL
	}
	if timeoutSec <= 0 {
		timeoutSec = 60
	}
	return &NLLBProvider{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}
}

// Name returns the provider name for logging/metrics.
func (p *NLLBProvider) Name() string {
	return "nllb-local"
}

// nllbTranslateRequest is the JSON payload for POST /translate.
type nllbTranslateRequest struct {
	Text   string `json:"text"`
	Source string `json:"source"`
	Target string `json:"target"`
}

// nllbTranslateResponse is the JSON payload returned by POST /translate.
type nllbTranslateResponse struct {
	TranslatedText string `json:"translatedText"`
}

// Translate translates text via the NLLB sidecar.
func (p *NLLBProvider) Translate(ctx context.Context, req TranslateRequest) (TranslateResponse, error) {
	if p.baseURL == "" {
		return TranslateResponse{}, fmt.Errorf("%w: NLLB sidecar URL is empty", ErrNotConfigured)
	}

	source := req.SourceLang
	if source == "" || source == "auto" {
		source = "en"
	}
	target := req.TargetLang
	if target == "" {
		target = "en"
	}

	body, err := json.Marshal(nllbTranslateRequest{
		Text:   req.Text,
		Source: source,
		Target: target,
	})
	if err != nil {
		return TranslateResponse{}, fmt.Errorf("nllb marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/translate", bytes.NewReader(body))
	if err != nil {
		return TranslateResponse{}, fmt.Errorf("nllb create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return TranslateResponse{}, fmt.Errorf("nllb request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return TranslateResponse{}, fmt.Errorf("nllb returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var out nllbTranslateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return TranslateResponse{}, fmt.Errorf("nllb decode response: %w", err)
	}

	translated := strings.TrimSpace(out.TranslatedText)
	if translated == "" {
		return TranslateResponse{}, fmt.Errorf("%w: NLLB returned empty text", ErrEmptyResponse)
	}

	return TranslateResponse{
		TranslatedText: translated,
		Provider:       p.Name(),
	}, nil
}
