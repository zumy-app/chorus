package handlers

import (
	"net/http"
	"strconv"

	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

// AdminTranslationsHandler surfaces the translation queue (durable jobs) so
// admins and moderators can audit delivery and retry translations that failed
// because of provider/config issues. Route-level RequireRole guards apply.
type AdminTranslationsHandler struct {
	queue      *services.TranslationQueueService
	translator *services.TranslationService
}

func NewAdminTranslationsHandler(queue *services.TranslationQueueService, translator *services.TranslationService) *AdminTranslationsHandler {
	return &AdminTranslationsHandler{queue: queue, translator: translator}
}

// List returns translation jobs, optionally filtered by status and text query.
// Query params: status (pending|processing|done|failed|all), q, limit.
func (h *AdminTranslationsHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	jobs, err := h.queue.List(c.Query("status"), c.Query("q"), limit)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Unable to load translation jobs"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"jobs": jobs, "total": len(jobs)})
}

// Retry forces an immediate re-attempt of a pending or failed job. Useful after
// fixing a translation provider config issue.
func (h *AdminTranslationsHandler) Retry(c *gin.Context) {
	if err := h.queue.RetryByID(c.Param("id")); err != nil {
		WriteError(c, middleware.ErrValidation("Unable to retry translation"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Translation re-queued"})
}

// Health reports the readiness of every provider in the translation chain so
// config issues (e.g. a missing API key) are diagnosable from the admin UI.
func (h *AdminTranslationsHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"providers": h.translator.ProviderHealth()})
}
