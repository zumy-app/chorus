package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"

	"github.com/chorus/messenger/internal/models"
)

// FluencyScoreService computes the learner's readiness-to-next-CEFR score from
// its components (curriculum completion, vocabulary depth, grammar mastery,
// scenario completion, consistency) and applies level-up/level-down transitions
// when a learner crosses a confidence threshold.
type FluencyScoreService struct {
	db *sql.DB
}

func NewFluencyScoreService(db *sql.DB) *FluencyScoreService {
	return &FluencyScoreService{db: db}
}

// Recalc recomputes and stores the readiness score + a snapshot for a user/pair.
func (s *FluencyScoreService) Recalc(ctx context.Context, userID, targetLang, nativeLang string) (*models.FluencySummary, error) {
	if nativeLang == "" {
		nativeLang = "en"
	}
	var level string
	var activeCourseID, activeUnitID string
	_ = s.db.QueryRowContext(ctx, `
		SELECT current_cefr_level, COALESCE(active_course_id::text,''), COALESCE(active_unit_id::text,'')
		FROM user_language_profiles
		WHERE user_id = $1 AND target_language = $2 AND native_language = $3`,
		userID, targetLang, nativeLang).Scan(&level, &activeCourseID, &activeUnitID)

	if activeCourseID == "" {
		// Fall back to a profile-derived base score.
		var score int
		_ = s.db.QueryRowContext(ctx, `SELECT readiness_score FROM user_language_profiles
			WHERE user_id = $1 AND target_language = $2 AND native_language = $3`,
			userID, targetLang, nativeLang).Scan(&score)
		return &models.FluencySummary{ReadinessScore: score, ReadinessPercent: score / 10, Label: fluencyLabel(level, score), ComponentScores: map[string]int{}}, nil
	}

	components := map[string]int{
		"curriculum":  s.curriculumScore(ctx, userID, activeCourseID, level),
		"vocabulary":  s.vocabularyScore(ctx, userID, targetLang, activeCourseID, level),
		"grammar":     s.grammarScore(ctx, userID, targetLang, activeCourseID, level),
		"scenarios":   s.scenarioScore(ctx, userID, targetLang, activeCourseID, level),
		"consistency": s.consistencyScore(ctx, userID, targetLang),
	}

	readiness := int(math.Round(
		0.30*float64(components["curriculum"]) +
			0.25*float64(components["vocabulary"]) +
			0.20*float64(components["grammar"]) +
			0.15*float64(components["scenarios"]) +
			0.10*float64(components["consistency"])))
	if readiness > 1000 {
		readiness = 1000
	}
	if readiness < 0 {
		readiness = 0
	}

	summary := models.FluencySummary{
		ReadinessScore:   readiness,
		ReadinessPercent: readiness / 10,
		Label:            fluencyLabel(level, readiness),
		ComponentScores:  components,
	}

	_, _ = s.db.ExecContext(ctx, `
		UPDATE user_language_profiles SET readiness_score = $4, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND target_language = $2 AND native_language = $3`,
		userID, targetLang, nativeLang, readiness)

	compJSON, _ := json.Marshal(components)
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO fluency_score_snapshots (user_id, target_language, current_cefr_level, readiness_score, component_scores)
		VALUES ($1, $2, $3, $4, $5)`,
		userID, targetLang, level, readiness, string(compJSON))

	return &summary, nil
}

// CurriculumScore: weighted completion of the current level's units.
func (s *FluencyScoreService) curriculumScore(ctx context.Context, userID, courseID, level string) int {
	var score float64
	_ = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(AVG(progress_pct), 0) FROM user_unit_progress p
		JOIN curriculum_units u ON u.id = p.unit_id
		WHERE p.user_id = $1 AND u.course_id = $2 AND u.cefr_level = $3`,
		userID, courseID, level).Scan(&score)
	// scale 0-100 -> 0-1000
	return int(math.Round(score * 10))
}

// VocabularyScore: percent of level target lexical items at stage 3+ plus a
// production bonus.
func (s *FluencyScoreService) vocabularyScore(ctx context.Context, userID, targetLang, courseID, level string) int {
	var total, mastered int
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE mastery_stage >= 3)
		FROM vocabulary WHERE user_id = $1 AND language = $2
		  AND (curriculum_unit_id IS NULL OR curriculum_unit_id IN (
			SELECT id FROM curriculum_units WHERE course_id = $3 AND cefr_level = $4
		  ))`, userID, targetLang, courseID, level).Scan(&total, &mastered)
	if total == 0 {
		// credit any vocabulary at all (recently mined) for a fresh learner
		var anyCount int
		_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM vocabulary WHERE user_id = $1 AND language = $2`, userID, targetLang).Scan(&anyCount)
		if anyCount > 0 {
			return 300
		}
		return 0
	}
	pct := float64(mastered) / float64(total)
	return int(math.Round(pct * 1000))
}

// GrammarScore: average user_grammar_mastery confidence for the level's points.
func (s *FluencyScoreService) grammarScore(ctx context.Context, userID, targetLang, courseID, level string) int {
	var avg float64
	_ = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(AVG(m.confidence), 0) FROM user_grammar_mastery m
		JOIN grammar_points g ON g.id = m.grammar_point_id
		WHERE m.user_id = $1 AND m.target_language = $2 AND g.course_id = $3 AND g.cefr_level = $4`,
		userID, targetLang, courseID, level).Scan(&avg)
	return int(math.Round(avg * 1000))
}

// ScenarioScore: required scenario completion for the level.
func (s *FluencyScoreService) scenarioScore(ctx context.Context, userID, targetLang, courseID, level string) int {
	var total, done int
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE EXISTS (
			SELECT 1 FROM scenario_runs r WHERE r.user_id = $3 AND r.scenario_id = sc.id AND r.status = 'completed'
		))
		FROM scenario_scripts sc WHERE sc.course_id = $1 AND sc.cefr_level = $2`,
		courseID, level, userID).Scan(&total, &done)
	if total == 0 {
		return 500 // no scenarios required yet — neutral
	}
	return int(math.Round(float64(done) / float64(total) * 1000))
}

// ConsistencyScore: recent active days + daily goal completion.
func (s *FluencyScoreService) consistencyScore(ctx context.Context, userID, targetLang string) int {
	var activeDays int
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM daily_learning_stats
		WHERE user_id = $1 AND target_language = $2 AND activity_date >= CURRENT_DATE - INTERVAL '14 days'`,
		userID, targetLang).Scan(&activeDays)
	if activeDays == 0 {
		return 0
	}
	if activeDays >= 7 {
		return 1000
	}
	if activeDays >= 3 {
		return 700
	}
	return 400
}

// ApplyLevelTransition moves a profile up a band when readiness >= 900 and the
// level's checkpoint unit is passed.
func (s *FluencyScoreService) ApplyLevelTransition(ctx context.Context, userID, targetLang, nativeLang string) error {
	if nativeLang == "" {
		nativeLang = "en"
	}
	var level string
	var score int
	_ = s.db.QueryRowContext(ctx, `
		SELECT current_cefr_level, readiness_score FROM user_language_profiles
		WHERE user_id = $1 AND target_language = $2 AND native_language = $3`,
		userID, targetLang, nativeLang).Scan(&level, &score)
	if score < 900 {
		return nil
	}
	next := map[string]string{"A1": "A2", "A2": "B1", "B1": "B2"}[level]
	if next == "" || level == "B2" {
		return nil
	}
	var checkpointPassed bool
	_ = s.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM user_unit_progress p JOIN curriculum_units u ON u.id = p.unit_id
			WHERE p.user_id = $1 AND u.cefr_level = $2 AND u.checkpoint_required AND p.status = 'completed'
		)`, userID, level).Scan(&checkpointPassed)
	if !checkpointPassed {
		return nil
	}
	_, _ = s.db.ExecContext(ctx, `
		UPDATE user_language_profiles SET current_cefr_level = $4, readiness_score = 0, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND target_language = $2 AND native_language = $3`,
		userID, targetLang, nativeLang, next)
	return nil
}
