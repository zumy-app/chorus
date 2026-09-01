package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type ModerationHandler struct {
	moderation *services.ModerationService
}

func NewModerationHandler(moderation *services.ModerationService) *ModerationHandler {
	return &ModerationHandler{moderation: moderation}
}

// Block creates a directed block. Enforcement treats the relationship as
// mutual: either direction stops direct-chat creation and messaging.
func (h *ModerationHandler) Block(c *gin.Context) {
	userID := c.GetString("userID")
	var req struct {
		BlockedUserID string `json:"blockedUserId" binding:"required"`
		Reason        string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid request"))
		return
	}
	if err := h.moderation.Block(c.Request.Context(), userID, req.BlockedUserID, req.Reason); err != nil {
		if err == services.ErrBlockUserNotFound {
			WriteError(c, middleware.ErrNotFound("User not found"))
			return
		}
		WriteError(c, middleware.ErrValidation("Unable to block user"))
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "User blocked"})
}

func (h *ModerationHandler) Unblock(c *gin.Context) {
	userID := c.GetString("userID")
	blockedID := c.Param("userId")
	if err := h.moderation.Unblock(c.Request.Context(), userID, blockedID); err != nil {
		WriteError(c, middleware.ErrInternal("Failed to unblock user"))
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (h *ModerationHandler) ListBlocked(c *gin.Context) {
	userID := c.GetString("userID")
	blocks, err := h.moderation.GetBlocked(c.Request.Context(), userID)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to load blocked users"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"blocks": blocks, "total": len(blocks)})
}

// GetBlockStatus returns the block relationship between the caller and another
// user. It is the per-user probe the client uses to render Block/Unblock (and
// the mutual/ghost state) on every surface the target user appears in.
// GET /api/v1/blocks/:userId/status
func (h *ModerationHandler) GetBlockStatus(c *gin.Context) {
	userID := c.GetString("userID")
	status, err := h.moderation.BlockStatus(c.Request.Context(), userID, c.Param("userId"))
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to load block status"))
		return
	}
	c.JSON(http.StatusOK, status)
}

// Report files a moderation report against a user or message.
func (h *ModerationHandler) Report(c *gin.Context) {
	userID := c.GetString("userID")
	var req models.ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid request"))
		return
	}
	report, err := h.moderation.Report(c.Request.Context(), userID, req)
	if err != nil {
		switch err {
		case services.ErrBlockUserNotFound, services.ErrReportMessageNotFound:
			WriteError(c, middleware.ErrNotFound("Message not found"))
		default:
			WriteError(c, middleware.ErrValidation("Unable to file report"))
		}
		return
	}
	c.JSON(http.StatusCreated, report)
}

// ListReports returns moderation reports (moderator+). Query params: status
// (open|resolved|dismissed|all, default open) and q (match reported username/
// email).
func (h *ModerationHandler) ListReports(c *gin.Context) {
	status := strings.TrimSpace(c.Query("status"))
	q := strings.TrimSpace(c.Query("q"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	reports, total, err := h.moderation.ListReports(c.Request.Context(), status, q, limit, offset)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to load reports"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"reports": reports, "total": total})
}

// ReportStats returns open-report figures for the moderation console.
func (h *ModerationHandler) ReportStats(c *gin.Context) {
	stats, err := h.moderation.GetReportStats(c.Request.Context())
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to load report stats"))
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *ModerationHandler) ResolveReport(c *gin.Context) {
	resolverID := c.GetString("userID")
	if err := h.moderation.ResolveReport(c.Request.Context(), resolverID, c.Param("id")); err != nil {
		if err == services.ErrReportNotFound {
			WriteError(c, middleware.ErrNotFound("Open report not found"))
			return
		}
		WriteError(c, middleware.ErrInternal("Failed to resolve report"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Report resolved"})
}

func (h *ModerationHandler) DismissReport(c *gin.Context) {
	resolverID := c.GetString("userID")
	var req struct {
		Note string `json:"note"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.moderation.DismissReport(c.Request.Context(), resolverID, c.Param("id"), req.Note); err != nil {
		if err == services.ErrReportNotFound {
			WriteError(c, middleware.ErrNotFound("Open report not found"))
			return
		}
		WriteError(c, middleware.ErrInternal("Failed to dismiss report"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Report dismissed"})
}
