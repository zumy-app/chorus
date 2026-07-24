// Package translation provides an abstraction over different translation backends.
// Supported providers: OpenAI-compatible APIs (OpenRouter, OpenCode Go, OpenAI, Codex),
// Ollama (local LLM), and the legacy translator engine (llama.cpp).
package translation

import (
	"context"
	"errors"
)

// Common errors.
var (
	ErrNotConfigured = errors.New("translation provider not configured")
	ErrEmptyResponse = errors.New("translation provider returned empty response")
	ErrNoProvider    = errors.New("no translation provider configured")
)

// TranslateRequest holds the parameters for a translation request.
type TranslateRequest struct {
	Text       string
	SourceLang string // "auto" for auto-detect
	TargetLang string
}

// TranslateResponse holds the result of a translation.
type TranslateResponse struct {
	TranslatedText string
	Provider       string // name of the provider that handled the request
}

// Provider defines the interface for translation backends.
type Provider interface {
	// Translate translates the given text from source to target language.
	Translate(ctx context.Context, req TranslateRequest) (TranslateResponse, error)

	// Name returns a human-readable name for this provider (for logging/metrics).
	Name() string
}