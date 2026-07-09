package services

import (
	"testing"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
)

func TestUserGetByID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewUserService(db)

	mock.ExpectQuery(`SELECT id, username, email, display_name, native_language, target_languages, created_at, last_active_at FROM users WHERE id = \$1`).
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "display_name", "native_language", "target_languages", "created_at", "last_active_at"}).
			AddRow("user-1", "testuser", "test@example.com", "Test User", "en", pq.Array([]string{"es"}), time.Now(), time.Now()))

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

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUserGetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewUserService(db)

	mock.ExpectQuery(`SELECT id, username, email, display_name, native_language, target_languages, created_at, last_active_at FROM users WHERE id = \$1`).
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

	mock.ExpectQuery(`UPDATE users SET display_name = COALESCE\(NULLIF\(\$2, ''\), display_name\), native_language = COALESCE\(NULLIF\(\$3, ''\), native_language\), target_languages = COALESCE\(\$4, target_languages\) WHERE id = \$1 RETURNING id, username, email, display_name, native_language, target_languages, created_at, last_active_at`).
		WithArgs("user-1", sqlmock.AnyArg(), sqlmock.AnyArg(), pq.Array(req.TargetLanguages)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "display_name", "native_language", "target_languages", "created_at", "last_active_at"}).
			AddRow("user-1", "testuser", "test@example.com", "Updated Name", "fr", pq.Array([]string{"en", "de"}), time.Now(), time.Now()))

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
