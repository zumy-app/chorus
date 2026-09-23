package database

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestMigrations_IdempotentPatterns guards the daily-release invariant: Migrate()
// runs on every backend boot, so every statement must be safe to re-run.
// This static test fails the PR if a future migration is added without
// IF NOT EXISTS / IF EXISTS guards (the exact class of bug that breaks
// zero-downtime daily deploys with "already exists" errors).
func TestMigrations_IdempotentPatterns(t *testing.T) {
	path := filepath.Join("postgres.go")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read postgres.go: %v", err)
	}
	content := string(src)

	// Extract the migration strings: backtick-quoted SQL literals inside Migrate.
	// We check each statement individually so failures point at the bad statement.
	stmtRe := regexp.MustCompile("`([^`]+)`")
	stmts := stmtRe.FindAllStringSubmatch(content, -1)
	if len(stmts) == 0 {
		t.Fatal("no SQL migration strings found in postgres.go")
	}

	failures := 0
	for _, m := range stmts {
		stmt := strings.TrimSpace(m[1])
		upper := strings.ToUpper(stmt)
		if !strings.HasPrefix(upper, "CREATE TABLE") &&
			!strings.HasPrefix(upper, "CREATE INDEX") &&
			!strings.HasPrefix(upper, "CREATE UNIQUE INDEX") &&
			!strings.HasPrefix(upper, "ALTER TABLE") &&
			!strings.HasPrefix(upper, "CREATE EXTENSION") &&
			!strings.HasPrefix(upper, "INSERT INTO") &&
			!strings.HasPrefix(upper, "DO $$") &&
			!strings.HasPrefix(upper, "DO$") {
			continue // Go code or non-migration literal; ignore.
		}

		snip := stmt
		if len(snip) > 80 {
			snip = snip[:80] + "..."
		}

		switch {
		case strings.HasPrefix(upper, "CREATE TABLE"):
			if !strings.Contains(upper, "IF NOT EXISTS") {
				t.Errorf("CREATE TABLE without IF NOT EXISTS: %q", snip)
				failures++
			}
		case strings.HasPrefix(upper, "CREATE INDEX") || strings.HasPrefix(upper, "CREATE UNIQUE INDEX"):
			if !strings.Contains(upper, "IF NOT EXISTS") {
				t.Errorf("CREATE INDEX without IF NOT EXISTS: %q", snip)
				failures++
			}
		case strings.HasPrefix(upper, "CREATE EXTENSION"):
			if !strings.Contains(upper, "IF NOT EXISTS") {
				t.Errorf("CREATE EXTENSION without IF NOT EXISTS: %q", snip)
				failures++
			}
		case strings.Contains(upper, "ADD COLUMN"):
			if !strings.Contains(upper, "ADD COLUMN IF NOT EXISTS") {
				t.Errorf("ADD COLUMN without IF NOT EXISTS: %q", snip)
				failures++
			}
		case strings.Contains(upper, "DROP COLUMN"):
			if !strings.Contains(upper, "DROP COLUMN IF EXISTS") {
				t.Errorf("DROP COLUMN without IF EXISTS: %q", snip)
				failures++
			}
		case strings.Contains(upper, "DROP CONSTRAINT"):
			if !strings.Contains(upper, "DROP CONSTRAINT IF EXISTS") {
				t.Errorf("DROP CONSTRAINT without IF EXISTS: %q", snip)
				failures++
			}
			// ADD CONSTRAINT after a DROP ... IF EXISTS in the same statement list
			// is the sanctioned idempotent pattern (drop-then-add); bare ADD
			// CONSTRAINT without a preceding drop is flagged below via the
			// standalone check.
		}
	}

	// Standalone ADD CONSTRAINT statements must be paired with a DROP IF EXISTS
	// for the same constraint name somewhere in the migration list (drop-then-add
	// is idempotent; add-alone fails on second boot).
	addRe := regexp.MustCompile(`(?i)ADD CONSTRAINT\s+(\w+)`)
	dropRe := regexp.MustCompile(`(?i)DROP CONSTRAINT IF EXISTS\s+(\w+)`)
	drops := map[string]bool{}
	for _, m := range dropRe.FindAllStringSubmatch(content, -1) {
		drops[strings.ToLower(m[1])] = true
	}
	for _, m := range addRe.FindAllStringSubmatch(content, -1) {
		name := strings.ToLower(m[1])
		if !drops[name] {
			t.Errorf("ADD CONSTRAINT %q has no matching DROP CONSTRAINT IF EXISTS (not re-runnable)", m[1])
			failures++
		}
	}

	if failures > 0 {
		t.Fatalf("%d non-idempotent migration statement(s) found", failures)
	}
}

// TestMigrations_DoubleRunLive runs Migrate() twice against a real Postgres when
// DATABASE_URL is set (CI mobile-blackbox job provides it). Skipped otherwise so
// PR gates without Docker stay fast and green.
func TestMigrations_DoubleRunLive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live double-migrate in short mode (covered by mobile-blackbox CI job)")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping live migration double-run")
	}
	db, err := Connect(dsn)
	if err != nil {
		t.Skipf("postgres not reachable: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("first Migrate failed: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("second Migrate failed (migrations not idempotent): %v", err)
	}
}
