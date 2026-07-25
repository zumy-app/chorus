package config

import (
	"github.com/chorus/messenger/pkg/translation"
	"os"
)

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

	// Translation provider configuration (unified provider abstraction).
	// See backend/.env.example for full documentation.
	TranslationProviderName string
	TranslationProviderURL  string
	TranslationProviderKey  string
	TranslationProviderModel string

	// Grammar AI analysis configuration (OpenAI-compatible API).
	GrammarAPIURL string
	GrammarAPIKey string
	GrammarModel  string
}

func Load() *Config {
	return &Config{
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

		// Translation provider defaults (unified).
		// Supported provider names: opencode, openai, deepseek, ollama, translator-engine
		TranslationProviderName:  getEnv("TRANSLATION_PROVIDER_NAME", string(translation.ProviderOpenCode)),
		TranslationProviderURL:   getEnv("TRANSLATION_PROVIDER_API_URL", ""),
		TranslationProviderKey:   getEnv("TRANSLATION_PROVIDER_API_KEY", ""),
		TranslationProviderModel: getEnv("TRANSLATION_PROVIDER_MODEL", ""),

		// Grammar API defaults to the OpenCode Go cloud API with deepseek-v4-flash.
		GrammarAPIURL: getEnv("GRAMMAR_API_URL", "https://opencode.ai/zen/go/v1"),
		GrammarAPIKey: getEnv("GRAMMAR_API_KEY", ""),
		GrammarModel:  getEnv("GRAMMAR_MODEL", "deepseek-v4-flash"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}