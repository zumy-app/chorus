package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/redis/go-redis/v9"
)

// ErrInvalidSessionMode is returned when StartSession gets a mode outside the
// DB CHECK set. Surfaced as HTTP 400 (not 500) by LearningHandler.
var ErrInvalidSessionMode = errors.New("invalid session mode")

// SessionCache is the transient Redis layer for active sessions (plan V3
// section B): the 15-item session payload compiles into Redis under a 1-hour
// TTL so mid-session polling never touches Postgres; completion flushes
// durable state and drops the cache.
type SessionCache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

// sessionSnapshot is the cached transient state of an in-flight session.
type sessionSnapshot struct {
	SessionID  string         `json:"sessionId"`
	UserID     string         `json:"userId"`
	Mode       string         `json:"mode"`
	Status     string         `json:"status"`
	TargetLang string         `json:"targetLanguage"`
	Planned    int            `json:"plannedItemCount"`
	Completed  int            `json:"completedItemCount"`
	Score      int            `json:"score"`
	XPAwarded  int            `json:"xpAwarded"`
	Items      []snapshotItem `json:"items"`
}

type snapshotItem struct {
	ID           string                 `json:"id"`
	Ordinal      int                    `json:"ordinal"`
	ItemType     string                 `json:"itemType"`
	ActivityType string                 `json:"activityType"`
	Status       string                 `json:"status"`
	Question     models.SessionQuestion `json:"question"`
}

// SessionComposerService builds daily/quick-drill/vocabulary practice sessions
// from due SRS cards, current-unit lesson steps, and recent grammar weaknesses,
// grades answers using the PracticeService, and books completed sessions into
// daily stats + activity events + readiness so the dashboard updates live.
type SessionComposerService struct {
	db         *sql.DB
	practice   *PracticeService
	curriculum *CurriculumService
	profiles   *LearningProfileService
	lessons    *LessonService
	grammar    *GrammarPointService
	cache      SessionCache
}

func NewSessionComposerService(db *sql.DB, practice *PracticeService, curriculum *CurriculumService, profiles *LearningProfileService, lessons *LessonService, grammar *GrammarPointService) *SessionComposerService {
	return &SessionComposerService{db: db, practice: practice, curriculum: curriculum, profiles: profiles, lessons: lessons, grammar: grammar}
}

// SetSessionCache wires the transient Redis layer (optional: nil disables it
// and every path falls back to Postgres).
func (s *SessionComposerService) SetSessionCache(cache SessionCache) {
	s.cache = cache
}

const sessionCacheTTL = time.Hour

// redisCacheAdapter adapts *redis.Client to the SessionCache/RealTalkCache
// interfaces (redis returns command objects rather than plain errors).
type redisCacheAdapter struct{ rdb *redis.Client }

// NewRedisCacheAdapter wires a *redis.Client into the cache interfaces.
func NewRedisCacheAdapter(rdb *redis.Client) *redisCacheAdapter {
	return &redisCacheAdapter{rdb: rdb}
}

func (a *redisCacheAdapter) Get(ctx context.Context, key string) (string, error) {
	return a.rdb.Get(ctx, key).Result()
}

func (a *redisCacheAdapter) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return a.rdb.Set(ctx, key, value, ttl).Err()
}

func (a *redisCacheAdapter) Del(ctx context.Context, keys ...string) error {
	return a.rdb.Del(ctx, keys...).Err()
}

func sessionCacheKey(sessionID string) string {
	return "learningsession:" + sessionID
}

// saveSnapshot caches the transient session state.
func (s *SessionComposerService) saveSnapshot(ctx context.Context, snap *sessionSnapshot) {
	if s.cache == nil || snap == nil {
		return
	}
	if data, err := json.Marshal(snap); err == nil {
		_ = s.cache.Set(ctx, sessionCacheKey(snap.SessionID), string(data), sessionCacheTTL)
	}
}

// loadSnapshot returns the cached snapshot when warm.
func (s *SessionComposerService) loadSnapshot(ctx context.Context, sessionID string) *sessionSnapshot {
	if s.cache == nil {
		return nil
	}
	cached, err := s.cache.Get(ctx, sessionCacheKey(sessionID))
	if err != nil || cached == "" {
		return nil
	}
	var snap sessionSnapshot
	if json.Unmarshal([]byte(cached), &snap) != nil {
		return nil
	}
	return &snap
}

func (s *SessionComposerService) dropSnapshot(ctx context.Context, sessionID string) {
	if s.cache == nil {
		return
	}
	_ = s.cache.Del(ctx, sessionCacheKey(sessionID))
}

// sessionComposedItem is a not-yet-persisted session item being assembled.
type sessionComposedItem struct {
	itemType     string
	activityType string
	vocabularyID string
	grammarID    string
	payload      any
}

// StartSession creates a learning session and its ordered items. Mode is one of
// daily, quick_drill, vocabulary, grammar, streak_recovery.
func (s *SessionComposerService) StartSession(ctx context.Context, userID string, req models.StartSessionRequest) (*models.StartSessionResponse, error) {
	var targetLang, nativeLang string
	targetLang = normalizeLang(req.TargetLanguage)
	nativeLang = normalizeLang(req.NativeLanguage)
	if nativeLang == "" {
		nativeLang = "en"
	}
	mode := req.Mode
	if mode == "" {
		mode = "daily"
	}
	// Reject unknown modes with a validation error instead of letting the
	// DB CHECK constraint turn them into opaque 500s (acceptance TC-LEARN-04).
	switch mode {
	case "daily", "quick_drill", "vocabulary", "lesson", "scenario", "grammar", "streak_recovery":
	default:
		return nil, fmt.Errorf("%w: %q (want one of daily, quick_drill, vocabulary, lesson, scenario, grammar, streak_recovery)", ErrInvalidSessionMode, mode)
	}

	profile, err := s.profiles.GetProfile(ctx, userID, targetLang, nativeLang)
	if err != nil {
		return nil, err
	}
	targetCount := profile.DailyGoalItems
	if targetCount <= 0 {
		targetCount = 10
	}

	var composed []sessionComposedItem

	// 1. Due SRS cards (up to 60%).
	maxReviews := int(math.Floor(float64(targetCount) * 0.6))
	if maxReviews < 1 {
		maxReviews = 1
	}
	dueCards, _ := s.practice.GetDueCards(ctx, userID, targetLang, maxReviews)
	usedVocab := map[string]bool{}
	for _, card := range dueCards {
		usedVocab[card.ID] = true
		stage := nextStageForCard(&card)
		template, q := s.practice.BuildVocabQuestion(&card, stage)
		composed = append(composed, sessionComposedItem{
			itemType: "vocabulary", activityType: template,
			vocabularyID: card.ID,
			payload:      vocabItemPayload(&card, q, stage),
		})
	}

	// 2. One lesson step (current unit), capped at 40% of the remainder.
	if mode == "daily" || mode == "vocabulary" {
		if step, lessonID, unitID := s.curriculum.NextLessonStep(ctx, userID, profile); step != nil {
			q := lessonStepQuestion(step)
			correct, promptType := stepAnswerInfo(step)
			composed = append(composed, sessionComposedItem{
				itemType: "lesson_step", activityType: step.Type,
				payload: map[string]any{
					"prompt":        q.Prompt,
					"promptType":    promptType,
					"correctAnswer": correct,
					"stepId":        step.ID,
					"lessonId":      lessonID,
					"unitId":        unitID,
					"activityType":  step.Type,
				},
			})
		}
	}

	// 3. Grammar drills from the SRS ledger (real items, not an acknowledgement
	// stub): due/unseen points first, up to 40% of the session.
	if (mode == "grammar" || mode == "daily") && s.grammar != nil {
		grammarQuota := int(math.Floor(float64(targetCount) * 0.4))
		if mode == "grammar" {
			grammarQuota = targetCount
		}
		if grammarQuota < 1 {
			grammarQuota = 1
		}
		drills, _ := s.grammar.GetDueItems(ctx, userID, targetLang, grammarQuota)
		for _, d := range drills {
			if len(composed) >= targetCount && mode != "grammar" {
				break
			}
			composed = append(composed, grammarDrillComposedItem(d))
		}
	}

	// 4. Fill remaining with new cards (recent accepted words) not already in
	// the session, so a learner sees fresh cards rather than duplicates.
	if len(composed) < targetCount {
		need := targetCount - len(composed)
		newCards, _ := s.practice.GetNewCards(ctx, userID, targetLang, need*2)
		for _, card := range newCards {
			if len(composed) >= targetCount {
				break
			}
			if usedVocab[card.ID] {
				continue
			}
			usedVocab[card.ID] = true
			template, q := s.practice.BuildVocabQuestion(&card, stageRecognition)
			composed = append(composed, sessionComposedItem{
				itemType: "vocabulary", activityType: template,
				vocabularyID: card.ID,
				payload:      vocabItemPayload(&card, q, stageRecognition),
			})
		}
	}

	// Interleave item types so the learner doesn't do all vocab then all grammar.
	composed = interleaveItems(composed)

	// Persist the session.
	var sessionID string
	var sourceUnitID, sourceLessonID any
	for _, it := range composed {
		if it.itemType == "lesson_step" && it.payload != nil {
			if p, ok := it.payload.(map[string]any); ok {
				if id, ok := p["lessonId"].(string); ok && id != "" {
					sourceLessonID = id
				}
				if id, ok := p["unitId"].(string); ok && id != "" {
					sourceUnitID = id
				}
			}
		}
	}
	payloadJSON, _ := json.Marshal(composed)
	_ = payloadJSON
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO learning_sessions (
			user_id, target_language, mode, status, source_unit_id, source_lesson_id,
			planned_item_count, completed_item_count, score, xp_awarded
		) VALUES ($1, $2, $3, 'in_progress', $4, $5, $6, 0, 0, 0)
		RETURNING id::text`, userID, targetLang, mode, sourceUnitID, sourceLessonID, len(composed)).Scan(&sessionID)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	questions := make([]models.SessionQuestion, 0, len(composed))
	for i, it := range composed {
		var itemID string
		payload, _ := json.Marshal(it.payload)
		err := s.db.QueryRowContext(ctx, `
			INSERT INTO learning_session_items (
				session_id, ordinal, item_type, vocabulary_id, grammar_point_id,
				lesson_step_id, payload, status
			) VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending')
			RETURNING id::text`,
			sessionID, i+1, it.itemType, nullStr(it.vocabularyID), nullStr(it.grammarID),
			lessonStepIDFromPayload(it.payload), string(payload)).Scan(&itemID)
		if err != nil {
			return nil, err
		}
		q := questionFromPayload(it.payload)
		q.ID = itemID
		q.ItemType = it.itemType
		q.ActivityType = it.activityType
		questions = append(questions, q)
	}

	session := &models.LearningSession{
		ID: sessionID, UserID: userID, TargetLanguage: targetLang, Mode: mode,
		Status: "in_progress", PlannedItemCount: len(composed), StartedAt: time.Now(),
	}

	// Compile the transient session state into the Redis layer: mid-session
	// polls and resume reads hit the cache instead of Postgres.
	snap := &sessionSnapshot{
		SessionID: sessionID, UserID: userID, Mode: mode, Status: "in_progress",
		TargetLang: targetLang, Planned: len(composed),
	}
	for i, q := range questions {
		snap.Items = append(snap.Items, snapshotItem{
			ID: q.ID, Ordinal: i + 1, ItemType: q.ItemType, ActivityType: q.ActivityType,
			Status: "pending", Question: q,
		})
	}
	s.saveSnapshot(ctx, snap)

	return &models.StartSessionResponse{Session: session, Items: questions}, nil
}

// GetSession returns a session with its item states (poll / resume). Serves
// from the Redis transient layer first; on a cache miss (cold, expired, or a
// Redis partition mid-session) it regenerates the operational state from
// Postgres so the learner never loses the run.
func (s *SessionComposerService) GetSession(ctx context.Context, userID, sessionID string) (*models.LearningSession, error) {
	if snap := s.loadSnapshot(ctx, sessionID); snap != nil && snap.UserID == userID {
		sess := &models.LearningSession{
			ID: snap.SessionID, UserID: snap.UserID, TargetLanguage: snap.TargetLang,
			Mode: snap.Mode, Status: snap.Status, PlannedItemCount: snap.Planned,
			CompletedItemCount: snap.Completed, Score: snap.Score, XPAwarded: snap.XPAwarded,
		}
		if sess.PlannedItemCount > 0 {
			sess.ProgressPct = sess.CompletedItemCount * 100 / sess.PlannedItemCount
		}
		return sess, nil
	}
	var sess models.LearningSession
	err := s.db.QueryRowContext(ctx, `
		SELECT id::text, user_id, target_language, mode, status,
		       COALESCE(source_unit_id::text,''), COALESCE(source_lesson_id::text,''),
		       planned_item_count, completed_item_count, score, xp_awarded, started_at
		FROM learning_sessions WHERE id = $1 AND user_id = $2`, sessionID, userID).Scan(
		&sess.ID, &sess.UserID, &sess.TargetLanguage, &sess.Mode, &sess.Status,
		&sess.SourceUnitID, &sess.SourceLessonID, &sess.PlannedItemCount,
		&sess.CompletedItemCount, &sess.Score, &sess.XPAwarded, &sess.StartedAt)
	if err != nil {
		return nil, err
	}
	if sess.PlannedItemCount > 0 {
		sess.ProgressPct = sess.CompletedItemCount * 100 / sess.PlannedItemCount
	}
	return &sess, nil
}

// AnswerItem grades a session item answer and advances the session. It returns
// the updated item + next pending item.
func (s *SessionComposerService) AnswerItem(ctx context.Context, userID, sessionID, itemID string, req models.AnswerSessionItemRequest) (*models.AnswerSessionItemResponse, error) {
	var (
		itemType, status, payloadJSON         string
		vocabularyID, grammarID, lessonStepID string
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT item_type, COALESCE(vocabulary_id::text,''), COALESCE(grammar_point_id::text,''),
		       COALESCE(lesson_step_id::text,''), COALESCE(payload::text,''), status
		FROM learning_session_items WHERE id = $1 AND session_id = $2`, itemID, sessionID).Scan(
		&itemType, &vocabularyID, &grammarID, &lessonStepID, &payloadJSON, &status)
	if err != nil {
		return nil, err
	}
	if status != "pending" {
		return nil, fmt.Errorf("item already answered")
	}

	var payload map[string]any
	_ = json.Unmarshal([]byte(payloadJSON), &payload)

	resp := &models.AnswerSessionItemResponse{Feedback: models.SessionFeedback{}}

	switch itemType {
	case "vocabulary":
		card, err := s.practice.loadForGrade(ctx, userID, vocabularyID)
		if err != nil {
			return nil, err
		}
		stage := intFromAny(payload["stage"])
		if stage == 0 {
			stage = stageRecognition
		}
		promptType := strFromAny(payload["promptType"])
		answerText := req.Answer.Text
		if req.Answer.Choice != "" {
			answerText = req.Answer.Choice
		}
		correctAnswer := strFromAny(payload["answer"])
		correct, quality := s.practice.GradeAnswer(correctAnswer, answerText, promptType)
		if err := s.practice.UpdateVocabAfterAttempt(ctx, card, stage, quality, strFromAny(payload["activityType"])); err != nil {
			return nil, err
		}
		_ = s.practice.RecordAttempt(ctx, card, stage, strFromAny(payload["activityType"]), payload["prompt"], answerText, correct, quality, req.LatencyMs, sessionID)
		resp.Correct = correct
		resp.Quality = quality
		resp.Feedback.Message = feedbackForQuality(correct, correctAnswer)
		resp.Feedback.CorrectAnswer = correctAnswer
		resp.Feedback.MasteryState = card.MasteryState

	case "lesson_step":
		stepCorrect, stepScore := s.gradeLessonStep(ctx, lessonStepID, payload, req.Answer)
		resp.Correct = stepCorrect
		resp.Quality = stepScore
		resp.Feedback.Message = stepFeedback(stepCorrect)
		resp.Feedback.CorrectAnswer = strFromAny(payload["correctAnswer"])

	case "grammar":
		// Real drill item (cloze/mcq/reconstruction): grade against the ledger-
		// tracked answer, record the attempt, and advance the SM-2 state.
		if grammarItemID := strFromAny(payload["grammarItemId"]); grammarItemID != "" && s.grammar != nil {
			answerText := req.Answer.Text
			if req.Answer.Choice != "" {
				answerText = req.Answer.Choice
			}
			drill := models.GrammarDrillItem{
				Correct: strFromAny(payload["answer"]),
			}
			correct := GradeDrillAnswer(drill, answerText, strFromAny(payload["cefrLevel"]))
			quality := 3
			if correct {
				quality = 5
				if req.LatencyMs > 8000 {
					quality = 4
				}
			} else {
				quality = 1
			}
			if err := s.grammar.RecordItemAttempt(ctx, userID, grammarItemID, correct, quality); err != nil {
				log.Printf("[Session] grammar attempt %s: %v", grammarItemID, err)
			}
			resp.Correct = correct
			resp.Quality = quality
			resp.Feedback.Message = grammarDrillFeedback(correct, strFromAny(payload["note"]))
			resp.Feedback.CorrectAnswer = strFromAny(payload["answer"])
		} else {
			// Legacy acknowledgement review: record the grammar point as reviewed.
			s.touchGrammarMastery(ctx, userID, targetLangOfSession(ctx, s.db, sessionID), grammarID)
			resp.Correct = true
			resp.Quality = 4
			resp.Feedback.Message = "Grammar point reviewed. Keep it fresh in your next conversations."
		}
	default:
		return nil, fmt.Errorf("unknown item type %q", itemType)
	}

	// Mark the item answered; store the graded result for resume/polling.
	resultJSON, _ := json.Marshal(map[string]any{
		"correct": resp.Correct, "quality": resp.Quality, "score": scaleScore(resp.Quality),
		"userAnswer": req.Answer.Text,
	})
	_, _ = s.db.ExecContext(ctx, `UPDATE learning_session_items SET status = 'answered', result = $1 WHERE id = $2`, string(resultJSON), itemID)

	// Bump session counters.
	_, _ = s.db.ExecContext(ctx, `
		UPDATE learning_sessions SET completed_item_count = completed_item_count + 1,
			score = score + $3
		WHERE id = $1 AND user_id = $2`, sessionID, userID, scaleScore(resp.Quality))

	// Sync the transient Redis state so mid-session polls stay off Postgres.
	s.syncSnapshot(ctx, sessionID, itemID, scaleScore(resp.Quality))

	next := s.nextPendingItem(ctx, sessionID, itemID)
	resp.NextItem = next
	if next != nil {
		_ = s.trackItemActivity(ctx, sessionID, userID)
	}
	return resp, nil
}

// syncSnapshot advances the cached session state after a durable answer write.
// Missing snapshots (cache cold/expired) are simply skipped — the next
// GetSession regenerates from Postgres.
func (s *SessionComposerService) syncSnapshot(ctx context.Context, sessionID, itemID string, score int) {
	snap := s.loadSnapshot(ctx, sessionID)
	if snap == nil {
		return
	}
	for i := range snap.Items {
		if snap.Items[i].ID == itemID {
			snap.Items[i].Status = "answered"
		}
	}
	snap.Completed++
	snap.Score += score
	s.saveSnapshot(ctx, snap)
}

func (s *SessionComposerService) CompleteSession(ctx context.Context, userID, sessionID string) (*models.LearningSession, error) {
	sess, err := s.GetSession(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	if sess.Status == "completed" {
		return sess, nil
	}

	// Xp: 10 per correct answer (approximate from score ceiling).
	xp := sess.CompletedItemCount * 8
	if xp > 200 {
		xp = 200
	}

	// Session completion flushes every durable bookkeeping write in a single
	// transactional record update (plan V3 Phase 8): the session row, daily
	// stats, the activity event, and the readiness bump commit atomically, so
	// a crash mid-completion can never leave half-booked progress.
	err = s.withTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			UPDATE learning_sessions SET status = 'completed', xp_awarded = $3, completed_at = CURRENT_TIMESTAMP
			WHERE id = $1 AND user_id = $2`, sessionID, userID, xp); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO daily_learning_stats (
				user_id, target_language, activity_date, xp, items_completed, reviews_completed, minutes_active
			) VALUES ($1, $2, CURRENT_DATE, $3, $4, $4, GREATEST(1, LEAST(15, $4)))
			ON CONFLICT (user_id, target_language, activity_date) DO UPDATE SET
				xp = daily_learning_stats.xp + $3,
				items_completed = daily_learning_stats.items_completed + $4,
				reviews_completed = daily_learning_stats.reviews_completed + $4,
				minutes_active = daily_learning_stats.minutes_active + GREATEST(1, LEAST(15, $4)),
				updated_at = CURRENT_TIMESTAMP`,
			userID, sess.TargetLanguage, xp, sess.CompletedItemCount); err != nil {
			return err
		}
		payloadJSON, _ := json.Marshal(map[string]any{"mode": sess.Mode})
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO user_activity_events (user_id, target_language, event_type, source_type, source_id, xp, payload)
			VALUES ($1, $2, 'session_completed', 'learning', $3, $4, $5)`,
			userID, sess.TargetLanguage, nullStr(sessionID), xp, string(payloadJSON)); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE user_language_profiles SET readiness_score = LEAST(1000, GREATEST(0, readiness_score + $3)),
				updated_at = CURRENT_TIMESTAMP
			WHERE user_id = $1 AND target_language = $2`, userID, sess.TargetLanguage, xp/10); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Durable state is flushed; the transient Redis mirror is no longer needed.
	s.dropSnapshot(ctx, sessionID)

	updated, _ := s.GetSession(ctx, userID, sessionID)
	return updated, nil
}

// withTx runs fn inside a single database transaction, rolling back on error.
func (s *SessionComposerService) withTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// gradeLessonStep grades a lesson step stored in a session item against its
// answer_key JSON (MCQ choices or accepted answers).
func (s *SessionComposerService) gradeLessonStep(ctx context.Context, stepID string, payload map[string]any, ans models.SessionAnswerRequest) (bool, int) {
	correctAnswer := strFromAny(payload["correctAnswer"])
	answerText := ans.Text
	if ans.Choice != "" {
		answerText = ans.Choice
	}
	if correctAnswer == "" {
		return true, 4 // free-form step: accept as producing language
	}
	correct, quality := s.practice.GradeAnswer(correctAnswer, answerText, strFromAny(payload["promptType"]))
	return correct, quality
}

func (s *SessionComposerService) nextPendingItem(ctx context.Context, sessionID string, afterItemID string) *models.SessionQuestion {
	var lastOrdinal int
	_ = s.db.QueryRowContext(ctx, `
		SELECT ordinal FROM learning_session_items WHERE id = $1`, afterItemID).Scan(&lastOrdinal)
	var payload []byte
	var id, itemType, activityType string
	err := s.db.QueryRowContext(ctx, `
		SELECT id::text, item_type, payload, status FROM learning_session_items
		WHERE session_id = $1 AND ordinal > $2 AND status = 'pending'
		ORDER BY ordinal LIMIT 1`, sessionID, lastOrdinal).Scan(&id, &itemType, &payload, &activityType)
	if err != nil {
		return nil
	}
	var p map[string]any
	_ = json.Unmarshal(payload, &p)
	q := questionFromPayload(p)
	q.ID = id
	q.ItemType = itemType
	q.ActivityType = strFromAny(p["activityType"])
	return &q
}

func (s *SessionComposerService) trackItemActivity(ctx context.Context, sessionID, userID string) error {
	return nil
}

// BookRecovery books a small amount of activity today to preserve an at-risk
// streak. Used by the streak-recovery endpoint.
func (s *SessionComposerService) BookRecovery(ctx context.Context, userID, targetLang, nativeLang string) (int, error) {
	if nativeLang == "" {
		nativeLang = "en"
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO daily_learning_stats (user_id, target_language, activity_date, xp, items_completed, minutes_active)
		VALUES ($1, $2, CURRENT_DATE, 5, 1, 1)
		ON CONFLICT (user_id, target_language, activity_date) DO UPDATE SET
			items_completed = daily_learning_stats.items_completed + 1,
			minutes_active = daily_learning_stats.minutes_active + 1, updated_at = CURRENT_TIMESTAMP`,
		userID, targetLang)
	return 1, err
}

func (s *SessionComposerService) bookStats(ctx context.Context, userID, targetLang string, items, xp int) {
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO daily_learning_stats (
			user_id, target_language, activity_date, xp, items_completed, reviews_completed, minutes_active
		) VALUES ($1, $2, CURRENT_DATE, $3, $4, $4, GREATEST(1, LEAST(15, $4)))
		ON CONFLICT (user_id, target_language, activity_date) DO UPDATE SET
			xp = daily_learning_stats.xp + $3,
			items_completed = daily_learning_stats.items_completed + $4,
			reviews_completed = daily_learning_stats.reviews_completed + $4,
			minutes_active = daily_learning_stats.minutes_active + GREATEST(1, LEAST(15, $4)),
			updated_at = CURRENT_TIMESTAMP`,
		userID, targetLang, xp, items)
}

func (s *SessionComposerService) awardActivity(ctx context.Context, userID, targetLang, eventType, sourceType, sourceID string, xp int, payload map[string]any) {
	p, _ := json.Marshal(payload)
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO user_activity_events (user_id, target_language, event_type, source_type, source_id, xp, payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, userID, targetLang, eventType, sourceType, nullStr(sourceID), xp, string(p))
}

func (s *SessionComposerService) bumpReadiness(ctx context.Context, userID, targetLang string, delta int) {
	_, _ = s.db.ExecContext(ctx, `
		UPDATE user_language_profiles SET readiness_score = LEAST(1000, GREATEST(0, readiness_score + $3)),
			updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND target_language = $2`, userID, targetLang, delta)
}

func (s *SessionComposerService) weakestGrammar(ctx context.Context, userID, targetLang string) (string, string, bool) {
	var gpID, title string
	err := s.db.QueryRowContext(ctx, `
		SELECT g.id::text, g.title FROM user_grammar_mastery m
		JOIN grammar_points g ON g.id = m.grammar_point_id
		WHERE m.user_id = $1 AND m.target_language = $2 AND m.next_review_at <= CURRENT_TIMESTAMP
		ORDER BY m.confidence ASC LIMIT 1`, userID, targetLang).Scan(&gpID, &title)
	if err != nil {
		return "", "", false
	}
	return gpID, title, true
}

func (s *SessionComposerService) touchGrammarMastery(ctx context.Context, userID, targetLang, grammarID string) {
	if grammarID == "" {
		return
	}
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO user_grammar_mastery (user_id, grammar_point_id, target_language, confidence, seen_count, correct_count, next_review_at)
		VALUES ($1, $2, $3, 0.3, 1, 0, CURRENT_TIMESTAMP + INTERVAL '1 day')
		ON CONFLICT (user_id, grammar_point_id) DO UPDATE SET
			seen_count = user_grammar_mastery.seen_count + 1,
			confidence = LEAST(1.0, user_grammar_mastery.confidence + 0.05),
			next_review_at = CURRENT_TIMESTAMP + INTERVAL '1 day',
			last_seen_at = CURRENT_TIMESTAMP
		`, userID, grammarID, targetLang)
}

func targetLangOfSession(ctx context.Context, db *sql.DB, sessionID string) string {
	var lang string
	_ = db.QueryRowContext(ctx, `SELECT target_language FROM learning_sessions WHERE id = $1`, sessionID).Scan(&lang)
	return lang
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func nextStageForCard(card *VocabularyCard) int {
	// A card just below mastery advances one stage per successful session. Start
	// fresh cards at recognition.
	if card.MasteryStage < 1 {
		return stageRecognition
	}
	if card.MasteryStage >= 4 {
		return stageProduction
	}
	return card.MasteryStage
}

func vocabItemPayload(card *VocabularyCard, q models.SessionQuestion, stage int) map[string]any {
	answer := card.Term
	// stageRecognition ("What does this word mean?") expects the English meaning,
	// not the Spanish term. All other stages expect the Spanish term.
	if stage == stageRecognition || q.PromptType == "recognition" {
		if card.Translation != "" {
			answer = card.Translation
		} else if card.Definition != "" {
			answer = card.Definition
		}
	}
	p := map[string]any{
		"prompt":       q.Prompt,
		"promptType":   q.PromptType,
		"activityType": q.ActivityType,
		"stage":        stage,
		"cardId":       card.ID,
		"answer":       answer,
	}
	return p
}

func lessonStepQuestion(step *models.CurriculumStep) models.SessionQuestion {
	var p map[string]any
	_ = json.Unmarshal(mustJSON(step.Prompt), &p)
	text := strFromAny(p["text"])
	if text == "" {
		text = strFromAny(p["title"])
	}
	choices := stringSliceFromAny(p["choices"])
	q := models.SessionQuestion{
		ActivityType: step.Type,
		PromptType:   step.Type,
		Prompt: models.SessionPrompt{
			Text:    text,
			Source:  strFromAny(p["source"]),
			Choices: choices,
		},
	}
	return q
}

// stepAnswerInfo extracts the graded answer + prompt type from a step so a
// session can reconstruct a gradeable MCQ.
func stepAnswerInfo(step *models.CurriculumStep) (string, string) {
	if step == nil {
		return "", "recognition"
	}
	var ak map[string]any
	_ = json.Unmarshal(mustJSON(step.AnswerKey), &ak)
	correct := strFromAny(ak["correct"])
	promptType := step.Type
	switch step.Type {
	case "mcq":
		promptType = "recognition"
	case "cloze":
		promptType = "cued_recall"
	case "free_recall":
		promptType = "production"
	}
	return correct, promptType
}

func lessonStepIDFromPayload(payload any) interface{} {
	if payload == nil {
		return nil
	}
	if p, ok := payload.(map[string]any); ok {
		if id, ok := p["stepId"].(string); ok && id != "" {
			return id
		}
	}
	return nil
}

func questionFromPayload(payload any) models.SessionQuestion {
	m, ok := payload.(map[string]any)
	if !ok || m == nil {
		return models.SessionQuestion{}
	}
	var prompt models.SessionPrompt
	if raw, ok := m["prompt"].(map[string]any); ok {
		rawB, _ := json.Marshal(raw)
		_ = json.Unmarshal(rawB, &prompt)
	} else if sp, ok := m["prompt"].(models.SessionPrompt); ok {
		prompt = sp
	} else {
		rawB, _ := json.Marshal(m["prompt"])
		_ = json.Unmarshal(rawB, &prompt)
	}
	promptType := strFromAny(m["promptType"])
	if promptType == "" {
		promptType = "recognition"
	}
	return models.SessionQuestion{
		PromptType:   promptType,
		ActivityType: strFromAny(m["activityType"]),
		DrillType:    strFromAny(m["drillType"]),
		Prompt:       prompt,
	}
}

// interleaveItems spreads non-vocabulary items so the learner doesn't do all
// vocabulary then all grammar.
func interleaveItems(items []sessionComposedItem) []sessionComposedItem {
	var vocab, other []sessionComposedItem
	for _, it := range items {
		if it.itemType == "vocabulary" {
			vocab = append(vocab, it)
		} else {
			other = append(other, it)
		}
	}
	out := make([]sessionComposedItem, 0, len(items))
	freq := 1
	if len(other) > 0 && len(vocab) > 0 {
		freq = maxInt(1, len(vocab)/len(other))
	}
	vi, oi := 0, 0
	for vi < len(vocab) || oi < len(other) {
		if oi < len(other) && (vi == len(vocab) || (vi > 0 && vi%freq == 0)) {
			out = append(out, other[oi])
			oi++
		}
		if vi < len(vocab) {
			out = append(out, vocab[vi])
			vi++
		}
	}
	return out
}

// (loadForGrade is a thin adapter on PracticeService.)
func (p *PracticeService) loadForGrade(ctx context.Context, userID, cardID string) (*VocabularyCard, error) {
	card, err := p.GetCardByID(ctx, userID, cardID)
	if err != nil {
		return nil, err
	}
	return card, nil
}

func scaleScore(quality int) int {
	switch {
	case quality >= 4:
		return 200
	case quality == 3:
		return 150
	case quality == 2:
		return 100
	case quality == 1:
		return 50
	default:
		return 0
	}
}

func feedbackForQuality(correct bool, correctAnswer string) string {
	if correct {
		return "Correct! Nice work."
	}
	return "Not quite. Review the correct form and try again next time."
}

// grammarDrillComposedItem turns a GrammarPointService drill into a composed
// session item. The payload keeps everything needed to grade offline:
// the correct answer, the item id (for the attempt ledger), the point id, the
// CEFR level (accent policy), and the teaching note.
func grammarDrillComposedItem(d models.GrammarDrillItem) sessionComposedItem {
	text := d.SentenceWithBlank
	if text == "" {
		text = d.Prompt
	}
	return sessionComposedItem{
		itemType:     "grammar",
		activityType: "grammar_drill",
		grammarID:    d.GrammarPointID,
		payload: map[string]any{
			"prompt":         models.SessionPrompt{Text: text, Choices: d.Choices, Source: d.Prompt},
			"promptType":     "cued_recall",
			"activityType":   "grammar_drill",
			"answer":         d.Correct,
			"note":           d.Note,
			"grammarItemId":  d.ID,
			"grammarPointId": d.GrammarPointID,
			"pointTitle":     d.PointTitle,
			"itemType":       d.ItemType,
			"drillType":      d.ItemType,
			"cefrLevel":      d.CEFRLevel,
			"choices":        d.Choices,
		},
	}
}

func grammarDrillFeedback(correct bool, note string) string {
	if correct {
		if note != "" {
			return "Correct! " + note
		}
		return "Correct! Nice work."
	}
	if note != "" {
		return "Not quite. " + note
	}
	return "Not quite. Review the correct form and try again next time."
}

func stepFeedback(correct bool) string {
	if correct {
		return "Great. Move on."
	}
	return "Almost — check the suggested phrasing."
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func strFromAny(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func intFromAny(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	}
	return 0
}

func stringSliceFromAny(v any) []string {
	switch t := v.(type) {
	case []interface{}:
		out := make([]string, 0, len(t))
		for _, e := range t {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return t
	}
	return nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
