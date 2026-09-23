package services

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

var errSparky = errors.New("sparky down")

type fakeSparkyAnswerer struct {
	content  string
	provider string
	err      error
}

func (f *fakeSparkyAnswerer) GenerateSparkyAnswer(contextText, language, nativeLanguage, query string) (string, string, error) {
	return f.content, f.provider, f.err
}

func TestSparkyQueue_EnqueueInsertsAndPublishes(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	q := NewSparkyQueueService(db, deadRedis(), nil, nil, nil, nil)

	mock.ExpectQuery(`INSERT INTO sparky_jobs \(user_id, chat_id, context, query, language, native_language\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6\) RETURNING id`).
		WithArgs("user-1", "chat-1", "ctx", "what?", "es", "en").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("job-1"))

	id, err := q.EnqueueForAsk("user-1", "chat-1", "ctx", "es", "en", "what?")
	if err != nil {
		t.Fatalf("EnqueueForAsk failed: %v", err)
	}
	if id != "job-1" {
		t.Fatalf("expected job-1, got %q", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSparkyQueue_EnqueueDefaultsLanguage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	q := NewSparkyQueueService(db, deadRedis(), nil, nil, nil, nil)

	mock.ExpectQuery(`INSERT INTO sparky_jobs .* VALUES \(\$1, \$2, \$3, \$4, \$5, \$6\) RETURNING id`).
		WithArgs("user-1", nil, "ctx", "hi", "en", "en").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("job-2"))

	if _, err := q.EnqueueForAsk("user-1", "", "ctx", "  ", "", "hi"); err != nil {
		t.Fatalf("EnqueueForAsk failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSparkyQueue_GetJob(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	q := NewSparkyQueueService(db, nil, nil, nil, nil, nil)

	mock.ExpectQuery(`FROM sparky_jobs WHERE id = \$1 AND user_id = \$2`).
		WithArgs("job-1", "user-1").
		WillReturnRows(sqlmock.NewRows([]string{"chat_id", "status", "provider_used", "last_error", "result", "attempts"}).
			AddRow("chat-1", "done", "ai", "", "answer text", 1))

	job, err := q.GetJob("job-1", "user-1")
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}
	if job.Status != "done" || job.Content != "answer text" || job.ChatID != "chat-1" {
		t.Fatalf("unexpected job: %+v", job)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSparkyQueue_GetJob_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	q := NewSparkyQueueService(db, nil, nil, nil, nil, nil)

	mock.ExpectQuery(`FROM sparky_jobs WHERE id = \$1 AND user_id = \$2`).
		WithArgs("missing", "user-1").
		WillReturnRows(sqlmock.NewRows([]string{"chat_id", "status", "provider_used", "last_error", "result", "attempts"}))

	if _, err := q.GetJob("missing", "user-1"); err == nil {
		t.Fatalf("expected not-found error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
