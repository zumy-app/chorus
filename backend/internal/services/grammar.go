package services

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/chorus/messenger/internal/models"
)

type GrammarService struct {
	translationService *TranslationService
	httpClient         *http.Client
}

func NewGrammarService(translationService *TranslationService) *GrammarService {
	return &GrammarService{
		translationService: translationService,
		httpClient:         &http.Client{},
	}
}

// AnalyzeGrammar performs CEFR-level grammar analysis on a message
func (s *GrammarService) AnalyzeGrammar(ctx context.Context, text string, language string) (*models.GrammarAnalysis, error) {
	// Detect grammar patterns
	patterns := s.detectGrammarPatterns(text, language)
	
	// Determine CEFR difficulty level
	difficulty := s.determineCEFRLevel(text, patterns, language)
	
	// Generate explanations for patterns
	explanations := s.generateExplanations(patterns, language)
	
	return &models.GrammarAnalysis{
		Difficulty:   difficulty,
		Patterns:     patterns,
		Explanations: explanations,
	}, nil
}

// detectGrammarPatterns identifies grammatical structures in the text
func (s *GrammarService) detectGrammarPatterns(text string, language string) []string {
	patterns := []string{}
	lowerText := strings.ToLower(text)
	
	switch language {
	case "es": // Spanish
		if strings.Contains(lowerText, "él") || strings.Contains(lowerText, "ella") {
			patterns = append(patterns, "third_person_singular")
		}
		if strings.Contains(lowerText, "que") && strings.Contains(lowerText, "el ") {
			patterns = append(patterns, "relative_clause")
		}
		if strings.Contains(lowerText, "ría") || strings.Contains(lowerText, "rías") {
			patterns = append(patterns, "conditional_tense")
		}
		if strings.Contains(lowerText, "si ") {
			patterns = append(patterns, "conditional_clause")
		}
		
	case "fr": // French
		if strings.Contains(lowerText, "je ") || strings.Contains(lowerText, "tu ") {
			patterns = append(patterns, "present_tense")
		}
		if strings.Contains(lowerText, "avoir") || strings.Contains(lowerText, "être") {
			patterns = append(patterns, "auxiliary_verb")
		}
		if strings.Contains(lowerText, "que ") || strings.Contains(lowerText, "qui ") {
			patterns = append(patterns, "subordinate_clause")
		}
		
	case "de": // German
		if strings.Contains(lowerText, "der ") || strings.Contains(lowerText, "die ") || strings.Contains(lowerText, "das ") {
			patterns = append(patterns, "definite_article")
		}
		if strings.Contains(lowerText, "haben") || strings.Contains(lowerText, "sein") {
			patterns = append(patterns, "auxiliary_verb")
		}
		
	case "ja": // Japanese
		if strings.Contains(text, "です") || strings.Contains(text, "ます") {
			patterns = append(patterns, "polite_form")
		}
		if strings.Contains(text, "た") {
			patterns = append(patterns, "past_tense")
		}
		
	case "zh": // Chinese
		if strings.Contains(text, "的") {
			patterns = append(patterns, "possessive_particle")
		}
		if strings.Contains(text, "了") {
			patterns = append(patterns, "completion_particle")
		}
		
	default: // English and others
		if strings.Contains(lowerText, "if ") {
			patterns = append(patterns, "conditional")
		}
		if strings.Contains(lowerText, "would") || strings.Contains(lowerText, "could") {
			patterns = append(patterns, "modal_verb")
		}
		if strings.Contains(lowerText, "ing ") {
			patterns = append(patterns, "present_participle")
		}
	}
	
	// Common patterns across languages
	if strings.Count(text, ",") > 2 {
		patterns = append(patterns, "complex_sentence")
	}
	
	words := strings.Fields(text)
	if len(words) > 20 {
		patterns = append(patterns, "long_sentence")
	}
	
	return patterns
}

// determineCEFRLevel assigns CEFR level based on complexity
func (s *GrammarService) determineCEFRLevel(text string, patterns []string, language string) string {
	words := strings.Fields(text)
	wordCount := len(words)
	patternCount := len(patterns)
	
	// Calculate complexity score
	complexityScore := 0
	
	// Word count factor
	if wordCount > 30 {
		complexityScore += 3
	} else if wordCount > 15 {
		complexityScore += 2
	} else {
		complexityScore += 1
	}
	
	// Pattern complexity
	complexPatterns := []string{"conditional", "subjunctive", "relative_clause", "passive_voice", "complex_sentence"}
	for _, pattern := range patterns {
		for _, complex := range complexPatterns {
			if pattern == complex {
				complexityScore += 2
			}
		}
	}
	
	// More patterns = more complex
	complexityScore += patternCount / 2
	
	// Assign CEFR level based on score
	if complexityScore <= 3 {
		return "A1" // Beginner
	} else if complexityScore <= 5 {
		return "A2" // Elementary
	} else if complexityScore <= 7 {
		return "B1" // Intermediate
	} else if complexityScore <= 9 {
		return "B2" // Upper Intermediate
	} else if complexityScore <= 11 {
		return "C1" // Advanced
	}
	return "C2" // Proficiency
}

// generateExplanations creates human-readable explanations for patterns
func (s *GrammarService) generateExplanations(patterns []string, language string) []string {
	explanations := []string{}
	
	explanationMap := map[string]string{
		"present_tense":         "This sentence uses the present tense to describe current actions or states.",
		"past_tense":            "This sentence uses the past tense to describe completed actions.",
		"future_tense":          "This sentence uses the future tense to describe upcoming actions.",
		"conditional":           "This sentence contains a conditional clause (if/then structure).",
		"modal_verb":            "This sentence uses modal verbs (would, could, should) to express possibility or obligation.",
		"present_participle":    "This sentence uses the -ing form to describe ongoing actions.",
		"complex_sentence":      "This is a complex sentence with multiple clauses.",
		"relative_clause":       "This sentence contains a relative clause that provides additional information.",
		"passive_voice":         "This sentence uses passive voice construction.",
		"subordinate_clause":    "This sentence includes a subordinate clause.",
		"third_person_singular": "This sentence uses third person singular conjugation.",
		"conditional_tense":     "This sentence uses the conditional tense to express hypothetical situations.",
		"conditional_clause":    "This sentence contains a conditional clause structure.",
		"auxiliary_verb":        "This sentence uses auxiliary verbs (have, be, etc.).",
		"definite_article":      "This sentence uses definite articles (the, der, die, das).",
		"polite_form":           "This sentence uses polite/formal language forms.",
		"completion_particle":   "This sentence uses particles to indicate completion.",
		"possessive_particle":   "This sentence uses possessive particles.",
		"long_sentence":         "This is a longer, more complex sentence structure.",
	}
	
	for _, pattern := range patterns {
		if explanation, exists := explanationMap[pattern]; exists {
			explanations = append(explanations, explanation)
		} else {
			explanations = append(explanations, fmt.Sprintf("Pattern: %s", pattern))
		}
	}
	
	return explanations
}

// AnalyzeMessageGrammar analyzes grammar for a specific message
func (s *GrammarService) AnalyzeMessageGrammar(ctx context.Context, messageID string, targetLanguage string, messageText string) (*models.GrammarAnalysis, error) {
	// If the message needs translation first, translate it
	var textToAnalyze string
	if targetLanguage != "" {
		// Assume we need to analyze the translated version
		textToAnalyze = messageText // In real implementation, might need to fetch/translate
	} else {
		textToAnalyze = messageText
	}
	
	return s.AnalyzeGrammar(ctx, textToAnalyze, targetLanguage)
}

// BatchAnalyzeMessages analyzes multiple messages for learning purposes
func (s *GrammarService) BatchAnalyzeMessages(ctx context.Context, messages []string, language string) ([]*models.GrammarAnalysis, error) {
	results := make([]*models.GrammarAnalysis, len(messages))
	
	for i, msg := range messages {
		analysis, err := s.AnalyzeGrammar(ctx, msg, language)
		if err != nil {
			return nil, err
		}
		results[i] = analysis
	}
	
	return results, nil
}

// GetGrammarSuggestions provides learning suggestions based on detected patterns
func (s *GrammarService) GetGrammarSuggestions(ctx context.Context, userLevel string, targetLanguage string) ([]string, error) {
	suggestions := []string{}
	
	// Provide suggestions based on user's current level
	switch userLevel {
	case "A1", "A2":
		suggestions = append(suggestions, 
			"Practice basic sentence structures",
			"Learn common verb conjugations",
			"Focus on present tense usage",
		)
	case "B1", "B2":
		suggestions = append(suggestions,
			"Practice complex sentences with conjunctions",
			"Learn conditional structures",
			"Master irregular verbs",
		)
	case "C1", "C2":
		suggestions = append(suggestions,
			"Practice subjunctive mood",
			"Use advanced idiomatic expressions",
			"Master formal and informal registers",
		)
	}
	
	return suggestions, nil
}

// ExportGrammarReport exports learning analytics
type GrammarReport struct {
	TotalAnalyzed      int                 `json:"totalAnalyzed"`
	LevelDistribution  map[string]int      `json:"levelDistribution"`
	CommonPatterns     map[string]int      `json:"commonPatterns"`
	ProgressTimeline   []ProgressPoint     `json:"progressTimeline"`
	Recommendations    []string            `json:"recommendations"`
}

type ProgressPoint struct {
	Date  string `json:"date"`
	Level string `json:"level"`
	Score int    `json:"score"`
}

func (s *GrammarService) GenerateGrammarReport(ctx context.Context, userID string, language string) (*GrammarReport, error) {
	// This would fetch user's grammar analysis history and generate report
	// For now, return a sample structure
	report := &GrammarReport{
		TotalAnalyzed: 0,
		LevelDistribution: make(map[string]int),
		CommonPatterns: make(map[string]int),
		ProgressTimeline: []ProgressPoint{},
		Recommendations: []string{
			"Continue practicing at your current level",
			"Try reading more complex materials",
		},
	}
	
	return report, nil
}
