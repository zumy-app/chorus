package services

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
)

func newCallServiceMock(t *testing.T) (*CallService, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	svc := NewCallService(db, nil, nil)
	return svc, mock, func() { db.Close() }
}

func TestCallQA_GenerateOffer_DefaultStun(t *testing.T) {
	t.Setenv("WEBRTC_STUN_URLS", "")
	t.Setenv("WEBRTC_TURN_URLS", "")
	t.Setenv("WEBRTC_TURN_URL", "")
	svc, _, cleanup := newCallServiceMock(t)
	defer cleanup()
	offer, err := svc.GenerateWebRTCOffer(context.Background(), "call-1")
	if err != nil {
		t.Fatalf("offer: %v", err)
	}
	if offer.CallID != "call-1" || offer.Type != "offer" {
		t.Fatalf("unexpected offer %+v", offer)
	}
	if len(offer.ICEServers) == 0 || offer.ICEServers[0].URLs[0] != "stun:stun.l.google.com:19302" {
		t.Fatalf("default stun wrong %+v", offer.ICEServers)
	}
}

func TestCallQA_GenerateOffer_CustomStun(t *testing.T) {
	t.Setenv("WEBRTC_STUN_URLS", "stun:custom.example.com:3478")
	t.Setenv("WEBRTC_TURN_URLS", "")
	t.Setenv("WEBRTC_TURN_URL", "")
	svc, _, cleanup := newCallServiceMock(t)
	defer cleanup()
	offer, _ := svc.GenerateWebRTCOffer(context.Background(), "c1")
	if offer.ICEServers[0].URLs[0] != "stun:custom.example.com:3478" {
		t.Fatalf("custom stun got %v", offer.ICEServers[0].URLs)
	}
}

func TestCallQA_ValidSignalTypes(t *testing.T) {
	for _, typ := range []string{"offer", "answer", "ice-candidate", "screen-share-start", "screen-share-stop", "video-toggle"} {
		if !validSignalTypes[typ] {
			t.Fatalf("should be valid %s", typ)
		}
	}
	if validSignalTypes["invalid"] {
		t.Fatal("invalid should not be valid")
	}
}

func TestCallQA_InitiateCall_ChatNotFound(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id FROM chat_participants WHERE chat_id=$1`)).
		WithArgs("chat-1").WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
	_, err := svc.InitiateCall(context.Background(), "chat-1", "u1", "audio")
	if err != ErrChatNotFound {
		t.Fatalf("expected ErrChatNotFound got %v", err)
	}
}

func TestCallQA_InitiateCall_NotParticipant(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id FROM chat_participants WHERE chat_id=$1`)).
		WithArgs("chat-1").WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow("u2"))
	_, err := svc.InitiateCall(context.Background(), "chat-1", "u1", "audio")
	if err != ErrNotParticipant {
		t.Fatalf("expected ErrNotParticipant got %v", err)
	}
}

func TestCallQA_InitiateCall_Success(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id FROM chat_participants WHERE chat_id=$1`)).
		WithArgs("chat-1").WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow("u1").AddRow("u2"))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO call_sessions`)).
		WithArgs(sqlmock.AnyArg(), "chat-1", sqlmock.AnyArg(), "u1", "audio", "active", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO call_participants`)).
		WithArgs(sqlmock.AnyArg(), "u1").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO call_participants`)).
		WithArgs(sqlmock.AnyArg(), "u2").WillReturnResult(sqlmock.NewResult(1, 1))
	sess, err := svc.InitiateCall(context.Background(), "chat-1", "u1", "audio")
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}
	if sess.ChatID != "chat-1" || sess.Status != "active" {
		t.Fatalf("bad session %+v", sess)
	}
	if len(sess.Participants) != 2 {
		t.Fatalf("participants %v", sess.Participants)
	}
}

func TestCallQA_InitiateCall_VideoType(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id FROM chat_participants WHERE chat_id=$1`)).
		WithArgs("chat-1").WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow("u1"))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO call_sessions`)).
		WithArgs(sqlmock.AnyArg(), "chat-1", sqlmock.AnyArg(), "u1", "video", "active", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO call_participants`)).
		WithArgs(sqlmock.AnyArg(), "u1").WillReturnResult(sqlmock.NewResult(1, 1))
	sess, err := svc.InitiateCall(context.Background(), "chat-1", "u1", "video")
	if err != nil {
		t.Fatalf("video init: %v", err)
	}
	if sess.Type != "video" {
		t.Fatalf("expected video got %s", sess.Type)
	}
}

func TestCallQA_EndCall_AlreadyEnded(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1","u2"]`, "audio", "ended", now, now))
	err := svc.EndCallAs(context.Background(), "call-1", "u1")
	if err != ErrCallAlreadyEnded {
		t.Fatalf("expected already ended got %v", err)
	}
}

func TestCallQA_EndCall_NotParticipant(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1","u2"]`, "audio", "active", now, nil))
	err := svc.EndCallAs(context.Background(), "call-1", "u9")
	if err != ErrNotParticipant {
		t.Fatalf("expected not participant got %v", err)
	}
}

func TestCallQA_GetCallSession_NotFound(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("missing").WillReturnError(sql.ErrNoRows)
	_, err := svc.GetCallSession(context.Background(), "missing")
	if err != ErrCallNotFound {
		t.Fatalf("expected not found got %v", err)
	}
}

func TestCallQA_HandleSignal_InvalidType(t *testing.T) {
	svc, _, cleanup := newCallServiceMock(t)
	defer cleanup()
	err := svc.HandleSignal(context.Background(), "call-1", "u1", "invalid-type", "", "", nil)
	if err != ErrInvalidSignal {
		t.Fatalf("expected invalid signal got %v", err)
	}
}

func TestCallQA_HandleSignal_MissingSDP(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1","u2"]`, "audio", "active", now, nil))
	err := svc.HandleSignal(context.Background(), "call-1", "u1", "offer", "", "", nil)
	if err == nil || err.Error() != "sdp required for offer" {
		t.Fatalf("expected sdp required got %v", err)
	}
}

func TestCallQA_HandleSignal_MissingCandidate(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1","u2"]`, "audio", "active", now, nil))
	err := svc.HandleSignal(context.Background(), "call-1", "u1", "ice-candidate", "", "", nil)
	if err == nil || err.Error() != "candidate required for ice-candidate" {
		t.Fatalf("expected candidate required got %v", err)
	}
}

func TestCallQA_HandleSignal_AlreadyEnded(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1","u2"]`, "audio", "ended", now, now))
	err := svc.HandleSignal(context.Background(), "call-1", "u1", "offer", "sdp-data", "", nil)
	if err != ErrCallAlreadyEnded {
		t.Fatalf("expected ended got %v", err)
	}
}

func TestCallQA_HandleSignal_NotParticipant(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1","u2"]`, "audio", "active", now, nil))
	err := svc.HandleSignal(context.Background(), "call-1", "u9", "screen-share-start", "", "", nil)
	if err != ErrNotParticipant {
		t.Fatalf("expected not participant got %v", err)
	}
}

func TestCallQA_PublishCaption_EmptyText(t *testing.T) {
	svc, _, cleanup := newCallServiceMock(t)
	defer cleanup()
	_, err := svc.PublishLiveCaption(context.Background(), "call-1", "u1", "   ", "en")
	if err == nil || err.Error() != "text required" {
		t.Fatalf("expected text required got %v", err)
	}
}

func TestCallQA_PublishCaption_NotParticipant(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1","u2"]`, "audio", "active", now, nil))
	_, err := svc.PublishLiveCaption(context.Background(), "call-1", "u9", "hello", "en")
	if err != ErrNotParticipant {
		t.Fatalf("expected not participant got %v", err)
	}
}

func TestCallQA_PublishCaption_CallEnded(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1","u2"]`, "audio", "ended", now, now))
	_, err := svc.PublishLiveCaption(context.Background(), "call-1", "u1", "hello", "en")
	if err != ErrCallAlreadyEnded {
		t.Fatalf("expected ended got %v", err)
	}
}

func TestCallQA_PublishCaption_Success(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1","u2"]`, "audio", "active", now, nil))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT DISTINCT native_language FROM users WHERE id = ANY`)).
		WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"native_language"}).AddRow("en").AddRow("es"))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM call_transcripts WHERE call_id=$1`)).
		WithArgs("call-1").WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO call_transcripts`)).
		WithArgs(sqlmock.AnyArg(), "call-1", sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	seg, err := svc.PublishLiveCaption(context.Background(), "call-1", "u1", "Hola mundo", "es")
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if seg.OriginalText != "Hola mundo" {
		t.Fatalf("text %s", seg.OriginalText)
	}
	if seg.SpeakerID != "u1" {
		t.Fatalf("speaker %s", seg.SpeakerID)
	}
}

func TestCallQA_GetCaptionsPaginated_NotParticipant(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1","u2"]`, "audio", "active", now, nil))
	_, _, err := svc.GetCaptionsPaginated(context.Background(), "call-1", "u9", 10, 0)
	if err != ErrNotParticipant {
		t.Fatalf("expected not participant got %v", err)
	}
}

func TestCallQA_GetCaptionsPaginated_Pagination(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1","u2"]`, "audio", "active", now, nil))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, call_id, segments, created_at FROM call_transcripts WHERE call_id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "call_id", "segments", "created_at"}).
			AddRow("tr-1", "call-1", `[{"speakerId":"u1","startTime":1,"endTime":2,"originalText":"hello","originalLanguage":"en","translations":{},"confidence":1}]`, now))
	segs, total, err := svc.GetCaptionsPaginated(context.Background(), "call-1", "u1", 10, 0)
	if err != nil {
		t.Fatalf("paginated: %v", err)
	}
	if total != 1 || len(segs) != 1 {
		t.Fatalf("expected 1 got %d total %d", len(segs), total)
	}
	// offset beyond total
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1","u2"]`, "audio", "active", now, nil))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, call_id, segments, created_at FROM call_transcripts WHERE call_id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "call_id", "segments", "created_at"}).
			AddRow("tr-1", "call-1", `[{"speakerId":"u1","startTime":1,"endTime":2,"originalText":"hello","originalLanguage":"en","translations":{},"confidence":1}]`, now))
	segs, total, err = svc.GetCaptionsPaginated(context.Background(), "call-1", "u1", 10, 5)
	if err != nil || len(segs) != 0 || total != 1 {
		t.Fatalf("offset beyond: %v segs %d total %d", err, len(segs), total)
	}
}

func TestCallQA_Bookmark_OutOfRange(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1"]`, "audio", "active", now, nil))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, call_id, segments, created_at FROM call_transcripts WHERE call_id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "call_id", "segments", "created_at"}).
			AddRow("tr-1", "call-1", `[{"speakerId":"u1","startTime":1,"endTime":2,"originalText":"hello","originalLanguage":"en","translations":{},"confidence":1}]`, now))
	_, err := svc.BookmarkCaptionWithPhrase(context.Background(), "call-1", "u1", 5, "")
	if err == nil || err.Error() != "segment index out of range" {
		t.Fatalf("expected out of range got %v", err)
	}
}

func TestCallQA_Bookmark_NotParticipant(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1"]`, "audio", "active", now, nil))
	_, err := svc.BookmarkCaptionWithPhrase(context.Background(), "call-1", "u9", 0, "hello")
	if err != ErrNotParticipant {
		t.Fatalf("expected not participant got %v", err)
	}
}

func TestCallQA_Bookmark_EmptySegment(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1"]`, "audio", "active", now, nil))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, call_id, segments, created_at FROM call_transcripts WHERE call_id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "call_id", "segments", "created_at"}).
			AddRow("tr-1", "call-1", `[{"speakerId":"u1","startTime":1,"endTime":2,"originalText":"","originalLanguage":"en","translations":{},"confidence":1}]`, now))
	_, err := svc.BookmarkCaptionWithPhrase(context.Background(), "call-1", "u1", 0, "   ")
	if err == nil || err.Error() != "empty segment" {
		t.Fatalf("expected empty segment got %v", err)
	}
	_ = models.VocabularyEntry{}
}

func TestCallQA_SearchTranscripts_Filter(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock: %v", err)
	}
	defer db.Close()
	svc := NewCallService(db, nil, nil)
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT ct.id, ct.call_id, ct.segments, ct.created_at`)).
		WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"id", "call_id", "segments", "created_at"}).
			AddRow("tr-1", "call-1", `[{"speakerId":"u1","startTime":1,"endTime":2,"originalText":"Hello world","originalLanguage":"en","translations":{"es":"Hola mundo"},"confidence":1}]`, now).
			AddRow("tr-2", "call-2", `[{"speakerId":"u1","startTime":1,"endTime":2,"originalText":"Goodbye","originalLanguage":"en","translations":{},"confidence":1}]`, now))
	results, err := svc.SearchTranscripts(context.Background(), "u1", "hello", "")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 filtered got %d", len(results))
	}
}

func TestCallQA_DeleteTranscript_NotParticipant(t *testing.T) {
	svc, mock, cleanup := newCallServiceMock(t)
	defer cleanup()
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`)).
		WithArgs("call-1").WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "participants", "type", "status", "started_at", "ended_at"}).
			AddRow("call-1", "chat-1", `["u1"]`, "audio", "active", now, nil))
	err := svc.DeleteCallTranscript(context.Background(), "call-1", "u9")
	if err != ErrNotParticipant {
		t.Fatalf("expected not participant got %v", err)
	}
}

func TestCallQA_TranscribeAndPublish_NoSTT(t *testing.T) {
	svc, _, cleanup := newCallServiceMock(t)
	defer cleanup()
	_, err := svc.TranscribeAndPublish(context.Background(), "call-1", "u1", []byte("audio"), "en")
	if err == nil || err.Error() != "speech service not configured" {
		t.Fatalf("expected stt not configured got %v", err)
	}
}
