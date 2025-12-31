package config

import "os"

type Config struct {
	Environment           string
	DatabaseURL           string
	RedisURL              string
	JWTSecret             string
	GoogleTranslateAPIKey string
	Port                  string
}

func Load() *Config {
	return &Config{
		Environment:           getEnv("ENVIRONMENT", "development"),
		DatabaseURL:           getEnv("DATABASE_URL", "postgres://messenger:password@localhost:5432/messenger_dev?sslmode=disable"),
		RedisURL:              getEnv("REDIS_URL", "localhost:6379"),
		JWTSecret:             getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		GoogleTranslateAPIKey: getEnv("GOOGLE_TRANSLATE_API_KEY", ""),
		Port:                  getEnv("PORT", "8080"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
