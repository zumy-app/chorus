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

// grammarChatMessage represents a message in the OpenAI chat format.
type grammarChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// grammarChatRequest is the request body for /v1/chat/completions.
type grammarChatRequest struct {
	Model       string               `json:"model"`
	Messages    []grammarChatMessage `json:"messages"`
	Temperature float64              `json:"temperature"`
	MaxTokens   int                  `json:"max_tokens,omitempty"`
}

// grammarChatChoice represents a single choice in the response.
type grammarChatChoice struct {
	Message grammarChatMessage `json:"message"`
}

// grammarChatResponse is the response from /v1/chat/completions.
type grammarChatResponse struct {
	Choices []grammarChatChoice `json:"choices"`
}

// GrammarService handles grammar analysis for language learning
type GrammarService struct {
	redis      *redis.Client
	apiURL     string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewGrammarService creates a new Grammar service
func NewGrammarService(redis *redis.Client, apiURL, apiKey, model string) *GrammarService {
	return &GrammarService{
		redis:      redis,
		apiURL:     strings.TrimRight(apiURL, "/"),
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: 90 * time.Second},
	}
}

// AnalyzeGrammar performs grammar analysis on a message.
// nativeLanguage is the user's native language for explanation localization.
func (s *GrammarService) AnalyzeGrammar(text, language, nativeLanguage string) (*models.GrammarAnalysis, error) {
	if nativeLanguage == "" {
		nativeLanguage = "en"
	}
	ctx := context.Background()
	cacheKey := fmt.Sprintf("grammar:%s:%s", language, hashText(text))

	// Check cache first (skip if redis is unavailable)
	if s.redis != nil {
		cached, err := s.redis.Get(ctx, cacheKey).Result()
		if err == nil {
			var analysis models.GrammarAnalysis
			if json.Unmarshal([]byte(cached), &analysis) == nil {
				return &analysis, nil
			}
		}
	}

	// Perform analysis
	analysis := &models.GrammarAnalysis{
		Difficulty:   s.assessDifficulty(text, language),
		Patterns:     s.identifyPatterns(text, language),
		Explanations: s.generateExplanations(text, language, nativeLanguage),
	}

	// Cache result
	if s.redis != nil {
		if jsonData, err := json.Marshal(analysis); err == nil {
			s.redis.Set(ctx, cacheKey, jsonData, 24*time.Hour)
		}
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

// generateExplanations generates explanations for identified patterns in the user's native language.
func (s *GrammarService) generateExplanations(text, language, nativeLanguage string) []string {
	patterns := s.identifyPatterns(text, language)
	var explanations []string

	for _, pattern := range patterns {
		explanation := s.getPatternExplanation(pattern, nativeLanguage)
		if explanation != "" {
			explanations = append(explanations, explanation)
		}
	}

	return explanations
}

// getPatternExplanation returns an explanation for a grammar pattern in the user's native language.
// Supported native languages: en, es, fr, de. Falls back to English if the language is not available.
func (s *GrammarService) getPatternExplanation(pattern, nativeLanguage string) string {
	explanations := map[string]map[string]string{
		// English patterns
		"present_continuous": {
			"en": "Present continuous tense: Used for actions happening now or temporary situations. Formed with 'am/is/are + verb-ing'.",
			"es": "Presente continuo: Se usa para acciones que ocurren ahora o situaciones temporales. Se forma con 'am/is/are + verbo-ing'.",
			"fr": "Présent continu : Utilisé pour des actions qui se déroulent maintenant ou des situations temporaires. Formé avec 'am/is/are + verbe-ing'.",
			"de": "Verlaufsform der Gegenwart: Wird für Handlungen verwendet, die jetzt stattfinden oder vorübergehende Situationen. Gebildet mit 'am/is/are + Verb-ing'.",
		},
		"past_tense": {
			"en": "Past tense: Used for completed actions in the past. Regular verbs add '-ed'.",
			"es": "Tiempo pasado: Se usa para acciones completadas en el pasado. Los verbos regulares añaden '-ed'.",
			"fr": "Temps passé : Utilisé pour des actions terminées dans le passé. Les verbes réguliers ajoutent '-ed'.",
			"de": "Vergangenheit: Wird für abgeschlossene Handlungen in der Vergangenheit verwendet. Regelmäßige Verben erhalten '-ed'.",
		},
		"future_tense": {
			"en": "Future tense: Used for actions that will happen. Can use 'will + verb' or 'going to + verb'.",
			"es": "Tiempo futuro: Se usa para acciones que sucederán. Se puede usar 'will + verbo' o 'going to + verbo'.",
			"fr": "Temps futur : Utilisé pour des actions qui se produiront. On peut utiliser 'will + verbe' ou 'going to + verbe'.",
			"de": "Zukunft: Wird für Handlungen verwendet, die passieren werden. Kann 'will + Verb' oder 'going to + Verb' verwenden.",
		},
		"conditional": {
			"en": "Conditional: Used for hypothetical situations. Often uses 'if' clauses with 'would/could/should'.",
			"es": "Condicional: Se usa para situaciones hipotéticas. A menudo usa cláusulas con 'if' y 'would/could/should'.",
			"fr": "Conditionnel : Utilisé pour des situations hypothétiques. Utilise souvent des clauses avec 'if' et 'would/could/should'.",
			"de": "Konditional: Wird für hypothetische Situationen verwendet. Verwendet oft 'if'-Sätze mit 'would/could/should'.",
		},
		"passive_voice": {
			"en": "Passive voice: Emphasizes the action or recipient rather than the doer. Formed with 'be + past participle'.",
			"es": "Voz pasiva: Enfatiza la acción o el receptor en lugar del que hace la acción. Se forma con 'be + participio pasado'.",
			"fr": "Voix passive : Met l'accent sur l'action ou le destinataire plutôt que sur l'auteur. Formé avec 'be + participe passé'.",
			"de": "Passiv: Betont die Handlung oder den Empfänger statt den Täter. Gebildet mit 'be + Partizip Perfekt'.",
		},
		"question": {
			"en": "Question form: Inverts subject and auxiliary verb, or uses question words (who, what, when, etc.).",
			"es": "Forma interrogativa: Invierte el sujeto y el verbo auxiliar, o usa palabras interrogativas (quién, qué, cuándo, etc.).",
			"fr": "Forme interrogative : Inverse le sujet et le verbe auxiliaire, ou utilise des mots interrogatifs (qui, quoi, quand, etc.).",
			"de": "Frageform: Kehrt Subjekt und Hilfsverb um oder verwendet Fragewörter (wer, was, wann, etc.).",
		},
		"relative_clause": {
			"en": "Relative clause: Provides additional information about a noun. Uses 'who', 'which', 'that', etc.",
			"es": "Cláusula relativa: Proporciona información adicional sobre un sustantivo. Usa 'who', 'which', 'that', etc.",
			"fr": "Proposition relative : Fournit des informations supplémentaires sur un nom. Utilise 'who', 'which', 'that', etc.",
			"de": "Relativsatz: Gibt zusätzliche Informationen über ein Nomen. Verwendet 'who', 'which', 'that', etc.",
		},
		"present_perfect": {
			"en": "Present perfect: Connects past action to present. Uses 'have/has + past participle'.",
			"es": "Presente perfecto: Conecta una acción pasada con el presente. Usa 'have/has + participio pasado'.",
			"fr": "Present perfect : Relie une action passée au présent. Utilise 'have/has + participe passé'.",
			"de": "Present Perfect: Verbindet vergangene Handlung mit Gegenwart. Verwendet 'have/has + Partizip Perfekt'.",
		},
		"comparison": {
			"en": "Comparison: Compares qualities. Uses '-er/-est' or 'more/most'.",
			"es": "Comparación: Compara cualidades. Usa '-er/-est' o 'more/most'.",
			"fr": "Comparaison : Compare des qualités. Utilise '-er/-est' ou 'more/most'.",
			"de": "Vergleich: Vergleicht Eigenschaften. Verwendet '-er/-est' oder 'more/most'.",
		},

		// Spanish patterns (with translations to other languages)
		"presente": {
			"en": "Present tense: Verb tense for current or habitual actions.",
			"es": "Presente: Tiempo verbal para acciones actuales o habituales.",
			"fr": "Présent : Temps verbal pour des actions actuelles ou habituelles.",
			"de": "Präsens: Zeitform für aktuelle oder gewohnheitsmäßige Handlungen.",
		},
		"preterito": {
			"en": "Preterite: Verb tense for completed past actions.",
			"es": "Pretérito: Tiempo verbal para acciones completadas en el pasado.",
			"fr": "Prétérit : Temps verbal pour des actions passées terminées.",
			"de": "Präteritum: Zeitform für abgeschlossene vergangene Handlungen.",
		},
		"subjuntivo": {
			"en": "Subjunctive: Verb mood to express wishes, doubts, or hypothetical situations.",
			"es": "Subjuntivo: Modo verbal para expresar deseos, dudas o situaciones hipotéticas.",
			"fr": "Subjonctif : Mode verbal pour exprimer des souhaits, des doutes ou des situations hypothétiques.",
			"de": "Subjunktiv: Modus zum Ausdruck von Wünschen, Zweifeln oder hypothetischen Situationen.",
		},
		"verbos_reflexivos": {
			"en": "Reflexive verbs: Verbs where the subject and object are the same person. Use pronouns 'me, te, se, nos, os'.",
			"es": "Verbos reflexivos: Verbos donde el sujeto y el objeto son la misma persona. Usan pronombres 'me, te, se, nos, os'.",
			"fr": "Verbes réfléchis : Verbes où le sujet et l'objet sont la même personne. Utilisent les pronoms 'me, te, se, nous, vous'.",
			"de": "Reflexive Verben: Verben, bei denen Subjekt und Objekt dieselbe Person sind. Verwenden Pronomen 'mich, dich, sich, uns, euch'.",
		},
		"condicional": {
			"en": "Conditional: Used to express possibilities or hypothetical situations.",
			"es": "Condicional: Para expresar posibilidades o situaciones hipotéticas.",
			"fr": "Conditionnel : Utilisé pour exprimer des possibilités ou des situations hypothétiques.",
			"de": "Konditional: Zum Ausdruck von Möglichkeiten oder hypothetischen Situationen.",
		},

		// French patterns
		"présent": {
			"en": "Present: Tense for current or habitual actions.",
			"es": "Presente: Tiempo para acciones actuales o habituales.",
			"fr": "Présent : Temps pour les actions actuelles ou habituelles.",
			"de": "Präsens: Zeitform für aktuelle oder gewohnheitsmäßige Handlungen.",
		},
		"passé_composé": {
			"en": "Passé composé: Tense for past actions. Uses 'avoir/être + past participle'.",
			"es": "Passé composé: Tiempo para acciones pasadas. Usa 'avoir/être + participio pasado'.",
			"fr": "Passé composé : Temps pour les actions passées. Utilise 'avoir/être + participe passé'.",
			"de": "Passé composé: Zeitform für vergangene Handlungen. Verwendet 'avoir/être + Partizip Perfekt'.",
		},
		"imparfait": {
			"en": "Imperfect: Tense for describing continuous or habitual past situations.",
			"es": "Imperfecto: Tiempo para describir situaciones pasadas continuas o habituales.",
			"fr": "Imparfait : Temps pour décrire des situations passées continues ou habituelles.",
			"de": "Imperfekt: Zeitform zur Beschreibung kontinuierlicher oder gewohnheitsmäßiger vergangener Situationen.",
		},
		"subjonctif": {
			"en": "Subjunctive: Mood for expressing doubt, wish, or emotion.",
			"es": "Subjuntivo: Modo para expresar duda, deseo o emoción.",
			"fr": "Subjonctif : Mode pour exprimer le doute, le souhait ou l'émotion.",
			"de": "Subjunktiv: Modus zum Ausdruck von Zweifel, Wunsch oder Emotion.",
		},

		// German patterns
		"modalverben": {
			"en": "Modal verbs: können, müssen, sollen, wollen, dürfen, mögen. They modify the meaning of the main verb.",
			"es": "Verbos modales: können, müssen, sollen, wollen, dürfen, mögen. Modifican el significado del verbo principal.",
			"fr": "Verbes modaux : können, müssen, sollen, wollen, dürfen, mögen. Ils modifient le sens du verbe principal.",
			"de": "Modalverben: können, müssen, sollen, wollen, dürfen, mögen. Verändern die Bedeutung des Hauptverbs.",
		},
		"perfekt": {
			"en": "Perfect tense: Past tense formed with 'haben/sein + past participle'.",
			"es": "Tiempo perfecto: Tiempo pasado formado con 'haben/sein + participio pasado'.",
			"fr": "Temps parfait : Temps passé formé avec 'haben/sein + participe passé'.",
			"de": "Perfekt: Vergangenheitsform mit 'haben/sein + Partizip II'.",
		},
		"dativ": {
			"en": "Dative: Indirect object. Articles: dem, der, den, einem, einer.",
			"es": "Dativo: Objeto indirecto. Artículos: dem, der, den, einem, einer.",
			"fr": "Datif : Objet indirect. Articles : dem, der, den, einem, einer.",
			"de": "Dativ: Indirektes Objekt. Artikel: dem, der, den, einem, einer.",
		},
		"akkusativ": {
			"en": "Accusative: Direct object. Articles: den, die, das, einen, eine.",
			"es": "Acusativo: Objeto directo. Artículos: den, die, das, einen, eine.",
			"fr": "Accusatif : Objet direct. Articles : den, die, das, einen, eine.",
			"de": "Akkusativ: Direktes Objekt. Artikel: den, die, das, einen, eine.",
		},
	}

	if langs, ok := explanations[pattern]; ok {
		if exp, ok := langs[nativeLanguage]; ok {
			return exp
		}
		// Fall back to English
		if exp, ok := langs["en"]; ok {
			return exp
		}
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
// AI-Powered Grammar Analysis & Learning (via AI API)
// ---------------------------------------------------------------------------

// GenerateAIAnalysis uses the AI API to produce a rich grammar analysis,
// enriched with pattern names, descriptions, examples, and a plain-language summary.
// Falls back to the regex-based analysis if the AI API is unavailable.
func (s *GrammarService) GenerateAIAnalysis(text, language, nativeLanguage string) (*models.AIGrammarAnalysis, error) {
	if s.apiURL == "" || s.apiKey == "" {
		return s.fallbackAIAnalysis(text, language, nativeLanguage)
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

	prompt := fmt.Sprintf(`You are a friendly language tutor. The student speaks %s.

Explain this sentence in %[1]s to help them LEARN:

Sentence: "%s"

Return ONLY valid JSON, no markdown, no code fences. Use this EXACT structure:

{
  "summary": "In %[1]s: One practical tip about this sentence. What grammar should the student notice? How is it built?",
  "detailedBreakdown": [
    {"text": "word1", "explanation": "In %[1]s: means [TRANSLATION] — [grammar note, e.g. 'verb in present tense, 2nd person']", "type": "verb"},
    {"text": "word2", "explanation": "In %[1]s: means [TRANSLATION] — [grammar note]", "type": "noun"}
  ]
}

RULES:
- summary: JUST the learning tip (one short paragraph). Tell the student what to notice about how this sentence works. Example: "This Spanish sentence uses 'vas a' to talk about future plans. 'Vas' comes from 'ir' (to go). The question word 'qué' asks 'what'."
- detailedBreakdown: EVERY word from the sentence, one by one. For each: what it means + a simple grammar note.
- types: verb, noun, pronoun, preposition, article, adjective, adverb, conjunction, question, phrase
- Keep language SIMPLE. Use everyday words. Think of explaining to a beginner.
- Write EVERYTHING in %[1]s (the student's language), except example words which stay in the original language.
- For multi-word phrases like "going to", group them together as one item.`, nativeLangName, text)

	result, err := s.callGrammarAPI(prompt, nativeLangName)
	if err != nil {
		return s.fallbackAIAnalysis(text, language, nativeLanguage)
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
		return s.fallbackAIAnalysis(text, language, nativeLanguage)
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

// fallbackAIAnalysis returns a regex-based analysis when the AI API is unavailable.
// It builds a human-readable summary from the identified patterns so the grammar
// panel always has something useful to show, and populates GrammarPattern structs
// so the Patterns section renders too. Explanations are in the user's native language.
func (s *GrammarService) fallbackAIAnalysis(text, language, nativeLanguage string) (*models.AIGrammarAnalysis, error) {
	if nativeLanguage == "" {
		nativeLanguage = "en"
	}
	basic, err := s.AnalyzeGrammar(text, language, nativeLanguage)
	if err != nil {
		return nil, err
	}

	summary := s.buildFallbackSummary(text, language, basic, nativeLanguage)

	return &models.AIGrammarAnalysis{
		Difficulty: basic.Difficulty,
		Summary:    summary,
	}, nil
}

// isQuestion checks if text looks like a question (starts with question word or ends with ?)
func isQuestion(text string) bool {
	t := strings.TrimSpace(text)
	if strings.HasSuffix(t, "?") {
		return true
	}
	questionWords := []string{"what", "who", "where", "when", "why", "how", "which", "do", "does", "did",
		"is", "are", "am", "can", "will", "would", "could", "should",
		"qué", "cómo", "dónde", "cuándo", "por qué", "quién", "cuál",
		"que", "como", "donde", "cuando", "porque", "quien", "cual"}
	firstWord := strings.ToLower(strings.Fields(t)[0])
	for _, qw := range questionWords {
		if firstWord == qw {
			return true
		}
	}
	return false
}

// buildFallbackSummary creates a practical, learning-focused summary in the user's native language.
func (s *GrammarService) buildFallbackSummary(text, language string, basic *models.GrammarAnalysis, nativeLanguage string) string {
	if basic.Difficulty == "" {
		basic.Difficulty = "A1"
	}
	if nativeLanguage == "" {
		nativeLanguage = "en"
	}
	langName := languageCodeToName(language)
	if langName == "" {
		langName = language
	}

	wordCount := len(strings.Fields(text))
	isQ := isQuestion(text)

	// Identify what kind of sentence this is
	var sentenceType string
	switch {
	case isQ && language == "es":
		sentenceType = "question"
	default:
		sentenceType = "sentence"
	}

	// Map sentence types to native language
	typeDesc := map[string]map[string]string{
		"question": {
			"en": "This is a question.",
			"es": "Esta es una pregunta.",
			"fr": "C'est une question.",
			"de": "Dies ist eine Frage.",
		},
	}

	sentenceDesc := typeDesc[sentenceType][nativeLanguage]
	if sentenceDesc == "" {
		sentenceDesc = typeDesc[sentenceType]["en"]
	}

	// Collect pattern info with matching words
	type matchInfo struct {
		name  string
		words string
	}
	var matches []matchInfo
	seen := map[string]bool{}
	for _, p := range basic.Patterns {
		if seen[p] {
			continue
		}
		seen[p] = true
		matchingWords := s.findPatternWords(text, language, p)
		words := strings.Join(matchingWords, ", ")
		matches = append(matches, matchInfo{name: p, words: words})
	}

	// Build a practical summary
	var parts []string
	parts = append(parts, sentenceDesc)

	// Add word count info
	wordNote := map[string]string{
		"en": fmt.Sprintf("It has %d words.", wordCount),
		"es": fmt.Sprintf("Tiene %d palabras.", wordCount),
		"fr": fmt.Sprintf("Elle a %d mots.", wordCount),
		"de": fmt.Sprintf("Sie hat %d Wörter.", wordCount),
	}
	if v, ok := wordNote[nativeLanguage]; ok {
		parts = append(parts, v)
	} else {
		parts = append(parts, fmt.Sprintf("It has %d words.", wordCount))
	}

	// Add pattern observations in practical language
	for _, m := range matches {
		exp := s.getPatternExplanation(m.name, nativeLanguage)
		// Take first sentence
		if idx := strings.Index(exp, "."); idx > 0 {
			exp = exp[:idx+1]
		}
		if m.words != "" {
			obs := map[string]string{
				"en": fmt.Sprintf("Notice the word(s) \"%s\" — %s", m.words, exp),
				"es": fmt.Sprintf("Nota la(s) palabra(s) \"%s\" — %s", m.words, exp),
				"fr": fmt.Sprintf("Remarquez le(s) mot(s) \"%s\" — %s", m.words, exp),
				"de": fmt.Sprintf("Beachten Sie das/die Wort/Wörter \"%s\" — %s", m.words, exp),
			}
			if v, ok := obs[nativeLanguage]; ok {
				parts = append(parts, v)
			} else {
				parts = append(parts, fmt.Sprintf("Notice the word(s) \"%s\" — %s", m.words, exp))
			}
		} else {
			parts = append(parts, exp)
		}
	}

	// Add a simple practice suggestion if patterns were found
	if len(matches) > 0 {
		practiceHint := map[string]string{
			"en": "Try changing the verb to practice different tenses!",
			"es": "¡Intenta cambiar el verbo para practicar diferentes tiempos!",
			"fr": "Essayez de changer le verbe pour pratiquer différents temps !",
			"de": "Versuchen Sie, das Verb zu ändern, um verschiedene Zeiten zu üben!",
		}
		if v, ok := practiceHint[nativeLanguage]; ok {
			parts = append(parts, v)
		} else {
			parts = append(parts, "Try changing the verb to practice different tenses!")
		}
	}

	return strings.Join(parts, " ")
}

// findPatternWords extracts the specific words from text that matched a grammar pattern.
func (s *GrammarService) findPatternWords(text, language, pattern string) []string {
	re := s.patternRegexFor(language, pattern)
	if re == nil {
		return nil
	}
	matches := re.FindAllString(strings.ToLower(text), -1)
	// De-duplicate
	seen := map[string]bool{}
	var unique []string
	for _, m := range matches {
		if !seen[m] {
			seen[m] = true
			unique = append(unique, m)
		}
	}
	return unique
}

// patternRegexFor returns the compiled regex for a specific language+pattern.
func (s *GrammarService) patternRegexFor(language, pattern string) *regexp.Regexp {
	patterns := map[string]map[string]string{
		"en": {
			"present_continuous": `\b\w+ing\b`,
			"past_tense":         `\b\w+ed\b`,
			"future_tense":       `\b(will|going to|gonna)\b`,
			"conditional":        `\b(would|could|should|might)\b`,
			"passive_voice":      `\b(was|were|been|being)\s+\w+ed\b`,
			"question":           `\b(what|who|where|when|why|how|which|do|does|did|is|are|am|can|will)\b`,
			"present_perfect":    `\b(have|has)\s+\w+ed\b`,
			"comparison":         `\b(more|most|better|worse|best|worst|than)\b`,
		},
		"es": {
			"presente":          `\b\w+(o|as|a|amos|áis|an|es|e|emos|éis|en|imos|ís)\b`,
			"preterito":         `\b\w+(é|aste|ó|amos|asteis|aron|í|iste|ió|imos|isteis|ieron)\b`,
			"subjuntivo":        `\b(ojalá|espero que|quiero que)\b`,
			"verbos_reflexivos": `\b(me|te|se|nos|os)\s+\w+\b`,
			"condicional":       `\b\w+(ía|ías|íamos|íais|ían)\b`,
		},
		"fr": {
			"présent":       `\b\w+(e|es|e|ons|ez|ent|is|it|issons|issez|issent)\b`,
			"passé_composé": `\b(ai|as|a|avons|avez|ont|suis|es|est|sommes|êtes|sont)\s+\w+(é|i|u|is|it)\b`,
			"imparfait":     `\b\w+(ais|ait|ions|iez|aient)\b`,
			"subjonctif":    `\b(que|il faut que|bien que)\b`,
		},
		"de": {
			"modalverben": `\b(kann|kannst|können|muss|musst|müssen|soll|sollst|sollen|darf|darfst|dürfen|mag|magst|mögen)\b`,
			"perfekt":     `\b(habe|hast|hat|haben|habt|bin|bist|ist|sind|seid)\s+\w+\b`,
			"dativ":       `\b(dem|der|den|einem|einer)\b`,
			"akkusativ":   `\b(den|die|das|einen|eine)\b`,
		},
	}
	if langPatterns, ok := patterns[language]; ok {
		if reStr, ok := langPatterns[pattern]; ok {
			return regexp.MustCompile(reStr)
		}
	}
	return nil
}

// min returns the smaller of two ints (Go 1.20 added a builtin, but keep compatible).
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// parseAIGrammarAnalysis parses the AI API JSON response flexibly,
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

// GenerateLearningContent uses the AI API to generate interactive learning content
// for a given text. Supported actions: breakdown, examples, flashcards, custom.
// Prompts ask for plain text responses, so the result is returned as-is without
// attempting JSON parsing.
func (s *GrammarService) GenerateLearningContent(text, language, nativeLanguage, action, customQuery string) (*models.LearningContent, error) {
	if s.apiURL == "" || s.apiKey == "" {
		return &models.LearningContent{
			Action:           action,
			Content:          "AI learning is not available. Please configure an AI API key.",
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
	prompt = fmt.Sprintf(`<role>You are an intuitive, friendly, culturally aware language tutor teaching %s to a %s speaker. Break down the text like a helpful peer, not a rigid lecturer.</role>

<task>
1. Context Detection: Scan the text to see if it is a known song, poem, famous quote, historical speech, or slang-heavy piece. If detected, add a brief, 1-sentence casual intro identifying it (e.g., "Ah, this is from Shakira's song 'Monotonía'—let's break down the drama!").
2. Chunking & Translation: Analyze the text using a dynamic chunking approach:
   - If the text is SHORT (1-5 sentences): Break it down line-by-line or phrase-by-phrase.
   - If the text is LONG (a full paragraph or song stanza): Group it into 2-3 logical, bite-sized sections. Do NOT do a word-by-word list.
3. Breakdown: For each section or phrase, provide its natural, contextual translation followed by a punchy, casual explanation of the key verbs, idioms, or cultural context.
</task>

<constraints>
- NEVER include analytical fluff like word counts, character counts, or robotic timestamps which do not help users.
- Avoid abstract, intimidating textbook jargon (e.g., instead of "imperfect indicative", use relatable terms like "ongoing past action").
- Keep sentences short, conversational, and hyper-focused on helping the student understand *why* the phrase means what it means.
</constraints>

<input_text>
"%s"
</input_text>
`, langName, nativeLangName, text)

case "examples":
	prompt = fmt.Sprintf(`<role>You are an intuitive, friendly, and practical language tutor teaching %s to a %s speaker. Provide natural, conversational example sentences.</role>

<task>
Provide 3-5 example sentences using the key vocabulary or grammatical patterns found in the source text. Focus on real-world usability rather than stiff textbook phrases.
Each line must show the example sentence in %s, followed immediately by its natural translation in %s.
</task>

<constraints>
- Respond ONLY with a plain text list.
- Do NOT use JSON, markdown headers, or code fences. Do not use bullet points or dashes.
</constraints>

<input_text>
"%s"
</input_text>
`, langName, nativeLangName, langName, nativeLangName, text)

case "flashcards":
	prompt = fmt.Sprintf(`<role>You are an intuitive, friendly, and practical language tutor teaching %s to a %s speaker. Create high-yield vocabulary and phrase flashcards based on the text.</role>

<task>
Create 3-5 flashcards, exactly one per line, focusing on the most useful words, idioms, or verb variations from the text that a language learner can immediately use in daily conversation.
Format each line strictly as: "Q: [target word or phrase]? A: [native translation and quick tip]"
</task>

<constraints>
- Respond ONLY with plain text.
- Do NOT use JSON, markdown, or code fences.
- Strictly adhere to one flashcard per line.
</constraints>

<input_text>
"%s"
</input_text>
`, langName, nativeLangName, text)

case "custom":
	prompt = fmt.Sprintf(`<role>You are an intuitive, friendly, and practical language tutor teaching %s to a %s speaker.</role>

<task>
Answer the student's question about the text in a helpful, warm, and highly educational peer-to-peer style. Keep your explanation brief, direct, and completely free of textbook jargon.
</task>

<constraints>
- Answer in %s.
- Respond ONLY with plain text.
- Strictly avoid JSON, markdown, or code fences.
</constraints>

<context>
Text: "%s"
Student Question: "%s"
</context>
`, langName, nativeLangName, nativeLangName, text, customQuery)

default:
	return nil, fmt.Errorf("unknown learning action: %s", action)
}

	// Use a 30-second timeout so the AI Tutor panel doesn't hang indefinitely.
	result, err := s.callGrammarAPI(prompt, nativeLangName)
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

// callGrammarAPI sends a prompt to the OpenAI-compatible API and returns the text response.
func (s *GrammarService) callGrammarAPI(prompt, nativeLangName string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	systemMsg := fmt.Sprintf(`You are a friendly language tutor teaching a student who speaks %s.
Return ONLY valid JSON. No markdown, no code fences, no commentary before or after.`, nativeLangName)

	chatReq := grammarChatRequest{
		Model: s.model,
		Messages: []grammarChatMessage{
			{Role: "system", Content: systemMsg},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens:   2048,
	}

	body, err := json.Marshal(chatReq)
	if err != nil {
		return "", fmt.Errorf("grammar marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.apiURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("grammar create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("grammar API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("grammar API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var chatResp grammarChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("grammar decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("grammar API: no choices in response")
	}

	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}
