package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/google/uuid"
)

var (
	ErrCallNotFound     = errors.New("call session not found")
	ErrCallAlreadyEnded = errors.New("call already ended")
	ErrNotParticipant   = errors.New("user is not a participant of this call")
	ErrChatNotFound     = errors.New("chat not found")
	ErrInvalidSignal    = errors.New("invalid signal type")
)

var validSignalTypes = map[string]bool{
	"offer":             true,
	"answer":            true,
	"ice-candidate":     true,
	"ice_candidate":     true,
	"screen-share-start": true,
	"screen-share-stop": true,
	"video-toggle":      true,
}

type CallService struct {
	db                 *sql.DB
	translationService *TranslationService
	sttService         *SpeechToTextService
	hub                *WebSocketHub
	pubsub             *PubSubService
}

func NewCallService(db *sql.DB, translationService *TranslationService, sttService *SpeechToTextService) *CallService {
	return &CallService{
		db:                 db,
		translationService: translationService,
		sttService:         sttService,
	}
}

func (s *CallService) SetHub(hub *WebSocketHub) { s.hub = hub }
func (s *CallService) SetPubSub(pubsub *PubSubService) { s.pubsub = pubsub }

func (s *CallService) InitiateCall(ctx context.Context, chatID string, initiatorID string, callType string) (*models.CallSession, error) {
	if callType != "audio" && callType != "video" {
		callType = "audio"
	}
	participants, err := s.getChatParticipants(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat participants: %w", err)
	}
	if len(participants) == 0 {
		return nil, ErrChatNotFound
	}
	isMember := false
	for _, p := range participants {
		if p == initiatorID {
			isMember = true
			break
		}
	}
	if !isMember {
		return nil, ErrNotParticipant
	}
	callID := uuid.New().String()
	participantsJSON, _ := json.Marshal(participants)
	now := time.Now()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO call_sessions (id, chat_id, participants, initiator_id, type, status, started_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, callID, chatID, participantsJSON, initiatorID, callType, "active", now)
	if err != nil {
		if strings.Contains(err.Error(), "column \"participants\"") || strings.Contains(err.Error(), "initiator_id") {
			_, err2 := s.db.ExecContext(ctx, `
				INSERT INTO call_sessions (id, chat_id, type, status, started_at)
				VALUES ($1, $2, $3, $4, $5)
			`, callID, chatID, callType, "active", now)
			if err2 != nil {
				return nil, fmt.Errorf("failed to create call session: %w", err2)
			}
		} else {
			return nil, fmt.Errorf("failed to create call session: %w", err)
		}
	}
	for _, pid := range participants {
		_, _ = s.db.ExecContext(ctx, `INSERT INTO call_participants (call_id, user_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, callID, pid)
	}
	session := &models.CallSession{
		ID:           callID,
		ChatID:       chatID,
		Participants: participants,
		Type:         callType,
		Status:       "active",
		StartedAt:    now,
	}
	s.notifyParticipants(callID, chatID, participants, initiatorID, "call_incoming", map[string]interface{}{
		"callId":      callID,
		"chatId":      chatID,
		"type":        callType,
		"initiatorId": initiatorID,
		"participants": participants,
	})
	return session, nil
}

func (s *CallService) EndCall(ctx context.Context, callID string) error {
	return s.EndCallAs(ctx, callID, "")
}

func (s *CallService) EndCallAs(ctx context.Context, callID string, userID string) error {
	session, err := s.GetCallSession(ctx, callID)
	if err != nil {
		return err
	}
	if userID != "" {
		isPart := false
		for _, p := range session.Participants {
			if p == userID {
				isPart = true
				break
			}
		}
		if !isPart {
			return ErrNotParticipant
		}
	}
	if session.Status == "ended" {
		return ErrCallAlreadyEnded
	}
	now := time.Now()
	result, err := s.db.ExecContext(ctx, `UPDATE call_sessions SET status='ended', ended_at=$1 WHERE id=$2 AND status='active'`, now, callID)
	if err != nil {
		return fmt.Errorf("failed to end call: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrCallAlreadyEnded
	}
	s.notifyParticipants(callID, session.ChatID, session.Participants, userID, "call_ended", map[string]interface{}{
		"callId":  callID,
		"chatId":  session.ChatID,
		"endedBy": userID,
		"endedAt": now,
	})
	return nil
}

func (s *CallService) GetCallSession(ctx context.Context, callID string) (*models.CallSession, error) {
	var session models.CallSession
	var participantsJSON []byte
	var endedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `SELECT id, chat_id, participants, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`, callID).Scan(
		&session.ID, &session.ChatID, &participantsJSON, &session.Type, &session.Status, &session.StartedAt, &endedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrCallNotFound
		}
		if strings.Contains(err.Error(), "column \"participants\"") {
			err2 := s.db.QueryRowContext(ctx, `SELECT id, chat_id, type, status, started_at, ended_at FROM call_sessions WHERE id=$1`, callID).Scan(
				&session.ID, &session.ChatID, &session.Type, &session.Status, &session.StartedAt, &endedAt,
			)
			if err2 != nil {
				if err2 == sql.ErrNoRows {
					return nil, ErrCallNotFound
				}
				return nil, err2
			}
			participants, _ := s.getCallParticipants(ctx, callID)
			session.Participants = participants
			if endedAt.Valid {
				session.EndedAt = &endedAt.Time
			}
			return &session, nil
		}
		return nil, err
	}
	_ = json.Unmarshal(participantsJSON, &session.Participants)
	if len(session.Participants) == 0 {
		participants, _ := s.getCallParticipants(ctx, callID)
		if len(participants) > 0 {
			session.Participants = participants
		}
	}
	if endedAt.Valid {
		session.EndedAt = &endedAt.Time
	}
	return &session, nil
}

func (s *CallService) getCallParticipants(ctx context.Context, callID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT user_id FROM call_participants WHERE call_id=$1`, callID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var uid string
		_ = rows.Scan(&uid)
		out = append(out, uid)
	}
	return out, nil
}

func (s *CallService) SaveTranscriptSegment(ctx context.Context, callID string, segment models.TranscriptSegment) error {
	var transcriptID string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM call_transcripts WHERE call_id=$1`, callID).Scan(&transcriptID)
	if err == sql.ErrNoRows {
		transcriptID = uuid.New().String()
		segments := []models.TranscriptSegment{segment}
		segmentsJSON, _ := json.Marshal(segments)
		_, err = s.db.ExecContext(ctx, `INSERT INTO call_transcripts (id, call_id, segments, created_at) VALUES ($1,$2,$3,$4)`, transcriptID, callID, segmentsJSON, time.Now())
		if err != nil {
			return fmt.Errorf("failed to create transcript: %w", err)
		}
	} else if err != nil {
		return err
	} else {
		segmentJSON, _ := json.Marshal([]models.TranscriptSegment{segment})
		_, err = s.db.ExecContext(ctx, `UPDATE call_transcripts SET segments = segments || $1::jsonb WHERE id=$2`, string(segmentJSON), transcriptID)
		if err != nil {
			return fmt.Errorf("failed to update transcript: %w", err)
		}
	}
	return nil
}

func (s *CallService) GetCallTranscript(ctx context.Context, callID string) (*models.CallTranscript, error) {
	var transcript models.CallTranscript
	var segmentsJSON []byte
	err := s.db.QueryRowContext(ctx, `SELECT id, call_id, segments, created_at FROM call_transcripts WHERE call_id=$1`, callID).Scan(
		&transcript.ID, &transcript.CallID, &segmentsJSON, &transcript.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transcript not found")
		}
		return nil, err
	}
	_ = json.Unmarshal(segmentsJSON, &transcript.Segments)
	if transcript.Segments == nil {
		transcript.Segments = []models.TranscriptSegment{}
	}
	return &transcript, nil
}

func (s *CallService) TranscribeAndTranslate(ctx context.Context, callID string, speakerID string, audioData []byte, targetLanguages []string) (*models.TranscriptSegment, error) {
	transcription, language, confidence, err := s.sttService.TranscribeAudio(ctx, audioData)
	if err != nil {
		return nil, fmt.Errorf("failed to transcribe audio: %w", err)
	}
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
	if err := s.SaveTranscriptSegment(ctx, callID, segment); err != nil {
		return nil, err
	}
	return &segment, nil
}

func (s *CallService) GetUserCallHistory(ctx context.Context, userID string, limit int, offset int) ([]models.CallSession, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT cs.id, cs.chat_id, cs.participants, cs.type, cs.status, cs.started_at, cs.ended_at
		FROM call_sessions cs
		WHERE cs.participants @> $1::jsonb
		ORDER BY cs.started_at DESC LIMIT $2 OFFSET $3
	`, fmt.Sprintf(`["%s"]`, userID), limit, offset)
	if err != nil {
		if strings.Contains(err.Error(), "column \"participants\"") {
			return s.getUserCallHistoryFallback(ctx, userID, limit, offset)
		}
		if strings.Contains(err.Error(), "operator does not exist") {
			return s.getUserCallHistoryFallback(ctx, userID, limit, offset)
		}
		return nil, fmt.Errorf("failed to query call history: %w", err)
	}
	defer rows.Close()
	var sessions []models.CallSession
	for rows.Next() {
		var sess models.CallSession
		var participantsJSON []byte
		var endedAt sql.NullTime
		if err := rows.Scan(&sess.ID, &sess.ChatID, &participantsJSON, &sess.Type, &sess.Status, &sess.StartedAt, &endedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(participantsJSON, &sess.Participants)
		if endedAt.Valid {
			sess.EndedAt = &endedAt.Time
		}
		sessions = append(sessions, sess)
	}
	if sessions == nil {
		sessions = []models.CallSession{}
	}
	return sessions, nil
}

func (s *CallService) getUserCallHistoryFallback(ctx context.Context, userID string, limit, offset int) ([]models.CallSession, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT cs.id, cs.chat_id, cs.type, cs.status, cs.started_at, cs.ended_at
		FROM call_sessions cs
		INNER JOIN call_participants cp ON cp.call_id = cs.id AND cp.user_id=$1
		ORDER BY cs.started_at DESC LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query call history: %w", err)
	}
	defer rows.Close()
	var sessions []models.CallSession
	for rows.Next() {
		var sess models.CallSession
		var endedAt sql.NullTime
		if err := rows.Scan(&sess.ID, &sess.ChatID, &sess.Type, &sess.Status, &sess.StartedAt, &endedAt); err != nil {
			return nil, err
		}
		participants, _ := s.getCallParticipants(ctx, sess.ID)
		sess.Participants = participants
		if endedAt.Valid {
			sess.EndedAt = &endedAt.Time
		}
		sessions = append(sessions, sess)
	}
	if sessions == nil {
		sessions = []models.CallSession{}
	}
	return sessions, nil
}

func (s *CallService) DeleteCallTranscript(ctx context.Context, callID string, userID string) error {
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
		return ErrNotParticipant
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM call_transcripts WHERE call_id=$1`, callID)
	if err != nil {
		return fmt.Errorf("failed to delete transcript: %w", err)
	}
	return nil
}

func (s *CallService) getChatParticipants(ctx context.Context, chatID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT user_id FROM chat_participants WHERE chat_id=$1`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var participants []string
	for rows.Next() {
		var userID string
		_ = rows.Scan(&userID)
		participants = append(participants, userID)
	}
	return participants, nil
}

type WebRTCOffer struct {
	CallID     string      `json:"callId"`
	SDP        string      `json:"sdp"`
	Type       string      `json:"type"`
	ICEServers []ICEServer `json:"iceServers"`
}

type ICEServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

func (s *CallService) GenerateWebRTCOffer(ctx context.Context, callID string) (*WebRTCOffer, error) {
	return &WebRTCOffer{
		CallID: callID,
		Type:   "offer",
		SDP:    "",
		ICEServers: iceServers(),
	}, nil
}

func iceServers() []ICEServer {
	stun := os.Getenv("WEBRTC_STUN_URLS")
	if stun == "" {
		stun = "stun:stun.l.google.com:19302"
	}
	servers := []ICEServer{{URLs: strings.Split(stun, ",")}}
	if turnURLs := os.Getenv("WEBRTC_TURN_URLS"); turnURLs != "" {
		servers = append(servers, ICEServer{
			URLs:       strings.Split(turnURLs, ","),
			Username:   os.Getenv("WEBRTC_TURN_USERNAME"),
			Credential: os.Getenv("WEBRTC_TURN_CREDENTIAL"),
		})
	} else if os.Getenv("WEBRTC_TURN_URL") != "" {
		servers = append(servers, ICEServer{
			URLs:       []string{os.Getenv("WEBRTC_TURN_URL")},
			Username:   os.Getenv("WEBRTC_TURN_USERNAME"),
			Credential: os.Getenv("WEBRTC_TURN_CREDENTIAL"),
		})
	}
	return servers
}

func (s *CallService) HandleSignal(ctx context.Context, callID string, senderID string, signalType string, sdp string, candidate string, extra map[string]interface{}) error {
	signalType = strings.ToLower(strings.TrimSpace(signalType))
	if signalType == "ice_candidate" {
		signalType = "ice-candidate"
	}
	if signalType == "screen_share_start" {
		signalType = "screen-share-start"
	}
	if signalType == "screen_share_stop" {
		signalType = "screen-share-stop"
	}
	if !validSignalTypes[signalType] {
		return ErrInvalidSignal
	}
	session, err := s.GetCallSession(ctx, callID)
	if err != nil {
		return err
	}
	if session.Status != "active" {
		return ErrCallAlreadyEnded
	}
	isPart := false
	for _, p := range session.Participants {
		if p == senderID {
			isPart = true
			break
		}
	}
	if !isPart {
		return ErrNotParticipant
	}
	if signalType == "offer" || signalType == "answer" {
		if sdp == "" {
			if v, ok := extra["sdp"]; ok {
				if str, ok := v.(string); ok {
					sdp = str
				}
			}
		}
		if sdp == "" {
			return fmt.Errorf("sdp required for %s", signalType)
		}
	}
	if signalType == "ice-candidate" {
		if candidate == "" {
			if v, ok := extra["candidate"]; ok {
				if str, ok := v.(string); ok {
					candidate = str
				}
				if m, ok := v.(map[string]interface{}); ok {
					b, _ := json.Marshal(m)
					candidate = string(b)
				}
			}
		}
		if candidate == "" {
			return fmt.Errorf("candidate required for ice-candidate")
		}
	}
	payload := map[string]interface{}{
		"callId": callID,
		"chatId": session.ChatID,
		"from":   senderID,
		"type":   signalType,
	}
	if sdp != "" {
		payload["sdp"] = sdp
	}
	if candidate != "" {
		payload["candidate"] = candidate
	}
	for k, v := range extra {
		if _, exists := payload[k]; !exists {
			payload[k] = v
		}
	}
	for _, pid := range session.Participants {
		if pid == senderID {
			continue
		}
		if s.hub != nil {
			s.hub.SendToUser(pid, "webrtc_signal", payload)
		}
		if s.pubsub != nil {
			_ = s.pubsub.PublishToUser(pid, "webrtc_signal", payload)
		}
	}
	return nil
}

func (s *CallService) notifyParticipants(callID, chatID string, participants []string, excludeUserID string, eventType string, payload interface{}) {
	for _, pid := range participants {
		if excludeUserID != "" && pid == excludeUserID && eventType == "call_incoming" {
			continue
		}
		if s.hub != nil {
			s.hub.SendToUser(pid, eventType, payload)
		}
		if s.pubsub != nil {
			_ = s.pubsub.PublishToUser(pid, eventType, payload)
		}
	}
}

func (s *CallService) PublishLiveCaption(ctx context.Context, callID, speakerID, text, originalLanguage string) (*models.TranscriptSegment, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("text required")
	}
	if originalLanguage == "" {
		originalLanguage = "en"
	}
	session, err := s.GetCallSession(ctx, callID)
	if err != nil {
		return nil, err
	}
	if session.Status != "active" {
		return nil, ErrCallAlreadyEnded
	}
	isPart := false
	for _, p := range session.Participants {
		if p == speakerID {
			isPart = true
			break
		}
	}
	if !isPart {
		return nil, ErrNotParticipant
	}
	targetLangs := s.participantTargetLanguages(ctx, session.Participants, originalLanguage)
	translations := make(map[string]string)
	for _, tl := range targetLangs {
		if s.translationService != nil {
			if tr, err := s.translationService.Translate(text, tl); err == nil && tr != "" {
				translations[tl] = tr
			}
		}
	}
	now := float64(time.Now().UnixMilli()) / 1000
	segment := models.TranscriptSegment{
		SpeakerID:        speakerID,
		StartTime:        now,
		EndTime:          now,
		OriginalText:     text,
		OriginalLanguage: originalLanguage,
		Translations:     translations,
		Confidence:       1.0,
	}
	if err := s.SaveTranscriptSegment(ctx, callID, segment); err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"callId":    callID,
		"chatId":    session.ChatID,
		"segment":   segment,
		"speakerId": speakerID,
	}
	for _, pid := range session.Participants {
		if s.hub != nil {
			s.hub.SendToUser(pid, "live_caption", payload)
		}
		if s.pubsub != nil {
			_ = s.pubsub.PublishToUser(pid, "live_caption", payload)
		}
	}
	return &segment, nil
}

func (s *CallService) GetCaptionsPaginated(ctx context.Context, callID, userID string, limit, offset int) ([]models.TranscriptSegment, int, error) {
	session, err := s.GetCallSession(ctx, callID)
	if err != nil {
		return nil, 0, err
	}
	isPart := false
	for _, p := range session.Participants {
		if p == userID {
			isPart = true
			break
		}
	}
	if !isPart {
		return nil, 0, ErrNotParticipant
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	transcript, err := s.GetCallTranscript(ctx, callID)
	if err != nil {
		return []models.TranscriptSegment{}, 0, nil
	}
	total := len(transcript.Segments)
	if offset >= total {
		return []models.TranscriptSegment{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return transcript.Segments[offset:end], total, nil
}

func (s *CallService) BookmarkCaption(ctx context.Context, callID, userID string, segmentIndex int) (*models.VocabularyEntry, error) {
	session, err := s.GetCallSession(ctx, callID)
	if err != nil {
		return nil, err
	}
	isPart := false
	for _, p := range session.Participants {
		if p == userID {
			isPart = true
			break
		}
	}
	if !isPart {
		return nil, ErrNotParticipant
	}
	transcript, err := s.GetCallTranscript(ctx, callID)
	if err != nil {
		return nil, fmt.Errorf("transcript not found")
	}
	if segmentIndex < 0 || segmentIndex >= len(transcript.Segments) {
		return nil, fmt.Errorf("segment index out of range")
	}
	seg := transcript.Segments[segmentIndex]
	term := strings.TrimSpace(seg.OriginalText)
	if term == "" {
		return nil, fmt.Errorf("empty segment")
	}
	lang := seg.OriginalLanguage
	if lang == "" {
		lang = "en"
	}
	translation := ""
	for _, v := range seg.Translations {
		translation = v
		break
	}
	if translation == "" && s.translationService != nil {
		if tr, err := s.translationService.Translate(term, "en"); err == nil {
			translation = tr
		}
	}
	var nativeLang string
	_ = s.db.QueryRowContext(ctx, `SELECT native_language FROM users WHERE id=$1`, userID).Scan(&nativeLang)
	_ = nativeLang
	now := time.Now()
	var entry models.VocabularyEntry
	entry.LearningData = &models.LearningData{}
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO vocabulary (user_id, term, language, translation, definition, context_sentence, source_type, next_review)
		VALUES ($1,$2,$3,$4,$5,$6,'scenario', CURRENT_TIMESTAMP + INTERVAL '1 day')
		ON CONFLICT (user_id, term, language) DO UPDATE SET translation=EXCLUDED.translation
		RETURNING id, user_id, term, language, translation, definition, next_review, interval_days, created_at
	`, userID, term, lang, translation, "", term).Scan(
		&entry.ID, &entry.UserID, &entry.Term, &entry.Language, &entry.Translation, &entry.Definition,
		&entry.LearningData.NextReview, &entry.LearningData.Interval, &entry.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to bookmark caption: %w", err)
	}
	entry.Context.Sentence = term
	_ = now
	return &entry, nil
}

func (s *CallService) participantTargetLanguages(ctx context.Context, participants []string, originalLanguage string) []string {
	if len(participants) == 0 {
		return nil
	}
	query := `SELECT DISTINCT native_language FROM users WHERE id = ANY($1)`
	ids := "{" + strings.Join(participants, ",") + "}"
	rows, err := s.db.QueryContext(ctx, query, ids)
	if err != nil {
		return nil
	}
	defer rows.Close()
	seen := map[string]struct{}{}
	var out []string
	for rows.Next() {
		var lang string
		if err := rows.Scan(&lang); err == nil && lang != "" && lang != originalLanguage {
			if _, ok := seen[lang]; !ok {
				seen[lang] = struct{}{}
				out = append(out, lang)
			}
		}
	}
	return out
}

func (s *CallService) SearchTranscripts(ctx context.Context, userID string, query string, language string) ([]models.CallTranscript, error) {
	sqlQuery := `
		SELECT ct.id, ct.call_id, ct.segments, ct.created_at
		FROM call_transcripts ct
		INNER JOIN call_sessions cs ON ct.call_id = cs.id
		WHERE cs.participants @> $1::jsonb
	`
	args := []interface{}{fmt.Sprintf(`["%s"]`, userID)}
	needsFallback := false
	if language != "" {
		sqlQuery += ` AND ct.segments::text ILIKE $2`
		args = append(args, "%"+query+"%")
	}
	sqlQuery += ` ORDER BY ct.created_at DESC LIMIT 50`
	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		if strings.Contains(err.Error(), "column \"participants\"") {
			needsFallback = true
		} else {
			return nil, fmt.Errorf("failed to search transcripts: %w", err)
		}
	}
	if needsFallback {
		return s.searchTranscriptsFallback(ctx, userID, query)
	}
	defer rows.Close()
	return s.filterTranscripts(rows, query)
}

func (s *CallService) searchTranscriptsFallback(ctx context.Context, userID, query string) ([]models.CallTranscript, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT ct.id, ct.call_id, ct.segments, ct.created_at
		FROM call_transcripts ct
		INNER JOIN call_participants cp ON cp.call_id = ct.call_id AND cp.user_id=$1
		ORDER BY ct.created_at DESC LIMIT 50
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to search transcripts: %w", err)
	}
	defer rows.Close()
	return s.filterTranscripts(rows, query)
}

func (s *CallService) filterTranscripts(rows *sql.Rows, query string) ([]models.CallTranscript, error) {
	var transcripts []models.CallTranscript
	lowerQuery := strings.ToLower(query)
	for rows.Next() {
		var t models.CallTranscript
		var segmentsJSON []byte
		if err := rows.Scan(&t.ID, &t.CallID, &segmentsJSON, &t.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(segmentsJSON, &t.Segments)
		if t.Segments == nil {
			t.Segments = []models.TranscriptSegment{}
		}
		if query != "" {
			var filtered []models.TranscriptSegment
			for _, seg := range t.Segments {
				if strings.Contains(strings.ToLower(seg.OriginalText), lowerQuery) {
					filtered = append(filtered, seg)
					continue
				}
				for _, trans := range seg.Translations {
					if strings.Contains(strings.ToLower(trans), lowerQuery) {
						filtered = append(filtered, seg)
						break
					}
				}
			}
			if len(filtered) > 0 {
				t.Segments = filtered
				transcripts = append(transcripts, t)
			}
		} else {
			transcripts = append(transcripts, t)
		}
	}
	if transcripts == nil {
		transcripts = []models.CallTranscript{}
	}
	return transcripts, nil
}
