package services

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestVideoQA_HandleSignal_VideoToggle_Success(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1","u2"]`, "video", "active", now, nil))
	if err := svc.HandleSignal(context.Background(), "call-1", "u1", "video-toggle", "", "", map[string]interface{}{"enabled": true}); err != nil {
		t.Fatalf("video-toggle: %v", err)
	}
}

func TestVideoQA_HandleSignal_ScreenShare_StartStop(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	now := time.Now()
	for _, typ := range []string{"screen-share-start", "screen-share-stop"} {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
			WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
				AddRow("call-1", "chat-1", `["u1","u2"]`, "video", "active", now, nil))
		if err := svc.HandleSignal(context.Background(), "call-1", "u1", typ, "", "", nil); err != nil {
			t.Fatalf("%s: %v", typ, err)
		}
	}
}

func TestVideoQA_HandleSignal_ScreenShare_UnderscoreAlias(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1","u2"]`, "video", "active", now, nil))
	if err := svc.HandleSignal(context.Background(), "call-1", "u1", "screen_share_start", "", "", nil); err != nil {
		t.Fatalf("underscore alias: %v", err)
	}
}

func TestVideoQA_InitiateCall_Video_SetsVideoType(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id FROM chat_participants WHERE chat_id=$1`)).
		WithArgs("chat-1").WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow("u1").AddRow("u2"))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO call_sessions`)).
		WithArgs(sqlmock.AnyArg(), "chat-1", sqlmock.AnyArg(), "u1", "video", "active", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO call_participants`)).
		WithArgs(sqlmock.AnyArg(), "u1").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO call_participants`)).
		WithArgs(sqlmock.AnyArg(), "u2").WillReturnResult(sqlmock.NewResult(1, 1))
	sess, err := svc.InitiateCall(context.Background(), "chat-1", "u1", "video")
	if err != nil {
		t.Fatalf("initiate video: %v", err)
	}
	if sess.Type != "video" {
		t.Fatalf("expected video got %s", sess.Type)
	}
	if sess.Status != "active" {
		t.Fatalf("status %s", sess.Status)
	}
}

func TestVideoQA_InitiateCall_AudioFallbackForUnknown(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id FROM chat_participants WHERE chat_id=$1`)).
		WithArgs("chat-1").WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow("u1"))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO call_sessions`)).
		WithArgs(sqlmock.AnyArg(), "chat-1", sqlmock.AnyArg(), "u1", "audio", "active", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO call_participants`)).
		WithArgs(sqlmock.AnyArg(), "u1").WillReturnResult(sqlmock.NewResult(1, 1))
	sess, err := svc.InitiateCall(context.Background(), "chat-1", "u1", "unknown-type")
	if err != nil {
		t.Fatalf("fallback: %v", err)
	}
	if sess.Type != "audio" {
		t.Fatalf("fallback type %s", sess.Type)
	}
}

func TestVideoQA_GenerateOffer_TurnServers(t *testing.T) {
	t.Setenv("WEBRTC_STUN_URLS", "stun:stun.example.com:3478")
	t.Setenv("WEBRTC_TURN_URLS", "turn:turn.example.com:3478")
	t.Setenv("WEBRTC_TURN_USERNAME", "user")
	t.Setenv("WEBRTC_TURN_CREDENTIAL", "pass")
	svc, _, cleanup := newCallServiceMock(t)
	defer cleanup()
	offer, err := svc.GenerateWebRTCOffer(context.Background(), "call-v1")
	if err != nil {
		t.Fatalf("offer: %v", err)
	}
	if len(offer.ICEServers) != 2 {
		t.Fatalf("expected 2 ice servers got %d", len(offer.ICEServers))
	}
	if offer.ICEServers[1].Username != "user" || offer.ICEServers[1].Credential != "pass" {
		t.Fatalf("turn creds %+v", offer.ICEServers[1])
	}
}

func TestVideoQA_GetCallSession_VideoFields(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-v1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-v1", "chat-1", `["u1","u2"]`, "video", "active", now, nil))
	sess, err := svc.GetCallSession(context.Background(), "call-v1")
	if err != nil {
		t.Fatalf("get video session: %v", err)
	}
	if sess.Type != "video" || sess.ID != "call-v1" {
		t.Fatalf("bad %+v", sess)
	}
}

func TestVideoQA_HandleSignal_IceCandidate_JSONCandidate(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1","u2"]`, "video", "active", now, nil))
	extra := map[string]interface{}{"candidate": map[string]interface{}{"candidate": "candidate:1", "sdpMid": "0"}}
	if err := svc.HandleSignal(context.Background(), "call-1", "u1", "ice-candidate", "", "", extra); err != nil {
		t.Fatalf("ice candidate map: %v", err)
	}
}

func TestVideoQA_HandleSignal_Answer_RequiresSDP(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1","u2"]`, "video", "active", now, nil))
	err := svc.HandleSignal(context.Background(), "call-1", "u1", "answer", "", "", nil)
	if err == nil || err.Error() != "sdp required for answer" {
		t.Fatalf("expected sdp required for answer got %v", err)
	}
}

func TestVideoQA_PublishCaption_VideoCall_Success(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
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
	seg, err := svc.PublishLiveCaption(context.Background(), "call-v1", "u1", "Hello video", "en")
	if err != nil {
		t.Fatalf("publish video caption: %v", err)
	}
	if seg.OriginalText != "Hello video" {
		t.Fatalf("text %s", seg.OriginalText)
	}
}

func TestVideoQA_ValidSignalTypes_Includes_Video(t *testing.T) {
	if !validSignalTypes["video-toggle"] {
		t.Fatal("video-toggle should be valid")
	}
	if !validSignalTypes["screen-share-start"] || !validSignalTypes["screen-share-stop"] {
		t.Fatal("screen share types should be valid")
	}
}
