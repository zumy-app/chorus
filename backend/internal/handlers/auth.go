package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"

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
	resetBaseURL       string
}

func NewAuthHandler(authService *services.AuthService, userService *services.UserService, invitationService *services.InvitationService, emailSender services.EmailSender, entitlementService *services.EntitlementService, resetBaseURL string) *AuthHandler {
	return &AuthHandler{
		authService:        authService,
		userService:        userService,
		invitationService:  invitationService,
		emailSender:        emailSender,
		entitlementService: entitlementService,
		resetBaseURL:       strings.TrimRight(resetBaseURL, "?"),
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid registration data. Check email format and required fields."})
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if strings.TrimSpace(req.Username) == "" {
		req.Username = req.Email
	}
	if strings.TrimSpace(req.DisplayName) == "" {
		// Use email prefix as display name
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
			c.JSON(403, gin.H{"error": "A valid invitation for this email is required."})
		case errors.Is(err, services.ErrEmailAlreadyRegistered):
			c.JSON(409, gin.H{"error": "Email is already registered"})
		case errors.Is(err, services.ErrUsernameAlreadyRegistered):
			c.JSON(409, gin.H{"error": "Username is already registered"})
		default:
			c.JSON(400, gin.H{"error": "Registration failed. Please try again."})
		}
		return
	}

	accessToken, err := h.authService.GenerateAccessToken(user.ID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to generate access token"})
		return
	}

	refreshToken, err := h.authService.GenerateRefreshToken(user.ID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to generate refresh token"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invitation token is required."})
		return
	}
	if h.invitationService == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Invitations are not enabled."})
		return
	}
	email, err := h.invitationService.EmailByToken(token)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "This invitation is invalid, expired, or already used."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"email": email})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	user, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(401, gin.H{"error": "Invalid credentials"})
		return
	}

	accessToken, err := h.authService.GenerateAccessToken(user.ID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to generate access token"})
		return
	}

	refreshToken, err := h.authService.GenerateRefreshToken(user.ID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to generate refresh token"})
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
		c.JSON(400, gin.H{"error": "Please provide a valid email address."})
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
		c.JSON(400, gin.H{"error": "Invalid request. A new password of at least 8 characters is required."})
		return
	}

	userID, err := h.authService.ResetPassword(req.Token, req.Password)
	if err != nil {
		c.JSON(400, gin.H{"error": "This reset link is invalid or has expired."})
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
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	userID, err := h.authService.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(401, gin.H{"error": "Invalid refresh token"})
		return
	}

	accessToken, err := h.authService.GenerateAccessToken(userID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to generate access token"})
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
		c.JSON(404, gin.H{"error": "User not found"})
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
		c.JSON(404, gin.H{"error": "User not found"})
		return
	}

	entitlements := h.entitlementService.ResolveNow(user)
	c.JSON(200, entitlements)
}

func (h *AuthHandler) UpdateMe(c *gin.Context) {
	userID := c.GetString("userID")

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	user, err := h.userService.Update(userID, req)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(200, user)
}

// DeleteMe performs a privacy-preserving account deletion. The account is
// soft-deleted so existing conversations retain their structure, while all
// sessions are revoked and the user can no longer authenticate or be found.
func (h *AuthHandler) DeleteMe(c *gin.Context) {
	userID := c.GetString("userID")
	if err := h.userService.SetDeleted(userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err := h.authService.DeleteUserRefreshTokens(userID); err != nil {
		log.Printf("Failed to revoke refresh tokens for deleted user %s: %v", userID, err)
	}
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) SearchUsers(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(400, gin.H{"error": "Query parameter 'q' is required"})
		return
	}

	limit := 10
	users, err := h.userService.Search(query, limit)
	if err != nil {
		c.JSON(500, gin.H{"error": "Search failed"})
		return
	}

	c.JSON(200, gin.H{
		"users":   users,
		"total":   len(users),
		"hasMore": len(users) >= limit,
	})
}
