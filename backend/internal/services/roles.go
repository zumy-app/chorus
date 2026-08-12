package services

import (
	"database/sql"
	"log"
	"strings"
)

// Application roles, ordered by ascending privilege. Every authenticated user
// has exactly one role; role checks use RoleRank comparisons so a role can
// satisfy any permission at or below its rank.
const (
	RoleMember    = "member"
	RoleModerator = "moderator"
	RoleAdmin     = "admin"
)

// ValidRole reports whether r is one of the known roles.
func ValidRole(r string) bool {
	switch r {
	case RoleMember, RoleModerator, RoleAdmin:
		return true
	}
	return false
}

// RoleRank returns the privilege level of a role (member=1, admin=3).
// Unknown roles rank as 0 (below member).
func RoleRank(r string) int {
	switch r {
	case RoleMember:
		return 1
	case RoleModerator:
		return 2
	case RoleAdmin:
		return 3
	}
	return 0
}

// RoleAtLeast reports whether role satisfies a minimum role requirement.
func RoleAtLeast(role, min string) bool {
	return RoleRank(role) >= RoleRank(min)
}

// EnsureAdminRoles promotes the configured admin email addresses to the admin
// role (no-op if they do not exist yet). Runs at startup so the email allowlist
// continues to work as the authoritative list of administrators. All other
// existing users are left as-is (they default to member on registration).
func EnsureAdminRoles(db *sql.DB, adminEmails []string) error {
	for _, email := range adminEmails {
		email = strings.ToLower(strings.TrimSpace(email))
		if email == "" {
			continue
		}
		if _, err := db.Exec(`UPDATE users SET role = $1
			WHERE LOWER(email) = $2 AND deleted_at IS NULL`, RoleAdmin, email); err != nil {
			return err
		}
	}
	// Emit a one-time warning if an admin email does not match any user, so a
	// typo in the env is obvious instead of silently granting nothing.
	if len(adminEmails) > 0 {
		for _, email := range adminEmails {
			email = strings.ToLower(strings.TrimSpace(email))
			if email == "" {
				continue
			}
			var count int
			if err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE LOWER(email) = $1`, email).Scan(&count); err != nil {
				return err
			}
			if count == 0 {
				log.Printf("[Roles] WAITLIST_ADMIN_EMAILS entry %q does not match any user", email)
			}
		}
	}
	return nil
}
