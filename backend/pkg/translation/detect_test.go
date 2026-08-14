package translation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestNormalizeDetectedCode(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"ES", "es"},
		{"\"es\"", "es"},
		{"`es`", "es"},
		{"es\n", "es"},
		{"es, it's Spanish", "es"},
		{"en ", "en"},
		{"spanish", ""},
		{"english", ""},
		{"x", ""},
		{"", ""},
		{"es-MX", "es"},
		{"  fr  ", "fr"},
	}
	for _, c := range cases {
		if got := normalizeDetectedCode(c.in); got != c.want {
			t.Errorf("normalizeDetectedCode(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestDetectLanguageFromLLM(t *testing.T) {
	var called atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Add(1)
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("authorization header = %q, want Bearer test-key", got)
		}
		var req detectChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if req.Model == "" {
			t.Error("expected a model in the request")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"es"}}]}`))
	}))
	defer srv.Close()

	code, err := detectLanguageFromLLM(context.Background(), srv.Client(),
		srv.URL+"/chat/completions", "test-key", "model-x",
		detectLanguageSystemPrompt, detectLanguageUserPrompt("Buenos días"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != "es" {
		t.Fatalf("detected code = %q, want %q", code, "es")
	}
	if called.Load() == 0 {
		t.Fatal("expected the detect endpoint to be hit")
	}
}

func TestDetectLanguageFromLLM_NonCodeReply(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"I think this is Spanish"}}]}`))
	}))
	defer srv.Close()

	if _, err := detectLanguageFromLLM(context.Background(), srv.Client(),
		srv.URL+"/chat/completions", "", "model-x",
		detectLanguageSystemPrompt, detectLanguageUserPrompt("Buenos días")); err == nil {
		t.Fatal("expected an error for a non-code reply")
	}
}

func TestChainProvider_DetectRoutesToFirstDetector(t *testing.T) {
	good := &stubDetector{code: "fr", err: nil}
	chain := NewChainProvider([]Provider{good})
	code, err := chain.DetectLanguage(context.Background(), "Bonjour")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != "fr" {
		t.Fatalf("code = %q, want fr", code)
	}
}

func TestChainProvider_DetectSkipsNonDetectors(t *testing.T) {
	chain := NewChainProvider([]Provider{&stubNonDetector{}})
	if _, err := chain.DetectLanguage(context.Background(), "hola"); err == nil {
		t.Fatal("expected an error when no provider supports detection")
	}
}

type stubDetector struct {
	code string
	err  error
}

func (s *stubDetector) Name() string { return "stub" }

func (s *stubDetector) Translate(ctx context.Context, req TranslateRequest) (TranslateResponse, error) {
	return TranslateResponse{}, nil
}

func (s *stubDetector) DetectLanguage(ctx context.Context, text string) (string, error) {
	return s.code, s.err
}

type stubNonDetector struct{}

func (s *stubNonDetector) Name() string { return "no-detect" }

func (s *stubNonDetector) Translate(ctx context.Context, req TranslateRequest) (TranslateResponse, error) {
	return TranslateResponse{}, nil
}

// Ensure stubDetector satisfies the Provider and LanguageDetector interfaces.
var _ Provider = (*stubDetector)(nil)
var _ LanguageDetector = (*stubDetector)(nil)