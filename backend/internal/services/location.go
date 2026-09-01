package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/google/uuid"
)

// Location media type constants. The 'location' type is a first-class member of
// the media_attachments CHECK constraint (task 6.7) so a shared map pin is
// queryable from the same read paths as every other attachment (chat history via
// AttachMedia, the media gallery, and media search).
const (
	typeLocation = "location"
	// locationMime is the MIME used for a location row. The media_attachments
	// table declares file_name/mime_type NOT NULL, so a pin carries these
	// placeholder values (no file bytes exist for a location).
	locationMime = "application/vnd.chorus.location"
)

// LocationService persists location-sharing messages (task 6.7): it validates
// the coordinates, creates the parent message + a media_attachments row of type
// 'location' in one transaction, and returns the message with the attachment
// populated so callers can broadcast it exactly like a text message. The client
// renders a map pin from latitude/longitude; the URL holds a link to the point
// and the history/gallery/search read paths expose the coordinates.
type LocationService struct {
	db *sql.DB
}

// NewLocationService creates a LocationService backed by the given DB.
func NewLocationService(db *sql.DB) *LocationService {
	return &LocationService{db: db}
}

// Validation bounds for a real-world coordinate pair.
const (
	minLatitude  = -90.0
	maxLatitude  = 90.0
	minLongitude = -180.0
	maxLongitude = 180.0
)

// SendLocation persists a shared location and returns the created message (with
// Media[0] populated). The caller must be a participant of chatID (checked by
// the handler). label, when non-empty, becomes the message text and is stored as
// the attachment's location_name; otherwise a default string is used so a chat
// still shows something meaningful.
func (s *LocationService) SendLocation(ctx context.Context, chatID, userID string, latitude, longitude float64, label string, replyToID *string) (*models.Message, error) {
	if err := validateCoordinates(latitude, longitude); err != nil {
		return nil, err
	}

	label = strings.TrimSpace(label)
	text := label
	if text == "" {
		text = "Shared a location"
	}
	// Cap the label so the VARCHAR(255) column and message text stay valid.
	const maxLabel = 255
	if len(label) > maxLabel {
		label = label[:maxLabel]
	}

	attachmentID := uuid.NewString()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	message, err := createMessageTx(ctx, tx, chatID, userID, text)
	if err != nil {
		return nil, err
	}
	if replyToID != nil && *replyToID != "" {
		if err := setReplyTo(ctx, tx, message.ID, *replyToID); err != nil {
			return nil, err
		}
		message.ReplyToID = replyToID
	}

	att := models.MediaAttachment{
		ID:           attachmentID,
		MessageID:    message.ID,
		ChatID:       chatID,
		Type:         typeLocation,
		FileName:     "location",
		FileSize:     0,
		MimeType:     locationMime,
		URL:          mapURL(latitude, longitude),
		Latitude:     &latitude,
		Longitude:    &longitude,
		LocationName: label,
		CreatedAt:    time.Now().UTC(),
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO media_attachments (id, message_id, type, file_name, file_size, mime_type, url, latitude, longitude, location_name)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, att.ID, att.MessageID, att.Type, att.FileName, att.FileSize, att.MimeType, att.URL, latitude, longitude, nullable(label)); err != nil {
		return nil, fmt.Errorf("failed to record location: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	message.Media = []models.MediaAttachment{att}
	return message, nil
}

// setReplyTo stamps the reply_to_id cursor on a freshly-created message when the
// sender chose to reply to another message.
func setReplyTo(ctx context.Context, tx *sql.Tx, messageID, replyToID string) error {
	if replyToID == "" {
		return nil
	}
	_, err := tx.ExecContext(ctx, `UPDATE messages SET reply_to_id = $1 WHERE id = $2`, replyToID, messageID)
	return err
}

// validateCoordinates rejects NaN/Inf and out-of-range lat/lng pairs so a
// malformed pin can never be persisted.
func validateCoordinates(latitude, longitude float64) error {
	if math.IsNaN(latitude) || math.IsInf(latitude, 0) || latitude < minLatitude || latitude > maxLatitude {
		return errors.New("latitude must be a finite number between -90 and 90")
	}
	if math.IsNaN(longitude) || math.IsInf(longitude, 0) || longitude < minLongitude || longitude > maxLongitude {
		return errors.New("longitude must be a finite number between -180 and 180")
	}
	return nil
}

// nullable returns the string as a *string, or nil when empty, so a location
// without a label stores NULL (not an empty string) in location_name.
func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// mapURL builds a client-facing link to the shared point using the keyless
// OpenStreetMap embed (self-host friendly — no API key, no quota). The format is
// a bbox around the point plus a marker so the URL renders both an embeddable map
// and a sharable destination page. Clients that render their own map tile layer
// can ignore this and use latitude/longitude directly.
func mapURL(latitude, longitude float64) string {
	const pad = 0.004
	return fmt.Sprintf(
		"https://www.openstreetmap.org/export/embed.html?bbox=%.6f,%.6f,%.6f,%.6f&layer=mapnik&marker=%.6f,%.6f",
		longitude-pad, latitude-pad, longitude+pad, latitude+pad, latitude, longitude,
	)
}
