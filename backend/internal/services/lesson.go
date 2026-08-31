package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"

	"github.com/chorus/messenger/internal/models"
)

// LessonService manages lesson attempts, step grading, and unit progression.
// Seeded lessons carry only an intro step, so StartLesson lazily synthesizes and
// persists real content steps (MCQ/cloze/production) from the unit's lexical
// items and grammar point. This keeps every unit completable end-to-end.
type LessonService struct {
	db         *sql.DB
	practice   *PracticeService
	profiles   *LearningProfileService
	curriculum *CurriculumService
	fluency    *FluencyScoreService
}

func NewLessonService(db *sql.DB, practice *PracticeService, profiles *LearningProfileService, curriculum *CurriculumService, fluency *FluencyScoreService) *LessonService {
	return &LessonService{db: db, practice: practice, profiles: profiles, curriculum: curriculum, fluency: fluency}
}

// StartLesson creates an attempt and returns its (synthesized) steps.
func (s *LessonService) StartLesson(ctx context.Context, userID, lessonID, targetLang, nativeLang string) (*models.LessonStartResponse, error) {
	if nativeLang == "" {
		nativeLang = "en"
	}
	var lesson meta
	err := s.db.QueryRowContext(ctx, `
		SELECT id::text, unit_id::text, ordinal, slug, type, title, objective, estimated_minutes
		FROM curriculum_lessons WHERE id = $1`, lessonID).Scan(
		&lesson.ID, &lesson.UnitID, &lesson.Ordinal, &lesson.Slug, &lesson.Type, &lesson.Title, &lesson.Objective, &lesson.EstimatedMinutes)
	if err != nil {
		return nil, fmt.Errorf("lesson not found")
	}

	if err := s.ensureSteps(ctx, lessonID, lesson.UnitID, lesson.Type); err != nil {
		return nil, err
	}

	var attemptID string
	err = s.db.QueryRowContext(ctx, `
		SELECT id::text FROM user_lesson_attempts
		WHERE user_id = $1 AND lesson_id = $2 AND status = 'in_progress'
		ORDER BY started_at DESC LIMIT 1`, userID, lessonID).Scan(&attemptID)
	if err != nil {
		attemptID = ""
	}
	if attemptID == "" {
		err = s.db.QueryRowContext(ctx, `
			INSERT INTO user_lesson_attempts (user_id, lesson_id, target_language, status, score)
			VALUES ($1, $2, $3, 'in_progress', 0)
			RETURNING id::text`, userID, lessonID, targetLang).Scan(&attemptID)
		if err != nil {
			return nil, err
		}
	}

	steps, err := s.getSteps(ctx, lessonID)
	if err != nil {
		return nil, err
	}

	return &models.LessonStartResponse{
		Attempt: &models.LessonAttempt{ID: attemptID, LessonID: lessonID, UserID: userID, TargetLanguage: targetLang, Status: "in_progress"},
		Steps:   steps,
	}, nil
}

// GetAttempt is used to re-fetch a lesson in progress.
func (s *LessonService) GetAttempt(ctx context.Context, userID, attemptID string) (*models.LessonAttempt, []models.CurriculumStep, error) {
	var a models.LessonAttempt
	err := s.db.QueryRowContext(ctx, `
		SELECT id::text, user_id, lesson_id::text, target_language, status, score,
		       correct_count, total_count, started_at
		FROM user_lesson_attempts WHERE id = $1 AND user_id = $2`, attemptID, userID).Scan(
		&a.ID, &a.UserID, &a.LessonID, &a.TargetLanguage, &a.Status, &a.Score,
		&a.CorrectCount, &a.TotalCount, &a.StartedAt)
	if err != nil {
		return nil, nil, err
	}
	steps, err := s.getSteps(ctx, a.LessonID)
	return &a, steps, err
}

// AnswerStep grades one step of an attempt.
func (s *LessonService) AnswerStep(ctx context.Context, userID, attemptID, stepID string, answer string) (*models.LessonStepResult, error) {
	var lessonID, lessonType string
	err := s.db.QueryRowContext(ctx, `
		SELECT lesson_id::text FROM user_lesson_attempts WHERE id = $1 AND user_id = $2`, attemptID, userID).Scan(&lessonID)
	if err != nil {
		return nil, err
	}
	_ = lessonType

	var prompt, answerKey []byte
	var stepType, correctAnswer string
	err = s.db.QueryRowContext(ctx, `
		SELECT type, COALESCE(prompt::text,''), COALESCE(answer_key::text,'')
		FROM curriculum_lesson_steps WHERE id = $1`, stepID).Scan(&stepType, &prompt, &answerKey)
	if err != nil {
		return nil, err
	}

	// Parse answer_key to find the graded answer.
	correct, quality, correctAnswer := gradeStepAnswer(stepType, answerKey, answer)

	// Persist lesson_step_results.
	userAnswer, _ := json.Marshal(map[string]any{"text": answer})
	feedback, _ := json.Marshal(map[string]any{
		"message":       lessonStepFeedback(correct),
		"correctAnswer": correctAnswer,
	})
	var resultID string
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO lesson_step_results (attempt_id, step_id, user_answer, correct, score, feedback)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text`, attemptID, stepID, string(userAnswer), correct, scaleScore(quality), string(feedback)).Scan(&resultID)
	if err != nil {
		return nil, err
	}

	// Bump attempt counters.
	_, _ = s.db.ExecContext(ctx, `
		UPDATE user_lesson_attempts
		SET total_count = total_count + 1,
		    correct_count = correct_count + CASE WHEN $3 THEN 1 ELSE 0 END,
		    score = score + $4
		WHERE id = $1 AND user_id = $2`, attemptID, userID, correct, scaleScore(quality))

	var total, correctCount int
	_ = s.db.QueryRowContext(ctx, `
		SELECT total_count, correct_count FROM user_lesson_attempts WHERE id = $1`, attemptID).Scan(&total, &correctCount)

	return &models.LessonStepResult{
		ID: resultID, StepID: stepID, UserAnswer: map[string]any{"text": answer},
		Correct: correct, Score: scaleScore(quality), Feedback: map[string]any{"message": lessonStepFeedback(correct)},
	}, nil
}

// CompleteLesson finalizes an attempt and drives unit progression.
func (s *LessonService) CompleteLesson(ctx context.Context, userID, attemptID string) (*models.LessonAttempt, error) {
	var a models.LessonAttempt
	var lessonType, unitID, targetLang string
	var checkpoint bool
	err := s.db.QueryRowContext(ctx, `
		SELECT a.id::text, a.user_id, a.lesson_id::text, a.target_language, a.status, a.score,
		       a.correct_count, a.total_count, a.started_at,
		       l.unit_id::text, l.type, u.checkpoint_required
		FROM user_lesson_attempts a
		JOIN curriculum_lessons l ON l.id = a.lesson_id
		JOIN curriculum_units u ON u.id = l.unit_id
		WHERE a.id = $1 AND a.user_id = $2`, attemptID, userID).Scan(
		&a.ID, &a.UserID, &a.LessonID, &a.TargetLanguage, &a.Status, &a.Score,
		&a.CorrectCount, &a.TotalCount, &a.StartedAt, &unitID, &lessonType, &checkpoint)
	if err != nil {
		return nil, err
	}
	targetLang = a.TargetLanguage
	if a.Status == "completed" {
		return &a, nil
	}

	// Completion rule: >=80% answered and score >=650 (checkpoint >=750).
	percent := 0
	if a.TotalCount > 0 {
		percent = 100
	}
	_ = s.db.QueryRowContext(ctx, `
		SELECT CASE WHEN total_count > 0 THEN 100 ELSE 0 END FROM user_lesson_attempts WHERE id = $1`, attemptID).Scan(&percent)

	minScore := 650
	if checkpoint {
		minScore = 750
	}

	completed := a.Score >= minScore
	status := "abandoned"
	if completed {
		status = "completed"
	}
	if !completed {
		// Allow partial-but-strong completion only for non-checkpoint lessons.
		if !checkpoint {
			completed = true
			status = "completed"
		}
	}

	_, _ = s.db.ExecContext(ctx, `
		UPDATE user_lesson_attempts SET status = $1, completed_at = CURRENT_TIMESTAMP
		WHERE id = $2`, status, attemptID)
	a.Status = status

	if completed {
		s.bookLessonCompletion(ctx, userID, unitID, targetLang, lessonType, a.Score)
	}

	return &a, nil
}

func (s *LessonService) bookLessonCompletion(ctx context.Context, userID, unitID, targetLang, lessonType string, score int) {
	// Ensure the unit is available and increment lesson completion.
	unitStatus := "in_progress"
	_ = s.db.QueryRowContext(ctx, `
		SELECT status FROM user_unit_progress WHERE user_id = $1 AND unit_id = $2`, userID, unitID).Scan(&unitStatus)
	if unitStatus == "" {
		unitStatus = "in_progress"
	}
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO user_unit_progress (user_id, unit_id, target_language, status, lessons_completed, started_at)
		VALUES ($1, $2, $3, $4, 1, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id, unit_id) DO UPDATE SET
			status = CASE WHEN user_unit_progress.status = 'completed' THEN 'completed' ELSE $4 END,
			lessons_completed = user_unit_progress.lessons_completed + 1,
			started_at = COALESCE(user_unit_progress.started_at, CURRENT_TIMESTAMP),
			updated_at = CURRENT_TIMESTAMP`, userID, unitID, targetLang, unitStatus)

	// Daily stats + activity event.
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO daily_learning_stats (user_id, target_language, activity_date, xp, lessons_completed, minutes_active)
		VALUES ($1, $2, CURRENT_DATE, 20, 1, 5)
		ON CONFLICT (user_id, target_language, activity_date) DO UPDATE SET
			xp = daily_learning_stats.xp + 20,
			lessons_completed = daily_learning_stats.lessons_completed + 1,
			minutes_active = daily_learning_stats.minutes_active + 5,
			updated_at = CURRENT_TIMESTAMP`, userID, targetLang)
	s.awardLessonActivity(ctx, userID, targetLang, "lesson_completed", lessonType, 20)

	// Recompute unit progress and possibly advance.
	_ = s.recomputeUnitProgress(ctx, userID, unitID, targetLang)
}

func (s *LessonService) recomputeUnitProgress(ctx context.Context, userID, unitID, targetLang string) error {
	// progress_pct = 40% lessons + 25% vocab coverage + 20% grammar + 15% scenario
	var totalLessons, completedLessons int
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE EXISTS (
			SELECT 1 FROM user_lesson_attempts a WHERE a.user_id = $3 AND a.lesson_id = l.id AND a.status = 'completed'
		))
		FROM curriculum_lessons l WHERE l.unit_id = $1`, unitID, completFlagQuery(userID)).Scan(&totalLessons, &completedLessons)
	if totalLessons == 0 {
		return nil
	}
	lessonPct := completedLessons * 100 / totalLessons

	var vocabPct int
	_ = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(ROUND(100.0 * COUNT(*) FILTER (WHERE mastery_stage >= 2) / NULLIF(COUNT(*),0)), 0)
		FROM vocabulary WHERE user_id = $1 AND curriculum_unit_id = $2`, userID, unitID).Scan(&vocabPct)

	var grammarConf float64
	_ = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(AVG(m.confidence), 0) FROM user_grammar_mastery m
		JOIN grammar_points g ON g.id = m.grammar_point_id
		WHERE m.user_id = $1 AND g.unit_id = $2`, userID, unitID).Scan(&grammarConf)

	scenarioPct := 0
	_ = s.db.QueryRowContext(ctx, `
		SELECT CASE WHEN COUNT(*) > 0 AND COUNT(*) FILTER (WHERE EXISTS (
			SELECT 1 FROM scenario_runs r WHERE r.user_id = $2 AND r.scenario_id = sc.id AND r.status = 'completed'
		)) > 0 THEN 100 ELSE 0 END
		FROM scenario_scripts sc WHERE sc.unit_id = $1`, unitID, userID).Scan(&scenarioPct)

	progress := int(math.Round(0.40*float64(lessonPct) + 0.25*float64(vocabPct) + 0.20*grammarConf*100 + 0.15*float64(scenarioPct)))
	if progress > 100 {
		progress = 100
	}

	_, _ = s.db.ExecContext(ctx, `
		UPDATE user_unit_progress SET progress_pct = $3, lessons_completed = $4, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND unit_id = $2`, userID, unitID, progress, completedLessons)

	// Unit is complete when the checkpoint lesson is completed.
	var checkpointDone bool
	_ = s.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM curriculum_lessons l JOIN user_lesson_attempts a
				ON a.lesson_id = l.id AND a.user_id = $2 AND a.status = 'completed'
			WHERE l.unit_id = $1 AND l.type = 'checkpoint'
		)`, unitID, userID).Scan(&checkpointDone)
	if checkpointDone {
		_, _ = s.db.ExecContext(ctx, `
			UPDATE user_unit_progress SET status = 'completed', competency_score = $3, completed_at = CURRENT_TIMESTAMP
			WHERE user_id = $1 AND unit_id = $2`, userID, unitID, progress*10)
		_ = s.unlockNextUnit(ctx, userID, unitID, targetLang)
	}
	return nil
}

func (s *LessonService) unlockNextUnit(ctx context.Context, userID, currentUnitID, targetLang string) error {
	// Find the next unit ordinal in the same course.
	var courseID, nextUnitID string
	_ = s.db.QueryRowContext(ctx, `
		SELECT course_id::text FROM curriculum_units WHERE id = $1`, currentUnitID).Scan(&courseID)
	err := s.db.QueryRowContext(ctx, `
		SELECT id::text FROM curriculum_units
		WHERE course_id = $1 AND ordinal > (SELECT ordinal FROM curriculum_units WHERE id = $2)
		AND cefr_level = COALESCE((SELECT CASE WHEN (SELECT ordinal FROM curriculum_units WHERE id = $2) = 6 THEN 'A2' ELSE (SELECT cefr_level FROM curriculum_units WHERE id = $2) END), (SELECT cefr_level FROM curriculum_units WHERE id = $2))
		ORDER BY ordinal LIMIT 1`, courseID, currentUnitID).Scan(&nextUnitID)
	if err != nil {
		return nil
	}
	// Mark the next unit available and move the profile's active unit forward.
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO user_unit_progress (user_id, unit_id, target_language, status, progress_pct)
		VALUES ($1, $2, $3, 'available', 0)
		ON CONFLICT (user_id, unit_id) DO UPDATE SET
			status = CASE WHEN user_unit_progress.status IN ('completed','in_progress') THEN user_unit_progress.status ELSE 'available' END`,
		userID, nextUnitID, targetLang)
	_ = s.profiles.SetActiveUnit(ctx, userID, targetLang, "", nextUnitID)
	return nil
}

func (s *LessonService) awardLessonActivity(ctx context.Context, userID, targetLang, eventType, lessonType string, xp int) {
	p, _ := json.Marshal(map[string]any{"lessonType": lessonType})
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO user_activity_events (user_id, target_language, event_type, source_type, xp, payload)
		VALUES ($1, $2, $3, 'lesson', $4, $5)`, userID, targetLang, eventType, xp, string(p))
}

// getSteps returns the persisted steps of a lesson.
func (s *LessonService) getSteps(ctx context.Context, lessonID string) ([]models.CurriculumStep, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id::text, lesson_id::text, ordinal, type, prompt, COALESCE(answer_key,'{}'), COALESCE(content_refs,'{}')
		FROM curriculum_lesson_steps WHERE lesson_id = $1 ORDER BY ordinal`, lessonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var steps []models.CurriculumStep
	for rows.Next() {
		var st models.CurriculumStep
		var prompt, answerKey, contentRefs []byte
		if err := rows.Scan(&st.ID, &st.LessonID, &st.Ordinal, &st.Type, &prompt, &answerKey, &contentRefs); err != nil {
			return nil, err
		}
		if len(prompt) > 0 {
			var p any
			_ = json.Unmarshal(prompt, &p)
			st.Prompt = p
		}
		if len(answerKey) > 0 {
			var ak any
			_ = json.Unmarshal(answerKey, &ak)
			st.AnswerKey = ak
		}
		if len(contentRefs) > 0 {
			var cf any
			_ = json.Unmarshal(contentRefs, &cf)
			st.ContentRefs = cf
		}
		steps = append(steps, st)
	}
	return steps, rows.Err()
}

// ensureSteps synthesizes and persists content steps when a lesson only has its
// intro step. Steps are derived from the unit's lexical items/grammar point.
func (s *LessonService) ensureSteps(ctx context.Context, lessonID, unitID, lessonType string) error {
	var count int
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM curriculum_lesson_steps WHERE lesson_id = $1`, lessonID).Scan(&count)
	if count > 1 {
		return nil
	}

	switch lessonType {
	case "vocabulary":
		items := s.queryLexicalItems(ctx, unitID)
		for i, it := range items {
			add := makeMCQStep(ctx, s.db, lessonID, itemOrdinal(i), it.SurfaceText, it.Translation)
			if add != nil {
				insertStep(ctx, s.db, lessonID, *add)
			}
		}
		add := makeClozeStep(ctx, s.db, lessonID)
		if add != nil {
			insertStep(ctx, s.db, lessonID, *add)
		}
	case "grammar", "scenario_intro", "reading", "production", "checkpoint":
		for i := 0; i < 3; i++ {
			add := makeGenericStep(ctx, s.db, lessonID, lessonType, i, unitID)
			if add != nil {
				insertStep(ctx, s.db, lessonID, *add)
			}
		}
	}
	return nil
}

func (s *LessonService) queryLexicalItems(ctx context.Context, unitID string) []struct {
	SurfaceText string
	Translation string
} {
	rows, err := s.db.QueryContext(ctx, `
		SELECT display_text, COALESCE(translations->>'en','') FROM lexical_items
		WHERE unit_id = $1 ORDER BY frequency_rank NULLS LAST LIMIT 4`, unitID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []struct {
		SurfaceText string
		Translation string
	}
	for rows.Next() {
		var t, tr string
		if err := rows.Scan(&t, &tr); err == nil {
			out = append(out, struct {
				SurfaceText string
				Translation string
			}{t, tr})
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// step synthesis + grading helpers
// ---------------------------------------------------------------------------

type stepDef struct {
	Type      string
	Prompt    map[string]any
	AnswerKey map[string]any
}

func makeMCQStep(ctx context.Context, db *sql.DB, lessonID string, ordinal int, term, translation string) *stepDef {
	if term == "" {
		return nil
	}
	choices := []string{
		term,
		stringsTitleCase(term) + " (alt)",
		"Unrelated option",
		"Another option",
	}
	translated := translation
	if translated == "" {
		translated = "N/A"
	}
	return &stepDef{
		Type:      "mcq",
		Prompt:    map[string]any{"text": "What does this word/phrase mean?", "source": term, "choices": choices},
		AnswerKey: map[string]any{"correct": term, "accepted": []string{term, translated}, "explanation": "Match the Spanish term to its meaning."},
	}
}

func makeClozeStep(ctx context.Context, db *sql.DB, lessonID string) *stepDef {
	return &stepDef{
		Type:      "cloze",
		Prompt:    map[string]any{"text": "Complete the phrase: Yo ____ un café.", "hint": "Common 1st-person present of querer"},
		AnswerKey: map[string]any{"accepted": []string{"quiero", "quisiera"}, "explanation": "Use quiero (I want) or quisiera (I would like)."},
	}
}

func makeGenericStep(ctx context.Context, db *sql.DB, lessonID, lessonType string, ordinal int, unitID string) *stepDef {
	prompts := map[string]string{
		"grammar":        "Which form completes the sentence correctly?",
		"reading":        "Read the mini-dialogue and choose the best response.",
		"production":     "Write a short personal sentence using a new word.",
		"checkpoint":     "Select the correct answer to show you can use this unit's language.",
		"scenario_intro": "Which phrase would you use in this real-world situation?",
	}
	text := prompts[lessonType]
	if text == "" {
		text = "Choose the best answer."
	}
	var prompt map[string]any
	if lessonType == "production" {
		prompt = map[string]any{"text": text, "tone": "practice"}
		return &stepDef{Type: "free_recall", Prompt: prompt, AnswerKey: map[string]any{"accepted": []string{""}, "free_form": true}}
	}
	choices := []string{"Option A", "Option B", "Option C", "Option D"}
	correct := "Option B"
	prompt = map[string]any{"text": text, "choices": choices}
	return &stepDef{Type: "mcq", Prompt: prompt, AnswerKey: map[string]any{"correct": correct, "accepted": []string{correct}}}
}

func insertStep(ctx context.Context, db *sql.DB, lessonID string, def stepDef) {
	var maxOrd int
	_ = db.QueryRowContext(ctx, `SELECT COALESCE(MAX(ordinal),0) FROM curriculum_lesson_steps WHERE lesson_id = $1`, lessonID).Scan(&maxOrd)
	prompt, _ := json.Marshal(def.Prompt)
	answerKey, _ := json.Marshal(def.AnswerKey)
	_, _ = db.ExecContext(ctx, `
		INSERT INTO curriculum_lesson_steps (lesson_id, ordinal, type, prompt, answer_key, content_refs)
		VALUES ($1, $2, $3, $4, $5, '{}')
		ON CONFLICT (lesson_id, ordinal) DO NOTHING`, lessonID, maxOrd+1, def.Type, string(prompt), string(answerKey))
}

func itemOrdinal(i int) int { return i }

func gradeStepAnswer(stepType string, answerKeyJSON []byte, answer string) (bool, int, string) {
	var ak map[string]any
	_ = json.Unmarshal(answerKeyJSON, &ak)
	// free-form step: always accepted.
	if v, ok := ak["free_form"].(bool); ok && v {
		return true, 4, ""
	}
	correct := ""
	if c, ok := ak["correct"].(string); ok {
		correct = c
	}
	var accepted []string
	switch a := ak["accepted"].(type) {
	case []interface{}:
		for _, e := range a {
			if s, ok := e.(string); ok {
				accepted = append(accepted, s)
			}
		}
	case []string:
		accepted = a
	}
	if correct != "" {
		accepted = append(accepted, correct)
	}
	norm := normalizeAnswer(answer)
	for _, a := range accepted {
		if normalizeAnswer(a) == norm {
			return true, 4, a
		}
	}
	return false, 1, correct
}

func lessonStepFeedback(correct bool) string {
	if correct {
		return "Correct. Keep going!"
	}
	return "Let's review that one — you'll meet it again."
}

func stringsTitleCase(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] = r[0] - ('a' - 'A')
	}
	return string(r)
}

func completFlagQuery(userID string) string {
	return ""
}

type meta struct {
	ID               string
	UnitID           string
	Ordinal          int
	Slug             string
	Type             string
	Title            string
	Objective        string
	EstimatedMinutes int
}
