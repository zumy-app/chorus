package services

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Deterministic development fixtures (rescue plan A3/D3). These accounts back
// the acceptance suite (docs/TEST_SPEC.md) and manual emulator walkthroughs.
// Development only — production environments never run SeedDevData.
const (
	DevPassword      = "ChorusDev123!"
	DevLearnerEmail  = "alice.dev@chorus.test"
	DevLearner2Email = "bob.dev@chorus.test"
	DevTutorEmail    = "sofia.tutor@chorus.test"
	DevInviteEmail   = "invite.dev@chorus.test"
	DevInviteToken   = "chorus-dev-invite-2026"
)

// SeedDevData provisions (or resets) the deterministic dev fixtures:
// two learners, one approved tutor with marketplace data (application,
// certificate, availability, reviews, trial credit), and an invitation with a
// known token for testing the invite-gated registration path. Idempotent:
// fixture accounts are deleted and recreated so every run yields the same
// state (cascade deletes their chats, applications, bookings, etc.).
func SeedDevData(db *sql.DB) error {
	aliceID, err := upsertDevUser(db, DevLearnerEmail, "alice.dev", "Alice Dev", "en", "{es}")
	if err != nil {
		return fmt.Errorf("seed learner alice: %w", err)
	}
	bobID, err := upsertDevUser(db, DevLearner2Email, "bob.dev", "Bob Dev", "es", "{en}")
	if err != nil {
		return fmt.Errorf("seed learner bob: %w", err)
	}
	sofiaID, err := upsertDevUser(db, DevTutorEmail, "sofia.tutor", "Sofia Tutor", "en", "{es}")
	if err != nil {
		return fmt.Errorf("seed tutor sofia: %w", err)
	}

	if err := seedTutorMarketplace(db, sofiaID, aliceID, bobID); err != nil {
		return err
	}
	if err := seedDevInvitation(db); err != nil {
		return err
	}
	return nil
}

// upsertDevUser deletes any existing fixture account (cascade removes its
// dependent rows) and recreates it fresh, returning the new user id.
func upsertDevUser(db *sql.DB, email, username, displayName, nativeLanguage, targetLanguages string) (string, error) {
	if _, err := db.Exec(`DELETE FROM users WHERE email = $1`, email); err != nil {
		return "", err
	}
	// Cost 10 keeps seeding fast; production registrations use the service
	// default (14). Dev-only credentials.
	hash, err := bcrypt.GenerateFromPassword([]byte(DevPassword), 10)
	if err != nil {
		return "", err
	}
	var id string
	err = db.QueryRow(`
		INSERT INTO users (username, email, password_hash, display_name, native_language, target_languages)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text
	`, username, email, string(hash), displayName, nativeLanguage, targetLanguages).Scan(&id)
	return id, err
}

// seedTutorMarketplace gives Sofia an approved application, a certificate,
// availability slots for the next days, reviews from both learners, and gives
// Alice a trial credit — enough state for the tutor marketplace TCs to browse,
// open a profile, and book.
func seedTutorMarketplace(db *sql.DB, teacherID, studentA, studentB string) error {
	var appID string
	err := db.QueryRow(`
		INSERT INTO teacher_applications (user_id, bio, languages, expertise, rate_cents, video_url, status)
		VALUES ($1, $2, '{es}', 'Conversational Spanish, DELE A1-B1 prep, pronunciation coaching', 2500, '', 'approved')
		RETURNING id::text
	`, teacherID, "Hola! I am Sofia, a certified Spanish teacher with 8 years of experience helping English speakers speak with confidence.").Scan(&appID)
	if err != nil {
		return fmt.Errorf("seed teacher application: %w", err)
	}
	if _, err := db.Exec(`
		INSERT INTO teacher_certificates (application_id, type, issuer, year, file_url, verified)
		VALUES ($1, 'language_certificate', 'Instituto Cervantes', 2018, 'https://example.com/certs/sofia-dele.pdf', true)
	`, appID); err != nil {
		return fmt.Errorf("seed teacher certificate: %w", err)
	}

	if _, err := db.Exec(`DELETE FROM tutor_availability WHERE teacher_user_id = $1`, teacherID); err != nil {
		return err
	}
	base := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Hour)
	for i := 0; i < 4; i++ {
		start := base.Add(time.Duration(i) * 24 * time.Hour)
		if _, err := db.Exec(`
			INSERT INTO tutor_availability (teacher_user_id, start_time, end_time)
			VALUES ($1, $2, $3)
		`, teacherID, start, start.Add(time.Hour)); err != nil {
			return fmt.Errorf("seed availability slot %d: %w", i, err)
		}
	}

	reviews := []struct {
		student string
		rating  int
		comment string
	}{
		{studentA, 5, "Sofia made my first lesson feel easy. She speaks slowly and explains everything."},
		{studentB, 4, "Great conversation practice and very patient with my pronunciation."},
	}
	for _, r := range reviews {
		if _, err := db.Exec(`
			INSERT INTO tutor_reviews (teacher_user_id, student_user_id, rating, comment)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (teacher_user_id, student_user_id) DO UPDATE SET rating = EXCLUDED.rating, comment = EXCLUDED.comment
		`, teacherID, r.student, r.rating, r.comment); err != nil {
			return fmt.Errorf("seed review: %w", err)
		}
	}

	if _, err := db.Exec(`
		INSERT INTO tutor_trial_credits (user_id, credits)
		VALUES ($1, 1)
		ON CONFLICT (user_id) DO UPDATE SET credits = 1, updated_at = CURRENT_TIMESTAMP
	`, studentA); err != nil {
		return fmt.Errorf("seed trial credit: %w", err)
	}
	return nil
}

// seedDevInvitation provisions an invitation bound to DevInviteEmail whose
// token is the well-known DevInviteToken, so TC-REG-02 can exercise the real
// invitation consumption path deterministically.
func seedDevInvitation(db *sql.DB) error {
	if _, err := db.Exec(`DELETE FROM invitations WHERE email = $1`, DevInviteEmail); err != nil {
		return err
	}
	if _, err := db.Exec(`DELETE FROM users WHERE email = $1`, DevInviteEmail); err != nil {
		return err
	}
	sum := sha256.Sum256([]byte(DevInviteToken))
	_, err := db.Exec(`
		INSERT INTO invitations (email, token_hash, expires_at, status)
		VALUES ($1, $2, $3, 'sent')
	`, DevInviteEmail, hex.EncodeToString(sum[:]), time.Now().Add(30*24*time.Hour))
	return err
}
