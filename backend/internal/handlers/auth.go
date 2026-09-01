package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService        *services.AuthService
	userService        *services.UserService
	invitationService  *services.InvitationService
	emailSender        services.EmailSender
	entitlementService *services.EntitlementService
	moderation         *services.ModerationService
	otpService         *services.OTPService
	resetBaseURL       string
}

func (h *AuthHandler) SetOTPService(s *services.OTPService) { h.otpService = s }

func NewAuthHandler(authService *services.AuthService, userService *services.UserService, invitationService *services.InvitationService, emailSender services.EmailSender, entitlementService *services.EntitlementService, moderation *services.ModerationService, resetBaseURL string) *AuthHandler {
	return &AuthHandler{
		authService:        authService,
		userService:        userService,
		invitationService:  invitationService,
		emailSender:        emailSender,
		entitlementService: entitlementService,
		moderation:         moderation,
		resetBaseURL:       strings.TrimRight(resetBaseURL, "?"),
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid registration data. Check email format and required fields."))
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if strings.TrimSpace(req.Username) == "" {
		req.Username = req.Email
	}
	if strings.TrimSpace(req.DisplayName) == "" {
		// Prefer the composed first + last name, then fall back to the email
		// prefix so a registration without any name still yields a displayName.
		req.DisplayName = services.ComposeDisplayName(req.FirstName, req.LastName)
	}
	if strings.TrimSpace(req.DisplayName) == "" {
		parts := strings.Split(req.Email, "@")
		req.DisplayName = parts[0]
	}
	if strings.TrimSpace(req.NativeLanguage) == "" {
		req.NativeLanguage = "en"
	}
	var user *models.User
	var err error
	if h.invitationService != nil {
		user, err = h.authService.RegisterWithInvitation(req)
	} else {
		user, err = h.authService.Register(req)
	}
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidInvitation):
			WriteError(c, middleware.ErrForbidden("A valid invitation for this email is required."))
		case errors.Is(err, services.ErrEmailAlreadyRegistered):
			WriteError(c, middleware.ErrConflict("Email is already registered"))
		case errors.Is(err, services.ErrUsernameAlreadyRegistered):
			WriteError(c, middleware.ErrConflict("Username is already registered"))
		default:
			WriteError(c, middleware.ErrValidation("Registration failed. Please try again."))
		}
		return
	}

	accessToken, err := h.authService.GenerateAccessToken(user.ID)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to generate access token"))
		return
	}

	refreshToken, err := h.authService.GenerateRefreshToken(user.ID)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to generate refresh token"))
		return
	}

	// Best-effort welcome email; never fail registration because of it.
	if h.emailSender != nil {
		subject, html := services.RegistrationWelcomeEmail(user.DisplayName)
		if err := h.emailSender.Send(user.Email, subject, html); err != nil {
			log.Printf("Failed to send registration welcome email to %s: %v", user.Email, err)
		}
	}

	c.JSON(201, gin.H{
		"user": user,
		"tokens": models.AuthTokens{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    86400, // 24 hours
		},
	})
}

// InviteInfo resolves the email bound to a valid (unexpired, unredeemed)
// invitation token so the register page can prefill it for the user.
func (h *AuthHandler) InviteInfo(c *gin.Context) {
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		WriteError(c, middleware.ErrValidation("Invitation token is required."))
		return
	}
	if h.invitationService == nil {
		WriteError(c, middleware.ErrForbidden("Invitations are not enabled."))
		return
	}
	email, err := h.invitationService.EmailByToken(token)
	if err != nil {
		WriteError(c, middleware.ErrForbidden("This invitation is invalid, expired, or already used."))
		return
	}
	c.JSON(http.StatusOK, gin.H{"email": email})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid request"))
		return
	}

	user, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		WriteError(c, middleware.ErrAuth("Invalid credentials"))
		return
	}

	if user.TwoFactorEnabled && user.PhoneVerified {
		tempToken, err := h.authService.Generate2FATempToken(user.ID)
		if err != nil {
			WriteError(c, middleware.ErrInternal("Failed to generate 2FA token"))
			return
		}
		masked := ""
		if user.Phone != nil {
			masked = services.MaskPhone(*user.Phone)
		}
		c.JSON(200, gin.H{"requires2FA": true, "tempToken": tempToken, "phoneMasked": masked})
		return
	}

	accessToken, err := h.authService.GenerateAccessToken(user.ID)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to generate access token"))
		return
	}

	refreshToken, err := h.authService.GenerateRefreshToken(user.ID)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to generate refresh token"))
		return
	}

	c.JSON(200, gin.H{
		"user": user,
		"tokens": models.AuthTokens{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    86400,
		},
	})
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req models.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Please provide a valid email address."))
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// Respond identically whether or not the account exists to avoid leaking
	// which emails are registered.
	respond := func() {
		c.JSON(200, gin.H{"message": "If an account exists for that email, a password reset link has been sent."})
	}

	token, err := h.authService.CreatePasswordResetToken(req.Email)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			respond()
			return
		}
		log.Printf("Failed to create password reset token for %s: %v", req.Email, err)
		respond()
		return
	}

	link := h.resetBaseURL + "?token=" + token
	subject, html := services.PasswordResetEmail(link)
	if h.emailSender == nil {
		log.Printf("Password reset email for %s not sent: SMTP sender not configured", req.Email)
		respond()
		return
	}
	if err := h.emailSender.Send(req.Email, subject, html); err != nil {
		log.Printf("Failed to send password reset email to %s: %v", req.Email, err)
	}
	respond()
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req models.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid request. A new password of at least 8 characters is required."))
		return
	}

	userID, err := h.authService.ResetPassword(req.Token, req.Password)
	if err != nil {
		WriteError(c, middleware.ErrValidation("This reset link is invalid or has expired."))
		return
	}
	if err := h.authService.DeleteUserRefreshTokens(userID); err != nil {
		log.Printf("Failed to revoke refresh tokens for user %s after password reset: %v", userID, err)
	}

	c.JSON(200, gin.H{"message": "Your password has been reset. Please log in with your new password."})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid request"))
		return
	}

	userID, err := h.authService.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		WriteError(c, middleware.ErrAuth("Invalid refresh token"))
		return
	}

	accessToken, err := h.authService.GenerateAccessToken(userID)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to generate access token"))
		return
	}

	c.JSON(200, gin.H{
		"accessToken": accessToken,
		"expiresIn":   86400,
	})
}

func (h *AuthHandler) GetMe(c *gin.Context) {
	userID := c.GetString("userID")

	user, err := h.userService.GetByID(userID)
	if err != nil {
		WriteError(c, middleware.ErrNotFound("User not found"))
		return
	}

	c.JSON(200, user)
}

// GetMyEntitlements returns the fully-resolved entitlements for the current
// user: effective plan, grace deadline, quotas, and whether monetization
// surfaces (upsell, ads) apply. Phase 0 resolves only; nothing is gated yet.
func (h *AuthHandler) GetMyEntitlements(c *gin.Context) {
	userID := c.GetString("userID")

	user, err := h.userService.GetByID(userID)
	if err != nil {
		WriteError(c, middleware.ErrNotFound("User not found"))
		return
	}

	entitlements := h.entitlementService.ResolveNow(user)
	c.JSON(200, entitlements)
}

// OnboardMe captures the user's first + last name and composes the displayName
// (first + last) unless the request supplies an explicit override. It is the
// backup for the post-registration onboarding step (REQ 2.1).
func (h *AuthHandler) OnboardMe(c *gin.Context) {
	userID := c.GetString("userID")

	var req models.OnboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("First and last name are required."))
		return
	}

	user, err := h.userService.Onboard(userID, req.FirstName, req.LastName, req.DisplayName)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to update your name"))
		return
	}

	c.JSON(200, user)
}

func (h *AuthHandler) UpdateMe(c *gin.Context) {
	userID := c.GetString("userID")

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid request"))
		return
	}

	user, err := h.userService.Update(userID, req)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to update user"))
		return
	}

	c.JSON(200, user)
}

func (h *AuthHandler) SearchUsers(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		WriteError(c, middleware.ErrValidation("Query parameter 'q' is required"))
		return
	}

	limit := 10
	users, err := h.userService.Search(query, limit)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Search failed"))
		return
	}

	// Surface block/report status (task 7.1): mark which results the caller has
	// blocked so the directory can render Block/Unblock everywhere.
	if h.moderation != nil && len(users) > 0 {
		ptrs := make([]*models.User, len(users))
		for i := range users {
			ptrs[i] = &users[i]
		}
		if err := h.moderation.EnrichUsers(c.Request.Context(), c.GetString("userID"), ptrs); err != nil {
			WriteError(c, middleware.ErrInternal("Search failed"))
			return
		}
	}

	c.JSON(200, gin.H{
		"users":   users,
		"total":   len(users),
		"hasMore": len(users) >= limit,
	})
}
