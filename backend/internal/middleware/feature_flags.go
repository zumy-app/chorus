package middleware

import (
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequireFlag returns a Gin middleware that rejects the request with 403 unless
// the resolved feature flag is enabled for the authenticated user.
func RequireFlag(svc *services.FeatureFlagService, key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := c.Get("userID")
		if !ok {
			WriteError(c, ErrAuth("authentication required"))
			c.Abort()
			return
		}

		uid, err := uuid.Parse(userID.(string))
		if err != nil {
			WriteError(c, ErrAuth("invalid user id"))
			c.Abort()
			return
		}

		role, _ := c.Get("userRole")
		roleStr, _ := role.(string)
		isAdmin := roleStr == services.RoleAdmin

		betaAccess, _ := c.Get("userBetaAccess")
		isBeta, _ := betaAccess.(bool)

		enabled, err := svc.IsEnabled(c.Request.Context(), uid, key, isAdmin, isBeta || isAdmin)
		if err != nil {
			WriteError(c, NewError(KindInternal, "failed to resolve feature flag"))
			c.Abort()
			return
		}
		if !enabled {
			WriteError(c, ErrForbidden("feature not enabled"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// FlagContextMiddleware injects the resolved flag map into the Gin context so
// handlers can read it without a second DB round-trip. It is optional; most
// handlers should use RequireFlag on routes instead.
func FlagContextMiddleware(svc *services.FeatureFlagService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := c.Get("userID")
		if !ok {
			c.Next()
			return
		}

		uid, err := uuid.Parse(userID.(string))
		if err != nil {
			c.Next()
			return
		}

		role, _ := c.Get("userRole")
		roleStr, _ := role.(string)
		isAdmin := roleStr == services.RoleAdmin

		betaAccess, _ := c.Get("userBetaAccess")
		isBeta, _ := betaAccess.(bool)

		flags, err := svc.ResolveForUser(c.Request.Context(), uid, isAdmin, isBeta || isAdmin)
		if err == nil {
			c.Set("featureFlags", flags)
		}

		c.Next()
	}
}

// FeatureFlagFromContext reads a single flag from the context map injected by
// FlagContextMiddleware. Returns false if the map is missing or the key is unknown.
func FeatureFlagFromContext(c *gin.Context, key string) bool {
	val, ok := c.Get("featureFlags")
	if !ok {
		return false
	}
	flags, ok := val.(map[string]bool)
	if !ok {
		return false
	}
	enabled, ok := flags[key]
	return ok && enabled
}
