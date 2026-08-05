package services

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/chorus/messenger/internal/models"
	"github.com/lib/pq"
)

var ErrInvalidWaitlistRequest = errors.New("email, spoken language, target languages, and at least one reason are required")

func ValidateWaitlistRequest(req models.WaitlistRequest) error {
	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.SpokenLanguage) == "" ||
		len(req.TargetLanguages) == 0 || len(req.Reasons) == 0 {
		return ErrInvalidWaitlistRequest
	}
	return nil
}

type WaitlistService struct{ db *sql.DB }

func NewWaitlistService(db *sql.DB) *WaitlistService { return &WaitlistService{db: db} }

// Submit is idempotent per email and returns the original queue position.
func (s *WaitlistService) Submit(req models.WaitlistRequest) (*models.WaitlistEntry, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.SpokenLanguage = strings.TrimSpace(req.SpokenLanguage)
	if err := ValidateWaitlistRequest(req); err != nil {
		return nil, err
	}
	entry := &models.WaitlistEntry{}
	err := s.db.QueryRow(`
		INSERT INTO waitlist_entries (email, spoken_language, target_languages, reasons)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
		RETURNING id, email, spoken_language, target_languages, reasons, status, queue_position, created_at`,
		req.Email, req.SpokenLanguage, pq.Array(req.TargetLanguages), pq.Array(req.Reasons),
	).Scan(&entry.ID, &entry.Email, &entry.SpokenLanguage, pq.Array(&entry.TargetLanguages),
		pq.Array(&entry.Reasons), &entry.Status, &entry.QueuePosition, &entry.CreatedAt)
	if err != nil {
		return nil, err
	}
	return entry, nil
}
