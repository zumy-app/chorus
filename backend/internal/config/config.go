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
//
// Env keys (canonical):
//
//	PROVIDER_<NAME>_URL     -> URL          (e.g. PROVIDER_OPENROUTER_URL)
//	PROVIDER_<NAME>_KEY     -> Key          (e.g. PROVIDER_OPENROUTER_KEY)
//	PROVIDER_<NAME>_TIMEOUT -> Timeout
//	MODEL_TRANSLATION_<NAME> -> ModelTranslation
//	MODEL_GRAMMAR_<NAME>     -> ModelGrammar
//
// The provider type is derived from the canonical name (openrouter, opencode,
// nvidia, ollama, libretranslate, ...), so the order keys in a chain must use
// those names.
type ProviderDef struct {
	Name             string // canonical name, e.g. "openrouter"
	URL              string // base URL
	Key              string // API key
	Timeout          int    // seconds; 0 = use provider default
	ModelTranslation string // model used when this provider handles a translation
	ModelGrammar     string // model used when this provider handles grammar analysis
}

// TranslationModel returns the model to use for translation jobs.
func (d ProviderDef) TranslationModel() string {
	return d.ModelTranslation
}

// GrammarModel returns the model to use for grammar analysis.
func (d ProviderDef) GrammarModel() string {
	return d.ModelGrammar
}

// EffectiveType returns the provider type, deriving it from the provider name.
func (d ProviderDef) EffectiveType() string {
	if t, ok := providerTypeByName[d.Name]; ok {
		return string(t)
	}
	return ""
}

// providerTypeByName maps canonical provider names to their provider type.
var providerTypeByName = map[string]translation.ProviderType{
	"openrouter":        translation.ProviderOpenRouter,
	"opencode":          translation.ProviderOpenCode,
	"openai":            translation.ProviderOpenAI,
	"deepseek":          translation.ProviderDeepSeek,
	"nvidia":            translation.ProviderNvidia,
	"ollama":            translation.ProviderOllama,
	"libretranslate":    translation.ProviderLibreTranslate,
	"translator-engine": translation.ProviderEngine,
}

// Config holds all application configuration.
type Config struct {
	Environment           string
	DatabaseURL           string
	RedisURL              string
	JWTSecret             string
	GoogleTranslateAPIKey string
	Port                  string
	SMTPHost              string
	SMTPPort              int
	SMTPUsername          string
	SMTPPassword          string
	SMTPFromEmail         string
	SMTPFromName          string
	InviteBaseURL         string
	PasswordResetBaseURL  string
	InviteTTLHours        int
	AdminEmails           []string

	// TranslationChainTimeout is the total timeout for the translation chain
	// (across all providers tried sequentially), in seconds.
	TranslationChainTimeout int

	// LogLevel controls verbosity: "debug", "info", "warn", "error"
	LogLevel string

	// Provider chain — ordered list of aliases (e.g. ["openrouter", "ollama"]).
	// Each alias maps into the Providers map.
	TranslationProviderOrder []string
	GrammarProviderOrder     []string
	Providers                map[string]ProviderDef // key = alias, e.g. "primary", "ollama_local"
}

// Load reads configuration from environment variables.
func Load() *Config {
	smtpUsername := getEnv("MAILU_SMTP_USERNAME", "")
	smtpFrom := getEnv("MAILU_SMTP_FROM", "")
	if smtpFrom == "" {
		smtpFrom = smtpUsername
	}
	cfg := &Config{
		Environment:           getEnv("ENVIRONMENT", "development"),
		DatabaseURL:           getEnv("DATABASE_URL", "postgres://messenger:password@localhost:5432/messenger_dev?sslmode=disable"),
		RedisURL:              getEnv("REDIS_URL", "localhost:6379"),
		JWTSecret:             getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		GoogleTranslateAPIKey: getEnv("GOOGLE_TRANSLATE_API_KEY", ""),
		Port:                  getEnv("PORT", "8080"),
		SMTPHost:              getEnv("MAILU_SMTP_HOST", ""),
		SMTPPort:              getEnvInt("MAILU_SMTP_PORT", 465),
		SMTPUsername:          smtpUsername,
		SMTPPassword:          getEnv("MAILU_SMTP_PASSWORD", ""),
		SMTPFromEmail:         smtpFrom,
		SMTPFromName:          getEnv("MAILU_SMTP_FROM_NAME", "Chorus"),
		InviteBaseURL:         getEnv("INVITE_BASE_URL", "http://localhost:3000/register"),
		PasswordResetBaseURL:  getEnv("PASSWORD_RESET_BASE_URL", "http://localhost:3000/reset-password"),
		InviteTTLHours:        getEnvInt("INVITE_TTL_HOURS", 168),
		AdminEmails:           splitAndTrim(getEnv("WAITLIST_ADMIN_EMAILS", "")),

		LogLevel: getEnv("LOG_LEVEL", "info"),

		TranslationChainTimeout: getEnvInt("TRANSLATION_CHAIN_TIMEOUT", 120),

		Providers: make(map[string]ProviderDef),
	}
	if len(cfg.AdminEmails) == 0 && cfg.SMTPUsername != "" {
		cfg.AdminEmails = []string{cfg.SMTPUsername}
	}

	// Parse provider chain orders.
	transOrder := getEnv("TRANSLATION_FALLBACK_ORDER", "")
	grammarOrder := getEnv("GRAMMAR_FALLBACK_ORDER", "")

	if transOrder != "" {
		cfg.TranslationProviderOrder = splitAndTrim(transOrder)
	}
	if grammarOrder != "" {
		cfg.GrammarProviderOrder = splitAndTrim(grammarOrder)
	}

	// Parse individual provider definitions: PROVIDER_<NAME>_URL / _KEY / _TIMEOUT,
// plus MODEL_TRANSLATION_<NAME> and MODEL_GRAMMAR_<NAME> below.
	knownFields := []struct {
		suffix string
		field  string
	}{
		{"_TIMEOUT", "TIMEOUT"},
		{"_KEY", "KEY"},
		{"_URL", "URL"},
	}
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, "PROVIDER_") {
			continue
		}
		eqIdx := strings.Index(env, "=")
		if eqIdx == -1 {
			continue
		}
		key := env[:eqIdx]
		value := env[eqIdx+1:]
		rest := key[len("PROVIDER_"):]

		var name, field string
		for _, entry := range knownFields {
			if strings.HasSuffix(rest, entry.suffix) {
				name = strings.ToLower(rest[:len(rest)-len(entry.suffix)])
				field = entry.field
				break
			}
		}
		if name == "" {
			continue
		}

		def := cfg.Providers[name]
		switch field {
		case "URL":
			def.URL = value
		case "KEY":
			def.Key = value
		case "TIMEOUT":
			if v, err := strconv.Atoi(value); err == nil {
				def.Timeout = v
			}
		}
		def.Name = name
		cfg.Providers[name] = def
	}

	// Task-specific models: MODEL_TRANSLATION_<NAME> / MODEL_GRAMMAR_<NAME>.
	for _, env := range os.Environ() {
		var prefix, field string
		switch {
		case strings.HasPrefix(env, "MODEL_TRANSLATION_"):
			prefix, field = "MODEL_TRANSLATION_", "ModelTranslation"
		case strings.HasPrefix(env, "MODEL_GRAMMAR_"):
			prefix, field = "MODEL_GRAMMAR_", "ModelGrammar"
		default:
			continue
		}
		eqIdx := strings.Index(env, "=")
		if eqIdx == -1 {
			continue
		}
		name := strings.ToLower(env[len(prefix):eqIdx])
		if name == "" {
			continue
		}
		def := cfg.Providers[name]
		def.Name = name
		switch field {
		case "ModelTranslation":
			def.ModelTranslation = env[eqIdx+1:]
		case "ModelGrammar":
			def.ModelGrammar = env[eqIdx+1:]
		}
		cfg.Providers[name] = def
	}

	if len(cfg.TranslationProviderOrder) == 0 {
		log.Printf("[Config] TRANSLATION_FALLBACK_ORDER not set — no translation providers will be configured")
	}
	if len(cfg.GrammarProviderOrder) == 0 {
		log.Printf("[Config] GRAMMAR_FALLBACK_ORDER not set — grammar analysis will use the regex fallback")
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
	Provider string // provider name, e.g. "openrouter"
	EnvKey   string // the env var that is missing/empty, if applicable
	Message  string // human-readable description
}

// EnvKeyFor builds the canonical env var name for a provider field, e.g.
// EnvKeyFor("openrouter", "KEY") == "PROVIDER_OPENROUTER_KEY".
func EnvKeyFor(alias, field string) string {
	return fmt.Sprintf("PROVIDER_%s_%s", strings.ToUpper(alias), field)
}

// ModelTranslationKeyFor returns the env var for a provider's translation model,
// e.g. ModelTranslationKeyFor("openrouter") == "MODEL_TRANSLATION_OPENROUTER".
func ModelTranslationKeyFor(alias string) string {
	return fmt.Sprintf("MODEL_TRANSLATION_%s", strings.ToUpper(alias))
}

// ModelGrammarKeyFor returns the env var for a provider's grammar model,
// e.g. ModelGrammarKeyFor("openrouter") == "MODEL_GRAMMAR_OPENROUTER".
func ModelGrammarKeyFor(alias string) string {
	return fmt.Sprintf("MODEL_GRAMMAR_%s", strings.ToUpper(alias))
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
					EnvKey:   EnvKeyFor(alias, "KEY"),
					Message: fmt.Sprintf("%q appears in %s but has no PROVIDER_%s_* configuration — it will be skipped",
						alias, orderEnvKey, strings.ToUpper(alias)),
				})
				continue
			}
			if def.EffectiveType() == "" {
				warnings = append(warnings, Warning{
					Provider: alias,
					EnvKey:   EnvKeyFor(alias, "TYPE"),
					Message: fmt.Sprintf("provider %q is not a known provider name (set %s or use a supported name)",
						alias, EnvKeyFor(alias, "TYPE")),
				})
			}
		}
	}
	checkOrder(c.TranslationProviderOrder, "TRANSLATION_FALLBACK_ORDER")
	checkOrder(c.GrammarProviderOrder, "GRAMMAR_FALLBACK_ORDER")

	// Every defined provider: check required fields per type.
	for alias, def := range c.Providers {
		pType := translation.ProviderType(def.EffectiveType())
		if pType == "" {
			continue // already warned above
		}
		if translation.NeedsAPIKey(pType) {
			if def.Key == "" {
				warnings = append(warnings, Warning{
					Provider: alias,
					EnvKey:   EnvKeyFor(alias, "KEY"),
					Message: fmt.Sprintf("provider %q (%s) needs an API key but %s is empty — it will be skipped",
						alias, def.EffectiveType(), EnvKeyFor(alias, "KEY")),
				})
			}
			if def.URL == "" {
				warnings = append(warnings, Warning{
					Provider: alias,
					EnvKey:   EnvKeyFor(alias, "URL"),
					Message: fmt.Sprintf("provider %q (%s) has no %s — a built-in default will be used",
						alias, def.EffectiveType(), EnvKeyFor(alias, "URL")),
				})
			}
		}
		if !translation.IsLocalProvider(pType) {
			models := []struct {
				key   string
				value string
			}{
				{ModelTranslationKeyFor(alias), def.ModelTranslation},
				{ModelGrammarKeyFor(alias), def.ModelGrammar},
			}
			missing := []string{}
			for _, m := range models {
				if m.value == "" {
					missing = append(missing, m.key)
				}
			}
			if len(missing) > 0 {
				warnings = append(warnings, Warning{
					Provider: alias,
					EnvKey:   strings.Join(missing, " / "),
					Message: fmt.Sprintf("provider %q (%s) has no task model set (%s) — built-in defaults will be used",
						alias, def.EffectiveType(), strings.Join(missing, ", ")),
				})
			}
		}
	}

	// Ensure each chain keeps an offline/local fallback so translation keeps
	// working when every cloud key is missing or every cloud quota is exhausted.
	if len(c.TranslationProviderOrder) > 0 && !orderHasLocal(c.TranslationProviderOrder, c.Providers) {
		warnings = append(warnings, Warning{
			Provider: "chain",
			EnvKey:   "TRANSLATION_FALLBACK_ORDER",
			Message:  "translation chain has no offline/local provider (ollama, libretranslate, translator-engine) as a guaranteed fallback",
		})
	}
	if len(c.GrammarProviderOrder) > 0 && !orderHasLocal(c.GrammarProviderOrder, c.Providers) {
		warnings = append(warnings, Warning{
			Provider: "chain",
			EnvKey:   "GRAMMAR_FALLBACK_ORDER",
			Message:  "grammar chain has no offline/local provider (ollama) as a guaranteed fallback",
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
		if translation.IsLocalProvider(translation.ProviderType(def.EffectiveType())) {
			return true
		}
	}
	return false
}
