package services

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/redis/go-redis/v9"
)

// GrammarService handles grammar analysis for language learning
type GrammarService struct {
	redis *redis.Client
}

// NewGrammarService creates a new Grammar service
func NewGrammarService(redis *redis.Client) *GrammarService {
	return &GrammarService{
		redis: redis,
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
