package models

import "time"

type PlacementQuestion struct {
	ID        string   `json:"id"`
	Ref       string   `json:"ref"`
	ItemType  string   `json:"itemType"`
	Module    string   `json:"module,omitempty"`
	CEFRLevel string   `json:"cefrLevel"`
	Prompt    any      `json:"prompt"`
	Choices   []string `json:"choices,omitempty"`
}

type PlacementStartResponse struct {
	AttemptID      string            `json:"attemptId"`
	Status         string            `json:"status"`
	Question       PlacementQuestion `json:"question"`
	TotalQuestions int               `json:"totalQuestions"`
}

type PlacementAnswerRequest struct {
	Answer string `json:"answer" binding:"required"`
}

type PlacementResult struct {
	AttemptID      string   `json:"attemptId"`
	EstimatedCEFR  string   `json:"estimatedCefr"`
	ReadinessScore int      `json:"readinessScore"`
	ActiveUnitID   string   `json:"activeUnitId"`
	UnitsSkipped   []string `json:"skippedUnits,omitempty"`
}

type PlacementItem struct {
	ID              string    `json:"id" db:"id"`
	CourseID        string    `json:"courseId" db:"course_id"`
	CEFRLevel       string    `json:"cefrLevel" db:"cefr_level"`
	Module          string    `json:"module" db:"module"` // receptive_vocab, grammar_production, discourse_reading
	ItemType        string    `json:"itemType" db:"item_type"`
	Prompt          any       `json:"prompt" db:"prompt"`
	Choices         []string  `json:"choices" db:"choices"`
	Correct         string    `json:"correct" db:"correct"`
	AcceptVariants  []string  `json:"acceptVariants" db:"accept_variants"`
	DifficultyValue int       `json:"difficultyValue" db:"difficulty_value"`
	IsActive        bool      `json:"isActive" db:"is_active"`
	CreatedAt       time.Time `json:"createdAt" db:"created_at"`
}
