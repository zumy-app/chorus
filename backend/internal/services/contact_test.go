package services

import (
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
	"github.com/lib/pq"
)

func TestHashIdentifier_NormalizesAndIsStable(t *testing.T) {
	// Same raw identifier with different casing/whitespace must hash identically,
	// so a client hash always matches the server-computed hash.
	a := HashIdentifier("  Alice@Example.COM ")
	b := HashIdentifier("alice@example.com")
	if a != b {
		t.Fatalf("expected identical hashes for normalized same identifier: %s vs %s", a, b)
	}
	if len(a) != 64 {
		t.Fatalf("expected 64-char sha256 hex, got %d chars", len(a))
	}
	if HashIdentifier("alice@example.com") == HashIdentifier("bob@example.com") {
		t.Fatal("expected different hashes for different identifiers")
	}
}

func TestContactScanHashed_MatchesOnPlatform(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewContactService(db)

	aliceHash := HashIdentifier("alice@example.com")
	bobHash := HashIdentifier("bob@example.com")
	// carol's email hash is NOT in the submitted set -> should not be returned.

	query := regexp.QuoteMeta(`
		SELECT id, username, email, display_name, native_language, target_languages
		FROM users
		WHERE id != $1 AND deleted_at IS NULL AND suspended_at IS NULL`)
	mock.ExpectQuery(query).
		WithArgs("requester-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "username", "email", "display_name", "native_language", "target_languages",
		}).
			AddRow("user-alice", "alice", "alice@example.com", "Alice", "en", pq.Array([]string{"es"})).
			AddRow("user-bob", "bob", "bob@example.com", "Bob", "en", pq.Array([]string{"fr"})).
			AddRow("user-carol", "carol", "carol@example.com", "Carol", "en", pq.Array([]string{"de"})))

	matches, err := s.ScanHashed("requester-1", []string{aliceHash, bobHash, HashIdentifier("nobody@example.com")})
	if err != nil {
		t.Fatalf("ScanHashed failed: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d: %v", len(matches), matches)
	}
	byHash := map[string]models.ContactMatch{}
	for _, m := range matches {
		byHash[m.EmailHash] = m
	}
	if _, ok := byHash[aliceHash]; !ok {
		t.Fatalf("expected alice matched: %+v", matches)
	}
	if _, ok := byHash[bobHash]; !ok {
		t.Fatalf("expected bob matched: %+v", matches)
	}
	for _, m := range matches {
		if m.EmailHash == HashIdentifier("carol@example.com") {
			t.Fatalf("carol should NOT be matched: %+v", matches)
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestContactScanHashed_EmptyReturnsNone(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewContactService(db)
	matches, err := s.ScanHashed("requester-1", nil)
	if err != nil {
		t.Fatalf("ScanHashed(nil) failed: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(matches))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestContactScanHashed_TooManyHashes(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewContactService(db)
	hashes := make([]string, MaxContactScanHashes+1)
	_, err = s.ScanHashed("requester-1", hashes)
	if !errors.Is(err, ErrTooManyContactHashes) {
		t.Fatalf("expected ErrTooManyContactHashes, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
