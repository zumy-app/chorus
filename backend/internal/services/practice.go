package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/observability"
)

// Type aliases so the practice service can work with the learning card model
// without prefixing every reference with `models.`.
type (
	VocabularyCard  = models.VocabularyCard
	SessionQuestion = models.SessionQuestion
	SessionPrompt   = models.SessionPrompt
	SessionFeedback = models.SessionFeedback
)

const (
	stageRecognition = models.StageRecognition
	stageCuedRecall  = models.StageCuedRecall
	stageFreeRecall  = models.StageFreeRecall
	stageProduction  = models.StageProduction
	stageSpontaneous = models.StageSpontaneous

	stateNew       = models.StateNew
	stateLearning  = models.StateLearning
	stateReviewing = models.StateReviewing
	stateMastered  = models.StateMastered
	stateLeech     = models.StateLeech
)

// PracticeService grades depth-of-processing SRS answers and mutates the
// stage-aware spaced-repetition state on a vocabulary card. It implements the
// ladder from section 9 of the implementation plan: recognition (1), cued
// recall (2), free recall (3), production (4), spontaneous use (5).
//
// The service is deliberately db-agnostic about session accounting: it only
// owns the card-level SRS transitions and records practice attempts. Session
// composition/accounting lives in SessionComposerService.
type PracticeService struct {
	db *sql.DB
}

func NewPracticeService(db *sql.DB) *PracticeService {
	return &PracticeService{db: db}
}

// GradeAnswer decides whether an answer to a given prompt type is correct and
// assigns a quality score 0-5 (see plan section 9 quality mapping).
func (s *PracticeService) GradeAnswer(correctAnswer string, answerText string, promptType string) (bool, int) {
	if correctAnswer == "" {
		return false, 0
	}
	ans := normalizeAnswer(answerText)
	correct := normalizeAnswer(correctAnswer)

	// Prompt types accept different answer forms. For MCQ the answer is already
	// one of the choices; for cloze/free recall we do a loose match. An empty
	// answer must never score as correct (strings.Contains(x, "") is true).
	match := ans != "" && (ans == correct || strings.Contains(ans, correct) || strings.Contains(correct, ans))

	switch promptType {
	case "cued_recall", "cloze", "recognition":
		if match {
			return true, 4
		}
		return false, 1
	case "free_recall", "production", "translation":
		if match {
			return true, 4
		}
		if wordDistance(ans, correct) <= 1 {
			return true, 3
		}
		return false, 1
	default:
		if match {
			return true, 4
		}
		return false, 1
	}
}

// UpdateVocabAfterAttempt mutates the SRS state of a card according to the
// stage-aware SM-2 variant described in plan section 9. It persists the change.
func (s *PracticeService) UpdateVocabAfterAttempt(ctx context.Context, card *VocabularyCard, stage int, quality int, activityType string) error {
	prevStage := card.MasteryStage
	prevState := card.MasteryState
	now := time.Now()
	correct := quality >= 3
	observability.ObservePracticeAttempt(stage, correct, 0)
	card.ReviewCount++
	if quality < 3 {
		card.Lapses++
		card.StageSuccessCount = 0
		card.IntervalDays = 1
		card.EaseFactor = math.Max(1.30, card.EaseFactor-0.20)
		card.NextReview = now.AddDate(0, 0, 1)
		if stage >= 3 {
			card.MasteryStage = max(1, card.MasteryStage-1)
		}
		card.MasteryState = learningOrNew(card)
		if card.MasteryState == stateLeech && prevState != stateLeech {
			observability.IncPracticeLeech()
		}
		if prevStage != card.MasteryStage {
			observability.ObservePracticePromotion(prevStage, card.MasteryStage)
		}
		return s.persistCard(ctx, card)
	}

	card.CorrectCount++
	card.StageSuccessCount++
	if stage == 4 {
		card.ProductionSuccessCount++
	}

	if quality >= 4 && card.StageSuccessCount >= requiredSuccesses(stage) {
		card.MasteryStage = min(4, card.MasteryStage+1)
		card.StageSuccessCount = 0
	}

	card.EaseFactor = clamp(1.30, 3.00, card.EaseFactor+easeDelta(quality))
	card.IntervalDays = nextInterval(card.IntervalDays, card.EaseFactor, stage)
	card.NextReview = now.AddDate(0, 0, int(card.IntervalDays))
	card.MasteryState = masteryStateOf(card)
	if card.MasteryState == stateLeech && prevState != stateLeech {
		observability.IncPracticeLeech()
	}
	if card.MasteryState == stateMastered && prevState != stateMastered {
		observability.IncPracticeMastered()
	}
	if prevStage != card.MasteryStage {
		observability.ObservePracticePromotion(prevStage, card.MasteryStage)
	}
	return s.persistCard(ctx, card)
}

func (s *PracticeService) persistCard(ctx context.Context, card *VocabularyCard) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE vocabulary SET
			review_count = $1, correct_count = $2, interval_days = $3,
			next_review = $4, mastery_stage = $5, mastery_state = $6,
			ease_factor = $7, lapses = $8, stage_success_count = $9,
			production_success_count = $10, spontaneous_use_count = $11,
			last_seen_at = CURRENT_TIMESTAMP
		WHERE id = $12 AND user_id = $13`,
		card.ReviewCount, card.CorrectCount, card.IntervalDays, card.NextReview,
		card.MasteryStage, card.MasteryState, card.EaseFactor, card.Lapses,
		card.StageSuccessCount, card.ProductionSuccessCount, card.SpontaneousUseCount,
		card.ID, card.UserID)
	if err != nil {
		return fmt.Errorf("update card %s: %w", card.ID, err)
	}
	return nil
}

// RecordAttempt persists a single practice answer in vocabulary_practice_attempts.
func (s *PracticeService) RecordAttempt(ctx context.Context, card *VocabularyCard, stage int, activityType string, prompt any, answer string, correct bool, quality int, latencyMs int, sourceSessionID string) error {
	promptJSON, _ := json.Marshal(prompt)
	answerJSON, _ := json.Marshal(map[string]any{"text": answer})
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO vocabulary_practice_attempts (
			vocabulary_id, user_id, target_language, stage, activity_type,
			prompt, answer, correct, quality, latency_ms, source_session_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		card.ID, card.UserID, card.Language, stage, activityType,
		string(promptJSON), string(answerJSON), correct, quality, nullInt(latencyMs), nullStr(sourceSessionID))
	if err != nil {
		return fmt.Errorf("record attempt: %w", err)
	}
	return nil
}

// GetDueCards returns the user's due vocabulary cards for a language. When
// mastery is achieved (production-level mastery), cards are spaced out and only
// appear when genuinely due.
func (s *PracticeService) GetDueCards(ctx context.Context, userID, targetLanguage string, limit int) ([]VocabularyCard, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id::text, user_id, term, language, COALESCE(translation,''), COALESCE(definition,''),
		       COALESCE(lemma,''), COALESCE(normalized_term,''), COALESCE(part_of_speech,'unknown'),
		       is_chunk, source_type, COALESCE(source_message_id::text,''),
		       COALESCE(cefr_level,''), COALESCE(curriculum_unit_id::text,''),
		       route_status, mastery_stage, mastery_state, COALESCE(ease_factor, 2.5),
		       lapses, stage_success_count, production_success_count, spontaneous_use_count,
		       teachability_score, confidence, review_count, correct_count,
		       COALESCE(interval_days,1), next_review, COALESCE(context_sentence,''),
		       COALESCE(context_message_id::text,''), COALESCE(context_chat_id::text,''),
		       created_at, first_seen_at, last_seen_at
		FROM vocabulary
		WHERE user_id = $1 AND language = $2
		  AND next_review <= CURRENT_TIMESTAMP
		ORDER BY next_review ASC
		LIMIT $3`, userID, targetLanguage, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCards(rows)
}

// GetCardByID loads a single vocabulary card for grading/resume.
func (s *PracticeService) GetCardByID(ctx context.Context, userID, cardID string) (*VocabularyCard, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id::text, user_id, term, language, COALESCE(translation,''), COALESCE(definition,''),
		       COALESCE(lemma,''), COALESCE(normalized_term,''), COALESCE(part_of_speech,'unknown'),
		       is_chunk, source_type, COALESCE(source_message_id::text,''),
		       COALESCE(cefr_level,''), COALESCE(curriculum_unit_id::text,''),
		       route_status, mastery_stage, mastery_state, COALESCE(ease_factor, 2.5),
		       lapses, stage_success_count, production_success_count, spontaneous_use_count,
		       teachability_score, confidence, review_count, correct_count,
		       COALESCE(interval_days,1), next_review, COALESCE(context_sentence,''),
		       COALESCE(context_message_id::text,''), COALESCE(context_chat_id::text,''),
		       created_at, first_seen_at, last_seen_at
		FROM vocabulary
		WHERE id = $1 AND user_id = $2`, cardID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cards, err := scanCards(rows)
	if err != nil || len(cards) == 0 {
		return nil, fmt.Errorf("vocabulary card not found")
	}
	return &cards[0], nil
}

// GetUnpromptedCards returns cards not yet reaching stage 2 to seed a fresh
// learner when they have no due reviews yet.
func (s *PracticeService) GetNewCards(ctx context.Context, userID, targetLanguage string, limit int) ([]VocabularyCard, error) {
	if limit <= 0 {
		limit = 5
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id::text, user_id, term, language, COALESCE(translation,''), COALESCE(definition,''),
		       COALESCE(lemma,''), COALESCE(normalized_term,''), COALESCE(part_of_speech,'unknown'),
		       is_chunk, source_type, COALESCE(source_message_id::text,''),
		       COALESCE(cefr_level,''), COALESCE(curriculum_unit_id::text,''),
		       route_status, mastery_stage, mastery_state, COALESCE(ease_factor, 2.5),
		       lapses, stage_success_count, production_success_count, spontaneous_use_count,
		       teachability_score, confidence, review_count, correct_count,
		       COALESCE(interval_days,1), next_review, COALESCE(context_sentence,''),
		       COALESCE(context_message_id::text,''), COALESCE(context_chat_id::text,''),
		       created_at, first_seen_at, last_seen_at
		FROM vocabulary
		WHERE user_id = $1 AND language = $2
		  AND mastery_stage < 2
		ORDER BY teachability_score DESC, created_at ASC
		LIMIT $3`, userID, targetLanguage, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCards(rows)
}

func scanCards(rows *sql.Rows) ([]VocabularyCard, error) {
	cards := []VocabularyCard{}
	for rows.Next() {
		var c VocabularyCard
		var sourceMsg, cefr, unitID, ctxMsg, ctxChat sql.NullString
		var nextReview time.Time
		if err := rows.Scan(
			&c.ID, &c.UserID, &c.Term, &c.Language, &c.Translation, &c.Definition,
			&c.Lemma, &c.NormalizedTerm, &c.PartOfSpeech, &c.IsChunk, &c.SourceType,
			&sourceMsg, &cefr, &unitID, &c.RouteStatus, &c.MasteryStage, &c.MasteryState,
			&c.EaseFactor, &c.Lapses, &c.StageSuccessCount, &c.ProductionSuccessCount,
			&c.SpontaneousUseCount, &c.TeachabilityScore, &c.Confidence, &c.ReviewCount,
			&c.CorrectCount, &c.IntervalDays, &nextReview, &c.ContextSentence,
			&ctxMsg, &ctxChat, &c.CreatedAt, &c.FirstSeenAt, &c.LastSeenAt,
		); err != nil {
			return nil, err
		}
		if sourceMsg.Valid {
			c.SourceMessageID = sourceMsg.String
		}
		if cefr.Valid {
			c.CEFRLevel = cefr.String
		}
		if unitID.Valid {
			c.CurriculumUnitID = unitID.String
		}
		if ctxMsg.Valid {
			c.ContextMessageID = ctxMsg.String
		}
		if ctxChat.Valid {
			c.ContextChatID = ctxChat.String
		}
		c.NextReview = nextReview
		cards = append(cards, c)
	}
	return cards, rows.Err()
}

func isMastered(card *VocabularyCard) bool {
	if card.MasteryStage < 4 {
		return false
	}
	if card.ProductionSuccessCount >= 2 {
		return true
	}
	if card.ProductionSuccessCount >= 1 && card.SpontaneousUseCount >= 1 {
		return true
	}
	return false
}

func (s *PracticeService) TouchSpontaneousUse(ctx context.Context, userID, normalizedTerm, targetLanguage string) error {
	card, err := s.loadCardByNormalized(ctx, userID, normalizedTerm, targetLanguage)
	if err != nil {
		return nil
	}
	prevStage := card.MasteryStage
	prevState := card.MasteryState
	card.SpontaneousUseCount++
	observability.IncPracticeSpontaneous()
	if card.MasteryStage == 4 && card.ProductionSuccessCount >= 1 {
		card.MasteryStage = 5
		observability.ObservePracticePromotion(prevStage, 5)
	}
	if isMastered(card) {
		card.MasteryState = stateMastered
		card.IntervalDays = 14
		card.NextReview = time.Now().AddDate(0, 0, 14)
		if prevState != stateMastered {
			observability.IncPracticeMastered()
		}
	} else if card.MasteryState == stateNew {
		card.MasteryState = stateLearning
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE vocabulary SET
			spontaneous_use_count = $1,
			mastery_stage = $2,
			mastery_state = $3,
			interval_days = $4,
			next_review = $5,
			last_seen_at = CURRENT_TIMESTAMP
		WHERE id = $6 AND user_id = $7`,
		card.SpontaneousUseCount, card.MasteryStage, card.MasteryState,
		card.IntervalDays, card.NextReview, card.ID, card.UserID)
	return err
}

func (s *PracticeService) loadCardByNormalized(ctx context.Context, userID, normalizedTerm, lang string) (*VocabularyCard, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id::text, user_id, term, language, COALESCE(translation,''), COALESCE(definition,''),
		       COALESCE(lemma,''), COALESCE(normalized_term,''), COALESCE(part_of_speech,'unknown'),
		       is_chunk, source_type, COALESCE(source_message_id::text,''),
		       COALESCE(cefr_level,''), COALESCE(curriculum_unit_id::text,''),
		       route_status, mastery_stage, mastery_state, COALESCE(ease_factor, 2.5),
		       lapses, stage_success_count, production_success_count, spontaneous_use_count,
		       teachability_score, confidence, review_count, correct_count,
		       COALESCE(interval_days,1), next_review, COALESCE(context_sentence,''),
		       COALESCE(context_message_id::text,''), COALESCE(context_chat_id::text,''),
		       created_at, first_seen_at, last_seen_at
		FROM vocabulary
		WHERE user_id = $1 AND language = $2 AND normalized_term = $3 LIMIT 1`, userID, lang, normalizedTerm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cards, err := scanCards(rows)
	if err != nil || len(cards) == 0 {
		return nil, fmt.Errorf("vocabulary card not found")
	}
	return &cards[0], nil
}

func (s *PracticeService) ReviewCard(ctx context.Context, userID, cardID, answerText string, latencyMs int) (*VocabularyCard, bool, int, error) {
	card, err := s.GetCardByID(ctx, userID, cardID)
	if err != nil {
		return nil, false, 0, err
	}
	stage := nextStageForCard(card)
	promptType := promptTypeForStage(stage)
	correct, quality := s.GradeAnswer(card.Term, answerText, promptType)
	activityType := s.stageTemplate(card, stage)
	if err := s.UpdateVocabAfterAttempt(ctx, card, stage, quality, activityType); err != nil {
		return nil, correct, quality, err
	}
	_ = s.RecordAttempt(ctx, card, stage, activityType, map[string]any{"text": card.Term}, answerText, correct, quality, latencyMs, "")
	return card, correct, quality, nil
}

// BuildVocabQuestion turns a card and an activity/stage into a client
// SessionQuestion prompt.
func (s *PracticeService) BuildVocabQuestion(card *VocabularyCard, stage int) (string, SessionQuestion) {
	template := s.stageTemplate(card, stage)
	// terminal prompt shape: text = instruction, source = target term, can be
	// used as a fallback when the client only needs text-based cards.
	var t SessionPrompt
	switch stage {
	case stageRecognition:
		t = SessionPrompt{Text: "What does this word mean?", Source: card.Term, Translation: card.Translation}
	case stageCuedRecall:
		t = SessionPrompt{Text: "Fill in the missing word from your message.", Term: card.Term}
	case stageFreeRecall:
		t = SessionPrompt{Text: "Type the Spanish for: " + meaningOf(card), Source: meaningOf(card)}
	case stageProduction:
		t = SessionPrompt{Text: fmt.Sprintf("Use '%s' in a new Spanish sentence.", card.Term), GrammarHint: card.PartOfSpeech}
	default:
		t = SessionPrompt{Text: "What does this word mean?", Source: card.Term, Translation: card.Translation}
	}
	q := SessionQuestion{
		ItemType:     "vocabulary",
		ActivityType: template,
		PromptType:   promptTypeForStage(stage),
		Prompt:       t,
	}
	return template, q
}

func (s *PracticeService) stageTemplate(card *VocabularyCard, stage int) string {
	switch stage {
	case stageCuedRecall:
		return "cued_recall"
	case stageFreeRecall:
		return "free_recall"
	case stageProduction:
		return "production"
	default:
		return "recognition"
	}
}

func promptTypeForStage(stage int) string {
	switch stage {
	case stageRecognition:
		return "recognition"
	case stageCuedRecall:
		return "cued_recall"
	case stageFreeRecall:
		return "free_recall"
	case stageProduction:
		return "production"
	default:
		return "recognition"
	}
}

func meaningOf(card *VocabularyCard) string {
	if card.Translation != "" {
		return card.Translation
	}
	return card.Definition
}

// normalizeAnswer lowercases, trims, and strips punctuation for lenient matching.
func normalizeAnswer(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	// fold accents to their base letters, strip punctuation, collapse spaces.
	s = foldAccents(s)
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			return r
		}
		return -1
	}, s)
	return strings.Join(strings.Fields(s), " ")
}

func foldAccents(s string) string {
	replacer := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
		"¿", "", "¡", "",
	)
	return replacer.Replace(s)
}

// wordDistance is a naive edit distance used to accept near-miss production.
func wordDistance(a, b string) int {
	ar := []rune(a)
	br := []rune(b)
	n, m := len(ar), len(br)
	if n == 0 {
		return m
	}
	if m == 0 {
		return n
	}
	prev := make([]int, m+1)
	cur := make([]int, m+1)
	for j := 0; j <= m; j++ {
		prev[j] = j
	}
	for i := 1; i <= n; i++ {
		cur[0] = i
		for j := 1; j <= m; j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			cur[j] = min(min(cur[j-1]+1, prev[j]+1), prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[m]
}

func requiredSuccesses(stage int) int {
	switch stage {
	case 1:
		return 1
	case 2:
		return 1
	case 3:
		return 2
	case 4:
		return 2
	case 5:
		return 1
	default:
		return 2
	}
}

func easeDelta(quality int) float64 {
	switch {
	case quality >= 5:
		return 0.15
	case quality == 4:
		return 0.05
	case quality == 3:
		return -0.05
	default:
		return -0.20
	}
}

func nextInterval(current float64, ease float64, stage int) float64 {
	if current <= 0 {
		current = 1
	}
	switch stage {
	case 1:
		return 2
	case 2:
		return 3
	case 3:
		return 5
	case 4:
		if current < 7 {
			return 7
		}
		return math.Ceil(current * ease)
	case 5:
		if current < 14 {
			return 14
		}
		return math.Ceil(current * ease)
	default:
		return math.Ceil(current * ease)
	}
}

func clamp(lo, hi, v float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func learningOrNew(card *VocabularyCard) string {
	if card.MasteryStage <= 1 && card.ReviewCount <= 1 {
		return stateNew
	}
	return stateLearning
}

func masteryStateOf(card *VocabularyCard) string {
	if card.Lapses >= 3 && card.MasteryStage <= 2 {
		return stateLeech
	}
	if isMastered(card) {
		return stateMastered
	}
	if card.MasteryStage >= 3 {
		return stateReviewing
	}
	if card.MasteryStage >= 2 {
		return stateLearning
	}
	return stateNew
}

func nullInt(v int) interface{} {
	if v <= 0 {
		return nil
	}
	return v
}
