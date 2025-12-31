package services

import (
	"database/sql"
	"encoding/json"

	"github.com/chorus/messenger/internal/models"
	"github.com/lib/pq"
)

type UserService struct {
	db *sql.DB
}

func NewUserService(db *sql.DB) *UserService {
	return &UserService{db: db}
}

func (s *UserService) GetByID(userID string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, username, email, display_name, native_language, target_languages, created_at, last_active_at
		FROM users
		WHERE id = $1
	`

	err := s.db.QueryRow(query, userID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.DisplayName,
		&user.NativeLanguage,
		pq.Array(&user.TargetLanguages),
		&user.CreatedAt,
		&user.LastActiveAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) Update(userID string, req models.UpdateUserRequest) (*models.User, error) {
	user := &models.User{}
	query := `
		UPDATE users
		SET display_name = COALESCE(NULLIF($2, ''), display_name),
		    target_languages = COALESCE($3, target_languages)
		WHERE id = $1
		RETURNING id, username, email, display_name, native_language, target_languages, created_at, last_active_at
	`

	var displayName *string
	if req.DisplayName != "" {
		displayName = &req.DisplayName
	}

	err := s.db.QueryRow(query, userID, displayName, pq.Array(req.TargetLanguages)).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.DisplayName,
		&user.NativeLanguage,
		pq.Array(&user.TargetLanguages),
		&user.CreatedAt,
		&user.LastActiveAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) Search(query string, limit int) ([]models.User, error) {
	users := []models.User{}

	sqlQuery := `
		SELECT id, username, email, display_name, native_language, target_languages, created_at, last_active_at
		FROM users
		WHERE username ILIKE $1 OR display_name ILIKE $1
		LIMIT $2
	`

	rows, err := s.db.Query(sqlQuery, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.DisplayName,
			&user.NativeLanguage,
			pq.Array(&user.TargetLanguages),
			&user.CreatedAt,
			&user.LastActiveAt,
		)
		if err != nil {
			continue
		}
		users = append(users, user)
	}

	return users, nil
}

func (s *UserService) GetMultiple(userIDs []string) (map[string]*models.User, error) {
	users := make(map[string]*models.User)

	if len(userIDs) == 0 {
		return users, nil
	}

	query := `
		SELECT id, username, email, display_name, native_language, target_languages, created_at, last_active_at
		FROM users
		WHERE id = ANY($1)
	`

	rows, err := s.db.Query(query, pq.Array(userIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.DisplayName,
			&user.NativeLanguage,
			pq.Array(&user.TargetLanguages),
			&user.CreatedAt,
			&user.LastActiveAt,
		)
		if err != nil {
			continue
		}
		users[user.ID] = user
	}

	return users, nil
}

// Helper to convert JSON field
func scanJSON(src interface{}, dest interface{}) error {
	if src == nil {
		return nil
	}
	
	bytes, ok := src.([]byte)
	if !ok {
		return nil
	}
	
	return json.Unmarshal(bytes, dest)
}
