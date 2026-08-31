package services

import (
	"context"
	"database/sql"
	"time"

	"github.com/chorus/messenger/internal/models"
)

// Seed queue tuning. seedCardLimit is the max number of curated curriculum
// lexical items materialized into a learner's vocabulary queue; seedInitialDue
// of them are due immediately and the rest are staggered a day at a time so the
// queue presents a drip-feed instead of a wall of new cards. seedUnitWindow is
// how many curriculum units (starting at the active unit) provide seed items.
const (
	seedCardLimit  = 24
	seedUnitWindow = 3
	seedInitialDue = 6
	seedSourceType = "lesson"
	seedTeachScore = 70.0
)

// SeedQueueService implements the FR-32 "seed path": it idempotently
// materializes curated curriculum lexical items for a (native->target) pair at
// the learner's level into the learner's vocabulary rows (source_type='lesson',
// curriculum_unit_id / cefr_level set). Once materialized they flow through the
// normal spaced-repetition pipeline (GetDueCards / GetNewCards / SRS updates),
// so a fresh learner with zero mined words still has a real queue.
type SeedQueueService struct {
	db           *sql.DB
	profiles     *LearningProfileService
	capabilities *LearningCapabilityService
}

func NewSeedQueueService(db *sql.DB, profiles *LearningProfileService, capabilities *LearningCapabilityService) *SeedQueueService {
	return &SeedQueueService{db: db, profiles: profiles, capabilities: capabilities}
}

// EnsureSeeded makes sure the learner has seeded vocabulary cards for the pair,
// returning how many new seed cards it added. It is safe to call on every
// session/queue read: it is idempotent (existing normalized terms are skipped).
func (s *SeedQueueService) EnsureSeeded(ctx context.Context, userID, nativeLang, targetLang string) (int, error) {
	targetLang = normalizeLang(targetLang)
	nativeLang = normalizeLang(nativeLang)
	if nativeLang == "" {
		nativeLang = "en"
	}

	cap, err := s.capabilities.GetCapability(ctx, nativeLang, targetLang)
	if err != nil {
		return 0, err
	}
	if !cap.SRSEnabled {
		return 0, nil
	}
	// Seed sequences are only defined for curated (course-backed) pairs.
	if cap.SupportTier != string(models.LearningSupportFullCourse) &&
		cap.SupportTier != string(models.LearningSupportBetaAIAssisted) {
		return 0, nil
	}
	courseID := cap.ActiveCourseID
	if courseID == "" {
		return 0, nil
	}

	profile, err := s.profiles.GetProfile(ctx, userID, targetLang, nativeLang)
	if err != nil {
		return 0, err
	}

	unitIDs, err := s.seedUnits(ctx, courseID, profile.ActiveUnitID)
	if err != nil {
		return 0, err
	}
	if len(unitIDs) == 0 {
		return 0, nil
	}

	return s.addSeedCards(ctx, userID, targetLang, courseID, unitIDs)
}

// seedUnits picks the unit ids to seed from: the learner's active unit (when
// set) and the next seedUnitWindow-1 units in the course, else the first
// seedUnitWindow units. Ordered by curriculum ordinal.
func (s *SeedQueueService) seedUnits(ctx context.Context, courseID, activeUnitID string) ([]string, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if activeUnitID != "" {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id::text FROM curriculum_units
			WHERE course_id = $1
			  AND (id = $2 OR ordinal > (SELECT ordinal FROM curriculum_units WHERE id = $2))
			ORDER BY ordinal
			LIMIT $3`, courseID, activeUnitID, seedUnitWindow)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id::text FROM curriculum_units
			WHERE course_id = $1
			ORDER BY ordinal
			LIMIT $2`, courseID, seedUnitWindow)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// addSeedCards inserts up to seedCardLimit curated lexical items from the given
// units that the learner does not already have, staggering their next_review.
func (s *SeedQueueService) addSeedCards(ctx context.Context, userID, targetLang, courseID string, unitIDs []string) (int, error) {
	existing, err := s.existingSeedTerms(ctx, userID, targetLang, unitIDs)
	if err != nil {
		return 0, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT li.id::text, COALESCE(li.unit_id::text,''), li.lemma, li.display_text,
		       li.part_of_speech, li.cefr_level, li.frequency_rank, li.is_chunk,
		       COALESCE(li.translations->>'en','')
		FROM lexical_items li
		WHERE li.course_id = $1 AND li.unit_id = ANY($2)
		ORDER BY li.frequency_rank NULLS LAST, li.lemma
		LIMIT $3`, courseID, unitIDs, seedCardLimit)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type seedCandidate struct {
		id, unitID, lemma, display, pos, cefr, en string
		freq                                      int
		isChunk                                   bool
	}
	cands := []seedCandidate{}
	for rows.Next() {
		var c seedCandidate
		var freq sql.NullInt64
		if err := rows.Scan(&c.id, &c.unitID, &c.lemma, &c.display, &c.pos, &c.cefr, &freq, &c.isChunk, &c.en); err != nil {
			return 0, err
		}
		c.freq = int(freq.Int64)
		cands = append(cands, c)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	added := 0
	seen := map[string]bool{}
	now := time.Now()
	for i, c := range cands {
		norm := NormalizeLearningTerm(c.display, targetLang)
		if norm == "" || seen[norm] {
			continue
		}
		if _, ok := existing[norm]; ok {
			continue
		}
		seen[norm] = true
		if _, err := s.insertSeedCard(ctx, userID, targetLang, c.id, c.unitID, c.display, c.lemma, c.pos, c.cefr, c.en, c.isChunk, staggerSeedReview(now, i)); err != nil {
			continue
		}
		added++
	}
	return added, nil
}

// existingSeedTerms returns the normalized terms the learner already has cards
// for within the seed unit window, so EnsureSeeded does not duplicate them.
func (s *SeedQueueService) existingSeedTerms(ctx context.Context, userID, targetLang string, unitIDs []string) (map[string]bool, error) {
	set := map[string]bool{}
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT normalized_term FROM vocabulary
		WHERE user_id = $1 AND language = $2 AND curriculum_unit_id = ANY($3)`,
		userID, targetLang, unitIDs)
	if err != nil {
		return set, err
	}
	defer rows.Close()
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return set, err
		}
		if t != "" {
			set[t] = true
		}
	}
	return set, rows.Err()
}

// insertSeedCard creates one vocabulary card from a curated lexical item. It
// returns false (no error) when a card with the same normalized term already
// exists, so the operation stays idempotent.
func (s *SeedQueueService) insertSeedCard(ctx context.Context, userID, targetLang, lexicalID, unitID, display, lemma, pos, cefr, translation string, isChunk bool, nextReview time.Time) (bool, error) {
	norm := NormalizeLearningTerm(display, targetLang)
	if norm == "" {
		return false, nil
	}
	var existingID string
	_ = s.db.QueryRowContext(ctx, `
		SELECT id::text FROM vocabulary
		WHERE user_id = $1 AND language = $2 AND normalized_term = $3`,
		userID, targetLang, norm).Scan(&existingID)
	if existingID != "" {
		return false, nil
	}

	var id string
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO vocabulary (
			user_id, term, language, translation, definition, lemma, normalized_term,
			part_of_speech, is_chunk, source_type, source_message_id, cefr_level,
			curriculum_lexical_item_id, curriculum_unit_id, route_status,
			mastery_stage, mastery_state, ease_factor, lapses, stage_success_count,
			production_success_count, spontaneous_use_count, teachability_score, confidence,
			context_sentence, next_review, interval_days, first_seen_at, last_seen_at
		)
		VALUES ($1,$2,$3,$4,$4,$5,$6,$7,$8,$9,NULL,$10,$11,$12,'current_unit',
			1,'new',2.50,0,0,0,0,$13,0.60,'',$14,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
		ON CONFLICT (user_id, term, language) DO NOTHING
		RETURNING id::text`,
		userID, display, targetLang, translation, lemma, norm,
		pos, translation, seedSourceType, lexicalID, unitID,
		seedTeachScore, nextReview).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil // conflict -> already seeded
	}
	if err != nil {
		return false, err
	}
	return id != "", nil
}

// staggerSeedReview makes the first seedInitialDue cards due now, then spaces
// the rest one day at a time so the queue drips new seed material.
func staggerSeedReview(now time.Time, idx int) time.Time {
	if idx < seedInitialDue {
		return now
	}
	return now.AddDate(0, 0, idx-seedInitialDue+1)
}
