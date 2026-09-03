// Package observability implements Chorus's NFR-18/NFR-25 observability
// surface: structured logs, Prometheus metrics, per-server health/readiness
// endpoints, and OpenTelemetry traces exported to Arize Phoenix.
//
// This file wires the Prometheus side. A dedicated registry (rather than the
// process-wide default) keeps `/metrics` confined to the metrics this service
// owns while still including the standard Go runtime and process collectors so
// a Grafana dashboard can show goroutines, memory, and open connections next to
// application-level gauges. The registry is built once and is safe to call from
// any goroutine.
package observability

import (
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Version is the current service version reported by /health and exported as
// the app_info gauge so dashboards can label series per release.
const Version = "2.0.0"

// Metrics registry + collectors. They are package level (not per-instance) so a
// single, stable set of series is exposed regardless of how many handlers share
// the package. Registration happens exactly once (guarded by setupOnce).
var (
	setupOnce    sync.Once
	registry     *prometheus.Registry
	appInfo      *prometheus.GaugeVec
	httpReqs     *prometheus.CounterVec
	httpDuration *prometheus.HistogramVec

	wsActiveConnections prometheus.Gauge
	wsActiveUsers       prometheus.Gauge
	wsTotalConnections  prometheus.Counter
	wsFastDropped       prometheus.Counter

	messagesSentTotal *prometheus.CounterVec
	messageLatency    *prometheus.HistogramVec

	translationRequests *prometheus.CounterVec
	translationLatency  *prometheus.HistogramVec
	translationTokens   *prometheus.CounterVec

	wordMiningJobs     *prometheus.CounterVec
	wordMiningDuration *prometheus.HistogramVec
	wordMiningItems    *prometheus.CounterVec
	wordMiningPending  prometheus.Gauge

	practiceAttempts    *prometheus.CounterVec
	practiceDuration    *prometheus.HistogramVec
	practicePromotions  *prometheus.CounterVec
	practiceLeech       prometheus.Counter
	practiceSpontaneous prometheus.Counter
	practiceMastered    prometheus.Counter
	practiceDueGauge    prometheus.Gauge

	startedAt = time.Now()
)

const (
	namespace = "chorus"
	subsystem = "backend"
)

// SetupMetrics builds and registers the Prometheus collectors. It is idempotent:
// only the first call registers, every subsequent call is a no-op. It should be
// called once during server startup, before the router serves /metrics.
func SetupMetrics() *prometheus.Registry {
	setupOnce.Do(func() {
		registry = prometheus.NewRegistry()

		appInfo = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "app_info",
			Help:      "Static build/version information for the running backend.",
		}, []string{"version", "environment"})
		registry.MustRegister(appInfo)
		appInfo.WithLabelValues(Version, envOr("ENVIRONMENT", "development")).Set(1)

		// HTTP request count + latency (seconds). Path is kept at the Gin route
		// pattern (not the raw URL) so cardinality stays bounded.
		httpReqs = prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "http_requests_total",
			Help:      "Total HTTP requests handled, by method, route, and status class.",
		}, []string{"method", "path", "status"})
		registry.MustRegister(httpReqs)

		httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request latency in seconds, by method and route.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		}, []string{"method", "path"})
		registry.MustRegister(httpDuration)

		// WebSocket connection gauges (NFR-18 "connection") — how many clients,
		// how many distinct online users, and cumulative accepted/summarily
		// dropped connections. The hub exposes a gauge setter the observability
		// package calls on registration/unregistration.
		wsActiveConnections = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "ws_active_connections",
			Help:      "Current number of established WebSocket client connections.",
		})
		registry.MustRegister(wsActiveConnections)

		wsActiveUsers = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "ws_active_users",
			Help:      "Current number of distinct users with at least one WebSocket connection.",
		})
		registry.MustRegister(wsActiveUsers)

		wsTotalConnections = prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "ws_connections_total",
			Help:      "Total WebSocket connections accepted since process start.",
		})
		registry.MustRegister(wsTotalConnections)

		wsFastDropped = prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "ws_fast_dropped_total",
			Help:      "Partially-sent WebSocket broadcasts dropped because a client's write buffer was full.",
		})
		registry.MustRegister(wsFastDropped)

		// Message send (NFR-18 "message").
		messagesSentTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "messages_sent_total",
			Help:      "Messages persisted through the send path, by outcome.",
		}, []string{"status"})
		registry.MustRegister(messagesSentTotal)

		messageLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "message_send_duration_seconds",
			Help:      "Time to persist a message end-to-end, by outcome.",
			Buckets:   []float64{0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		}, []string{"status"})
		registry.MustRegister(messageLatency)

		// Translation latency (NFR-18 "translation latency"). A single cache hit
		// vs. a real provider call have wildly different profiles, so the
		// cache_hit label keeps the distribution meaningful.
		translationRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "translation_requests_total",
			Help:      "Translation calls issued, by provider, cache hit, and outcome.",
		}, []string{"provider", "cache_hit", "status"})
		registry.MustRegister(translationRequests)

		translationLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "translation_duration_seconds",
			Help:      "Translation latency in seconds, by provider and cache hit.",
			Buckets:   []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
		}, []string{"provider", "cache_hit"})
		registry.MustRegister(translationLatency)

		translationTokens = prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "translation_tokens_total",
			Help:      "Cumulative model tokens consumed by translation, by provider.",
		}, []string{"provider"})
		registry.MustRegister(translationTokens)

		wordMiningJobs = prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "word_mining_jobs_total",
			Help:      "Word-mining jobs by outcome (enqueued, done, failed, ignored).",
		}, []string{"status"})
		registry.MustRegister(wordMiningJobs)

		wordMiningDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "word_mining_duration_seconds",
			Help:      "Time to process a word-mining job, by outcome.",
			Buckets:   []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
		}, []string{"status"})
		registry.MustRegister(wordMiningDuration)

		wordMiningItems = prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "word_mining_items_total",
			Help:      "Mined vocabulary items by status and route (candidate, auto_added, route).",
		}, []string{"status", "route"})
		registry.MustRegister(wordMiningItems)

		wordMiningPending = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "word_mining_pending_jobs",
			Help:      "Current pending/processing word-mining jobs (DB polled).",
		})
		registry.MustRegister(wordMiningPending)

		practiceAttempts = prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "practice_attempts_total",
			Help:      "Practice attempts by stage and outcome (correct/incorrect).",
		}, []string{"stage", "outcome"})
		registry.MustRegister(practiceAttempts)

		practiceDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "practice_duration_seconds",
			Help:      "Time to grade a practice attempt, by stage.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2},
		}, []string{"stage"})
		registry.MustRegister(practiceDuration)

		practicePromotions = prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "practice_stage_promotions_total",
			Help:      "Stage promotions by from_stage and to_stage.",
		}, []string{"from_stage", "to_stage"})
		registry.MustRegister(practicePromotions)

		practiceLeech = prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "practice_leech_total",
			Help:      "Cards marked leech (lapses>=3 at low stage).",
		})
		registry.MustRegister(practiceLeech)

		practiceSpontaneous = prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "practice_spontaneous_total",
			Help:      "Spontaneous-use promotions (stage 5).",
		})
		registry.MustRegister(practiceSpontaneous)

		practiceMastered = prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "practice_mastered_total",
			Help:      "Cards reaching mastered state (production-gated).",
		})
		registry.MustRegister(practiceMastered)

		practiceDueGauge = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "practice_due_cards",
			Help:      "Current due cards gauge (sampled).",
		})
		registry.MustRegister(practiceDueGauge)

		// Standard runtime/process collectors so the Grafana dashboard can show
		// goroutines, memory, CPU, and open file descriptors alongside app data.
		registry.MustRegister(prometheus.NewGoCollector())
		registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	})

	return registry
}

// Registry returns the Prometheus registry, initializing it if needed.
func Registry() *prometheus.Registry {
	if registry == nil {
		SetupMetrics()
	}
	return registry
}

// MetricsHandler returns an http.Handler for the /metrics scrape endpoint backed
// by the custom registry.
func MetricsHandler() http.Handler {
	return promhttp.HandlerFor(Registry(), promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// MetricsMiddleware records method/route/status + latency for every matched Gin
// route. Routes are recorded by their registered pattern (c.FullPath()), keeping
// cardinality bounded even under URL-param abuse. /metrics itself is registered
// on the same router but scrubbed from the path label to avoid scraping noise.
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		// Only observe when a route matched; 404s from the NoRoute handler are
		// surfaced as their raw path, which is acceptable and bounded.
		path := c.FullPath()
		if path == "/metrics" || path == "" {
			return
		}
		dur := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		httpReqs.WithLabelValues(c.Request.Method, path, status).Inc()
		httpDuration.WithLabelValues(c.Request.Method, path).Observe(dur)
	}
}

// SetWSConnections updates the active WS gauge from the hub. Called by
// WebSocketHub after registering/unregistering a client.
func SetWSConnections(activeConns, activeUsers int) {
	if wsActiveConnections == nil {
		return
	}
	wsActiveConnections.Set(float64(activeConns))
	wsActiveUsers.Set(float64(activeUsers))
}

// IncWSConnections records that a WebSocket connection was accepted.
func IncWSConnections() {
	if wsTotalConnections == nil {
		return
	}
	wsTotalConnections.Inc()
}

// IncWSFastDropped records a broadcast that could not be buffered.
func IncWSFastDropped() {
	if wsFastDropped == nil {
		return
	}
	wsFastDropped.Inc()
}

// ObserveMessageSend records a persisted message and its elapsed time.
func ObserveMessageSend(status string, elapsed time.Duration) {
	if messagesSentTotal == nil {
		return
	}
	messagesSentTotal.WithLabelValues(status).Inc()
	messageLatency.WithLabelValues(status).Observe(elapsed.Seconds())
}

// ObserveTranslation records a translation outcome: request count, latency, and
// token consumption. cacheHit reports whether the result was served from Redis.
func ObserveTranslation(provider string, cacheHit bool, status string, elapsed time.Duration, tokens int) {
	if translationRequests == nil {
		return
	}
	cacheLabel := "miss"
	if cacheHit {
		cacheLabel = "hit"
	}
	translationRequests.WithLabelValues(provider, cacheLabel, status).Inc()
	translationLatency.WithLabelValues(provider, cacheLabel).Observe(elapsed.Seconds())
	if tokens > 0 {
		translationTokens.WithLabelValues(provider).Add(float64(tokens))
	}
}

func ObserveWordMiningJob(status string, elapsed time.Duration) {
	if wordMiningJobs == nil {
		return
	}
	wordMiningJobs.WithLabelValues(status).Inc()
	if elapsed > 0 && wordMiningDuration != nil {
		wordMiningDuration.WithLabelValues(status).Observe(elapsed.Seconds())
	}
}

func ObserveWordMiningItem(status, route string) {
	if wordMiningItems == nil {
		return
	}
	wordMiningItems.WithLabelValues(status, route).Inc()
}

func SetWordMiningPending(n int) {
	if wordMiningPending == nil {
		return
	}
	wordMiningPending.Set(float64(n))
}

func ObservePracticeAttempt(stage int, correct bool, elapsed time.Duration) {
	if practiceAttempts == nil {
		return
	}
	outcome := "incorrect"
	if correct {
		outcome = "correct"
	}
	practiceAttempts.WithLabelValues(stageLabel(stage), outcome).Inc()
	if elapsed > 0 && practiceDuration != nil {
		practiceDuration.WithLabelValues(stageLabel(stage)).Observe(elapsed.Seconds())
	}
}

func ObservePracticePromotion(fromStage, toStage int) {
	if practicePromotions == nil {
		return
	}
	practicePromotions.WithLabelValues(stageLabel(fromStage), stageLabel(toStage)).Inc()
}

func IncPracticeLeech() {
	if practiceLeech == nil {
		return
	}
	practiceLeech.Inc()
}

func IncPracticeSpontaneous() {
	if practiceSpontaneous == nil {
		return
	}
	practiceSpontaneous.Inc()
}

func IncPracticeMastered() {
	if practiceMastered == nil {
		return
	}
	practiceMastered.Inc()
}

func SetPracticeDue(n int) {
	if practiceDueGauge == nil {
		return
	}
	practiceDueGauge.Set(float64(n))
}

func stageLabel(s int) string {
	switch s {
	case 1:
		return "1_recognition"
	case 2:
		return "2_cued"
	case 3:
		return "3_free"
	case 4:
		return "4_production"
	case 5:
		return "5_spontaneous"
	default:
		return "unknown"
	}
}

// envOr reads an env var returning def when it is unset or empty.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
