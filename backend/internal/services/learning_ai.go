package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/chorus/messenger/internal/models"
)

// LearningAIService wraps the grammar provider chain for learning-specific LLM
// tasks: vocabulary extraction and scenario roleplay. It mirrors the small
// provider-fallback pattern from GrammarService (tries each endpoint in order,
// falling through on failure). When no endpoint is configured or all fail, the
// caller falls back to deterministic behavior (tokenize/scripted replies).
type LearningAIService struct {
	endpoints []GrammarEndpoint
}

func NewLearningAIService(endpoints []GrammarEndpoint) *LearningAIService {
	return &LearningAIService{endpoints: endpoints}
}

// HasProviders reports whether at least one endpoint can be tried.
func (s *LearningAIService) HasProviders() bool {
	return s != nil && len(s.endpoints) > 0
}

// MinedCandidate is the raw extraction result from the model before routing.
// Fields below "Reason" are internal enrichment populated during
// classify/dedupe and are not part of the AI extraction contract.
type MinedCandidate struct {
	SurfaceText  string   `json:"surface_text"`
	Lemma        string   `json:"lemma"`
	PartOfSpeech string   `json:"part_of_speech"`
	IsChunk      bool     `json:"is_chunk"`
	Translation  string   `json:"translation"`
	Definition   string   `json:"definition"`
	CEFRLevel    string   `json:"cefr_level"`
	GrammarTags  []string `json:"grammar_tags"`
	IsProperNoun bool     `json:"is_proper_noun"`
	Confidence   float64  `json:"confidence"`
	Reason       string   `json:"reason"`

	// internal enrichment
	UserID              string  `json:"-"`
	Language            string  `json:"-"`
	NormalizedText      string  `json:"-"`
	ContextSentence     string  `json:"-"`
	SourceType          string  `json:"-"`
	SourceMessageID     string  `json:"-"`
	MessageID           string  `json:"-"`
	ChatID              string  `json:"-"`
	CurriculumLexicalID string  `json:"-"`
	CurriculumUnitID    string  `json:"-"`
	RouteStatus         string  `json:"-"`
	RouteReason         string  `json:"-"`
	TeachabilityScore   float64 `json:"-"`
}

type wordExtractionOutput struct {
	Items []MinedCandidate `json:"items"`
}

// languageName returns a human label for an ISO code used in prompts.
func languageName(code string) string {
	switch strings.ToLower(code) {
	case "es":
		return "Spanish"
	case "en":
		return "English"
	case "fr":
		return "French"
	case "de":
		return "German"
	case "it":
		return "Italian"
	case "pt":
		return "Portuguese"
	case "hi":
		return "Hindi"
	case "zh":
		return "Chinese"
	case "ar":
		return "Arabic"
	case "ru":
		return "Russian"
	case "ur":
		return "Urdu"
	case "bn":
		return "Bengali"
	default:
		return strings.ToUpper(code)
	}
}

// ExtractVocabulary asks the model to return strict JSON of useful vocabulary and
// chunks from a message. It returns (nil, nil) when there are no endpoints.
func (s *LearningAIService) ExtractVocabulary(ctx context.Context, text, targetLang, nativeLang, cefr string) ([]MinedCandidate, error) {
	if !s.HasProviders() {
		return nil, nil
	}
	targetName := languageName(targetLang)
	nativeName := languageName(nativeLang)
	if cefr == "" {
		cefr = "A1"
	}

	system := "You are a language-learning vocabulary miner. Extract only useful vocabulary and reusable chunks from a learner's real message context. Return strict JSON. Do not include markdown or prose."

	user := fmt.Sprintf(`Target language: %s
Native language: %s
Learner CEFR: %s
Source text:
%s

Return JSON with the shape:
{"items": [{"surface_text": string, "lemma": string, "part_of_speech": string, "is_chunk": boolean, "translation": string, "definition": string, "cefr_level": "A1"|"A2"|"B1"|"B2", "grammar_tags": string[], "is_proper_noun": boolean, "confidence": number, "reason": string}]}

Rules: prefer useful words and chunks over every token; include multi-word chunks when more teachable; exclude names, URLs, emoji-only items, filler, and rare slang; return 0-8 items; tag CEFR conservatively.`, targetName, nativeName, cefr, text)

	raw, err := s.complete(ctx, system, user, nativeName)
	if err != nil {
		return nil, err
	}
	var out wordExtractionOutput
	if err := parseStrictJSON(raw, &out); err != nil {
		return nil, fmt.Errorf("word extraction parse: %w", err)
	}
	return out.Items, nil
}

// ScenarioReply is the LLM return shape for a roleplay turn.
type ScenarioReply struct {
	AIMessage        string        `json:"ai_message"`
	Translation      string        `json:"translation"`
	PhaseComplete    bool          `json:"phase_complete"`
	NextPhaseOrdinal int           `json:"next_phase_ordinal,omitempty"`
	Nudge            ScenarioNudge `json:"nudge"`
}

type ScenarioNudge struct {
	Show            bool     `json:"show"`
	Text            string   `json:"text"`
	SuggestedChunks []string `json:"suggested_chunks"`
}

// GenerateScenarioReply asks the model for the AI partner's next message in a
// roleplay. Payload is a generic struct so the prompt can be shaped per scenario.
func (s *LearningAIService) GenerateScenarioReply(ctx context.Context, p map[string]any) (*ScenarioReply, error) {
	if !s.HasProviders() {
		return nil, nil
	}
	nativeName := "English"
	if v, ok := p["native_language"].(string); ok {
		nativeName = languageName(v)
	}

	system := "You are the AI scene partner in a language-learning roleplay. Stay in character. Use clear language at the learner's CEFR level. Keep replies short. Do not complete the learner's task for them. Move the scenario forward one phase at a time. Return strict JSON."

	payload, _ := json.Marshal(p)
	raw, err := s.complete(ctx, system, string(payload), nativeName)
	if err != nil {
		return nil, err
	}
	var reply ScenarioReply
	if err := parseStrictJSON(raw, &reply); err != nil {
		return nil, fmt.Errorf("scenario reply parse: %w", err)
	}
	return &reply, nil
}

// complete runs a prompt against each endpoint in order and returns raw text.
func (s *LearningAIService) complete(ctx context.Context, system, user, nativeLangName string) (string, error) {
	var lastErr error
	for i, ep := range s.endpoints {
		prompt := system + "\n\n" + user
		start := time.Now()
		result, err := ep.call(ctx, prompt, nativeLangName, true)
		_ = i
		_ = start
		if err == nil {
			return result, nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("all learning AI endpoints exhausted: %w", lastErr)
}

// parseStrictJSON tolerates a model wrapping JSON in a fenced code block.
func parseStrictJSON(raw string, out interface{}) error {
	raw = strings.TrimSpace(raw)
	raw = stripCodeFence(raw)
	raw = strings.TrimSpace(raw)
	// If the model returned a bare array (some providers do), wrap it.
	if strings.HasPrefix(raw, "[") {
		raw = `{"items":` + raw + `}`
	}
	if err := json.Unmarshal([]byte(raw), out); err != nil {
		// Fall back to extracting the first {...} block.
		start := strings.Index(raw, "{")
		end := strings.LastIndex(raw, "}")
		if start >= 0 && end > start {
			if err2 := json.Unmarshal([]byte(raw[start:end+1]), out); err2 == nil {
				return nil
			}
		}
		return err
	}
	return nil
}

func stripCodeFence(s string) string {
	if !strings.Contains(s, "```") {
		return s
	}
	var b strings.Builder
	inFence := false
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if !inFence {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// GradeProduction evaluates a learner's produced text against the task prompt
// and returns a structured result (score, feedback, corrections, CEFR band).
// It powers the async grading queue (plan V3 Phase 4). Returns the provider
// used, or "heuristic-fallback" when no endpoint answered — the queue then
// still completes the job with a deterministic offline grade.
func (s *LearningAIService) GradeProduction(ctx context.Context, p models.GradingJobPayload) (*models.GradingJobResult, string, error) {
	if !s.HasProviders() {
		return heuristicGrade(p), "heuristic-fallback", nil
	}
	nativeName := languageName(p.NativeLanguage)
	if nativeName == "" {
		nativeName = "English"
	}
	targetName := languageName(p.TargetLanguage)

	system := fmt.Sprintf(`You are an experienced %s language teacher grading a learner exercise. Grade fairly at the learner's CEFR level (%s). Return ONLY strict JSON:
{"score": 0-10 integer, "maxScore": 10, "feedback": "2-4 sentences of encouraging, specific feedback written in %s", "corrections": [{"span": "the learner's exact wrong text", "correction": "the corrected text", "grammarTag": "short grammar concept", "explanation": "one-sentence why, in %s"}], "cefrBand": "A1|A2|B1|B2", "accuracyScore": 0.0-1.0}`,
		targetName, p.CEFRLevel, nativeName, nativeName)

	userPayload, _ := json.Marshal(map[string]any{
		"task_type":     p.TaskType,
		"prompt":        p.Prompt,
		"user_text":     p.UserText,
		"cefr_level":    p.CEFRLevel,
		"focus_grammar": p.FocusGrammar,
		"focus_vocab":   p.FocusVocab,
	})

	raw, err := s.complete(ctx, system, string(userPayload), nativeName)
	if err != nil {
		return heuristicGrade(p), "heuristic-fallback", err
	}
	var out models.GradingJobResult
	if err := parseStrictJSON(raw, &out); err != nil {
		return heuristicGrade(p), "heuristic-fallback", fmt.Errorf("grading parse: %w", err)
	}
	if out.MaxScore == 0 {
		out.MaxScore = 10
	}
	if out.Score > out.MaxScore {
		out.Score = out.MaxScore
	}
	return &out, "ai", nil
}

// heuristicGrade is the deterministic offline grader used when no AI endpoint
// is reachable: length-based engagement score with actionable feedback, so the
// async pipeline always resolves a job instead of hanging the UI skeleton.
func heuristicGrade(p models.GradingJobPayload) *models.GradingJobResult {
	words := strings.Fields(p.UserText)
	res := &models.GradingJobResult{MaxScore: 10, CEFRBand: p.CEFRLevel}
	switch {
	case len(words) >= 40:
		res.Score = 8
	case len(words) >= 20:
		res.Score = 6
	case len(words) >= 8:
		res.Score = 4
	case len(words) > 0:
		res.Score = 2
	default:
		res.Score = 0
	}
	if p.FocusGrammar != "" && strings.Contains(strings.ToLower(p.UserText), strings.ToLower(p.FocusGrammar)) {
		res.Score = min(res.Score+1, 10)
	}
	res.AccuracyScore = float64(res.Score) / 10.0
	res.Feedback = "Offline review: your response was recorded and will be re-graded with full AI feedback when the tutor engine is reachable. Keep writing!"
	return res
}
