package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chorus/messenger/internal/models"
)

// TeacherAssignmentService implements plan V3 Phase 5: teachers create
// assignments for students they have a tutoring relationship with, students
// submit work, and submissions flow into the async grading queue so the API
// returns immediately (202 + job_id) while AI feedback is generated in the
// background and pushed over WebSocket.
type TeacherAssignmentService struct {
	db      *sql.DB
	grading *GradingQueueService
}

func NewTeacherAssignmentService(db *sql.DB, grading *GradingQueueService) *TeacherAssignmentService {
	return &TeacherAssignmentService{db: db, grading: grading}
}

// CreateAssignment validates the tutoring relationship (an existing booking,
// any status, between this teacher and student), validates the type, and
// registers the baseline record.
func (s *TeacherAssignmentService) CreateAssignment(ctx context.Context, teacherID string, req models.CreateTeacherAssignmentRequest) (*models.TeacherAssignment, error) {
	if req.NativeLanguage == "" {
		req.NativeLanguage = "en"
	}
	// Booking relationship check: optional booking_id must belong to this
	// teacher+student pair; without one, any prior booking between the pair
	// establishes the relationship.
	if req.BookingID != nil && *req.BookingID != "" {
		var exists bool
		err := s.db.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM tutor_bookings
				WHERE id = $1 AND teacher_user_id = $2 AND student_user_id = $3)`,
			*req.BookingID, teacherID, req.StudentID).Scan(&exists)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("booking %s does not belong to this teacher/student pair", *req.BookingID)
		}
	} else {
		var related bool
		err := s.db.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM tutor_bookings
				WHERE teacher_user_id = $1 AND student_user_id = $2)`,
			teacherID, req.StudentID).Scan(&related)
		if err != nil {
			return nil, err
		}
		if !related {
			return nil, fmt.Errorf("no tutoring relationship with this student")
		}
	}

	contentJSON, _ := json.Marshal(req.Content)
	var a models.TeacherAssignment
	var contentBytes []byte
	var dueDate sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO teacher_assignments (
			teacher_id, student_id, booking_id, type, title, instructions, content,
			due_date, target_language, native_language, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'pending')
		RETURNING id::text, teacher_id::text, student_id::text, COALESCE(booking_id::text,''),
			type, title, instructions, content, due_date, target_language, native_language,
			status, created_at, updated_at`,
		teacherID, req.StudentID, nullStr(derefStr(req.BookingID)), req.Type, req.Title, req.Instructions,
		string(contentJSON), nullTime(req.DueDate), req.TargetLanguage, req.NativeLanguage).
		Scan(&a.ID, &a.TeacherID, &a.StudentID, &a.BookingID, &a.Type, &a.Title, &a.Instructions,
			&contentBytes, &dueDate, &a.TargetLanguage, &a.NativeLanguage, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if len(contentBytes) > 0 {
		var c any
		_ = json.Unmarshal(contentBytes, &c)
		a.Content = c
	}
	if dueDate.Valid {
		t := dueDate.Time
		a.DueDate = &t
	}
	return &a, nil
}

// ListTeacherAssignments returns the teacher's roster of assignments.
func (s *TeacherAssignmentService) ListTeacherAssignments(ctx context.Context, teacherID string) ([]models.TeacherAssignment, error) {
	return s.list(ctx, `
		SELECT id::text, teacher_id::text, student_id::text, COALESCE(booking_id::text,''),
		       type, title, instructions, content, due_date, target_language, native_language,
		       status, created_at, updated_at
		FROM teacher_assignments WHERE teacher_id = $1 ORDER BY created_at DESC`, teacherID)
}

// ListStudentAssignments returns the student's outstanding + recent work.
func (s *TeacherAssignmentService) ListStudentAssignments(ctx context.Context, studentID string) ([]models.TeacherAssignment, error) {
	return s.list(ctx, `
		SELECT id::text, teacher_id::text, student_id::text, COALESCE(booking_id::text,''),
		       type, title, instructions, content, due_date, target_language, native_language,
		       status, created_at, updated_at
		FROM teacher_assignments WHERE student_id = $1 ORDER BY created_at DESC`, studentID)
}

func (s *TeacherAssignmentService) list(ctx context.Context, query string, args ...any) ([]models.TeacherAssignment, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.TeacherAssignment
	for rows.Next() {
		var a models.TeacherAssignment
		var contentBytes []byte
		var dueDate sql.NullTime
		if err := rows.Scan(&a.ID, &a.TeacherID, &a.StudentID, &a.BookingID, &a.Type, &a.Title,
			&a.Instructions, &contentBytes, &dueDate, &a.TargetLanguage, &a.NativeLanguage,
			&a.Status, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		if len(contentBytes) > 0 {
			var c any
			_ = json.Unmarshal(contentBytes, &c)
			a.Content = c
		}
		if dueDate.Valid {
			t := dueDate.Time
			a.DueDate = &t
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetAssignment returns one assignment (teacher or student of the pair only).
func (s *TeacherAssignmentService) GetAssignment(ctx context.Context, userID, assignmentID string) (*models.TeacherAssignment, error) {
	var a models.TeacherAssignment
	var contentBytes []byte
	var dueDate sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id::text, teacher_id::text, student_id::text, COALESCE(booking_id::text,''),
		       type, title, instructions, content, due_date, target_language, native_language,
		       status, created_at, updated_at
		FROM teacher_assignments
		WHERE id = $1 AND (teacher_id = $2 OR student_id = $2)`, assignmentID, userID).
		Scan(&a.ID, &a.TeacherID, &a.StudentID, &a.BookingID, &a.Type, &a.Title, &a.Instructions,
			&contentBytes, &dueDate, &a.TargetLanguage, &a.NativeLanguage, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("assignment not found")
	}
	if err != nil {
		return nil, err
	}
	if len(contentBytes) > 0 {
		var c any
		_ = json.Unmarshal(contentBytes, &c)
		a.Content = c
	}
	if dueDate.Valid {
		t := dueDate.Time
		a.DueDate = &t
	}

	// Attach the latest submission when present.
	var sub models.AssignmentSubmission
	var subContent, aiFB, teacherFB []byte
	var score sql.NullInt64
	var reviewedAt sql.NullTime
	err = s.db.QueryRowContext(ctx, `
		SELECT id::text, assignment_id::text, student_id::text, content,
		       COALESCE(ai_feedback::text,''), COALESCE(teacher_feedback::text,''),
		       score, submitted_at, reviewed_at
		FROM assignment_submissions WHERE assignment_id = $1 AND student_id = $2
		ORDER BY submitted_at DESC LIMIT 1`, assignmentID, a.StudentID).
		Scan(&sub.ID, &sub.AssignmentID, &sub.StudentID, &subContent, &aiFB, &teacherFB, &score, &sub.SubmittedAt, &reviewedAt)
	if err == nil {
		if len(subContent) > 0 {
			var c any
			_ = json.Unmarshal(subContent, &c)
			sub.Content = c
		}
		if len(aiFB) > 0 {
			var c any
			_ = json.Unmarshal(aiFB, &c)
			sub.AIFeedback = c
		}
		if len(teacherFB) > 0 {
			var c any
			_ = json.Unmarshal(teacherFB, &c)
			sub.TeacherFeedback = c
		}
		if score.Valid {
			v := int(score.Int64)
			sub.Score = &v
		}
		if reviewedAt.Valid {
			t := reviewedAt.Time
			sub.ReviewedAt = &t
		}
		a.Submission = &sub
	}
	return &a, nil
}

// SubmitAssignment inserts (or replaces) the student's submission, marks the
// assignment submitted, and registers an un-evaluated entry into the async
// grading queue. Returns the submission plus the grading job id (202 flow).
func (s *TeacherAssignmentService) SubmitAssignment(ctx context.Context, userID, assignmentID string, req models.SubmitAssignmentRequest) (*models.AssignmentSubmission, string, error) {
	a, err := s.GetAssignment(ctx, userID, assignmentID)
	if err != nil {
		return nil, "", err
	}
	if a.StudentID != userID {
		return nil, "", fmt.Errorf("only the assigned student can submit")
	}
	if a.Status == "submitted" || a.Status == "reviewed" {
		return nil, "", fmt.Errorf("assignment already submitted")
	}

	contentJSON, _ := json.Marshal(req.Content)
	var sub models.AssignmentSubmission
	var subContent []byte
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO assignment_submissions (assignment_id, student_id, content)
		VALUES ($1, $2, $3)
		ON CONFLICT (assignment_id, student_id) DO UPDATE SET
			content = EXCLUDED.content, submitted_at = CURRENT_TIMESTAMP
		RETURNING id::text, assignment_id::text, student_id::text, content, submitted_at`,
		assignmentID, userID, string(contentJSON)).
		Scan(&sub.ID, &sub.AssignmentID, &sub.StudentID, &subContent, &sub.SubmittedAt)
	if err != nil {
		return nil, "", err
	}
	if len(subContent) > 0 {
		var c any
		_ = json.Unmarshal(subContent, &c)
		sub.Content = c
	}

	_, _ = s.db.ExecContext(ctx, `
		UPDATE teacher_assignments SET status = 'submitted', updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`, assignmentID)

	// Enqueue the async grading job (immediate ack; AI feedback lands via
	// WebSocket push / polling).
	userText := extractSubmissionText(req.Content)
	payload := models.GradingJobPayload{
		TaskType:       assignmentTypeToTask(a.Type),
		Prompt:         a.Title + ". " + a.Instructions,
		UserText:       userText,
		CEFRLevel:      cefrForAssignment(a),
		TargetLanguage: a.TargetLanguage,
		NativeLanguage: a.NativeLanguage,
	}
	jobType := "writing"
	if payload.TaskType == "production" {
		jobType = "production"
	}
	jobID, err := s.grading.Enqueue(ctx, userID, jobType, payload, sub.ID)
	if err != nil {
		// The submission is durable; grading can be retried. Surface the job
		// failure without failing the submit.
		return &sub, "", nil
	}
	return &sub, jobID, nil
}

// ReviewAssignment records teacher feedback + score and completes the loop.
func (s *TeacherAssignmentService) ReviewAssignment(ctx context.Context, teacherID, assignmentID string, req models.ReviewAssignmentRequest) (*models.AssignmentSubmission, error) {
	a, err := s.GetAssignment(ctx, teacherID, assignmentID)
	if err != nil {
		return nil, err
	}
	if a.TeacherID != teacherID {
		return nil, fmt.Errorf("only the assigning teacher can review")
	}
	feedbackJSON, _ := json.Marshal(req.TeacherFeedback)
	res, err := s.db.ExecContext(ctx, `
		UPDATE assignment_submissions SET teacher_feedback = $1, score = $2, reviewed_at = CURRENT_TIMESTAMP
		WHERE assignment_id = $3`,
		string(feedbackJSON), nullIntPtr(req.Score), assignmentID)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, fmt.Errorf("no submission to review")
	}
	_, _ = s.db.ExecContext(ctx, `
		UPDATE teacher_assignments SET status = 'reviewed', updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`, assignmentID)
	return nil, nil
}

func assignmentTypeToTask(t string) string {
	switch t {
	case "writing":
		return "writing"
	case "scenario":
		return "scenario"
	default:
		return "production"
	}
}

func cefrForAssignment(a *models.TeacherAssignment) string {
	if c, ok := a.Content.(map[string]any); ok {
		if s, ok := c["cefrLevel"].(string); ok && s != "" {
			return s
		}
	}
	return "A2"
}

// extractSubmissionText pulls the learner's free text out of a submission
// content envelope ({text: "..."} or plain string).
func extractSubmissionText(content any) string {
	switch c := content.(type) {
	case string:
		return c
	case map[string]any:
		if t, ok := c["text"].(string); ok {
			return t
		}
		b, _ := json.Marshal(c)
		return string(b)
	default:
		b, _ := json.Marshal(content)
		return string(b)
	}
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func nullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

func nullIntPtr(i *int) any {
	if i == nil {
		return nil
	}
	return *i
}
