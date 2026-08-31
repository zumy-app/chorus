package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
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
	AIMessage        string   `json:"ai_message"`
	Translation      string   `json:"translation"`
	PhaseComplete    bool     `json:"phase_complete"`
	NextPhaseOrdinal int      `json:"next_phase_ordinal,omitempty"`
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
		result, err := ep.call(prompt, nativeLangName, true)
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
