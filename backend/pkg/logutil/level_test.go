package logutil

import (
	"testing"
	"time"
)

func TestSetLevelFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"DEBUG", LevelDebug},
		{"debug", LevelDebug},
		{"INFO", LevelInfo},
		{"info", LevelInfo},
		{"WARN", LevelWarn},
		{"warning", LevelWarn},
		{"ERROR", LevelError},
		{"error", LevelError},
		{"invalid", LevelInfo}, // Default level should remain unchanged or unaffected
	}

	for _, tt := range tests {
		SetLevelFromString(tt.input)
		if tt.input != "invalid" && currentLevel != tt.expected {
			t.Errorf("SetLevelFromString(%q) = %v; expected %v", tt.input, currentLevel, tt.expected)
		}
	}
}

func TestLoggingFunctions(t *testing.T) {
	SetLevelFromString("DEBUG")

	// Ensure logging functions execute without panicking
	t.Run("Debugf", func(t *testing.T) {
		Debugf("test debug %s", "arg")
	})

	t.Run("Infof", func(t *testing.T) {
		Infof("test info %d", 123)
	})

	t.Run("Warnf", func(t *testing.T) {
		Warnf("test warn")
	})

	t.Run("Errorf", func(t *testing.T) {
		Errorf("test error")
	})

	t.Run("Duration", func(t *testing.T) {
		start := time.Now()
		time.Sleep(1 * time.Millisecond)
		Duration("test-op", start)
		Duration("test-op-with-args", start, "context info")
	})
}
