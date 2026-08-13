package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

func newModerationHandlerForTest(t *testing.T) (*ModerationHandler, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	return NewModerationHandler(services.NewModerationService(db)), mock, func() { db.Close() }
}

// serveModeration wires a handler method behind a gin route with the userID
// context value set (as the auth middleware would) and runs a request.
func serveModeration(t *testing.T, handler func(*gin.Context), method, routePattern, path, userID string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	router := setupTestRouter()
	router.Handle(method, routePattern, func(c *gin.Context) {
		c.Set("userID", userID)
		handler(c)
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, body)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	router.ServeHTTP(w, req)
	return w
}

func mustJSONBody(t *testing.T, v interface{}) io.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	return bytes.NewBuffer(b)
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder, out interface{}) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
		t.Fatalf("decode response failed: %v (body: %s)", err, w.Body.String())
	}
}

// --- Block ---

func TestBlockHandler_Success(t *testing.T) {
	h, mock, cleanup := newModerationHandlerForTest(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM users WHERE id = \$1 AND deleted_at IS NULL\)`).
		WithArgs("u2").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec(`INSERT INTO blocked_users \(blocker_id, blocked_id, reason\) VALUES \(\$1, \$2, \$3\)`).
		WithArgs("u1", "u2", "").WillReturnResult(sqlmock.NewResult(1, 1))

	w := serveModeration(t, h.Block, http.MethodPost, "/blocks", "/blocks", "u1",
		mustJSONBody(t, map[string]string{"blockedUserId": "u2"}))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestBlockHandler_SelfBlock(t *testing.T) {
	h, _, cleanup := newModerationHandlerForTest(t)
	defer cleanup()

	w := serveModeration(t, h.Block, http.MethodPost, "/blocks", "/blocks", "u1",
		mustJSONBody(t, map[string]string{"blockedUserId": "u1"}))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBlockHandler_UserNotFound(t *testing.T) {
	h, mock, cleanup := newModerationHandlerForTest(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM users WHERE id = \$1 AND deleted_at IS NULL\)`).
		WithArgs("ghost").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	w := serveModeration(t, h.Block, http.MethodPost, "/blocks", "/blocks", "u1",
		mustJSONBody(t, map[string]string{"blockedUserId": "ghost"}))

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBlockHandler_InvalidRequest(t *testing.T) {
	h, _, cleanup := newModerationHandlerForTest(t)
	defer cleanup()

	w := serveModeration(t, h.Block, http.MethodPost, "/blocks", "/blocks", "u1",
		bytes.NewBufferString(`{}`))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Unblock ---

func TestUnblockHandler_Success(t *testing.T) {
	h, mock, cleanup := newModerationHandlerForTest(t)
	defer cleanup()

	mock.ExpectExec(`DELETE FROM blocked_users WHERE blocker_id = \$1 AND blocked_id = \$2`).
		WithArgs("u1", "u2").WillReturnResult(sqlmock.NewResult(0, 1))

	w := serveModeration(t, h.Unblock, http.MethodDelete, "/blocks/:userId", "/blocks/u2", "u1", nil)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUnblockHandler_Error(t *testing.T) {
	h, mock, cleanup := newModerationHandlerForTest(t)
	defer cleanup()

	mock.ExpectExec(`DELETE FROM blocked_users WHERE blocker_id = \$1 AND blocked_id = \$2`).
		WithArgs("u1", "u2").WillReturnError(errTestDB)

	w := serveModeration(t, h.Unblock, http.MethodDelete, "/blocks/:userId", "/blocks/u2", "u1", nil)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

// --- ListBlocked ---

func TestListBlockedHandler_Success(t *testing.T) {
	h, mock, cleanup := newModerationHandlerForTest(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{
		"id", "blocker_id", "blocked_id", "reason", "created_at",
		"u.id", "u.username", "u.display_name", "u.email",
	}).AddRow("b1", "u1", "u2", "spam", time.Now(), "u2", "alice", "Alice", "alice@example.com")
	mock.ExpectQuery(`SELECT b.id, b.blocker_id, b.blocked_id, b.reason, b.created_at,`).
		WithArgs("u1").WillReturnRows(rows)

	w := serveModeration(t, h.ListBlocked, http.MethodGet, "/blocks", "/blocks", "u1", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Blocks []models.Block `json:"blocks"`
		Total  int            `json:"total"`
	}
	decodeBody(t, w, &resp)
	if len(resp.Blocks) != 1 || resp.Total != 1 || resp.Blocks[0].BlockedID != "u2" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestListBlockedHandler_Error(t *testing.T) {
	h, mock, cleanup := newModerationHandlerForTest(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT b.id, b.blocker_id, b.blocked_id, b.reason, b.created_at,`).
		WithArgs("u1").WillReturnError(errTestDB)

	w := serveModeration(t, h.ListBlocked, http.MethodGet, "/blocks", "/blocks", "u1", nil)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Report ---

func TestReportHandler_UserReport(t *testing.T) {
	h, mock, cleanup := newModerationHandlerForTest(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM users WHERE id = \$1 AND deleted_at IS NULL\)`).
		WithArgs("u2").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`INSERT INTO reports \(reporter_id, type, reported_user_id, message_id, chat_id, reason, status\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\) RETURNING id, created_at`).
		WithArgs("u1", "user", "u2", nil, nil, "harassment", services.ReportOpen).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow("r1", time.Now()))

	w := serveModeration(t, h.Report, http.MethodPost, "/reports", "/reports", "u1",
		mustJSONBody(t, models.ReportRequest{Type: "user", ReportedUserID: "u2", Reason: "harassment"}))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var report models.Report
	decodeBody(t, w, &report)
	if report.ID != "r1" || report.ReportedUserID != "u2" || report.Status != services.ReportOpen {
		t.Fatalf("unexpected report: %+v", report)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestReportHandler_MessageReport(t *testing.T) {
	h, mock, cleanup := newModerationHandlerForTest(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT sender_id, chat_id FROM messages WHERE id = \$1`).
		WithArgs("m1").WillReturnRows(sqlmock.NewRows([]string{"sender_id", "chat_id"}).AddRow("u2", "chat-1"))
	mock.ExpectQuery(`INSERT INTO reports \(reporter_id, type, reported_user_id, message_id, chat_id, reason, status\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\) RETURNING id, created_at`).
		WithArgs("u1", "message", "u2", "m1", "chat-1", "spam", services.ReportOpen).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow("r2", time.Now()))

	w := serveModeration(t, h.Report, http.MethodPost, "/reports", "/reports", "u1",
		mustJSONBody(t, models.ReportRequest{Type: "message", MessageID: "m1", Reason: "spam"}))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var report models.Report
	decodeBody(t, w, &report)
	if report.MessageID == nil || *report.MessageID != "m1" || report.ReportedUserID != "u2" {
		t.Fatalf("unexpected report: %+v", report)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestReportHandler_SelfReport(t *testing.T) {
	h, mock, cleanup := newModerationHandlerForTest(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM users WHERE id = \$1 AND deleted_at IS NULL\)`).
		WithArgs("u1").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	w := serveModeration(t, h.Report, http.MethodPost, "/reports", "/reports", "u1",
		mustJSONBody(t, models.ReportRequest{Type: "user", ReportedUserID: "u1", Reason: "oops"}))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReportHandler_MessageNotFound(t *testing.T) {
	h, mock, cleanup := newModerationHandlerForTest(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT sender_id, chat_id FROM messages WHERE id = \$1`).
		WithArgs("missing").WillReturnRows(sqlmock.NewRows([]string{"sender_id", "chat_id"}))

	w := serveModeration(t, h.Report, http.MethodPost, "/reports", "/reports", "u1",
		mustJSONBody(t, models.ReportRequest{Type: "message", MessageID: "missing", Reason: "spam"}))

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReportHandler_InvalidTarget(t *testing.T) {
	h, _, cleanup := newModerationHandlerForTest(t)
	defer cleanup()

	// No reported user, no message, empty reason.
	w := serveModeration(t, h.Report, http.MethodPost, "/reports", "/reports", "u1",
		mustJSONBody(t, models.ReportRequest{Type: "user"}))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReportHandler_InvalidJSON(t *testing.T) {
	h, _, cleanup := newModerationHandlerForTest(t)
	defer cleanup()

	w := serveModeration(t, h.Report, http.MethodPost, "/reports", "/reports", "u1",
		bytes.NewBufferString(`{not json`))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- ListReports ---

func TestListReportsHandler_Success(t *testing.T) {
	h, mock, cleanup := newModerationHandlerForTest(t)
	defer cleanup()

	// Count query (status filter applied).
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM reports r INNER JOIN users repu ON repu.id = r.reported_user_id WHERE 1=1 AND r.status = \$1`).
		WithArgs("open").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// List query.
	rows := sqlmock.NewRows([]string{
		"id", "reporter_id", "type", "reported_user_id", "message_id", "chat_id",
		"reason", "status", "resolver_id", "resolution_note", "created_at", "resolved_at",
		"rep.id", "rep.username", "rep.display_name", "rep.email",
		"repu.id", "repu.username", "repu.display_name", "repu.email",
	}).AddRow("r1", "reporter-1", "user", "target-1", nil, nil,
		"spam", "open", nil, "", time.Now(), nil,
		"reporter-1", "reporter", "Reporter", "reporter@example.com",
		"target-1", "target", "Target", "target@example.com")
	mock.ExpectQuery(`SELECT r.id, r.reporter_id, r.type, r.reported_user_id, r.message_id, r.chat_id,`).
		WithArgs("open", 50, 0).WillReturnRows(rows)

	w := serveModeration(t, h.ListReports, http.MethodGet, "/reports", "/reports?status=open", "u1", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Reports []models.Report `json:"reports"`
		Total   int             `json:"total"`
	}
	decodeBody(t, w, &resp)
	if len(resp.Reports) != 1 || resp.Total != 1 || resp.Reports[0].Status != "open" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestListReportsHandler_Error(t *testing.T) {
	h, mock, cleanup := newModerationHandlerForTest(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM reports`).WillReturnError(errTestDB)

	w := serveModeration(t, h.ListReports, http.MethodGet, "/reports", "/reports", "u1", nil)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

// --- ReportStats ---

func TestReportStatsHandler_Success(t *testing.T) {
	h, mock, cleanup := newModerationHandlerForTest(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM reports WHERE status = 'open'`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM reports WHERE type = 'user' AND status = 'open'`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM reports WHERE type = 'message' AND status = 'open'`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM reports WHERE status = 'resolved' AND resolved_at >= CURRENT_DATE`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))

	w := serveModeration(t, h.ReportStats, http.MethodGet, "/reports/stats", "/reports/stats", "u1", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var stats models.ReportStats
	decodeBody(t, w, &stats)
	if stats.OpenReports != 3 || stats.UserReports != 2 || stats.MessageReports != 1 || stats.ResolvedToday != 4 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// --- Resolve / Dismiss ---

func TestResolveReportHandler_Success(t *testing.T) {
	h, mock, cleanup := newModerationHandlerForTest(t)
	defer cleanup()

	mock.ExpectExec(`UPDATE reports SET status = 'resolved', resolver_id = \$2, resolved_at = CURRENT_TIMESTAMP WHERE id = \$1 AND status = 'open'`).
		WithArgs("r1", "mod-1").WillReturnResult(sqlmock.NewResult(0, 1))

	w := serveModeration(t, h.ResolveReport, http.MethodPost, "/reports/:id/resolve", "/reports/r1/resolve", "mod-1", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestResolveReportHandler_NotFound(t *testing.T) {
	h, mock, cleanup := newModerationHandlerForTest(t)
	defer cleanup()

	mock.ExpectExec(`UPDATE reports SET status = 'resolved', resolver_id = \$2, resolved_at = CURRENT_TIMESTAMP WHERE id = \$1 AND status = 'open'`).
		WithArgs("r1", "mod-1").WillReturnResult(sqlmock.NewResult(0, 0))

	w := serveModeration(t, h.ResolveReport, http.MethodPost, "/reports/:id/resolve", "/reports/r1/resolve", "mod-1", nil)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDismissReportHandler_Success(t *testing.T) {
	h, mock, cleanup := newModerationHandlerForTest(t)
	defer cleanup()

	mock.ExpectExec(`UPDATE reports SET status = 'dismissed', resolver_id = \$2, resolution_note = \$3, resolved_at = CURRENT_TIMESTAMP WHERE id = \$1 AND status = 'open'`).
		WithArgs("r1", "mod-1", "Duplicate of r2").WillReturnResult(sqlmock.NewResult(0, 1))

	w := serveModeration(t, h.DismissReport, http.MethodPost, "/reports/:id/dismiss", "/reports/r1/dismiss", "mod-1",
		mustJSONBody(t, map[string]string{"note": "Duplicate of r2"}))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestDismissReportHandler_NoNote(t *testing.T) {
	h, mock, cleanup := newModerationHandlerForTest(t)
	defer cleanup()

	mock.ExpectExec(`UPDATE reports SET status = 'dismissed', resolver_id = \$2, resolution_note = \$3, resolved_at = CURRENT_TIMESTAMP WHERE id = \$1 AND status = 'open'`).
		WithArgs("r1", "mod-1", "").WillReturnResult(sqlmock.NewResult(0, 1))

	w := serveModeration(t, h.DismissReport, http.MethodPost, "/reports/:id/dismiss", "/reports/r1/dismiss", "mod-1", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestDismissReportHandler_NotFound(t *testing.T) {
	h, mock, cleanup := newModerationHandlerForTest(t)
	defer cleanup()

	mock.ExpectExec(`UPDATE reports SET status = 'dismissed', resolver_id = \$2, resolution_note = \$3, resolved_at = CURRENT_TIMESTAMP WHERE id = \$1 AND status = 'open'`).
		WithArgs("r1", "mod-1", "").WillReturnResult(sqlmock.NewResult(0, 0))

	w := serveModeration(t, h.DismissReport, http.MethodPost, "/reports/:id/dismiss", "/reports/r1/dismiss", "mod-1", nil)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
