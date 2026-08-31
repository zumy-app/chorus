package services

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
)

var errAnalyzer = errors.New("analyzer down")

type fakeGrammarAnalyzer struct {
	analysis *models.AIGrammarAnalysis
	provider string
	err      error
}

func (f *fakeGrammarAnalyzer) GenerateAIAnalysis(text, language, nativeLanguage string) (*models.AIGrammarAnalysis, string, error) {
	return f.analysis, f.provider, f.err
}

func TestGrammarQueue_EnqueueInsertsAndPublishes(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	q := NewGrammarQueueService(db, deadRedis(), nil, nil, nil)

	mock.ExpectQuery(`INSERT INTO grammar_jobs \(user_id, chat_id, message_id, text, language, native_language\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6\) RETURNING id`).
		WithArgs("user-1", nil, "msg-1", "Hola", "es", "en").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("job-1"))

	id, err := q.EnqueueForAnalysis("user-1", "", "msg-1", "Hola", "es", "en")
	if err != nil {
		t.Fatalf("EnqueueForAnalysis failed: %v", err)
	}
	if id != "job-1" {
		t.Fatalf("expected job-1, got %q", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGrammarQueue_EnqueueDefaultsLanguage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	q := NewGrammarQueueService(db, deadRedis(), nil, nil, nil)

	mock.ExpectQuery(`INSERT INTO grammar_jobs .* VALUES \(\$1, \$2, \$3, \$4, \$5, \$6\) RETURNING id`).
		WithArgs("user-1", nil, nil, "hi", "en", "en").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("job-2"))

	if _, err := q.EnqueueForAnalysis("user-1", "", "", "hi", "  ", ""); err != nil {
		t.Fatalf("EnqueueForAnalysis failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGrammarQueue_GetJob(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	q := NewGrammarQueueService(db, nil, nil, nil, nil)

	mock.ExpectQuery(`SELECT COALESCE\(message_id::text,''\), COALESCE\(chat_id::text,''\), status, COALESCE\(provider_used,''\), COALESCE\(result::text,''\), COALESCE\(last_error,''\), attempts FROM grammar_jobs WHERE id = \$1 AND user_id = \$2`).
		WithArgs("job-1", "user-1").
		WillReturnRows(sqlmock.NewRows([]string{"message_id", "chat_id", "status", "provider_used", "result", "last_error", "attempts"}).
			AddRow("msg-1", "chat-1", "done", "openrouter", `{"difficulty":"B1","summary":"A friendly sentence."}`, "", 1))

	job, err := q.GetJob("job-1", "user-1")
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}
	if job.Status != "done" || job.ProviderUsed != "openrouter" || job.Analysis == nil || job.Analysis.Difficulty != "B1" {
		t.Fatalf("unexpected job: %+v", job)
	}
	if job.Analysis.Summary != "A friendly sentence." {
		t.Fatalf("unexpected summary: %q", job.Analysis.Summary)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGrammarQueue_GetJob_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	q := NewGrammarQueueService(db, nil, nil, nil, nil)

	mock.ExpectQuery(`FROM grammar_jobs WHERE id = \$1 AND user_id = \$2`).
		WithArgs("job-x", "user-1").
		WillReturnRows(sqlmock.NewRows([]string{"message_id", "chat_id", "status", "provider_used", "result", "last_error", "attempts"}))

	if _, err := q.GetJob("job-x", "user-1"); err == nil {
		t.Fatal("expected error for missing job")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGrammarQueue_RetryByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	q := NewGrammarQueueService(db, deadRedis(), nil, nil, nil)

	mock.ExpectQuery(`UPDATE grammar_jobs\s+SET status = 'pending', processing_at = NULL, last_error = NULL, next_attempt_at = CURRENT_TIMESTAMP WHERE id = \$1 AND status <> 'done' RETURNING id`).
		WithArgs("job-1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("job-1"))

	if err := q.retryJob("job-1"); err != nil {
		t.Fatalf("retryJob failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGrammarQueue_RetryByID_Missing(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	q := NewGrammarQueueService(db, nil, nil, nil, nil)

	mock.ExpectQuery(`RETURNING id`).
		WithArgs("job-x").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	if err := q.retryJob("job-x"); err == nil {
		t.Fatal("expected error for missing job")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGrammarQueue_ProcessJob_Done(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	var notified []*GrammarJobResult
	analyzer := &fakeGrammarAnalyzer{
		analysis: &models.AIGrammarAnalysis{Difficulty: "B2", Summary: "Nice sentence."},
		provider: "ollama",
	}
	q := NewGrammarQueueService(db, nil, analyzer, nil, func(userID string, p *GrammarJobResult) {
		notified = append(notified, p)
	})

	claimCols := []string{"id", "user_id", "message_id", "chat_id", "text", "language", "native_language", "attempts"}
	mock.ExpectQuery(`UPDATE grammar_jobs\s+SET status = 'processing'.*RETURNING id, user_id, COALESCE\(message_id::text,''\)`).
		WillReturnRows(sqlmock.NewRows(claimCols).AddRow("job-1", "user-1", "msg-1", "chat-1", "Hola", "es", "en", 1))
	mock.ExpectExec(`UPDATE grammar_jobs\s+SET status = 'done', result = \$1, provider_used = \$2, last_error = NULL, processing_at = NULL, completed_at = CURRENT_TIMESTAMP, provider_used_lineage = \$2, prompt_version = \$3, latency_ms = \$4 WHERE id = \$5`).
		WithArgs(sqlmock.AnyArg(), "ollama", "v2", sqlmock.AnyArg(), "job-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	q.processJob("job-1")

	if len(notified) != 2 {
		t.Fatalf("expected 2 notifications (processing + done), got %d", len(notified))
	}
	if notified[0].Status != "processing" || notified[1].Status != "done" {
		t.Fatalf("unexpected notification order: %+v, %+v", notified[0], notified[1])
	}
	if notified[1].ProviderUsed != "ollama" || notified[1].Analysis == nil {
		t.Fatalf("unexpected done payload: %+v", notified[1])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGrammarQueue_ProcessJob_Failed(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	var notified []*GrammarJobResult
	analyzer := &fakeGrammarAnalyzer{err: errAnalyzer}
	q := NewGrammarQueueService(db, nil, analyzer, nil, func(userID string, p *GrammarJobResult) {
		notified = append(notified, p)
	})

	claimCols := []string{"id", "user_id", "message_id", "chat_id", "text", "language", "native_language", "attempts"}
	mock.ExpectQuery(`UPDATE grammar_jobs\s+SET status = 'processing'.*RETURNING id, user_id, COALESCE\(message_id::text,''\)`).
		WillReturnRows(sqlmock.NewRows(claimCols).AddRow("job-1", "user-1", "msg-1", "chat-1", "Hola", "es", "en", 1))
	mock.ExpectExec(`UPDATE grammar_jobs\s+SET status = \$1, last_error = \$2, processing_at = NULL, next_attempt_at = CURRENT_TIMESTAMP \+ \(\$3 \* INTERVAL '1 second'\) WHERE id = \$4`).
		WithArgs("failed", "analyzer down", 20, "job-1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT user_id FROM grammar_jobs WHERE id = \$1`).
		WithArgs("job-1").
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow("user-1"))

	q.processJob("job-1")

	if len(notified) != 2 {
		t.Fatalf("expected 2 notifications (processing + failed), got %d", len(notified))
	}
	if notified[1].Status != "failed" || notified[1].Error != "analyzer down" {
		t.Fatalf("unexpected failed payload: %+v", notified[1])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGrammarQueue_MarkFailedBackoff(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	q := &GrammarQueueService{db: db}

	// Attempt 2 of 5 -> delay doubles once: 40s.
	mock.ExpectExec(`UPDATE grammar_jobs\s+SET status = \$1, last_error = \$2, processing_at = NULL, next_attempt_at = CURRENT_TIMESTAMP \+ \(\$3 \* INTERVAL '1 second'\) WHERE id = \$4`).
		WithArgs("failed", "boom", 40, "job-1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT user_id FROM grammar_jobs WHERE id = \$1`).
		WithArgs("job-1").
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow("user-1"))

	q.markFailed("job-1", 2, errors.New("boom"))
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
