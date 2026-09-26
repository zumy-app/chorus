package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestKindStatusMapping(t *testing.T) {
	cases := []struct {
		kind   Kind
		status int
	}{
		{KindValidation, http.StatusBadRequest},
		{KindAuth, http.StatusUnauthorized},
		{KindForbidden, http.StatusForbidden},
		{KindNotFound, http.StatusNotFound},
		{KindConflict, http.StatusConflict},
		{KindRateLimit, http.StatusTooManyRequests},
		{KindTranslation, http.StatusBadGateway},
		{KindTooLarge, http.StatusRequestEntityTooLarge},
		{KindUnavailable, http.StatusServiceUnavailable},
		{KindDelivery, http.StatusBadGateway},
		{KindInternal, http.StatusInternalServerError},
		{"unknown-kind", http.StatusInternalServerError},
	}
	for _, c := range cases {
		if got := c.kind.Status(); got != c.status {
			t.Fatalf("Kind(%q).Status() = %d, want %d", c.kind, got, c.status)
		}
	}
}

func TestErrorConstructorsAndClassification(t *testing.T) {
	validation := ErrValidation("Invalid request")
	if validation.Kind != KindValidation || validation.Message != "Invalid request" {
		t.Fatalf("unexpected validation error: %+v", validation)
	}
	if validation.Retryable {
		t.Fatal("validation errors must not be retryable")
	}
	if !ErrRateLimit("slow down").Retryable {
		t.Fatal("rate-limit errors must be retryable")
	}
	if !ErrTranslation("provider down").Retryable {
		t.Fatal("translation errors must be retryable")
	}
	if !ErrUnavailable("not configured").Retryable {
		t.Fatal("unavailable errors must be retryable")
	}
	if !ErrDelivery("queued").Retryable {
		t.Fatal("delivery errors must be retryable")
	}
	if ErrAuth("bad token").Retryable {
		t.Fatal("auth errors must not be retryable")
	}

	if kind, ok := KindOf(ErrNotFound("nope")); !ok || kind != KindNotFound {
		t.Fatalf("KindOf returned %q,%v; want not_found,true", kind, ok)
	}
	if _, ok := KindOf(errors.New("plain")); ok {
		t.Fatal("KindOf must not classify plain errors")
	}
	if _, ok := KindOf(nil); ok {
		t.Fatal("KindOf(nil) must not be ok")
	}
}

func TestErrorfFormatsAndWrapping(t *testing.T) {
	err := Errorf(KindValidation, "email %q is invalid", "a@b")
	if err.Message != `email "a@b" is invalid` {
		t.Fatalf("Errorf message = %q", err.Message)
	}

	base := ErrTranslation("provider failed")
	causeErr := errors.New("connection refused")
	wrapped := base.WithCause(causeErr)

	// WithCause returns a copy, so identity comparison with the base pointer is
	// not meaningful; classification is done via errors.As / KindOf, and the
	// wrapped *cause* is what errors.Is should reach.
	var api *APIError
	if !errors.As(wrapped, &api) {
		t.Fatal("errors.As must recover the APIError")
	}
	if api.Kind != KindTranslation || api.Message != "provider failed" {
		t.Fatalf("recovered APIError = %+v", api)
	}
	if api.Cause == nil || api.Cause.Error() != "connection refused" {
		t.Fatalf("unexpected cause: %v", api.Cause)
	}
	if !errors.Is(wrapped, causeErr) {
		t.Fatal("errors.Is must reach the wrapped cause")
	}
	if wrapped.Error() != "provider failed" {
		t.Fatalf("Error() should return client message, got %q", wrapped.Error())
	}
}

func TestFromErrNormalizes(t *testing.T) {
	if got := FromErr(nil); got.Kind != KindInternal || !got.Retryable {
		t.Fatalf("FromErr(nil) = %+v", got)
	}

	typed := ErrConflict("already registered")
	if got := FromErr(typed); got != typed {
		t.Fatalf("FromErr should pass through APIError unchanged: %+v", got)
	}

	plain := FromErr(errors.New("boom"))
	if plain.Kind != KindInternal || plain.Retryable != true {
		t.Fatalf("FromErr(plain) = %+v", plain)
	}
	if plain.Cause == nil || plain.Cause.Error() != "boom" {
		t.Fatalf("plain error cause not retained: %v", plain.Cause)
	}
	if plain.Message != "Something went wrong" {
		t.Fatalf("plain error must not leak message, got %q", plain.Message)
	}
}

func TestStatusForAndIsRetryable(t *testing.T) {
	if StatusFor(ErrValidation("x")) != http.StatusBadRequest {
		t.Fatal("StatusFor validation should be 400")
	}
	if StatusFor(ErrTranslation("x")) != http.StatusBadGateway {
		t.Fatal("StatusFor translation should be 502")
	}
	if StatusFor(errors.New("plain")) != http.StatusInternalServerError {
		t.Fatal("StatusFor plain should be 500")
	}
	if IsRetryable(ErrRateLimit("x")) != true {
		t.Fatal("rate-limit should be retryable")
	}
	if IsRetryable(ErrAuth("x")) != false {
		t.Fatal("auth should not be retryable")
	}
	if IsRetryable(errors.New("plain")) != false {
		t.Fatal("plain errors should not be retryable")
	}
}

func TestWriteErrorEnvelopeAndStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name          string
		err           error
		status        int
		wantKind      string
		wantMessage   string
		wantRetryable bool
	}{
		{"validation", ErrValidation("Invalid request"), 400, "validation", "Invalid request", false},
		{"auth", ErrAuth("Invalid credentials"), 401, "auth", "Invalid credentials", false},
		{"rate-limit", ErrRateLimit("Too many requests"), 429, "rate_limit", "Too many requests", true},
		{"translation", ErrTranslation("Translation failed"), 502, "translation", "Translation failed", true},
		{"unavailable", ErrUnavailable("Not configured"), 503, "unavailable", "Not configured", true},
		{"delivery", ErrDelivery("Email queued"), 502, "delivery", "Email queued", true},
		{"not-found", ErrNotFound("Chat not found"), 404, "not_found", "Chat not found", false},
		{"internal-plain", errors.New("db exploded"), 500, "internal", "Something went wrong", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/test", func(ctx *gin.Context) { WriteError(ctx, c.err) })
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			router.ServeHTTP(w, req)

			if w.Code != c.status {
				t.Fatalf("status = %d, want %d (body %s)", w.Code, c.status, w.Body.String())
			}
			resp := struct {
				Error struct {
					Kind      string `json:"kind"`
					Message   string `json:"message"`
					Retryable bool   `json:"retryable"`
				} `json:"error"`
			}{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode failed: %v (body %s)", err, w.Body.String())
			}
			if resp.Error.Kind != c.wantKind {
				t.Fatalf("kind = %q, want %q", resp.Error.Kind, c.wantKind)
			}
			if resp.Error.Message != c.wantMessage {
				t.Fatalf("message = %q, want %q", resp.Error.Message, c.wantMessage)
			}
			if resp.Error.Retryable != c.wantRetryable {
				t.Fatalf("retryable = %v, want %v", resp.Error.Retryable, c.wantRetryable)
			}
		})
	}
}

func TestRecoveryMiddlewareEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Recovery())
	router.GET("/boom", func(ctx *gin.Context) { panic("kaboom") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	resp := struct {
		Error struct {
			Kind      string `json:"kind"`
			Message   string `json:"message"`
			Retryable bool   `json:"retryable"`
		} `json:"error"`
	}{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode failed: %v (body %s)", err, w.Body.String())
	}
	if resp.Error.Kind != "internal" || resp.Error.Message != "Something went wrong" || !resp.Error.Retryable {
		t.Fatalf("unexpected envelope: %+v", resp.Error)
	}
}

func TestWithRetryableOverride(t *testing.T) {
	base := ErrAuth("expired")
	if base.Retryable {
		t.Fatal("auth should default to non-retryable")
	}
	overridden := base.WithRetryable(true)
	if !overridden.Retryable || overridden.Message != "expired" {
		t.Fatalf("override failed: %+v", overridden)
	}
	// The original is untouched.
	if base.Retryable {
		t.Fatal("origin must be immutable")
	}
}
