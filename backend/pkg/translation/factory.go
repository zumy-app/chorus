package translation

import (
	"fmt"
	"strings"
)

// ProviderType represents the type of translation provider.
type ProviderType string

const (
	// ProviderOpenCode uses OpenCode Go (OpenAI-compatible API).
	ProviderOpenCode ProviderType = "opencode"
	// ProviderOpenAI uses OpenAI API.
	ProviderOpenAI ProviderType = "openai"
	// ProviderDeepSeek uses DeepSeek API (OpenAI-compatible).
	ProviderDeepSeek ProviderType = "deepseek"
	// ProviderOllama uses a local Ollama instance.
	ProviderOllama ProviderType = "ollama"
	// ProviderEngine uses the legacy llama.cpp translator engine.
	ProviderEngine ProviderType = "translator-engine"
)

// Config holds the configuration for creating a translation provider.
type Config struct {
	// Provider is the name of the provider to create.
	// Supported values: "opencode", "openai", "deepseek", "ollama", "translator-engine"
	Provider ProviderType `json:"provider"`

	// APIURL is the base URL for the provider's API.
	// Examples:
	//   - opencode:  https://api.opencode.com/v1
	//   - openai:    https://api.openai.com/v1
	//   - deepseek:  https://api.deepseek.com/v1
	//   - ollama:    http://localhost:11434
	//   - translator-engine: http://localhost:5002
	APIURL string `json:"api_url"`

	// APIKey is the API key for authentication.
	// Required for: opencode, openai, deepseek
	// Not needed for: ollama, translator-engine
	APIKey string `json:"api_key"`

	// Model is the model name to use for translation.
	// Examples:
	//   - opencode:  gpt-4o-mini
	//   - openai:    gpt-4o-mini
	//   - deepseek:  deepseek-chat
	//   - ollama:    qwen2.5:3b
	//   - translator-engine: (not used)
	Model string `json:"model"`
}

// NewProvider creates a translation provider based on the given configuration.
// Returns an error if the provider type is unknown.
func NewProvider(cfg Config) (Provider, error) {
	switch cfg.Provider {
	case ProviderOpenCode, ProviderOpenAI, ProviderDeepSeek:
		// All OpenAI-compatible providers use the same implementation.
		baseURL := cfg.APIURL
		if baseURL == "" {
			switch cfg.Provider {
			case ProviderOpenCode:
				baseURL = "https://api.opencode.com/v1"
			case ProviderOpenAI:
				baseURL = "https://api.openai.com/v1"
			case ProviderDeepSeek:
				baseURL = "https://api.deepseek.com/v1"
			}
		}
		model := cfg.Model
		if model == "" {
			switch cfg.Provider {
			case ProviderDeepSeek:
				model = "deepseek-chat"
			default:
				model = "gpt-4o-mini"
			}
		}
		return NewOpenAIProvider(baseURL, cfg.APIKey, model), nil

	case ProviderOllama:
		baseURL := cfg.APIURL
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		model := cfg.Model
		if model == "" {
			model = "qwen2.5:3b"
		}
		return NewOllamaProvider(baseURL, model), nil

	case ProviderEngine:
		baseURL := cfg.APIURL
		if baseURL == "" {
			baseURL = "http://localhost:5002"
		}
		return NewEngineProvider(baseURL), nil

	default:
		validProviders := []string{
			string(ProviderOpenCode),
			string(ProviderOpenAI),
			string(ProviderDeepSeek),
			string(ProviderOllama),
			string(ProviderEngine),
		}
		return nil, fmt.Errorf("unknown translation provider type: %q (valid: %s)",
			cfg.Provider, strings.Join(validProviders, ", "))
	}
}