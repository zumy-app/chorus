package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/chorus/messenger/pkg/translation"
)

// ProviderDef holds the configuration for a single provider in the chain.
type ProviderDef struct {
	Type    string // "opencode", "ollama", "translator-engine", etc.
	APIURL  string
	APIKey  string
	Model   string
	Timeout int // seconds; 0 = use default
}

// Config holds all application configuration.
type Config struct {
	Environment           string
	DatabaseURL           string
	RedisURL              string
	JWTSecret             string
	GoogleTranslateAPIKey string
	Port                  string
	AppwriteEndpoint      string
	AppwriteProjectID     string
	AppwriteAPIKey        string
	AppwriteDatabaseID    string

	// Legacy single-provider config (used when EXTERNAL_LLM_PROVIDER_ORDER is not set).
	TranslationProviderName  string
	TranslationProviderURL   string
	TranslationProviderKey   string
	TranslationProviderModel string

	// TranslationChainTimeout is the total timeout for the translation chain
	// (across all providers tried sequentially), in seconds.
	TranslationChainTimeout int

	// LogLevel controls verbosity: "debug", "info", "warn", "error"
	LogLevel string

	// Provider chain — ordered list of aliases (e.g. ["openrouter", "nvidia", "ollama_local"]).
	// Each alias maps into the Providers map.
	// Used for BOTH translation and grammar AI analysis.
	ExternalLLMProviderOrder []string
	Providers                map[string]ProviderDef // key = alias, e.g. "openrouter", "ollama_local"
}

// Load reads configuration from environment variables.
func Load() *Config {
	cfg := &Config{
		Environment:           getEnv("ENVIRONMENT", "development"),
		DatabaseURL:           getEnv("DATABASE_URL", "postgres://messenger:password@localhost:5432/messenger_dev?sslmode=disable"),
		RedisURL:              getEnv("REDIS_URL", "localhost:6379"),
		JWTSecret:             getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		GoogleTranslateAPIKey: getEnv("GOOGLE_TRANSLATE_API_KEY", ""),
		Port:                  getEnv("PORT", "8080"),
		AppwriteEndpoint:      getEnv("APPWRITE_ENDPOINT", ""),
		AppwriteProjectID:     getEnv("APPWRITE_PROJECT_ID", ""),
		AppwriteAPIKey:        getEnv("APPWRITE_API_KEY", ""),
		AppwriteDatabaseID:    getEnv("APPWRITE_DATABASE_ID", ""),

		// Legacy single-provider config.
		TranslationProviderName:  getEnv("TRANSLATION_PROVIDER_NAME", string(translation.ProviderOpenCode)),
		TranslationProviderURL:   getEnv("TRANSLATION_PROVIDER_API_URL", ""),
		TranslationProviderKey:   getEnv("TRANSLATION_PROVIDER_API_KEY", ""),
		TranslationProviderModel: getEnv("TRANSLATION_PROVIDER_MODEL", ""),

		LogLevel: getEnv("LOG_LEVEL", "info"),

		TranslationChainTimeout: getEnvInt("TRANSLATION_CHAIN_TIMEOUT", 120),

		Providers: make(map[string]ProviderDef),
	}

	// Parse the unified provider order (used for BOTH translation and grammar).
	order := getEnv("EXTERNAL_LLM_PROVIDER_ORDER", "")
	if order != "" {
		cfg.ExternalLLMProviderOrder = splitAndTrim(order)
	}

	// Parse individual provider definitions.
	// Format: PROVIDER_<ALIAS>_<KEY>
	// Example: PROVIDER_OPENROUTER_TYPE=opencode
	//          PROVIDER_OPENROUTER_API_URL=https://openrouter.ai
	//          PROVIDER_OPENROUTER_API_KEY=sk-...
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, "PROVIDER_") {
			continue
		}
		// env is "PROVIDER_<ALIAS>_<KEY>=<VALUE>"
		eqIdx := strings.Index(env, "=")
		if eqIdx == -1 {
			continue
		}
		key := env[:eqIdx]     // PROVIDER_ALIAS_KEY
		value := env[eqIdx+1:] // the value

		// Strip "PROVIDER_" prefix -> "ALIAS_KEY"
		rest := key[len("PROVIDER_"):]

		// Match known field suffixes from longest to shortest
		// so "API_KEY" is matched before just "KEY", and multi-part
		// aliases like "opencode_go" are handled correctly.
		type suffixEntry struct {
			suffix string // with leading underscore, e.g. "_API_KEY"
			field  string // the field name, e.g. "API_KEY"
		}
		knownFields := []suffixEntry{
			{"_API_KEY", "API_KEY"},
			{"_API_URL", "API_URL"},
			{"_TIMEOUT", "TIMEOUT"},
			{"_MODEL", "MODEL"},
			{"_TYPE", "TYPE"},
		}
		var alias string
		var field string
		for _, entry := range knownFields {
			if strings.HasSuffix(rest, entry.suffix) {
				alias = strings.ToLower(rest[:len(rest)-len(entry.suffix)])
				field = entry.field
				break
			}
		}
		if alias == "" {
			continue
		}

		def := cfg.Providers[alias]
		switch field {
		case "TYPE":
			def.Type = value
		case "API_URL":
			def.APIURL = value
		case "API_KEY":
			def.APIKey = value
		case "MODEL":
			def.Model = value
		case "TIMEOUT":
			if v, err := strconv.Atoi(value); err == nil {
				def.Timeout = v
			}
		}
		cfg.Providers[alias] = def
	}

	// If no order is specified, default to legacy single-provider mode.
	// The caller (main.go) handles this by checking len(ExternalLLMProviderOrder)==0.
	if len(cfg.ExternalLLMProviderOrder) == 0 {
		log.Printf("[Config] EXTERNAL_LLM_PROVIDER_ORDER not set, using legacy single-provider config")
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}