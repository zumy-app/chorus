package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// TestStaggerSeedReview verifies the seed drip-feed: the first batch is due
// immediately and subsequent cards are spaced a day at a time.
func TestStaggerSeedReview(t *testing.T) {
	now := time.Now()
	for i := 0; i < seedInitialDue; i++ {
		if got := staggerSeedReview(now, i); !got.Before(now.Add(2 * time.Hour)) {
			t.Fatalf("index %d should be due now, got %v", i, got)
		}
	}
	// index seedInitialDue -> now+1 day, seedInitialDue+1 -> now+2 days.
	if got := staggerSeedReview(now, seedInitialDue); !got.After(now.Add(20 * time.Hour)) {
		t.Errorf("expected next-day spacing, got %v", got)
	}
	if got := staggerSeedReview(now, seedInitialDue+1); got.Before(staggerSeedReview(now, seedInitialDue).Add(20 * time.Hour)) {
		t.Errorf("expected +2 day spacing, got %v", got)
	}
}

// TestEnsureSeeded_VocabOnly verifies that a pair without a curated course
// (no learning_pair_capabilities row => vocab_only fallback) yields no seed
// cards and no error, so the queue is personal-only.
func TestEnsureSeeded_VocabOnlyNoRow(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`FROM learning_pair_capabilities`).
		WithArgs("en", "es").
		WillReturnError(sql.ErrNoRows)

	s := NewSeedQueueService(db, nil, NewLearningCapabilityService(db))
	count, err := s.EnsureSeeded(context.Background(), "user-1", "en", "es")
	if err != nil {
		t.Fatalf("EnsureSeeded: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 seeded cards for vocab-only pair, got %d", count)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestSeedUnits_ActiveUnitWindow verifies the seed unit picker starts at the
// learner's active unit and takes up to seedUnitWindow units.
func TestSeedUnits_ActiveUnitWindow(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id::text FROM curriculum_units`).
		WithArgs("course-1", "unit-2", seedUnitWindow).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("unit-2").AddRow("unit-3").AddRow("unit-4"))

	s := NewSeedQueueService(db, nil, NewLearningCapabilityService(db))
	ids, err := s.seedUnits(context.Background(), "course-1", "unit-2")
	if err != nil {
		t.Fatalf("seedUnits: %v", err)
	}
	if len(ids) != 3 || ids[0] != "unit-2" {
		t.Fatalf("unexpected unit window: %v", ids)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestInsertSeedCard verifies a new curriculum lexical item is materialized as
// a vocabulary card (seed path).
func TestInsertSeedCard(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	// Dedupe pre-check: no existing card.
	mock.ExpectQuery(`SELECT id::text FROM vocabulary`).
		WithArgs("user-1", "es", "cafe con leche").
		WillReturnError(sql.ErrNoRows)
	// Insert returns the new card id.
	mock.ExpectQuery(`INSERT INTO vocabulary`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("card-1"))

	s := NewSeedQueueService(db, nil, NewLearningCapabilityService(db))
	inserted, err := s.insertSeedCard(context.Background(), "user-1", "es", "lex-1", "unit-1", "café con leche", "café con leche", "noun_phrase", "A1", "coffee with milk", true, time.Now())
	if err != nil {
		t.Fatalf("insertSeedCard: %v", err)
	}
	if !inserted {
		t.Fatalf("expected a new seed card to be inserted")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestInsertSeedCard_Dedupes verifies seeding never duplicates an existing card.
func TestInsertSeedCard_Dedupes(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id::text FROM vocabulary`).
		WithArgs("user-1", "es", "hola").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("existing-card"))

	s := NewSeedQueueService(db, nil, NewLearningCapabilityService(db))
	inserted, err := s.insertSeedCard(context.Background(), "user-1", "es", "lex-1", "unit-1", "hola", "hola", "interjection", "A1", "hello", false, time.Now())
	if err != nil {
		t.Fatalf("insertSeedCard: %v", err)
	}
	if inserted {
		t.Fatalf("expected an existing card to be skipped")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
