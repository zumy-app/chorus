package models

import "time"

type TeacherLesson struct {
	ID               string              `json:"id" db:"id"`
	TeacherID        string              `json:"teacherId" db:"teacher_id"`
	TargetLanguage   string              `json:"targetLanguage" db:"target_language"`
	NativeLanguage   string              `json:"nativeLanguage" db:"native_language"`
	Title            string              `json:"title" db:"title"`
	Objective        string              `json:"objective" db:"objective"`
	EstimatedMinutes int                 `json:"estimatedMinutes" db:"estimated_minutes"`
	CEFRLevel        string              `json:"cefrLevel,omitempty" db:"cefr_level"`
	IsPublished      bool                `json:"isPublished" db:"is_published"`
	CreatedAt        time.Time           `json:"createdAt" db:"created_at"`
	UpdatedAt        time.Time           `json:"updatedAt" db:"updated_at"`
	Steps            []TeacherLessonStep `json:"steps,omitempty"`
}

type TeacherLessonStep struct {
	ID        string `json:"id" db:"id"`
	LessonID  string `json:"lessonId" db:"lesson_id"`
	Ordinal   int    `json:"ordinal" db:"ordinal"`
	Type      string `json:"type" db:"type"` // intro, mcq, cloze, free_recall, production, reading_passage, writing, gap_fill
	Prompt    any    `json:"prompt" db:"prompt"`
	AnswerKey any    `json:"answerKey" db:"answer_key"`
}

type TeacherAssignment struct {
	ID             string                  `json:"id" db:"id"`
	TeacherID      string                  `json:"teacherId" db:"teacher_id"`
	StudentID      string                  `json:"studentId" db:"student_id"`
	BookingID      *string                 `json:"bookingId,omitempty" db:"booking_id"`
	Type           string                  `json:"type" db:"type"` // vocabulary_push, reading, writing, scenario, lesson, mixed
	Title          string                  `json:"title" db:"title"`
	Instructions   string                  `json:"instructions" db:"instructions"`
	Content        any                     `json:"content" db:"content"`
	DueDate        *time.Time              `json:"dueDate,omitempty" db:"due_date"`
	TargetLanguage string                  `json:"targetLanguage" db:"target_language"`
	NativeLanguage string                  `json:"nativeLanguage" db:"native_language"`
	Status         string                  `json:"status" db:"status"` // pending, in_progress, submitted, reviewed
	CreatedAt      time.Time               `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time               `json:"updatedAt" db:"updated_at"`
	Submission     *AssignmentSubmission   `json:"submission,omitempty"`
}

type AssignmentSubmission struct {
	ID              string     `json:"id" db:"id"`
	AssignmentID    string     `json:"assignmentId" db:"assignment_id"`
	StudentID       string     `json:"studentId" db:"student_id"`
	Content         any        `json:"content" db:"content"`
	AIFeedback      any        `json:"aiFeedback,omitempty" db:"ai_feedback"`
	TeacherFeedback any        `json:"teacherFeedback,omitempty" db:"teacher_feedback"`
	Score           *int       `json:"score,omitempty" db:"score"`
	SubmittedAt     time.Time  `json:"submittedAt" db:"submitted_at"`
	ReviewedAt      *time.Time `json:"reviewedAt,omitempty" db:"reviewed_at"`
}

type StudentProgressSummary struct {
	StudentID           string              `json:"studentId"`
	StudentName         string              `json:"studentName"`
	TargetLanguage      string              `json:"targetLanguage"`
	VocabByStage        map[int]int         `json:"vocabByStage"`
	TopGrammarErrors    []GrammarSummary    `json:"topGrammarErrors"`
	WeeklyXP            int                 `json:"weeklyXp"`
	ActiveAssignments   []TeacherAssignment `json:"activeAssignments"`
	RecentSubmissions   []AssignmentSubmission `json:"recentSubmissions"`
}

type CreateTeacherAssignmentRequest struct {
	StudentID      string     `json:"studentId" binding:"required"`
	BookingID      *string    `json:"bookingId,omitempty"`
	Type           string     `json:"type" binding:"required,oneof=vocabulary_push reading writing scenario lesson mixed"`
	Title          string     `json:"title" binding:"required"`
	Instructions   string     `json:"instructions"`
	Content        any        `json:"content" binding:"required"`
	DueDate        *time.Time `json:"dueDate,omitempty"`
	TargetLanguage string     `json:"targetLanguage" binding:"required"`
	NativeLanguage string     `json:"nativeLanguage"`
}

type SubmitAssignmentRequest struct {
	Content any `json:"content" binding:"required"`
}

type ReviewAssignmentRequest struct {
	TeacherFeedback any  `json:"teacherFeedback" binding:"required"`
	Score           *int `json:"score,omitempty"`
}
