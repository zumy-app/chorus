package services

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/chorus/messenger/internal/models"
	"github.com/lib/pq"
)

var ErrInvalidWaitlistRequest = errors.New("email, spoken languages, and at least one reason are required")

func ValidateWaitlistRequest(req models.WaitlistRequest) error {
	if strings.TrimSpace(req.Email) == "" || len(req.SpokenLanguages) == 0 || len(req.Reasons) == 0 {
		return ErrInvalidWaitlistRequest
	}
	return nil
}

type WaitlistService struct{ db *sql.DB }

func NewWaitlistService(db *sql.DB) *WaitlistService { return &WaitlistService{db: db} }

// Submit is idempotent per email. Re-submitting the same email updates the
// stored preferences and returns the original queue position, along with
// alreadyJoined=true so callers can tell the user their prefs were refreshed.
func (s *WaitlistService) Submit(req models.WaitlistRequest) (*models.WaitlistEntry, bool, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if err := ValidateWaitlistRequest(req); err != nil {
		return nil, false, err
	}
	entry := &models.WaitlistEntry{}
	var inserted bool
	err := s.db.QueryRow(`
		INSERT INTO waitlist_entries (email, spoken_languages, target_languages, reasons, comments)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (email) DO UPDATE SET
			spoken_languages = EXCLUDED.spoken_languages,
			target_languages = EXCLUDED.target_languages,
			reasons = EXCLUDED.reasons,
			comments = EXCLUDED.comments
		RETURNING (xmax = 0) AS inserted, id, email, spoken_languages, target_languages, reasons, comments, status, queue_position, created_at`,
		req.Email, pq.Array(req.SpokenLanguages), pq.Array(req.TargetLanguages), pq.Array(req.Reasons), req.Comments,
	).Scan(&inserted, &entry.ID, &entry.Email, pq.Array(&entry.SpokenLanguages), pq.Array(&entry.TargetLanguages),
		pq.Array(&entry.Reasons), &entry.Comments, &entry.Status, &entry.QueuePosition, &entry.CreatedAt)
	if err != nil {
		return nil, false, err
	}
	return entry, !inserted, nil
}
