package services

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
	"github.com/lib/pq"
)

// invitationTokenHash mirrors the sha256+hex hash used by the invitation flow.
func invitationTokenHash(t *testing.T, token string) string {
	t.Helper()
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// authUserColumns returns the projection used by auth/user queries so tests
// stay in sync with the schema.
func authUserColumns() string {
	return "id, username, email, display_name, native_language, target_language_level, target_languages, role, created_at, last_active_at, suspended_at, deleted_at, plan, plan_grace_until, premium_since, subscription_id, subscription_provider, subscription_plan_id, subscription_status, next_billing_date, last_payment_at"
}

func newUserRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "username", "email", "display_name", "native_language", "target_language_level", "target_languages",
		"role", "created_at", "last_active_at", "suspended_at", "deleted_at", "plan",
		"plan_grace_until", "premium_since", "subscription_id", "subscription_provider",
		"subscription_plan_id", "subscription_status", "next_billing_date", "last_payment_at",
	})
}

// TestRegister_PersistsTargetLanguageLevel verifies that Register writes the
// target language level supplied in the request into the users table.
func TestRegister_PersistsTargetLanguageLevel(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &AuthService{db: db, jwtSecret: "test-secret"}

	req := models.RegisterRequest{
		Username:            "testuser",
		Email:               "test@example.com",
		Password:            "Password123!",
		DisplayName:         "Test User",
		NativeLanguage:      "en",
		TargetLanguageLevel: "B2",
		TargetLanguages:     []string{"es"},
	}

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO users (username, email, password_hash, display_name, native_language, target_language_level, target_languages) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING `+authUserColumns())).
		WithArgs(req.Username, req.Email, sqlmock.AnyArg(), req.DisplayName, req.NativeLanguage, req.TargetLanguageLevel, pq.Array(req.TargetLanguages)).
		WillReturnRows(newUserRows().
			AddRow("user-1", "testuser", "test@example.com", "Test User", "en", "B2", pq.Array([]string{"es"}), "member", time.Now(), time.Now(), nil, nil, "free", nil, nil, nil, "paypal", nil, nil, nil, nil))

	user, err := s.Register(req)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if user.TargetLanguageLevel != "B2" {
		t.Fatalf("expected persisted level B2, got %q", user.TargetLanguageLevel)
	}
	if len(user.TargetLanguages) != 1 || user.TargetLanguages[0] != "es" {
		t.Fatalf("expected persisted target languages [es], got %v", user.TargetLanguages)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestRegisterWithInvitation_CarriesOverWaitlistPrefs verifies the core fix:
// when the registration form submits empty target language prefs (the default
// /register page does not ask for them), the values captured on the waitlist
// entry are carried over to the new user profile.
func TestRegisterWithInvitation_CarriesOverWaitlistPrefs(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &AuthService{db: db, jwtSecret: "test-secret"}

	token := "invite-token-123"
	req := models.RegisterRequest{
		Username:       "testuser",
		Email:          "test@example.com",
		Password:       "Password123!",
		DisplayName:    "Test User",
		NativeLanguage: "en",
		// Frontend default: no level, empty target languages.
		TargetLanguages: []string{},
		InviteToken:     token,
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE invitations SET redeemed_at = CURRENT_TIMESTAMP WHERE token_hash = $1 AND email = $2 AND redeemed_at IS NULL AND expires_at > CURRENT_TIMESTAMP RETURNING id, waitlist_entry_id`)).
		WithArgs(invitationTokenHash(t, token), "test@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "waitlist_entry_id"}).AddRow("inv-1", "wl-1"))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT target_languages, target_language_level FROM waitlist_entries WHERE id = $1`)).
		WithArgs("wl-1").
		WillReturnRows(sqlmock.NewRows([]string{"target_languages", "target_language_level"}).
			AddRow(pq.Array([]string{"es", "fr"}), "B1"))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO users (username, email, password_hash, display_name, native_language, target_language_level, target_languages) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING `+authUserColumns())).
		WithArgs(req.Username, req.Email, sqlmock.AnyArg(), req.DisplayName, req.NativeLanguage, "B1", pq.Array([]string{"es", "fr"})).
		WillReturnRows(newUserRows().
			AddRow("user-1", "testuser", "test@example.com", "Test User", "en", "B1", pq.Array([]string{"es", "fr"}), "member", time.Now(), time.Now(), nil, nil, "free", nil, nil, nil, "paypal", nil, nil, nil, nil))
	mock.ExpectCommit()

	user, err := s.RegisterWithInvitation(req)
	if err != nil {
		t.Fatalf("RegisterWithInvitation failed: %v", err)
	}
	if user.TargetLanguageLevel != "B1" {
		t.Fatalf("expected waitlist level B1 to carry over, got %q", user.TargetLanguageLevel)
	}
	if len(user.TargetLanguages) != 2 || user.TargetLanguages[0] != "es" || user.TargetLanguages[1] != "fr" {
		t.Fatalf("expected waitlist target languages [es fr] to carry over, got %v", user.TargetLanguages)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestRegisterWithInvitation_RequestOverridesWaitlist verifies that explicitly
// supplied preferences (e.g. a registration form that does collect them) win
// over the waitlist entry values.
func TestRegisterWithInvitation_RequestOverridesWaitlist(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &AuthService{db: db, jwtSecret: "test-secret"}

	token := "invite-token-456"
	req := models.RegisterRequest{
		Username:            "testuser",
		Email:               "test@example.com",
		Password:            "Password123!",
		DisplayName:         "Test User",
		NativeLanguage:      "en",
		TargetLanguageLevel: "C1",
		TargetLanguages:     []string{"de"},
		InviteToken:         token,
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE invitations SET redeemed_at = CURRENT_TIMESTAMP WHERE token_hash = $1 AND email = $2 AND redeemed_at IS NULL AND expires_at > CURRENT_TIMESTAMP RETURNING id, waitlist_entry_id`)).
		WithArgs(invitationTokenHash(t, token), "test@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "waitlist_entry_id"}).AddRow("inv-2", "wl-2"))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT target_languages, target_language_level FROM waitlist_entries WHERE id = $1`)).
		WithArgs("wl-2").
		WillReturnRows(sqlmock.NewRows([]string{"target_languages", "target_language_level"}).
			AddRow(pq.Array([]string{"es", "fr"}), "B1"))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO users (username, email, password_hash, display_name, native_language, target_language_level, target_languages) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING `+authUserColumns())).
		WithArgs(req.Username, req.Email, sqlmock.AnyArg(), req.DisplayName, req.NativeLanguage, "C1", pq.Array([]string{"de"})).
		WillReturnRows(newUserRows().
			AddRow("user-1", "testuser", "test@example.com", "Test User", "en", "C1", pq.Array([]string{"de"}), "member", time.Now(), time.Now(), nil, nil, "free", nil, nil, nil, "paypal", nil, nil, nil, nil))
	mock.ExpectCommit()

	user, err := s.RegisterWithInvitation(req)
	if err != nil {
		t.Fatalf("RegisterWithInvitation failed: %v", err)
	}
	if user.TargetLanguageLevel != "C1" {
		t.Fatalf("expected request level C1 to override waitlist B1, got %q", user.TargetLanguageLevel)
	}
	if len(user.TargetLanguages) != 1 || user.TargetLanguages[0] != "de" {
		t.Fatalf("expected request target languages [de] to override waitlist, got %v", user.TargetLanguages)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestRegisterWithInvitation_WaitlistLookupFailure_FallsBackToDefaults verifies
// that a missing waitlist entry does not block registration; the user is still
// created with an empty target language set and the default A1 level.
func TestRegisterWithInvitation_WaitlistLookupFailure_FallsBackToDefaults(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &AuthService{db: db, jwtSecret: "test-secret"}

	token := "invite-token-789"
	req := models.RegisterRequest{
		Username:       "testuser",
		Email:          "test@example.com",
		Password:       "Password123!",
		DisplayName:    "Test User",
		NativeLanguage: "en",
		InviteToken:    token,
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE invitations SET redeemed_at = CURRENT_TIMESTAMP WHERE token_hash = $1 AND email = $2 AND redeemed_at IS NULL AND expires_at > CURRENT_TIMESTAMP RETURNING id, waitlist_entry_id`)).
		WithArgs(invitationTokenHash(t, token), "test@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "waitlist_entry_id"}).AddRow("inv-3", "wl-3"))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT target_languages, target_language_level FROM waitlist_entries WHERE id = $1`)).
		WithArgs("wl-3").
		WillReturnError(sqlmock.ErrCancelled)
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO users (username, email, password_hash, display_name, native_language, target_language_level, target_languages) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING `+authUserColumns())).
		WithArgs(req.Username, req.Email, sqlmock.AnyArg(), req.DisplayName, req.NativeLanguage, "A1", pq.Array([]string{})).
		WillReturnRows(newUserRows().
			AddRow("user-1", "testuser", "test@example.com", "Test User", "en", "A1", pq.Array([]string{}), "member", time.Now(), time.Now(), nil, nil, "free", nil, nil, nil, "paypal", nil, nil, nil, nil))
	mock.ExpectCommit()

	user, err := s.RegisterWithInvitation(req)
	if err != nil {
		t.Fatalf("RegisterWithInvitation failed: %v", err)
	}
	if user.TargetLanguageLevel != "A1" {
		t.Fatalf("expected fallback level A1, got %q", user.TargetLanguageLevel)
	}
	if len(user.TargetLanguages) != 0 {
		t.Fatalf("expected empty fallback target languages, got %v", user.TargetLanguages)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestRegisterWithInvitation_InvalidInvitation verifies the invitation must
// still be valid for the waitlist carry-over path to run at all.
func TestRegisterWithInvitation_InvalidInvitation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &AuthService{db: db, jwtSecret: "test-secret"}

	req := models.RegisterRequest{
		Username:       "testuser",
		Email:          "test@example.com",
		Password:       "Password123!",
		DisplayName:    "Test User",
		NativeLanguage: "en",
		InviteToken:    "bogus-token",
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE invitations SET redeemed_at = CURRENT_TIMESTAMP WHERE token_hash = $1 AND email = $2 AND redeemed_at IS NULL AND expires_at > CURRENT_TIMESTAMP RETURNING id, waitlist_entry_id`)).
		WithArgs(invitationTokenHash(t, "bogus-token"), "test@example.com").
		WillReturnError(sqlmock.ErrCancelled)

	_, err = s.RegisterWithInvitation(req)
	if !errors.Is(err, ErrInvalidInvitation) {
		t.Fatalf("expected ErrInvalidInvitation, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestWaitlistSubmit_PersistsTargetLanguageLevel verifies the waitlist entry
// stores the target language level chosen on the waitlist form so it can later
// be carried over when the user registers via invitation.
func TestWaitlistSubmit_PersistsTargetLanguageLevel(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewWaitlistService(db)

	req := models.WaitlistRequest{
		Email:               "learner@example.com",
		SpokenLanguages:     []string{"en"},
		TargetLanguages:     []string{"es", "fr"},
		TargetLanguageLevel: "B1",
		Reasons:             []string{"travel"},
		Comments:            "want to practice",
	}

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO waitlist_entries (email, spoken_languages, target_languages, target_language_level, reasons, comments) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (email) DO UPDATE SET spoken_languages = EXCLUDED.spoken_languages, target_languages = EXCLUDED.target_languages, target_language_level = EXCLUDED.target_language_level, reasons = EXCLUDED.reasons, comments = EXCLUDED.comments RETURNING (xmax = 0) AS inserted, id, email, spoken_languages, target_languages, target_language_level, reasons, comments, status, queue_position, created_at`)).
		WithArgs("learner@example.com", pq.Array([]string{"en"}), pq.Array([]string{"es", "fr"}), "B1", pq.Array([]string{"travel"}), "want to practice").
		WillReturnRows(sqlmock.NewRows([]string{"inserted", "id", "email", "spoken_languages", "target_languages", "target_language_level", "reasons", "comments", "status", "queue_position", "created_at"}).
			AddRow(true, "wl-1", "learner@example.com", pq.Array([]string{"en"}), pq.Array([]string{"es", "fr"}), "B1", pq.Array([]string{"travel"}), "want to practice", "approved", 1, time.Now()))

	entry, existing, err := s.Submit(req)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
	if existing {
		t.Fatal("expected new entry, got existing=true")
	}
	if entry.TargetLanguageLevel != "B1" {
		t.Fatalf("expected persisted level B1, got %q", entry.TargetLanguageLevel)
	}
	if len(entry.TargetLanguages) != 2 {
		t.Fatalf("expected persisted target languages [es fr], got %v", entry.TargetLanguages)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestLogin_ReturnsTargetLanguageLevel verifies Login selects and returns the
// persisted target language level so the profile round-trips correctly.
func TestLogin_ReturnsTargetLanguageLevel(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &AuthService{db: db, jwtSecret: "test-secret"}
	passwordHash, _ := s.HashPassword("Password123!")

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, username, email, password_hash, display_name, native_language, target_language_level, target_languages, role, created_at, last_active_at, suspended_at, deleted_at, plan, plan_grace_until, premium_since, subscription_id, subscription_provider, subscription_plan_id, subscription_status, next_billing_date, last_payment_at FROM users WHERE username = $1 OR email = $1`)).
		WithArgs("test@example.com").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "username", "email", "password_hash", "display_name", "native_language", "target_language_level", "target_languages",
			"role", "created_at", "last_active_at", "suspended_at", "deleted_at", "plan",
			"plan_grace_until", "premium_since", "subscription_id", "subscription_provider",
			"subscription_plan_id", "subscription_status", "next_billing_date", "last_payment_at",
		}).
			AddRow("user-1", "testuser", "test@example.com", passwordHash, "Test User", "en", "B2", pq.Array([]string{"es"}), "member", time.Now(), time.Now(), nil, nil, "free", nil, nil, nil, "paypal", nil, nil, nil, nil))

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET last_active_at = CURRENT_TIMESTAMP WHERE id = $1`)).
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	user, err := s.Login("test@example.com", "Password123!")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if user.TargetLanguageLevel != "B2" {
		t.Fatalf("expected level B2 from Login, got %q", user.TargetLanguageLevel)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
