package config

import (
	"fmt"
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
	SMTPHost              string
	SMTPPort              int
	SMTPUsername          string
	SMTPPassword          string
	SMTPFromEmail         string
	InviteBaseURL         string
	InviteTTLHours        int
	AdminEmails           []string

	// Legacy single-provider config (used when PROVIDER_ORDER is not set).
	TranslationProviderName  string
	TranslationProviderURL   string
	TranslationProviderKey   string
	TranslationProviderModel string

	// TranslationChainTimeout is the total timeout for the translation chain
	// (across all providers tried sequentially), in seconds.
	TranslationChainTimeout int

	// LogLevel controls verbosity: "debug", "info", "warn", "error"
	LogLevel string

	// Grammar AI analysis legacy config.
	GrammarAPIURL string
	GrammarAPIKey string
	GrammarModel  string

	// Provider chain — ordered list of aliases (e.g. ["primary", "secondary", "local"]).
	// Each alias maps into the Providers map.
	TranslationProviderOrder []string
	GrammarProviderOrder     []string
	Providers                map[string]ProviderDef // key = alias, e.g. "primary", "ollama_local"
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
		SMTPHost:              getEnv("MAILU_SMTP_HOST", ""),
		SMTPPort:              getEnvInt("MAILU_SMTP_PORT", 465),
		SMTPUsername:          getEnv("MAILU_SMTP_USERNAME", ""),
		SMTPPassword:          getEnv("MAILU_SMTP_PASSWORD", ""),
		SMTPFromEmail:         getEnv("MAILU_SMTP_FROM", ""),
		InviteBaseURL:         getEnv("INVITE_BASE_URL", "http://localhost:3000/register"),
		InviteTTLHours:        getEnvInt("INVITE_TTL_HOURS", 168),
		AdminEmails:           splitAndTrim(getEnv("WAITLIST_ADMIN_EMAILS", "")),

		// Legacy single-provider config.
		TranslationProviderName:  getEnv("TRANSLATION_PROVIDER_NAME", string(translation.ProviderOpenCode)),
		TranslationProviderURL:   getEnv("TRANSLATION_PROVIDER_API_URL", ""),
		TranslationProviderKey:   getEnv("TRANSLATION_PROVIDER_API_KEY", ""),
		TranslationProviderModel: getEnv("TRANSLATION_PROVIDER_MODEL", ""),

		LogLevel: getEnv("LOG_LEVEL", "info"),

		GrammarAPIURL: getEnv("GRAMMAR_API_URL", "https://opencode.ai/zen/go/v1"),
		GrammarAPIKey: getEnv("GRAMMAR_API_KEY", ""),
		GrammarModel:  getEnv("GRAMMAR_MODEL", "deepseek-v4-flash"),

		TranslationChainTimeout: getEnvInt("TRANSLATION_CHAIN_TIMEOUT", 120),

		Providers: make(map[string]ProviderDef),
	}
	if len(cfg.AdminEmails) == 0 && cfg.SMTPUsername != "" {
		cfg.AdminEmails = []string{cfg.SMTPUsername}
	}

	// Parse provider chain orders.
	transOrder := getEnv("TRANSLATION_PROVIDER_ORDER", "")
	// GRAMMAR_ANALYSIS_PROVIDER_ORDER is the canonical name; keep the legacy
	// GRAMMAR_PROVIDER_ORDER as a fallback for existing setups.
	grammarOrder := getEnv("GRAMMAR_ANALYSIS_PROVIDER_ORDER", "")
	if grammarOrder == "" {
		grammarOrder = getEnv("GRAMMAR_PROVIDER_ORDER", "")
	}

	if transOrder != "" {
		cfg.TranslationProviderOrder = splitAndTrim(transOrder)
	}
	if grammarOrder != "" {
		cfg.GrammarProviderOrder = splitAndTrim(grammarOrder)
	}

	// Parse individual provider definitions.
	// Format: PROVIDER_<ALIAS>_<KEY>
	// Example: PROVIDER_PRIMARY_TYPE=opencode
	//          PROVIDER_PRIMARY_API_URL=https://opencode.ai/zen/go/v1
	//          PROVIDER_PRIMARY_API_KEY=sk-...
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
	// The caller (main.go) handles this by checking len(TranslationProviderOrder)==0.
	if len(cfg.TranslationProviderOrder) == 0 {
		log.Printf("[Config] TRANSLATION_PROVIDER_ORDER not set, using legacy single-provider config")
	}
	if len(cfg.GrammarProviderOrder) == 0 {
		log.Printf("[Config] GRAMMAR_ANALYSIS_PROVIDER_ORDER not set, using legacy single-provider config")
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

// Warning describes a configuration problem found during the startup check.
// It never contains secret values — only env var names.
type Warning struct {
	Provider string // provider alias, e.g. "openrouter"
	EnvKey   string // the env var that is missing/empty, if applicable
	Message  string // human-readable description
}

// EnvKeyFor builds the canonical env var name for a provider field, e.g.
// EnvKeyFor("openrouter", "API_KEY") == "PROVIDER_OPENROUTER_API_KEY".
func EnvKeyFor(alias, field string) string {
	return fmt.Sprintf("PROVIDER_%s_%s", strings.ToUpper(alias), field)
}

// Validate inspects the loaded provider config and returns a list of warnings
// for missing or invalid configuration (empty API keys, referenced-but-undefined
// providers, missing local fallbacks, ...). It is intended to run at startup;
// a warning does not stop the server — the chain simply skips the broken
// provider at runtime and falls through to the next one.
func (c *Config) Validate() []Warning {
	var warnings []Warning

	// Chain aliases referenced in an order but never defined.
	checkOrder := func(order []string, orderEnvKey string) {
		for _, alias := range order {
			def, ok := c.Providers[alias]
			if !ok {
				warnings = append(warnings, Warning{
					Provider: alias,
					EnvKey:   EnvKeyFor(alias, "TYPE"),
					Message: fmt.Sprintf("%q appears in %s but has no PROVIDER_%s_* configuration — it will be skipped",
						alias, orderEnvKey, strings.ToUpper(alias)),
				})
				continue
			}
			if strings.TrimSpace(def.Type) == "" {
				warnings = append(warnings, Warning{
					Provider: alias,
					EnvKey:   EnvKeyFor(alias, "TYPE"),
					Message:  fmt.Sprintf("provider %q has no TYPE (set %s)", alias, EnvKeyFor(alias, "TYPE")),
				})
			}
		}
	}
	checkOrder(c.TranslationProviderOrder, "TRANSLATION_PROVIDER_ORDER")
	checkOrder(c.GrammarProviderOrder, "GRAMMAR_ANALYSIS_PROVIDER_ORDER")

	// Every defined provider: check required fields per type.
	for alias, def := range c.Providers {
		pType := translation.ProviderType(def.Type)
		if def.Type == "" {
			continue // already warned above
		}
		if translation.NeedsAPIKey(pType) {
			if def.APIKey == "" {
				warnings = append(warnings, Warning{
					Provider: alias,
					EnvKey:   EnvKeyFor(alias, "API_KEY"),
					Message: fmt.Sprintf("provider %q (%s) needs an API key but %s is empty — it will be skipped",
						alias, def.Type, EnvKeyFor(alias, "API_KEY")),
				})
			}
			if def.APIURL == "" {
				warnings = append(warnings, Warning{
					Provider: alias,
					EnvKey:   EnvKeyFor(alias, "API_URL"),
					Message: fmt.Sprintf("provider %q (%s) has no API URL set — a built-in default will be used",
						alias, def.Type),
				})
			}
		}
		if !translation.IsLocalProvider(pType) && def.Model == "" {
			warnings = append(warnings, Warning{
				Provider: alias,
				EnvKey:   EnvKeyFor(alias, "MODEL"),
				Message: fmt.Sprintf("provider %q (%s) has no MODEL set — a built-in default will be used",
					alias, def.Type),
			})
		}
	}

	// Ensure each chain keeps an offline/local fallback so translation keeps
	// working when every cloud key is missing or every cloud quota is exhausted.
	if len(c.TranslationProviderOrder) > 0 && !orderHasLocal(c.TranslationProviderOrder, c.Providers) {
		warnings = append(warnings, Warning{
			Provider: "chain",
			EnvKey:   "TRANSLATION_PROVIDER_ORDER",
			Message:  "translation chain has no offline/local provider (ollama, libretranslate, translator-engine) as a guaranteed fallback",
		})
	}
	if len(c.GrammarProviderOrder) > 0 && !orderHasLocal(c.GrammarProviderOrder, c.Providers) {
		warnings = append(warnings, Warning{
			Provider: "chain",
			EnvKey:   "GRAMMAR_ANALYSIS_PROVIDER_ORDER",
			Message:  "grammar chain has no offline/local provider (ollama) as a guaranteed fallback",
		})
	}

	// Legacy single-provider mode (used only when no chain order is set).
	if len(c.TranslationProviderOrder) == 0 &&
		translation.NeedsAPIKey(translation.ProviderType(c.TranslationProviderName)) &&
		c.TranslationProviderKey == "" {
		warnings = append(warnings, Warning{
			Provider: c.TranslationProviderName,
			EnvKey:   "TRANSLATION_PROVIDER_API_KEY",
			Message:  fmt.Sprintf("legacy translation provider %q needs an API key but TRANSLATION_PROVIDER_API_KEY is empty", c.TranslationProviderName),
		})
	}

	return warnings
}

func orderHasLocal(order []string, providers map[string]ProviderDef) bool {
	for _, alias := range order {
		def, ok := providers[alias]
		if !ok {
			continue
		}
		if translation.IsLocalProvider(translation.ProviderType(def.Type)) {
			return true
		}
	}
	return false
}
