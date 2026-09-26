package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	dto "github.com/prometheus/client_model/go"
)

// TestSetupMetricsIdempotent ensures repeated setup never panics or re-registers
// the collectors (which would panic on duplicate registration).
func TestSetupMetricsIdempotent(t *testing.T) {
	reg := SetupMetrics()
	reg2 := SetupMetrics()
	if reg == nil || reg2 == nil {
		t.Fatal("SetupMetrics must return a registry")
	}
}

func TestMetricsHandlerExposesAppInfo(t *testing.T) {
	SetupMetrics()

	w := httptest.NewRecorder()
	handler := MetricsHandler()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from /metrics, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "chorus_backend_app_info") {
		t.Fatalf("expected app_info metric in scrape, got:\n%s", head(body))
	}
	if !strings.Contains(body, "go_goroutines") {
		t.Fatalf("expected go_goroutines (runtime collector) in scrape, got:\n%s", head(body))
	}
}

func TestMetricsMiddleware(t *testing.T) {
	SetupMetrics()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(MetricsMiddleware())
	router.GET("/health", func(c *gin.Context) { c.String(200, "ok") })
	router.GET("/metrics", gin.WrapH(MetricsHandler()))

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/health", nil))
	if n := metricValue(t, "chorus_backend_http_requests_total", map[string]string{"method": "GET", "path": "/health", "status": "200"}); n < 1 {
		t.Fatalf("expected middleware to record the request, got %v", n)
	}
	// /metrics must NOT be observed (scrubbed to avoid scrape noise).
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/metrics", nil))
	if n := metricValue(t, "chorus_backend_http_requests_total", map[string]string{"method": "GET", "path": "", "status": "200"}); n > 0 {
		t.Fatalf("expected /metrics to be scrubbed, got %v", n)
	}
}

func TestObserveTranslation(t *testing.T) {
	SetupMetrics()
	ObserveTranslation("ollama", false, "ok", 50*time.Millisecond, 12)

	if n := metricValue(t, "chorus_backend_translation_requests_total", map[string]string{"provider": "ollama", "cache_hit": "miss", "status": "ok"}); n < 1 {
		t.Fatalf("expected translation request counter to increment, got %v", n)
	}
	if n := metricValue(t, "chorus_backend_translation_tokens_total", map[string]string{"provider": "ollama"}); n < 12 {
		t.Fatalf("expected translation tokens >= 12, got %v", n)
	}
}

func TestObserveMessageSend(t *testing.T) {
	SetupMetrics()
	ObserveMessageSend("sent", 5*time.Millisecond)

	if n := metricValue(t, "chorus_backend_messages_sent_total", map[string]string{"status": "sent"}); n < 1 {
		t.Fatalf("expected message counter to increment, got %v", n)
	}
}

func TestWSConnectionGauges(t *testing.T) {
	SetupMetrics()
	IncWSConnections()
	SetWSConnections(3, 2)

	if v := metricValue(t, "chorus_backend_ws_active_connections", nil); v != 3 {
		t.Fatalf("expected ws_active_connections=3, got %v", v)
	}
	if v := metricValue(t, "chorus_backend_ws_active_users", nil); v != 2 {
		t.Fatalf("expected ws_active_users=2, got %v", v)
	}
	if n := metricValue(t, "chorus_backend_ws_connections_total", nil); n < 1 {
		t.Fatalf("expected ws_connections_total >= 1, got %v", n)
	}
}

// metricValue gathers the registry and returns the value of the named metric
// whose labels match labels. A nil labels map matches the sole (or unlabeled)
// series. Returns -1 when no matching series exists.
func metricValue(t *testing.T, name string, labels map[string]string) float64 {
	t.Helper()
	mfs, err := Registry().Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			if !labelsMatch(m, labels) {
				continue
			}
			if c := m.GetCounter(); c != nil {
				return c.GetValue()
			}
			if g := m.GetGauge(); g != nil {
				return g.GetValue()
			}
		}
	}
	return -1
}

func labelsMatch(m *dto.Metric, want map[string]string) bool {
	if want == nil {
		return true
	}
	for _, lp := range m.GetLabel() {
		if v, ok := want[lp.GetName()]; ok && lp.GetValue() != v {
			return false
		}
	}
	return true
}

func head(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) > 5 {
		return strings.Join(lines[:5], "\n")
	}
	return s
}
