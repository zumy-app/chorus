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
	// ProviderOpenRouter uses OpenRouter (OpenAI-compatible API).
	ProviderOpenRouter ProviderType = "openrouter"
	// ProviderOllama uses a local Ollama instance.
	ProviderOllama ProviderType = "ollama"
	// ProviderEngine uses the legacy llama.cpp translator engine.
	ProviderEngine ProviderType = "translator-engine"
	// ProviderNvidia uses NVIDIA AI Foundation endpoints (OpenAI-compatible).
	ProviderNvidia ProviderType = "nvidia"
	// ProviderLibreTranslate uses a local LibreTranslate (Argos/OPUS-MT) sidecar.
	ProviderLibreTranslate ProviderType = "libretranslate"
)

// Config holds the configuration for creating a translation provider.
type Config struct {
	// Provider is the name of the provider to create.
	// Supported values: "opencode", "openai", "deepseek", "nvidia", "ollama", "translator-engine"
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

	// Timeout is the HTTP client timeout in seconds.
	// 0 or negative means use the provider-specific default.
	Timeout int `json:"timeout"`
}

// NeedsAPIKey reports whether a provider type requires an API key to function.
// Local/offline providers (ollama, translator-engine, libretranslate) run
// without one; every cloud/OpenAI-compatible provider needs a key.
func NeedsAPIKey(t ProviderType) bool {
	switch t {
	case ProviderOpenCode, ProviderOpenAI, ProviderOpenRouter, ProviderDeepSeek, ProviderNvidia:
		return true
	default:
		return false
	}
}

// IsLocalProvider reports whether a provider type runs locally (offline) and
// can therefore serve as a guaranteed fallback when cloud providers are
// unavailable or exhausted.
func IsLocalProvider(t ProviderType) bool {
	switch t {
	case ProviderOllama, ProviderEngine, ProviderLibreTranslate:
		return true
	default:
		return false
	}
}

// Configured reports whether the configuration has everything required to
// create a working provider. Cloud providers need an API key; local/offline
// providers always count as configured as long as the type is valid.
func (c Config) Configured() bool {
	if NeedsAPIKey(c.Provider) {
		return c.APIKey != ""
	}
	return IsLocalProvider(c.Provider) || c.Provider == ""
}

// NewProvider creates a translation provider based on the given configuration.
// Returns an error if the provider type is unknown.
func NewProvider(cfg Config) (Provider, error) {
	switch cfg.Provider {
	case ProviderOpenCode, ProviderOpenAI, ProviderOpenRouter, ProviderDeepSeek, ProviderNvidia:
		// All OpenAI-compatible providers use the same implementation.
		baseURL := cfg.APIURL
		if baseURL == "" {
			switch cfg.Provider {
			case ProviderOpenCode:
				baseURL = "https://api.opencode.com/v1"
			case ProviderOpenAI:
				baseURL = "https://api.openai.com/v1"
			case ProviderOpenRouter:
				baseURL = "https://openrouter.ai/api/v1"
			case ProviderDeepSeek:
				baseURL = "https://api.deepseek.com/v1"
			case ProviderNvidia:
				baseURL = "https://integrate.api.nvidia.com/v1"
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
		return NewOpenAIProvider(baseURL, cfg.APIKey, model, cfg.Timeout), nil

	case ProviderOllama:
		baseURL := cfg.APIURL
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		model := cfg.Model
		if model == "" {
			model = "qwen2.5:3b"
		}
		return NewOllamaProvider(baseURL, model, cfg.Timeout), nil

	case ProviderEngine:
		baseURL := cfg.APIURL
		if baseURL == "" {
			baseURL = "http://localhost:5002"
		}
		return NewEngineProvider(baseURL), nil

	case ProviderLibreTranslate:
		baseURL := cfg.APIURL
		if baseURL == "" {
			baseURL = LibreTranslateDefaultBaseURL
		}
		return NewLibreTranslateProvider(baseURL, cfg.APIKey, cfg.Timeout), nil

	default:
		validProviders := []string{
			string(ProviderOpenCode),
			string(ProviderOpenAI),
			string(ProviderOpenRouter),
			string(ProviderDeepSeek),
			string(ProviderNvidia),
			string(ProviderOllama),
			string(ProviderEngine),
			string(ProviderLibreTranslate),
		}
		return nil, fmt.Errorf("unknown translation provider type: %q (valid: %s)",
			cfg.Provider, strings.Join(validProviders, ", "))
	}
}