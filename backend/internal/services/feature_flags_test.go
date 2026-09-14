package services

import (
	"os"
	"testing"
)

func TestFeatureFlagService_DefaultFlags(t *testing.T) {
	svc := NewFeatureFlagService()

	if !svc.IsEnabled("learning_v3_engine", "user-123") {
		t.Errorf("expected learning_v3_engine to be true by default")
	}
	if !svc.IsEnabled("redis_session_cache", "user-123") {
		t.Errorf("expected redis_session_cache to be true by default")
	}
	if svc.IsEnabled("non_existent_flag", "user-123") {
		t.Errorf("expected non_existent_flag to be false by default")
	}
}

func TestFeatureFlagService_EnvOverride(t *testing.T) {
	svc := NewFeatureFlagService()

	os.Setenv("FEATURE_CUSTOM_CANARY", "true")
	defer os.Unsetenv("FEATURE_CUSTOM_CANARY")

	if !svc.IsEnabled("custom_canary", "user-456") {
		t.Errorf("expected custom_canary to be enabled via env override")
	}

	os.Setenv("FEATURE_LEARNING_V3_ENGINE", "false")
	defer os.Unsetenv("FEATURE_LEARNING_V3_ENGINE")

	if svc.IsEnabled("learning_v3_engine", "user-456") {
		t.Errorf("expected learning_v3_engine to be disabled via env override")
	}
}

func TestFeatureFlagService_CanaryPercentage(t *testing.T) {
	svc := NewFeatureFlagService()

	os.Setenv("CANARY_PERCENTAGE_EXPERIMENTAL_FLOW", "50")
	defer os.Unsetenv("CANARY_PERCENTAGE_EXPERIMENTAL_FLOW")

	// Same user should consistently evaluate to the same value
	res1 := svc.IsEnabled("experimental_flow", "alice-100")
	res2 := svc.IsEnabled("experimental_flow", "alice-100")
	if res1 != res2 {
		t.Errorf("expected deterministic canary evaluation for same user")
	}
}
