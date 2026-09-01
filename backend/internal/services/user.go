package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"
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

const userColumns = "id, username, email, display_name, first_name, last_name, native_language, target_languages, role, created_at, last_active_at, suspended_at, deleted_at, plan, plan_grace_until, premium_since, subscription_id, subscription_provider, subscription_plan_id, subscription_status, next_billing_date, last_payment_at, avatar_url, phone, phone_verified, phone_verified_at, two_factor_enabled"

// scanUser scans one row of the userColumns projection into a User.
func scanUser(sc interface{ Scan(...interface{}) error }) (*models.User, error) {
	user := &models.User{}
	err := sc.Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.DisplayName,
		&user.FirstName,
		&user.LastName,
		&user.NativeLanguage,
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
		&user.AvatarURL,
		&user.Phone,
		&user.PhoneVerified,
		&user.PhoneVerifiedAt,
		&user.TwoFactorEnabled,
	)
	if err != nil {
		return nil, err
	}
	// The initials avatar color is never persisted; it is derived from the
	// stable user ID so it is deterministic across clients (REQ 2.2 / FR-20).
	if user.AvatarColor == "" && user.ID != "" {
		user.AvatarColor = DeterministicAvatarColor(user.ID)
	}
	return user, nil
}

// avatarColorPalette is a curated set of hex colors used for the deterministic
// initials avatar. Chosen for readability on both light and dark UI.
var avatarColorPalette = []string{
	"#6366F1", // indigo
	"#8B5CF6", // violet
	"#EC4899", // pink
	"#F43F5E", // rose
	"#F59E0B", // amber
	"#10B981", // emerald
	"#06B6D4", // cyan
	"#3B82F6", // blue
	"#84CC16", // lime
	"#D946EF", // fuchsia
}

// DeterministicAvatarColor returns a stable hex color for the given seed
// (typically the user ID). The same seed always yields the same color, so the
// avatar is consistent across devices, sessions, and clients (REQ 2.2 / FR-20).
func DeterministicAvatarColor(seed string) string {
	if seed == "" {
		return avatarColorPalette[0]
	}
	h := fnv.New32a()
	h.Write([]byte(seed))
	return avatarColorPalette[h.Sum32()%uint32(len(avatarColorPalette))]
}

func (s *UserService) GetByID(userID string) (*models.User, error) {
	return scanUser(s.db.QueryRow(`SELECT `+userColumns+` FROM users WHERE id = $1`, userID))
}

func (s *UserService) Update(userID string, req models.UpdateUserRequest) (*models.User, error) {
	query := `UPDATE users SET first_name = COALESCE(NULLIF($2, ''), first_name), last_name = COALESCE(NULLIF($3, ''), last_name), display_name = COALESCE(NULLIF($4, ''), TRIM(CONCAT_WS(' ', NULLIF(TRIM(COALESCE(NULLIF($2, ''), first_name)), ''), NULLIF(TRIM(COALESCE(NULLIF($3, ''), last_name)), ''))), display_name), native_language = COALESCE(NULLIF($5, ''), native_language), target_languages = COALESCE($6, target_languages) WHERE id = $1 RETURNING ` + userColumns
	return scanUser(s.db.QueryRow(query, userID, nilIfEmpty(req.FirstName), nilIfEmpty(req.LastName), req.DisplayName, nilIfEmpty(req.NativeLanguage), pq.Array(req.TargetLanguages)))
}

// Onboard captures the user's first and last name post-registration and
// persists the composed displayName (first + last) unless the caller supplied
// an explicit override. It is idempotent: calling it again just rewrites the
// structured name and recomposes the displayName.
func (s *UserService) Onboard(userID, firstName, lastName, displayName string) (*models.User, error) {
	if displayName == "" {
		displayName = ComposeDisplayName(firstName, lastName)
	}
	query := `UPDATE users SET first_name = $2, last_name = $3, display_name = $4 WHERE id = $1 RETURNING ` + userColumns
	return scanUser(s.db.QueryRow(query, userID, strings.TrimSpace(firstName), strings.TrimSpace(lastName), strings.TrimSpace(displayName)))
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ComposeDisplayName joins first and last name into a single display name,
// omitting whichever part is blank. Returns "" when both are blank.
func ComposeDisplayName(first, last string) string {
	first = strings.TrimSpace(first)
	last = strings.TrimSpace(last)
	switch {
	case first == "" && last == "":
		return ""
	case first == "":
		return last
	case last == "":
		return first
	default:
		return first + " " + last
	}
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
