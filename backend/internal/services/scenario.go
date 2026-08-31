package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/lib/pq"
)

// ScenarioService runs real-world roleplay: list scenarios, open a run, advance
// phases as the learner produces target-language turns, grade each turn, and
// complete the run with XP, vocabulary reinjection, and readiness credit.
//
// Roleplay replies come from the LearningAIService when configured; a
// deterministic scripted fallback keeps the ordering-coffee flow fully working
// offline. Intent coverage is detected with a lightweight keyword matcher.
type ScenarioService struct {
	db         *sql.DB
	ai         *LearningAIService
	practice   *PracticeService
	fluency    *FluencyScoreService
	curriculum *CurriculumService
}

func NewScenarioService(db *sql.DB, ai *LearningAIService, practice *PracticeService, fluency *FluencyScoreService, curriculum *CurriculumService) *ScenarioService {
	return &ScenarioService{db: db, ai: ai, practice: practice, fluency: fluency, curriculum: curriculum}
}

// ListScenarios returns scenario cards with the learner's progress per card.
func (s *ScenarioService) ListScenarios(ctx context.Context, userID, targetLang, nativeLang string) ([]models.ScenarioScript, error) {
	cap, err := s.capability(ctx, nativeLang, targetLang)
	if err != nil || cap.ActiveCourseID == "" {
		return []models.ScenarioScript{}, nil
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT sc.id::text, sc.course_id::text, COALESCE(sc.unit_id::text,''), sc.slug, sc.title,
		       sc.domain, sc.cefr_level, sc.can_do_statement, sc.ai_role_name, sc.ai_role_description,
		       sc.opening_line, sc.max_turns, sc.estimated_minutes, sc.completion_criteria
		FROM scenario_scripts sc
		WHERE sc.course_id = $1 ORDER BY sc.cefr_level, sc.created_at`, cap.ActiveCourseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ScenarioScript
	for rows.Next() {
		var sc models.ScenarioScript
		var criteria []byte
		if err := rows.Scan(&sc.ID, &sc.CourseID, &sc.UnitID, &sc.Slug, &sc.Title, &sc.Domain,
			&sc.CEFRLevel, &sc.CanDoStatement, &sc.AIRoleName, &sc.AIRoleDescription,
			&sc.OpeningLine, &sc.MaxTurns, &sc.EstimatedMinutes, &criteria); err != nil {
			return nil, err
		}
		if len(criteria) > 0 {
			var c any
			_ = json.Unmarshal(criteria, &c)
			sc.CompletionCriteria = c
		}
		var completed bool
		_ = s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM scenario_runs WHERE scenario_id = $1 AND user_id = $2 AND status = 'completed')`, sc.ID, userID).Scan(&completed)
		if completed {
			sc.Metadata = map[string]any{"completed": true}
		} else {
			sc.Metadata = map[string]any{"completed": false}
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

// GetScenario returns one scenario with its phases.
func (s *ScenarioService) GetScenario(ctx context.Context, userID, scenarioID string) (*models.ScenarioScript, error) {
	var sc models.ScenarioScript
	var criteria []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT sc.id::text, sc.course_id::text, COALESCE(sc.unit_id::text,''), sc.slug, sc.title,
		       sc.domain, sc.cefr_level, sc.can_do_statement, sc.ai_role_name, sc.ai_role_description,
		       sc.opening_line, sc.max_turns, sc.estimated_minutes, sc.completion_criteria
		FROM scenario_scripts sc WHERE sc.id = $1`, scenarioID).Scan(
		&sc.ID, &sc.CourseID, &sc.UnitID, &sc.Slug, &sc.Title, &sc.Domain, &sc.CEFRLevel,
		&sc.CanDoStatement, &sc.AIRoleName, &sc.AIRoleDescription, &sc.OpeningLine,
		&sc.MaxTurns, &sc.EstimatedMinutes, &criteria)
	if err != nil {
		return nil, err
	}
	if len(criteria) > 0 {
		var c any
		_ = json.Unmarshal(criteria, &c)
		sc.CompletionCriteria = c
	}
	phases, err := s.getPhases(ctx, scenarioID)
	if err != nil {
		return nil, err
	}
	sc.Phases = phases
	return &sc, nil
}

// StartScenario opens a run and returns the AI opening turn.
func (s *ScenarioService) StartScenario(ctx context.Context, userID, scenarioID, targetLang, nativeLang string) (*models.ScenarioStartResponse, error) {
	if nativeLang == "" {
		nativeLang = "en"
	}
	var sc models.ScenarioScript
	var criteria []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT sc.id::text, sc.course_id::text, COALESCE(sc.unit_id::text,''), sc.slug, sc.title,
		       sc.domain, sc.cefr_level, sc.ai_role_name, sc.ai_role_description,
		       sc.opening_line, sc.max_turns, sc.estimated_minutes, sc.completion_criteria
		FROM scenario_scripts sc WHERE sc.id = $1`, scenarioID).Scan(
		&sc.ID, &sc.CourseID, &sc.UnitID, &sc.Slug, &sc.Title, &sc.Domain, &sc.CEFRLevel,
		&sc.AIRoleName, &sc.AIRoleDescription, &sc.OpeningLine, &sc.MaxTurns,
		&sc.EstimatedMinutes, &criteria)
	if err != nil {
		return nil, err
	}

	// Reuse an existing in-progress run.
	var runID, scaffold string
	err = s.db.QueryRowContext(ctx, `
		SELECT id::text, scaffold_level FROM scenario_runs
		WHERE scenario_id = $1 AND user_id = $2 AND status = 'in_progress'
		ORDER BY started_at DESC LIMIT 1`, scenarioID, userID).Scan(&runID, &scaffold)
	if err != nil {
		scaffold = "guided"
		err = s.db.QueryRowContext(ctx, `
			INSERT INTO scenario_runs (user_id, scenario_id, target_language, native_language, status, scaffold_level)
			VALUES ($1, $2, $3, $4, 'in_progress', $5)
			RETURNING id::text`, userID, scenarioID, targetLang, nativeLang, scaffold).Scan(&runID)
		if err != nil {
			return nil, err
		}
	}

	// Store the opening AI turn.
	translation := ""
	if s.ai != nil {
		if t, ok := s.translateText(ctx, userID, sc.OpeningLine, nativeLang); ok {
			translation = t
		}
	}
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO scenario_turns (run_id, ordinal, speaker, text, translation, phase_ordinal)
		VALUES ($1, 0, 'ai', $2, $3, 4)`, runID, sc.OpeningLine, translation)

	run := &models.ScenarioRun{
		ID: runID, UserID: userID, ScenarioID: scenarioID, TargetLanguage: targetLang,
		NativeLanguage: nativeLang, Status: "in_progress", ScaffoldLevel: scaffold,
		CurrentPhaseOrdinal: 1, StartedAt: time.Now(),
	}
	phases, _ := s.getPhases(ctx, scenarioID)
	if len(phases) > 0 {
		run.CurrentPhase = &phases[0]
		run.SuggestedChunks = phases[0].ChunkBank
	}

	return &models.ScenarioStartResponse{Run: run, AIResponse: models.ScenarioAIReply{
		AIMessage: sc.OpeningLine, Translation: translation, SuggestedChunks: chunksForPhase(phases, 1),
	}}, nil
}

// GetRun returns a run with its turns and current phase.
func (s *ScenarioService) GetRun(ctx context.Context, userID, runID string) (*models.ScenarioRun, error) {
	var r models.ScenarioRun
	var phaseScores, metadata []byte
	var completedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id::text, user_id, scenario_id::text, target_language, native_language, status,
		       scaffold_level, current_phase_ordinal, COALESCE(phase_scores::text,'[]'), covered_intents,
		       score, xp_awarded, started_at
		FROM scenario_runs WHERE id = $1 AND user_id = $2`, runID, userID).Scan(
		&r.ID, &r.UserID, &r.ScenarioID, &r.TargetLanguage, &r.NativeLanguage, &r.Status,
		&r.ScaffoldLevel, &r.CurrentPhaseOrdinal, &phaseScores, pq.Array(&r.CoveredIntents),
		&r.Score, &r.XPAwarded, &r.StartedAt)
	_ = metadata
	_ = completedAt
	if err != nil {
		return nil, err
	}
	_ = phaseScores
	turns, err := s.getTurns(ctx, runID)
	if err != nil {
		return nil, err
	}
	r.Turns = turns
	phases, _ := s.getPhases(ctx, r.ScenarioID)
	if len(phases) > 0 && r.CurrentPhaseOrdinal >= 1 && r.CurrentPhaseOrdinal <= len(phases) {
		r.CurrentPhase = &phases[r.CurrentPhaseOrdinal-1]
		r.SuggestedChunks = phases[r.CurrentPhaseOrdinal-1].ChunkBank
	}
	return &r, nil
}

// SendMessage processes one learner turn: grades intents, decides phase advance,
// generates the AI reply (AI or scripted fallback), and updates the run.
func (s *ScenarioService) SendMessage(ctx context.Context, userID, runID, message string) (*models.ScenarioAIReply, error) {
	var runID2, scenarioID, status, targetLang, nativeLang, scaffold string
	var currentPhase, score, xp int
	err := s.db.QueryRowContext(ctx, `
		SELECT id::text, scenario_id::text, status, target_language, native_language,
		       scaffold_level, current_phase_ordinal, score, xp_awarded
		FROM scenario_runs WHERE id = $1 AND user_id = $2`, runID, userID).Scan(
		&runID2, &scenarioID, &status, &targetLang, &nativeLang, &scaffold, &currentPhase, &score, &xp)
	if err != nil {
		return nil, err
	}
	if status != "in_progress" {
		return nil, fmt.Errorf("scenario already %s", status)
	}

	phases, _ := s.getPhases(ctx, scenarioID)
	var phase *models.ScenarioPhase
	if currentPhase >= 1 && currentPhase <= len(phases) {
		phase = &phases[currentPhase-1]
	}
	if phase == nil {
		return nil, fmt.Errorf("scenario phase out of range")
	}

	// Record the user turn.
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO scenario_turns (run_id, ordinal, speaker, text, phase_ordinal)
		SELECT run_id, COALESCE(MAX(ordinal),0)+1, 'user', $2, $3 FROM scenario_turns WHERE run_id = $1`,
		runID, message, currentPhase)

	intents := detectIntents(message, phase.RequiredIntents)
	covered := mergeStrings(runCoveredIntents(ctx, s.db, runID), intents)
	_ = covered

	// Decide if this phase is complete.
	phaseComplete := intentsCovered(intents, phase.RequiredIntents)

	var reply models.ScenarioAIReply
	if s.ai != nil && s.ai.HasProviders() {
		reply = s.genAIReply(ctx, userID, scenarioID, targetLang, nativeLang, phase, message, currentPhase, runCoveredIntents(ctx, s.db, runID))
	}
	if reply.AIMessage == "" {
		reply = scriptedReply(phase, message, intents, phaseComplete, scaffold)
	}

	// Advance phase if complete.
	nextPhase := currentPhase
	if phaseComplete {
		nextPhase = currentPhase + 1
	}
	if nextPhase > len(phases) {
		nextPhase = len(phases)
	}

	// Store the AI turn.
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO scenario_turns (run_id, ordinal, speaker, text, translation, phase_ordinal)
		SELECT run_id, COALESCE(MAX(ordinal),0)+1, 'ai', $2, $3, $4 FROM scenario_turns WHERE run_id = $1`,
		runID, reply.AIMessage, reply.Translation, nextPhase)

	// Mark any used chunks + spontaneous use.
	if len(intents) > 0 {
		s.touchChunkVocabulary(ctx, userID, targetLang, message, intents)
	}

	// Update run state.
	allIntents := mergeStrings(runCoveredIntents(ctx, s.db, runID), intents)
	newScore := score + 50
	newPhase := nextPhase
	runComplete := newPhase >= len(phases) && intentsCovered(allIntents, requiredIntentsFor(scenarioID, s, ctx))
	status2 := "in_progress"
	if runComplete {
		status2 = "completed"
		newScore = 700
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE scenario_runs SET current_phase_ordinal = $2, covered_intents = $3, score = $4, status = $5::text,
			completed_at = CASE WHEN $5::text = 'completed' THEN CURRENT_TIMESTAMP ELSE completed_at END
		WHERE id = $1`, runID, newPhase, pq.Array(allIntents), newScore, status2)
	if err != nil {
		log.Printf("[Scenario] update run %s: %v", runID, err)
	}

	reply.Score = newScore
	reply.CoveredIntents = allIntents
	reply.NextPhaseOrdinal = nextPhase
	reply.PhaseComplete = phaseComplete
	reply.RunCompleted = runComplete

	if runComplete {
		reply.Summary = s.finishRun(ctx, userID, runID, scenarioID, targetLang, nativeLang, allIntents, newScore)
		reply.RunCompleted = true
	}
	return &reply, nil
}

// RequestHint returns suggested chunks for the current phase (second pass).
func (s *ScenarioService) RequestHint(ctx context.Context, userID, runID string) ([]models.Chunk, error) {
	run, err := s.GetRun(ctx, userID, runID)
	if err != nil {
		return nil, err
	}
	if run.CurrentPhase != nil {
		return run.CurrentPhase.ChunkBank, nil
	}
	return []models.Chunk{}, nil
}

// CompleteScenario finalizes an in-progress run.
func (s *ScenarioService) CompleteScenario(ctx context.Context, userID, runID string) (*models.ScenarioAIReply, error) {
	run, err := s.GetRun(ctx, userID, runID)
	if err != nil {
		return nil, err
	}
	if run.Status == "completed" {
		return &models.ScenarioAIReply{RunCompleted: true}, nil
	}
	summary := s.finishRun(ctx, userID, runID, run.ScenarioID, run.TargetLanguage, run.NativeLanguage, run.CoveredIntents, run.Score)
	return &models.ScenarioAIReply{RunCompleted: true, Summary: summary}, nil
}

func (s *ScenarioService) finishRun(ctx context.Context, userID, runID, scenarioID, targetLang, nativeLang string, intents []string, score int) *models.ScenarioSummaryResult {
	if score < 700 {
		score = 700
	}
	_, _ = s.db.ExecContext(ctx, `
		UPDATE scenario_runs SET status = 'completed', score = $3, xp_awarded = $4, completed_at = CURRENT_TIMESTAMP
		WHERE id = $1`, runID, score, 100)
	// Daily stats + activity event + readiness.
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO daily_learning_stats (user_id, target_language, activity_date, xp, scenarios_completed, minutes_active)
		VALUES ($1, $2, CURRENT_DATE, 50, 1, 5)
		ON CONFLICT (user_id, target_language, activity_date) DO UPDATE SET
			xp = daily_learning_stats.xp + 50,
			scenarios_completed = daily_learning_stats.scenarios_completed + 1,
			minutes_active = daily_learning_stats.minutes_active + 5, updated_at = CURRENT_TIMESTAMP`,
		userID, targetLang)
	p, _ := json.Marshal(map[string]any{"scenarioId": scenarioID})
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO user_activity_events (user_id, target_language, event_type, source_type, source_id, xp, payload)
		VALUES ($1, $2, 'scenario_completed', 'scenario', $3, 50, $4)`,
		userID, targetLang, scenarioID, string(p))
	// bump readiness + recompute unit progress for the scenario's unit.
	_, _ = s.db.ExecContext(ctx, `
		UPDATE user_language_profiles SET readiness_score = LEAST(1000, readiness_score + 40), updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND target_language = $2 AND native_language = $3`, userID, targetLang, nativeLang)
	var unitID string
	_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(unit_id::text,'') FROM scenario_scripts WHERE id = $1`, scenarioID).Scan(&unitID)
	if unitID != "" {
		var progress int
		_ = s.db.QueryRowContext(ctx, `SELECT progress_pct FROM user_unit_progress WHERE user_id = $1 AND unit_id = $2`, userID, unitID).Scan(&progress)
		_, _ = s.db.ExecContext(ctx, `
			INSERT INTO user_unit_progress (user_id, unit_id, target_language, status, progress_pct, lessons_completed)
			VALUES ($1, $2, $3, 'in_progress', GREATEST(15, $4), 0)
			ON CONFLICT (user_id, unit_id) DO UPDATE SET progress_pct = GREATEST(15, user_unit_progress.progress_pct), updated_at = CURRENT_TIMESTAMP`,
			userID, unitID, targetLang, progress)
	}
	return &models.ScenarioSummaryResult{Score: score, XPAwarded: 100, VocabularyAdded: len(intents)}
}

// touchChunkVocabulary marks any produced chunk as a spontaneous use in the
// learner's vocabulary (production-stage mastery credit).
func (s *ScenarioService) touchChunkVocabulary(ctx context.Context, userID, targetLang, message string, intents []string) {
	if s.practice == nil {
		return
	}
	for _, w := range strings.Fields(strings.ToLower(message)) {
		w = NormalizeLearningTerm(w, targetLang)
		if w == "" || len([]rune(w)) <= 2 {
			continue
		}
		_ = s.practice.TouchSpontaneousUse(ctx, userID, w, targetLang)
	}
	_ = intents
}

func (s *ScenarioService) genAIReply(ctx context.Context, userID, scenarioID, targetLang, nativeLang string, phase *models.ScenarioPhase, message string, currentPhase int, covered []string) models.ScenarioAIReply {
	payload := map[string]any{
		"scenario":        scenarioID,
		"target_language": targetLang,
		"native_language": nativeLang,
		"cefr_level":      "A1",
		"current_phase": map[string]any{
			"title":            phase.Title,
			"learner_goal":     phase.LearnerGoal,
			"required_intents": phase.RequiredIntents,
		},
		"history":         covered,
		"learner_message": message,
	}
	rep, err := s.ai.GenerateScenarioReply(ctx, payload)
	if err != nil {
		return models.ScenarioAIReply{}
	}
	reply := models.ScenarioAIReply{
		AIMessage: rep.AIMessage, Translation: rep.Translation,
		PhaseComplete: rep.PhaseComplete, NextPhaseOrdinal: rep.NextPhaseOrdinal,
	}
	if reply.AIMessage == "" {
		reply.AIMessage = "¿Y luego qué más?"
	}
	return reply
}

func (s *ScenarioService) translateText(ctx context.Context, userID, text, nativeLang string) (string, bool) {
	if s.ai == nil {
		return "", false
	}
	// Reuse the translation service is preferred but the scenario service has no
	// direct handle; just return empty (the UI shows target text only).
	return "", false
}

func (s *ScenarioService) capability(ctx context.Context, nativeLang, targetLang string) (*models.LearningPairCapability, error) {
	cap := &models.LearningPairCapability{}
	var active sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT support_tier, active_course_id::text FROM learning_pair_capabilities
		WHERE native_language = $1 AND target_language = $2`, nativeLang, targetLang).Scan(&cap.SupportTier, &active)
	if err == sql.ErrNoRows {
		return cap, fmt.Errorf("no capability")
	}
	if err != nil {
		return nil, err
	}
	if active.Valid {
		cap.ActiveCourseID = active.String
	}
	return cap, nil
}

func (s *ScenarioService) getPhases(ctx context.Context, scenarioID string) ([]models.ScenarioPhase, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id::text, scenario_id::text, ordinal, title, learner_goal,
		       required_intents, chunk_bank
		FROM scenario_phases WHERE scenario_id = $1 ORDER BY ordinal`, scenarioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var phases []models.ScenarioPhase
	for rows.Next() {
		var ph models.ScenarioPhase
		var chunkJSON []byte
		if err := rows.Scan(&ph.ID, &ph.ScenarioID, &ph.Ordinal, &ph.Title, &ph.LearnerGoal,
			pq.Array(&ph.RequiredIntents), &chunkJSON); err != nil {
			return nil, err
		}
		if len(chunkJSON) > 0 {
			var chunks []models.Chunk
			_ = json.Unmarshal(chunkJSON, &chunks)
			ph.ChunkBank = chunks
		}
		phases = append(phases, ph)
	}
	return phases, rows.Err()
}

func (s *ScenarioService) getTurns(ctx context.Context, runID string) ([]models.ScenarioTurn, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id::text, run_id::text, ordinal, speaker, text, COALESCE(translation,''), phase_ordinal, evaluation
		FROM scenario_turns WHERE run_id = $1 ORDER BY ordinal`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var turns []models.ScenarioTurn
	for rows.Next() {
		var t models.ScenarioTurn
		var evalJSON []byte
		if err := rows.Scan(&t.ID, &t.RunID, &t.Ordinal, &t.Speaker, &t.Text, &t.Translation, &t.PhaseOrdinal, &evalJSON); err != nil {
			return nil, err
		}
		if len(evalJSON) > 0 {
			var e any
			_ = json.Unmarshal(evalJSON, &e)
			t.Evaluation = e
		}
		turns = append(turns, t)
	}
	return turns, rows.Err()
}

func requiredIntentsFor(scenarioID string, s *ScenarioService, ctx context.Context) []string {
	phases, _ := s.getPhases(ctx, scenarioID)
	req := []string{}
	for _, p := range phases {
		for _, i := range p.RequiredIntents {
			req = append(req, i)
		}
	}
	return req
}

func runCoveredIntents(ctx context.Context, db *sql.DB, runID string) []string {
	var out []string
	_ = db.QueryRowContext(ctx, `SELECT COVERED_intents FROM scenario_runs WHERE id = $1`, runID).Scan(pq.Array(&out))
	return out
}

func detectIntents(message string, required []string) []string {
	lower := strings.ToLower(message)
	norm := NormalizeLearningTerm(message, "es")
	var hit []string
	for _, r := range required {
		if intentMatches(norm, lower, r) {
			hit = append(hit, r)
		}
	}
	return hit
}

func intentMatches(norm, lower, intent string) bool {
	bank := map[string][]string{
		"greet":       {"hola", "buenos dias", "buenas tardes", "buenos días", "qué tal", "que tal"},
		"order_drink": {"quisiera", "quiero", "café", "cafe", "me puede dar", "pedir", "un café", "un te"},
		"customize":   {"para llevar", "sin azúcar", "sin azucar", "con leche", "tiene leche", "leche de avena", "azucar"},
		"pay":         {"cuánto cuesta", "cuanto cuesta", "precio", "tarjeta", "cuesta", "cuanto", "cuánto"},
		"close":       {"gracias", "adiós", "adios", "buen día", "buen dia", "hasta luego"},
	}
	keywords := bank[intent]
	for _, k := range keywords {
		if strings.Contains(norm, NormalizeLearningTerm(k, "es")) || strings.Contains(lower, k) {
			return true
		}
	}
	return false
}

func intentsCovered(got, required []string) bool {
	if len(required) == 0 {
		return true
	}
	set := map[string]bool{}
	for _, g := range got {
		set[g] = true
	}
	for _, r := range required {
		if !set[r] {
			return false
		}
	}
	return true
}

// scriptedReply produces a deterministic, in-character reply based on the phase
// and which required intents the learner covered.
func scriptedReply(phase *models.ScenarioPhase, message string, intents []string, phaseComplete bool, scaffold string) models.ScenarioAIReply {
	reply := models.ScenarioAIReply{}
	reply.SuggestedChunks = phase.ChunkBank
	reply.CoveredIntents = intents

	if !phaseComplete {
		reply.AIMessage = "¿Disculpa? No te entendí del todo. Prueba con una de las sugerencias."
		reply.Translation = "Sorry? I didn't quite understand. Try one of the suggestions."
		reply.PhaseComplete = false
		return reply
	}

	switch phase.Title {
	case "Greeting":
		reply.AIMessage = "¡Hola! Bienvenido. ¿Qué te gustaría pedir hoy?"
		reply.Translation = "Hello! Welcome. What would you like to order today?"
	case "Order":
		reply.AIMessage = "¡Perfecto! ¿Algo para acompañar, o para llevar?"
		reply.Translation = "Perfect! Anything to go with it, or to go?"
	case "Customization":
		reply.AIMessage = "Claro, sin problema. ¿Eso es todo?"
		reply.Translation = "Sure, no problem. Is that all?"
	case "Payment":
		reply.AIMessage = "Son tres euros con cincuenta. ¿Pagas con tarjeta o efectivo?"
		reply.Translation = "That's three euros fifty. Paying by card or cash?"
	case "Closing":
		reply.AIMessage = "¡Gracias a ti! Que tengas un buen día."
		reply.Translation = "Thank you! Have a great day."
	default:
		reply.AIMessage = "¡Muy bien! ¿Algo más?"
		reply.Translation = "Great! Anything else?"
	}
	reply.PhaseComplete = true
	return reply
}

func chunksForPhase(phases []models.ScenarioPhase, ordinal int) []models.Chunk {
	for _, p := range phases {
		if p.Ordinal == ordinal {
			return p.ChunkBank
		}
	}
	return nil
}

func mergeStrings(a, b []string) []string {
	set := map[string]bool{}
	for _, s := range a {
		set[s] = true
	}
	for _, s := range b {
		set[s] = true
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	return out
}
