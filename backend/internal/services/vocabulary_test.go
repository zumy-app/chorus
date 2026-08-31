package services

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// TestLearnedWords verifies the FR-26 known-words resolver: it reads terms from
// the vocabulary table where interval_days is at/above the learned threshold,
// then returns the normalized, de-duplicated set that the translation pipeline
// uses to skip known words (e.g. "hola"/"hi" are never re-translated).
func TestLearnedWords(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	svc := NewVocabularyService(db, nil)

	mock.ExpectQuery(`SELECT term FROM vocabulary WHERE user_id = \$1 AND interval_days >= \$2`).
		WithArgs("user-1", learnedWordThresholdDays).
		WillReturnRows(sqlmock.NewRows([]string{"term"}).
			AddRow("Hola").
			AddRow("Mundo").
			AddRow("hola").
			AddRow("!!!"))

	set, err := svc.LearnedWords("user-1")
	if err != nil {
		t.Fatalf("LearnedWords failed: %v", err)
	}
	if len(set) != 2 {
		t.Fatalf("expected 2 normalized learned words, got %d: %v", len(set), set)
	}
	for _, w := range []string{"hola", "mundo"} {
		if _, ok := set[w]; !ok {
			t.Fatalf("expected normalized term %q in learned set, got %v", w, set)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestLearnedWords_EmptyBank verifies the resolver returns an empty set when
// the user has no vocabulary rows at/above the learned threshold.
func TestLearnedWords_EmptyBank(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	svc := NewVocabularyService(db, nil)

	mock.ExpectQuery(`SELECT term FROM vocabulary WHERE user_id = \$1 AND interval_days >= \$2`).
		WithArgs("user-2", learnedWordThresholdDays).
		WillReturnRows(sqlmock.NewRows([]string{"term"}))

	set, err := svc.LearnedWords("user-2")
	if err != nil {
		t.Fatalf("LearnedWords failed: %v", err)
	}
	if len(set) != 0 {
		t.Fatalf("expected empty learned set, got %v", set)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
