package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

func newCallHandlerForQA(t *testing.T) (*CallHandler, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	svc := services.NewCallService(db, nil, nil)
	h := NewCallHandler(svc)
	return h, mock, func() { db.Close() }
}

func serveCall(t *testing.T, h func(*gin.Context), method, pattern, path, userID string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	router := setupTestRouter()
	router.Handle(method, pattern, func(c *gin.Context) {
		if userID != "" {
			c.Set("userID", userID)
		}
		h(c)
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, bytes.NewReader(body))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	router.ServeHTTP(w, req)
	return w
}

func TestCallQA_Initiate_MissingChatID(t *testing.T) {
	h, _, cleanup := newCallHandlerForQA(t)
	defer cleanup()
	body, _ := json.Marshal(map[string]string{"type": "audio"})
	w := serveCall(t, h.InitiateCall, "POST", "/calls/initiate", "/calls/initiate", "u1", body)
	if w.Code != 400 {
		t.Fatalf("expected 400 got %d: %s", w.Code, w.Body.String())
	}
}

func TestCallQA_Initiate_NotParticipant(t *testing.T) {
	h, mock, cleanup := newCallHandlerForQA(t)
	defer cleanup()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id FROM chat_participants WHERE chat_id=$1`)).
		WithArgs("chat-1").WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow("u2"))
	body, _ := json.Marshal(models.InitiateCallRequest{ChatID: "chat-1", Type: "audio"})
	w := serveCall(t, h.InitiateCall, "POST", "/calls/initiate", "/calls/initiate", "u1", body)
	if w.Code != 403 {
		t.Fatalf("expected 403 got %d: %s", w.Code, w.Body.String())
	}
}

func TestCallQA_Initiate_ChatNotFound(t *testing.T) {
	h, mock, cleanup := newCallHandlerForQA(t)
	defer cleanup()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id FROM chat_participants WHERE chat_id=$1`)).
		WithArgs("chat-1").WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
	body, _ := json.Marshal(models.InitiateCallRequest{ChatID: "chat-1", Type: "audio"})
	w := serveCall(t, h.InitiateCall, "POST", "/calls/initiate", "/calls/initiate", "u1", body)
	if w.Code != 404 {
		t.Fatalf("expected 404 got %d: %s", w.Code, w.Body.String())
	}
}

func TestCallQA_Initiate_Success(t *testing.T) {
	h, mock, cleanup := newCallHandlerForQA(t)
	defer cleanup()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id FROM chat_participants WHERE chat_id=$1`)).
		WithArgs("chat-1").WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow("u1").AddRow("u2"))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO call_sessions`)).
		WithArgs(sqlmock.AnyArg(), "chat-1", sqlmock.AnyArg(), "u1", "audio", "active", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO call_participants`)).
		WithArgs(sqlmock.AnyArg(), "u1").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO call_participants`)).
		WithArgs(sqlmock.AnyArg(), "u2").WillReturnResult(sqlmock.NewResult(1, 1))
	body, _ := json.Marshal(models.InitiateCallRequest{ChatID: "chat-1", Type: "audio"})
	w := serveCall(t, h.InitiateCall, "POST", "/calls/initiate", "/calls/initiate", "u1", body)
	if w.Code != 201 {
		t.Fatalf("expected 201 got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["session"] == nil || resp["offer"] == nil {
		t.Fatalf("expected session+offer %v", resp)
	}
}

func TestCallQA_End_NotFound(t *testing.T) {
	h, mock, cleanup := newCallHandlerForQA(t)
	defer cleanup()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("missing").WillReturnError(sql.ErrNoRows)
	// fallback query also fails -> still not found
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("missing").WillReturnError(sql.ErrNoRows)
	w := serveCall(t, h.EndCall, "POST", "/calls/:callId/end", "/calls/missing/end", "u1", nil)
	if w.Code != 404 {
		t.Fatalf("expected 404 got %d: %s", w.Code, w.Body.String())
	}
}

func TestCallQA_End_NotParticipant(t *testing.T) {
	h, mock, cleanup := newCallHandlerForQA(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1","u2"]`, "audio", "active", now, nil))
	w := serveCall(t, h.EndCall, "POST", "/calls/:callId/end", "/calls/call-1/end", "u9", nil)
	if w.Code != 403 {
		t.Fatalf("expected 403 got %d: %s", w.Code, w.Body.String())
	}
}

func TestCallQA_End_AlreadyEnded(t *testing.T) {
	h, mock, cleanup := newCallHandlerForQA(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1"]`, "audio", "ended", now, now))
	w := serveCall(t, h.EndCall, "POST", "/calls/:callId/end", "/calls/call-1/end", "u1", nil)
	if w.Code != 400 {
		t.Fatalf("expected 400 got %d: %s", w.Code, w.Body.String())
	}
}

func TestCallQA_GetSession_NotFound(t *testing.T) {
	h, mock, cleanup := newCallHandlerForQA(t)
	defer cleanup()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("missing").WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("missing").WillReturnError(sql.ErrNoRows)
	w := serveCall(t, h.GetCallSession, "GET", "/calls/:callId", "/calls/missing", "u1", nil)
	if w.Code != 404 {
		t.Fatalf("expected 404 got %d: %s", w.Code, w.Body.String())
	}
}

func TestCallQA_GetSession_NotParticipant(t *testing.T) {
	h, mock, cleanup := newCallHandlerForQA(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1"]`, "audio", "active", now, nil))
	w := serveCall(t, h.GetCallSession, "GET", "/calls/:callId", "/calls/call-1", "u9", nil)
	if w.Code != 403 {
		t.Fatalf("expected 403 got %d: %s", w.Code, w.Body.String())
	}
}

func TestCallQA_GetSession_Success(t *testing.T) {
	h, mock, cleanup := newCallHandlerForQA(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1","u2"]`, "audio", "active", now, nil))
	w := serveCall(t, h.GetCallSession, "GET", "/calls/:callId", "/calls/call-1", "u1", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
}

func TestCallQA_PostCaption_MissingText(t *testing.T) {
	h, _, cleanup := newCallHandlerForQA(t)
	defer cleanup()
	body, _ := json.Marshal(map[string]string{"language": "en"})
	w := serveCall(t, h.PostCaption, "POST", "/calls/:callId/captions", "/calls/call-1/captions", "u1", body)
	if w.Code != 400 {
		t.Fatalf("expected 400 got %d: %s", w.Code, w.Body.String())
	}
}

func TestCallQA_PostCaption_NotParticipant(t *testing.T) {
	h, mock, cleanup := newCallHandlerForQA(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1"]`, "audio", "active", now, nil))
	body, _ := json.Marshal(map[string]string{"text": "hello", "language": "en"})
	w := serveCall(t, h.PostCaption, "POST", "/calls/:callId/captions", "/calls/call-1/captions", "u9", body)
	if w.Code != 403 {
		t.Fatalf("expected 403 got %d: %s", w.Code, w.Body.String())
	}
}

func TestCallQA_PostCaption_AlreadyEnded(t *testing.T) {
	h, mock, cleanup := newCallHandlerForQA(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1"]`, "audio", "ended", now, now))
	body, _ := json.Marshal(map[string]string{"text": "hello"})
	w := serveCall(t, h.PostCaption, "POST", "/calls/:callId/captions", "/calls/call-1/captions", "u1", body)
	if w.Code != 400 {
		t.Fatalf("expected 400 got %d: %s", w.Code, w.Body.String())
	}
}

func TestCallQA_PostCaption_Success(t *testing.T) {
	h, mock, cleanup := newCallHandlerForQA(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1","u2"]`, "audio", "active", now, nil))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT DISTINCT native_language FROM users WHERE id = ANY`)).
		WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"native_language"}).AddRow("en"))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM call_transcripts WHERE call_id=$1`)).
		WithArgs("call-1").WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO call_transcripts`)).
		WithArgs(sqlmock.AnyArg(), "call-1", sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	body, _ := json.Marshal(map[string]string{"text": "Hello captioned", "language": "en"})
	w := serveCall(t, h.PostCaption, "POST", "/calls/:callId/captions", "/calls/call-1/captions", "u1", body)
	if w.Code != 201 {
		t.Fatalf("expected 201 got %d: %s", w.Code, w.Body.String())
	}
}

func TestCallQA_GetCaptions_NotParticipant(t *testing.T) {
	h, mock, cleanup := newCallHandlerForQA(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1"]`, "audio", "active", now, nil))
	w := serveCall(t, h.GetCaptions, "GET", "/calls/:callId/captions", "/calls/call-1/captions", "u9", nil)
	if w.Code != 403 {
		t.Fatalf("expected 403 got %d: %s", w.Code, w.Body.String())
	}
}

func TestCallQA_GetCaptions_Success(t *testing.T) {
	h, mock, cleanup := newCallHandlerForQA(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1"]`, "audio", "active", now, nil))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, call_id, segments, created_at FROM call_transcripts WHERE call_id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "call_id", "segments", "created_at"}).
			AddRow("tr-1", "call-1", `[]`, now))
	w := serveCall(t, h.GetCaptions, "GET", "/calls/:callId/captions", "/calls/call-1/captions", "u1", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
}

func TestCallQA_Bookmark_OutOfRange(t *testing.T) {
	h, mock, cleanup := newCallHandlerForQA(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1"]`, "audio", "active", now, nil))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, call_id, segments, created_at FROM call_transcripts WHERE call_id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "call_id", "segments", "created_at"}).
			AddRow("tr-1", "call-1", `[{"speakerId":"u1","startTime":1,"endTime":2,"originalText":"hello","originalLanguage":"en","translations":{},"confidence":1}]`, now))
	w := serveCall(t, h.BookmarkCaption, "POST", "/calls/:callId/captions/:index/bookmark", "/calls/call-1/captions/5/bookmark", "u1", []byte(`{}`))
	if w.Code != 400 {
		t.Fatalf("expected 400 got %d: %s", w.Code, w.Body.String())
	}
}

func TestCallQA_Signal_InvalidType(t *testing.T) {
	h, _, cleanup := newCallHandlerForQA(t)
	defer cleanup()
	body, _ := json.Marshal(map[string]string{"type": "bad-type"})
	w := serveCall(t, h.HandleWebRTCSignaling, "POST", "/calls/:callId/signal", "/calls/call-1/signal", "u1", body)
	if w.Code != 400 {
		t.Fatalf("expected 400 got %d: %s", w.Code, w.Body.String())
	}
}

func TestCallQA_Signal_MissingSDP(t *testing.T) {
	h, mock, cleanup := newCallHandlerForQA(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1","u2"]`, "audio", "active", now, nil))
	body, _ := json.Marshal(map[string]string{"type": "offer"})
	w := serveCall(t, h.HandleWebRTCSignaling, "POST", "/calls/:callId/signal", "/calls/call-1/signal", "u1", body)
	if w.Code != 400 {
		t.Fatalf("expected 400 got %d: %s", w.Code, w.Body.String())
	}
}

func TestCallQA_Signal_Success(t *testing.T) {
	h, mock, cleanup := newCallHandlerForQA(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1","u2"]`, "audio", "active", now, nil))
	body, _ := json.Marshal(map[string]string{"type": "offer", "sdp": "v=0\r\n..."})
	w := serveCall(t, h.HandleWebRTCSignaling, "POST", "/calls/:callId/signal", "/calls/call-1/signal", "u1", body)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
}

func TestCallQA_Transcribe_RequiresAudio(t *testing.T) {
	h, _, cleanup := newCallHandlerForQA(t)
	defer cleanup()
	body, _ := json.Marshal(map[string]string{"language": "en"})
	w := serveCall(t, h.TranscribeCaption, "POST", "/calls/:callId/transcribe", "/calls/call-1/transcribe", "u1", body)
	if w.Code != 400 {
		t.Fatalf("expected 400 got %d: %s", w.Code, w.Body.String())
	}
}

func TestCallQA_Transcribe_InvalidBase64(t *testing.T) {
	h, _, cleanup := newCallHandlerForQA(t)
	defer cleanup()
	body, _ := json.Marshal(map[string]string{"audio": "!!! not base64 !!!"})
	w := serveCall(t, h.TranscribeCaption, "POST", "/calls/:callId/transcribe", "/calls/call-1/transcribe", "u1", body)
	if w.Code != 400 {
		t.Fatalf("expected 400 got %d: %s", w.Code, w.Body.String())
	}
}

func TestCallQA_SearchTranscripts_RequiresQuery(t *testing.T) {
	h, _, cleanup := newCallHandlerForQA(t)
	defer cleanup()
	w := serveCall(t, h.SearchTranscripts, "GET", "/calls/transcripts/search", "/calls/transcripts/search", "u1", nil)
	if w.Code != 400 {
		t.Fatalf("expected 400 got %d: %s", w.Code, w.Body.String())
	}
}
