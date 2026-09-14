package models

import "time"

type CurriculumCourse struct {
	ID             string    `json:"id" db:"id"`
	TargetLanguage string    `json:"targetLanguage" db:"target_language"`
	NativeLanguage string    `json:"nativeLanguage" db:"native_language"`
	Title          string    `json:"title" db:"title"`
	Version        string    `json:"version" db:"version"`
	IsActive       bool      `json:"isActive" db:"is_active"`
	SupportTier    string    `json:"supportTier" db:"support_tier"`
	Metadata       any       `json:"metadata,omitempty" db:"metadata"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
}

type CurriculumUnit struct {
	ID                 string    `json:"id" db:"id"`
	CourseID           string    `json:"courseId" db:"course_id"`
	CEFRLevel          string    `json:"cefrLevel" db:"cefr_level"`
	Ordinal            int       `json:"ordinal" db:"ordinal"`
	Slug               string    `json:"slug" db:"slug"`
	Title              string    `json:"title" db:"title"`
	CanDoStatement     string    `json:"canDoStatement" db:"can_do_statement"`
	Description        string    `json:"description" db:"description"`
	EstimatedMinutes   int       `json:"estimatedMinutes" db:"estimated_minutes"`
	CheckpointRequired bool      `json:"checkpointRequired" db:"checkpoint_required"`
	CreatedAt          time.Time `json:"createdAt" db:"created_at"`
}

type LessonSummary struct {
	ID               string `json:"id"`
	UnitID           string `json:"unitId"`
	Ordinal          int    `json:"ordinal"`
	Slug             string `json:"slug"`
	Type             string `json:"type"`
	Title            string `json:"title"`
	Objective        string `json:"objective"`
	EstimatedMinutes int    `json:"estimatedMinutes"`
	Status           string `json:"status"`
}

type UnitProgressSummary struct {
	ID                 string          `json:"id"`
	CourseID           string          `json:"courseId"`
	CEFRLevel          string          `json:"cefrLevel"`
	Ordinal            int             `json:"ordinal"`
	Slug               string          `json:"slug"`
	Title              string          `json:"title"`
	CanDoStatement     string          `json:"canDoStatement"`
	Description        string          `json:"description"`
	EstimatedMinutes   int             `json:"estimatedMinutes"`
	CheckpointRequired bool            `json:"checkpointRequired"`
	Status             string          `json:"status"`
	ProgressPct        int             `json:"progressPct"`
	CompetencyScore    int             `json:"competencyScore"`
	LessonsCompleted   int             `json:"lessonsCompleted"`
	CheckpointScore    *int            `json:"checkpointScore,omitempty"`
	StartedAt          *time.Time      `json:"startedAt,omitempty"`
	CompletedAt        *time.Time      `json:"completedAt,omitempty"`
	Lessons            []LessonSummary `json:"lessons,omitempty"`
}

type LearningPath struct {
	Capability LearningPairCapability `json:"capability"`
	Profile    UserLanguageProfile    `json:"profile"`
	Units      []UnitProgressSummary  `json:"units"`
}

type CurriculumLesson struct {
	ID               string           `json:"id" db:"id"`
	UnitID           string           `json:"unitId" db:"unit_id"`
	Ordinal          int              `json:"ordinal" db:"ordinal"`
	Slug             string           `json:"slug" db:"slug"`
	Type             string           `json:"type" db:"type"`
	Title            string           `json:"title" db:"title"`
	Objective        string           `json:"objective" db:"objective"`
	EstimatedMinutes int              `json:"estimatedMinutes" db:"estimated_minutes"`
	Source           string           `json:"source,omitempty" db:"source"`
	IsActive         bool             `json:"isActive" db:"is_active"`
	Steps            []CurriculumStep `json:"steps,omitempty"`
}

type CurriculumStep struct {
	ID          string `json:"id" db:"id"`
	LessonID    string `json:"lessonId" db:"lesson_id"`
	Ordinal     int    `json:"ordinal" db:"ordinal"`
	Type        string `json:"type" db:"type"`
	Prompt      any    `json:"prompt" db:"prompt"`
	AnswerKey   any    `json:"answerKey,omitempty" db:"answer_key"`
	ContentRefs any    `json:"contentRefs,omitempty" db:"content_refs"`
}

type LessonAttempt struct {
	ID             string             `json:"id" db:"id"`
	UserID         string             `json:"userId" db:"user_id"`
	LessonID       string             `json:"lessonId" db:"lesson_id"`
	TargetLanguage string             `json:"targetLanguage" db:"target_language"`
	Status         string             `json:"status" db:"status"`
	Score          int                `json:"score" db:"score"`
	CorrectCount   int                `json:"correctCount" db:"correct_count"`
	TotalCount     int                `json:"totalCount" db:"total_count"`
	StartedAt      time.Time          `json:"startedAt" db:"started_at"`
	CompletedAt    time.Time          `json:"completedAt,omitempty" db:"completed_at"`
	Steps          []LessonStepResult `json:"steps,omitempty"`
}

type LessonStepResult struct {
	ID         string `json:"id" db:"id"`
	StepID     string `json:"stepId" db:"step_id"`
	UserAnswer any    `json:"userAnswer" db:"user_answer"`
	Correct    bool   `json:"correct" db:"correct"`
	Score      int    `json:"score" db:"score"`
	Feedback   any    `json:"feedback" db:"feedback"`
}

type AnswerLessonStepRequest struct {
	UserAnswer string `json:"answer" binding:"required"`
}

type LessonStartResponse struct {
	Attempt *LessonAttempt   `json:"attempt"`
	Steps   []CurriculumStep `json:"steps"`
}

type GrammarPoint struct {
	ID               string    `json:"id" db:"id"`
	CourseID         string    `json:"courseId" db:"course_id"`
	UnitID           *string   `json:"unitId,omitempty" db:"unit_id"`
	Slug             string    `json:"slug" db:"slug"`
	CEFRLevel        string    `json:"cefrLevel" db:"cefr_level"`
	Title            string    `json:"title" db:"title"`
	ShortExplanation string    `json:"shortExplanation" db:"short_explanation"`
	RuleText         string    `json:"ruleText" db:"rule_text"`
	CommonTrap       string    `json:"commonTrap" db:"common_trap"`
	Ordinal          int       `json:"ordinal" db:"ordinal"`
	Examples         any       `json:"examples" db:"examples"`
	Prerequisites    []string  `json:"prerequisites" db:"prerequisites"`
	IsActive         bool      `json:"isActive" db:"is_active"`
	Metadata         any       `json:"metadata,omitempty" db:"metadata"`
	CreatedAt        time.Time `json:"createdAt" db:"created_at"`
}

type LexicalItem struct {
	ID           string    `json:"id" db:"id"`
	CourseID     string    `json:"courseId" db:"course_id"`
	UnitID       *string   `json:"unitId,omitempty" db:"unit_id"`
	Language     string    `json:"language" db:"language"`
	Lemma        string    `json:"lemma" db:"lemma"`
	DisplayText  string    `json:"displayText" db:"display_text"`
	PartOfSpeech string    `json:"partOfSpeech" db:"part_of_speech"`
	CEFRLevel    string    `json:"cefrLevel" db:"cefr_level"`
	Translations any       `json:"translations" db:"translations"`
	Forms        any       `json:"forms" db:"forms"`
	Tags         []string  `json:"tags" db:"tags"`
	Frequency    int       `json:"frequency" db:"frequency"`
	IsChunk      bool      `json:"isChunk" db:"is_chunk"`
	Metadata     any       `json:"metadata,omitempty" db:"metadata"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
}

type ReadingPassage struct {
	ID            string    `json:"id" db:"id"`
	CourseID      string    `json:"courseId" db:"course_id"`
	UnitID        *string   `json:"unitId,omitempty" db:"unit_id"`
	CEFRLevel     string    `json:"cefrLevel" db:"cefr_level"`
	Title         string    `json:"title" db:"title"`
	Body          string    `json:"body" db:"body"`
	WordCount     int       `json:"wordCount" db:"word_count"`
	Questions     any       `json:"questions" db:"questions"`
	VocabularyIDs []string  `json:"vocabularyIds" db:"vocabulary_ids"`
	IsActive      bool      `json:"isActive" db:"is_active"`
	CreatedAt     time.Time `json:"createdAt" db:"created_at"`
}
