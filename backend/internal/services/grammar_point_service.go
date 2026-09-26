package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

const (
	// grammarLessonCacheTTL bounds the micro-lesson Redis cache. Lesson content
	// is deploy-synced (seeded via embedded SQL), so a 24h TTL is safe and keeps
	// drill opens off Postgres entirely on the hot path.
	grammarLessonCacheTTL = 24 * time.Hour
)

// GrammarPointService powers the V3 grammar system: due drill selection driven
// by the user_grammar_mastery SRS ledger, micro-lesson content, and attempt
// recording with SM-2 interval updates.
type GrammarPointService struct {
	db    *sql.DB
	redis *redis.Client
}

func NewGrammarPointService(db *sql.DB, rdb *redis.Client) *GrammarPointService {
	return &GrammarPointService{db: db, redis: rdb}
}

// GetDueItems returns drill items for grammar points due for review (or never
// seen) in the user's active course. Unseen points are introduced first, then
// overdue reviews, then the rest ordered by CEFR level so a session always has
// teachable material.
func (s *GrammarPointService) GetDueItems(ctx context.Context, userID, targetLang string, limit int) ([]models.GrammarDrillItem, error) {
	if limit <= 0 {
		limit = 5
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT gi.id::text, gi.grammar_point_id::text, gp.title, gp.cefr_level, gi.ordinal, gi.item_type,
		       gi.prompt, COALESCE(gi.sentence_with_blank,''), gi.choices, gi.correct, gi.note
		FROM grammar_items gi
		JOIN grammar_points gp ON gp.id = gi.grammar_point_id
		JOIN curriculum_courses c ON c.id = gp.course_id
		LEFT JOIN user_grammar_mastery m ON m.grammar_point_id = gi.grammar_point_id AND m.user_id = $1
		WHERE c.target_language = $2 AND gi.is_active = true
		ORDER BY (m.next_review_at IS NULL) DESC,
		         CASE WHEN m.next_review_at IS NULL THEN 0
		              WHEN m.next_review_at <= CURRENT_TIMESTAMP THEN 1 ELSE 2 END,
		         m.next_review_at ASC,
		         gp.cefr_level, gp.ordinal, gi.ordinal
		LIMIT $3`, userID, targetLang, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []models.GrammarDrillItem{}
	usedPoints := map[string]int{}
	for rows.Next() {
		var it models.GrammarDrillItem
		var choices pq.StringArray
		if err := rows.Scan(&it.ID, &it.GrammarPointID, &it.PointTitle, &it.CEFRLevel, &it.Ordinal, &it.ItemType,
			&it.Prompt, &it.SentenceWithBlank, &choices, &it.Correct, &it.Note); err != nil {
			return nil, err
		}
		// At most two drill items per point per session so the learner always
		// sees breadth across points before depth.
		if usedPoints[it.GrammarPointID] >= 2 {
			continue
		}
		usedPoints[it.GrammarPointID]++
		it.Choices = choices
		items = append(items, it)
	}
	return items, rows.Err()
}

// GetMicroLesson returns the teachable card for a grammar point (rule, common
// trap, examples, prerequisites). Cached in Redis: content is deploy-synced so
// the cache never goes stale within a deployment.
func (s *GrammarPointService) GetMicroLesson(ctx context.Context, pointID string) (*models.GrammarMicroLesson, error) {
	cacheKey := fmt.Sprintf("grammar:lesson:%s", pointID)
	if s.redis != nil {
		if cached, err := s.redis.Get(ctx, cacheKey).Result(); err == nil && cached != "" {
			var lesson models.GrammarMicroLesson
			if json.Unmarshal([]byte(cached), &lesson) == nil {
				return &lesson, nil
			}
		}
	}

	var lesson models.GrammarMicroLesson
	var examples []byte
	var shortExplanation, ruleText string
	var prereqs pq.StringArray
	err := s.db.QueryRowContext(ctx, `
		SELECT id::text, title, cefr_level, COALESCE(short_explanation,''), COALESCE(rule_text,''),
		       COALESCE(common_trap,''), examples, prerequisites
		FROM grammar_points WHERE id = $1`, pointID).
		Scan(&lesson.PointID, &lesson.Title, &lesson.CEFRLevel, &shortExplanation, &ruleText,
			&lesson.CommonTrap, &examples, &prereqs)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("grammar point not found")
	}
	if err != nil {
		return nil, err
	}
	lesson.RuleText = ruleText
	if lesson.RuleText == "" {
		lesson.RuleText = shortExplanation
	}
	if len(examples) > 0 {
		var ex any
		if json.Unmarshal(examples, &ex) == nil {
			lesson.Examples = ex
		}
	}
	lesson.Prerequisites = prereqs

	if s.redis != nil {
		if data, err := json.Marshal(lesson); err == nil {
			s.redis.Set(ctx, cacheKey, data, grammarLessonCacheTTL)
		}
	}
	return &lesson, nil
}

// RecordItemAttempt stores a drill attempt and advances the point's SM-2 ledger
// row (ease factor, repetitions, interval, next review) in a single upsert.
func (s *GrammarPointService) RecordItemAttempt(ctx context.Context, userID, itemID string, correct bool, quality int) error {
	var pointID, targetLang string
	err := s.db.QueryRowContext(ctx, `
		SELECT gi.grammar_point_id::text, c.target_language
		FROM grammar_items gi
		JOIN grammar_points gp ON gp.id = gi.grammar_point_id
		JOIN curriculum_courses c ON c.id = gp.course_id
		WHERE gi.id = $1`, itemID).Scan(&pointID, &targetLang)
	if err != nil {
		return fmt.Errorf("grammar item %s: %w", itemID, err)
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO user_grammar_item_attempts (user_id, grammar_item_id, grammar_point_id, correct, quality)
		VALUES ($1, $2, $3, $4, $5)`, userID, itemID, pointID, correct, quality); err != nil {
		return err
	}
	return s.updateMastery(ctx, userID, pointID, targetLang, correct, quality)
}

// updateMastery applies an SM-2 step to the user_grammar_mastery ledger.
func (s *GrammarPointService) updateMastery(ctx context.Context, userID, pointID, targetLang string, correct bool, quality int) error {
	if quality < 0 {
		quality = 0
	}
	if quality > 5 {
		quality = 5
	}

	// Current state (0-valued when the point has never been seen).
	var (
		ease         = 2.5
		reps         = 0
		interval     = 0
		masteryStage = 1
	)
	var stage sql.NullInt64
	var easeN sql.NullFloat64
	var repsN, intN sql.NullInt64
	_ = s.db.QueryRowContext(ctx, `
		SELECT mastery_stage, ease_factor, repetitions, interval_days
		FROM user_grammar_mastery WHERE user_id = $1 AND grammar_point_id = $2`,
		userID, pointID).Scan(&stage, &easeN, &repsN, &intN)
	if stage.Valid {
		masteryStage = int(stage.Int64)
	}
	if easeN.Valid {
		ease = easeN.Float64
	}
	if repsN.Valid {
		reps = int(repsN.Int64)
	}
	if intN.Valid {
		interval = int(intN.Int64)
	}

	// SM-2 step.
	if quality >= 3 {
		reps++
		ease = ease + (0.1 - float64(5-quality)*(0.08+float64(5-quality)*0.02))
		if ease < 1.3 {
			ease = 1.3
		}
		switch {
		case reps == 1:
			interval = 1
		case reps == 2:
			interval = 6
		default:
			interval = int(math.Round(float64(interval) * ease))
			if interval < 1 {
				interval = 1
			}
		}
		if masteryStage < 5 {
			masteryStage++
		}
	} else {
		reps = 0
		interval = 1
		if masteryStage > 1 {
			masteryStage--
		}
	}
	nextReview := time.Now().AddDate(0, 0, interval)

	correctInc, errorInc := 0, 1
	if correct {
		correctInc, errorInc = 1, 0
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_grammar_mastery (
			user_id, grammar_point_id, target_language, confidence, seen_count, correct_count, error_count,
			mastery_stage, ease_factor, repetitions, interval_days, next_review_at, last_reviewed_at
		) VALUES ($1, $2, $3, $4, 1, $5, $6, $7, $8, $9, $10, $11, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id, grammar_point_id) DO UPDATE SET
			seen_count = user_grammar_mastery.seen_count + 1,
			correct_count = user_grammar_mastery.correct_count + $5,
			error_count = user_grammar_mastery.error_count + $6,
			confidence = LEAST(1.0, GREATEST(0.0,
				(user_grammar_mastery.correct_count + $5)::float / GREATEST(1, user_grammar_mastery.seen_count + 1))),
			mastery_stage = $7,
			ease_factor = $8,
			repetitions = $9,
			interval_days = $10,
			next_review_at = $11,
			last_reviewed_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP`,
		userID, pointID, targetLang, 0.3, correctInc, errorInc, masteryStage, ease, reps, interval, nextReview)
	return err
}

// GradeDrillAnswer grades a learner answer against a drill item using the same
// level-aware normalization as placement (accent-forgiving at A1/A2, strict at
// B1+).
func GradeDrillAnswer(item models.GrammarDrillItem, answer, cefrLevel string) bool {
	a := collapseWhitespace(strings.ToLower(strings.TrimSpace(answer)))
	if a == "" {
		return false
	}
	e := collapseWhitespace(strings.ToLower(strings.TrimSpace(item.Correct)))
	if cefrLevel == "A1" || cefrLevel == "A2" {
		return stripDiacritics(a) == stripDiacritics(e)
	}
	return a == e
}
