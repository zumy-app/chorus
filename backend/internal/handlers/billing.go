package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

// BillingHandler exposes the subscription surface: the user's own subscription
// and checkout (protected), premium management + analytics (admin), and the
// PayPal webhook (public).
type BillingHandler struct {
	billing      *services.BillingService
	paypal       *services.PayPalClient
	entitlements *services.EntitlementService
}

func NewBillingHandler(billing *services.BillingService, paypal *services.PayPalClient, entitlements *services.EntitlementService) *BillingHandler {
	return &BillingHandler{billing: billing, paypal: paypal, entitlements: entitlements}
}

// GetMySubscription returns the current user's subscription view.
func (h *BillingHandler) GetMySubscription(c *gin.Context) {
	info, err := h.billing.GetSubscription(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		WriteError(c, middleware.ErrNotFound("User not found"))
		return
	}
	c.JSON(http.StatusOK, info)
}

// Checkout creates a provider subscription and returns the approval URL.
// Plan: monthly | annual. Optional return_url / cancel_url query params
// (absolute or path); defaults resolve against the request Origin.
func (h *BillingHandler) Checkout(c *gin.Context) {
	var req models.CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid request: plan must be monthly or annual"))
		return
	}
	origin := c.GetHeader("Origin")
	if origin == "" {
		origin = "http://localhost:3000"
	}
	returnURL := c.Query("return_url")
	if returnURL == "" {
		returnURL = "/premium"
	}
	cancelURL := c.Query("cancel_url")
	if cancelURL == "" {
		cancelURL = "/pricing"
	}
	abs := func(p string) string {
		if len(p) > 0 && p[0] == '/' {
			return origin + p
		}
		return p
	}

	resp, err := h.billing.Checkout(c.Request.Context(), c.GetString("userID"), req.Plan, abs(returnURL), abs(cancelURL))
	if err != nil {
		if errors.Is(err, services.ErrPayPalUnconfigured) || errors.Is(err, services.ErrPayPalInvalidPlan) {
			WriteError(c, middleware.ErrUnavailable("Payments are not configured yet"))
			return
		}
		WriteError(c, middleware.ErrInternal("Failed to start checkout"))
		return
	}
	c.JSON(http.StatusCreated, resp)
}

// GrantPlan applies an admin grant (or delegates to revoke when plan=free).
func (h *BillingHandler) GrantPlan(c *gin.Context) {
	var req models.GrantPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid request"))
		return
	}
	actorID := c.GetString("userID")
	user, err := h.billing.GrantPremium(c.Request.Context(), &actorID, c.Param("id"), req)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			WriteError(c, middleware.ErrNotFound("User not found"))
		case errors.Is(err, services.ErrInvalidGrace):
			WriteError(c, middleware.ErrValidation("Invalid grace window (days must be > 0, until must be a future RFC3339 time)"))
		case errors.Is(err, services.ErrUserNotPremium):
			WriteError(c, middleware.ErrValidation("User has no active premium to revoke"))
		default:
			WriteError(c, middleware.ErrInternal("Failed to update plan"))
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":        "Plan updated",
		"plan":           user.Plan,
		"effectivePlan":  h.effectivePlan(user),
		"planGraceUntil": user.PlanGraceUntil,
	})
}

// RevokePlan drops a user back to free (immediately or after a grace window).
func (h *BillingHandler) RevokePlan(c *gin.Context) {
	var req models.GrantPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid request"))
		return
	}
	actorID := c.GetString("userID")
	user, err := h.billing.RevokePremium(c.Request.Context(), &actorID, c.Param("id"), req)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			WriteError(c, middleware.ErrNotFound("User not found"))
		case errors.Is(err, services.ErrUserNotPremium):
			WriteError(c, middleware.ErrValidation("User has no active premium to revoke"))
		default:
			WriteError(c, middleware.ErrInternal("Failed to update plan"))
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":        "Plan revoked",
		"plan":           user.Plan,
		"planGraceUntil": user.PlanGraceUntil,
	})
}

// ListPremiumUsers returns premium (stored or in-grace) users for the console.
func (h *BillingHandler) ListPremiumUsers(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	users, total, err := h.billing.ListPremiumUsers(c.Request.Context(), c.Query("q"), limit, offset)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Unable to load premium users"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users, "total": total})
}

// PremiumAnalytics returns aggregated premium metrics for the dashboard.
func (h *BillingHandler) PremiumAnalytics(c *gin.Context) {
	stats, err := h.billing.PremiumAnalytics(c.Request.Context())
	if err != nil {
		WriteError(c, middleware.ErrInternal("Unable to load premium analytics"))
		return
	}
	c.JSON(http.StatusOK, stats)
}

// PlanHistory returns the audit trail for a single user.
func (h *BillingHandler) PlanHistory(c *gin.Context) {
	history, err := h.billing.PlanHistory(c.Request.Context(), c.Param("id"))
	if err != nil {
		WriteError(c, middleware.ErrInternal("Unable to load plan history"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"history": history})
}

// Webhook ingests a PayPal webhook. Signature verification is mandatory except
// when PAYPAL_WEBHOOK_INSECURE=true (dev only).
func (h *BillingHandler) Webhook(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		WriteError(c, middleware.ErrValidation("Unable to read body"))
		return
	}
	if h.paypal == nil {
		WriteError(c, middleware.ErrUnavailable("Payments are not configured"))
		return
	}
	if err := h.paypal.VerifyWebhook(c.Request.Context(),
		c.GetHeader("PAYPAL-TRANSMISSION-ID"),
		c.GetHeader("PAYPAL-TRANSMISSION-TIME"),
		c.GetHeader("PAYPAL-CERT-URL"),
		c.GetHeader("PAYPAL-AUTH-ALGO"),
		c.GetHeader("PAYPAL-TRANSMISSION-SIG"),
		body); err != nil {
		WriteError(c, middleware.ErrAuth("Webhook verification failed"))
		return
	}
	if err := h.billing.ApplyWebhook(c.Request.Context(), body); err != nil {
		WriteError(c, middleware.ErrValidation("Unable to process event"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// effectivePlan resolves the post-change effective plan for response payloads.
func (h *BillingHandler) effectivePlan(u *models.User) string {
	return h.entitlements.ResolveNow(u).EffectivePlan
}
