package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/chorus/messenger/internal/models"
)

const (
	placementTotalQuestions = 12
	placementItemsPerLevel  = 3
)

// PlacementService runs an adaptive (IRT-lite) placement test. It samples
// vocabulary/grammar items across CEFR bands, moves the ability estimate up or
// down per answer, and on completion writes the assigned level + readiness +
// starting unit to the user's language profile.
type PlacementService struct {
	db         *sql.DB
	curriculum *CurriculumService
	profiles   *LearningProfileService
}

func NewPlacementService(db *sql.DB, curriculum *CurriculumService, profiles *LearningProfileService) *PlacementService {
	return &PlacementService{db: db, curriculum: curriculum, profiles: profiles}
}

// StartPlacement opens an attempt and returns the first question.
func (s *PlacementService) StartPlacement(ctx context.Context, userID, targetLang, nativeLang string) (*models.PlacementStartResponse, error) {
	if nativeLang == "" {
		nativeLang = "en"
	}
	items, err := s.buildItemBank(ctx, targetLang, nativeLang)
	if err != nil {
		return nil, err
	}

	meta := placementMeta{Ability: 250, Items: items, ItemIndex: 0}
	metaJSON, _ := json.Marshal(meta)

	var attemptID string
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO placement_attempts (user_id, target_language, native_language, status, ability_estimate, metadata)
		VALUES ($1, $2, $3, 'in_progress', 250, $4)
		RETURNING id::text`, userID, targetLang, nativeLang, string(metaJSON)).Scan(&attemptID)
	if err != nil {
		return nil, err
	}

	// Mark profile as in_progress.
	_ = s.updatePlacementStatus(ctx, userID, targetLang, nativeLang, models.PlacementStatusInProgress)

	first := q(attemptID, items[0])
	return &models.PlacementStartResponse{
		AttemptID:      attemptID,
		Status:         "in_progress",
		Question:       first,
		TotalQuestions: len(items),
	}, nil
}

// GetPlacement returns the attempt and its next unanswered question.
func (s *PlacementService) GetPlacement(ctx context.Context, userID, attemptID string) (*models.PlacementStartResponse, error) {
	meta, err := s.loadMeta(ctx, attemptID, userID)
	if err != nil {
		return nil, err
	}
	var status string
	_ = s.db.QueryRowContext(ctx, `SELECT status FROM placement_attempts WHERE id = $1`, attemptID).Scan(&status)
	resp := &models.PlacementStartResponse{AttemptID: attemptID, Status: status, TotalQuestions: len(meta.Items)}
	if status == "completed" {
		return resp, nil
	}
	if meta.ItemIndex < len(meta.Items) {
		resp.Question = q(attemptID, meta.Items[meta.ItemIndex])
	}
	return resp, nil
}

// AnswerPlacement grades the current question and advances. When the attempt has
// covered enough items it finalizes and writes the assigned level to the profile.
func (s *PlacementService) AnswerPlacement(ctx context.Context, userID, attemptID, answer string) (*models.PlacementResult, error) {
	meta, err := s.loadMeta(ctx, attemptID, userID)
	if err != nil {
		return nil, err
	}
	if meta.ItemIndex >= len(meta.Items) {
		return s.finalize(ctx, userID, attemptID, &meta)
	}

	item := meta.Items[meta.ItemIndex]
	correct := normalizeAnswer(answer) == normalizeAnswer(item.Correct)
	meta.Ability = updatePlacementAbility(meta.Ability, levelToValue(item.CEFR), correct)

	// Record the response.
	promptJSON, _ := json.Marshal(map[string]any{"text": item.Prompt})
	ansJSON, _ := json.Marshal(map[string]any{"text": answer})
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO placement_responses (attempt_id, item_ref, item_type, cefr_level, prompt, user_answer, correct, score)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		attemptID, item.Ref, item.Type, item.CEFR, string(promptJSON), string(ansJSON), correct, int(scoreForPlacement(correct)))

	meta.ItemIndex++
	metaJSON, _ := json.Marshal(meta)
	_, _ = s.db.ExecContext(ctx, `UPDATE placement_attempts SET ability_estimate = $2, metadata = $3
		WHERE id = $1`, attemptID, meta.Ability, string(metaJSON))

	if meta.ItemIndex >= len(meta.Items) {
		return s.finalize(ctx, userID, attemptID, &meta)
	}
	return nil, fmt.Errorf("not complete")
}

// SkipPlacement sets profile to skipped -> A1 start.
func (s *PlacementService) SkipPlacement(ctx context.Context, userID, targetLang, nativeLang string) (*models.PlacementResult, error) {
	if nativeLang == "" {
		nativeLang = "en"
	}
	var attemptID string
	_ = s.db.QueryRowContext(ctx, `
		SELECT id::text FROM placement_attempts WHERE user_id = $1 AND target_language = $2 AND native_language = $3
		ORDER BY started_at DESC LIMIT 1`, userID, targetLang, nativeLang).Scan(&attemptID)
	_ = s.updatePlacementStatus(ctx, userID, targetLang, nativeLang, models.PlacementStatusSkipped)
	unitID := s.startUnitID(ctx, "A1", targetLang, nativeLang)
	_ = s.profiles.SetActiveUnit(ctx, userID, targetLang, nativeLang, unitID)
	return &models.PlacementResult{AttemptID: attemptID, EstimatedCEFR: "A1", ReadinessScore: 0, ActiveUnitID: unitID}, nil
}

// SelectLevel applies an onboarding self-selected starting level
// (beginner | intermediate | advanced) to the learner's profile without running
// the placement test. It maps the bucket to a CEFR seed, points the profile at
// the first unit for that band, and marks the placement status self_selected so
// the "find your level" onboarding card is no longer surfaced.
func (s *PlacementService) SelectLevel(ctx context.Context, userID, level, targetLang, nativeLang string) (*models.PlacementResult, error) {
	if nativeLang == "" {
		nativeLang = "en"
	}
	cefr := cefrFromSelfSelection(level)
	if cefr == "" {
		return nil, fmt.Errorf("invalid self-selected level %q: must be beginner, intermediate, or advanced", level)
	}

	// Ensure a profile row exists before seeding it.
	if _, err := s.profiles.GetProfile(ctx, userID, targetLang, nativeLang); err != nil {
		return nil, err
	}

	unitID := s.startUnitID(ctx, cefr, targetLang, nativeLang)
	readiness := readinessSeedForSelfSelection(cefr)

	_, err := s.db.ExecContext(ctx, `
		UPDATE user_language_profiles
		SET current_cefr_level = $4, readiness_score = $5, placement_status = $6,
		    active_unit_id = $7, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND target_language = $2 AND native_language = $3`,
		userID, targetLang, nativeLang, cefr, readiness, models.PlacementStatusSelfSelected, nullStr(unitID))
	if err != nil {
		return nil, err
	}

	return &models.PlacementResult{
		AttemptID:      "",
		EstimatedCEFR:  cefr,
		ReadinessScore: readiness,
		ActiveUnitID:   unitID,
	}, nil
}

func (s *PlacementService) finalize(ctx context.Context, userID, attemptID string, meta *placementMeta) (*models.PlacementResult, error) {
	level := levelFromAbility(int(meta.Ability))
	readiness := readinessWithinLevel(level, int(meta.Ability))

	// Mark profile completed.
	var targetLang, nativeLang string
	_ = s.db.QueryRowContext(ctx, `
		SELECT target_language, native_language FROM placement_attempts WHERE id = $1`, attemptID).Scan(&targetLang, &nativeLang)
	_ = s.updatePlacementStatus(ctx, userID, targetLang, nativeLang, models.PlacementStatusCompleted)
	_, _ = s.db.ExecContext(ctx, `UPDATE placement_attempts SET status = 'completed', estimated_cefr = $2, readiness_score = $3, completed_at = CURRENT_TIMESTAMP WHERE id = $1`,
		attemptID, level, readiness)

	unitID := s.startUnitID(ctx, level, targetLang, nativeLang)
	_, _ = s.db.ExecContext(ctx, `
		UPDATE user_language_profiles SET current_cefr_level = $4, readiness_score = $5, active_unit_id = $6, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND target_language = $2 AND native_language = $3`,
		userID, targetLang, nativeLang, level, readiness, nullStr(unitID))

	return &models.PlacementResult{
		AttemptID: attemptID, EstimatedCEFR: level, ReadinessScore: readiness, ActiveUnitID: unitID,
	}, nil
}

func (s *PlacementService) buildItemBank(ctx context.Context, targetLang, nativeLang string) ([]placementItem, error) {
	cap, err := s.capabilitiesFor(ctx, nativeLang, targetLang)
	if err != nil || cap.ActiveCourseID == "" {
		return buildPlacementFallback(), nil
	}

	// Load the course's core lexical items, preferring lower frequency rank
	// (questions keep a clear "which Spanish word means X?" shape, so only items
	// with an English gloss are usable).
	rows, err := s.db.QueryContext(ctx, `
		SELECT cefr_level, display_text, COALESCE(translations->>'en','')
		FROM lexical_items
		WHERE course_id = $1
		ORDER BY cefr_level, frequency_rank NULLS LAST, display_text`, cap.ActiveCourseID)
	if err != nil {
		return nil, err
	}

	termInfo := map[string]string{}     // term -> english gloss
	levelTerms := map[string][]string{} // cefr -> ordered terms
	for rows.Next() {
		var level, term, trans string
		if err := rows.Scan(&level, &term, &trans); err != nil {
			continue
		}
		if trans == "" {
			continue
		}
		if _, ok := termInfo[term]; ok {
			continue
		}
		termInfo[term] = trans
		levelTerms[level] = append(levelTerms[level], term)
	}
	rows.Close()

	if len(termInfo) == 0 {
		return buildPlacementFallback(), nil
	}

	// Global distractor pool (any distinct term in the course).
	pool := make([]string, 0, len(termInfo))
	for t := range termInfo {
		pool = append(pool, t)
	}

	items := []placementItem{}
	idx := 0
	for _, level := range []string{"A1", "A2", "B1", "B2"} {
		targets := levelTerms[level]
		if len(targets) > placementItemsPerLevel {
			targets = targets[:placementItemsPerLevel]
		}
		for _, term := range targets {
			items = append(items, placementItem{
				Ref:     fmt.Sprintf("%s-vocab-%d", level, idx),
				Type:    "vocabulary",
				CEFR:    level,
				Prompt:  fmt.Sprintf("Which Spanish word means %q?", termInfo[term]),
				Choices: buildChoices(term, pool),
				Correct: term,
			})
			idx++
		}
	}

	if len(items) < placementTotalQuestions-1 {
		items = append(items, buildPlacementFallback()...)
	}
	if len(items) > placementTotalQuestions {
		items = items[:placementTotalQuestions]
	}
	return items, nil
}

func buildPlacementFallback() []placementItem {
	return []placementItem{
		{Ref: "A1-grammar-1", Type: "grammar", CEFR: "A1", Prompt: "Complete: \"Yo ____ hablando español.\"", Choices: shuffleStrings([]string{"estoy", "soy", "es", "eres"}, "A1-grammar-1"), Correct: "estoy"},
		{Ref: "A1-grammar-2", Type: "grammar", CEFR: "A1", Prompt: "Choose the greeting:", Choices: shuffleStrings([]string{"Hola", "Adiós", "Gracias", "Por favor"}, "A1-grammar-2"), Correct: "Hola"},
		{Ref: "A2-grammar-1", Type: "grammar", CEFR: "A2", Prompt: "Yesterday I ____ al cine.", Choices: shuffleStrings([]string{"fui", "voy", "ir", "va"}, "A2-grammar-1"), Correct: "fui"},
		{Ref: "A2-grammar-2", Type: "grammar", CEFR: "A2", Prompt: "\"I am going to travel\" = ____ viajar.", Choices: shuffleStrings([]string{"Voy a", "Soy", "Estoy", "Voy de"}, "A2-grammar-2"), Correct: "Voy a"},
		{Ref: "B1-grammar-1", Type: "grammar", CEFR: "B1", Prompt: "Choose: \"If I had time, I ____ travel.\"", Choices: shuffleStrings([]string{"viajaría", "viajo", "viajaré", "viajé"}, "B1-grammar-1"), Correct: "viajaría"},
		{Ref: "B2-grammar-1", Type: "grammar", CEFR: "B2", Prompt: "Choose the more nuanced opinion opener:", Choices: shuffleStrings([]string{"Por un lado", "Es bueno", "Me gusta", "No sé"}, "B2-grammar-1"), Correct: "Por un lado"},
	}
}

func (s *PlacementService) startUnitID(ctx context.Context, level, targetLang, nativeLang string) string {
	cap, err := s.capabilitiesFor(ctx, nativeLang, targetLang)
	if err != nil || cap.ActiveCourseID == "" {
		return ""
	}
	var id string
	_ = s.db.QueryRowContext(ctx, `
		SELECT id::text FROM curriculum_units WHERE course_id = $1 AND cefr_level = $2
		ORDER BY ordinal LIMIT 1`, cap.ActiveCourseID, level).Scan(&id)
	return id
}

func (s *PlacementService) updatePlacementStatus(ctx context.Context, userID, targetLang, nativeLang, status string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE user_language_profiles SET placement_status = $4, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND target_language = $2 AND native_language = $3`,
		userID, targetLang, nativeLang, status)
	return err
}

func (s *PlacementService) capabilitiesFor(ctx context.Context, nativeLang, targetLang string) (*models.LearningPairCapability, error) {
	// Reuse the capability service via a small query to find the active course.
	cap := &models.LearningPairCapability{}
	var active sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT support_tier, active_course_id::text FROM learning_pair_capabilities
		WHERE native_language = $1 AND target_language = $2`, nativeLang, targetLang).Scan(&cap.SupportTier, &active)
	if err == sql.ErrNoRows {
		return &models.LearningPairCapability{}, fmt.Errorf("no course")
	}
	if err != nil {
		return nil, err
	}
	if active.Valid {
		cap.ActiveCourseID = active.String
	}
	return cap, nil
}

func (s *PlacementService) loadMeta(ctx context.Context, attemptID, userID string) (placementMeta, error) {
	var meta placementMeta
	var metaJSON []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(metadata::text,'') FROM placement_attempts WHERE id = $1 AND user_id = $2`,
		attemptID, userID).Scan(&metaJSON)
	if err != nil {
		return meta, err
	}
	if err := json.Unmarshal(metaJSON, &meta); err != nil {
		return meta, err
	}
	return meta, nil
}

// ---------------------------------------------------------------------------
// item bank + adaptive helpers
// ---------------------------------------------------------------------------

type placementItem struct {
	Ref     string   `json:"ref"`
	Type    string   `json:"type"`
	CEFR    string   `json:"cefr"`
	Prompt  string   `json:"prompt"`
	Choices []string `json:"choices"`
	Correct string   `json:"correct"`
}

type placementMeta struct {
	Ability   float64         `json:"ability"`
	ItemIndex int             `json:"itemIndex"`
	Items     []placementItem `json:"items"`
}

func levelToValue(level string) int {
	switch level {
	case "A1":
		return 100
	case "A2":
		return 350
	case "B1":
		return 650
	case "B2":
		return 850
	default:
		return 250
	}
}

func levelFromAbility(ability int) string {
	switch {
	case ability >= 800:
		return "B2"
	case ability >= 550:
		return "B1"
	case ability >= 250:
		return "A2"
	default:
		return "A1"
	}
}

// cefrFromSelfSelection maps an onboarding self-selected level bucket to a
// starting CEFR band. Beginner lands at A1, Intermediate at B1, and Advanced at
// B2 (the top band the launch curriculum covers; C1/C2 are post-launch). It is
// intentionally coarse: the placement test remains the fine-grained path, while
// self-selection is the quick "I know my level" signal.
func cefrFromSelfSelection(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "beginner":
		return "A1"
	case "intermediate":
		return "B1"
	case "advanced":
		return "B2"
	default:
		return ""
	}
}

// readinessSeedForSelfSelection returns the starting readiness score for a
// self-selected CEFR band: the lower bound of that band, so a learner who
// self-assesses into A2/B1/B2 isn't shown a 0% fluency while the score has not
// yet been recomputed from their learning activity.
func readinessSeedForSelfSelection(cefr string) int {
	switch cefr {
	case "A2":
		return 250
	case "B1":
		return 550
	case "B2":
		return 800
	default:
		return 0
	}
}

func readinessWithinLevel(level string, ability int) int {
	lo, hi := 0, 1000
	switch level {
	case "A1":
		lo, hi = 0, 249
	case "A2":
		lo, hi = 250, 549
	case "B1":
		lo, hi = 550, 799
	case "B2":
		lo, hi = 800, 1000
	}
	score := 0
	if hi > lo {
		score = int(math.Round(float64(ability-lo) / float64(hi-lo) * 1000))
	}
	if score > 1000 {
		score = 1000
	}
	if score < 0 {
		score = 0
	}
	return score
}

func q(attemptID string, item placementItem) models.PlacementQuestion {
	return models.PlacementQuestion{
		Ref: item.Ref, ItemType: item.Type, CEFRLevel: item.CEFR,
		Prompt:  map[string]any{"text": item.Prompt},
		Choices: item.Choices,
	}
}

func scoreForPlacement(correct bool) int {
	if correct {
		return 100
	}
	return 0
}

// updatePlacementAbility moves the ability estimate toward the item difficulty
// on correct answers and away on errors, always in the direction the answer
// implies. A small base step guarantees a correct answer never lowers the
// estimate (and a wrong answer never raises it) even when the item difficulty
// is far from the current estimate.
func updatePlacementAbility(ability float64, itemValue int, correct bool) float64 {
	const (
		baseStep = 25.0
		slope    = 0.35
	)
	value := float64(itemValue)
	if correct {
		ability += baseStep + slope*math.Max(0, value-ability)
	} else {
		ability -= baseStep + slope*math.Max(0, ability-value)
	}
	if ability < 0 {
		ability = 0
	}
	if ability > 1000 {
		ability = 1000
	}
	return ability
}

// buildChoices returns a shuffled 4-option set for a placement question. The
// correct term is included; the first three distinct distractor terms come from
// the pool, topped up from a curated list when the course bank is thin. The
// correct answer is never hard-coded to a fixed position.
func buildChoices(correct string, pool []string) []string {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	choices := []string{correct}
	used := map[string]bool{correct: true}

	for _, idx := range rng.Perm(len(pool)) {
		if len(choices) == 4 {
			break
		}
		t := pool[idx]
		if !used[t] {
			choices = append(choices, t)
			used[t] = true
		}
	}

	// Refill with common Spanish words when the bank can't supply enough
	// distractors. These never collide with the seeded course terms and can't be
	// the correct answer for a different question's fixed option (each item
	// builds its own pool).
	for _, curated := range []string{"es", "también", "porque", "después", "siempre", "nunca", "muy", "ahora"} {
		if len(choices) == 4 {
			break
		}
		if !used[curated] {
			choices = append(choices, curated)
			used[curated] = true
		}
	}
	return shuffleStrings(choices, correct)
}

// shuffleStrings returns a deterministic-ish copy with the slice order
// randomized so the correct option is never always first across renders of the
// same item ref.
func shuffleStrings(in []string, seed string) []string {
	out := make([]string, len(in))
	copy(out, in)
	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(len(seed))))
	rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}
