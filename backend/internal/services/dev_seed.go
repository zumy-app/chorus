package services

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// Deterministic development fixtures (rescue plan A3/D3). These accounts back
// the acceptance suite (docs/TEST_SPEC.md) and manual emulator walkthroughs.
// Development only — production environments never run SeedDevData.
const (
	DevPassword = "ChorusDev123!"
	// Email local-parts encode the learning direction (native-target) so the
	// purpose of each fixture is obvious at login time:
	// alice speaks English and learns Spanish, bob is the mirror.
	DevLearnerEmail  = "alice.en-es@chorus.test"
	DevLearner2Email = "bob.es-en@chorus.test"
	DevTutorEmail    = "sofia.tutor@chorus.test"
	DevInviteEmail   = "invite.dev@chorus.test"
	DevInviteToken   = "chorus-dev-invite-2026"
)

// Legacy fixture emails from before the direction-signifying rename
// (alice.dev/bob.dev). Deleted on seed so stale rows don't linger.
const (
	legacyLearnerEmail  = "alice.dev@chorus.test"
	legacyLearner2Email = "bob.dev@chorus.test"
)

// SeedDevData provisions (or resets) the deterministic dev fixtures:
// two learners, one approved tutor with marketplace data (application,
// certificate, availability, reviews, trial credit), and an invitation with a
// known token for testing the invite-gated registration path. Idempotent:
// fixture accounts are deleted and recreated so every run yields the same
// state (cascade deletes their chats, applications, bookings, etc.).
func SeedDevData(db *sql.DB) error {
	// Drop every row that would block fixture-user deletion: any FK to
	// users(id) WITHOUT ON DELETE CASCADE (e.g. chats.created_by — real
	// test traffic creates chats owned by fixtures, and the delete used to
	// die with 23503). Discovered when the acceptance runner's reseed failed
	// after live messaging created fixture-owned chats.
	fixtureEmails := []string{
		DevLearnerEmail, DevLearner2Email, DevTutorEmail, DevInviteEmail,
		legacyLearnerEmail, legacyLearner2Email,
	}
	if err := deleteFixtureDependents(db, fixtureEmails); err != nil {
		return err
	}
	// Drop pre-rename fixture rows first (cascade removes their chats etc.).
	for _, legacy := range []string{legacyLearnerEmail, legacyLearner2Email} {
		if _, err := db.Exec(`DELETE FROM users WHERE email = $1`, legacy); err != nil {
			return fmt.Errorf("seed cleanup legacy %s: %w", legacy, err)
		}
	}
	aliceID, err := upsertDevUser(db, DevLearnerEmail, "alice.en-es", "Alice Dev", "en", "{es}")
	if err != nil {
		return fmt.Errorf("seed learner alice: %w", err)
	}
	bobID, err := upsertDevUser(db, DevLearner2Email, "bob.es-en", "Bob Dev", "es", "{en}")
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

// deleteFixtureDependents deletes, for the given emails, every row that would
// block fixture-user deletion, in dependency order:
//  1. rows in tables with a RESTRICT/NO ACTION FK to chats(id)/messages(id)
//     that point into the doomed chats (e.g. vocabulary context refs);
//  2. rows in tables with a RESTRICT/NO ACTION FK to users(id) (e.g. fixture
//     messages, refresh tokens) — discovered from the catalog so future
//     migrations stay seed-safe;
//  3. the doomed chats themselves.
// Everything else cascades with the user delete. Dev-only; fixture chats and
// their learned artifacts are removed with the fixtures (real test traffic
// creates fixture-owned chats, which used to die with 23503 on reseed).
func deleteFixtureDependents(db *sql.DB, emails []string) error {
	// Order matters: second-order refs (e.g. vocabulary → messages) must go
	// before the rows they point at; doomed chats go last.
	second, err := dependentColumns(db, `c.confrelid IN ('chats'::regclass, 'messages'::regclass)`)
	if err != nil {
		return err
	}
	for _, d := range second {
		var q string
		switch d.parent {
		case "public.chats", "chats":
			q = fmt.Sprintf(
				`DELETE FROM %s WHERE %s IN (SELECT id FROM chats WHERE created_by IN (SELECT id FROM users WHERE email = ANY ($1)))`,
				quoteIdentQualified(d.table), quoteIdent(d.column),
			)
		default: // messages
			q = fmt.Sprintf(
				`DELETE FROM %s WHERE %s IN (SELECT id FROM messages WHERE chat_id IN (SELECT id FROM chats WHERE created_by IN (SELECT id FROM users WHERE email = ANY ($1))))`,
				quoteIdentQualified(d.table), quoteIdent(d.column),
			)
		}
		if _, err := db.Exec(q, pq.Array(emails)); err != nil {
			return fmt.Errorf("seed cleanup %s: %w", d.table, err)
		}
	}

	direct, err := dependentColumns(db, `c.confrelid = 'users'::regclass`)
	if err != nil {
		return err
	}
	for _, d := range direct {
		// chats are removed below after their own dependents are gone.
		if d.table == "public.chats" || d.table == "chats" {
			continue
		}
		q := fmt.Sprintf(
			`DELETE FROM %s WHERE %s IN (SELECT id FROM users WHERE email = ANY ($1))`,
			quoteIdentQualified(d.table), quoteIdent(d.column),
		)
		if _, err := db.Exec(q, pq.Array(emails)); err != nil {
			return fmt.Errorf("seed cleanup %s: %w", d.table, err)
		}
	}

	if _, err := db.Exec(
		`DELETE FROM chats WHERE created_by IN (SELECT id FROM users WHERE email = ANY ($1))`,
		pq.Array(emails),
	); err != nil {
		return fmt.Errorf("seed cleanup chats: %w", err)
	}
	return nil
}

// dependentColumns lists (table, column, parent) for RESTRICT/NO ACTION FKs
// into the given parent relation(s). SET NULL / SET DEFAULT / CASCADE targets
// never block a delete and are excluded.
func dependentColumns(db *sql.DB, parentPred string) ([]struct {
	table, column, parent string
}, error) {
	rows, err := db.Query(fmt.Sprintf(`
		SELECT DISTINCT c.conrelid::regclass::text, a.attname, c.confrelid::regclass::text
		FROM pg_constraint c
		JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY (c.conkey)
		WHERE %s
		  AND c.contype = 'f'
		  AND c.confdeltype IN ('a', 'r')
	`, parentPred))
	if err != nil {
		return nil, fmt.Errorf("seed list dependents: %w", err)
	}
	defer rows.Close()
	var out []struct {
		table, column, parent string
	}
	for rows.Next() {
		var d struct {
			table, column, parent string
		}
		if err := rows.Scan(&d.table, &d.column, &d.parent); err != nil {
			return nil, fmt.Errorf("seed scan dependents: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("seed read dependents: %w", err)
	}
	return out, nil
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func quoteIdentQualified(name string) string {
	parts := strings.Split(name, ".")
	quoted := make([]string, 0, len(parts))
	for _, p := range parts {
		quoted = append(quoted, quoteIdent(p))
	}
	return strings.Join(quoted, ".")
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
