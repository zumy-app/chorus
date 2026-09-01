package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/chorus/messenger/internal/models"
	"github.com/lib/pq"
)

// GalleryService provides the chat-scoped media gallery (task 6.5): the photos,
// videos, audio, documents and links shared in a specific chat. It always
// enforces participant scoping so a caller can only ever read media from chats
// they belong to, and it returns per-type counts so the client can badge the
// Media / Docs / Links tabs without extra round-trips.
type GalleryService struct {
	db *sql.DB
}

// NewGalleryService creates a GalleryService backed by the given DB.
func NewGalleryService(db *sql.DB) *GalleryService {
	return &GalleryService{db: db}
}

// galleryTypeSet normalizes a client-supplied type filter. It accepts a
// comma-separated list of concrete media types (image, video, audio, document,
// link) and also translates the friendly tab aliases so one query param drives
// all three gallery tabs: "media" → image,video, "docs" → document, "links" →
// link. Aliases may be mixed with concrete types (e.g. "media,link"). An empty
// input returns nil (no type filter → all types).
func galleryTypeSet(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	seen := map[string]bool{}
	var types []string
	add := func(ts ...string) {
		for _, t := range ts {
			if !seen[t] {
				seen[t] = true
				types = append(types, t)
			}
		}
	}
	for _, part := range strings.Split(raw, ",") {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "media":
			add("image", "video")
		case "docs":
			add("document")
		case "links":
			add("link")
		case "locations", "location":
			add("location")
		case "image", "video", "audio", "document", "link":
			add(strings.TrimSpace(part))
		}
	}
	return types
}

// GetChatGallery returns a page of the media gallery for a single chat. The
// typeFilter is optional; empty returns every type. Access is restricted to
// chat participants via the chat_participants join, mirroring the media search
// path so non-participants simply get an empty gallery.
func (s *GalleryService) GetChatGallery(ctx context.Context, userID, chatID, typeFilter string, limit, offset int) (*models.ChatMediaGallery, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	if offset < 0 {
		offset = 0
	}

	// Participant-scoped base: only items in chats the user belongs to, joined
	// to the sender so the "Shared by" info is available on each row.
	base := `
		FROM media_attachments ma
		JOIN messages m ON ma.message_id = m.id
		JOIN users su ON m.sender_id = su.id
		JOIN chat_participants cp ON m.chat_id = cp.chat_id
		WHERE cp.user_id = $1
		AND m.chat_id = $2
		AND m.deleted_at IS NULL
	`
	args := []interface{}{userID, chatID}
	argNum := 3

	if types := galleryTypeSet(typeFilter); len(types) > 0 {
		base += fmt.Sprintf(" AND ma.type = ANY($%d)", argNum)
		args = append(args, pq.Array(types))
		argNum++
	}

	counts, err := s.galleryCounts(ctx, userID, chatID)
	if err != nil {
		return nil, err
	}

	// Total matching this filter (separate from the per-type counts, which span
	// the whole chat).
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*)"+base, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count gallery items: %w", err)
	}

	if total == 0 {
		return &models.ChatMediaGallery{
			Items:   []models.MediaAttachment{},
			Total:   0,
			HasMore: false,
			Counts:  counts,
		}, nil
	}

	query := `
		SELECT ma.id, ma.message_id, ma.type, ma.file_name, ma.file_size,
		       ma.mime_type, ma.url, COALESCE(ma.thumbnail_url, ''),
		       ma.created_at, m.chat_id,
		       ma.latitude, ma.longitude, COALESCE(ma.location_name, ''),
		       COALESCE(su.id, ''), COALESCE(su.display_name, ''), COALESCE(su.username, '')
	` + base +
		fmt.Sprintf(" ORDER BY ma.created_at DESC LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to load gallery items: %w", err)
	}
	defer rows.Close()

	media := []models.MediaAttachment{}
	for rows.Next() {
		var att models.MediaAttachment
		var thumbnail sql.NullString
		var latitude, longitude sql.NullFloat64
		var senderID, senderName, senderUsername string

		if err := rows.Scan(
			&att.ID, &att.MessageID, &att.Type, &att.FileName, &att.FileSize,
			&att.MimeType, &att.URL, &thumbnail, &att.CreatedAt, &att.ChatID,
			&latitude, &longitude, &att.LocationName,
			&senderID, &senderName, &senderUsername,
		); err != nil {
			return nil, err
		}

		if thumbnail.Valid && thumbnail.String != "" {
			att.ThumbnailURL = &thumbnail.String
		}
		if latitude.Valid {
			att.Latitude = &latitude.Float64
		}
		if longitude.Valid {
			att.Longitude = &longitude.Float64
		}

		if senderID != "" || senderName != "" || senderUsername != "" {
			att.Sender = &models.User{
				ID:          senderID,
				DisplayName: senderName,
				Username:    senderUsername,
			}
		}

		media = append(media, att)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &models.ChatMediaGallery{
		Items:   media,
		Total:   total,
		HasMore: offset+len(media) < total,
		Counts:  counts,
	}, nil
}

// galleryCounts sums media rows by type across the whole chat (regardless of
// the current filter) so the client can render tab badges.
func (s *GalleryService) galleryCounts(ctx context.Context, userID, chatID string) (models.GalleryTypeCounts, error) {
	var counts models.GalleryTypeCounts

	rows, err := s.db.QueryContext(ctx, `
		SELECT ma.type, COUNT(*)
		FROM media_attachments ma
		JOIN messages m ON ma.message_id = m.id
		JOIN chat_participants cp ON m.chat_id = cp.chat_id
		WHERE cp.user_id = $1
		AND m.chat_id = $2
		AND m.deleted_at IS NULL
		GROUP BY ma.type
	`, userID, chatID)
	if err != nil {
		return counts, fmt.Errorf("failed to count gallery types: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var typ string
		var n int
		if err := rows.Scan(&typ, &n); err != nil {
			return counts, err
		}
		switch typ {
		case "image":
			counts.Image = n
		case "video":
			counts.Video = n
		case "audio":
			counts.Audio = n
		case "document":
			counts.Document = n
		case "link":
			counts.Link = n
		case "location":
			counts.Location = n
		}
	}

	return counts, rows.Err()
}
