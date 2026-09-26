package services

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
	"github.com/redis/go-redis/v9"
)

// deadRedis returns a client that cannot connect, forcing publish to fail fast
// (errors are logged and swallowed) so tests avoid spawning worker goroutines
// that would race sqlmock.
func deadRedis() *redis.Client {
	return redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
}

var errProvider = errors.New("provider down")

func TestTranslationQueue_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	q := NewTranslationQueueService(db, nil, nil, nil, nil)

	now := time.Now()
	mock.ExpectQuery(`SELECT id, message_id, chat_id, text, COALESCE\(source_lang,''\), target_lang, priority, status, COALESCE\(result,''\), attempts, COALESCE\(last_error,''\), created_at, next_attempt_at, completed_at, COALESCE\(provider,''\), COALESCE\(model,''\), COALESCE\(prompt_version,''\), COALESCE\(latency_ms,0\), COALESCE\(tokens,0\), COALESCE\(cache_hit,false\) FROM translation_jobs WHERE 1=1 AND status = \$1 ORDER BY created_at DESC LIMIT \$2`).
		WithArgs("failed", 50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "message_id", "chat_id", "text", "source_lang", "target_lang", "priority", "status", "result", "attempts", "last_error", "created_at", "next_attempt_at", "completed_at", "provider", "model", "prompt_version", "latency_ms", "tokens", "cache_hit"}).
			AddRow("job-1", "msg-1", "chat-1", "Hola", "en", "es", 1, "failed", "", 6, "provider down", now, now.Add(time.Hour), nil, "", "", "v2", 0, 0, false))

	jobs, err := q.List("failed", "", 50)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].TargetLang != "es" || jobs[0].Status != "failed" || jobs[0].Attempts != 6 || jobs[0].Priority != 1 {
		t.Fatalf("unexpected job: %+v", jobs[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestTranslationQueue_RetryByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	q := NewTranslationQueueService(db, deadRedis(), nil, nil, nil)

	mock.ExpectQuery(`UPDATE translation_jobs\s+SET status = 'pending', processing_at = NULL, last_error = NULL, next_attempt_at = CURRENT_TIMESTAMP WHERE id = \$1 AND status <> 'done' RETURNING message_id`).
		WithArgs("job-1").
		WillReturnRows(sqlmock.NewRows([]string{"message_id"}).AddRow("msg-1"))

	if err := q.RetryByID("job-1"); err != nil {
		t.Fatalf("RetryByID failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestTranslationQueue_RetryByID_Missing(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	q := NewTranslationQueueService(db, nil, nil, nil, nil)

	mock.ExpectQuery(`UPDATE translation_jobs\s+SET status = 'pending', processing_at = NULL, last_error = NULL, next_attempt_at = CURRENT_TIMESTAMP WHERE id = \$1 AND status <> 'done' RETURNING message_id`).
		WithArgs("job-9").
		WillReturnRows(sqlmock.NewRows([]string{"message_id"})) // no rows

	if err := q.RetryByID("job-9"); err == nil {
		t.Fatal("expected error for missing job")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestTranslationQueue_EnqueueSkipsOwnLanguage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := &TranslationQueueService{db: db}
	msg := &models.Message{ID: "m-1", ChatID: "chat-1", Text: "hello", OriginalLanguage: "en"}

	// All targets equal to source or empty -> nothing is inserted or spawned.
	s.EnqueueForMessage(msg, "en", []string{"en", "", "  "}, 0)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestTranslationQueue_MarkFailedBackoff(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	q := &TranslationQueueService{db: db}

	// Attempt 2 of 6 -> delay doubles once: 60s.
	mock.ExpectExec(`UPDATE translation_jobs\s+SET status = \$1, last_error = \$2, processing_at = NULL,\s*next_attempt_at = CURRENT_TIMESTAMP \+ \(\$3 \* INTERVAL '1 second'\)\s*WHERE id = \$4`).
		WithArgs("failed", "provider down", 60, "job-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	q.markFailed("job-1", 2, errProvider)

	// Attempt 6 of 6 -> exhausted, next_attempt pushed 24h out for manual retry.
	mock.ExpectExec(`UPDATE translation_jobs\s+SET status = \$1, last_error = \$2, processing_at = NULL,\s*next_attempt_at = CURRENT_TIMESTAMP \+ \(\$3 \* INTERVAL '1 second'\)\s*WHERE id = \$4`).
		WithArgs("failed", "provider down", 86400, "job-2").
		WillReturnResult(sqlmock.NewResult(0, 1))
	q.markFailed("job-2", 6, errProvider)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestTranslationQueue_Stats(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	q := &TranslationQueueService{db: db}

	mock.ExpectQuery(`SELECT status, COUNT\(\*\) FROM translation_jobs GROUP BY status`).
		WillReturnRows(sqlmock.NewRows([]string{"status", "count"}).
			AddRow("pending", 3).
			AddRow("processing", 1).
			AddRow("done", 12).
			AddRow("failed", 2))

	pending, processing, done, failed := q.Stats()
	if pending != 3 || processing != 1 || done != 12 || failed != 2 {
		t.Fatalf("unexpected stats: %d %d %d %d", pending, processing, done, failed)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
