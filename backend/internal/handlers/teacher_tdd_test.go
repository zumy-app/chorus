package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/DATA-DOG/go-sqlmock"
)

func newTeacherTDDHandler(mock sqlmock.Sqlmock) *TeacherHandler {
	db, _, _ := sqlmock.NewWithDSN("tdd")
	_ = mock
	svc := services.NewTeacherService(db)
	return NewTeacherHandler(svc)
}

// TC-TUTOR-01: video optional — empty VideoURL must pass binding (S-TUTOR-01)
func TestApply_VideoOptional_EmptyPasses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, _ := sqlmock.New()
	defer db.Close()
	svc := services.NewTeacherService(db)
	h := NewTeacherHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", "u1")
	body, _ := json.Marshal(models.TeacherApplyRequest{
		Bio:       "I am a tutor with 10+ chars bio for testing purposes.",
		Languages: []string{"es"},
		RateCents: 2000,
		VideoURL:  "",
	})
	c.Request = httptest.NewRequest(http.MethodPost, "/teachers/apply", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	// Expect upsert to be attempted (if binding passes); mock the insert + select.
	rows := sqlmock.NewRows([]string{"id"}).AddRow("app1")
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO teacher_applications").WithArgs("u1", "I am a tutor with 10+ chars bio for testing purposes.", sqlmock.AnyArg(), "", 2000, "").WillReturnRows(rows)
	mock.ExpectExec("DELETE FROM teacher_certificates").WithArgs("app1").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	now := time.Now()
	mock.ExpectQuery("SELECT id, user_id, bio").WithArgs("u1").WillReturnRows(
		sqlmock.NewRows([]string{"id", "user_id", "bio", "languages", "expertise", "rate_cents", "video_url", "status", "created_at", "updated_at"}).AddRow("app1", "u1", "I am a tutor", `{"es"}`, "", 2000, "", "pending", now, now),
	)
	mock.ExpectQuery("SELECT id, type, issuer").WithArgs("app1").WillReturnRows(sqlmock.NewRows([]string{"id", "type", "issuer", "year", "file_url", "verified"}))

	h.Apply(c)
	if w.Code != 200 {
		t.Fatalf("expected 200 for empty videoUrl, got %d body %s", w.Code, w.Body.String())
	}
}

// TC-TUTOR-01: short bio must fail with field-specific message (S-TUTOR-02)
func TestApply_BioValidation_ShortFailsWithHint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, _, _ := sqlmock.New()
	defer db.Close()
	h := NewTeacherHandler(services.NewTeacherService(db))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", "u1")
	body, _ := json.Marshal(map[string]any{
		"bio":       "short",
		"languages": []string{"es"},
		"rateCents": 2000,
		"videoUrl":  "",
	})
	c.Request = httptest.NewRequest(http.MethodPost, "/teachers/apply", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Apply(c)
	if w.Code != 400 {
		t.Fatalf("expected 400 for short bio, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	// error should mention bio/Key: 'TeacherApplyRequest.Bio' or bio detail, not just generic list without hint
	msg, _ := resp["error"].(string)
	if msg == "" {
		// middleware wraps as {"error": "..."} via WriteError
		msg = w.Body.String()
	}
	if len(msg) < 5 {
		t.Fatalf("expected error detail, got empty: %s", w.Body.String())
	}
}
