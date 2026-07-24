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

	// Translation provider configuration (Phase 4 unified provider abstraction).
	TranslationProvider   string
	OpenAIBaseURL         string
	OpenAIAPIKey          string
	OpenAIModel           string
	OllamaURL             string
	OllamaModel           string
	TranslatorEngineURL   string
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

		// Translation provider defaults (Phase 4).
		TranslationProvider: getEnv("TRANSLATION_PROVIDER", string(translation.ProviderOpenAI)),
		OpenAIBaseURL:       getEnv("OPENAI_BASE_URL", "https://api.opencode.com/v1"),
		OpenAIAPIKey:        getEnv("OPENAI_API_KEY", ""),
		OpenAIModel:         getEnv("OPENAI_MODEL", "gpt-4o-mini"),
		OllamaURL:           getEnv("OLLAMA_URL", "http://localhost:11434"),
		OllamaModel:         getEnv("OLLAMA_MODEL", "qwen2.5:3b"),
		TranslatorEngineURL: getEnv("TRANSLATOR_ENGINE_URL", "http://translator-engine:5000"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}