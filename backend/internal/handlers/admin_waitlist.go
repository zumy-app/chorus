package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

type AdminWaitlistHandler struct {
	db            *sql.DB
	users         *services.UserService
	invitations   *services.InvitationService
	notifications *services.NotificationService
	adminEmails   map[string]bool
	inviteURL     string
}

func NewAdminWaitlistHandler(db *sql.DB, users *services.UserService, invitations *services.InvitationService, notifications *services.NotificationService, adminEmails []string, inviteURL string) *AdminWaitlistHandler {
	allowed := make(map[string]bool, len(adminEmails))
	for _, email := range adminEmails {
		allowed[strings.ToLower(strings.TrimSpace(email))] = true
	}
	return &AdminWaitlistHandler{db: db, users: users, invitations: invitations, notifications: notifications, adminEmails: allowed, inviteURL: strings.TrimRight(inviteURL, "?")}
}

func (h *AdminWaitlistHandler) authorize(c *gin.Context) bool {
	user, err := h.users.GetByID(c.GetString("userID"))
	if err != nil || !h.adminEmails[strings.ToLower(user.Email)] {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return false
	}
	return true
}

// Status reports whether the authenticated user is an admin. Unlike authorize
// it never 403s — the response body is authoritative.
func (h *AdminWaitlistHandler) Status(c *gin.Context) {
	user, err := h.users.GetByID(c.GetString("userID"))
	isAdmin := false
	if err == nil {
		isAdmin = h.adminEmails[strings.ToLower(user.Email)]
	}
	c.JSON(http.StatusOK, gin.H{"isAdmin": isAdmin})
}

// List returns waitlist entries. Query params: status (pending|approved|declined|all,
// default pending) and q (substring match on email).
func (h *AdminWaitlistHandler) List(c *gin.Context) {
	if !h.authorize(c) {
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	if status == "" {
		status = "pending"
	}
	q := strings.TrimSpace(c.Query("q"))

	query := `SELECT id, email, spoken_languages, target_languages, reasons, comments, status, queue_position, created_at, approved_at
		FROM waitlist_entries WHERE 1=1`
	args := []interface{}{}
	if status != "all" {
		args = append(args, status)
		query += fmt.Sprintf(` AND status = $%d`, len(args))
	}
	if q != "" {
		args = append(args, "%"+strings.ToLower(q)+"%")
		query += fmt.Sprintf(` AND LOWER(email) LIKE $%d`, len(args))
	}
	query += ` ORDER BY queue_position`

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(500, gin.H{"error": "Unable to load waitlist"})
		return
	}
	defer rows.Close()
	entries := []models.WaitlistEntry{}
	for rows.Next() {
		var entry models.WaitlistEntry
		if err := rows.Scan(&entry.ID, &entry.Email, pq.Array(&entry.SpokenLanguages), pq.Array(&entry.TargetLanguages), pq.Array(&entry.Reasons), &entry.Comments, &entry.Status, &entry.QueuePosition, &entry.CreatedAt, &entry.ApprovedAt); err != nil {
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
	subject, html := services.InvitationEmail(link)
	if err := h.notifications.Send(entry.Email, subject, html); err != nil {
		c.JSON(502, gin.H{"error": "Invitation created, but email delivery is pending retry. Check the email log."})
		return
	}
	c.JSON(200, gin.H{"message": "Invitation sent"})
}

// Decline rejects a pending waitlist entry. Approved entries are not affected.
func (h *AdminWaitlistHandler) Decline(c *gin.Context) {
	if !h.authorize(c) {
		return
	}
	result, err := h.db.Exec(`UPDATE waitlist_entries SET status = 'declined', approved_at = NULL
		WHERE id = $1 AND status = 'pending'`, c.Param("id"))
	if err != nil {
		c.JSON(500, gin.H{"error": "Unable to decline entry"})
		return
	}
	if count, _ := result.RowsAffected(); count != 1 {
		c.JSON(404, gin.H{"error": "Pending waitlist entry not found"})
		return
	}
	c.JSON(200, gin.H{"message": "Entry declined"})
}

// ResendInvite issues a brand-new invitation for an already-approved entry and
// emails it again. Earlier unused invitations are automatically invalidated by
// the invitation flow (only the newest token for the email can be redeemed).
func (h *AdminWaitlistHandler) ResendInvite(c *gin.Context) {
	if !h.authorize(c) {
		return
	}
	var entryID, email string
	err := h.db.QueryRow(`SELECT id, email FROM waitlist_entries WHERE id = $1 AND status = 'approved'`, c.Param("id")).
		Scan(&entryID, &email)
	if err != nil {
		c.JSON(404, gin.H{"error": "Approved waitlist entry not found"})
		return
	}
	token, err := h.invitations.Create(entryID, email)
	if err != nil {
		c.JSON(500, gin.H{"error": "Unable to create invitation"})
		return
	}
	link := h.inviteURL + "?invite=" + token
	subject, html := services.InvitationEmail(link)
	if err := h.notifications.Send(email, subject, html); err != nil {
		c.JSON(502, gin.H{"error": "Invitation created, but email delivery is pending retry. Check the email log."})
		return
	}
	c.JSON(200, gin.H{"message": "Invitation re-sent"})
}

// Stats aggregates user, waitlist and email-delivery numbers for the admin UI.
func (h *AdminWaitlistHandler) Stats(c *gin.Context) {
	if !h.authorize(c) {
		return
	}
	s := models.AdminStats{}
	count := func(query string, dest *int) {
		h.db.QueryRow(query).Scan(dest)
	}
	count(`SELECT COUNT(*) FROM users`, &s.TotalUsers)
	count(`SELECT COUNT(*) FROM waitlist_entries WHERE status = 'pending'`, &s.WaitlistPending)
	count(`SELECT COUNT(*) FROM waitlist_entries WHERE status = 'approved'`, &s.WaitlistApproved)
	count(`SELECT COUNT(*) FROM waitlist_entries WHERE status = 'declined'`, &s.WaitlistDeclined)
	count(`SELECT COUNT(*) FROM email_outbox WHERE status = 'pending'`, &s.EmailsPending)
	count(`SELECT COUNT(*) FROM email_outbox WHERE status = 'sent'`, &s.EmailsSent)
	count(`SELECT COUNT(*) FROM email_outbox WHERE status = 'failed'`, &s.EmailsFailed)
	c.JSON(200, s)
}

// Emails lists the durable email outbox so admins can audit delivery.
func (h *AdminWaitlistHandler) Emails(c *gin.Context) {
	if !h.authorize(c) {
		return
	}
	entries, err := h.notifications.List(c.Query("status"))
	if err != nil {
		c.JSON(500, gin.H{"error": "Unable to load email log"})
		return
	}
	c.JSON(200, gin.H{"emails": entries})
}

// RetryEmail forces an immediate re-send of a pending or failed email.
func (h *AdminWaitlistHandler) RetryEmail(c *gin.Context) {
	if !h.authorize(c) {
		return
	}
	if err := h.notifications.RetryByID(c.Param("id")); err != nil {
		c.JSON(502, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "Email sent"})
}
