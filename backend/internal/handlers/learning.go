package handlers

import (
	"net/http"

	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

// LearningHandler serves the pair-aware learning endpoints: capabilities,
// profile, dashboard, and roadmap path. All routes require auth; the caller's
// user id is resolved from the request context by AuthMiddleware.
type LearningHandler struct {
	capabilities *services.LearningCapabilityService
	profiles     *services.LearningProfileService
	dashboard    *services.LearningDashboardService
	curriculum   *services.CurriculumService
}

func NewLearningHandler(
	capabilities *services.LearningCapabilityService,
	profiles *services.LearningProfileService,
	dashboard *services.LearningDashboardService,
	curriculum *services.CurriculumService,
) *LearningHandler {
	return &LearningHandler{
		capabilities: capabilities,
		profiles:     profiles,
		dashboard:    dashboard,
		curriculum:   curriculum,
	}
}

// GetCapabilities resolves how much structured learning support exists for a
// pair. GET /api/v1/learning/capabilities?nativeLanguage=en&targetLanguage=es
func (h *LearningHandler) GetCapabilities(c *gin.Context) {
	native := c.Query("nativeLanguage")
	target := c.Query("targetLanguage")
	capability, err := h.capabilities.GetCapability(c.Request.Context(), native, target)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve learning capability"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": capability})
}

// GetProfile returns (and lazily creates) the caller's learning profile for a
// pair. GET /api/v1/learning/profile?targetLanguage=es&nativeLanguage=en
func (h *LearningHandler) GetProfile(c *gin.Context) {
	userID := c.GetString("userID")
	profile, err := h.profiles.GetProfile(c.Request.Context(), userID, c.Query("targetLanguage"), c.Query("nativeLanguage"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load learning profile"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": profile})
}

// UpdateProfile applies user-editable profile settings.
// PUT /api/v1/learning/profile
func (h *LearningHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetString("userID")
	var req models.LearningProfileUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	profile, err := h.profiles.UpdateProfile(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update learning profile"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": profile})
}

// GetDashboard returns the aggregated Learn dashboard.
// GET /api/v1/learning/dashboard?targetLanguage=es&nativeLanguage=en
func (h *LearningHandler) GetDashboard(c *gin.Context) {
	userID := c.GetString("userID")
	dash, err := h.dashboard.GetDashboard(c.Request.Context(), userID, c.Query("targetLanguage"), c.Query("nativeLanguage"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load dashboard"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": dash})
}

// GetPath returns the roadmap (units with progress) for the caller's pair.
// GET /api/v1/learning/path?targetLanguage=es&nativeLanguage=en
func (h *LearningHandler) GetPath(c *gin.Context) {
	userID := c.GetString("userID")
	target := c.Query("targetLanguage")
	native := c.Query("nativeLanguage")

	capability, err := h.capabilities.GetCapability(c.Request.Context(), native, target)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve learning capability"})
		return
	}
	profile, err := h.profiles.GetProfile(c.Request.Context(), userID, target, native)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load learning profile"})
		return
	}
	path, err := h.curriculum.GetLearningPath(c.Request.Context(), userID, profile, capability)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load learning path"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": path})
}
