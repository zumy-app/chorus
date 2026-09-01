package services

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
)

// TestSearchMedia verifies the dedicated media endpoint (task 6.3): it only
// matches media metadata (file name / type / mime), not message text, and it
// respects the media type + chat filters.
func TestSearchMedia(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewSearchService(db, nil)
	now := time.Now()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM media_attachments`).
		WithArgs("user-1", "%cat%", "image").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT ma\.id, ma\.message_id, ma\.type`).
		WithArgs("user-1", "%cat%", "image", 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "message_id", "type", "file_name", "file_size", "mime_type", "url", "thumbnail_url", "created_at", "chat_id", "latitude", "longitude", "location_name"}).
			AddRow("att-1", "msg-1", "image", "cat-photo.jpg", 2048, "image/jpeg", "http://cdn/att-1.jpg", "http://cdn/att-1-t.jpg", now, "chat-1", nil, nil, ""))

	res, err := s.SearchMedia("user-1", models.SearchRequest{Query: "cat", Limit: 20, MediaType: "image"})
	if err != nil {
		t.Fatalf("SearchMedia failed: %v", err)
	}
	if res.Total != 1 {
		t.Fatalf("expected Total 1, got %d", res.Total)
	}
	if len(res.Media) != 1 {
		t.Fatalf("expected 1 media result, got %d", len(res.Media))
	}
	att := res.Media[0]
	if att.ID != "att-1" || att.Type != "image" || att.MessageID != "msg-1" {
		t.Fatalf("unexpected attachment: %+v", att)
	}
	if att.ChatID != "chat-1" {
		t.Fatalf("expected ChatID 'chat-1', got %q", att.ChatID)
	}
	if att.ThumbnailURL == nil || *att.ThumbnailURL != "http://cdn/att-1-t.jpg" {
		t.Fatalf("expected thumbnail URL, got %v", att.ThumbnailURL)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestSearchMediaNoResults verifies an empty media search still returns a
// well-formed, non-nil result (empty slice, zero total).
func TestSearchMediaNoResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewSearchService(db, nil)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM media_attachments`).
		WithArgs("user-1", "%zzz%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	res, err := s.SearchMedia("user-1", models.SearchRequest{Query: "zzz", Limit: 20})
	if err != nil {
		t.Fatalf("SearchMedia failed: %v", err)
	}
	if res.Total != 0 {
		t.Fatalf("expected Total 0, got %d", res.Total)
	}
	if res.HasMore {
		t.Fatal("expected HasMore false")
	}
	if res.Media == nil {
		t.Fatal("expected non-nil (empty) media slice")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestSearchMessagesUniversal verifies that the universal search (task 6.3)
// returns both matching text messages and their media attachments in a single
// result, and that a sender is populated on message results.
func TestSearchMessagesUniversal(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// Nil redis: the cache write is nil-guarded in SearchMessages, so a nil
	// client keeps the test offline and fast.
	s := NewSearchService(db, nil)
	now := time.Now()

	// 1) message count
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM messages`).
		WithArgs("user-1", "hello").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// 2) message select (message + sender columns)
	mock.ExpectQuery(`SELECT m\.id, m\.chat_id, m\.sender_id, m\.text`).
		WithArgs("user-1", "hello", 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "text", "original_language", "translations", "delivery_status", "reply_to_id", "created_at", "display_name", "username"}).
			AddRow("msg-1", "chat-1", "sender-1", "hello world", "en", "{}", "sent", nil, now, "Alice", "alice"))

	// 3) media count
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM media_attachments`).
		WithArgs("user-1", "%hello%", "hello").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// 4) media select
	mock.ExpectQuery(`SELECT ma\.id, ma\.message_id, ma\.type`).
		WithArgs("user-1", "%hello%", "hello", 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "message_id", "type", "file_name", "file_size", "mime_type", "url", "thumbnail_url", "created_at", "chat_id", "latitude", "longitude", "location_name"}).
			AddRow("att-1", "msg-1", "video", "hello.mp4", 99999, "video/mp4", "http://cdn/hello.mp4", "", now, "chat-1", nil, nil, ""))

	res, err := s.SearchMessages("user-1", models.SearchRequest{Query: "hello", Limit: 20})
	if err != nil {
		t.Fatalf("SearchMessages failed: %v", err)
	}

	if res.Total != 1 {
		t.Fatalf("expected message Total 1, got %d", res.Total)
	}
	if res.MediaTotal != 1 {
		t.Fatalf("expected MediaTotal 1, got %d", res.MediaTotal)
	}
	if len(res.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(res.Messages))
	}
	if len(res.Media) != 1 {
		t.Fatalf("expected 1 media, got %d", len(res.Media))
	}

	msg := res.Messages[0]
	if msg.ID != "msg-1" || msg.Text != "hello world" || msg.ChatID != "chat-1" {
		t.Fatalf("unexpected message: %+v", msg)
	}
	if msg.Sender == nil || msg.Sender.DisplayName != "Alice" {
		t.Fatalf("expected sender 'Alice' populated, got %+v", msg.Sender)
	}

	att := res.Media[0]
	if att.ID != "att-1" || att.Type != "video" || att.ChatID != "chat-1" {
		t.Fatalf("unexpected attachment: %+v", att)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
