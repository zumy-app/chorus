package middleware

import (
	"strings"

	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates the Bearer token, loads the user, and blocks
// suspended/deleted accounts. It sets userID and userRole in the context so
// downstream handlers and the RequireRole middleware can use them without a
// second DB round-trip.
func AuthMiddleware(authService *services.AuthService, userService *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			WriteError(c, ErrAuth("Authorization header required"))
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			WriteError(c, ErrAuth("Invalid authorization header format"))
			return
		}

		token := parts[1]
		userID, err := authService.ValidateAccessToken(token)
		if err != nil {
			WriteError(c, ErrAuth("Invalid or expired token"))
			return
		}

		user, err := userService.GetByID(userID)
		if err != nil {
			WriteError(c, ErrAuth("User not found"))
			return
		}
		// Suspended/deleted accounts are blocked immediately, revoking access
		// even before their JWT expires.
		if user.SuspendedAt != nil || user.DeletedAt != nil {
			WriteError(c, ErrForbidden("Account is disabled"))
			return
		}

		role := user.Role
		if !services.ValidRole(role) {
			role = services.RoleMember
		}

		c.Set("userID", userID)
		c.Set("userRole", role)
		c.Next()
	}
}

// RequireRole gates a handler to users whose role is at least the given role.
// It reads userRole (set by AuthMiddleware) instead of re-querying the DB.
func RequireRole(minRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("userRole")
		roleStr, _ := role.(string)
		if !services.RoleAtLeast(roleStr, minRole) {
			WriteError(c, ErrForbidden("You do not have permission to perform this action"))
			return
		}
		c.Next()
	}
}
