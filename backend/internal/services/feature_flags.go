package services

import (
	"crypto/sha256"
	"encoding/binary"
	"os"
	"strconv"
	"strings"
)

type FeatureFlagService struct {
	envOverrides map[string]bool
}

func NewFeatureFlagService() *FeatureFlagService {
	return &FeatureFlagService{
		envOverrides: make(map[string]bool),
	}
}

// IsEnabled determines if a feature flag is enabled for a given user.
// It supports:
// 1. Exact environment variable overrides (e.g. FEATURE_GRAMMAR_V3=true/false)
// 2. Percentage canary rollouts (e.g. CANARY_PERCENTAGE_GRAMMAR_V3=10)
func (s *FeatureFlagService) IsEnabled(flagName, userID string) bool {
	envKey := "FEATURE_" + strings.ToUpper(strings.ReplaceAll(flagName, "-", "_"))
	if val := os.Getenv(envKey); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			return enabled
		}
	}

	canaryKey := "CANARY_PERCENTAGE_" + strings.ToUpper(strings.ReplaceAll(flagName, "-", "_"))
	if canaryStr := os.Getenv(canaryKey); canaryStr != "" {
		if pct, err := strconv.Atoi(canaryStr); err == nil && pct > 0 {
			if pct >= 100 {
				return true
			}
			if userID == "" {
				return false
			}
			// Deterministic hash of userID + flagName to bucket 0..99
			h := sha256.Sum256([]byte(userID + ":" + flagName))
			hashInt := binary.BigEndian.Uint32(h[:4])
			bucket := int(hashInt % 100)
			return bucket < pct
		}
	}

	// Default fallback values
	switch flagName {
	case "learning_v3_engine":
		return true
	case "redis_session_cache":
		return true
	default:
		return false
	}
}
