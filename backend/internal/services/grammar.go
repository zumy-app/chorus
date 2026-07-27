package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/pkg/logutil"
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
	Temperature float64              `json:"temperature,omitempty"`
	MaxTokens   int                  `json:"max_tokens,omitempty"`
}

// ollamaChatRequest is the request body for Ollama's /api/chat endpoint.
// Ollama wraps model params inside an "options" object and uses num_predict instead of max_tokens.
type ollamaChatRequest struct {
	Model    string               `json:"model"`
	Messages []grammarChatMessage `json:"messages"`
	Stream   bool                 `json:"stream"`
	Options  map[string]any       `json:"options,omitempty"`
}

// grammarChatChoice represents a single choice in the response.
type grammarChatChoice struct {
	Message grammarChatMessage `json:"message"`
}

// grammarChatResponse is the response from /v1/chat/completions.
type grammarChatResponse struct {
	Choices []grammarChatChoice `json:"choices"`
}

// GrammarEndpoint holds the config for a single AI API endpoint in the chain.
type GrammarEndpoint struct {
	Name         string
	ProviderType string // "opencode", "openai", "ollama", etc.
	APIURL       string
	APIKey       string
	Model        string
	Timeout      time.Duration
	client       *http.Client
}

// NewGrammarEndpoint creates a ready-to-use endpoint.
func NewGrammarEndpoint(name, providerType, apiURL, apiKey, model string, timeout int) GrammarEndpoint {
	d := 99999999 * time.Second
	if timeout > 0 {
		d = time.Duration(timeout) * time.Second
	}
	return GrammarEndpoint{
		Name:         name,
		ProviderType: providerType,
		APIURL:       strings.TrimRight(apiURL, "/"),
		APIKey:       apiKey,
		Model:        model,
		Timeout:      d,
		client:       &http.Client{Timeout: d},
	}
}

// GrammarService handles grammar analysis for language learning
type GrammarService struct {
	redis     *redis.Client
	endpoints []GrammarEndpoint
}

// NewGrammarService creates a new Grammar service with an ordered list of endpoints.
// Endpoints are tried in sequence; if one fails (e.g. 429), the next is tried.
func NewGrammarService(redis *redis.Client, endpoints []GrammarEndpoint) *GrammarService {
	return &GrammarService{
		redis:     redis,
		endpoints: endpoints,
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
// Falls back to the regex-based analysis if all AI endpoints are unavailable.
// Returns the analysis, the provider name that succeeded (or "regex-fallback"), and any error.
func (s *GrammarService) GenerateAIAnalysis(text, language, nativeLanguage string) (*models.AIGrammarAnalysis, string, error) {
	if len(s.endpoints) == 0 {
		log.Printf("[Grammar] no AI endpoints configured, using regex fallback")
		analysis, err := s.fallbackAIAnalysis(text, language, nativeLanguage)
		return analysis, "regex-fallback", err
	}

	// Cache version — bump this to invalidate all cached grammar analyses after prompt/model changes.
	const cacheVersion = "v2"
	cacheKey := fmt.Sprintf("ai_grammar:%s:%s:%s:%s", cacheVersion, language, nativeLanguage, hashText(text))
	if s.redis != nil {
		cached, err := s.redis.Get(context.Background(), cacheKey).Result()
		if err == nil && cached != "" {
			var result models.AIGrammarAnalysis
			if json.Unmarshal([]byte(cached), &result) == nil {
				return &result, "cache", nil
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

	prompt := fmt.Sprintf(`<role>You are a friendly, practical language tutor. The student speaks %[1]s and is learning %[4]s. Your goal is to help them understand a sentence in a simple, conversational way — like a helpful friend, not a textbook.</role>

<task>Analyze this sentence and return a structured JSON response:

Sentence: "%[2]s"
</task>

<output_format>
Return ONLY valid JSON (no markdown, no code fences). Use this EXACT structure:

{
  "difficulty": "A1-C2 level estimate",
  "summary": "In %[1]s: A short, friendly overview of what the sentence means and what's interesting about it. Write 2-3 sentences max. Focus on the overall meaning and any tricky parts.",
  "sentenceStructure": "In %[1]s: Briefly explain how the sentence is organized. For example: 'The first part talks about X, then it shifts to Y' or 'This is a question that starts with a question word'.",
  "keyPhrases": [
    {"phrase": "original phrase in %[4]s", "translation": "what it means in %[1]s", "context": "when or how you'd use this phrase in real life"}
  ],
  "detailedBreakdown": [
    {"text": "word or short phrase", "translation": "meaning in %[1]s", "role": "simple description like 'action word' or 'describing word' or 'connector'", "type": "verb|noun|pronoun|preposition|article|adjective|adverb|conjunction|phrase", "note": "optional: a brief, plain-language grammar note if the word is interesting"}
  ],
  "grammarNotes": [
    {"title": "Simple, jargon-free title like 'Talking about the past' or 'Expressing a wish'", "explanation": "In %[1]s: Plain-language explanation of the grammar concept. No technical terms like 'subjunctive' or 'preterite'. Explain it like you would to a friend.", "examples": ["word1", "word2"]}
  ]
}
</output_format>

<rules>
- summary: Be conversational and practical. NEVER mention word counts. NEVER say "Notice the word(s)...". Just explain what the sentence means and what's interesting.
- sentenceStructure: Keep it to 1-2 sentences. Describe how the sentence flows.
- keyPhrases: Pick 2-4 important or interesting phrases. Skip obvious words like "the" or "and".
- detailedBreakdown: Cover EVERY word or short phrase from the sentence. For each: what it means, what role it plays (in simple terms), and an optional grammar note if it's interesting.
- grammarNotes: Pick 2-4 grammar concepts from the sentence. Use SIMPLE titles — never use terms like "Subjunctive", "Preterite", "Reflexive", "Conditional". Instead use titles like "Talking about wishes", "Describing past actions", "Actions that reflect back". Explain each concept in everyday language.
- types for breakdown: verb, noun, pronoun, preposition, article, adjective, adverb, conjunction, phrase
- Write EVERYTHING in %[1]s (the student's language), except example words which stay in the original language (%[4]s).
- For multi-word phrases like "going to" or "se llama", group them as one item.
- Keep ALL language simple and accessible. Imagine explaining to someone who has never studied grammar formally.
</rules>`, nativeLangName, text, "", langName)

	logutil.Infof("[Grammar] Calling AI API for text: %.50s... (language=%s, native=%s)", text, language, nativeLanguage)
	result, providerUsed, err := s.callGrammarAPI(prompt, nativeLangName)
	if err != nil {
		logutil.Errorf("[Grammar] all AI endpoints failed: %v", err)
		analysis, err := s.fallbackAIAnalysis(text, language, nativeLanguage)
		return analysis, "regex-fallback", err
	}
	logutil.Infof("[Grammar] provider %q returned %d characters", providerUsed, len(result))

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
		logutil.Warnf("[Grammar] JSON parsing failed, falling back to regex: %v", err)
		analysis, err := s.fallbackAIAnalysis(text, language, nativeLanguage)
		return analysis, "regex-fallback", err
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

	return aiResult, providerUsed, nil
}

// fallbackAIAnalysis returns a regex-based analysis when the AI API is unavailable.
// It builds a human-readable summary from the identified patterns so the grammar
// panel always has something useful to show. Explanations are in the user's native language.
func (s *GrammarService) fallbackAIAnalysis(text, language, nativeLanguage string) (*models.AIGrammarAnalysis, error) {
	if nativeLanguage == "" {
		nativeLanguage = "en"
	}
	basic, err := s.AnalyzeGrammar(text, language, nativeLanguage)
	if err != nil {
		return nil, err
	}

	summary := s.buildFallbackSummary(text, language, basic, nativeLanguage)
	grammarNotes := s.buildFallbackGrammarNotes(text, language, basic, nativeLanguage)

	return &models.AIGrammarAnalysis{
		Difficulty:   basic.Difficulty,
		Summary:      summary,
		GrammarNotes: grammarNotes,
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

	isQ := isQuestion(text)

	var parts []string

	if isQ {
		qDesc := map[string]string{
			"en": "This is a question.",
			"es": "Esta es una pregunta.",
			"fr": "C'est une question.",
			"de": "Dies ist eine Frage.",
		}
		if v, ok := qDesc[nativeLanguage]; ok {
			parts = append(parts, v)
		} else {
			parts = append(parts, qDesc["en"])
		}
	}

	// Describe patterns in plain language
	plainTitles := map[string]map[string]string{
		"present_continuous": {"en": "This sentence describes something happening right now.", "es": "Esta oración describe algo que está pasando ahora mismo."},
		"past_tense":         {"en": "This sentence talks about something that already happened.", "es": "Esta oración habla de algo que ya pasó."},
		"future_tense":       {"en": "This sentence talks about something that will happen.", "es": "Esta oración habla de algo que va a pasar."},
		"conditional":        {"en": "This sentence expresses a possibility or hypothetical situation.", "es": "Esta oración expresa una posibilidad o situación hipotética."},
		"passive_voice":      {"en": "This sentence focuses on the action rather than who did it.", "es": "Esta oración se enfoca en la acción más que en quién la hizo."},
		"question":           {"en": "This is a question asking for information.", "es": "Esta es una pregunta que busca información."},
		"present_perfect":    {"en": "This sentence connects a past action to the present.", "es": "Esta oración conecta una acción pasada con el presente."},
		"comparison":         {"en": "This sentence compares things.", "es": "Esta oración compara cosas."},
		"presente":           {"en": "This sentence describes something happening now or a general truth.", "es": "Esta oración describe algo que pasa ahora o una verdad general."},
		"preterito":          {"en": "This sentence talks about a completed action in the past.", "es": "Esta oración habla de una acción completada en el pasado."},
		"subjuntivo":         {"en": "This sentence expresses a wish, doubt, or something uncertain.", "es": "Esta oración expresa un deseo, duda o algo incierto."},
		"verbos_reflexivos":  {"en": "This sentence uses verbs where the action reflects back on the person doing it.", "es": "Esta oración usa verbos donde la acción se refleja en quien la hace."},
		"condicional":        {"en": "This sentence talks about what would happen under certain conditions.", "es": "Esta oración habla de lo que pasaría bajo ciertas condiciones."},
	}

	seen := map[string]bool{}
	for _, p := range basic.Patterns {
		if seen[p] {
			continue
		}
		seen[p] = true
		if titles, ok := plainTitles[p]; ok {
			if v, ok := titles[nativeLanguage]; ok {
				parts = append(parts, v)
			} else if v, ok := titles["en"]; ok {
				parts = append(parts, v)
			}
		}
	}

	if len(parts) == 0 {
		defaults := map[string]string{
			"en": "This is a straightforward sentence. Tap on the words below to see what each one means.",
			"es": "Esta es una oración sencilla. Toca las palabras abajo para ver qué significa cada una.",
			"fr": "C'est une phrase simple. Appuyez sur les mots ci-dessous pour voir ce que chacun signifie.",
			"de": "Dies ist ein einfacher Satz. Tippen Sie auf die Wörter unten, um zu sehen, was jedes bedeutet.",
		}
		if v, ok := defaults[nativeLanguage]; ok {
			return v
		}
		return defaults["en"]
	}

	return strings.Join(parts, " ")
}

// buildFallbackGrammarNotes creates plain-language grammar notes from regex patterns.
func (s *GrammarService) buildFallbackGrammarNotes(text, language string, basic *models.GrammarAnalysis, nativeLanguage string) []models.GrammarNote {
	plainNotes := map[string]map[string]models.GrammarNote{
		"present_continuous": {
			"en": {Title: "Happening right now", Explanation: "When you want to talk about something happening at this very moment, you use this form. It's built with 'am/is/are' plus a word ending in '-ing'.", Examples: []string{}},
			"es": {Title: "Pasando ahora mismo", Explanation: "Cuando quieres hablar de algo que está pasando en este momento, usas esta forma. Se construye con 'am/is/are' más una palabra que termina en '-ing'.", Examples: []string{}},
		},
		"past_tense": {
			"en": {Title: "Talking about the past", Explanation: "This is how you describe something that already happened. Regular words add '-ed' at the end, but many common words change completely (like 'go' becomes 'went').", Examples: []string{}},
			"es": {Title: "Hablando del pasado", Explanation: "Así es como describes algo que ya pasó. Las palabras regulares añaden '-ed' al final, pero muchas palabras comunes cambian completamente (como 'go' se convierte en 'went').", Examples: []string{}},
		},
		"future_tense": {
			"en": {Title: "Talking about what's coming", Explanation: "When you want to talk about something that hasn't happened yet, you use 'will' or 'going to' before the action word.", Examples: []string{}},
			"es": {Title: "Hablando de lo que viene", Explanation: "Cuando quieres hablar de algo que todavía no ha pasado, usas 'will' o 'going to' antes de la palabra de acción.", Examples: []string{}},
		},
		"conditional": {
			"en": {Title: "What if...?", Explanation: "This is used when talking about something that might happen or could happen under certain conditions. Words like 'would', 'could', and 'if' are common here.", Examples: []string{}},
			"es": {Title: "¿Qué pasaría si...?", Explanation: "Se usa cuando hablas de algo que podría pasar bajo ciertas condiciones. Palabras como 'would', 'could' e 'if' son comunes aquí.", Examples: []string{}},
		},
		"presente": {
			"en": {Title: "Describing now or general truths", Explanation: "This form is used for things happening now or things that are generally true. The word endings change depending on who is doing the action (I, you, he/she, we, they).", Examples: []string{}},
			"es": {Title: "Describiendo el ahora o verdades generales", Explanation: "Esta forma se usa para cosas que pasan ahora o que son generalmente ciertas. Las terminaciones cambian dependiendo de quién hace la acción (yo, tú, él/ella, nosotros, ellos).", Examples: []string{}},
		},
		"preterito": {
			"en": {Title: "Completed actions in the past", Explanation: "This form describes actions that started and finished in the past. The word endings change depending on who did the action.", Examples: []string{}},
			"es": {Title: "Acciones completadas en el pasado", Explanation: "Esta forma describe acciones que empezaron y terminaron en el pasado. Las terminaciones cambian dependiendo de quién hizo la acción.", Examples: []string{}},
		},
		"subjuntivo": {
			"en": {Title: "Expressing wishes or uncertainty", Explanation: "This form is used when talking about things that aren't certain — wishes, doubts, emotions, or things you hope will happen. It often appears after words like 'que' or 'ojalá'.", Examples: []string{}},
			"es": {Title: "Expresando deseos o incertidumbre", Explanation: "Esta forma se usa cuando hablas de cosas que no son seguras — deseos, dudas, emociones o cosas que esperas que pasen. A menudo aparece después de palabras como 'que' u 'ojalá'.", Examples: []string{}},
		},
		"verbos_reflexivos": {
			"en": {Title: "Actions that reflect back", Explanation: "Some verbs describe actions that a person does to themselves — like washing yourself, calling yourself, or getting yourself ready. These verbs use small words like 'me', 'te', 'se' before the action word.", Examples: []string{}},
			"es": {Title: "Acciones que se reflejan", Explanation: "Algunos verbos describen acciones que una persona se hace a sí misma — como lavarse, llamarse o prepararse. Estos verbos usan palabritas como 'me', 'te', 'se' antes de la palabra de acción.", Examples: []string{}},
		},
		"condicional": {
			"en": {Title: "What would happen", Explanation: "This form talks about what would happen under certain conditions. It's like saying 'I would go' or 'she would eat'. The word endings typically include 'ía'.", Examples: []string{}},
			"es": {Title: "Lo que pasaría", Explanation: "Esta forma habla de lo que pasaría bajo ciertas condiciones. Es como decir 'yo iría' o 'ella comería'. Las terminaciones suelen incluir 'ía'.", Examples: []string{}},
		},
	}

	var notes []models.GrammarNote
	seen := map[string]bool{}
	for _, p := range basic.Patterns {
		if seen[p] {
			continue
		}
		seen[p] = true
		if noteMap, ok := plainNotes[p]; ok {
			if note, ok := noteMap[nativeLanguage]; ok {
				words := s.findPatternWords(text, language, p)
				note.Examples = words
				notes = append(notes, note)
			} else if note, ok := noteMap["en"]; ok {
				words := s.findPatternWords(text, language, p)
				note.Examples = words
				notes = append(notes, note)
			}
		}
	}

	return notes
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
// handling both the new rich format and the legacy format.
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
	if ss, ok := raw["sentenceStructure"].(string); ok {
		result.SentenceStructure = ss
	}

	// Parse keyPhrases
	if kpRaw, ok := raw["keyPhrases"].([]interface{}); ok {
		for _, kp := range kpRaw {
			if km, ok := kp.(map[string]interface{}); ok {
				phrase := models.KeyPhrase{}
				if p, ok := km["phrase"].(string); ok {
					phrase.Phrase = p
				}
				if t, ok := km["translation"].(string); ok {
					phrase.Translation = t
				}
				if c, ok := km["context"].(string); ok {
					phrase.Context = c
				}
				result.KeyPhrases = append(result.KeyPhrases, phrase)
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
				if t, ok := bm["translation"].(string); ok {
					item.Translation = t
				}
				if r, ok := bm["role"].(string); ok {
					item.Role = r
				}
				if n, ok := bm["note"].(string); ok {
					item.Note = n
				}

				// Explanation can be a string OR a nested object — handle both
				switch exp := bm["explanation"].(type) {
				case string:
					item.Explanation = exp
				case map[string]interface{}:
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

	// Parse grammarNotes
	if gnRaw, ok := raw["grammarNotes"].([]interface{}); ok {
		for _, gn := range gnRaw {
			if gm, ok := gn.(map[string]interface{}); ok {
				note := models.GrammarNote{}
				if t, ok := gm["title"].(string); ok {
					note.Title = t
				}
				if e, ok := gm["explanation"].(string); ok {
					note.Explanation = e
				}
				if exRaw, ok := gm["examples"].([]interface{}); ok {
					for _, ex := range exRaw {
						if exStr, ok := ex.(string); ok {
							note.Examples = append(note.Examples, exStr)
						}
					}
				}
				result.GrammarNotes = append(result.GrammarNotes, note)
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
	if len(s.endpoints) == 0 {
		return &models.LearningContent{
			Action:           action,
			Content:          "AI learning is not available. Please configure an AI API endpoint.",
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
- Respond ONLY with plain text. Emoticons are ok. Just no JSON or XML 
- Do NOT use JSON, markdown headers, code fences, or list bullets.
- NEVER include analytical fluff like word counts, character counts, or robotic timestamps.
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
	result, providerUsed, err := s.callGrammarAPI(prompt, nativeLangName)
	if err != nil {
		return &models.LearningContent{
			Action:           action,
			Content:          "Sorry, I couldn't generate learning content right now. Please try again.",
			Details:          []string{},
			SuggestedActions: []string{"breakdown", "examples", "flashcards"},
		}, nil
	}
	log.Printf("[Grammar] Learning content via %q", providerUsed)

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

// callGrammarAPI tries each configured endpoint in sequence.
// Returns the response text and the name of the provider that succeeded.
func (s *GrammarService) callGrammarAPI(prompt, nativeLangName string) (string, string, error) {
	var lastErr error
	for i, ep := range s.endpoints {
		logutil.Infof("[Grammar] trying endpoint %d/%d: %s (url=%s, model=%s)",
			i+1, len(s.endpoints), ep.Name, ep.APIURL, ep.Model)
		start := time.Now()
		result, err := ep.call(prompt, nativeLangName)
		logutil.Duration("Grammar", start, ep.Name)
		if err == nil {
			logutil.Infof("[Grammar] endpoint %d/%d %s succeeded (%d chars)",
				i+1, len(s.endpoints), ep.Name, len(result))
			return result, ep.Name, nil
		}
		lastErr = err
		logutil.Warnf("[Grammar] endpoint %d/%d %s failed: %v — trying next",
			i+1, len(s.endpoints), ep.Name, err)
	}
	return "", "", fmt.Errorf("all %d endpoints exhausted: %w", len(s.endpoints), lastErr)
}

// call sends a single chat completion request to this endpoint.
func (ep *GrammarEndpoint) call(prompt, nativeLangName string) (string, error) {
	start := time.Now()
	defer func() {
		logutil.Duration("GrammarEndpoint", start, ep.Name)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), ep.Timeout)
	defer cancel()

	systemMsg := fmt.Sprintf(`You are a friendly language tutor teaching a student who speaks %s. Follow the user's instructions for the response format.`, nativeLangName)

	// Determine API path and request format based on provider type
	var apiPath string
	var body []byte
	var err error

	switch ep.ProviderType {
	case "ollama":
		apiPath = "/api/chat"
		ollamaReq := ollamaChatRequest{
			Model: ep.Model,
			Messages: []grammarChatMessage{
				{Role: "system", Content: systemMsg},
				{Role: "user", Content: prompt},
			},
			Stream: false,
			Options: map[string]any{
				"temperature": 0.3,
				"num_predict": 4096,
			},
		}
		body, err = json.Marshal(ollamaReq)
	default:
		apiPath = "/chat/completions"
		chatReq := grammarChatRequest{
			Model: ep.Model,
			Messages: []grammarChatMessage{
				{Role: "system", Content: systemMsg},
				{Role: "user", Content: prompt},
			},
			Temperature: 0.3,
			MaxTokens:   4096,
		}
		body, err = json.Marshal(chatReq)
	}
	if err != nil {
		return "", fmt.Errorf("grammar marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", ep.APIURL+apiPath, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("grammar create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if ep.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+ep.APIKey)
	}

	resp, err := ep.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("grammar API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("grammar API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	if ep.ProviderType == "ollama" {
		// Ollama /api/chat returns {"model":"...","message":{"role":"...","content":"..."},"done":true}
		var ollamaResp struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Done bool `json:"done"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
			return "", fmt.Errorf("grammar decode ollama response: %w", err)
		}
		return strings.TrimSpace(ollamaResp.Message.Content), nil
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
