package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/lib/pq"
)

type UserService struct {
	db *sql.DB
}

func NewUserService(db *sql.DB) *UserService {
	return &UserService{db: db}
}

const userColumns = "id, username, email, display_name, native_language, target_language_level, target_languages, role, created_at, last_active_at, suspended_at, deleted_at, plan, plan_grace_until, premium_since, subscription_id, subscription_provider, subscription_plan_id, subscription_status, next_billing_date, last_payment_at"

// scanUser scans one row of the userColumns projection into a User.
func scanUser(sc interface{ Scan(...interface{}) error }) (*models.User, error) {
	user := &models.User{}
	err := sc.Scan(
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
		return nil, err
	}
	return user, nil
}

func (s *UserService) GetByID(userID string) (*models.User, error) {
	return scanUser(s.db.QueryRow(`SELECT `+userColumns+` FROM users WHERE id = $1`, userID))
}

func (s *UserService) Update(userID string, req models.UpdateUserRequest) (*models.User, error) {
	return scanUser(s.db.QueryRow(`
		UPDATE users
		SET display_name = COALESCE(NULLIF($2, ''), display_name),
		    native_language = COALESCE(NULLIF($3, ''), native_language),
		    target_language_level = COALESCE(NULLIF($4, ''), target_language_level),
		    target_languages = COALESCE($5, target_languages)
		WHERE id = $1
		RETURNING `+userColumns+`
	`, userID, nilIfEmpty(req.DisplayName), nilIfEmpty(req.NativeLanguage), nilIfEmpty(req.TargetLanguageLevel), pq.Array(req.TargetLanguages)))
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (s *UserService) Search(query string, limit int) ([]models.User, error) {
	rows, err := s.db.Query(`
		SELECT `+userColumns+`
		FROM users
		WHERE (username ILIKE $1 OR display_name ILIKE $1 OR email ILIKE $1)
		  AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2
	`, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []models.User{}
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			continue
		}
		users = append(users, *user)
	}
	return users, rows.Err()
}

func (s *UserService) GetMultiple(userIDs []string) (map[string]*models.User, error) {
	users := make(map[string]*models.User)
	if len(userIDs) == 0 {
		return users, nil
	}
	rows, err := s.db.Query(`SELECT `+userColumns+` FROM users WHERE id = ANY($1)`, pq.Array(userIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			continue
		}
		users[user.ID] = user
	}
	return users, rows.Err()
}

// ListUsers returns a paginated, filterable directory of non-deleted users.
// Filters: q (substring on username/email/display name), role, status
// (active|suspended), limit (default 50, max 200), offset.
func (s *UserService) ListUsers(q, role, status string, limit, offset int) ([]models.User, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	query := `SELECT ` + userColumns + ` FROM users WHERE deleted_at IS NULL`
	args := []interface{}{}
	addArg := func(v interface{}) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	if q != "" {
		query += fmt.Sprintf(` AND (username ILIKE %s OR display_name ILIKE %s OR email ILIKE %s)`,
			addArg("%"+q+"%"), addArg("%"+q+"%"), addArg("%"+q+"%"))
	}
	if ValidRole(role) {
		query += fmt.Sprintf(` AND role = %s`, addArg(role))
	}
	switch status {
	case "suspended":
		query += ` AND suspended_at IS NOT NULL`
	case "active":
		query += ` AND suspended_at IS NULL`
	}
	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT %s OFFSET %s`, addArg(limit), addArg(offset))

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := []models.User{}
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			continue
		}
		users = append(users, *user)
	}
	return users, rows.Err()
}

// SetRole assigns a role to a user and returns the updated role.
func (s *UserService) SetRole(userID, role string) error {
	if !ValidRole(role) {
		return fmt.Errorf("invalid role %q", role)
	}
	result, err := s.db.Exec(`UPDATE users SET role = $1 WHERE id = $2 AND deleted_at IS NULL`, role, userID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// SetSuspended toggles the soft-ban flag for a user.
func (s *UserService) SetSuspended(userID string, suspended bool) error {
	var query string
	if suspended {
		query = `UPDATE users SET suspended_at = CURRENT_TIMESTAMP WHERE id = $1 AND deleted_at IS NULL`
	} else {
		query = `UPDATE users SET suspended_at = NULL WHERE id = $1`
	}
	result, err := s.db.Exec(query, userID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// SetDeleted soft-deletes a user: the account is blocked from authenticating
// and hidden from the user directory, but their chats and history remain.
func (s *UserService) SetDeleted(userID string) error {
	result, err := s.db.Exec(`UPDATE users SET deleted_at = CURRENT_TIMESTAMP, suspended_at = NULL WHERE id = $1 AND deleted_at IS NULL`, userID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// IsActive reports whether the user can authenticate (not suspended/deleted).
func (s *UserService) IsActive(userID string) (bool, error) {
	var suspendedAt, deletedAt *time.Time
	err := s.db.QueryRow(`SELECT suspended_at, deleted_at FROM users WHERE id = $1`, userID).
		Scan(&suspendedAt, &deletedAt)
	if err != nil {
		return false, err
	}
	return suspendedAt == nil && deletedAt == nil, nil
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
