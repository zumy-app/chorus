package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

func TestSparkyHandler_AskReturns202(t *testing.T) {
	router := setupTestRouter()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	q := services.NewSparkyQueueService(db, nil, nil, nil, nil, nil)
	h := NewSparkyHandler(q)

	mock.ExpectQuery(`INSERT INTO sparky_jobs \(user_id, chat_id, context, query, language, native_language\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6\) RETURNING id`).
		WithArgs("user-1", nil, "ctx", "what?", "es", "en").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("job-1"))

	router.POST("/sparky/ask", func(c *gin.Context) {
		c.Set("userID", "user-1")
		h.AskSparky(c)
	})

	body := bytes.NewBufferString(`{"context":"ctx","query":"what?","language":"es","nativeLanguage":"en"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/sparky/ask", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		JobID  string `json:"jobId"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.JobID != "job-1" || resp.Status != "queued" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSparkyHandler_AskInvalidBody(t *testing.T) {
	router := setupTestRouter()

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	q := services.NewSparkyQueueService(db, nil, nil, nil, nil, nil)
	h := NewSparkyHandler(q)

	router.POST("/sparky/ask", func(c *gin.Context) {
		c.Set("userID", "user-1")
		h.AskSparky(c)
	})

	body := bytes.NewBufferString(`{"context":"","query":""}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/sparky/ask", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSparkyHandler_AskUnauthorized(t *testing.T) {
	router := setupTestRouter()

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	q := services.NewSparkyQueueService(db, nil, nil, nil, nil, nil)
	h := NewSparkyHandler(q)

	router.POST("/sparky/ask", h.AskSparky)

	body := bytes.NewBufferString(`{"context":"ctx","query":"q","language":"es","nativeLanguage":"en"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/sparky/ask", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSparkyHandler_GetJob(t *testing.T) {
	router := setupTestRouter()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	q := services.NewSparkyQueueService(db, nil, nil, nil, nil, nil)
	h := NewSparkyHandler(q)

	mock.ExpectQuery(`FROM sparky_jobs WHERE id = \$1 AND user_id = \$2`).
		WithArgs("job-1", "user-1").
		WillReturnRows(sqlmock.NewRows([]string{"chat_id", "status", "provider_used", "last_error", "result", "attempts"}).
			AddRow("chat-1", "done", "ai", "", "answer!", 1))

	router.GET("/sparky/jobs/:jobId", func(c *gin.Context) {
		c.Set("userID", "user-1")
		h.GetSparkyJob(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/sparky/jobs/job-1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp services.SparkyJobResult
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Content != "answer!" || resp.Status != "done" {
		t.Fatalf("unexpected job: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSparkyHandler_GetJob_NotFound(t *testing.T) {
	router := setupTestRouter()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	q := services.NewSparkyQueueService(db, nil, nil, nil, nil, nil)
	h := NewSparkyHandler(q)

	mock.ExpectQuery(`FROM sparky_jobs WHERE id = \$1 AND user_id = \$2`).
		WithArgs("missing", "user-1").
		WillReturnRows(sqlmock.NewRows([]string{"chat_id", "status", "provider_used", "last_error", "result", "attempts"}))

	router.GET("/sparky/jobs/:jobId", func(c *gin.Context) {
		c.Set("userID", "user-1")
		h.GetSparkyJob(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/sparky/jobs/missing", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

var _ = time.Now
