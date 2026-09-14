package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type TelemetryEvent struct {
	UserID        string         `json:"userId"`
	SessionID     string         `json:"sessionId,omitempty"`
	EventType     string         `json:"eventType"`
	TargetLang    string         `json:"targetLanguage,omitempty"`
	LatencyMs     int            `json:"latencyMs,omitempty"`
	InteractionUI string         `json:"interactionUi,omitempty"`
	Payload       map[string]any `json:"payload,omitempty"`
	Timestamp     time.Time      `json:"timestamp"`
}

type TelemetryArchiveService struct {
	endpoint   string // MinIO / S3 endpoint (e.g. http://localhost:9000)
	bucket     string
	eventChan  chan TelemetryEvent
	httpClient *http.Client
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	buffer     []TelemetryEvent
	mu         sync.Mutex
}

func NewTelemetryArchiveService(endpoint, bucket string) *TelemetryArchiveService {
	if bucket == "" {
		bucket = "chorus-telemetry"
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &TelemetryArchiveService{
		endpoint:   endpoint,
		bucket:     bucket,
		eventChan:  make(chan TelemetryEvent, 1000), // Non-blocking buffer
		httpClient: &http.Client{Timeout: 5 * time.Second},
		ctx:        ctx,
		cancel:     cancel,
		buffer:     make([]TelemetryEvent, 0, 100),
	}
	return s
}

func (s *TelemetryArchiveService) Start() {
	s.wg.Add(1)
	go s.worker()
	log.Printf("[Telemetry] Telemetry archive worker started (endpoint: %s, bucket: %s)", s.endpoint, s.bucket)
}

func (s *TelemetryArchiveService) Stop() {
	s.cancel()
	s.wg.Wait()
	// Flush remaining buffer
	s.flush()
	log.Println("[Telemetry] Telemetry archive worker stopped")
}

// RecordEvent queues a high-frequency interaction event.
// Completely non-blocking: drops event if buffer is saturated to protect server latency.
func (s *TelemetryArchiveService) RecordEvent(event TelemetryEvent) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	select {
	case s.eventChan <- event:
	default:
		// Queue full: drop without blocking application threads
	}
}

func (s *TelemetryArchiveService) worker() {
	defer s.wg.Done()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case event := <-s.eventChan:
			s.mu.Lock()
			s.buffer = append(s.buffer, event)
			if len(s.buffer) >= 50 {
				s.flushLocked()
			}
			s.mu.Unlock()
		case <-ticker.C:
			s.mu.Lock()
			if len(s.buffer) > 0 {
				s.flushLocked()
			}
			s.mu.Unlock()
		}
	}
}

func (s *TelemetryArchiveService) flush() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flushLocked()
}

func (s *TelemetryArchiveService) flushLocked() {
	if len(s.buffer) == 0 {
		return
	}
	batch := make([]TelemetryEvent, len(s.buffer))
	copy(batch, s.buffer)
	s.buffer = s.buffer[:0]

	if s.endpoint == "" {
		return // MinIO endpoint not configured, silent discard
	}

	go func(events []TelemetryEvent) {
		data, err := json.Marshal(events)
		if err != nil {
			return
		}
		now := time.Now()
		objectKey := fmt.Sprintf("%s/%04d/%02d/%02d/batch-%d.json",
			s.bucket, now.Year(), now.Month(), now.Day(), now.UnixNano())
		url := fmt.Sprintf("%s/%s", s.endpoint, objectKey)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, url, bytes.NewReader(data))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := s.httpClient.Do(req)
		if err == nil && resp != nil {
			_ = resp.Body.Close()
		}
	}(batch)
}
