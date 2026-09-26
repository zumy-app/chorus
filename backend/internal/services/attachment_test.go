package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// TestAttachmentService_ResolveType verifies that media types resolve from an
// explicit type field, the file extension, and known document extensions —
// the task 6.6 PDF/doc/xlsx class plus image/video/audio.
func TestAttachmentService_ResolveType(t *testing.T) {
	s := &AttachmentService{}

	cases := []struct {
		explicit string
		name     string
		want     string
		wantErr  bool
	}{
		{explicit: "document", name: "x", want: "document"},
		{explicit: "image", name: "x", want: "image"},
		{explicit: "VIDEO", name: "x", want: "video"},
		{explicit: "bogus", name: "x", want: "", wantErr: true},
		{name: "photo.JPG", want: "image"},
		{name: "movie.mp4", want: "video"},
		{name: "clip.mp3", want: "audio"},
		{name: "report.pdf", want: "document"},
		{name: "resume.doc", want: "document"},
		{name: "budget.xlsx", want: "document"},
		{name: "notes.txt", want: "document"},
		{name: "archive.unknown", want: "", wantErr: true},
	}

	for _, c := range cases {
		got, err := s.resolveType(c.explicit, c.name)
		if c.wantErr {
			if err == nil {
				t.Fatalf("resolveType(%q, %q): expected error, got %q", c.explicit, c.name, got)
			}
			continue
		}
		if err != nil {
			t.Fatalf("resolveType(%q, %q): unexpected error %v", c.explicit, c.name, err)
		}
		if got != c.want {
			t.Fatalf("resolveType(%q, %q) = %q, want %q", c.explicit, c.name, got, c.want)
		}
	}
}

// TestAttachmentService_SupportedType verifies the allowed set is enforced.
func TestAttachmentService_SupportedType(t *testing.T) {
	s := &AttachmentService{}
	for _, typ := range []string{"image", "video", "audio", "document"} {
		if !s.SupportedType(typ) {
			t.Fatalf("expected %q to be supported", typ)
		}
	}
	if s.SupportedType("memes") {
		t.Fatal("expected 'memes' to be unsupported")
	}
}

// TestAttachmentService_SendFile verifies the full share path: the message +
// media_attachments rows are written in a transaction, the bytes land on disk,
// and the returned message carries the populated attachment (task 6.6).
func TestAttachmentService_SendFile(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	dir := t.TempDir()
	service, err := NewAttachmentService(db, dir, "/media", 50<<20)
	if err != nil {
		t.Fatalf("NewAttachmentService failed: %v", err)
	}

	mock.ExpectBegin()

	mock.ExpectQuery(`INSERT INTO messages`).
		WithArgs("chat-9", "user-1", "report.pdf").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "chat_id", "sender_id", "text", "original_language",
			"translations", "delivery_status", "reply_to_id", "created_at",
		}).AddRow("msg-1", "chat-9", "user-1", "report.pdf", "", "{}", "sent", nil, time.Now()))

	mock.ExpectExec(`INSERT INTO media_attachments`).
		WithArgs(sqlmock.AnyArg(), "msg-1", "document", "report.pdf", int64(11), "application/pdf", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	msg, err := service.SendFile(context.Background(), "chat-9", "user-1", "", "report.pdf", "", strings.NewReader("hello world"), 11)
	if err != nil {
		t.Fatalf("SendFile failed: %v", err)
	}

	if msg.ID != "msg-1" || msg.ChatID != "chat-9" || msg.SenderID != "user-1" {
		t.Fatalf("unexpected message: %+v", msg)
	}
	if msg.Text != "report.pdf" {
		t.Fatalf("expected caption fallback to file name, got %q", msg.Text)
	}
	if len(msg.Media) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(msg.Media))
	}

	att := msg.Media[0]
	if att.Type != "document" || att.MessageID != "msg-1" || att.FileName != "report.pdf" || att.MimeType != "application/pdf" {
		t.Fatalf("unexpected attachment: %+v", att)
	}
	if att.URL != "/media/"+att.ID+"/report.pdf" {
		t.Fatalf("unexpected attachment URL: %q", att.URL)
	}

	// The bytes were actually written to disk under the upload dir.
	stored := filepath.Join(dir, att.ID, "report.pdf")
	data, err := os.ReadFile(stored)
	if err != nil {
		t.Fatalf("stored file not found: %v", err)
	}
	if string(data) != "hello world" {
		t.Fatalf("stored file content = %q, want %q", data, "hello world")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestAttachmentService_SendFileCaption verifies a supplied caption becomes the
// message text (rather than the file name).
func TestAttachmentService_SendFileCaption(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	service, err := NewAttachmentService(db, t.TempDir(), "/media", 50<<20)
	if err != nil {
		t.Fatalf("NewAttachmentService failed: %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO messages`).
		WithArgs("chat-9", "user-1", "see attached terms").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "chat_id", "sender_id", "text", "original_language",
			"translations", "delivery_status", "reply_to_id", "created_at",
		}).AddRow("msg-2", "chat-9", "user-1", "see attached terms", "", "{}", "sent", nil, time.Now()))
	mock.ExpectExec(`INSERT INTO media_attachments`).
		WithArgs(sqlmock.AnyArg(), "msg-2", "document", "terms.pdf", int64(4), "application/pdf", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	msg, err := service.SendFile(context.Background(), "chat-9", "user-1", "see attached terms", "terms.pdf", "document", strings.NewReader("ACME"), 4)
	if err != nil {
		t.Fatalf("SendFile failed: %v", err)
	}
	if msg.Text != "see attached terms" {
		t.Fatalf("expected caption as message text, got %q", msg.Text)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestAttachmentService_SendFileErrors verifies the size cap and the
// unsupported-type rail — neither path should write a DB row.
func TestAttachmentService_SendFileErrors(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	service, err := NewAttachmentService(db, t.TempDir(), "/media", 1024)
	if err != nil {
		t.Fatalf("NewAttachmentService failed: %v", err)
	}

	if _, err := service.SendFile(context.Background(), "chat-9", "user-1", "", "big.pdf", "", strings.NewReader("x"), 2048); err == nil {
		t.Fatal("expected an error for a file over the size cap")
	}

	if _, err := service.SendFile(context.Background(), "chat-9", "user-1", "", "notes.xyz", "bogus", strings.NewReader("x"), 1); err == nil {
		t.Fatal("expected an error for an unsupported type")
	}

	// No DB interactions should have been recorded.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestSanitizeFileName verifies that unsafe names are reduced to a safe
// basename that cannot traverse the upload directory.
func TestSanitizeFileName(t *testing.T) {
	cases := map[string]string{
		"report.pdf":                   "report.pdf",
		"../../etc/passwd":             "passwd",
		"file with spaces.pdf":         "file_with_spaces.pdf",
		`C:\Windows\system32\evil.dll`: "evil.dll",
		"":                             "",
	}
	for in, want := range cases {
		if got := sanitizeFileName(in); got != want {
			t.Fatalf("sanitizeFileName(%q) = %q, want %q", in, got, want)
		}
	}
}
