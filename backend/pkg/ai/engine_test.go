package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewEngine(t *testing.T) {
	e := NewEngine("", "")
	if e.BaseURL != "http://localhost:11434" {
		t.Errorf("expected default BaseURL http://localhost:11434, got %s", e.BaseURL)
	}
	if e.Model != "qwen2.5:1.5b-instruct" {
		t.Errorf("expected default Model qwen2.5:1.5b-instruct, got %s", e.Model)
	}

	custom := NewEngine("http://custom-ollama:11434/", "my-model")
	if custom.BaseURL != "http://custom-ollama:11434" {
		t.Errorf("expected trailing slash trimmed BaseURL, got %s", custom.BaseURL)
	}
	if custom.Model != "my-model" {
		t.Errorf("expected model my-model, got %s", custom.Model)
	}
}

func TestExecuteFastTranslation_EmptyText(t *testing.T) {
	e := NewEngine("http://localhost:11434", "test")
	res, err := e.ExecuteFastTranslation("  ", "en", "es")
	if err != nil {
		t.Fatalf("unexpected error on empty text: %v", err)
	}
	if res != "" {
		t.Errorf("expected empty result, got %q", res)
	}
}

func TestExecuteFastTranslation_Success(t *testing.T) {
	ts := httptest.Server{} // placeholder
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/chat" {
			t.Errorf("expected /api/chat path, got %s", r.URL.Path)
		}

		resp := map[string]any{
			"message": map[string]string{
				"role":    "assistant",
				"content": "Hola mundo",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()
	ts = *server

	e := NewEngine(ts.URL, "test-model")
	translated, err := e.ExecuteFastTranslation("Hello world", "en", "es")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if translated != "Hola mundo" {
		t.Errorf("expected 'Hola mundo', got %q", translated)
	}
}

func TestExecuteFastTranslation_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Ollama Service Unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	e := NewEngine(server.URL, "test-model")
	_, err := e.ExecuteFastTranslation("Hello", "en", "es")
	if err == nil {
		t.Fatal("expected error on 503 response, got nil")
	}
}

func TestProcessTutorAnalysis_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reportJSON := `{"language":"en","corrections":[{"original":"I goes","corrected":"I go","explanation":"Subject-verb agreement"}],"vocabulary":[],"notes":"Good attempt"}`
		resp := map[string]any{
			"message": map[string]string{
				"role":    "assistant",
				"content": reportJSON,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	e := NewEngine(server.URL, "test-model")
	report, err := e.ProcessTutorAnalysis("I goes to store", "en", "es")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Language != "en" {
		t.Errorf("expected language 'en', got %s", report.Language)
	}
	if len(report.Corrections) != 1 {
		t.Fatalf("expected 1 correction, got %d", len(report.Corrections))
	}
	if report.Corrections[0].Corrected != "I go" {
		t.Errorf("expected corrected text 'I go', got %s", report.Corrections[0].Corrected)
	}
}

func TestProcessTutorAnalysis_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"message": map[string]string{
				"role":    "assistant",
				"content": "Not valid json response",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	e := NewEngine(server.URL, "test-model")
	_, err := e.ProcessTutorAnalysis("Some text", "en", "es")
	if err == nil {
		t.Fatal("expected parse error for non-JSON content, got nil")
	}
}
