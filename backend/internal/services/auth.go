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
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	db        *sql.DB
	jwtSecret string
}

var (
	ErrEmailAlreadyRegistered    = errors.New("email already registered")
	ErrUsernameAlreadyRegistered = errors.New("username already registered")
	ErrUserNotFound              = errors.New("user not found")
	ErrInvalidResetToken         = errors.New("reset token is invalid, expired, or already used")
)

// passwordResetTTL is how long a password reset link stays valid.
const passwordResetTTL = 60 * time.Minute

func NewAuthService(db *sql.DB, jwtSecret string) *AuthService {
	return &AuthService{
		db:        db,
		jwtSecret: jwtSecret,
	}
}

type Claims struct {
	UserID string `json:"userId"`
	jwt.RegisteredClaims
}

func (s *AuthService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func (s *AuthService) CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (s *AuthService) GenerateAccessToken(userID string) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *AuthService) GenerateRefreshToken(userID string) (string, error) {
	tokenID := uuid.New().String()
	expiresAt := time.Now().Add(30 * 24 * time.Hour)

	query := `INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)`
	_, err := s.db.Exec(query, userID, tokenID, expiresAt)
	if err != nil {
		return "", err
	}

	return tokenID, nil
}

func (s *AuthService) ValidateAccessToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims.UserID, nil
	}

	return "", errors.New("invalid token")
}

func (s *AuthService) ValidateRefreshToken(tokenString string) (string, error) {
	var userID string
	var expiresAt time.Time

	query := `SELECT user_id, expires_at FROM refresh_tokens WHERE token = $1`
	err := s.db.QueryRow(query, tokenString).Scan(&userID, &expiresAt)
	if err != nil {
		return "", errors.New("invalid refresh token")
	}

	if time.Now().After(expiresAt) {
		s.DeleteRefreshToken(tokenString)
		return "", errors.New("refresh token expired")
	}

	return userID, nil
}

func (s *AuthService) DeleteRefreshToken(token string) error {
	query := `DELETE FROM refresh_tokens WHERE token = $1`
	_, err := s.db.Exec(query, token)
	return err
}

func (s *AuthService) Register(req models.RegisterRequest) (*models.User, error) {
	// Hash password
	passwordHash, err := s.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// Insert user
	user := &models.User{}
	query := `
		INSERT INTO users (username, email, password_hash, display_name, native_language, target_language_level, target_languages)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, username, email, display_name, native_language, target_language_level, target_languages, role, created_at, last_active_at, suspended_at, deleted_at, plan, plan_grace_until, premium_since, subscription_id, subscription_provider, subscription_plan_id, subscription_status, next_billing_date, last_payment_at
	`

	err = s.db.QueryRow(
		query,
		req.Username,
		req.Email,
		passwordHash,
		req.DisplayName,
		req.NativeLanguage,
		req.TargetLanguageLevel,
		pq.Array(req.TargetLanguages),
	).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.DisplayName,
		&user.NativeLanguage,
		&user.TargetLanguageLevel,
		pq.Array(&user.TargetLanguages),
		&user.Role,
		&user.CreatedAt,
		&user.LastActiveAt,
		&user.SuspendedAt,
		&user.DeletedAt,
		&user.Plan,
		&user.PlanGraceUntil,
		&user.PremiumSince,
		&user.SubscriptionID,
		&user.SubscriptionProvider,
		&user.SubscriptionPlanID,
		&user.SubscriptionStatus,
		&user.NextBillingDate,
		&user.LastPaymentAt,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				constraint := strings.ToLower(pqErr.Constraint)
				switch {
				case strings.Contains(constraint, "users_email"):
					return nil, ErrEmailAlreadyRegistered
				case strings.Contains(constraint, "users_username"):
					return nil, ErrUsernameAlreadyRegistered
				default:
					return nil, errors.New("account already exists")
				}
			}
		}
		return nil, err
	}

	return user, nil
}

// RegisterWithInvitation atomically consumes an email-bound invitation and
// creates the user, preventing concurrent use of the same invitation.
func (s *AuthService) RegisterWithInvitation(req models.RegisterRequest) (*models.User, error) {
	passwordHash, err := s.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	sum := sha256.Sum256([]byte(req.InviteToken))
	var invitationID string
	var waitlistEntryID string
	err = tx.QueryRow(`UPDATE invitations SET redeemed_at = CURRENT_TIMESTAMP
		WHERE token_hash = $1 AND email = $2 AND redeemed_at IS NULL AND expires_at > CURRENT_TIMESTAMP
		RETURNING id, waitlist_entry_id`, hex.EncodeToString(sum[:]), strings.ToLower(strings.TrimSpace(req.Email))).Scan(&invitationID, &waitlistEntryID)
	if err != nil {
		return nil, ErrInvalidInvitation
	}

	// Fetch target languages and level from the waitlist entry to persist them
	var targetLanguages []string
	var targetLanguageLevel string
	err = tx.QueryRow(`SELECT target_languages, target_language_level FROM waitlist_entries WHERE id = $1`, waitlistEntryID).Scan(pq.Array(&targetLanguages), &targetLanguageLevel)
	if err != nil {
		targetLanguages = []string{}
		targetLanguageLevel = "A1"
	}

	// If request has explicit values (e.g. from registration form), override with them
	if len(req.TargetLanguages) > 0 {
		targetLanguages = req.TargetLanguages
	}
	if req.TargetLanguageLevel != "" {
		targetLanguageLevel = req.TargetLanguageLevel
	}

	user := &models.User{}
	err = tx.QueryRow(`
		INSERT INTO users (username, email, password_hash, display_name, native_language, target_language_level, target_languages)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, username, email, display_name, native_language, target_language_level, target_languages, role, created_at, last_active_at, suspended_at, deleted_at, plan, plan_grace_until, premium_since, subscription_id, subscription_provider, subscription_plan_id, subscription_status, next_billing_date, last_payment_at`,
		req.Username, req.Email, passwordHash, req.DisplayName, req.NativeLanguage, targetLanguageLevel, pq.Array(targetLanguages),
	).Scan(&user.ID, &user.Username, &user.Email, &user.DisplayName, &user.NativeLanguage, &user.TargetLanguageLevel,
		pq.Array(&user.TargetLanguages), &user.Role, &user.CreatedAt, &user.LastActiveAt, &user.SuspendedAt, &user.DeletedAt,
		&user.Plan, &user.PlanGraceUntil, &user.PremiumSince, &user.SubscriptionID, &user.SubscriptionProvider,
		&user.SubscriptionPlanID, &user.SubscriptionStatus, &user.NextBillingDate, &user.LastPaymentAt)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return nil, ErrEmailAlreadyRegistered
		}
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) Login(username, password string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, username, email, password_hash, display_name, native_language, target_language_level, target_languages, role, created_at, last_active_at, suspended_at, deleted_at, plan, plan_grace_until, premium_since, subscription_id, subscription_provider, subscription_plan_id, subscription_status, next_billing_date, last_payment_at
		FROM users
		WHERE username = $1 OR email = $1
	`

	err := s.db.QueryRow(query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.NativeLanguage,
		&user.TargetLanguageLevel,
		pq.Array(&user.TargetLanguages),
		&user.Role,
		&user.CreatedAt,
		&user.LastActiveAt,
		&user.SuspendedAt,
		&user.DeletedAt,
		&user.Plan,
		&user.PlanGraceUntil,
		&user.PremiumSince,
		&user.SubscriptionID,
		&user.SubscriptionProvider,
		&user.SubscriptionPlanID,
		&user.SubscriptionStatus,
		&user.NextBillingDate,
		&user.LastPaymentAt,
	)

	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !s.CheckPassword(password, user.PasswordHash) {
		return nil, errors.New("invalid credentials")
	}

	// Suspended/deleted accounts cannot authenticate.
	if user.SuspendedAt != nil || user.DeletedAt != nil {
		return nil, errors.New("account is disabled")
	}

	// Update last active
	updateQuery := `UPDATE users SET last_active_at = CURRENT_TIMESTAMP WHERE id = $1`
	s.db.Exec(updateQuery, user.ID)

	return user, nil
}

// CreatePasswordResetToken generates a single-use reset token for the given
// email and stores only its hash. It returns ErrUserNotFound if no account
// matches. Previously issued (unused) tokens for the user are invalidated.
func (s *AuthService) CreatePasswordResetToken(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var userID string
	err := s.db.QueryRow(`SELECT id FROM users WHERE email = $1`, email).Scan(&userID)
	if err != nil {
		return "", ErrUserNotFound
	}
	if _, err := s.db.Exec(`DELETE FROM password_resets WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
		return "", err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := hex.EncodeToString(raw)
	sum := sha256.Sum256([]byte(token))
	// Compute expires_at in SQL (CURRENT_TIMESTAMP) so it stays consistent with
	// the comparison below no matter what timezone the backend runs in.
	_, err = s.db.Exec(`INSERT INTO password_resets (user_id, token_hash, expires_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP + $3 * INTERVAL '1 minute')`,
		userID, hex.EncodeToString(sum[:]), int(passwordResetTTL.Minutes()))
	if err != nil {
		return "", err
	}
	return token, nil
}

// ResetPassword atomically consumes a valid reset token and sets the new
// password. It returns the user ID on success. The caller should invalidate
// the user's refresh tokens afterwards.
func (s *AuthService) ResetPassword(token, newPassword string) (string, error) {
	newHash, err := s.HashPassword(newPassword)
	if err != nil {
		return "", err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	sum := sha256.Sum256([]byte(token))
	var userID string
	err = tx.QueryRow(`UPDATE password_resets SET used_at = CURRENT_TIMESTAMP
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > CURRENT_TIMESTAMP
		RETURNING user_id`, hex.EncodeToString(sum[:])).Scan(&userID)
	if err != nil {
		return "", ErrInvalidResetToken
	}
	if _, err := tx.Exec(`UPDATE users SET password_hash = $1 WHERE id = $2`, newHash, userID); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return userID, nil
}

// DeleteUserRefreshTokens revokes every refresh token a user has, e.g. after a
// password reset, forcing all devices to log in again.
func (s *AuthService) DeleteUserRefreshTokens(userID string) error {
	_, err := s.db.Exec(`DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
	return err
}
