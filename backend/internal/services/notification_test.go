package services

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

type mockSender struct {
	err error
}

func (m *mockSender) Send(to, subject, html string) error { return m.err }

func TestNotificationSendSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	svc := NewNotificationService(db, &mockSender{err: nil})

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO email_outbox (recipient, subject, body) VALUES ($1, $2, $3) RETURNING id`)).
		WithArgs("learner@example.com", "Welcome", "<p>hi</p>").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("email-1"))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE email_outbox SET status = 'sent', sent_at = CURRENT_TIMESTAMP, last_error = NULL WHERE id = $1`)).
		WithArgs("email-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.Send("Learner@Example.com", "Welcome", "<p>hi</p>"); err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestNotificationSendFailureSchedulesRetry(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	sendErr := errors.New("connection refused")
	svc := NewNotificationService(db, &mockSender{err: sendErr})

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO email_outbox (recipient, subject, body) VALUES ($1, $2, $3) RETURNING id`)).
		WithArgs("learner@example.com", "Welcome", "<p>hi</p>").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("email-1"))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT attempts FROM email_outbox WHERE id = $1`)).
		WithArgs("email-1").
		WillReturnRows(sqlmock.NewRows([]string{"attempts"}).AddRow(0))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE email_outbox SET attempts = $1, status = $2, last_error = $3, next_attempt_at = CURRENT_TIMESTAMP + ($4 * INTERVAL '1 second') WHERE id = $5`)).
		WithArgs(1, "pending", sendErr.Error(), 60, "email-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = svc.Send("learner@example.com", "Welcome", "<p>hi</p>")
	if err == nil {
		t.Fatal("expected Send to return the send error")
	}
	if !errors.Is(err, sendErr) {
		t.Fatalf("expected original send error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestNotificationRetryByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	svc := NewNotificationService(db, &mockSender{err: nil})

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT recipient, subject, body, status FROM email_outbox WHERE id = $1`)).
		WithArgs("email-1").
		WillReturnRows(sqlmock.NewRows([]string{"recipient", "subject", "body", "status"}).
			AddRow("learner@example.com", "Welcome", "<p>hi</p>", "pending"))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE email_outbox SET status = 'sent', sent_at = CURRENT_TIMESTAMP, last_error = NULL WHERE id = $1`)).
		WithArgs("email-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.RetryByID("email-1"); err != nil {
		t.Fatalf("RetryByID failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestNotificationRetryAlreadySent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	svc := NewNotificationService(db, &mockSender{err: nil})

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT recipient, subject, body, status FROM email_outbox WHERE id = $1`)).
		WithArgs("email-1").
		WillReturnRows(sqlmock.NewRows([]string{"recipient", "subject", "body", "status"}).
			AddRow("learner@example.com", "Welcome", "<p>hi</p>", "sent"))

	err = svc.RetryByID("email-1")
	if err == nil {
		t.Fatal("expected error for already-sent email")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestNotificationList(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	svc := NewNotificationService(db, &mockSender{})

	now := time.Now()
	mock.ExpectQuery(`SELECT id, recipient, subject, status, attempts, COALESCE\(last_error, ''\), created_at, next_attempt_at, sent_at FROM email_outbox WHERE status = \$1 ORDER BY created_at DESC LIMIT 100`).
		WithArgs("pending").
		WillReturnRows(sqlmock.NewRows([]string{"id", "recipient", "subject", "status", "attempts", "last_error", "created_at", "next_attempt_at", "sent_at"}).
			AddRow("email-1", "learner@example.com", "Welcome", "pending", 2, "boom", now, now.Add(time.Minute), nil))

	entries, err := svc.List("pending")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Status != "pending" || entries[0].LastError != "boom" || entries[0].Attempts != 2 {
		t.Fatalf("unexpected entry: %+v", entries[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
