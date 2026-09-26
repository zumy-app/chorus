package services

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// TestLocationService_SendLocation verifies the full share path (task 6.7): a
// location write inserts the parent message + a media_attachments row of type
// 'location' in one transaction, and the returned message carries the populated
// pin (lat/lng + a map URL).
func TestLocationService_SendLocation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	service := NewLocationService(db)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO messages`).
		WithArgs("chat-9", "user-1", "Shared a location").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "chat_id", "sender_id", "text", "original_language",
			"translations", "delivery_status", "reply_to_id", "created_at",
		}).AddRow("msg-loc", "chat-9", "user-1", "Shared a location", "", "{}", "sent", nil, time.Now()))
	mock.ExpectExec(`INSERT INTO media_attachments`).
		WithArgs(sqlmock.AnyArg(), "msg-loc", "location", "location", int64(0),
			"application/vnd.chorus.location", sqlmock.AnyArg(), 40.7128, -74.0060, nil).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	msg, err := service.SendLocation(context.Background(), "chat-9", "user-1", 40.7128, -74.0060, "", nil)
	if err != nil {
		t.Fatalf("SendLocation failed: %v", err)
	}

	if msg.ID != "msg-loc" || msg.ChatID != "chat-9" || msg.SenderID != "user-1" {
		t.Fatalf("unexpected message: %+v", msg)
	}
	if len(msg.Media) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(msg.Media))
	}

	att := msg.Media[0]
	if att.Type != "location" || att.MessageID != "msg-loc" {
		t.Fatalf("unexpected attachment: %+v", att)
	}
	if att.Latitude == nil || *att.Latitude != 40.7128 {
		t.Fatalf("expected latitude 40.7128, got %v", att.Latitude)
	}
	if att.Longitude == nil || *att.Longitude != -74.0060 {
		t.Fatalf("expected longitude -74.0060, got %v", att.Longitude)
	}
	if !strings.Contains(att.URL, "40.7128") || !strings.Contains(att.URL, "-74.0060") {
		t.Fatalf("expected a map URL carrying the coords, got %q", att.URL)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestLocationService_SendLocationLabel verifies a supplied label becomes the
// message text and the attachment's location_name.
func TestLocationService_SendLocationLabel(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	service := NewLocationService(db)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO messages`).
		WithArgs("chat-9", "user-1", "Soho").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "chat_id", "sender_id", "text", "original_language",
			"translations", "delivery_status", "reply_to_id", "created_at",
		}).AddRow("msg-loc2", "chat-9", "user-1", "Soho", "", "{}", "sent", nil, time.Now()))
	mock.ExpectExec(`INSERT INTO media_attachments`).
		WithArgs(sqlmock.AnyArg(), "msg-loc2", "location", "location", int64(0),
			"application/vnd.chorus.location", sqlmock.AnyArg(), 51.5074, -0.1278, "Soho").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	msg, err := service.SendLocation(context.Background(), "chat-9", "user-1", 51.5074, -0.1278, "Soho", nil)
	if err != nil {
		t.Fatalf("SendLocation failed: %v", err)
	}
	if msg.Text != "Soho" {
		t.Fatalf("expected label as message text, got %q", msg.Text)
	}
	att := msg.Media[0]
	if att.LocationName != "Soho" {
		t.Fatalf("expected location_name Soho, got %q", att.LocationName)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestLocationService_SendLocationErrors verifies out-of-range coordinates are
// rejected before any DB row is written.
func TestLocationService_SendLocationErrors(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	service := NewLocationService(db)

	cases := []struct {
		lat, lng float64
	}{
		{lat: 91.0, lng: 0},
		{lat: -90.5, lng: 0},
		{lat: 0, lng: 181.0},
		{lat: 0, lng: -181.0},
	}
	for _, c := range cases {
		if _, err := service.SendLocation(context.Background(), "chat-9", "user-1", c.lat, c.lng, "", nil); err == nil {
			t.Fatalf("expected an error for lat=%v lng=%v", c.lat, c.lng)
		}
	}

	// A NaN/Inf coordinate is also rejected.
	if _, err := service.SendLocation(context.Background(), "chat-9", "user-1", math.NaN(), 0, "", nil); err == nil {
		t.Fatal("expected an error for NaN latitude")
	}

	// No DB interactions should have been recorded.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestValidateCoordinates covers the bounds rail directly.
func TestValidateCoordinates(t *testing.T) {
	valid := [][2]float64{
		{0, 0},
		{90, 180},
		{-90, -180},
		{40.7128, -74.0060},
	}
	for _, c := range valid {
		if err := validateCoordinates(c[0], c[1]); err != nil {
			t.Fatalf("expected %v, %v to be valid: %v", c[0], c[1], err)
		}
	}
	invalid := [][2]float64{
		{90.1, 0},
		{-90.1, 0},
		{0, 180.1},
		{0, -180.1},
	}
	for _, c := range invalid {
		if err := validateCoordinates(c[0], c[1]); err == nil {
			t.Fatalf("expected %v, %v to be invalid", c[0], c[1])
		}
	}
}
