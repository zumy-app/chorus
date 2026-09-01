package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

const settingsTestColumns = "user_id, grammar_enabled, vocabulary_enabled, difficulty_level, transcript_recording, message_retention_days, translation_enabled, grammar_auto, highlights_enabled, last_seen_visibility, profile_photo_visibility, contacts_visibility, updated_at"

func TestSettingsHandler_GetSettings(t *testing.T) {
	router := setupTestRouter()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	settingsService := services.NewSettingsService(db)
	h := NewSettingsHandler(settingsService)

	mock.ExpectExec(`INSERT INTO user_settings \(user_id\) VALUES \(\$1\) ON CONFLICT \(user_id\) DO NOTHING`).
		WithArgs("user-1").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT ` + settingsTestColumns + ` FROM user_settings WHERE user_id = $1`)).
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "grammar_enabled", "vocabulary_enabled", "difficulty_level", "transcript_recording", "message_retention_days", "translation_enabled", "grammar_auto", "highlights_enabled", "last_seen_visibility", "profile_photo_visibility", "contacts_visibility", "updated_at"}).
			AddRow("user-1", true, true, "intermediate", true, 365, true, true, true, "everyone", "everyone", "everyone", time.Now()))

	router.GET("/users/me/settings", func(c *gin.Context) {
		c.Set("userID", "user-1")
		h.GetSettings(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/users/me/settings", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["translationEnabled"] != true {
		t.Fatalf("expected translationEnabled true, got %v", resp["translationEnabled"])
	}
	if resp["grammarAuto"] != true {
		t.Fatalf("expected grammarAuto true, got %v", resp["grammarAuto"])
	}
	if resp["highlightsEnabled"] != true {
		t.Fatalf("expected highlightsEnabled true, got %v", resp["highlightsEnabled"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSettingsHandler_UpdateSettings(t *testing.T) {
	router := setupTestRouter()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	settingsService := services.NewSettingsService(db)
	h := NewSettingsHandler(settingsService)

	mock.ExpectExec(`INSERT INTO user_settings \(user_id\) VALUES \(\$1\) ON CONFLICT \(user_id\) DO NOTHING`).
		WithArgs("user-1").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`UPDATE user_settings\s*SET translation_enabled = COALESCE\(\$2, translation_enabled\),\s*grammar_auto = COALESCE\(\$3, grammar_auto\),\s*highlights_enabled = COALESCE\(\$4, highlights_enabled\),\s*last_seen_visibility = COALESCE\(\$5, last_seen_visibility\),\s*profile_photo_visibility = COALESCE\(\$6, profile_photo_visibility\),\s*contacts_visibility = COALESCE\(\$7, contacts_visibility\),\s*transcript_recording = COALESCE\(\$8, transcript_recording\),\s*message_retention_days = COALESCE\(\$9, message_retention_days\),\s*updated_at = CURRENT_TIMESTAMP\s*WHERE user_id = \$1`).
		WithArgs("user-1", false, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT ` + settingsTestColumns + ` FROM user_settings WHERE user_id = $1`)).
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "grammar_enabled", "vocabulary_enabled", "difficulty_level", "transcript_recording", "message_retention_days", "translation_enabled", "grammar_auto", "highlights_enabled", "last_seen_visibility", "profile_photo_visibility", "contacts_visibility", "updated_at"}).
			AddRow("user-1", true, true, "intermediate", true, 365, false, true, true, "everyone", "everyone", "everyone", time.Now()))

	router.PUT("/users/me/settings", func(c *gin.Context) {
		c.Set("userID", "user-1")
		h.UpdateSettings(c)
	})

	body := bytes.NewBufferString(`{"translationEnabled":false}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/users/me/settings", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["translationEnabled"] != false {
		t.Fatalf("expected translationEnabled false, got %v", resp["translationEnabled"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSettingsHandler_UpdateSettings_Invalid(t *testing.T) {
	router := setupTestRouter()

	h := NewSettingsHandler(services.NewSettingsService(nil))
	router.PUT("/users/me/settings", func(c *gin.Context) {
		c.Set("userID", "user-1")
		h.UpdateSettings(c)
	})

	body := bytes.NewBufferString(`{not-json}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/users/me/settings", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", w.Code)
	}
}
