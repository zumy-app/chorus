package handlers

import (
	"net/http"
	"strconv"

	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

// AdminQualityHandler surfaces the FR-30 quality pipeline so admins can see the
// cross-model evaluation results and aggregated KPIs (accuracy, p95 latency,
// cost/1k tokens, cache hit rate). Route-level RequireRole guards apply.
type AdminQualityHandler struct {
	evaluator *services.QualityEvaluatorService
}

func NewAdminQualityHandler(evaluator *services.QualityEvaluatorService) *AdminQualityHandler {
	return &AdminQualityHandler{evaluator: evaluator}
}

// KPIs returns aggregated FR-30 quality metrics.
func (h *AdminQualityHandler) KPIs(c *gin.Context) {
	kpis, err := h.evaluator.KPIs()
	if err != nil {
		WriteError(c, middleware.ErrInternal("Unable to compute quality KPIs"))
		return
	}
	c.JSON(http.StatusOK, kpis)
}

// Enabled reports whether the cross-model evaluator is configured (has an
// evaluator model chain). Helpful for the admin UI to show a hint when off.
func (h *AdminQualityHandler) Enabled(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"enabled": h.evaluator.Enabled()})
}

// Requeue forces the evaluator to re-score every job that lacks an eval. This is
// the "nightly batch re-scores a sample" affordance, available on demand.
func (h *AdminQualityHandler) Requeue(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "200"))
	count, err := h.evaluator.RequeueUnevaluated(limit)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Unable to requeue quality evals"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"requeued": count})
}
