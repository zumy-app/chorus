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

func newVideoHandlerForQA(t *testing.T) (*CallHandler, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	svc := services.NewCallService(db, nil, nil)
	h := NewCallHandler(svc)
	return h, mock, func() { db.Close() }
}

func serveVideo(t *testing.T, h func(*gin.Context), method, pattern, path, userID string, body []byte) *httptest.ResponseRecorder {
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

func TestVideoQA_Initiate_Video_Success(t *testing.T) {
	h, mock, cleanup := newVideoHandlerForQA(t)
	defer cleanup()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id FROM chat_participants WHERE chat_id=$1`)).
		WithArgs("chat-1").WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow("u1").AddRow("u2"))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO call_sessions`)).
		WithArgs(sqlmock.AnyArg(), "chat-1", sqlmock.AnyArg(), "u1", "video", "active", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO call_participants`)).
		WithArgs(sqlmock.AnyArg(), "u1").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO call_participants`)).
		WithArgs(sqlmock.AnyArg(), "u2").WillReturnResult(sqlmock.NewResult(1, 1))
	body, _ := json.Marshal(models.InitiateCallRequest{ChatID: "chat-1", Type: "video"})
	w := serveVideo(t, h.InitiateCall, "POST", "/calls/initiate", "/calls/initiate", "u1", body)
	if w.Code != 201 {
		t.Fatalf("expected 201 got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("resp json: %v", err)
	}
	if resp["session"] == nil || resp["offer"] == nil {
		t.Fatalf("expected session+offer")
	}
}

func TestVideoQA_Signal_VideoToggle_Success(t *testing.T) {
	h, mock, cleanup := newVideoHandlerForQA(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-v1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-v1", "chat-1", `["u1","u2"]`, "video", "active", now, nil))
	body, _ := json.Marshal(map[string]interface{}{"type": "video-toggle", "data": map[string]interface{}{"enabled": true}})
	w := serveVideo(t, h.HandleWebRTCSignaling, "POST", "/calls/:callId/signal", "/calls/call-v1/signal", "u1", body)
	if w.Code != 200 {
		t.Fatalf("video-toggle 200 got %d %s", w.Code, w.Body.String())
	}
}

func TestVideoQA_Signal_ScreenShareStart_Success(t *testing.T) {
	h, mock, cleanup := newVideoHandlerForQA(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-v1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-v1", "chat-1", `["u1","u2"]`, "video", "active", now, nil))
	body, _ := json.Marshal(map[string]string{"type": "screen-share-start"})
	w := serveVideo(t, h.HandleWebRTCSignaling, "POST", "/calls/:callId/signal", "/calls/call-v1/signal", "u1", body)
	if w.Code != 200 {
		t.Fatalf("screen-share-start 200 got %d %s", w.Code, w.Body.String())
	}
}

func TestVideoQA_Signal_ScreenShareStop_Success(t *testing.T) {
	h, mock, cleanup := newVideoHandlerForQA(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-v1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-v1", "chat-1", `["u1","u2"]`, "video", "active", now, nil))
	body, _ := json.Marshal(map[string]string{"type": "screen-share-stop"})
	w := serveVideo(t, h.HandleWebRTCSignaling, "POST", "/calls/:callId/signal", "/calls/call-v1/signal", "u1", body)
	if w.Code != 200 {
		t.Fatalf("screen-share-stop 200 got %d %s", w.Code, w.Body.String())
	}
}

func TestVideoQA_Signal_IceCandidate_Success(t *testing.T) {
	h, mock, cleanup := newVideoHandlerForQA(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-v1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-v1", "chat-1", `["u1","u2"]`, "video", "active", now, nil))
	body, _ := json.Marshal(map[string]string{"type": "ice-candidate", "candidate": "candidate:1 1 UDP 123"})
	w := serveVideo(t, h.HandleWebRTCSignaling, "POST", "/calls/:callId/signal", "/calls/call-v1/signal", "u1", body)
	if w.Code != 200 {
		t.Fatalf("ice-candidate 200 got %d %s", w.Code, w.Body.String())
	}
}

func TestVideoQA_Signal_Offer_RequiresSDP(t *testing.T) {
	h, mock, cleanup := newVideoHandlerForQA(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-v1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-v1", "chat-1", `["u1","u2"]`, "video", "active", now, nil))
	body, _ := json.Marshal(map[string]string{"type": "offer"})
	w := serveVideo(t, h.HandleWebRTCSignaling, "POST", "/calls/:callId/signal", "/calls/call-v1/signal", "u1", body)
	if w.Code != 400 {
		t.Fatalf("offer without sdp 400 got %d %s", w.Code, w.Body.String())
	}
}

func TestVideoQA_Signal_NotParticipant_Forbidden(t *testing.T) {
	h, mock, cleanup := newVideoHandlerForQA(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-v1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-v1", "chat-1", `["u1","u2"]`, "video", "active", now, nil))
	body, _ := json.Marshal(map[string]string{"type": "video-toggle"})
	w := serveVideo(t, h.HandleWebRTCSignaling, "POST", "/calls/:callId/signal", "/calls/call-v1/signal", "u9", body)
	if w.Code != 403 {
		t.Fatalf("not participant 403 got %d %s", w.Code, w.Body.String())
	}
}

func TestVideoQA_Signal_AlreadyEnded_Rejects(t *testing.T) {
	h, mock, cleanup := newVideoHandlerForQA(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-v1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-v1", "chat-1", `["u1","u2"]`, "video", "ended", now, now))
	body, _ := json.Marshal(map[string]string{"type": "screen-share-start"})
	w := serveVideo(t, h.HandleWebRTCSignaling, "POST", "/calls/:callId/signal", "/calls/call-v1/signal", "u1", body)
	if w.Code != 400 {
		t.Fatalf("ended 400 got %d %s", w.Code, w.Body.String())
	}
}

func TestVideoQA_GetSession_Video_ReturnsType(t *testing.T) {
	h, mock, cleanup := newVideoHandlerForQA(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-v1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-v1", "chat-1", `["u1","u2"]`, "video", "active", now, nil))
	w := serveVideo(t, h.GetCallSession, "GET", "/calls/:callId", "/calls/call-v1", "u1", nil)
	if w.Code != 200 {
		t.Fatalf("200 got %d %s", w.Code, w.Body.String())
	}
	var sess models.CallSession
	if err := json.Unmarshal(w.Body.Bytes(), &sess); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if sess.Type != "video" {
		t.Fatalf("type video got %s", sess.Type)
	}
}

func TestVideoQA_ImmersiveCaptions_VideoCaptionFlow(t *testing.T) {
	h, mock, cleanup := newVideoHandlerForQA(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-v1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-v1", "chat-1", `["u1","u2"]`, "video", "active", now, nil))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT DISTINCT native_language FROM users WHERE id = ANY`)).
		WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"native_language"}).AddRow("en"))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM call_transcripts WHERE call_id=$1`)).
		WithArgs("call-v1").WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO call_transcripts`)).
		WithArgs(sqlmock.AnyArg(), "call-v1", sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	body, _ := json.Marshal(map[string]string{"text": "Hola video immersive", "language": "es"})
	w := serveVideo(t, h.PostCaption, "POST", "/calls/:callId/captions", "/calls/call-v1/captions", "u1", body)
	if w.Code != 201 {
		t.Fatalf("video caption 201 got %d %s", w.Code, w.Body.String())
	}
}
