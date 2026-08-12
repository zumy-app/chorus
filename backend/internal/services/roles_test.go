package services

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRoleRankAndAtLeast(t *testing.T) {
	cases := []struct {
		role string
		min  string
		want bool
	}{
		{"admin", "member", true},
		{"admin", "moderator", true},
		{"admin", "admin", true},
		{"moderator", "member", true},
		{"moderator", "admin", false},
		{"member", "member", true},
		{"member", "moderator", false},
		{"", "member", false},
		{"superuser", "member", false},
	}
	for _, c := range cases {
		if got := RoleAtLeast(c.role, c.min); got != c.want {
			t.Errorf("RoleAtLeast(%q, %q) = %v, want %v", c.role, c.min, got, c.want)
		}
	}
}

func TestValidRole(t *testing.T) {
	for _, r := range []string{RoleMember, RoleModerator, RoleAdmin} {
		if !ValidRole(r) {
			t.Errorf("expected %q to be valid", r)
		}
	}
	for _, r := range []string{"", "owner", "guest"} {
		if ValidRole(r) {
			t.Errorf("expected %q to be invalid", r)
		}
	}
}

func TestEnsureAdminRoles(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(`UPDATE users SET role = \$1 WHERE LOWER\(email\) = \$2 AND deleted_at IS NULL`).
		WithArgs(RoleAdmin, "uhsarp@gmail.com").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users WHERE LOWER\(email\) = \$1`).
		WithArgs("uhsarp@gmail.com").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	if err := EnsureAdminRoles(db, []string{"  UHSARP@Gmail.com "}); err != nil {
		t.Fatalf("EnsureAdminRoles failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestEnsureAdminRolesEmpty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	if err := EnsureAdminRoles(db, nil); err != nil {
		t.Fatalf("EnsureAdminRoles(nil) failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
