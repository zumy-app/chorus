package services

import (
	"database/sql"
	"errors"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	db        *sql.DB
	jwtSecret string
}

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
		INSERT INTO users (username, email, password_hash, display_name, native_language, target_languages)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, username, email, display_name, native_language, target_languages, created_at, last_active_at
	`

	err = s.db.QueryRow(
		query,
		req.Username,
		req.Email,
		passwordHash,
		req.DisplayName,
		req.NativeLanguage,
		req.TargetLanguages,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.DisplayName,
		&user.NativeLanguage,
		&user.TargetLanguages,
		&user.CreatedAt,
		&user.LastActiveAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) Login(username, password string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, username, email, password_hash, display_name, native_language, target_languages, created_at, last_active_at
		FROM users
		WHERE username = $1
	`

	err := s.db.QueryRow(query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.NativeLanguage,
		&user.TargetLanguages,
		&user.CreatedAt,
		&user.LastActiveAt,
	)

	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !s.CheckPassword(password, user.PasswordHash) {
		return nil, errors.New("invalid credentials")
	}

	// Update last active
	updateQuery := `UPDATE users SET last_active_at = CURRENT_TIMESTAMP WHERE id = $1`
	s.db.Exec(updateQuery, user.ID)

	return user, nil
}
