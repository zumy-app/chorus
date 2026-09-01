package handlers

import (
	"database/sql"
	"strconv"
	"strings"

	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type PayoutHandler struct {
	svc *services.PayoutService
}

func NewPayoutHandler(svc *services.PayoutService) *PayoutHandler {
	return &PayoutHandler{svc: svc}
}

func (h *PayoutHandler) GetOverview(c *gin.Context) {
	ov, err := h.svc.GetOverview(c.GetString("userID"))
	if err != nil {
		if err == sql.ErrNoRows {
			WriteError(c, middleware.ErrNotFound("Teacher not found"))
			return
		}
		WriteError(c, middleware.ErrInternal("Failed to load earnings"))
		return
	}
	c.JSON(200, gin.H{"overview": ov})
}

func (h *PayoutHandler) ListMethods(c *gin.Context) {
	methods, err := h.svc.ListMethods(c.GetString("userID"))
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to list payout methods"))
		return
	}
	c.JSON(200, gin.H{"methods": methods})
}

func (h *PayoutHandler) AddMethod(c *gin.Context) {
	var req models.CreatePayoutMethodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid payout method"))
		return
	}
	m, err := h.svc.AddMethod(c.GetString("userID"), req)
	if err != nil {
		WriteError(c, middleware.ErrValidation(err.Error()))
		return
	}
	c.JSON(201, gin.H{"method": m})
}

func (h *PayoutHandler) RemoveMethod(c *gin.Context) {
	if err := h.svc.RemoveMethod(c.GetString("userID"), c.Param("methodId")); err != nil {
		if err == sql.ErrNoRows {
			WriteError(c, middleware.ErrNotFound("Payout method not found"))
			return
		}
		WriteError(c, middleware.ErrInternal("Failed to remove method"))
		return
	}
	c.JSON(200, gin.H{"status": "deleted"})
}

func (h *PayoutHandler) SetDefaultMethod(c *gin.Context) {
	if err := h.svc.SetDefaultMethod(c.GetString("userID"), c.Param("methodId")); err != nil {
		if err == sql.ErrNoRows {
			WriteError(c, middleware.ErrNotFound("Payout method not found"))
			return
		}
		WriteError(c, middleware.ErrInternal("Failed to set default"))
		return
	}
	c.JSON(200, gin.H{"status": "ok"})
}

func (h *PayoutHandler) ListHistory(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	payouts, total, err := h.svc.ListPayouts(c.GetString("userID"), limit, offset)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to list payouts"))
		return
	}
	hasMore := offset+len(payouts) < total
	c.JSON(200, gin.H{"payouts": payouts, "total": total, "hasMore": hasMore})
}

func (h *PayoutHandler) RequestPayout(c *gin.Context) {
	var req models.CreatePayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid payout request"))
		return
	}
	if req.MethodID != nil {
		v := strings.TrimSpace(*req.MethodID)
		req.MethodID = &v
	}
	p, err := h.svc.RequestPayout(c.GetString("userID"), req)
	if err != nil {
		if strings.Contains(err.Error(), "available") || strings.Contains(err.Error(), "minimum") || strings.Contains(err.Error(), "positive") || strings.Contains(err.Error(), "no payout") || strings.Contains(err.Error(), "method not found") {
			WriteError(c, middleware.ErrValidation(err.Error()))
			return
		}
		WriteError(c, middleware.ErrInternal("Failed to create payout"))
		return
	}
	c.JSON(201, gin.H{"payout": p})
}
