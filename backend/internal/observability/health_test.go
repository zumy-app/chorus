package observability

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLivenessAlwaysHealthy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHealth("test-1.0.0")
	h.AddCheck("postgres", func(ctx context.Context) error {
		return errors.New("db down")
	})

	router := gin.New()
	router.GET("/health", h.Liveness())

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/health", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("liveness should stay 200 when a dependency is down, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"status":"healthy"`) {
		t.Fatalf("expected status healthy, got %s", body)
	}
	if !strings.Contains(body, `"version":"test-1.0.0"`) {
		t.Fatalf("expected version in payload, got %s", body)
	}
	if !strings.Contains(body, `"postgres":"unavailable"`) {
		t.Fatalf("expected dependency detail, got %s", body)
	}
}

func TestReadiness503And200(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Failing dependency -> 503.
	h := NewHealth("test-1.0.0")
	h.AddCheck("redis", func(ctx context.Context) error {
		return errors.New("redis down")
	})
	router := gin.New()
	router.GET("/health/ready", h.Readiness())

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/health/ready", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when a check fails, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"status":"not_ready"`) {
		t.Fatalf("expected not_ready, got %s", w.Body.String())
	}

	// All checks pass -> 200.
	h2 := NewHealth("test-1.0.0")
	h2.AddCheck("postgres", func(ctx context.Context) error { return nil })
	h2.AddCheck("redis", func(ctx context.Context) error { return nil })
	router2 := gin.New()
	router2.GET("/health/ready", h2.Readiness())

	w2 := httptest.NewRecorder()
	router2.ServeHTTP(w2, httptest.NewRequest("GET", "/health/ready", nil))
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 when all checks pass, got %d", w2.Code)
	}
	if !strings.Contains(w2.Body.String(), `"status":"ready"`) {
		t.Fatalf("expected ready, got %s", w2.Body.String())
	}
}

func TestEmptyChecksAreReady(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHealth("test-1.0.0")
	router := gin.New()
	router.GET("/health/ready", h.Readiness())

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/health/ready", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("no checks registered -> ready, got %d", w.Code)
	}
}
