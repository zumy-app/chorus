package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

type AdminWaitlistHandler struct {
	db          *sql.DB
	users       *services.UserService
	invitations *services.InvitationService
	email       services.EmailSender
	adminEmails map[string]bool
	inviteURL   string
}

func NewAdminWaitlistHandler(db *sql.DB, users *services.UserService, invitations *services.InvitationService, email services.EmailSender, adminEmails []string, inviteURL string) *AdminWaitlistHandler {
	allowed := make(map[string]bool, len(adminEmails))
	for _, email := range adminEmails {
		allowed[strings.ToLower(strings.TrimSpace(email))] = true
	}
	return &AdminWaitlistHandler{db: db, users: users, invitations: invitations, email: email, adminEmails: allowed, inviteURL: strings.TrimRight(inviteURL, "?")}
}

func (h *AdminWaitlistHandler) authorize(c *gin.Context) bool {
	user, err := h.users.GetByID(c.GetString("userID"))
	if err != nil || !h.adminEmails[strings.ToLower(user.Email)] {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return false
	}
	return true
}

func (h *AdminWaitlistHandler) List(c *gin.Context) {
	if !h.authorize(c) {
		return
	}
	rows, err := h.db.Query(`SELECT id, email, spoken_language, target_languages, reasons, status, queue_position, created_at
		FROM waitlist_entries WHERE status = 'pending' ORDER BY queue_position`)
	if err != nil {
		c.JSON(500, gin.H{"error": "Unable to load waitlist"})
		return
	}
	defer rows.Close()
	entries := []models.WaitlistEntry{}
	for rows.Next() {
		var entry models.WaitlistEntry
		if err := rows.Scan(&entry.ID, &entry.Email, &entry.SpokenLanguage, pq.Array(&entry.TargetLanguages), pq.Array(&entry.Reasons), &entry.Status, &entry.QueuePosition, &entry.CreatedAt); err != nil {
			c.JSON(500, gin.H{"error": "Unable to load waitlist"})
			return
		}
		entries = append(entries, entry)
	}
	c.JSON(200, gin.H{"entries": entries})
}

func (h *AdminWaitlistHandler) Approve(c *gin.Context) {
	if !h.authorize(c) {
		return
	}
	var entry models.WaitlistEntry
	err := h.db.QueryRow(`UPDATE waitlist_entries SET status = 'approved', approved_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND status = 'pending' RETURNING id, email`, c.Param("id")).Scan(&entry.ID, &entry.Email)
	if err != nil {
		c.JSON(404, gin.H{"error": "Pending waitlist entry not found"})
		return
	}
	token, err := h.invitations.Create(entry.ID, entry.Email)
	if err != nil {
		c.JSON(500, gin.H{"error": "Unable to create invitation"})
		return
	}
	link := h.inviteURL + "?invite=" + token
	if err := h.email.Send(entry.Email, "Your Chorus invitation", "<p>You are off the waitlist.</p><p><a href=\""+link+"\">Create your account</a></p>"); err != nil {
		c.JSON(502, gin.H{"error": "Invitation created, but email delivery failed. Please retry."})
		return
	}
	c.JSON(200, gin.H{"message": "Invitation sent"})
}
