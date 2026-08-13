package services

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lib/pq"
)

func TestHashPassword(t *testing.T) {
	s := &AuthService{}
	hash, err := s.HashPassword("myPassword123!")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword returned empty hash")
	}
	if !s.CheckPassword("myPassword123!", hash) {
		t.Fatal("CheckPassword returned false for correct password")
	}
	if s.CheckPassword("wrongPassword", hash) {
		t.Fatal("CheckPassword returned true for incorrect password")
	}
}

func TestGenerateAccessToken(t *testing.T) {
	s := &AuthService{jwtSecret: "test-secret"}
	token, err := s.GenerateAccessToken("user-123")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("GenerateAccessToken returned empty token")
	}
}

func TestValidateAccessToken_Valid(t *testing.T) {
	s := &AuthService{jwtSecret: "test-secret"}
	token, err := s.GenerateAccessToken("user-123")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	userID, err := s.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}
	if userID != "user-123" {
		t.Fatalf("expected user-123, got %s", userID)
	}
}

func TestValidateAccessToken_Invalid(t *testing.T) {
	s := &AuthService{jwtSecret: "test-secret"}
	_, err := s.ValidateAccessToken("invalid-token")
	if err == nil {
		t.Fatal("ValidateAccessToken should have failed for invalid token")
	}
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	s1 := &AuthService{jwtSecret: "secret1"}
	s2 := &AuthService{jwtSecret: "secret2"}
	token, _ := s1.GenerateAccessToken("user-123")
	_, err := s2.ValidateAccessToken(token)
	if err == nil {
		t.Fatal("ValidateAccessToken should fail with wrong secret")
	}
}

func TestValidateAccessToken_Expired(t *testing.T) {
	// Create a token that's already expired
	s := &AuthService{jwtSecret: "test-secret"}
	claims := Claims{
		UserID: "user-123",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(s.jwtSecret))

	_, err := s.ValidateAccessToken(tokenStr)
	if err == nil {
		t.Fatal("ValidateAccessToken should fail for expired token")
	}
}

func TestRegister_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &AuthService{db: db, jwtSecret: "test-secret"}

	req := models.RegisterRequest{
		Username:        "testuser",
		Email:           "test@example.com",
		Password:        "Password123!",
		DisplayName:     "Test User",
		NativeLanguage:  "en",
		TargetLanguages: []string{"es"},
	}

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO users (username, email, password_hash, display_name, native_language, target_languages) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, username, email, display_name, native_language, target_languages, role, created_at, last_active_at, suspended_at, deleted_at, plan, plan_grace_until`)).
		WithArgs(req.Username, req.Email, sqlmock.AnyArg(), req.DisplayName, req.NativeLanguage, pq.Array(req.TargetLanguages)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "display_name", "native_language", "target_languages", "role", "created_at", "last_active_at", "suspended_at", "deleted_at", "plan", "plan_grace_until"}).
			AddRow("user-1", "testuser", "test@example.com", "Test User", "en", pq.Array([]string{"es"}), "member", time.Now(), time.Now(), nil, nil, "free", nil))

	user, err := s.Register(req)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
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

func TestRegister_DuplicateEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &AuthService{db: db, jwtSecret: "test-secret"}

	req := models.RegisterRequest{
		Username:        "testuser",
		Email:           "existing@example.com",
		Password:        "Password123!",
		DisplayName:     "Test User",
		NativeLanguage:  "en",
		TargetLanguages: []string{"es"},
	}

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO users (username, email, password_hash, display_name, native_language, target_languages) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, username, email, display_name, native_language, target_languages, role, created_at, last_active_at, suspended_at, deleted_at, plan, plan_grace_until`)).
		WithArgs(req.Username, req.Email, sqlmock.AnyArg(), req.DisplayName, req.NativeLanguage, pq.Array(req.TargetLanguages)).
		WillReturnError(&pq.Error{Code: "23505", Constraint: "users_email_key"})

	_, err = s.Register(req)
	if !errors.Is(err, ErrEmailAlreadyRegistered) {
		t.Fatalf("expected ErrEmailAlreadyRegistered, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRegister_EmptyUsername(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &AuthService{db: db, jwtSecret: "test-secret"}

	// When username is empty, the service passes it as-is to the DB
	// (username derivation from email happens in the handler, not the service)
	req := models.RegisterRequest{
		Username:        "",
		Email:           "test@example.com",
		Password:        "Password123!",
		DisplayName:     "Test User",
		NativeLanguage:  "en",
		TargetLanguages: []string{"es"},
	}

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO users (username, email, password_hash, display_name, native_language, target_languages) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, username, email, display_name, native_language, target_languages, role, created_at, last_active_at, suspended_at, deleted_at, plan, plan_grace_until`)).
		WithArgs("", req.Email, sqlmock.AnyArg(), req.DisplayName, req.NativeLanguage, pq.Array(req.TargetLanguages)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "display_name", "native_language", "target_languages", "role", "created_at", "last_active_at", "suspended_at", "deleted_at", "plan", "plan_grace_until"}).
			AddRow("user-1", "", "test@example.com", "Test User", "en", pq.Array([]string{"es"}), "member", time.Now(), time.Now(), nil, nil, "free", nil))

	user, err := s.Register(req)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if user.Username != "" {
		t.Fatalf("expected empty username, got '%s'", user.Username)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &AuthService{db: db, jwtSecret: "test-secret"}
	passwordHash, _ := s.HashPassword("Password123!")

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, username, email, password_hash, display_name, native_language, target_languages, role, created_at, last_active_at, suspended_at, deleted_at, plan, plan_grace_until FROM users WHERE username = $1 OR email = $1`)).
		WithArgs("testuser").
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "password_hash", "display_name", "native_language", "target_languages", "role", "created_at", "last_active_at", "suspended_at", "deleted_at", "plan", "plan_grace_until"}).
			AddRow("user-1", "testuser", "test@example.com", passwordHash, "Test User", "en", pq.Array([]string{"es"}), "member", time.Now(), time.Now(), nil, nil, "free", nil))

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET last_active_at = CURRENT_TIMESTAMP WHERE id = $1`)).
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	user, err := s.Login("testuser", "Password123!")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if user.ID != "user-1" {
		t.Fatalf("expected user-1, got %s", user.ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &AuthService{db: db, jwtSecret: "test-secret"}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, username, email, password_hash, display_name, native_language, target_languages, role, created_at, last_active_at, suspended_at, deleted_at, plan, plan_grace_until FROM users WHERE username = $1 OR email = $1`)).
		WithArgs("testuser").
		WillReturnError(sql.ErrNoRows)

	_, err = s.Login("testuser", "Password123!")
	if err == nil || err.Error() != "invalid credentials" {
		t.Fatalf("expected 'invalid credentials', got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGenerateAndValidateRefreshToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &AuthService{db: db, jwtSecret: "test-secret"}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)`)).
		WithArgs("user-1", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	refreshToken, err := s.GenerateRefreshToken("user-1")
	if err != nil {
		t.Fatalf("GenerateRefreshToken failed: %v", err)
	}
	if refreshToken == "" {
		t.Fatal("GenerateRefreshToken returned empty token")
	}

	// Mock validation
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id, expires_at FROM refresh_tokens WHERE token = $1`)).
		WithArgs(refreshToken).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "expires_at"}).
			AddRow("user-1", time.Now().Add(24*time.Hour)))

	userID, err := s.ValidateRefreshToken(refreshToken)
	if err != nil {
		t.Fatalf("ValidateRefreshToken failed: %v", err)
	}
	if userID != "user-1" {
		t.Fatalf("expected user-1, got %s", userID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestDeleteRefreshToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &AuthService{db: db}

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM refresh_tokens WHERE token = $1`)).
		WithArgs("token-to-delete").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = s.DeleteRefreshToken("token-to-delete")
	if err != nil {
		t.Fatalf("DeleteRefreshToken failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreatePasswordResetToken_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &AuthService{db: db}

	mock.ExpectQuery(`SELECT id FROM users WHERE email = \$1`).
		WithArgs("learner@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("user-1"))
	mock.ExpectExec(`DELETE FROM password_resets WHERE user_id = \$1 AND used_at IS NULL`).
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO password_resets \(user_id, token_hash, expires_at\)\s+VALUES \(\$1, \$2, CURRENT_TIMESTAMP \+ \$3 \* INTERVAL '1 minute'\)`).
		WithArgs("user-1", sqlmock.AnyArg(), 60).
		WillReturnResult(sqlmock.NewResult(0, 1))

	token, err := s.CreatePasswordResetToken("Learner@Example.com")
	if err != nil {
		t.Fatalf("CreatePasswordResetToken failed: %v", err)
	}
	if len(token) != 64 {
		t.Fatalf("expected 64-char token, got %d chars", len(token))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreatePasswordResetToken_UnknownEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &AuthService{db: db}

	mock.ExpectQuery(`SELECT id FROM users WHERE email = \$1`).
		WithArgs("nobody@example.com").
		WillReturnError(sql.ErrNoRows)

	_, err = s.CreatePasswordResetToken("nobody@example.com")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestResetPassword_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &AuthService{db: db}

	mock.ExpectBegin()
	mock.ExpectQuery(`UPDATE password_resets SET used_at = CURRENT_TIMESTAMP`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow("user-1"))
	mock.ExpectExec(`UPDATE users SET password_hash = \$1 WHERE id = \$2`).
		WithArgs(sqlmock.AnyArg(), "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	userID, err := s.ResetPassword("some-token", "NewPassword123!")
	if err != nil {
		t.Fatalf("ResetPassword failed: %v", err)
	}
	if userID != "user-1" {
		t.Fatalf("expected user-1, got %s", userID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestResetPassword_InvalidToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &AuthService{db: db}

	mock.ExpectBegin()
	mock.ExpectQuery(`UPDATE password_resets SET used_at = CURRENT_TIMESTAMP`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)

	_, err = s.ResetPassword("bad-token", "NewPassword123!")
	if !errors.Is(err, ErrInvalidResetToken) {
		t.Fatalf("expected ErrInvalidResetToken, got %v", err)
	}
}

func TestDeleteUserRefreshTokens(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &AuthService{db: db}

	mock.ExpectExec(`DELETE FROM refresh_tokens WHERE user_id = \$1`).
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(0, 3))

	if err := s.DeleteUserRefreshTokens("user-1"); err != nil {
		t.Fatalf("DeleteUserRefreshTokens failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
