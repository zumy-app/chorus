package translation

import (
	"fmt"
)

// ProviderType represents the type of translation provider.
type ProviderType string

const (
	// ProviderOpenAI uses an OpenAI-compatible API (OpenRouter, OpenCode Go, OpenAI, Codex).
	ProviderOpenAI ProviderType = "openai"
	// ProviderOllama uses a local Ollama instance.
	ProviderOllama ProviderType = "ollama"
	// ProviderEngine uses the legacy llama.cpp translator engine.
	ProviderEngine ProviderType = "translator-engine"
)

// Config holds the configuration for creating a translation provider.
type Config struct {
	// Provider is the type of provider to create.
	Provider ProviderType `json:"provider"`

	// OpenAI-compatible settings.
	OpenAIBaseURL string `json:"openai_base_url"`
	OpenAIAPIKey  string `json:"openai_api_key"`
	OpenAIModel   string `json:"openai_model"`

	// Ollama settings.
	OllamaURL   string `json:"ollama_url"`
	OllamaModel string `json:"ollama_model"`

	// Translator engine settings.
	TranslatorEngineURL string `json:"translator_engine_url"`
}

// NewProvider creates a translation provider based on the given configuration.
// Returns an error if the provider type is unknown.
func NewProvider(cfg Config) (Provider, error) {
	switch cfg.Provider {
	case ProviderOpenAI:
		return NewOpenAIProvider(cfg.OpenAIBaseURL, cfg.OpenAIAPIKey, cfg.OpenAIModel), nil

	case ProviderOllama:
		return NewOllamaProvider(cfg.OllamaURL, cfg.OllamaModel), nil

	case ProviderEngine:
		return NewEngineProvider(cfg.TranslatorEngineURL), nil

	default:
		return nil, fmt.Errorf("unknown translation provider type: %q (valid: openai, ollama, translator-engine)", cfg.Provider)
	}
}