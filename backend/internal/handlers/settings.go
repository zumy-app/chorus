package handlers

import (
	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

// SettingsHandler exposes the per-account settings surface, including the
// FR-25 feature toggles (auto-translation, auto-grammar, learning highlights).
// Toggles live in Settings and are respected server-side: when translation is
// off, message translation jobs are not enqueued.
type SettingsHandler struct {
	settingsService *services.SettingsService
}

func NewSettingsHandler(settingsService *services.SettingsService) *SettingsHandler {
	return &SettingsHandler{settingsService: settingsService}
}

// GetSettings returns the caller's full settings row (feature toggles plus the
// Phase 2 grammar/vocabulary/transcript settings). A default row is created on
// first access so this never 404s.
func (h *SettingsHandler) GetSettings(c *gin.Context) {
	userID := c.GetString("userID")

	settings, err := h.settingsService.GetSettings(userID)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to load settings"))
		return
	}

	c.JSON(200, settings)
}

// UpdateSettings applies a partial update to the FR-25 feature toggles. Omitted
// toggles are left unchanged. It returns the authoritative settings row after
// the update.
func (h *SettingsHandler) UpdateSettings(c *gin.Context) {
	userID := c.GetString("userID")

	var req models.UpdateFeatureSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid settings request"))
		return
	}

	settings, err := h.settingsService.UpdateFeatureSettings(userID, req)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to save settings"))
		return
	}

	c.JSON(200, settings)
}
