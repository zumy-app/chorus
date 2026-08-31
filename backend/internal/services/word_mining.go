package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"regexp"
	"strings"
	"unicode"

	"github.com/chorus/messenger/internal/models"
	"github.com/lib/pq"
)

// autoAddThreshold: a mined item is silently added to the learner's vocabulary
// when its teachability score reaches this value. Manual saves always add.
// Candidates below the threshold stay as candidates awaiting confirmation.
const autoAddThreshold = 60.0

// WordMiningService extracts high-yield vocabulary and chunks from real target
// language messages, classifies them against the CEFR curriculum, dedupes them
// against the learner's vocabulary, and routes them into SRS or upcoming units.
//
// It is not a queue itself: the WordMiningQueueService owns durable job rows and
// retries; this service owns the extraction + classification + persistence.
type WordMiningService struct {
	db           *sql.DB
	ai           *LearningAIService
	translation  *TranslationService
	curriculum   *CurriculumService
	capabilities *LearningCapabilityService
}

func NewWordMiningService(db *sql.DB, ai *LearningAIService, translation *TranslationService, curriculum *CurriculumService, capabilities *LearningCapabilityService) *WordMiningService {
	return &WordMiningService{db: db, ai: ai, translation: translation, curriculum: curriculum, capabilities: capabilities}
}

// NormalizeLearningTerm lowers, trims, folds accents and punctuation, collapses
// whitespace, and strips inverted punctuation so equivalent terms dedupe.
func NormalizeLearningTerm(s string, lang string) string {
	_ = lang
	s = strings.ToLower(strings.TrimSpace(s))
	s = foldAccents(s)
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) || r == '\'' {
			return r
		}
		return -1
	}, s)
	return strings.Join(strings.Fields(s), " ")
}

// ProcessJob runs full extraction/classification/dedupe for a word_mining_jobs
// row and returns the number of accepted (auto-added) items. It is idempotent.
func (s *WordMiningService) ProcessJob(ctx context.Context, jobID string) (int, error) {
	var (
		userID, chatID, messageID, sourceType, sourceText, sourceLang, nativeLang, status string
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT user_id, COALESCE(chat_id::text,''), COALESCE(message_id::text,''),
		       source_type, source_text, source_language, native_language, status
		FROM word_mining_jobs WHERE id = $1`, jobID).Scan(
		&userID, &chatID, &messageID, &sourceType, &sourceText, &sourceLang, &nativeLang, &status)
	if err != nil {
		return 0, err
	}
	if status == "done" {
		return 0, nil
	}

	items, err := s.extract(ctx, sourceText, sourceLang, nativeLang)
	if err != nil {
		return 0, err
	}
	log.Printf("[Mining] job %s: extracted %d candidate(s) from %q", jobID, len(items), sourceText)

	accepted := 0
	for _, item := range items {
		candidate, err := s.classifyAndRoute(ctx, userID, sourceType, chatID, messageID, sourceText, item, sourceLang, nativeLang)
		if err != nil {
			log.Printf("[Mining] classify %q: %v", item.SurfaceText, err)
			continue
		}
		if candidate == nil {
			continue
		}
		// Dedupe against existing vocabulary; if present, just record a source
		// and move on (the card already exists and may already be scheduled).
		existing, err := s.findExistingVocabulary(ctx, userID, candidate.NormalizedText, sourceLang)
		if err != nil {
			continue
		}
		if existing != "" {
			if err := s.upsertMinedItem(ctx, candidate, jobID, "merged"); err != nil {
				continue
			}
			if err := s.touchExistingCard(ctx, userID, existing, sourceType, sourceText, candidate); err != nil {
				continue
			}
			continue
		}

		// Insert the mined item (candidate unless auto-added).
		status := "candidate"
		if candidate.TeachabilityScore >= autoAddThreshold {
			status = "auto_added"
		}
		minedID, err := s.insertMinedItem(ctx, candidate, jobID, status)
		if err != nil {
			log.Printf("[Mining] insert item %q: %v", candidate.SurfaceText, err)
			continue
		}
		log.Printf("[Mining] inserted item %q as %s (teach=%v route=%s)", candidate.SurfaceText, status, candidate.TeachabilityScore, candidate.RouteStatus)
		if status == "auto_added" {
			if _, err := s.addCardFromMinedItem(ctx, candidate, minedID); err == nil {
				accepted++
			}
		}
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE word_mining_jobs SET status = 'done', result = $1, completed_at = CURRENT_TIMESTAMP
		WHERE id = $2`, fmt.Sprintf(`{"accepted":%d}`, accepted), jobID)
	return accepted, err
}

// extract runs AI extraction when available and falls back to deterministic
// tokenization + stopword filtering + translation-backed definitions.
func (s *WordMiningService) extract(ctx context.Context, text, sourceLang, nativeLang string) ([]MinedCandidate, error) {
	if s.ai != nil && s.ai.HasProviders() {
		if items, err := s.ai.ExtractVocabulary(ctx, text, sourceLang, nativeLang, ""); err == nil && len(items) > 0 {
			// Sanitize: drop empty/duplicate surfaces.
			seen := map[string]bool{}
			out := items[:0]
			for _, it := range items {
				term := strings.TrimSpace(it.SurfaceText)
				if term == "" || seen[NormalizeLearningTerm(term, sourceLang)] {
					continue
				}
				seen[NormalizeLearningTerm(term, sourceLang)] = true
				it.SurfaceText = term
				out = append(out, it)
			}
			return out, nil
		}
		// fall through to deterministic extraction
	}
	return s.fallbackExtract(ctx, text, sourceLang, nativeLang)
}

func (s *WordMiningService) fallbackExtract(ctx context.Context, text, sourceLang, nativeLang string) ([]MinedCandidate, error) {
	tokens := tokenizeCandidates(text)
	var out []MinedCandidate
	seen := map[string]bool{}
	for _, t := range tokens {
		if len([]rune(t)) <= 3 || isStopword(t, sourceLang) {
			continue
		}
		norm := NormalizeLearningTerm(t, sourceLang)
		if seen[norm] {
			continue
		}
		seen[norm] = true
		tr := ""
		if s.translation != nil {
			if v, err := s.translation.Translate(t, nativeLang); err == nil {
				tr = v
			}
		}
		out = append(out, MinedCandidate{
			SurfaceText:  t,
			Lemma:        t,
			PartOfSpeech: "unknown",
			Translation:  tr,
			Definition:   tr,
			CEFRLevel:    "A1",
			Confidence:   0.35,
			Reason:       "fallback tokenization",
		})
		if len(out) >= 8 {
			break
		}
	}
	return out, nil
}

var wordRe = regexp.MustCompile(`[\pL\pN][\pL\pN'’\-]*`)

func tokenizeCandidates(s string) []string {
	var out []string
	for _, tok := range wordRe.FindAllString(s, -1) {
		tok = strings.Trim(tok, "'’\u2019-")
		if tok == "" {
			continue
		}
		// only consider words with at least one letter
		hasLetter := false
		for _, r := range tok {
			if unicode.IsLetter(r) {
				hasLetter = true
				break
			}
		}
		if hasLetter {
			out = append(out, tok)
		}
	}
	return out
}

// classifyAndRoute matches an extracted item against the curriculum and decides
// the route status and teachability score. It returns the enriched candidate.
func (s *WordMiningService) classifyAndRoute(ctx context.Context, userID, sourceType, chatID, messageID, sourceText string, item MinedCandidate, sourceLang, nativeLang string) (*MinedCandidate, error) {
	norm := NormalizeLearningTerm(item.SurfaceText, sourceLang)
	lemma := item.Lemma
	if lemma == "" {
		lemma = item.SurfaceText
	}
	normalized := &MinedCandidate{
		SurfaceText:  item.SurfaceText,
		Lemma:        lemma,
		NormalizedText: norm,
		PartOfSpeech: item.PartOfSpeech,
		IsChunk:      item.IsChunk || strings.Contains(item.SurfaceText, " "),
		Translation:  item.Translation,
		Definition:   item.Definition,
		CEFRLevel:    item.CEFRLevel,
		GrammarTags:  item.GrammarTags,
		IsProperNoun: item.IsProperNoun,
		Confidence:   item.Confidence,
		Language:     sourceLang,
		UserID:       userID,
		SourceType:   sourceType,
		MessageID:    messageID,
		ChatID:       chatID,
		SourceMessageID: messageID,
		ContextSentence: contextSentence(sourceText, item.SurfaceText),
	}
	if normalized.Confidence <= 0 {
		normalized.Confidence = 0.4
	}

	// Resolve the active course for the pair.
	courseID, err := s.activeCourseID(ctx, nativeLang, sourceLang)
	if err != nil {
		return nil, err
	}

	if courseID != "" {
		lexID, unitID, level := s.matchLexicalItem(ctx, courseID, sourceLang, lemma, item.SurfaceText)
		if lexID != "" {
			normalized.CurriculumLexicalID = lexID
			normalized.CurriculumUnitID = unitID
			if normalized.CEFRLevel == "" {
				normalized.CEFRLevel = level
			}
		}
	}

	// Deterministic route: current unit => current_unit, else bonus/upcoming.
	route, routeReason := s.determineRoute(ctx, userID, sourceLang, normalized.CurriculumUnitID, sliceSet(normalized.GrammarTags))
	normalized.RouteStatus = route
	normalized.RouteReason = routeReason
	normalized.TeachabilityScore = s.teachability(item, route)

	// Proper nouns always discouraged; very advanced relative to A1 default.
	if normalized.IsProperNoun {
		normalized.RouteStatus = "bonus"
		normalized.TeachabilityScore = math.Max(0, normalized.TeachabilityScore-20)
	}
	return normalized, nil
}

func (s *WordMiningService) activeCourseID(ctx context.Context, nativeLang, targetLang string) (string, error) {
	cap, err := s.capabilities.GetCapability(ctx, nativeLang, targetLang)
	if err != nil {
		return "", err
	}
	return cap.ActiveCourseID, nil
}

// matchLexicalItem looks up a curriculum lexical item by lemma, display_text, or
// forms aliases for the given course/language.
func (s *WordMiningService) matchLexicalItem(ctx context.Context, courseID, lang, lemma, display string) (lexID, unitID, level string) {
	// Prefer an exact (accent-folded) match via ILIKE on the display_text or
	// lemma, then relax to a substring match so minor morphological variants
	// still land in the right curriculum unit.
	err := s.db.QueryRowContext(ctx, `
		SELECT l.id::text, COALESCE(l.unit_id::text,''), l.cefr_level
		FROM lexical_items l
		WHERE l.course_id = $1 AND l.language = $2
		  AND (l.lemma ILIKE $3 OR l.display_text ILIKE $4)
		LIMIT 1`,
		courseID, lang, "%"+lemma+"%", "%"+display+"%").Scan(&lexID, &unitID, &level)
	if err != nil {
		_ = s.db.QueryRowContext(ctx, `
			SELECT l.id::text, COALESCE(l.unit_id::text,''), l.cefr_level
			FROM lexical_items l
			WHERE l.course_id = $1 AND l.language = $2 AND l.lemma ILIKE $3
			LIMIT 1`, courseID, lang, "%"+lemma+"%").Scan(&lexID, &unitID, &level)
	}
	return
}

// determineRoute classifies a matched/non-matched item relative to the user's
// active unit: upcoming_unit, current_unit, completed_unit, or bonus.
func (s *WordMiningService) determineRoute(ctx context.Context, userID, targetLang, unitID string, grammarTags []string) (string, string) {
	if unitID == "" {
		if len(grammarTags) > 0 {
			return "bonus", "no curriculum unit matched"
		}
		return "bonus", "no curriculum unit matched"
	}

	var profileActiveUnit string
	_ = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(active_unit_id::text,'') FROM user_language_profiles
		WHERE user_id = $1 AND target_language = $2`, userID, targetLang).Scan(&profileActiveUnit)

	if profileActiveUnit == "" || profileActiveUnit == unitID {
		return "current_unit", "matches the learner's active unit"
	}

	// If the matched unit belongs to a completed unit, it is reinforcement.
	var done bool
	_ = s.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM user_unit_progress WHERE user_id = $1 AND unit_id = $2 AND status = 'completed')`,
		userID, unitID).Scan(&done)
	if done {
		return "completed_unit", "reinforcement for a completed unit"
	}
	return "upcoming_unit", "belongs to a future unit"
}

func (s *WordMiningService) teachability(item MinedCandidate, route string) float64 {
	score := 0.0
	// personal_context: chat-mined words are always contextually relevant.
	score += 25
	// curriculum_relevance
	if route == "current_unit" {
		score += 20
	} else if route == "upcoming_unit" {
		score += 14
	} else if route == "completed_unit" {
		score += 10
	} else {
		score += 5
	}
	// frequency proxy: shorter common words score higher.
	chars := len([]rune(item.SurfaceText))
	if chars <= 6 {
		score += 15
	} else if chars <= 12 {
		score += 10
	} else {
		score += 5
	}
	// recency
	score += 15
	// not mastered (we don't know yet, reward newness)
	score += 10
	// confidence
	score += 10 * item.Confidence
	// chunk bonus
	if item.IsChunk {
		score += 5
	}
	if item.IsProperNoun {
		score -= 20
	}
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return math.Round(score)
}

func sliceSet(s []string) []string {
	return s
}

// findExistingVocabulary returns the id of an existing card for the normalized
// term, or "" when no card exists.
func (s *WordMiningService) findExistingVocabulary(ctx context.Context, userID, normalized, lang string) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `
		SELECT id::text FROM vocabulary
		WHERE user_id = $1 AND language = $2 AND normalized_term = $3
		LIMIT 1`, userID, lang, normalized).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return id, err
}

func (s *WordMiningService) touchExistingCard(ctx context.Context, userID, cardID, sourceType, sourceText string, candidate *MinedCandidate) error {
	sentence := contextSentence(sourceText, candidate.SurfaceText)
	// Update last_seen_at and increment spontaneous use when the card was seen
	// in a message the learner authored (outgoing target-language use).
	if candidate.SourceType == "chat" {
		_, _ = s.db.ExecContext(ctx, `
			UPDATE vocabulary SET spontaneous_use_count = spontaneous_use_count + 1,
				last_seen_at = CURRENT_TIMESTAMP
			WHERE id = $1 AND user_id = $2`, cardID, userID)
	}
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO vocabulary_sources (vocabulary_id, source_type, source_id, sentence, seen_count)
		VALUES ($1, $2, NULL, $3, 1)
		ON CONFLICT (vocabulary_id, source_type, source_id) DO UPDATE SET seen_count = vocabulary_sources.seen_count + 1`,
		cardID, sourceType, sentence)
	return nil
}

func (s *WordMiningService) insertMinedItem(ctx context.Context, c *MinedCandidate, jobID, status string) (string, error) {
	span, _ := json.Marshal(map[string]any{"start": 0, "length": len([]rune(c.SurfaceText))})
	var id string
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO mined_items (
			user_id, job_id, source_type, surface_text, lemma, normalized_text, language,
			part_of_speech, translation, definition, context_sentence, text_span,
			cefr_level, confidence, teachability_score, is_chunk, is_proper_noun,
			grammar_tags, curriculum_lexical_item_id, curriculum_unit_id,
			route_status, status, route_reason
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
		RETURNING id::text`,
		c.UserID, nullStr(jobID), c.SourceType, c.SurfaceText, c.Lemma, c.NormalizedText, c.Language,
		c.PartOfSpeech, c.Translation, c.Definition, c.ContextSentence, string(span),
		nullStr(c.CEFRLevel), c.Confidence, c.TeachabilityScore, c.IsChunk, c.IsProperNoun,
		stringSlice(c.GrammarTags), nullStr(c.CurriculumLexicalID), nullStr(c.CurriculumUnitID),
		c.RouteStatus, status, c.RouteReason).Scan(&id)
	return id, err
}

func (s *WordMiningService) upsertMinedItem(ctx context.Context, c *MinedCandidate, jobID, status string) error {
	_, err := s.insertMinedItem(ctx, c, jobID, status)
	return err
}

// AcceptMinedItem promotes a candidate into a real vocabulary card (manual
// confirm). It is also the route used by the tap-to-save flow.
func (s *WordMiningService) AcceptMinedItem(ctx context.Context, userID, minedID string) (*VocabularyCard, error) {
	var c MinedCandidate
	var jobID, sourceType, contextSentence, user, lang string
	err := s.db.QueryRowContext(ctx, `
		SELECT user_id, COALESCE(job_id::text,''), source_type, surface_text, lemma, normalized_text,
		       language, part_of_speech, translation, definition, context_sentence,
		       COALESCE(cefr_level,''), confidence, teachability_score, is_chunk, is_proper_noun,
		       grammar_tags, COALESCE(curriculum_lexical_item_id::text,''),
		       COALESCE(curriculum_unit_id::text,''), route_status
		FROM mined_items WHERE id = $1 AND user_id = $2`, minedID, userID).Scan(
		&user, &jobID, &sourceType, &c.SurfaceText, &c.Lemma, &c.NormalizedText,
		&lang, &c.PartOfSpeech, &c.Translation, &c.Definition, &contextSentence,
		&c.CEFRLevel, &c.Confidence, &c.TeachabilityScore, &c.IsChunk, &c.IsProperNoun,
		&c.GrammarTags, &c.CurriculumLexicalID, &c.CurriculumUnitID, &c.RouteStatus)
	if err != nil {
		return nil, err
	}
	c.UserID = user
	c.Language = lang
	c.ContextSentence = contextSentence
	c.SourceType = sourceType

	card, err := s.addCardFromMinedItem(ctx, &c, minedID)
	if err != nil {
		return nil, err
	}
	// mark item accepted
	_, _ = s.db.ExecContext(ctx, `UPDATE mined_items SET status = 'accepted' WHERE id = $1`, minedID)
	return card, nil
}

// IgnoreMinedItem marks a candidate as ignored.
func (s *WordMiningService) IgnoreMinedItem(ctx context.Context, userID, minedID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE mined_items SET status = 'ignored' WHERE id = $1 AND user_id = $2`, minedID, userID)
	return err
}

// GetCandidateItems returns mined candidates (or another status) for a language.
func (s *WordMiningService) GetCandidateItems(ctx context.Context, userID, targetLang, status string) ([]models.MinedItem, error) {
	if status == "" {
		status = "candidate"
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id::text, user_id, COALESCE(job_id::text,''), COALESCE(chat_id::text,''),
		       COALESCE(message_id::text,''), source_type, surface_text, lemma, normalized_text,
		       language, part_of_speech, translation, definition, context_sentence, text_span,
		       COALESCE(cefr_level,''), confidence, teachability_score, is_chunk, is_proper_noun,
		       grammar_tags, COALESCE(curriculum_lexical_item_id::text,''),
		       COALESCE(curriculum_unit_id::text,''), route_status, status, route_reason,
		       created_at, updated_at
		FROM mined_items
		WHERE user_id = $1 AND language = $2 AND status = $3
		ORDER BY teachability_score DESC, created_at DESC`,
		userID, targetLang, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []models.MinedItem
	for rows.Next() {
		var it models.MinedItem
		var span []byte
		if err := rows.Scan(
			&it.ID, &it.UserID, &it.JobID, &it.ChatID, &it.MessageID, &it.SourceType,
			&it.SurfaceText, &it.Lemma, &it.NormalizedText, &it.Language, &it.PartOfSpeech,
			&it.Translation, &it.Definition, &it.ContextSentence, &span, &it.CEFRLevel,
			&it.Confidence, &it.TeachabilityScore, &it.IsChunk, &it.IsProperNoun,
			pq.Array(&it.GrammarTags), &it.CurriculumLexicalID, &it.CurriculumUnitID,
			&it.RouteStatus, &it.Status, &it.RouteReason, &it.CreatedAt, &it.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// SaveManualVocabulary creates/refreshes a vocabulary card directly from a
// message (the tap-to-save flow). It bypasses the candidate state.
func (s *WordMiningService) SaveManualVocabulary(ctx context.Context, userID, term, lang, nativeLang string, messageID string) (*VocabularyCard, error) {
	term = strings.TrimSpace(term)
	if term == "" {
		return nil, fmt.Errorf("term is required")
	}
	var messageText, chatID string
	_ = s.db.QueryRowContext(ctx, `SELECT text, chat_id::text FROM messages WHERE id = $1 AND sender_id = $2`, messageID, userID).Scan(&messageText, &chatID)

	c := &MinedCandidate{
		SurfaceText: term,
		Lemma:       term,
		Language:    lang,
		PartOfSpeech: "unknown",
		CEFRLevel:   "A1",
		Confidence:  0.9,
		SourceType:  "chat",
		UserID:      userID,
	}
	c.NormalizedText = NormalizeLearningTerm(term, lang)
	if msg, err := s.getMessageContext(ctx, messageID); err == nil {
		messageText = msg.Text
		chatID = msg.ChatID
	}
	c.ContextSentence = contextSentence(messageText, term)
	if s.translation != nil {
		if v, err := s.translation.Translate(term, nativeLang); err == nil {
			c.Translation = v
			c.Definition = v
		}
	}
	// classify
	routed, err := s.classifyAndRoute(ctx, userID, "chat", chatID, messageID, messageText, *c, lang, nativeLang)
	if err != nil {
		routed = c
	}
	return s.addCardFromMinedItem(ctx, routed, "")
}

func (s *WordMiningService) getMessageContext(ctx context.Context, messageID string) (*models.Message, error) {
	m := &models.Message{}
	var translations []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT id::text, chat_id::text, sender_id::text, text, created_at
		FROM messages WHERE id = $1`, messageID).Scan(&m.ID, &m.ChatID, &m.SenderID, &m.Text, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	_ = translations
	return m, nil
}

// addCardFromMinedItem inserts (or updates) a vocabulary row for a mined item.
func (s *WordMiningService) addCardFromMinedItem(ctx context.Context, c *MinedCandidate, minedID string) (*VocabularyCard, error) {
	// dedupe safety
	existing := ""
	_ = s.db.QueryRowContext(ctx, `
		SELECT id::text FROM vocabulary WHERE user_id = $1 AND language = $2 AND normalized_term = $3`,
		c.UserID, c.Language, c.NormalizedText).Scan(&existing)
	if existing != "" {
		return s.loadCard(ctx, c.UserID, existing)
	}

	var id string
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO vocabulary (
			user_id, term, language, translation, definition, lemma, normalized_term,
			part_of_speech, is_chunk, source_type, source_message_id, cefr_level,
			curriculum_unit_id, route_status, mastery_stage, mastery_state,
			ease_factor, lapses, stage_success_count, production_success_count,
			spontaneous_use_count, teachability_score, confidence,
			context_sentence, context_message_id, context_chat_id,
			next_review, interval_days, first_seen_at, last_seen_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,1,'new',2.5,0,0,0,0,$15,$16,$17,$18,$19,CURRENT_TIMESTAMP,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
		RETURNING id::text`,
		c.UserID, c.SurfaceText, c.Language, c.Translation, c.Definition, c.Lemma, c.NormalizedText,
		c.PartOfSpeech, c.IsChunk, c.SourceType, nullStr(c.SourceMessageID), nullStr(c.CEFRLevel),
		nullStr(c.CurriculumUnitID), c.RouteStatus, c.TeachabilityScore, c.Confidence,
		c.ContextSentence, nullStr(c.MessageID), nullStr(c.ChatID)).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("add card: %w", err)
	}
	// record a vocabulary_sources row
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO vocabulary_sources (vocabulary_id, source_type, source_id, sentence)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (vocabulary_id, source_type, source_id) DO UPDATE SET seen_count = vocabulary_sources.seen_count + 1`,
		id, c.SourceType, nullStr(c.MessageID), c.ContextSentence)
	return s.loadCard(ctx, c.UserID, id)
}

func (s *WordMiningService) loadCard(ctx context.Context, userID, cardID string) (*VocabularyCard, error) {
	var c VocabularyCard
	var sourceMsg, cefr, unitID, ctxMsg, ctxChat sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id::text, user_id, term, language, COALESCE(translation,''), COALESCE(definition,''),
		       COALESCE(lemma,''), COALESCE(normalized_term,''), COALESCE(part_of_speech,'unknown'),
		       is_chunk, source_type, COALESCE(source_message_id::text,''),
		       COALESCE(cefr_level,''), COALESCE(curriculum_unit_id::text,''),
		       route_status, mastery_stage, mastery_state, COALESCE(ease_factor,2.5),
		       lapses, stage_success_count, production_success_count, spontaneous_use_count,
		       teachability_score, confidence, review_count, correct_count,
		       COALESCE(interval_days,1), next_review, COALESCE(context_sentence,''),
		       COALESCE(context_message_id::text,''), COALESCE(context_chat_id::text,''),
		       created_at, first_seen_at, last_seen_at
		FROM vocabulary WHERE id = $1 AND user_id = $2`, cardID, userID).Scan(
		&c.ID, &c.UserID, &c.Term, &c.Language, &c.Translation, &c.Definition,
		&c.Lemma, &c.NormalizedTerm, &c.PartOfSpeech, &c.IsChunk, &c.SourceType,
		&sourceMsg, &cefr, &unitID, &c.RouteStatus, &c.MasteryStage, &c.MasteryState,
		&c.EaseFactor, &c.Lapses, &c.StageSuccessCount, &c.ProductionSuccessCount,
		&c.SpontaneousUseCount, &c.TeachabilityScore, &c.Confidence, &c.ReviewCount,
		&c.CorrectCount, &c.IntervalDays, &c.NextReview, &c.ContextSentence,
		&ctxMsg, &ctxChat, &c.CreatedAt, &c.FirstSeenAt, &c.LastSeenAt)
	if err != nil {
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
	return &c, nil
}

// contextSentence finds the sentence containing a target term for SRS context.
func contextSentence(text, term string) string {
	if text == "" {
		return ""
	}
	lower := strings.ToLower(text)
	search := strings.ToLower(term)
	if idx := strings.Index(lower, search); idx >= 0 {
		s := text[max(0, idx-40):min(len(text), idx+len(term)+60)]
		return strings.TrimSpace(s)
	}
	if len(text) > 100 {
		return strings.TrimSpace(text[:100]) + "..."
	}
	return text
}

func isStopword(term, lang string) bool {
	term = NormalizeLearningTerm(term, lang)
	_, ok := spanishStopwords[term]
	return ok
}

var spanishStopwords = map[string]bool{
	"de": true, "la": true, "que": true, "el": true, "en": true, "y": true,
	"a": true, "los": true, "del": true, "se": true, "las": true, "por": true,
	"para": true, "con": true, "no": true, "una": true, "su": true, "al": true,
	"lo": true, "como": true, "más": true, "pero": true, "sus": true, "le": true,
	"ya": true, "o": true, "este": true, "sí": true, "porque": true, "esta": true,
	"entre": true, "cuando": true, "muy": true, "sin": true, "sobre": true,
	"también": true, "me": true, "hasta": true, "hay": true, "donde": true,
	"quien": true, "desde": true, "todo": true, "nos": true, "durante": true,
	"todos": true, "uno": true, "les": true, "ni": true, "contra": true,
	"otros": true, "ese": true, "eso": true, "ante": true, "ellos": true,
	"e": true, "mi": true, "antes": true, "algunos": true, "qué": true,
	"un": true, "son": true, "está": true, "estoy": true, "he": true, "ha": true,
	"tengo": true, "es": true,
}

func stringSlice(s []string) interface{} {
	if len(s) == 0 {
		return []string{}
	}
	return s
}
