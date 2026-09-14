package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io/fs"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSeedRunner_NormalizePair(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"en-es", "en_es"},
		{"EN-ES", "en_es"},
		{"en_es", "en_es"},
		{" ES-EN ", "es_en"},
	}

	for _, tc := range cases {
		got := normalizePair(tc.input)
		if got != tc.want {
			t.Errorf("normalizePair(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestSeedRunner_SkipsUnchangedFiles(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	runner := NewSeedRunner(db)
	ctx := context.Background()

	entries, err := fs.Glob(SeedFiles, "seeds/en_es/*.sql")
	if err != nil {
		t.Fatalf("failed to glob embedded seeds: %v", err)
	}

	// Mock DB returns matching checksum for each file so all are skipped
	for _, path := range entries {
		content, err := SeedFiles.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read embedded seed %s: %v", path, err)
		}
		sum := fmt.Sprintf("%x", sha256.Sum256(content))

		mock.ExpectQuery(`SELECT checksum FROM seed_checksums WHERE file = \$1`).
			WithArgs(path).
			WillReturnRows(sqlmock.NewRows([]string{"checksum"}).AddRow(sum))
	}

	applied, err := runner.Run(ctx, "en-es")
	if err != nil {
		t.Fatalf("runner.Run unexpected error: %v", err)
	}

	if len(applied) != 0 {
		t.Errorf("expected 0 files applied when all unchanged, got: %v", applied)
	}
}

func TestSeedRunner_Status(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	runner := NewSeedRunner(db)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"file", "checksum"}).
		AddRow("seeds/en_es/01_course.sql", "abcdef123456").
		AddRow("seeds/en_es/02_units.sql", "789012abcdef")

	mock.ExpectQuery(`SELECT file, checksum FROM seed_checksums ORDER BY file`).
		WillReturnRows(rows)

	status, err := runner.Status(ctx)
	if err != nil {
		t.Fatalf("runner.Status unexpected error: %v", err)
	}

	if len(status) != 2 {
		t.Errorf("got %d status entries, want 2", len(status))
	}
	if status["seeds/en_es/01_course.sql"] != "abcdef123456" {
		t.Errorf("unexpected checksum for 01_course.sql: %s", status["seeds/en_es/01_course.sql"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestSeedRunner_ExecutesNewFile(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	runner := NewSeedRunner(db)
	ctx := context.Background()

	entries, err := fs.Glob(SeedFiles, "seeds/en_es/*.sql")
	if err != nil {
		t.Fatalf("failed to glob embedded seeds: %v", err)
	}

	// 01_course.sql not yet in seed_checksums (returns ErrNoRows)
	mock.ExpectQuery(`SELECT checksum FROM seed_checksums WHERE file = \$1`).
		WithArgs(entries[0]).
		WillReturnError(sql.ErrNoRows)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO curriculum_courses`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO seed_checksums`).
		WithArgs(entries[0], sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// Remaining files return matching real hash so they skip
	for _, path := range entries[1:] {
		content, err := SeedFiles.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read embedded seed %s: %v", path, err)
		}
		sum := fmt.Sprintf("%x", sha256.Sum256(content))

		mock.ExpectQuery(`SELECT checksum FROM seed_checksums WHERE file = \$1`).
			WithArgs(path).
			WillReturnRows(sqlmock.NewRows([]string{"checksum"}).AddRow(sum))
	}

	applied, err := runner.Run(ctx, "en-es")
	if err != nil {
		t.Fatalf("runner.Run unexpected error: %v", err)
	}

	if len(applied) != 1 || applied[0] != entries[0] {
		t.Errorf("expected %s to be applied, got: %v", entries[0], applied)
	}
}
