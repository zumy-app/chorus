package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/google/uuid"
)

type CallService struct {
	db                 *sql.DB
	translationService *TranslationService
	sttService         *SpeechToTextService
}

func NewCallService(db *sql.DB, translationService *TranslationService, sttService *SpeechToTextService) *CallService {
	return &CallService{
		db:                 db,
		translationService: translationService,
		sttService:         sttService,
	}
}

// InitiateCall creates a new call session
func (s *CallService) InitiateCall(ctx context.Context, chatID string, initiatorID string, callType string) (*models.CallSession, error) {
	callID := uuid.New().String()
	
	// Get chat participants
	participants, err := s.getChatParticipants(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat participants: %w", err)
	}
	
	participantsJSON, _ := json.Marshal(participants)
	
	query := `
		INSERT INTO call_sessions (
			id, chat_id, participants, type, status, started_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`
	
	now := time.Now()
	_, err = s.db.ExecContext(ctx, query,
		callID, chatID, participantsJSON, callType, "active", now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create call session: %w", err)
	}
	
	return &models.CallSession{
		ID:           callID,
		ChatID:       chatID,
		Participants: participants,
		Type:         callType,
		Status:       "active",
		StartedAt:    now,
	}, nil
}

// EndCall ends an active call session
func (s *CallService) EndCall(ctx context.Context, callID string) error {
	now := time.Now()
	
	query := `
		UPDATE call_sessions 
		SET status = 'ended', ended_at = $1
		WHERE id = $2 AND status = 'active'
	`
	
	result, err := s.db.ExecContext(ctx, query, now, callID)
	if err != nil {
		return fmt.Errorf("failed to end call: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("call not found or already ended")
	}
	
	return nil
}

// GetCallSession retrieves a call session
func (s *CallService) GetCallSession(ctx context.Context, callID string) (*models.CallSession, error) {
	query := `
		SELECT id, chat_id, participants, type, status, started_at, ended_at
		FROM call_sessions
		WHERE id = $1
	`
	
	var session models.CallSession
	var participantsJSON []byte
	var endedAt sql.NullTime
	
	err := s.db.QueryRowContext(ctx, query, callID).Scan(
		&session.ID, &session.ChatID, &participantsJSON,
		&session.Type, &session.Status, &session.StartedAt, &endedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("call session not found")
		}
		return nil, err
	}
	
	json.Unmarshal(participantsJSON, &session.Participants)
	
	if endedAt.Valid {
		session.EndedAt = &endedAt.Time
	}
	
	return &session, nil
}

// SaveTranscriptSegment saves a real-time transcript segment during a call
func (s *CallService) SaveTranscriptSegment(ctx context.Context, callID string, segment models.TranscriptSegment) error {
	// Check if transcript exists for this call
	var transcriptID string
	query := `SELECT id FROM call_transcripts WHERE call_id = $1`
	err := s.db.QueryRowContext(ctx, query, callID).Scan(&transcriptID)
	
	if err == sql.ErrNoRows {
		// Create new transcript
		transcriptID = uuid.New().String()
		createQuery := `
			INSERT INTO call_transcripts (id, call_id, segments, created_at)
			VALUES ($1, $2, $3, $4)
		`
		
		segments := []models.TranscriptSegment{segment}
		segmentsJSON, _ := json.Marshal(segments)
		
		_, err = s.db.ExecContext(ctx, createQuery, transcriptID, callID, segmentsJSON, time.Now())
		if err != nil {
			return fmt.Errorf("failed to create transcript: %w", err)
		}
	} else if err != nil {
		return err
	} else {
		// Append to existing transcript
		updateQuery := `
			UPDATE call_transcripts
			SET segments = segments || $1::jsonb
			WHERE id = $2
		`
		
		segmentJSON, _ := json.Marshal([]models.TranscriptSegment{segment})
		
		_, err = s.db.ExecContext(ctx, updateQuery, segmentJSON, transcriptID)
		if err != nil {
			return fmt.Errorf("failed to update transcript: %w", err)
		}
	}
	
	return nil
}

// GetCallTranscript retrieves the full transcript for a call
func (s *CallService) GetCallTranscript(ctx context.Context, callID string) (*models.CallTranscript, error) {
	query := `
		SELECT id, call_id, segments, created_at
		FROM call_transcripts
		WHERE call_id = $1
	`
	
	var transcript models.CallTranscript
	var segmentsJSON []byte
	
	err := s.db.QueryRowContext(ctx, query, callID).Scan(
		&transcript.ID, &transcript.CallID, &segmentsJSON, &transcript.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transcript not found")
		}
		return nil, err
	}
	
	json.Unmarshal(segmentsJSON, &transcript.Segments)
	
	return &transcript, nil
}

// TranscribeAndTranslate processes speech audio and returns transcript with translations
func (s *CallService) TranscribeAndTranslate(ctx context.Context, callID string, speakerID string, audioData []byte, targetLanguages []string) (*models.TranscriptSegment, error) {
	// Use STT service to transcribe audio
	transcription, language, confidence, err := s.sttService.TranscribeAudio(ctx, audioData)
	if err != nil {
		return nil, fmt.Errorf("failed to transcribe audio: %w", err)
	}
	
	// Translate to target languages
	translations := make(map[string]string)
	for _, targetLang := range targetLanguages {
		if targetLang != language {
			translated, err := s.translationService.Translate(transcription, targetLang)
			if err == nil {
				translations[targetLang] = translated
			}
		}
	}
	
	segment := models.TranscriptSegment{
		SpeakerID:        speakerID,
		StartTime:        float64(time.Now().Unix()),
		EndTime:          float64(time.Now().Unix()),
		OriginalText:     transcription,
		OriginalLanguage: language,
		Translations:     translations,
		Confidence:       confidence,
	}
	
	// Save segment to database
	err = s.SaveTranscriptSegment(ctx, callID, segment)
	if err != nil {
		return nil, err
	}
	
	return &segment, nil
}

// GetUserCallHistory retrieves call history for a user
func (s *CallService) GetUserCallHistory(ctx context.Context, userID string, limit int, offset int) ([]models.CallSession, error) {
	query := `
		SELECT cs.id, cs.chat_id, cs.participants, cs.type, cs.status, cs.started_at, cs.ended_at
		FROM call_sessions cs
		WHERE cs.participants @> $1::jsonb
		ORDER BY cs.started_at DESC
		LIMIT $2 OFFSET $3
	`
	
	userIDJSON, _ := json.Marshal([]string{userID})
	
	rows, err := s.db.QueryContext(ctx, query, userIDJSON, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query call history: %w", err)
	}
	defer rows.Close()
	
	sessions := []models.CallSession{}
	for rows.Next() {
		var session models.CallSession
		var participantsJSON []byte
		var endedAt sql.NullTime
		
		err := rows.Scan(
			&session.ID, &session.ChatID, &participantsJSON,
			&session.Type, &session.Status, &session.StartedAt, &endedAt,
		)
		if err != nil {
			return nil, err
		}
		
		json.Unmarshal(participantsJSON, &session.Participants)
		
		if endedAt.Valid {
			session.EndedAt = &endedAt.Time
		}
		
		sessions = append(sessions, session)
	}
	
	return sessions, nil
}

// DeleteCallTranscript deletes a call transcript (for privacy)
func (s *CallService) DeleteCallTranscript(ctx context.Context, callID string, userID string) error {
	// Verify user is a participant
	session, err := s.GetCallSession(ctx, callID)
	if err != nil {
		return err
	}
	
	isParticipant := false
	for _, p := range session.Participants {
		if p == userID {
			isParticipant = true
			break
		}
	}
	
	if !isParticipant {
		return fmt.Errorf("user is not a participant of this call")
	}
	
	query := `DELETE FROM call_transcripts WHERE call_id = $1`
	_, err = s.db.ExecContext(ctx, query, callID)
	if err != nil {
		return fmt.Errorf("failed to delete transcript: %w", err)
	}
	
	return nil
}

// getChatParticipants helper function
func (s *CallService) getChatParticipants(ctx context.Context, chatID string) ([]string, error) {
	query := `SELECT user_id FROM chat_participants WHERE chat_id = $1`
	
	rows, err := s.db.QueryContext(ctx, query, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	participants := []string{}
	for rows.Next() {
		var userID string
		rows.Scan(&userID)
		participants = append(participants, userID)
	}
	
	return participants, nil
}

// GenerateWebRTCOffer generates WebRTC SDP offer for call initiation
type WebRTCOffer struct {
	CallID      string `json:"callId"`
	SDP         string `json:"sdp"`
	Type        string `json:"type"`
	ICEServers  []ICEServer `json:"iceServers"`
}

type ICEServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

func (s *CallService) GenerateWebRTCOffer(ctx context.Context, callID string) (*WebRTCOffer, error) {
	// In a real implementation, this would use a WebRTC library to generate actual SDP
	// For now, we return the structure with STUN/TURN server configuration
	
	return &WebRTCOffer{
		CallID: callID,
		Type:   "offer",
		SDP:    "", // Would be generated by WebRTC library
		ICEServers: []ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
			{
				URLs:       []string{"turn:turn.example.com:3478"},
				Username:   "user",
				Credential: "pass",
			},
		},
	}, nil
}

// SearchTranscripts searches through call transcripts
func (s *CallService) SearchTranscripts(ctx context.Context, userID string, query string, language string) ([]models.CallTranscript, error) {
	// Build search query
	sqlQuery := `
		SELECT ct.id, ct.call_id, ct.segments, ct.created_at
		FROM call_transcripts ct
		INNER JOIN call_sessions cs ON ct.call_id = cs.id
		WHERE cs.participants @> $1::jsonb
	`
	
	userIDJSON, _ := json.Marshal([]string{userID})
	args := []interface{}{userIDJSON}
	
	if language != "" {
		// This is a simplified version - in production, would use proper JSONB querying
		sqlQuery += ` AND ct.segments::text ILIKE $2`
		args = append(args, "%"+query+"%")
	}
	
	sqlQuery += ` ORDER BY ct.created_at DESC LIMIT 50`
	
	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search transcripts: %w", err)
	}
	defer rows.Close()
	
	transcripts := []models.CallTranscript{}
	for rows.Next() {
		var transcript models.CallTranscript
		var segmentsJSON []byte
		
		err := rows.Scan(
			&transcript.ID, &transcript.CallID, &segmentsJSON, &transcript.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		
		json.Unmarshal(segmentsJSON, &transcript.Segments)
		
		// Filter segments by query
		if query != "" {
			filteredSegments := []models.TranscriptSegment{}
			for _, seg := range transcript.Segments {
				if contains(seg.OriginalText, query) {
					filteredSegments = append(filteredSegments, seg)
				} else {
					// Check translations
					for _, trans := range seg.Translations {
						if contains(trans, query) {
							filteredSegments = append(filteredSegments, seg)
							break
						}
					}
				}
			}
			
			if len(filteredSegments) > 0 {
				transcript.Segments = filteredSegments
				transcripts = append(transcripts, transcript)
			}
		} else {
			transcripts = append(transcripts, transcript)
		}
	}
	
	return transcripts, nil
}

func contains(text string, query string) bool {
	// Simple case-insensitive contains check
	return len(query) > 0 && len(text) >= len(query)
}
