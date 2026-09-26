package services

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseOpenAISSE_DeltasInOrder(t *testing.T) {
	body := ": heartbeat\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n\ndata: [DONE]\n"
	var got []string
	if err := parseOpenAISSE(strings.NewReader(body), func(d string) { got = append(got, d) }); err != nil {
		t.Fatalf("parseOpenAISSE: %v", err)
	}
	if strings.Join(got, "") != "Hello world" {
		t.Fatalf("expected 'Hello world', got %q", got)
	}
}

func TestParseOpenAISSE_MissingDoneStillReturns(t *testing.T) {
	body := "data: {\"choices\":[{\"delta\":{\"content\":\"Hi\"}}]}\n"
	var got []string
	if err := parseOpenAISSE(strings.NewReader(body), func(d string) { got = append(got, d) }); err != nil {
		t.Fatalf("parseOpenAISSE: %v", err)
	}
	if strings.Join(got, "") != "Hi" {
		t.Fatalf("expected 'Hi', got %q", got)
	}
}

func TestParseOpenAISSE_BadChunkErrors(t *testing.T) {
	body := "data: {not-json}\n"
	if err := parseOpenAISSE(strings.NewReader(body), func(d string) {}); err == nil {
		t.Fatalf("expected decode error")
	}
}

func TestParseOllamaNDJSON_DeltasUntilDone(t *testing.T) {
	body := "{\"message\":{\"content\":\"Hola\"},\"done\":false}\n{\"message\":{\"content\":\" amigo\"},\"done\":true}\n"
	var got []string
	if err := parseOllamaNDJSON(strings.NewReader(body), func(d string) { got = append(got, d) }); err != nil {
		t.Fatalf("parseOllamaNDJSON: %v", err)
	}
	if strings.Join(got, "") != "Hola amigo" {
		t.Fatalf("expected 'Hola amigo', got %q", got)
	}
}

func TestEndpointCallStream_OpenAICompatible(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Error("no flusher")
			return
		}
		for _, tok := range []string{"Hel", "lo!"} {
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q}}]}\n\n", tok)
			flusher.Flush()
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	ep := NewGrammarEndpoint("test-openai", "openai", srv.URL, "key", "model", 10)
	var deltas []string
	full, err := ep.callStream(t.Context(), "prompt", "English", func(d string) { deltas = append(deltas, d) })
	if err != nil {
		t.Fatalf("callStream: %v", err)
	}
	if full != "Hello!" {
		t.Fatalf("expected 'Hello!', got %q", full)
	}
	if strings.Join(deltas, "") != "Hello!" {
		t.Fatalf("deltas do not reassemble: %q", deltas)
	}
}

func TestEndpointCallStream_OllamaNDJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		fmt.Fprint(w, "{\"message\":{\"content\":\"Buen\"},\"done\":false}\n{\"message\":{\"content\":\"os días\"},\"done\":true}\n")
	}))
	defer srv.Close()

	ep := NewGrammarEndpoint("test-ollama", "ollama", srv.URL, "", "model", 10)
	full, err := ep.callStream(t.Context(), "prompt", "Spanish", nil)
	if err != nil {
		t.Fatalf("callStream: %v", err)
	}
	if full != "Buenos días" {
		t.Fatalf("expected 'Buenos días', got %q", full)
	}
}

func TestEndpointCallStream_Non200Errors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":"bad key"}`)
	}))
	defer srv.Close()

	ep := NewGrammarEndpoint("test-bad", "openai", srv.URL, "bad", "model", 10)
	if _, err := ep.callStream(t.Context(), "prompt", "English", nil); err == nil {
		t.Fatalf("expected error for 401")
	}
}

func TestStripCodeFences(t *testing.T) {
	if got := stripCodeFences("```\nhello\n```"); got != "hello" {
		t.Fatalf("expected 'hello', got %q", got)
	}
	if got := stripCodeFences("plain"); got != "plain" {
		t.Fatalf("expected 'plain', got %q", got)
	}
}

func TestSparkyPrompt_ContainsQueryAndText(t *testing.T) {
	p := sparkyPrompt("Spanish", "English", "Hola", "what does this mean?")
	for _, want := range []string{"Hola", "what does this mean?", "Spanish", "English"} {
		if !strings.Contains(p, want) {
			t.Fatalf("prompt missing %q", want)
		}
	}
}
