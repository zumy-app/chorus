package services

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/chorus/messenger/internal/models"
)

var ErrInvalidInvitation = errors.New("invitation is invalid, expired, or already used")

type InvitationService struct {
	db  *sql.DB
	ttl time.Duration
}

func NewInvitationService(db *sql.DB, ttl time.Duration) *InvitationService {
	return &InvitationService{db: db, ttl: ttl}
}

func invitationHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *InvitationService) Create(entryID, email string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := hex.EncodeToString(raw)
	// Compute expires_at in SQL (CURRENT_TIMESTAMP) so it stays consistent with
	// the validity check no matter what timezone the backend runs in.
	_, err := s.db.Exec(`INSERT INTO invitations (waitlist_entry_id, email, token_hash, expires_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP + $4 * INTERVAL '1 hour')`,
		entryID, strings.ToLower(strings.TrimSpace(email)),
		invitationHash(token), int(s.ttl.Hours()))
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *InvitationService) Validate(token, email string) error {
	var invitationEmail string
	var expiresAt time.Time
	var redeemedAt sql.NullTime
	err := s.db.QueryRow(`SELECT email, expires_at, redeemed_at FROM invitations WHERE token_hash = $1`,
		invitationHash(token)).Scan(&invitationEmail, &expiresAt, &redeemedAt)
	if err != nil || redeemedAt.Valid || time.Now().After(expiresAt) ||
		!strings.EqualFold(strings.TrimSpace(email), invitationEmail) {
		return ErrInvalidInvitation
	}
	return nil
}

// EmailByToken returns the invite-bound email for a token that is still valid
// (unredeemed and unexpired). It lets the register page prefill the email
// address instead of asking the user to type it again.
func (s *InvitationService) EmailByToken(token string) (string, error) {
	var email string
	err := s.db.QueryRow(`SELECT email FROM invitations
		WHERE token_hash = $1 AND redeemed_at IS NULL AND expires_at > CURRENT_TIMESTAMP`,
		invitationHash(token)).Scan(&email)
	if err != nil {
		return "", ErrInvalidInvitation
	}
	return email, nil
}

func (s *InvitationService) Redeem(token string) error {
	result, err := s.db.Exec(`UPDATE invitations SET redeemed_at = CURRENT_TIMESTAMP
		WHERE token_hash = $1 AND redeemed_at IS NULL AND expires_at > CURRENT_TIMESTAMP`, invitationHash(token))
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return ErrInvalidInvitation
	}
	return nil
}

// CreateForContact issues a single-use, expiring invite from a registered user
// to an off-platform contact (REQ 2.4 / FR-22-23). Unlike the admin/waitlist
// flow, the invite is not tied to a waitlist entry. channel controls the
// delivery surface: 'email' is dispatched durably; 'sms'/'whatsapp' return a
// shareable link for the client to hand off through the device.
//
// recipient is the delivery target (email address for 'email', phone for
// sms/whatsapp). email is the address the invite may be redeemed against; when
// empty the invite is "open" and may be redeemed with any email (typical for an
// SMS/WhatsApp contact that has no email on file).
func (s *InvitationService) CreateForContact(inviterID, channel, recipient, email, name string) (string, *models.ContactInvite, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	token := hex.EncodeToString(raw)

	// Bind the invite to an email when we have one; otherwise leave it open.
	bindEmail := strings.ToLower(strings.TrimSpace(email))
	if channel == "email" {
		bindEmail = strings.ToLower(strings.TrimSpace(recipient))
	}

	status := "pending"
	if channel == "email" {
		status = "sent"
	}

	var id string
	var expiry time.Time
	err := s.db.QueryRow(`INSERT INTO invitations
		(waitlist_entry_id, inviter_user_id, email, token_hash, expires_at, channel, recipient, name, status, sent_at)
		VALUES (NULL, $1, $2, $3, CURRENT_TIMESTAMP + $4 * INTERVAL '1 hour', $5, $6, $7, $8,
			CASE WHEN $5 = 'email' THEN CURRENT_TIMESTAMP ELSE NULL END)
		RETURNING id, expires_at`,
		inviterID, bindEmail, invitationHash(token), int(s.ttl.Hours()), channel,
		strings.TrimSpace(recipient), strings.TrimSpace(name), status,
	).Scan(&id, &expiry)
	if err != nil {
		return "", nil, err
	}

	return token, &models.ContactInvite{
		ID:        id,
		InviterID: inviterID,
		Channel:   channel,
		Recipient: strings.TrimSpace(recipient),
		Name:      strings.TrimSpace(name),
		Token:     token,
		Status:    status,
		ExpiresAt: expiry,
		CreatedAt: time.Now(),
	}, nil
}

// ListForInviter returns the invites a user has sent, newest first, with a
// computed status (redeemed / expired / pending / sent). Tokens are stored
// hashed and are NOT re-exposed via this endpoint — inviters who need to resend
// simply create a fresh invite.
func (s *InvitationService) ListForInviter(inviterID string, limit, offset int) ([]models.ContactInvite, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := s.db.Query(`
		SELECT id, inviter_user_id, channel, recipient, COALESCE(name, ''),
			CASE
				WHEN redeemed_at IS NOT NULL THEN 'redeemed'
				WHEN expires_at < CURRENT_TIMESTAMP THEN 'expired'
				ELSE status
			END,
			expires_at, created_at, redeemed_at
		FROM invitations
		WHERE inviter_user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, inviterID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	invites := []models.ContactInvite{}
	for rows.Next() {
		var inv models.ContactInvite
		if err := rows.Scan(&inv.ID, &inv.InviterID, &inv.Channel, &inv.Recipient, &inv.Name,
			&inv.Status, &inv.ExpiresAt, &inv.CreatedAt, &inv.RedeemedAt); err != nil {
			return nil, err
		}
		invites = append(invites, inv)
	}
	return invites, rows.Err()
}
