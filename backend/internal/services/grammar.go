package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/redis/go-redis/v9"
)

// GrammarService handles grammar analysis for language learning
type GrammarService struct {
	redis       *redis.Client
	ollamaURL   string
	ollamaModel string
	httpClient  *http.Client
}

// NewGrammarService creates a new Grammar service
func NewGrammarService(redis *redis.Client, ollamaURL, ollamaModel string) *GrammarService {
	return &GrammarService{
		redis:       redis,
		ollamaURL:   ollamaURL,
		ollamaModel: ollamaModel,
		// 90s is the outer safety net; individual calls use context deadlines (30s).
		httpClient: &http.Client{Timeout: 90 * time.Second},
	}
}

// AnalyzeGrammar performs grammar analysis on a message
func (s *GrammarService) AnalyzeGrammar(text, language string) (*models.GrammarAnalysis, error) {
	ctx := context.Background()

	// Check cache first
	cacheKey := fmt.Sprintf("grammar:%s:%s", language, hashText(text))
	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var analysis models.GrammarAnalysis
		if json.Unmarshal([]byte(cached), &analysis) == nil {
			return &analysis, nil
		}
	}

	// Perform analysis
	analysis := &models.GrammarAnalysis{
		Difficulty:   s.assessDifficulty(text, language),
		Patterns:     s.identifyPatterns(text, language),
		Explanations: s.generateExplanations(text, language),
	}

	// Cache result
	if jsonData, err := json.Marshal(analysis); err == nil {
		s.redis.Set(ctx, cacheKey, jsonData, 24*time.Hour)
	}

	return analysis, nil
}

// assessDifficulty determines the CEFR difficulty level
func (s *GrammarService) assessDifficulty(text, language string) string {
	// Simple heuristic-based assessment
	// In production, this would use ML models or external APIs

	wordCount := len(strings.Fields(text))
	avgWordLength := float64(len(strings.ReplaceAll(text, " ", ""))) / float64(wordCount)

	// Check for complex structures
	hasSubordinates := containsSubordinateClause(text, language)
	hasPassive := containsPassiveVoice(text, language)
	hasConditional := containsConditional(text, language)

	complexityScore := 0

	// Word count scoring
	if wordCount > 15 {
		complexityScore += 2
	} else if wordCount > 10 {
		complexityScore += 1
	}

	// Average word length scoring
	if avgWordLength > 7 {
		complexityScore += 2
	} else if avgWordLength > 5 {
		complexityScore += 1
	}

	// Structure scoring
	if hasSubordinates {
		complexityScore += 2
	}
	if hasPassive {
		complexityScore += 1
	}
	if hasConditional {
		complexityScore += 2
	}

	// Map score to CEFR level
	switch {
	case complexityScore <= 1:
		return "A1"
	case complexityScore <= 2:
		return "A2"
	case complexityScore <= 4:
		return "B1"
	case complexityScore <= 6:
		return "B2"
	case complexityScore <= 8:
		return "C1"
	default:
		return "C2"
	}
}

// identifyPatterns identifies grammar patterns in the text
func (s *GrammarService) identifyPatterns(text, language string) []string {
	var patterns []string

	lowerText := strings.ToLower(text)

	// Common patterns based on language
	switch language {
	case "en":
		patterns = s.identifyEnglishPatterns(lowerText)
	case "es":
		patterns = s.identifySpanishPatterns(lowerText)
	case "fr":
		patterns = s.identifyFrenchPatterns(lowerText)
	case "de":
		patterns = s.identifyGermanPatterns(lowerText)
	default:
		patterns = s.identifyEnglishPatterns(lowerText) // Default to English
	}

	return patterns
}

// identifyEnglishPatterns identifies grammar patterns in English text
func (s *GrammarService) identifyEnglishPatterns(text string) []string {
	var patterns []string

	// Present continuous
	if regexp.MustCompile(`\b(am|is|are)\s+\w+ing\b`).MatchString(text) {
		patterns = append(patterns, "present_continuous")
	}

	// Past tense
	if regexp.MustCompile(`\b\w+ed\b`).MatchString(text) ||
		regexp.MustCompile(`\b(was|were|had|did)\b`).MatchString(text) {
		patterns = append(patterns, "past_tense")
	}

	// Future tense
	if regexp.MustCompile(`\b(will|going to|shall)\b`).MatchString(text) {
		patterns = append(patterns, "future_tense")
	}

	// Conditional
	if regexp.MustCompile(`\b(if|unless|would|could|should)\b`).MatchString(text) {
		patterns = append(patterns, "conditional")
	}

	// Passive voice
	if regexp.MustCompile(`\b(was|were|been|being)\s+\w+ed\b`).MatchString(text) {
		patterns = append(patterns, "passive_voice")
	}

	// Questions
	if strings.Contains(text, "?") ||
		regexp.MustCompile(`^(do|does|did|is|are|was|were|have|has|had|can|could|will|would)\b`).MatchString(text) {
		patterns = append(patterns, "question")
	}

	// Relative clauses
	if regexp.MustCompile(`\b(who|which|that|whom|whose|where|when)\b`).MatchString(text) {
		patterns = append(patterns, "relative_clause")
	}

	// Present perfect
	if regexp.MustCompile(`\b(have|has)\s+\w+ed\b`).MatchString(text) ||
		regexp.MustCompile(`\b(have|has)\s+(been|gone|done|seen|made)\b`).MatchString(text) {
		patterns = append(patterns, "present_perfect")
	}

	// Comparatives/Superlatives
	if regexp.MustCompile(`\b\w+(er|est)\b`).MatchString(text) ||
		regexp.MustCompile(`\b(more|most|less|least)\s+\w+\b`).MatchString(text) {
		patterns = append(patterns, "comparison")
	}

	return patterns
}

// identifySpanishPatterns identifies grammar patterns in Spanish text
func (s *GrammarService) identifySpanishPatterns(text string) []string {
	var patterns []string

	// Present tense
	if regexp.MustCompile(`\b\w+(o|as|a|amos|áis|an|es|e|emos|éis|en|is|imos|ís)\b`).MatchString(text) {
		patterns = append(patterns, "presente")
	}

	// Preterite
	if regexp.MustCompile(`\b\w+(é|aste|ó|amos|asteis|aron|í|iste|ió|imos|isteis|ieron)\b`).MatchString(text) {
		patterns = append(patterns, "preterito")
	}

	// Subjunctive
	if regexp.MustCompile(`\b(que|ojalá|espero que|quiero que)\b`).MatchString(text) {
		patterns = append(patterns, "subjuntivo")
	}

	// Reflexive verbs
	if regexp.MustCompile(`\b(me|te|se|nos|os)\s+\w+\b`).MatchString(text) {
		patterns = append(patterns, "verbos_reflexivos")
	}

	// Conditional
	if regexp.MustCompile(`\b\w+(ía|ías|íamos|íais|ían)\b`).MatchString(text) {
		patterns = append(patterns, "condicional")
	}

	return patterns
}

// identifyFrenchPatterns identifies grammar patterns in French text
func (s *GrammarService) identifyFrenchPatterns(text string) []string {
	var patterns []string

	// Present tense
	if regexp.MustCompile(`\b\w+(e|es|ons|ez|ent)\b`).MatchString(text) {
		patterns = append(patterns, "présent")
	}

	// Passé composé
	if regexp.MustCompile(`\b(ai|as|a|avons|avez|ont|suis|es|est|sommes|êtes|sont)\s+\w+(é|i|u|is|it)\b`).MatchString(text) {
		patterns = append(patterns, "passé_composé")
	}

	// Imparfait
	if regexp.MustCompile(`\b\w+(ais|ait|ions|iez|aient)\b`).MatchString(text) {
		patterns = append(patterns, "imparfait")
	}

	// Subjonctif
	if regexp.MustCompile(`\b(que|qu')\s+\w+\b`).MatchString(text) {
		patterns = append(patterns, "subjonctif")
	}

	return patterns
}

// identifyGermanPatterns identifies grammar patterns in German text
func (s *GrammarService) identifyGermanPatterns(text string) []string {
	var patterns []string

	// Modal verbs
	if regexp.MustCompile(`\b(kann|muss|soll|will|darf|mag)\b`).MatchString(text) {
		patterns = append(patterns, "modalverben")
	}

	// Perfect tense
	if regexp.MustCompile(`\b(habe|hast|hat|haben|habt|bin|bist|ist|sind|seid)\s+\w+\b`).MatchString(text) {
		patterns = append(patterns, "perfekt")
	}

	// Dative case indicators
	if regexp.MustCompile(`\b(dem|der|den|einem|einer)\b`).MatchString(text) {
		patterns = append(patterns, "dativ")
	}

	// Accusative case indicators
	if regexp.MustCompile(`\b(den|die|das|einen|eine)\b`).MatchString(text) {
		patterns = append(patterns, "akkusativ")
	}

	return patterns
}

// generateExplanations generates explanations for identified patterns
func (s *GrammarService) generateExplanations(text, language string) []string {
	patterns := s.identifyPatterns(text, language)
	var explanations []string

	for _, pattern := range patterns {
		explanation := s.getPatternExplanation(pattern, language)
		if explanation != "" {
			explanations = append(explanations, explanation)
		}
	}

	return explanations
}

// getPatternExplanation returns an explanation for a grammar pattern
func (s *GrammarService) getPatternExplanation(pattern, language string) string {
	explanations := map[string]string{
		// English patterns
		"present_continuous": "Present continuous tense: Used for actions happening now or temporary situations. Formed with 'am/is/are + verb-ing'.",
		"past_tense":         "Past tense: Used for completed actions in the past. Regular verbs add '-ed'.",
		"future_tense":       "Future tense: Used for actions that will happen. Can use 'will + verb' or 'going to + verb'.",
		"conditional":        "Conditional: Used for hypothetical situations. Often uses 'if' clauses with 'would/could/should'.",
		"passive_voice":      "Passive voice: Emphasizes the action or recipient rather than the doer. Formed with 'be + past participle'.",
		"question":           "Question form: Inverts subject and auxiliary verb, or uses question words (who, what, when, etc.).",
		"relative_clause":    "Relative clause: Provides additional information about a noun. Uses 'who', 'which', 'that', etc.",
		"present_perfect":    "Present perfect: Connects past action to present. Uses 'have/has + past participle'.",
		"comparison":         "Comparison: Compares qualities. Uses '-er/-est' or 'more/most'.",

		// Spanish patterns
		"presente":          "Presente: Tiempo verbal para acciones actuales o habituales.",
		"preterito":         "Pretérito: Tiempo verbal para acciones completadas en el pasado.",
		"subjuntivo":        "Subjuntivo: Modo verbal para expresar deseos, dudas o situaciones hipotéticas.",
		"verbos_reflexivos": "Verbos reflexivos: Verbos donde el sujeto y el objeto son la misma persona. Usan pronombres 'me, te, se, nos, os'.",
		"condicional":       "Condicional: Para expresar posibilidades o situaciones hipotéticas.",

		// French patterns
		"présent":       "Présent: Temps pour les actions actuelles ou habituelles.",
		"passé_composé": "Passé composé: Temps pour les actions passées. Utilise 'avoir/être + participe passé'.",
		"imparfait":     "Imparfait: Temps pour décrire des situations passées continues ou habituelles.",
		"subjonctif":    "Subjonctif: Mode pour exprimer le doute, le souhait ou l'émotion.",

		// German patterns
		"modalverben": "Modalverben: können, müssen, sollen, wollen, dürfen, mögen. Verändern die Bedeutung des Hauptverbs.",
		"perfekt":     "Perfekt: Vergangenheitsform mit 'haben/sein + Partizip II'.",
		"dativ":       "Dativ: Indirektes Objekt. Artikel: dem, der, den, einem, einer.",
		"akkusativ":   "Akkusativ: Direktes Objekt. Artikel: den, die, das, einen, eine.",
	}

	if explanation, ok := explanations[pattern]; ok {
		return explanation
	}

	return ""
}

// Helper functions

func countSentences(text string) int {
	// Count sentence-ending punctuation
	return len(regexp.MustCompile(`[.!?]+`).FindAllString(text, -1))
}

func containsSubordinateClause(text, language string) bool {
	subordinators := map[string][]string{
		"en": {"because", "although", "if", "when", "while", "unless", "since", "after", "before"},
		"es": {"porque", "aunque", "si", "cuando", "mientras", "desde que", "después de que"},
		"fr": {"parce que", "bien que", "si", "quand", "pendant que", "depuis que"},
		"de": {"weil", "obwohl", "wenn", "als", "während", "nachdem", "bevor"},
	}

	words, ok := subordinators[language]
	if !ok {
		words = subordinators["en"]
	}

	lowerText := strings.ToLower(text)
	for _, word := range words {
		if strings.Contains(lowerText, word) {
			return true
		}
	}
	return false
}

func containsPassiveVoice(text, language string) bool {
	if language == "en" {
		return regexp.MustCompile(`\b(was|were|been|being)\s+\w+(ed|en)\b`).MatchString(strings.ToLower(text))
	}
	return false
}

func containsConditional(text, language string) bool {
	conditionals := map[string][]string{
		"en": {"would", "could", "should", "might", "if"},
		"es": {"ría", "rías", "ríamos", "rían", "si"},
		"fr": {"rais", "rait", "rions", "riez", "raient", "si"},
		"de": {"würde", "könnte", "sollte", "wenn"},
	}

	words, ok := conditionals[language]
	if !ok {
		words = conditionals["en"]
	}

	lowerText := strings.ToLower(text)
	for _, word := range words {
		if strings.Contains(lowerText, word) {
			return true
		}
	}
	return false
}

func hashText(text string) string {
	// Simple hash for caching
	hash := 0
	for _, c := range text {
		hash = ((hash << 5) - hash) + int(c)
	}
	return fmt.Sprintf("%d", hash)
}

// GetGrammarSuggestions returns basic suggestions tailored by level and language.
func (s *GrammarService) GetGrammarSuggestions(level, language string) ([]string, error) {
	suggestions := []string{}
	base := map[string][]string{
		"en": {"Review conditional sentences", "Practice passive voice transformations", "Drill phrasal verbs in context"},
		"es": {"Repasa el subjuntivo con oraciones condicionales", "Practica perífrasis verbales", "Trabaja colocaciones comunes"},
		"fr": {"Travaille le subjonctif avec 'que'", "Révise le passé composé vs imparfait", "Pratique les pronoms relatifs"},
		"de": {"Übe Nebensätze mit 'weil' und 'dass'", "Wiederhole Perfekt vs Präteritum", "Festige die Kasusartikel"},
	}

	key := language
	if _, ok := base[key]; !ok {
		key = "en"
	}

	suggestions = append(suggestions, base[key]...)

	switch strings.ToUpper(level) {
	case "A1", "A2":
		suggestions = append(suggestions, "Focus on present tense sentence building")
	case "B1", "B2":
		suggestions = append(suggestions, "Incorporate conditional and relative clauses in practice")
	case "C1", "C2":
		suggestions = append(suggestions, "Polish nuance: discourse markers and stylistic variation")
	default:
		suggestions = append(suggestions, "Balance tense review with complex clause practice")
	}

	return suggestions, nil
}

// GenerateGrammarReport returns a lightweight progress summary (placeholder until analytics backend is available).
func (s *GrammarService) GenerateGrammarReport(userID, language string) (map[string]interface{}, error) {
	// In lieu of persisted analytics, surface recent-cache metrics and defaults.
	return map[string]interface{}{
		"userId":           userID,
		"language":         language,
		"recentAnalyses":   0,
		"strengths":        []string{"sentence structure"},
		"focusAreas":       []string{"conditional clauses", "passive voice"},
		"recommendedLevel": "B1",
	}, nil
}

// ---------------------------------------------------------------------------
// AI-Powered Grammar Analysis & Learning (via Ollama)
// ---------------------------------------------------------------------------

// GenerateAIAnalysis uses Ollama to produce a rich grammar analysis,
// enriched with pattern names, descriptions, examples, and a plain‑language summary.
// Falls back to the regex-based analysis if Ollama is unavailable.
func (s *GrammarService) GenerateAIAnalysis(text, language, nativeLanguage string) (*models.AIGrammarAnalysis, error) {
	if s.ollamaURL == "" {
		return s.fallbackAIAnalysis(text, language)
	}

	cacheKey := fmt.Sprintf("ai_grammar:%s:%s:%s", language, nativeLanguage, hashText(text))
	if s.redis != nil {
		cached, err := s.redis.Get(context.Background(), cacheKey).Result()
		if err == nil && cached != "" {
			var result models.AIGrammarAnalysis
			if json.Unmarshal([]byte(cached), &result) == nil {
				return &result, nil
			}
		}
	}

	langName := languageCodeToName(language)
	nativeLangName := languageCodeToName(nativeLanguage)
	if langName == "" {
		langName = language
	}
	if nativeLangName == "" {
		nativeLangName = nativeLanguage
	}

	prompt := fmt.Sprintf(`You are a language tutor. Your student's native language is %s.

Analyze this text and EXPLAIN EVERYTHING IN %s.

Text: "%s"

IDENTIFY THE LANGUAGE: Look at the text and determine what language it is written in (e.g. "I love Spanish" = English, "De repente ya no eras el mismo" = Spanish, etc.).

CRITICAL INSTRUCTION — You MUST write ALL of the following in %s (the student's native language):
- The summary
- Every pattern description
- Every word explanation
- The entire response except for the original text words

Write in SIMPLE, BEGINNER-FRIENDLY language. Avoid complex linguistic terms. Use everyday words.

Return a JSON object (no markdown, no code fences) with these fields:
{
  "difficulty": "CEFR level A1-C2",
  "summary": "First show the FULL SENTENCE TRANSLATION in %s. Then in 1-2 simple sentences explain the grammar.",
  "patterns": [
    {
      "name": "simple grammar name in %s (e.g. 'Acción pasada' for Spanish students, 'Passato prossimo' for Italian students, 'Past tense' for English students)",
      "description": "ONE short sentence in %s explaining when to use this. Keep it simple.",
      "example": "an example sentence in the SAME language as the original text"
    }
  ],
  "detailedBreakdown": [
    {
      "text": "each word from the original text, one at a time",
      "explanation": "Write a SIMPLE explanation in %s. Format: 'means [TRANSLATION] — [SIMPLE GRAMMAR NOTE]'. Example for 'eras': 'significa 'eras' — verbo en tiempo pasado, como 'you were''.",
      "type": "verb|noun|pronoun|preposition|article|adjective|adverb|conjunction|phrase|other"
    }
  ]
}

Remember: ALL explanations MUST be in %s.`, nativeLangName, nativeLangName, text, nativeLangName, nativeLangName, nativeLangName, nativeLangName, nativeLangName, nativeLangName)

	result, err := s.callOllama(prompt)
	if err != nil {
		return s.fallbackAIAnalysis(text, language)
	}

	// Strip markdown code fences and leading/trailing whitespace
	cleaned := strings.TrimSpace(result)
	if strings.HasPrefix(cleaned, "```") {
		if idx := strings.Index(cleaned, "\n"); idx != -1 {
			cleaned = cleaned[idx+1:]
		}
	}
	if idx := strings.LastIndex(cleaned, "```"); idx != -1 {
		cleaned = strings.TrimSpace(cleaned[:idx])
	}
	// Find the first '{' character to strip any non-JSON prefix
	if idx := strings.Index(cleaned, "{"); idx != -1 {
		cleaned = cleaned[idx:]
	}
	// Find the last '}' to strip any non-JSON suffix
	if idx := strings.LastIndex(cleaned, "}"); idx != -1 {
		cleaned = cleaned[:idx+1]
	}

	// Parse the structured response using a flexible approach
	aiResult, err := s.parseAIGrammarAnalysis(cleaned)
	if err != nil {
		return s.fallbackAIAnalysis(text, language)
	}

	// Fill in any missing patterns from regex analysis
	// Note: intentional skip of regex fallback patterns — they're internal identifiers,
	// not useful for users. If AI patterns are empty, we simply don't show patterns.

	// Cache the result
	if s.redis != nil {
		if jsonData, err := json.Marshal(aiResult); err == nil {
			s.redis.Set(context.Background(), cacheKey, jsonData, 24*time.Hour)
		}
	}

	return aiResult, nil
}

// fallbackAIAnalysis returns a regex-based analysis when Ollama is unavailable.
// It builds a human-readable summary from the identified patterns so the grammar
// panel always has something useful to show, and populates GrammarPattern structs
// so the Patterns section renders too.
func (s *GrammarService) fallbackAIAnalysis(text, language string) (*models.AIGrammarAnalysis, error) {
	basic, err := s.AnalyzeGrammar(text, language)
	if err != nil {
		return nil, err
	}

	// Build a readable summary from the identified patterns.
	summary := ""
	if len(basic.Explanations) > 0 {
		summary = strings.Join(basic.Explanations[:min(3, len(basic.Explanations))], " ")
	} else if basic.Difficulty != "" {
		summary = fmt.Sprintf("This text is at a %s level. Click AI Tutor for a detailed explanation.", basic.Difficulty)
	}

	// Convert regex pattern names to GrammarPattern structs.
	var patterns []models.GrammarPattern
	for _, p := range basic.Patterns {
		desc := s.getPatternExplanation(p, language)
		if desc == "" {
			desc = strings.ReplaceAll(p, "_", " ")
		}
		patterns = append(patterns, models.GrammarPattern{
			Name:        strings.ReplaceAll(p, "_", " "),
			Description: desc,
		})
	}

	return &models.AIGrammarAnalysis{
		Difficulty: basic.Difficulty,
		Summary:    summary,
		Patterns:   patterns,
	}, nil
}

// min returns the smaller of two ints (Go 1.20 added a builtin, but keep compatible).
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// parseAIGrammarAnalysis parses the Ollama JSON response flexibly,
// handling cases where detailedBreakdown items have nested objects instead of flat strings.
func (s *GrammarService) parseAIGrammarAnalysis(rawJSON string) (*models.AIGrammarAnalysis, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(rawJSON), &raw); err != nil {
		return nil, err
	}

	result := &models.AIGrammarAnalysis{}

	if d, ok := raw["difficulty"].(string); ok {
		result.Difficulty = d
	}
	if s, ok := raw["summary"].(string); ok {
		result.Summary = s
	}

	// Parse patterns
	if patternsRaw, ok := raw["patterns"].([]interface{}); ok {
		for _, p := range patternsRaw {
			if pm, ok := p.(map[string]interface{}); ok {
				gp := models.GrammarPattern{}
				if n, ok := pm["name"].(string); ok {
					gp.Name = n
				}
				if d, ok := pm["description"].(string); ok {
					gp.Description = d
				}
				if e, ok := pm["example"].(string); ok {
					gp.Example = e
				}
				result.Patterns = append(result.Patterns, gp)
			}
		}
	}

	// Parse detailed breakdown
	if breakdownRaw, ok := raw["detailedBreakdown"].([]interface{}); ok {
		for _, b := range breakdownRaw {
			if bm, ok := b.(map[string]interface{}); ok {
				item := models.BreakdownItem{}

				if t, ok := bm["text"].(string); ok {
					item.Text = t
				}
				if t, ok := bm["type"].(string); ok {
					item.Type = t
				}

				// Explanation can be a string OR a nested object — handle both
				switch exp := bm["explanation"].(type) {
				case string:
					item.Explanation = exp
				case map[string]interface{}:
					// Flatten nested object into a readable string
					parts := []string{}
					for k, v := range exp {
						parts = append(parts, fmt.Sprintf("%s: %v", k, v))
					}
					item.Explanation = strings.Join(parts, ", ")
				default:
					if bm["explanation"] != nil {
						item.Explanation = fmt.Sprintf("%v", bm["explanation"])
					}
				}

				result.DetailedBreakdown = append(result.DetailedBreakdown, item)
			}
		}
	}

	return result, nil
}

// regexPatternsToGrammarPatterns converts regex pattern names to GrammarPattern structs.
func (s *GrammarService) regexPatternsToGrammarPatterns(text, language string) []models.GrammarPattern {
	patterns := s.identifyPatterns(text, language)
	var result []models.GrammarPattern
	for _, p := range patterns {
		desc := s.getPatternExplanation(p, language)
		gp := models.GrammarPattern{
			Name:        p,
			Description: desc,
		}
		_ = gp // keep compiler happy—desc is already used
		result = append(result, models.GrammarPattern{
			Name:        p,
			Description: desc,
		})
	}
	return result
}

// GenerateLearningContent uses Ollama to generate interactive learning content
// for a given text. Supported actions: breakdown, examples, flashcards, custom.
// Prompts ask for plain text responses, so the result is returned as-is without
// attempting JSON parsing.
func (s *GrammarService) GenerateLearningContent(text, language, nativeLanguage, action, customQuery string) (*models.LearningContent, error) {
	if s.ollamaURL == "" {
		return &models.LearningContent{
			Action:           action,
			Content:          "AI learning is not available. Please configure an Ollama instance.",
			Details:          []string{},
			SuggestedActions: []string{"breakdown", "examples", "flashcards"},
		}, nil
	}

	langName := languageCodeToName(language)
	nativeLangName := languageCodeToName(nativeLanguage)
	if langName == "" {
		langName = language
	}
	if nativeLangName == "" {
		nativeLangName = nativeLanguage
	}

	var prompt string
	switch action {
	case "breakdown":
		prompt = fmt.Sprintf(`You are a language tutor teaching %s to a %s speaker. Provide a detailed grammar breakdown.

Text: "%s"

Respond ONLY with a plain text paragraph (no JSON, no markdown, no code fences) explaining the grammar in %s. Cover: sentence structure, verb conjugations, tenses, and any special rules. Make it beginner-friendly.
`, langName, nativeLangName, text, nativeLangName)

	case "examples":
		prompt = fmt.Sprintf(`You are a language tutor teaching %s to a %s speaker. Provide example sentences.

Text: "%s"

Respond ONLY with 3-5 example sentences in %s with their %s translations. Format as a plain text list. No JSON, no markdown, no code fences.
`, langName, nativeLangName, text, langName, nativeLangName)

	case "flashcards":
		prompt = fmt.Sprintf(`You are a language tutor teaching %s to a %s speaker. Create flashcards.

Text: "%s"

Respond ONLY with 3-5 flashcards, one per line, formatted as "Q: question? A: answer". No JSON, no markdown, no code fences.
`, langName, nativeLangName, text)

	case "custom":
		prompt = fmt.Sprintf(`You are a language tutor teaching %s to a %s speaker.

Text: "%s"
Student's question: "%s"

Answer in %s in a helpful, educational way. Respond ONLY with plain text. No JSON, no markdown, no code fences.
`, langName, nativeLangName, text, customQuery, nativeLangName)

	default:
		return nil, fmt.Errorf("unknown learning action: %s", action)
	}

	// Use a 30-second timeout so the AI Tutor panel doesn't hang indefinitely.
	result, err := s.callOllamaWithTimeout(prompt, 30*time.Second)
	if err != nil {
		return &models.LearningContent{
			Action:           action,
			Content:          "Sorry, I couldn't generate learning content right now. Please try again.",
			Details:          []string{},
			SuggestedActions: []string{"breakdown", "examples", "flashcards"},
		}, nil
	}

	// Strip any markdown fences the model may have added despite being told not to.
	cleaned := strings.TrimSpace(result)
	if strings.HasPrefix(cleaned, "```") {
		if idx := strings.Index(cleaned, "\n"); idx != -1 {
			cleaned = cleaned[idx+1:]
		}
		if idx := strings.LastIndex(cleaned, "```"); idx != -1 {
			cleaned = strings.TrimSpace(cleaned[:idx])
		}
	}
	if cleaned == "" {
		cleaned = "No response generated. Please try again."
	}

	// The prompts explicitly request plain text, so return the response directly
	// without attempting JSON parsing. This avoids the "Learning content generated."
	// placeholder that appeared when JSON parsing failed on valid plain-text responses.
	nextActions := nextActionsFor(action)
	return &models.LearningContent{
		Action:           action,
		Content:          cleaned,
		Details:          []string{},
		SuggestedActions: nextActions,
	}, nil
}

// nextActionsFor returns sensible follow-up action suggestions based on what was just done.
func nextActionsFor(action string) []string {
	switch action {
	case "breakdown":
		return []string{"examples", "flashcards", "custom"}
	case "examples":
		return []string{"breakdown", "flashcards", "custom"}
	case "flashcards":
		return []string{"breakdown", "examples", "custom"}
	default:
		return []string{"breakdown", "examples", "flashcards"}
	}
}

// callOllama sends a prompt to Ollama with a 30-second deadline.
func (s *GrammarService) callOllama(prompt string) (string, error) {
	return s.callOllamaWithTimeout(prompt, 30*time.Second)
}

// callOllamaWithTimeout sends a prompt to Ollama and returns the raw text response.
// The provided timeout is applied as a context deadline on the request.
func (s *GrammarService) callOllamaWithTimeout(prompt string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	reqBody := OllamaGenerateRequest{
		Model:  s.ollamaModel,
		Prompt: prompt,
		Stream: false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.ollamaURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var ollamaResp OllamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode Ollama response: %w", err)
	}

	return strings.TrimSpace(ollamaResp.Response), nil
}
