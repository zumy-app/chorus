package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

// AdminUsersHandler exposes the user directory and role/account management to
// moderators and admins. Route-level guards (middleware.RequireRole) enforce the
// minimum role per action; this handler only enforces admin-only invariants such
// as protecting other admins from demotion/deletion.
type AdminUsersHandler struct {
	users *services.UserService
	auth  *services.AuthService
}

func NewAdminUsersHandler(users *services.UserService, auth *services.AuthService) *AdminUsersHandler {
	return &AdminUsersHandler{users: users, auth: auth}
}

// List returns a paginated, filterable user directory.
// Query params: q, role (member|moderator|admin), status (active|suspended),
// limit (default 50, max 200), offset.
func (h *AdminUsersHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	users, err := h.users.ListUsers(c.Query("q"), c.Query("role"), c.Query("status"), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to load users"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users, "total": len(users)})
}

// SetRole assigns a role to a user. Admin-only. Admins cannot change their own
// role (prevents accidental self-lockout) nor the role of another admin.
func (h *AdminUsersHandler) SetRole(c *gin.Context) {
	actorID := c.GetString("userID")
	targetID := c.Param("id")
	if actorID == targetID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You cannot change your own role"})
		return
	}
	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	target, err := h.users.GetByID(targetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if target.Role == services.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You cannot change the role of another admin"})
		return
	}
	if err := h.users.SetRole(targetID, req.Role); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Role updated", "role": req.Role})
}

// Suspend soft-bans a user (blocks authentication). Moderator+.
func (h *AdminUsersHandler) Suspend(c *gin.Context) {
	if !h.canModerateTarget(c) {
		return
	}
	if err := h.users.SetSuspended(c.Param("id"), true); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	// Revoke their existing refresh tokens so sessions end immediately.
	h.auth.DeleteUserRefreshTokens(c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"message": "User suspended"})
}

// Unsuspend lifts a soft-ban. Moderator+.
func (h *AdminUsersHandler) Unsuspend(c *gin.Context) {
	if !h.canModerateTarget(c) {
		return
	}
	if err := h.users.SetSuspended(c.Param("id"), false); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User unsuspended"})
}

// Delete soft-deletes a user account (permanent block, history retained).
// Admin-only; admins cannot delete themselves or other admins.
func (h *AdminUsersHandler) Delete(c *gin.Context) {
	actorID := c.GetString("userID")
	targetID := c.Param("id")
	if actorID == targetID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You cannot delete your own account"})
		return
	}
	target, err := h.users.GetByID(targetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if target.Role == services.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You cannot delete another admin"})
		return
	}
	if err := h.users.SetDeleted(targetID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	h.auth.DeleteUserRefreshTokens(targetID)
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

// canModerateTarget blocks moderating admins or the caller themself.
func (h *AdminUsersHandler) canModerateTarget(c *gin.Context) bool {
	actorID := c.GetString("userID")
	targetID := c.Param("id")
	if actorID == targetID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You cannot moderate your own account"})
		return false
	}
	target, err := h.users.GetByID(targetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return false
	}
	if target.Role == services.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You cannot moderate an admin"})
		return false
	}
	return true
}
