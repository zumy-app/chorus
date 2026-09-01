package services

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/chorus/messenger/internal/models"
	"github.com/lib/pq"
)

type TeacherSrsService struct {
	db *sql.DB
}

func NewTeacherSrsService(db *sql.DB) *TeacherSrsService {
	return &TeacherSrsService{db: db}
}

func (s *TeacherSrsService) PushCards(teacherID string, req models.TeacherSrsPushRequest) (*models.TeacherSrsPush, error) {
	studentID := strings.TrimSpace(req.StudentID)
	if studentID == "" {
		return nil, fmt.Errorf("studentId required")
	}
	if teacherID == studentID {
		return nil, fmt.Errorf("cannot push to yourself")
	}
	language := strings.ToLower(strings.TrimSpace(req.Language))
	if language == "" {
		return nil, fmt.Errorf("language required")
	}
	if len(req.Cards) == 0 || len(req.Cards) > 20 {
		return nil, fmt.Errorf("cards must be 1-20")
	}
	for i, c := range req.Cards {
		t := strings.TrimSpace(c.Term)
		if t == "" {
			return nil, fmt.Errorf("cards[%d].term required", i)
		}
		if len(t) > 100 {
			return nil, fmt.Errorf("cards[%d].term too long", i)
		}
		req.Cards[i].Term = t
		req.Cards[i].Translation = strings.TrimSpace(c.Translation)
		req.Cards[i].Definition = strings.TrimSpace(c.Definition)
		req.Cards[i].ContextSentence = strings.TrimSpace(c.ContextSentence)
		if req.Cards[i].CefrLevel != "" {
			req.Cards[i].CefrLevel = strings.ToUpper(req.Cards[i].CefrLevel)
		}
	}
	var exists int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE id=$1 AND deleted_at IS NULL`, studentID).Scan(&exists); err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, fmt.Errorf("student not found")
	}
	var status string
	if err := s.db.QueryRow(`SELECT status FROM teacher_applications WHERE user_id=$1`, teacherID).Scan(&status); err != nil {
		return nil, fmt.Errorf("only approved teachers can push drills")
	}
	if status != "approved" {
		return nil, fmt.Errorf("only approved teachers can push drills")
	}
	var bookingID *string
	if req.BookingID != nil && strings.TrimSpace(*req.BookingID) != "" {
		bid := strings.TrimSpace(*req.BookingID)
		bookingID = &bid
		var bTeacher, bStudent, bStatus string
		if err := s.db.QueryRow(`SELECT teacher_user_id::text, student_user_id::text, status FROM tutor_bookings WHERE id=$1`, bid).Scan(&bTeacher, &bStudent, &bStatus); err != nil {
			return nil, fmt.Errorf("booking not found")
		}
		if bTeacher != teacherID || bStudent != studentID {
			return nil, fmt.Errorf("booking does not match teacher and student")
		}
		if bStatus != "completed" && bStatus != "confirmed" && bStatus != "pending" {
			return nil, fmt.Errorf("booking not in pushable state")
		}
	} else {
		var hasBooking int
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM tutor_bookings WHERE teacher_user_id=$1 AND student_user_id=$2`, teacherID, studentID).Scan(&hasBooking)
		if hasBooking == 0 {
			return nil, fmt.Errorf("no lesson relationship with this student")
		}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	note := strings.TrimSpace(req.Note)
	var pushID string
	if bookingID != nil {
		err = tx.QueryRow(`INSERT INTO teacher_srs_pushes (teacher_user_id, student_user_id, booking_id, language, note, item_count) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id::text`, teacherID, studentID, *bookingID, language, note, len(req.Cards)).Scan(&pushID)
	} else {
		err = tx.QueryRow(`INSERT INTO teacher_srs_pushes (teacher_user_id, student_user_id, language, note, item_count) VALUES ($1,$2,$3,$4,$5) RETURNING id::text`, teacherID, studentID, language, note, len(req.Cards)).Scan(&pushID)
	}
	if err != nil {
		return nil, err
	}
	for _, c := range req.Cards {
		term := c.Term
		translation := c.Translation
		definition := c.Definition
		if definition == "" {
			definition = fmt.Sprintf("Drill from teacher: %s", term)
		}
		normalized := strings.ToLower(strings.TrimSpace(term))
		lemma := normalized
		cefr := sql.NullString{}
		if c.CefrLevel != "" {
			cefr = sql.NullString{String: c.CefrLevel, Valid: true}
		}
		ctxSentence := c.ContextSentence
		if ctxSentence == "" {
			ctxSentence = term
		}
		var vocabID string
		err = tx.QueryRow(`
			INSERT INTO vocabulary (user_id, term, language, translation, definition, lemma, normalized_term, part_of_speech, is_chunk, source_type, route_status, mastery_stage, mastery_state, ease_factor, next_review, context_sentence, cefr_level)
			VALUES ($1,$2,$3,$4,$5,$6,$7,'unknown',false,'teacher_push','current_unit',1,'new',2.50,CURRENT_TIMESTAMP,$8,$9)
			ON CONFLICT (user_id, term, language) DO UPDATE SET translation=EXCLUDED.translation, definition=EXCLUDED.definition, context_sentence=EXCLUDED.context_sentence, cefr_level=COALESCE(EXCLUDED.cefr_level, vocabulary.cefr_level), next_review=LEAST(vocabulary.next_review, CURRENT_TIMESTAMP), last_seen_at=CURRENT_TIMESTAMP
			RETURNING id::text`, studentID, term, language, translation, definition, lemma, normalized, ctxSentence, cefr).Scan(&vocabID)
		if err != nil {
			return nil, fmt.Errorf("vocab upsert %q: %w", term, err)
		}
		if _, err := tx.Exec(`INSERT INTO teacher_srs_push_items (push_id, vocabulary_id, term, translation, definition) VALUES ($1,$2,$3,$4,$5)`, pushID, vocabID, term, translation, definition); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetPush(teacherID, pushID)
}

func (s *TeacherSrsService) GetPush(callerID, pushID string) (*models.TeacherSrsPush, error) {
	var p models.TeacherSrsPush
	var bookingID sql.NullString
	var note sql.NullString
	err := s.db.QueryRow(`SELECT p.id::text, p.teacher_user_id::text, p.student_user_id::text, p.booking_id::text, p.language, COALESCE(p.note,''), p.item_count, p.created_at::text, COALESCE((SELECT display_name FROM users WHERE id=p.teacher_user_id),''), COALESCE((SELECT display_name FROM users WHERE id=p.student_user_id),'') FROM teacher_srs_pushes p WHERE p.id=$1`, pushID).Scan(&p.ID, &p.TeacherUserID, &p.StudentUserID, &bookingID, &p.Language, &note, &p.ItemCount, &p.CreatedAt, &p.TeacherName, &p.StudentName)
	if err != nil {
		return nil, err
	}
	if callerID != p.TeacherUserID && callerID != p.StudentUserID {
		return nil, fmt.Errorf("not authorized")
	}
	if bookingID.Valid {
		v := bookingID.String
		p.BookingID = &v
	}
	p.Note = note.String
	items, _ := s.listItems(pushID)
	p.Items = items
	return &p, nil
}

func (s *TeacherSrsService) listItems(pushID string) ([]models.TeacherSrsPushItem, error) {
	rows, err := s.db.Query(`SELECT id::text, push_id::text, vocabulary_id::text, term, COALESCE(translation,''), COALESCE(definition,''), created_at::text FROM teacher_srs_push_items WHERE push_id=$1 ORDER BY created_at`, pushID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.TeacherSrsPushItem
	for rows.Next() {
		var it models.TeacherSrsPushItem
		if err := rows.Scan(&it.ID, &it.PushID, &it.VocabularyID, &it.Term, &it.Translation, &it.Definition, &it.CreatedAt); err != nil {
			continue
		}
		out = append(out, it)
	}
	if out == nil {
		out = []models.TeacherSrsPushItem{}
	}
	return out, rows.Err()
}

func (s *TeacherSrsService) ListPushes(callerID string, role string, peerID string, limit, offset int) ([]models.TeacherSrsPush, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	var where string
	var args []interface{}
	switch role {
	case "teacher":
		where = "p.teacher_user_id=$1"
		args = append(args, callerID)
		if peerID != "" {
			where += " AND p.student_user_id=$2"
			args = append(args, peerID)
		}
	case "student":
		where = "p.student_user_id=$1"
		args = append(args, callerID)
		if peerID != "" {
			where += " AND p.teacher_user_id=$2"
			args = append(args, peerID)
		}
	default:
		where = "(p.teacher_user_id=$1 OR p.student_user_id=$1)"
		args = append(args, callerID)
	}
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM teacher_srs_pushes p WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	orderArgs := append(append([]interface{}{}, args...), limit, offset)
	placeholderLimit := len(args) + 1
	placeholderOffset := len(args) + 2
	q := fmt.Sprintf(`SELECT p.id::text, p.teacher_user_id::text, p.student_user_id::text, p.booking_id::text, p.language, COALESCE(p.note,''), p.item_count, p.created_at::text, COALESCE((SELECT display_name FROM users WHERE id=p.teacher_user_id),''), COALESCE((SELECT display_name FROM users WHERE id=p.student_user_id),'') FROM teacher_srs_pushes p WHERE %s ORDER BY p.created_at DESC LIMIT $%d OFFSET $%d`, where, placeholderLimit, placeholderOffset)
	rows, err := s.db.Query(q, orderArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []models.TeacherSrsPush
	for rows.Next() {
		var p models.TeacherSrsPush
		var bookingID sql.NullString
		var note sql.NullString
		if err := rows.Scan(&p.ID, &p.TeacherUserID, &p.StudentUserID, &bookingID, &p.Language, &note, &p.ItemCount, &p.CreatedAt, &p.TeacherName, &p.StudentName); err != nil {
			continue
		}
		if bookingID.Valid {
			v := bookingID.String
			p.BookingID = &v
		}
		p.Note = note.String
		out = append(out, p)
	}
	if out == nil {
		out = []models.TeacherSrsPush{}
	}
	return out, total, rows.Err()
}

func (s *TeacherSrsService) ListPushesForSandbox(teacherID, studentID string) ([]models.TeacherSrsPush, error) {
	rows, err := s.db.Query(`SELECT p.id::text, p.teacher_user_id::text, p.student_user_id::text, p.booking_id::text, p.language, COALESCE(p.note,''), p.item_count, p.created_at::text FROM teacher_srs_pushes p WHERE p.teacher_user_id=$1 AND p.student_user_id=$2 ORDER BY p.created_at DESC LIMIT 20`, teacherID, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.TeacherSrsPush
	for rows.Next() {
		var p models.TeacherSrsPush
		var bookingID sql.NullString
		var note sql.NullString
		if err := rows.Scan(&p.ID, &p.TeacherUserID, &p.StudentUserID, &bookingID, &p.Language, &note, &p.ItemCount, &p.CreatedAt); err != nil {
			continue
		}
		if bookingID.Valid {
			v := bookingID.String
			p.BookingID = &v
		}
		p.Note = note.String
		out = append(out, p)
	}
	if out == nil {
		out = []models.TeacherSrsPush{}
	}
	return out, rows.Err()
}

var _ = pq.Array
