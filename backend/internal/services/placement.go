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
	"github.com/lib/pq"
)

const (
	// 20-item structure: 8 receptive_vocab -> 8 grammar_production -> 4 discourse_reading.
	placementTotalQuestions = 20
	// Below this many calibrated items the bank is padded with fallback MCQs so
	// the adaptive engine always has enough signal to converge.
	placementMinBankItems = 10
)

// placementModuleQuota fixes the module mix and ordering of the redesigned
// assessment (plan V3 Phase 1).
var placementModuleQuota = []struct {
	module string
	quota  int
}{
	{"receptive_vocab", 8},
	{"grammar_production", 8},
	{"discourse_reading", 4},
}

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
	correct := gradePlacementAnswer(item, answer)
	meta.Ability = updatePlacementAbility(meta.Ability, itemDifficulty(item), correct)

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

	// Calibrated item bank first (plan V3): placement_items rows seeded from the
	// deploy-synced SQL files. The hardcoded fallback bank only runs when the
	// course has no calibrated items at all.
	rows, err := s.db.QueryContext(ctx, `
		SELECT id::text, cefr_level, module, item_type, COALESCE(prompt::text,'{}'), choices, correct, accept_variants, difficulty_value
		FROM placement_items
		WHERE course_id = $1 AND is_active = true
		ORDER BY cefr_level, difficulty_value`, cap.ActiveCourseID)
	if err != nil {
		return nil, err
	}
	bank := map[string][]placementItem{}
	total := 0
	for rows.Next() {
		var it placementItem
		var promptJSON string
		var choices, variants pq.StringArray
		if err := rows.Scan(&it.Ref, &it.CEFR, &it.Module, &it.Type, &promptJSON, &choices, &it.Correct, &variants, &it.Difficulty); err != nil {
			rows.Close()
			return nil, err
		}
		var p any
		if json.Unmarshal([]byte(promptJSON), &p) == nil {
			it.Prompt = p
		} else {
			it.Prompt = map[string]any{"text": it.Correct}
		}
		it.Choices = choices
		it.AcceptVariants = variants
		if it.Difficulty == 0 {
			it.Difficulty = levelToValue(it.CEFR)
		}
		bank[it.Module] = append(bank[it.Module], it)
		total++
	}
	rows.Close()

	if total == 0 {
		return buildPlacementFallback(), nil
	}

	// Fixed module structure: 8 receptive -> 8 grammar production -> 4 discourse.
	items := []placementItem{}
	for _, mq := range placementModuleQuota {
		mod := bank[mq.module]
		if len(mod) > mq.quota {
			mod = mod[:mq.quota]
		}
		items = append(items, mod...)
	}

	// Thin banks are topped up with fallback MCQs (never exceed the 20 cap).
	if len(items) < placementMinBankItems {
		for _, fb := range buildPlacementFallback() {
			if len(items) >= placementTotalQuestions {
				break
			}
			items = append(items, fb)
		}
	}
	if len(items) > placementTotalQuestions {
		items = items[:placementTotalQuestions]
	}
	return items, nil
}

func buildPlacementFallback() []placementItem {
	fb := []placementItem{
		// A1: Present tense verbs, basic greetings & word order
		{Ref: "A1-verb-1", Type: "verb_conjugation", Module: "grammar_production", CEFR: "A1", Prompt: "Present Tense: Complete \"Ella ____ (hablar) tres idiomas fluidamente.\"", Choices: shuffleStrings([]string{"habla", "hablo", "hablan", "hablar"}, "A1-verb-1"), Correct: "habla"},
		{Ref: "A1-grammar-1", Type: "grammar", Module: "grammar_production", CEFR: "A1", Prompt: "Verb Estar vs Ser: \"Yo ____ estudiando en la biblioteca ahora mismo.\"", Choices: shuffleStrings([]string{"estoy", "soy", "es", "somos"}, "A1-grammar-1"), Correct: "estoy"},
		{Ref: "A1-syntax-1", Type: "sentence_structure", Module: "grammar_production", CEFR: "A1", Prompt: "Sentence Construction: Select the correct word order:", Choices: shuffleStrings([]string{"Me gusta mucho el café", "Café el me gusta mucho", "Gusta me el café mucho", "Mucho me café gusta"}, "A1-syntax-1"), Correct: "Me gusta mucho el café"},

		// A2: Preterite vs Imperfect tenses, plans & direct object pronouns
		{Ref: "A2-verb-1", Type: "tense_distinction", Module: "grammar_production", CEFR: "A2", Prompt: "Preterite Tense: \"Ayer nosotros ____ (ir) a la playa todo el día.\"", Choices: shuffleStrings([]string{"fuimos", "bamos", "iremos", "fueron"}, "A2-verb-1"), Correct: "fuimos"},
		{Ref: "A2-verb-2", Type: "tense_distinction", Module: "grammar_production", CEFR: "A2", Prompt: "Imperfect Tense: \"Cuando era niño, siempre ____ (vivir) en Madrid.\"", Choices: shuffleStrings([]string{"vivía", "viví", "viviré", "vivo"}, "A2-verb-2"), Correct: "vivía"},
		{Ref: "A2-syntax-1", Type: "pronouns", Module: "grammar_production", CEFR: "A2", Prompt: "Object Pronoun Placement: \"¿Le diste el libro a María? Sí, ____ di ayer.\"", Choices: shuffleStrings([]string{"se lo", "le lo", "lo se", "me lo"}, "A2-syntax-1"), Correct: "se lo"},

		// B1: Present Subjunctive, Conditionals & Complex Relative Clauses
		{Ref: "B1-verb-1", Type: "subjunctive_mood", Module: "grammar_production", CEFR: "B1", Prompt: "Present Subjunctive: \"Dudo que ellos ____ (llegar) a tiempo a la reunión.\"", Choices: shuffleStrings([]string{"lleguen", "llegarán", "llegan", "llegaron"}, "B1-verb-1"), Correct: "lleguen"},
		{Ref: "B1-grammar-1", Type: "conditional_clause", Module: "grammar_production", CEFR: "B1", Prompt: "Hypothetical Condition: \"Si tuviera suficiente dinero, ____ (comprar) una casa.\"", Choices: shuffleStrings([]string{"compraría", "compro", "compraré", "compraba"}, "B1-grammar-1"), Correct: "compraría"},
		{Ref: "B1-syntax-1", Type: "sentence_structure", Module: "grammar_production", CEFR: "B1", Prompt: "Relative Clause: \"El autor ____ libro leímos dará una conferencia mañana.\"", Choices: shuffleStrings([]string{"cuyo", "que", "quien", "cual"}, "B1-syntax-1"), Correct: "cuyo"},

		// B2: College/Academic Writing Level - Imperfect Subjunctive, Advanced Discourse & Syntax Nuance
		{Ref: "B2-verb-1", Type: "imperfect_subjunctive", Module: "grammar_production", CEFR: "B2", Prompt: "Past Subjunctive Hypothesis: \"Si hubieras estudiado más, ____ (obtener) mejores resultados en el examen académico.\"", Choices: shuffleStrings([]string{"habrías obtenido", "obtuviste", "obtengas", "obtendrás"}, "B2-verb-1"), Correct: "habrías obtenido"},
		{Ref: "B2-discourse-1", Type: "academic_discourse", Module: "discourse_reading", CEFR: "B2", Prompt: "College Academic Writing: Choose the best concessive connector to synthesize contrasting arguments in an essay: \"El proyecto presenta beneficios; ____, debemos analizar los riesgos socioeconómicos a largo plazo.\"", Choices: shuffleStrings([]string{"no obstante", "así que", "porque", "entonces"}, "B2-discourse-1"), Correct: "no obstante"},
		{Ref: "B2-syntax-1", Type: "advanced_syntax", Module: "discourse_reading", CEFR: "B2", Prompt: "Advanced Syntactic Register: Select the sentence that demonstrates proper formal academic prose:", Choices: shuffleStrings([]string{"Es fundamental que se consideren las repercusiones éticas del estudio.", "Es bueno que piensen en las cosas éticas del estudio.", "Tienen que ver la ética del estudio siempre.", "Hay que mirar si la ética del estudio está bien."}, "B2-syntax-1"), Correct: "Es fundamental que se consideren las repercusiones éticas del estudio."},
	}
	// Wrap plain-text prompts + seed difficulty so fallback items flow through
	// the same rendering/grading path as calibrated bank items.
	for i := range fb {
		fb[i].Prompt = map[string]any{"text": fb[i].Prompt}
		fb[i].Difficulty = levelToValue(fb[i].CEFR)
	}
	return fb
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
	Ref            string   `json:"ref"`
	Type           string   `json:"type"`   // L2_to_L1_context, gap_fill_type, sentence_reconstruction, passage_mcq
	Module         string   `json:"module"` // receptive_vocab, grammar_production, discourse_reading
	CEFR           string   `json:"cefr"`
	Prompt         any      `json:"prompt"` // rich JSONB payload; fallback items wrap plain text
	Choices        []string `json:"choices"`
	Correct        string   `json:"correct"`
	AcceptVariants []string `json:"acceptVariants"`
	Difficulty     int      `json:"difficulty"`
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
	prompt := item.Prompt
	if prompt == nil {
		prompt = map[string]any{"text": ""}
	}
	// Old in-flight attempts persisted the prompt as a plain string; keep that
	// shape so resume rendering never breaks.
	return models.PlacementQuestion{
		Ref: item.Ref, ItemType: item.Type, Module: item.Module, CEFRLevel: item.CEFR,
		Prompt:  prompt,
		Choices: item.Choices,
	}
}

func scoreForPlacement(correct bool) int {
	if correct {
		return 100
	}
	return 0
}

// itemDifficulty resolves the IRT b-parameter for an item. Calibrated bank
// items carry an explicit difficulty_value; fallback items default to their
// CEFR band anchor.
func itemDifficulty(item placementItem) int {
	if item.Difficulty > 0 {
		return item.Difficulty
	}
	return levelToValue(item.CEFR)
}

// gradePlacementAnswer checks the typed/selected answer against the correct
// form plus any accepted variants using level-aware normalization.
func gradePlacementAnswer(item placementItem, answer string) bool {
	candidates := append([]string{item.Correct}, item.AcceptVariants...)
	for _, expected := range candidates {
		if normalizeAnswerForLevel(answer, expected, item.CEFR) {
			return true
		}
	}
	return false
}

// normalizeAnswerForLevel compares a learner answer to the expected form.
// A1/A2 are forgiving: case-insensitive, whitespace-collapsed, accents stripped
// (early learners should not fail on "esta" vs "está"). B1+ require diacritics
// to match exactly — accent accuracy is part of what is being assessed.
func normalizeAnswerForLevel(answer, expected, cefr string) bool {
	a := collapseWhitespace(strings.TrimSpace(strings.ToLower(answer)))
	e := collapseWhitespace(strings.TrimSpace(strings.ToLower(expected)))
	if a == "" || e == "" {
		return false
	}
	if cefr == "A1" || cefr == "A2" {
		return stripDiacritics(a) == stripDiacritics(e)
	}
	return a == e
}

var diacriticReplacer = strings.NewReplacer(
	"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u",
	"ñ", "n", "ç", "c", "¡", "", "¿", "",
)

func stripDiacritics(s string) string {
	return diacriticReplacer.Replace(s)
}

func collapseWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// updatePlacementAbility moves the ability estimate with an IRT-lite logistic
// update (plan V3 Phase 1). P(theta, b) is the 1-parameter logistic probability
// of a correct response; a correct answer gains 80*(1-P) and a wrong answer
// loses 80*P, so items far below the learner's level barely move the estimate
// while items near the boundary carry the most information.
func updatePlacementAbility(ability float64, itemValue int, correct bool) float64 {
	const (
		logisticScale = 100.0 // steepness on the 0-1000 ability scale
		maxStep       = 80.0
	)
	p := 1.0 / (1.0 + math.Exp(-(ability-float64(itemValue))/logisticScale))
	if correct {
		ability += maxStep * (1.0 - p)
	} else {
		ability -= maxStep * p
	}
	if ability < 0 {
		ability = 0
	}
	if ability > 1000 {
		ability = 1000
	}
	return ability
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
