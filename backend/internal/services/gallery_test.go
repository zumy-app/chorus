package services

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
)

// TestGalleryService_GetChatGallery verifies the per-chat gallery (task 6.5):
// participant-scoped query, type-filter applied, sender populated, pagination
// metadata and per-type counts returned together.
func TestGalleryService_GetChatGallery(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewGalleryService(db)
	now := time.Now()

	// 1) per-type counts (independent of the type filter)
	mock.ExpectQuery(`SELECT ma\.type, COUNT\(\*\) FROM media_attachments`).
		WithArgs("user-1", "chat-1").
		WillReturnRows(sqlmock.NewRows([]string{"type", "count"}).
			AddRow("image", 2).
			AddRow("video", 1).
			AddRow("document", 3).
			AddRow("link", 4))

	// 2) filtered count for type=image,video (media tab)
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM media_attachments`).
		WithArgs("user-1", "chat-1", pq.Array([]string{"image", "video"})).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// 3) data query (media tab) with sender columns
	mock.ExpectQuery(`SELECT ma\.id, ma\.message_id, ma\.type, ma\.file_name, ma\.file_size,\s+ma\.mime_type, ma\.url, COALESCE\(ma\.thumbnail_url, ''\),\s+ma\.created_at, m\.chat_id,\s+ma\.latitude, ma\.longitude, COALESCE\(ma\.location_name, ''\),\s+COALESCE\(su\.id, ''\), COALESCE\(su\.display_name, ''\), COALESCE\(su\.username, ''\)`).
		WithArgs("user-1", "chat-1", pq.Array([]string{"image", "video"}), 30, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "message_id", "type", "file_name", "file_size", "mime_type",
			"url", "thumbnail_url", "created_at", "chat_id",
			"latitude", "longitude", "location_name",
			"sender_id", "sender_name", "sender_username",
		}).
			AddRow("att-1", "msg-1", "video", "clip.mp4", 999, "video/mp4", "http://cdn/clip.mp4", "http://cdn/clip-t.jpg", now, "chat-1", nil, nil, "", "u-maria", "Maria", "maria").
			AddRow("att-2", "msg-2", "image", "pic.jpg", 111, "image/jpeg", "http://cdn/pic.jpg", "", now, "chat-1", nil, nil, "", "", "", ""))

	res, err := s.GetChatGallery(context.Background(), "user-1", "chat-1", "media", 30, 0)
	if err != nil {
		t.Fatalf("GetChatGallery failed: %v", err)
	}

	if res.Total != 2 {
		t.Fatalf("expected Total 2, got %d", res.Total)
	}
	if len(res.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(res.Items))
	}
	if res.HasMore {
		t.Fatal("expected HasMore false")
	}
	if res.Counts.Image != 2 || res.Counts.Video != 1 || res.Counts.Document != 3 || res.Counts.Link != 4 {
		t.Fatalf("unexpected counts: %+v", res.Counts)
	}

	att := res.Items[0]
	if att.ID != "att-1" || att.Type != "video" || att.ChatID != "chat-1" {
		t.Fatalf("unexpected attachment: %+v", att)
	}
	if att.ThumbnailURL == nil || *att.ThumbnailURL != "http://cdn/clip-t.jpg" {
		t.Fatalf("expected thumbnail URL, got %v", att.ThumbnailURL)
	}
	if att.Sender == nil || att.Sender.DisplayName != "Maria" || att.Sender.Username != "maria" {
		t.Fatalf("expected sender 'Maria' populated, got %+v", att.Sender)
	}

	// Second row has no sender (empty sender columns should not leak a blank user).
	if res.Items[1].Sender != nil {
		t.Fatalf("expected nil sender when sender columns are empty, got %+v", res.Items[1].Sender)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestGalleryService_GetChatGalleryLinksFilter verifies the links tab alias and
// the empty-gallery shape (non-nil slice, zero total).
func TestGalleryService_GetChatGalleryLinksFilter(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	s := NewGalleryService(db)

	mock.ExpectQuery(`SELECT ma\.type, COUNT\(\*\) FROM media_attachments`).
		WithArgs("user-1", "chat-1").
		WillReturnRows(sqlmock.NewRows([]string{"type", "count"}).AddRow("link", 0))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM media_attachments`).
		WithArgs("user-1", "chat-1", pq.Array([]string{"link"})).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	res, err := s.GetChatGallery(context.Background(), "user-1", "chat-1", "links", 30, 0)
	if err != nil {
		t.Fatalf("GetChatGallery failed: %v", err)
	}
	if res.Total != 0 || res.HasMore {
		t.Fatalf("expected Total 0 and HasMore false, got %+v", res)
	}
	if res.Items == nil {
		t.Fatal("expected non-nil (empty) items slice")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestGalleryTypeSet ensures the friendly tab aliases and raw type lists map
// to the expected concrete media types.
func TestGalleryTypeSet(t *testing.T) {
	cases := []struct {
		in   string
		want int
		ok   bool
	}{
		{"", 0, true},
		{"media", 2, true},
		{"docs", 1, true},
		{"links", 1, true},
		{"image,video,audio", 3, true},
		{"link, document", 2, true},
		{"nonsense", 0, true},
		{"image, bogus", 1, true},
	}
	for _, c := range cases {
		got := galleryTypeSet(c.in)
		if c.ok && len(got) != c.want {
			t.Fatalf("galleryTypeSet(%q) = %v, want %d types", c.in, got, c.want)
		}
	}
}
