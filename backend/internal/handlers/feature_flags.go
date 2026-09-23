package handlers

import (
	"net/http"

	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FeatureFlagHandler exposes the resolved flag map to clients and admin CRUD.
type FeatureFlagHandler struct {
	svc *services.FeatureFlagService
}

func NewFeatureFlagHandler(svc *services.FeatureFlagService) *FeatureFlagHandler {
	return &FeatureFlagHandler{svc: svc}
}

// GetMyFlags returns the resolved feature-flag map for the authenticated user.
// This is the primary endpoint the mobile/web clients call on session start.
func (h *FeatureFlagHandler) GetMyFlags(c *gin.Context) {
	userID := c.GetString("userID")
	uid, err := uuid.Parse(userID)
	if err != nil {
		middleware.WriteError(c, middleware.ErrAuth("invalid user id"))
		return
	}

	role := c.GetString("userRole")
	isAdmin := role == services.RoleAdmin
	isBeta := c.GetBool("userBetaAccess") || isAdmin

	flags, err := h.svc.ResolveForUser(c.Request.Context(), uid, isAdmin, isBeta)
	if err != nil {
		middleware.WriteError(c, middleware.NewError(middleware.KindInternal, "failed to resolve flags"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"flags": flags})
}

// ListFlags returns all flag definitions for the admin panel.
func (h *FeatureFlagHandler) ListFlags(c *gin.Context) {
	flags, err := h.svc.ListFlags(c.Request.Context())
	if err != nil {
		middleware.WriteError(c, middleware.NewError(middleware.KindInternal, "failed to list flags"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"flags": flags})
}

// UpdateFlagTiersRequest is the body for updating a flag's audience tiers.
type UpdateFlagTiersRequest struct {
	AdminOnly  bool `json:"adminOnly"`
	BetaAccess bool `json:"betaAccess"`
	Stable     bool `json:"stable"`
}

// UpdateFlagTiers updates the audience tiers for a flag.
func (h *FeatureFlagHandler) UpdateFlagTiers(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		middleware.WriteError(c, middleware.ErrValidation("flag key is required"))
		return
	}

	var req UpdateFlagTiersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.WriteError(c, middleware.ErrValidation("invalid request body"))
		return
	}

	if err := h.svc.SetFlagTiers(c.Request.Context(), key, req.AdminOnly, req.BetaAccess, req.Stable); err != nil {
		middleware.WriteError(c, middleware.NewError(middleware.KindInternal, "failed to update flag"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// SetUserOverrideRequest is the body for a per-user override.
type SetUserOverrideRequest struct {
	Enabled bool `json:"enabled"`
}

// SetUserOverride creates or updates a per-user flag override.
func (h *FeatureFlagHandler) SetUserOverride(c *gin.Context) {
	key := c.Param("key")
	userID := c.Param("userId")
	if key == "" || userID == "" {
		middleware.WriteError(c, middleware.ErrValidation("flag key and user id are required"))
		return
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		middleware.WriteError(c, middleware.ErrValidation("invalid user id"))
		return
	}

	var req SetUserOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.WriteError(c, middleware.ErrValidation("invalid request body"))
		return
	}

	if err := h.svc.SetUserOverride(c.Request.Context(), uid, key, req.Enabled); err != nil {
		middleware.WriteError(c, middleware.NewError(middleware.KindInternal, "failed to set override"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// DeleteUserOverride removes a per-user flag override.
func (h *FeatureFlagHandler) DeleteUserOverride(c *gin.Context) {
	key := c.Param("key")
	userID := c.Param("userId")
	if key == "" || userID == "" {
		middleware.WriteError(c, middleware.ErrValidation("flag key and user id are required"))
		return
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		middleware.WriteError(c, middleware.ErrValidation("invalid user id"))
		return
	}

	if err := h.svc.DeleteUserOverride(c.Request.Context(), uid, key); err != nil {
		middleware.WriteError(c, middleware.NewError(middleware.KindInternal, "failed to delete override"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// PreviewUserFlags returns the resolved flag map for any user (admin only).
func (h *FeatureFlagHandler) PreviewUserFlags(c *gin.Context) {
	userID := c.Param("userId")
	uid, err := uuid.Parse(userID)
	if err != nil {
		middleware.WriteError(c, middleware.ErrValidation("invalid user id"))
		return
	}

	// Preview always resolves as a general user (not admin, not beta) so the
	// admin can see exactly what that user sees. Per-user overrides still apply.
	flags, err := h.svc.ResolveForUser(c.Request.Context(), uid, false, false)
	if err != nil {
		middleware.WriteError(c, middleware.NewError(middleware.KindInternal, "failed to resolve flags"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"flags": flags})
}
