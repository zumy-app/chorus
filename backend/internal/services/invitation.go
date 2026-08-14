package services

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"
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
