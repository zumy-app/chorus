// health.go implements the per-server health surface for NFR-18/NFR-19.
//
// Two endpoints are exposed:
//
//	GET /health        → liveness. 200 as long as the process is up; body carries
//	                     a per-dependency detail map for diagnostics. Load
//	                     balancers use this as their health check so a degraded
//	                     dependency never prematurely kills a healthy instance.
//	GET /health/ready  → readiness. 200 only when every registered checker
//	                     passes, otherwise 503. Used inside a rolling deploy to
//	                     keep traffic off a pod that cannot yet serve: once the
//	                     DB/Redis/translation-chain checks pass it flips ready.
package observability

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CheckFunc probes a single dependency for readiness. Implementations must be
// safe to call concurrently and must bound their own work with a context/timeout.
type CheckFunc func(ctx context.Context) error

// Health aggregates a set of named dependency checkers and serves liveness +
// readiness. Call AddCheck during startup after each dependency is constructed.
// With no checks registered the service is treated as ready (lean local mode)
// so liveness and readiness never disagree.
type Health struct {
	version string
	mu      sync.RWMutex
	checks  map[string]CheckFunc
	started time.Time
}

// NewHealth returns a Health carrying the given build version.
func NewHealth(version string) *Health {
	return &Health{
		version: version,
		checks:  make(map[string]CheckFunc),
		started: time.Now(),
	}
}

// AddCheck registers a named dependency checker. The name appears in the health
// payload (e.g. "postgres", "redis", "translation").
func (h *Health) AddCheck(name string, fn CheckFunc) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks[name] = fn
}

// RunChecks executes every registered checker with a shared timeout. It returns
// the per-check results and whether all passed. Checks whose timeout/cancellation
// fails a dependency, and an empty check set is treated as ready (per MarkReady).
func (h *Health) RunChecks(ctx context.Context) (map[string]string, bool) {
	h.mu.RLock()
	names := make([]string, 0, len(h.checks))
	for name := range h.checks {
		names = append(names, name)
	}
	h.mu.RUnlock()

	if len(names) == 0 {
		return map[string]string{"status": "ready"}, true
	}

	const probeTimeout = 3 * time.Second
	results := make(map[string]string, len(names))
	ready := true
	for _, name := range names {
		probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
		err := h.check(name, probeCtx)
		cancel()
		if err != nil {
			results[name] = "unavailable"
			ready = false
		} else {
			results[name] = "ok"
		}
	}
	return results, ready
}

// check runs a single checker by name, treating a nil/removed checker as ok.
func (h *Health) check(name string, ctx context.Context) error {
	h.mu.RLock()
	fn, ok := h.checks[name]
	h.mu.RUnlock()
	if !ok || fn == nil {
		return nil
	}
	return fn(ctx)
}

// Liveness returns the liveness handler. It always returns 200 (the process is
// up) so the load balancer never kills an instance just because a dependency is
// temporarily down; the body reports per-dependency detail for operators.
func (h *Health) Liveness() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"version":   h.version,
			"uptime_s":  int(time.Since(h.started).Seconds()),
			"checkTime": time.Now().UTC().Format(time.RFC3339),
			"checks":    h.currentStatus(),
		})
	}
}

// Readiness returns the readiness handler: 200 only when every registered
// dependency check passes, otherwise 503 with the failing checks listed.
func (h *Health) Readiness() gin.HandlerFunc {
	return func(c *gin.Context) {
		results, ready := h.RunChecks(c.Request.Context())
		if !ready {
			onReadinessFail(c, results)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":  "ready",
			"version": h.version,
			"checks":  results,
		})
	}
}

// currentStatus runs the checks for the liveness payload, best-effort.
func (h *Health) currentStatus() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	results, _ := h.RunChecks(ctx)
	return results
}

// onReadinessFail writes a 503 with per-check detail. It is a tiny function so
// the readiness handler reads cleanly.
func onReadinessFail(c *gin.Context, failing map[string]string) {
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"status": "not_ready",
		"checks": failing,
	})
}
