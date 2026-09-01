package services

import (
	"database/sql"
	"fmt"

	"github.com/chorus/messenger/internal/models"
)

// SettingsService owns the per-account user_settings row, including the FR-25
// feature toggles (translation_enabled, grammar_auto, highlights_enabled). It
// lazy-creates a default row the first time a user asks for their settings so
// the toggle endpoints never 404, and it exposes the toggles to callers that
// need to gate server-side work (translation gating on message send).
type SettingsService struct {
	db *sql.DB
}

func NewSettingsService(db *sql.DB) *SettingsService {
	return &SettingsService{db: db}
}

const settingsColumns = `user_id, grammar_enabled, vocabulary_enabled, difficulty_level,
	transcript_recording, message_retention_days,
	translation_enabled, grammar_auto, highlights_enabled,
	last_seen_visibility, profile_photo_visibility, contacts_visibility, updated_at`

// GetSettings returns the user's settings row, creating a default (all features
// enabled) row on first access.
func (s *SettingsService) GetSettings(userID string) (*models.UserSettings, error) {
	if err := s.ensureRow(userID); err != nil {
		return nil, err
	}
	return s.fetchSettings(userID)
}

// GetFeatureSettings returns only the FR-25 toggles. Used by handlers that must
// gate server-side work (e.g. auto-translation on message send) on the sender's
// feature flags.
func (s *SettingsService) GetFeatureSettings(userID string) (*models.FeatureSettings, error) {
	settings, err := s.GetSettings(userID)
	if err != nil {
		return nil, err
	}
	return &models.FeatureSettings{
		TranslationEnabled: settings.TranslationEnabled,
		GrammarAuto:        settings.GrammarAuto,
		HighlightsEnabled:  settings.HighlightsEnabled,
	}, nil
}

// UpdateFeatureSettings applies a partial update to the FR-25 toggles plus the
// 7.3 privacy visibilities. Omitted fields (nil pointers) are left unchanged.
// The full settings row is returned so the client can re-render from the
// authoritative state.
func (s *SettingsService) UpdateFeatureSettings(userID string, req models.UpdateFeatureSettingsRequest) (*models.UserSettings, error) {
	if err := s.ensureRow(userID); err != nil {
		return nil, err
	}
	_, err := s.db.Exec(`
		UPDATE user_settings
		SET translation_enabled = COALESCE($2, translation_enabled),
		    grammar_auto = COALESCE($3, grammar_auto),
		    highlights_enabled = COALESCE($4, highlights_enabled),
		    last_seen_visibility = COALESCE($5, last_seen_visibility),
		    profile_photo_visibility = COALESCE($6, profile_photo_visibility),
		    contacts_visibility = COALESCE($7, contacts_visibility),
		    updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1`, userID, req.TranslationEnabled, req.GrammarAuto, req.HighlightsEnabled, req.LastSeenVisibility, req.ProfilePhotoVisibility, req.ContactsVisibility)
	if err != nil {
		return nil, err
	}
	return s.fetchSettings(userID)
}

// ensureRow lazily creates the default settings row for a user. It is a no-op
// when the row already exists, so it is safe to call on every read/update.
func (s *SettingsService) ensureRow(userID string) error {
	if userID == "" {
		return fmt.Errorf("user id is required")
	}
	_, err := s.db.Exec(`INSERT INTO user_settings (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`, userID)
	return err
}

func (s *SettingsService) fetchSettings(userID string) (*models.UserSettings, error) {
	var st models.UserSettings
	err := s.db.QueryRow(`SELECT `+settingsColumns+` FROM user_settings WHERE user_id = $1`, userID).
		Scan(
			&st.UserID,
			&st.GrammarEnabled,
			&st.VocabularyEnabled,
			&st.DifficultyLevel,
			&st.TranscriptRecording,
			&st.MessageRetentionDays,
			&st.TranslationEnabled,
			&st.GrammarAuto,
			&st.HighlightsEnabled,
			&st.LastSeenVisibility,
			&st.ProfilePhotoVisibility,
			&st.ContactsVisibility,
			&st.UpdatedAt,
		)
	if err != nil {
		return nil, err
	}
	return &st, nil
}
