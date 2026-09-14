package services

import (
	"testing"
	"time"
)

func TestTelemetryArchiveService_BufferAndDropProtection(t *testing.T) {
	// Telemetry service with empty endpoint shouldn't error or block
	svc := NewTelemetryArchiveService("", "test-bucket")
	svc.Start()
	defer svc.Stop()

	// Rapidly record 100 events
	for i := 0; i < 100; i++ {
		svc.RecordEvent(TelemetryEvent{
			UserID:    "test-user",
			EventType: "keystroke_latency",
			LatencyMs: 42,
			Timestamp: time.Now(),
		})
	}

	// Should not panic or hang
}
