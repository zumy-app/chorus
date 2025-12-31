package handlers

import (
	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *services.AuthService
	userService *services.UserService
}

func NewAuthHandler(authService *services.AuthService, userService *services.UserService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		userService: userService,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	user, err := h.authService.Register(req)
	if err != nil {
		c.JSON(400, gin.H{"error": "Registration failed: " + err.Error()})
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

	c.JSON(201, gin.H{
		"user": user,
		"tokens": models.AuthTokens{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    86400, // 24 hours
		},
	})
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
