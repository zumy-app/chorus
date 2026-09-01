package services

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
)

const settingsTestColumns = "user_id, grammar_enabled, vocabulary_enabled, difficulty_level, transcript_recording, message_retention_days, translation_enabled, grammar_auto, highlights_enabled, last_seen_visibility, profile_photo_visibility, contacts_visibility, updated_at"

func settingsRow(userID string, translationEnabled, grammarAuto, highlights bool) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"user_id", "grammar_enabled", "vocabulary_enabled", "difficulty_level", "transcript_recording", "message_retention_days", "translation_enabled", "grammar_auto", "highlights_enabled", "last_seen_visibility", "profile_photo_visibility", "contacts_visibility", "updated_at"}).
		AddRow(userID, true, true, "intermediate", true, 365, translationEnabled, grammarAuto, highlights, "everyone", "everyone", "everyone", time.Now())
}

func TestSettingsService_GetSettings_DefaultsOnFirstAccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewSettingsService(db)

	// First access: the default row is inserted (no-op on conflict), then read.
	mock.ExpectExec(`INSERT INTO user_settings \(user_id\) VALUES \(\$1\) ON CONFLICT \(user_id\) DO NOTHING`).
		WithArgs("user-1").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT ` + settingsTestColumns + ` FROM user_settings WHERE user_id = \$1`).
		WithArgs("user-1").WillReturnRows(settingsRow("user-1", true, true, true))

	st, err := s.GetSettings("user-1")
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}
	if !st.TranslationEnabled || !st.GrammarAuto || !st.HighlightsEnabled {
		t.Fatalf("expected feature toggles to default to true, got %+v", st)
	}
	if st.UserID != "user-1" {
		t.Fatalf("expected user-1, got %s", st.UserID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSettingsService_GetFeatureSettings(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewSettingsService(db)

	mock.ExpectExec(`INSERT INTO user_settings \(user_id\) VALUES \(\$1\) ON CONFLICT \(user_id\) DO NOTHING`).
		WithArgs("user-1").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT ` + settingsTestColumns + ` FROM user_settings WHERE user_id = \$1`).
		WithArgs("user-1").WillReturnRows(settingsRow("user-1", false, true, false))

	fs, err := s.GetFeatureSettings("user-1")
	if err != nil {
		t.Fatalf("GetFeatureSettings failed: %v", err)
	}
	if fs.TranslationEnabled {
		t.Fatal("expected translation disabled")
	}
	if !fs.GrammarAuto {
		t.Fatal("expected grammar auto enabled")
	}
	if fs.HighlightsEnabled {
		t.Fatal("expected highlights disabled")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSettingsService_UpdateFeatureSettings_Partial(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewSettingsService(db)

	mock.ExpectExec(`INSERT INTO user_settings \(user_id\) VALUES \(\$1\) ON CONFLICT \(user_id\) DO NOTHING`).
		WithArgs("user-1").WillReturnResult(sqlmock.NewResult(1, 1))

	toggle := false
	mock.ExpectExec(`UPDATE user_settings\s*SET translation_enabled = COALESCE\(\$2, translation_enabled\),\s*grammar_auto = COALESCE\(\$3, grammar_auto\),\s*highlights_enabled = COALESCE\(\$4, highlights_enabled\),\s*last_seen_visibility = COALESCE\(\$5, last_seen_visibility\),\s*profile_photo_visibility = COALESCE\(\$6, profile_photo_visibility\),\s*contacts_visibility = COALESCE\(\$7, contacts_visibility\),\s*updated_at = CURRENT_TIMESTAMP\s*WHERE user_id = \$1`).
		WithArgs("user-1", &toggle, nil, nil, nil, nil, nil).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT ` + settingsTestColumns + ` FROM user_settings WHERE user_id = \$1`).
		WithArgs("user-1").WillReturnRows(settingsRow("user-1", false, true, true))

	st, err := s.UpdateFeatureSettings("user-1", models.UpdateFeatureSettingsRequest{
		TranslationEnabled: &toggle,
	})
	if err != nil {
		t.Fatalf("UpdateFeatureSettings failed: %v", err)
	}
	if st.TranslationEnabled {
		t.Fatal("expected translation disabled after update")
	}
	if !st.HighlightsEnabled {
		t.Fatal("expected highlights left unchanged (enabled)")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSettingsService_UpdateFeatureSettings_All(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewSettingsService(db)

	mock.ExpectExec(`INSERT INTO user_settings \(user_id\) VALUES \(\$1\) ON CONFLICT \(user_id\) DO NOTHING`).
		WithArgs("user-1").WillReturnResult(sqlmock.NewResult(1, 1))

	on, off := true, false
	mock.ExpectExec(`UPDATE user_settings\s*SET translation_enabled = COALESCE\(\$2, translation_enabled\),\s*grammar_auto = COALESCE\(\$3, grammar_auto\),\s*highlights_enabled = COALESCE\(\$4, highlights_enabled\),\s*last_seen_visibility = COALESCE\(\$5, last_seen_visibility\),\s*profile_photo_visibility = COALESCE\(\$6, profile_photo_visibility\),\s*contacts_visibility = COALESCE\(\$7, contacts_visibility\),\s*updated_at = CURRENT_TIMESTAMP\s*WHERE user_id = \$1`).
		WithArgs("user-1", &off, &on, &off, nil, nil, nil).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT ` + settingsTestColumns + ` FROM user_settings WHERE user_id = \$1`).
		WithArgs("user-1").WillReturnRows(settingsRow("user-1", false, true, false))

	st, err := s.UpdateFeatureSettings("user-1", models.UpdateFeatureSettingsRequest{
		TranslationEnabled: &off,
		GrammarAuto:        &on,
		HighlightsEnabled:  &off,
	})
	if err != nil {
		t.Fatalf("UpdateFeatureSettings failed: %v", err)
	}
	if st.TranslationEnabled || !st.GrammarAuto || st.HighlightsEnabled {
		t.Fatalf("unexpected settings after update: %+v", st)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
