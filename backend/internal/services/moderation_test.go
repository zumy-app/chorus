package services

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
)

func newModerationTestService(t *testing.T) (*ModerationService, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	return NewModerationService(db), mock, func() { db.Close() }
}

func TestBlock_Self(t *testing.T) {
	s, _, cleanup := newModerationTestService(t)
	defer cleanup()
	if err := s.Block(context.Background(), "u1", "u1", ""); err != ErrBlockSelf {
		t.Fatalf("expected ErrBlockSelf, got %v", err)
	}
}

func TestBlock_MissingUser(t *testing.T) {
	s, mock, cleanup := newModerationTestService(t)
	defer cleanup()
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM users WHERE id = \$1 AND deleted_at IS NULL\)`).
		WithArgs("ghost").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	if err := s.Block(context.Background(), "u1", "ghost", ""); err != ErrBlockUserNotFound {
		t.Fatalf("expected ErrBlockUserNotFound, got %v", err)
	}
}

func TestBlock_Success(t *testing.T) {
	s, mock, cleanup := newModerationTestService(t)
	defer cleanup()
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM users WHERE id = \$1 AND deleted_at IS NULL\)`).
		WithArgs("u2").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec(`INSERT INTO blocked_users \(blocker_id, blocked_id, reason\) VALUES \(\$1, \$2, \$3\)`).
		WithArgs("u1", "u2", "spam").
		WillReturnResult(sqlmock.NewResult(1, 1))
	if err := s.Block(context.Background(), "u1", "u2", "spam"); err != nil {
		t.Fatalf("Block failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUnblock(t *testing.T) {
	s, mock, cleanup := newModerationTestService(t)
	defer cleanup()
	mock.ExpectExec(`DELETE FROM blocked_users WHERE blocker_id = \$1 AND blocked_id = \$2`).
		WithArgs("u1", "u2").WillReturnResult(sqlmock.NewResult(0, 1))
	if err := s.Unblock(context.Background(), "u1", "u2"); err != nil {
		t.Fatalf("Unblock failed: %v", err)
	}
}

func TestGetBlocked(t *testing.T) {
	s, mock, cleanup := newModerationTestService(t)
	defer cleanup()
	rows := sqlmock.NewRows([]string{
		"id", "blocker_id", "blocked_id", "reason", "created_at",
		"u.id", "u.username", "u.display_name", "u.email",
	}).AddRow("b1", "u1", "u2", "spam", time.Now(), "u2", "alice", "Alice", "alice@example.com")
	mock.ExpectQuery(`SELECT b.id, b.blocker_id, b.blocked_id, b.reason, b.created_at,`).
		WithArgs("u1").WillReturnRows(rows)
	blocks, err := s.GetBlocked(context.Background(), "u1")
	if err != nil {
		t.Fatalf("GetBlocked failed: %v", err)
	}
	if len(blocks) != 1 || blocks[0].BlockedID != "u2" || blocks[0].Blocked.Username != "alice" {
		t.Fatalf("unexpected blocks: %+v", blocks)
	}
}

func TestIsBlocked_Mutual(t *testing.T) {
	s, mock, cleanup := newModerationTestService(t)
	defer cleanup()
	mock.ExpectQuery(`SELECT EXISTS\(`).
		WithArgs("u1", "u2").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	blocked, err := s.IsBlocked(context.Background(), "u1", "u2")
	if err != nil || !blocked {
		t.Fatalf("expected blocked=true, err=%v", err)
	}
}

func TestChatBlocked(t *testing.T) {
	s, mock, cleanup := newModerationTestService(t)
	defer cleanup()
	mock.ExpectQuery(`SELECT EXISTS\(`).
		WithArgs("chat-1", "u1").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	blocked, err := s.ChatBlocked(context.Background(), "chat-1", "u1")
	if err != nil || !blocked {
		t.Fatalf("expected blocked=true, err=%v", err)
	}
}

func TestReport_User(t *testing.T) {
	s, mock, cleanup := newModerationTestService(t)
	defer cleanup()
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM users WHERE id = \$1 AND deleted_at IS NULL\)`).
		WithArgs("u2").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`INSERT INTO reports \(reporter_id, type, reported_user_id, message_id, chat_id, reason, status\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\) RETURNING id, created_at`).
		WithArgs("u1", "user", "u2", nil, nil, "harassment", ReportOpen).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow("r1", time.Now()))
	report, err := s.Report(context.Background(), "u1", models.ReportRequest{
		Type: "user", ReportedUserID: "u2", Reason: "harassment",
	})
	if err != nil {
		t.Fatalf("Report failed: %v", err)
	}
	if report.ReportedUserID != "u2" || report.Status != ReportOpen {
		t.Fatalf("unexpected report: %+v", report)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestReport_MessageResolvesSender(t *testing.T) {
	s, mock, cleanup := newModerationTestService(t)
	defer cleanup()
	mock.ExpectQuery(`SELECT sender_id, chat_id FROM messages WHERE id = \$1`).
		WithArgs("m1").WillReturnRows(sqlmock.NewRows([]string{"sender_id", "chat_id"}).AddRow("u2", "chat-1"))
	mock.ExpectQuery(`INSERT INTO reports \(reporter_id, type, reported_user_id, message_id, chat_id, reason, status\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\) RETURNING id, created_at`).
		WithArgs("u1", "message", "u2", "m1", "chat-1", "spam", ReportOpen).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow("r2", time.Now()))
	report, err := s.Report(context.Background(), "u1", models.ReportRequest{
		Type: "message", MessageID: "m1", Reason: "spam",
	})
	if err != nil {
		t.Fatalf("Report failed: %v", err)
	}
	if report.ReportedUserID != "u2" || report.MessageID == nil || *report.MessageID != "m1" {
		t.Fatalf("unexpected report: %+v", report)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestReport_Self(t *testing.T) {
	s, mock, cleanup := newModerationTestService(t)
	defer cleanup()
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM users WHERE id = \$1 AND deleted_at IS NULL\)`).
		WithArgs("u1").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	if _, err := s.Report(context.Background(), "u1", models.ReportRequest{
		Type: "user", ReportedUserID: "u1", Reason: "oops",
	}); err != ErrReportSelf {
		t.Fatalf("expected ErrReportSelf, got %v", err)
	}
}

func TestReport_MessageNotFound(t *testing.T) {
	s, mock, cleanup := newModerationTestService(t)
	defer cleanup()
	mock.ExpectQuery(`SELECT sender_id, chat_id FROM messages WHERE id = \$1`).
		WithArgs("missing").WillReturnRows(sqlmock.NewRows([]string{"sender_id", "chat_id"}))
	if _, err := s.Report(context.Background(), "u1", models.ReportRequest{
		Type: "message", MessageID: "missing", Reason: "spam",
	}); err != ErrReportMessageNotFound {
		t.Fatalf("expected ErrReportMessageNotFound, got %v", err)
	}
}

func TestResolveReport(t *testing.T) {
	s, mock, cleanup := newModerationTestService(t)
	defer cleanup()
	mock.ExpectExec(`UPDATE reports SET status = 'resolved', resolver_id = \$2, resolved_at = CURRENT_TIMESTAMP WHERE id = \$1 AND status = 'open'`).
		WithArgs("r1", "mod-1").WillReturnResult(sqlmock.NewResult(0, 1))
	if err := s.ResolveReport(context.Background(), "mod-1", "r1"); err != nil {
		t.Fatalf("ResolveReport failed: %v", err)
	}
}

func TestResolveReport_NotFound(t *testing.T) {
	s, mock, cleanup := newModerationTestService(t)
	defer cleanup()
	mock.ExpectExec(`UPDATE reports SET status = 'resolved', resolver_id = \$2, resolved_at = CURRENT_TIMESTAMP WHERE id = \$1 AND status = 'open'`).
		WithArgs("r1", "mod-1").WillReturnResult(sqlmock.NewResult(0, 0))
	if err := s.ResolveReport(context.Background(), "mod-1", "r1"); err != ErrReportNotFound {
		t.Fatalf("expected ErrReportNotFound, got %v", err)
	}
}

func TestReportStats(t *testing.T) {
	s, mock, cleanup := newModerationTestService(t)
	defer cleanup()
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM reports WHERE status = 'open'`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM reports WHERE type = 'user' AND status = 'open'`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM reports WHERE type = 'message' AND status = 'open'`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM reports WHERE status = 'resolved' AND resolved_at >= CURRENT_DATE`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))
	stats, err := s.GetReportStats(context.Background())
	if err != nil {
		t.Fatalf("GetReportStats failed: %v", err)
	}
	if stats.OpenReports != 3 || stats.UserReports != 2 || stats.MessageReports != 1 || stats.ResolvedToday != 4 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}
