package handlers

import (
	"errors"
	"net/http"

	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type OTPHandler struct {
	otp  *services.OTPService
	auth *services.AuthService
	user *services.UserService
}

func NewOTPHandler(otp *services.OTPService, auth *services.AuthService, user *services.UserService) *OTPHandler {
	return &OTPHandler{otp: otp, auth: auth, user: user}
}

func (h *OTPHandler) GetStatus(c *gin.Context) {
	userID := c.GetString("userID")
	status, err := h.otp.GetStatus(userID)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to get phone status"))
		return
	}
	c.JSON(200, status)
}

func (h *OTPHandler) RequestOTP(c *gin.Context) {
	userID := c.GetString("userID")
	var req models.OTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Phone number is required in E.164 format (e.g. +14155551234)."))
		return
	}
	if err := h.otp.RequestOTP(userID, req.Phone); err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidPhone):
			WriteError(c, middleware.ErrValidation("Invalid phone number. Use E.164 format (e.g. +14155551234)."))
		case errors.Is(err, services.ErrRateLimited):
			WriteError(c, middleware.ErrRateLimit("Too many OTP requests. Try again later."))
		default:
			WriteError(c, middleware.ErrInternal("Failed to send OTP"))
		}
		return
	}
	masked := services.MaskPhone(services.NormalizePhone(req.Phone))
	c.JSON(200, gin.H{"message": "OTP sent via WhatsApp", "phoneMasked": masked})
}

func (h *OTPHandler) VerifyPhone(c *gin.Context) {
	userID := c.GetString("userID")
	var req models.OTPVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Phone and 6-digit code are required."))
		return
	}
	if err := h.otp.VerifyOTP(userID, req.Phone, req.Code); err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidPhone):
			WriteError(c, middleware.ErrValidation("Invalid phone number."))
		case errors.Is(err, services.ErrInvalidOTP):
			WriteError(c, middleware.ErrValidation("Invalid or expired code."))
		case errors.Is(err, services.ErrTooManyAttempts):
			WriteError(c, middleware.ErrRateLimit("Too many attempts. Request a new code."))
		case errors.Is(err, services.ErrOTPExpired):
			WriteError(c, middleware.ErrValidation("Code has expired. Request a new one."))
		default:
			WriteError(c, middleware.ErrInternal("Verification failed"))
		}
		return
	}
	status, _ := h.otp.GetStatus(userID)
	c.JSON(200, gin.H{"message": "Phone verified", "status": status})
}

func (h *OTPHandler) SetTwoFactor(c *gin.Context) {
	userID := c.GetString("userID")
	var req models.TwoFASetupRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		WriteError(c, middleware.ErrValidation("Field 'enabled' (boolean) is required."))
		return
	}
	if err := h.otp.SetTwoFactor(userID, *req.Enabled); err != nil {
		if errors.Is(err, services.ErrPhoneNotVerified) {
			WriteError(c, middleware.ErrValidation("Verify your phone number before enabling 2FA."))
			return
		}
		WriteError(c, middleware.ErrInternal("Failed to update 2FA setting"))
		return
	}
	status, _ := h.otp.GetStatus(userID)
	c.JSON(200, status)
}

func (h *OTPHandler) Verify2FA(c *gin.Context) {
	var req models.TwoFAVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("tempToken and 6-digit code are required."))
		return
	}
	userID, err := h.auth.Validate2FATempToken(req.TempToken)
	if err != nil {
		WriteError(c, middleware.ErrAuth("Invalid or expired 2FA token"))
		return
	}
	if err := h.otp.VerifyLoginOTP(userID, req.Code); err != nil {
		switch {
		case errors.Is(err, services.ErrPhoneNotVerified):
			WriteError(c, middleware.ErrValidation("Phone not verified"))
		case errors.Is(err, services.ErrInvalidOTP), errors.Is(err, services.ErrOTPExpired):
			WriteError(c, middleware.ErrAuth("Invalid or expired code"))
		case errors.Is(err, services.ErrTooManyAttempts):
			WriteError(c, middleware.ErrRateLimit("Too many attempts"))
		default:
			WriteError(c, middleware.ErrAuth("Invalid code"))
		}
		return
	}
	user, err := h.user.GetByID(userID)
	if err != nil {
		WriteError(c, middleware.ErrNotFound("User not found"))
		return
	}
	accessToken, err := h.auth.GenerateAccessToken(user.ID)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to generate access token"))
		return
	}
	refreshToken, err := h.auth.GenerateRefreshToken(user.ID)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to generate refresh token"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user, "tokens": models.AuthTokens{AccessToken: accessToken, RefreshToken: refreshToken, ExpiresIn: 86400}})
}
