package services

import (
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
	"github.com/lib/pq"
)

func TestUserGetByID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewUserService(db)

	mock.ExpectQuery(`SELECT id, username, email, display_name, first_name, last_name, native_language, target_languages, role, created_at, last_active_at, suspended_at, deleted_at, plan, plan_grace_until, premium_since, subscription_id, subscription_provider, subscription_plan_id, subscription_status, next_billing_date, last_payment_at, avatar_url FROM users WHERE id = \$1`).
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "display_name", "first_name", "last_name", "native_language", "target_languages", "role", "created_at", "last_active_at", "suspended_at", "deleted_at", "plan", "plan_grace_until", "premium_since", "subscription_id", "subscription_provider", "subscription_plan_id", "subscription_status", "next_billing_date", "last_payment_at", "avatar_url"}).
			AddRow("user-1", "testuser", "test@example.com", "Test User", "", "", "en", pq.Array([]string{"es"}), "member", time.Now(), time.Now(), nil, nil, "free", nil, nil, nil, "paypal", nil, nil, nil, nil, nil))

	user, err := s.GetByID("user-1")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if user.ID != "user-1" {
		t.Fatalf("expected user-1, got %s", user.ID)
	}
	if user.Email != "test@example.com" {
		t.Fatalf("expected test@example.com, got %s", user.Email)
	}
	if user.Role != "member" {
		t.Fatalf("expected member role, got %s", user.Role)
	}
	if user.SuspendedAt != nil {
		t.Fatal("expected user to be active")
	}
	// REQ 2.2 / FR-20: the initials avatar color is deterministic and derived
	// from the user ID; the upload URL is reserved (unset).
	if user.AvatarColor != DeterministicAvatarColor("user-1") {
		t.Fatalf("expected deterministic avatar color %s, got %s", DeterministicAvatarColor("user-1"), user.AvatarColor)
	}
	if user.AvatarURL != nil {
		t.Fatal("expected reserved avatarURL to be unset")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestDeterministicAvatarColor(t *testing.T) {
	// Same seed always yields the same color (determinism).
	if DeterministicAvatarColor("user-1") != DeterministicAvatarColor("user-1") {
		t.Fatal("avatar color must be deterministic for the same seed")
	}
	// Empty seed falls back to the first palette color.
	if DeterministicAvatarColor("") != "#6366F1" {
		t.Fatalf("expected default color for empty seed, got %s", DeterministicAvatarColor(""))
	}
	// Every non-empty seed produces a valid 6-digit hex color.
	hexRe := regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	seen := map[string]bool{}
	for _, seed := range []string{"a", "b", "c", "user-1", "someone@example.com", "user-2"} {
		c := DeterministicAvatarColor(seed)
		if !hexRe.MatchString(c) {
			t.Fatalf("expected hex color from seed %q, got %q", seed, c)
		}
		seen[c] = true
	}
	// At least two distinct colors appear so the palette is not degenerate.
	if len(seen) < 2 {
		t.Fatalf("expected palette to yield >1 distinct colors, got %d", len(seen))
	}
}

func TestUserGetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewUserService(db)

	mock.ExpectQuery(`SELECT id, username, email, display_name, first_name, last_name, native_language, target_languages, role, created_at, last_active_at, suspended_at, deleted_at, plan, plan_grace_until, premium_since, subscription_id, subscription_provider, subscription_plan_id, subscription_status, next_billing_date, last_payment_at, avatar_url FROM users WHERE id = \$1`).
		WithArgs("nonexistent").
		WillReturnError(sqlmock.ErrCancelled)

	_, err = s.GetByID("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUserUpdate_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewUserService(db)

	req := models.UpdateUserRequest{
		DisplayName:     "Updated Name",
		NativeLanguage:  "fr",
		TargetLanguages: []string{"en", "de"},
	}

	updateSQL := `UPDATE users SET first_name = COALESCE(NULLIF($2, ''), first_name), last_name = COALESCE(NULLIF($3, ''), last_name), display_name = COALESCE(NULLIF($4, ''), TRIM(CONCAT_WS(' ', NULLIF(TRIM(COALESCE(NULLIF($2, ''), first_name)), ''), NULLIF(TRIM(COALESCE(NULLIF($3, ''), last_name)), ''))), display_name), native_language = COALESCE(NULLIF($5, ''), native_language), target_languages = COALESCE($6, target_languages) WHERE id = $1 RETURNING id, username, email, display_name, first_name, last_name, native_language, target_languages, role, created_at, last_active_at, suspended_at, deleted_at, plan, plan_grace_until, premium_since, subscription_id, subscription_provider, subscription_plan_id, subscription_status, next_billing_date, last_payment_at, avatar_url`
	mock.ExpectQuery(regexp.QuoteMeta(updateSQL)).
		WithArgs("user-1", sqlmock.AnyArg(), sqlmock.AnyArg(), "Updated Name", sqlmock.AnyArg(), pq.Array(req.TargetLanguages)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "display_name", "first_name", "last_name", "native_language", "target_languages", "role", "created_at", "last_active_at", "suspended_at", "deleted_at", "plan", "plan_grace_until", "premium_since", "subscription_id", "subscription_provider", "subscription_plan_id", "subscription_status", "next_billing_date", "last_payment_at", "avatar_url"}).
			AddRow("user-1", "testuser", "test@example.com", "Updated Name", "", "", "fr", pq.Array([]string{"en", "de"}), "member", time.Now(), time.Now(), nil, nil, "free", nil, nil, nil, "paypal", nil, nil, nil, nil, nil))

	user, err := s.Update("user-1", req)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if user.DisplayName != "Updated Name" {
		t.Fatalf("expected 'Updated Name', got '%s'", user.DisplayName)
	}
	if user.NativeLanguage != "fr" {
		t.Fatalf("expected 'fr', got '%s'", user.NativeLanguage)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUserOnboard_ComposesDisplayName(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewUserService(db)

	onboardSQL := `UPDATE users SET first_name = $2, last_name = $3, display_name = $4 WHERE id = $1 RETURNING id, username, email, display_name, first_name, last_name, native_language, target_languages, role, created_at, last_active_at, suspended_at, deleted_at, plan, plan_grace_until, premium_since, subscription_id, subscription_provider, subscription_plan_id, subscription_status, next_billing_date, last_payment_at, avatar_url`
	mock.ExpectQuery(regexp.QuoteMeta(onboardSQL)).
		WithArgs("user-1", "John", "Smith", "John Smith").
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "display_name", "first_name", "last_name", "native_language", "target_languages", "role", "created_at", "last_active_at", "suspended_at", "deleted_at", "plan", "plan_grace_until", "premium_since", "subscription_id", "subscription_provider", "subscription_plan_id", "subscription_status", "next_billing_date", "last_payment_at", "avatar_url"}).
			AddRow("user-1", "testuser", "test@example.com", "John Smith", "John", "Smith", "en", pq.Array([]string{"es"}), "member", time.Now(), time.Now(), nil, nil, "free", nil, nil, nil, "paypal", nil, nil, nil, nil, nil))

	user, err := s.Onboard("user-1", "John", "Smith", "")
	if err != nil {
		t.Fatalf("Onboard failed: %v", err)
	}
	if user.FirstName != "John" {
		t.Fatalf("expected firstName 'John', got '%s'", user.FirstName)
	}
	if user.LastName != "Smith" {
		t.Fatalf("expected lastName 'Smith', got '%s'", user.LastName)
	}
	if user.DisplayName != "John Smith" {
		t.Fatalf("expected composed displayName 'John Smith', got '%s'", user.DisplayName)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUserOnboard_ExplicitDisplayName(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewUserService(db)

	onboardSQL := `UPDATE users SET first_name = $2, last_name = $3, display_name = $4 WHERE id = $1 RETURNING id, username, email, display_name, first_name, last_name, native_language, target_languages, role, created_at, last_active_at, suspended_at, deleted_at, plan, plan_grace_until, premium_since, subscription_id, subscription_provider, subscription_plan_id, subscription_status, next_billing_date, last_payment_at, avatar_url`
	mock.ExpectQuery(regexp.QuoteMeta(onboardSQL)).
		WithArgs("user-1", "John", "Smith", "JD").
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "display_name", "first_name", "last_name", "native_language", "target_languages", "role", "created_at", "last_active_at", "suspended_at", "deleted_at", "plan", "plan_grace_until", "premium_since", "subscription_id", "subscription_provider", "subscription_plan_id", "subscription_status", "next_billing_date", "last_payment_at", "avatar_url"}).
			AddRow("user-1", "testuser", "test@example.com", "JD", "John", "Smith", "en", pq.Array([]string{"es"}), "member", time.Now(), time.Now(), nil, nil, "free", nil, nil, nil, "paypal", nil, nil, nil, nil, nil))

	user, err := s.Onboard("user-1", "John", "Smith", "JD")
	if err != nil {
		t.Fatalf("Onboard failed: %v", err)
	}
	if user.FirstName != "John" {
		t.Fatalf("expected firstName 'John', got '%s'", user.FirstName)
	}
	if user.DisplayName != "JD" {
		t.Fatalf("expected override displayName 'JD', got '%s'", user.DisplayName)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestComposeDisplayName(t *testing.T) {
	cases := []struct {
		first, last, want string
	}{
		{"John", "Smith", "John Smith"},
		{"John", "", "John"},
		{"", "Smith", "Smith"},
		{"", "", ""},
		{"  John  ", "  Smith  ", "John Smith"},
	}
	for _, tc := range cases {
		got := ComposeDisplayName(tc.first, tc.last)
		if got != tc.want {
			t.Errorf("ComposeDisplayName(%q, %q) = %q, want %q", tc.first, tc.last, got, tc.want)
		}
	}
}
