package services

import (
	"database/sql"

	"github.com/chorus/messenger/internal/models"
)

type PrivacyService struct {
	db *sql.DB
}

func NewPrivacyService(db *sql.DB) *PrivacyService {
	return &PrivacyService{db: db}
}

func (s *PrivacyService) AreContacts(userA, userB string) bool {
	if userA == "" || userB == "" || userA == userB {
		return false
	}
	var exists bool
	err := s.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM chat_participants cp1
			JOIN chat_participants cp2 ON cp1.chat_id = cp2.chat_id
			JOIN chats c ON c.id = cp1.chat_id
			WHERE cp1.user_id = $1 AND cp2.user_id = $2 AND c.type = 'direct'
		)`, userA, userB).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

func (s *PrivacyService) fetchVisibility(userID string, field string) models.PrivacyVisibility {
	var vis string
	var q string
	switch field {
	case "last_seen":
		q = `SELECT last_seen_visibility FROM user_settings WHERE user_id = $1`
	case "profile_photo":
		q = `SELECT profile_photo_visibility FROM user_settings WHERE user_id = $1`
	case "contacts":
		q = `SELECT contacts_visibility FROM user_settings WHERE user_id = $1`
	default:
		return models.PrivacyEveryone
	}
	if err := s.db.QueryRow(q, userID).Scan(&vis); err != nil {
		if err == sql.ErrNoRows {
			return models.PrivacyEveryone
		}
		return models.PrivacyEveryone
	}
	switch models.PrivacyVisibility(vis) {
	case models.PrivacyEveryone, models.PrivacyContacts, models.PrivacyNobody:
		return models.PrivacyVisibility(vis)
	default:
		return models.PrivacyEveryone
	}
}

func (s *PrivacyService) CanView(viewerID, ownerID string, vis models.PrivacyVisibility) bool {
	if viewerID == ownerID {
		return true
	}
	switch vis {
	case models.PrivacyEveryone:
		return true
	case models.PrivacyNobody:
		return false
	case models.PrivacyContacts:
		return s.AreContacts(viewerID, ownerID)
	default:
		return true
	}
}

func (s *PrivacyService) CanViewLastSeen(viewerID, ownerID string) bool {
	return s.CanView(viewerID, ownerID, s.fetchVisibility(ownerID, "last_seen"))
}

func (s *PrivacyService) CanViewProfilePhoto(viewerID, ownerID string) bool {
	return s.CanView(viewerID, ownerID, s.fetchVisibility(ownerID, "profile_photo"))
}

func (s *PrivacyService) CanViewContacts(viewerID, ownerID string) bool {
	return s.CanView(viewerID, ownerID, s.fetchVisibility(ownerID, "contacts"))
}

func (s *PrivacyService) FilterUser(viewerID string, u *models.User) {
	if u == nil {
		return
	}
	if !s.CanViewProfilePhoto(viewerID, u.ID) {
		u.AvatarURL = nil
	}
	if !s.CanViewLastSeen(viewerID, u.ID) {
		u.LastActiveAt = u.CreatedAt
	}
}
