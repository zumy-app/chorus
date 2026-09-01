package services

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/lib/pq"
)

type TeacherService struct {
	db *sql.DB
}

func NewTeacherService(db *sql.DB) *TeacherService {
	return &TeacherService{db: db}
}

func (s *TeacherService) DB() *sql.DB { return s.db }

func (s *TeacherService) GetByUserID(userID string) (*models.TeacherApplication, error) {
	var app models.TeacherApplication
	err := s.db.QueryRow(`SELECT id, user_id, bio, languages, COALESCE(expertise,'') , rate_cents, COALESCE(video_url,''), status, created_at, updated_at FROM teacher_applications WHERE user_id=$1`, userID).Scan(&app.ID, &app.UserID, &app.Bio, pq.Array(&app.Languages), &app.Expertise, &app.RateCents, &app.VideoURL, &app.Status, &app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		return nil, err
	}
	certs, _ := s.listCerts(app.ID)
	app.Certificates = certs
	return &app, nil
}

func (s *TeacherService) listCerts(appID string) ([]models.TeacherCertificate, error) {
	rows, err := s.db.Query(`SELECT id, type, issuer, year, file_url, verified FROM teacher_certificates WHERE application_id=$1 ORDER BY created_at`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.TeacherCertificate
	for rows.Next() {
		var c models.TeacherCertificate
		if err := rows.Scan(&c.ID, &c.Type, &c.Issuer, &c.Year, &c.FileURL, &c.Verified); err != nil {
			continue
		}
		out = append(out, c)
	}
	if out == nil {
		out = []models.TeacherCertificate{}
	}
	return out, rows.Err()
}

func (s *TeacherService) Apply(userID string, req models.TeacherApplyRequest) (*models.TeacherApplication, error) {
	req.Bio = strings.TrimSpace(req.Bio)
	req.Expertise = strings.TrimSpace(req.Expertise)
	req.VideoURL = strings.TrimSpace(req.VideoURL)
	if req.Bio == "" || len(req.Languages) == 0 || req.RateCents <= 0 || req.VideoURL == "" {
		return nil, fmt.Errorf("missing required fields")
	}
	if len(req.Bio) > 1000 {
		return nil, fmt.Errorf("bio too long")
	}
	for i, l := range req.Languages {
		req.Languages[i] = strings.ToLower(strings.TrimSpace(l))
	}
	validCertTypes := map[string]bool{CertTeachingDegree: true, CertLanguageCertificate: true, CertOther: true}
	for _, c := range req.Certificates {
		if !validCertTypes[c.Type] {
			return nil, fmt.Errorf("invalid cert type %q", c.Type)
		}
		if strings.TrimSpace(c.Issuer) == "" || c.Year < 1900 || c.Year > 2030 || strings.TrimSpace(c.FileURL) == "" {
			return nil, fmt.Errorf("invalid certificate fields")
		}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var appID string
	err = tx.QueryRow(`INSERT INTO teacher_applications (user_id, bio, languages, expertise, rate_cents, video_url, status)
		VALUES ($1,$2,$3,$4,$5,$6,'pending')
		ON CONFLICT (user_id) DO UPDATE SET bio=EXCLUDED.bio, languages=EXCLUDED.languages, expertise=EXCLUDED.expertise, rate_cents=EXCLUDED.rate_cents, video_url=EXCLUDED.video_url, updated_at=CURRENT_TIMESTAMP
		RETURNING id`, userID, req.Bio, pq.Array(req.Languages), req.Expertise, req.RateCents, req.VideoURL).Scan(&appID)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`DELETE FROM teacher_certificates WHERE application_id=$1`, appID); err != nil {
		return nil, err
	}
	for _, c := range req.Certificates {
		if _, err := tx.Exec(`INSERT INTO teacher_certificates (application_id, type, issuer, year, file_url) VALUES ($1,$2,$3,$4,$5)`, appID, c.Type, strings.TrimSpace(c.Issuer), c.Year, strings.TrimSpace(c.FileURL)); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetByUserID(userID)
}

func (s *TeacherService) GetTutorProfile(userID string) (*models.TutorProfile, error) {
	q := `SELECT ta.id, ta.user_id, ta.bio, ta.languages, COALESCE(ta.expertise,''),
		ta.rate_cents, COALESCE(ta.video_url,''), ta.status, ta.created_at, ta.updated_at,
		u.display_name, COALESCE(u.avatar_url,''), 
		COALESCE((SELECT AVG(rating)::float FROM tutor_reviews WHERE teacher_user_id=ta.user_id),0),
		COALESCE((SELECT COUNT(*) FROM tutor_reviews WHERE teacher_user_id=ta.user_id),0),
		COALESCE((SELECT EXISTS(SELECT 1 FROM teacher_certificates WHERE application_id=ta.id AND verified=true)), false)
		FROM teacher_applications ta JOIN users u ON u.id=ta.user_id
		WHERE ta.user_id=$1`
	var p models.TutorProfile
	var avatarURL sql.NullString
	var langs []string
	err := s.db.QueryRow(q, userID).Scan(&p.ID, &p.UserID, &p.Bio, pq.Array(&langs), &p.Expertise, &p.RateCents, &p.VideoURL, &p.Status, &p.CreatedAt, &p.UpdatedAt, &p.DisplayName, &avatarURL, &p.RatingAvg, &p.RatingCount, &p.Verified)
	if err != nil {
		return nil, err
	}
	p.Languages = langs
	if avatarURL.Valid && avatarURL.String != "" {
		v := avatarURL.String
		p.AvatarURL = &v
	}
	certs, _ := s.listCerts(p.ID)
	p.Certificates = certs
	p.AvatarColor = avatarColorFor(p.UserID)
	return &p, nil
}

func (s *TeacherService) BrowseTutors(f models.TutorBrowseFilter) ([]models.TutorProfile, int, error) {
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Limit > 50 {
		f.Limit = 50
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	conds := []string{"ta.status='approved'"}
	args := []interface{}{}
	idx := 1
	if f.Language != "" {
		conds = append(conds, fmt.Sprintf("$%d = ANY(ta.languages)", idx))
		args = append(args, strings.ToLower(f.Language))
		idx++
	}
	if f.Search != "" {
		conds = append(conds, fmt.Sprintf("(u.display_name ILIKE $%d OR ta.bio ILIKE $%d OR COALESCE(ta.expertise,'') ILIKE $%d)", idx, idx, idx))
		args = append(args, "%"+f.Search+"%")
		idx++
	}
	if f.VerifiedOnly {
		conds = append(conds, `EXISTS(SELECT 1 FROM teacher_certificates tc WHERE tc.application_id=ta.id AND tc.verified=true)`)
	}
	if f.MaxRateCents != nil {
		conds = append(conds, fmt.Sprintf("ta.rate_cents <= $%d", idx))
		args = append(args, *f.MaxRateCents)
		idx++
	}
	if f.MinRateCents != nil {
		conds = append(conds, fmt.Sprintf("ta.rate_cents >= $%d", idx))
		args = append(args, *f.MinRateCents)
		idx++
	}
	if f.MinRating > 0 {
		conds = append(conds, fmt.Sprintf("COALESCE((SELECT AVG(rating) FROM tutor_reviews WHERE teacher_user_id=ta.user_id),0) >= $%d", idx))
		args = append(args, f.MinRating)
		idx++
	}
	where := strings.Join(conds, " AND ")
	countQ := `SELECT COUNT(*) FROM teacher_applications ta JOIN users u ON u.id=ta.user_id WHERE ` + where
	var total int
	if err := s.db.QueryRow(countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	order := "ORDER BY ta.created_at DESC"
	switch f.Sort {
	case "rating":
		order = "ORDER BY COALESCE((SELECT AVG(rating) FROM tutor_reviews WHERE teacher_user_id=ta.user_id),0) DESC, ta.created_at DESC"
	case "price_asc":
		order = "ORDER BY ta.rate_cents ASC"
	case "price_desc":
		order = "ORDER BY ta.rate_cents DESC"
	case "newest":
		order = "ORDER BY ta.created_at DESC"
	}
	q := fmt.Sprintf(`SELECT ta.id, ta.user_id, ta.bio, ta.languages, COALESCE(ta.expertise,''),
		ta.rate_cents, COALESCE(ta.video_url,''), ta.status, ta.created_at, ta.updated_at,
		u.display_name, COALESCE(u.avatar_url,''),
		COALESCE((SELECT AVG(rating)::float FROM tutor_reviews WHERE teacher_user_id=ta.user_id),0),
		COALESCE((SELECT COUNT(*) FROM tutor_reviews WHERE teacher_user_id=ta.user_id),0),
		COALESCE((SELECT EXISTS(SELECT 1 FROM teacher_certificates WHERE application_id=ta.id AND verified=true)), false)
		FROM teacher_applications ta JOIN users u ON u.id=ta.user_id
		WHERE %s %s LIMIT $%d OFFSET $%d`, where, order, idx, idx+1)
	args = append(args, f.Limit+1, f.Offset)
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []models.TutorProfile
	for rows.Next() {
		var p models.TutorProfile
		var avatarURL sql.NullString
		var langs []string
		if err := rows.Scan(&p.ID, &p.UserID, &p.Bio, pq.Array(&langs), &p.Expertise, &p.RateCents, &p.VideoURL, &p.Status, &p.CreatedAt, &p.UpdatedAt, &p.DisplayName, &avatarURL, &p.RatingAvg, &p.RatingCount, &p.Verified); err != nil {
			continue
		}
		p.Languages = langs
		if avatarURL.Valid && avatarURL.String != "" {
			v := avatarURL.String
			p.AvatarURL = &v
		}
		p.AvatarColor = avatarColorFor(p.UserID)
		out = append(out, p)
	}
	if out == nil {
		out = []models.TutorProfile{}
	}
	hasMore := false
	if len(out) > f.Limit {
		hasMore = true
		out = out[:f.Limit]
	}
	_ = hasMore
	return out, total, rows.Err()
}

func (s *TeacherService) GetTrialCredit(userID string) (*models.TrialCredit, error) {
	var tc models.TrialCredit
	err := s.db.QueryRow(`SELECT user_id, credits, updated_at, granted_at FROM tutor_trial_credits WHERE user_id=$1`, userID).Scan(&tc.UserID, &tc.Credits, &tc.UpdatedAt, &tc.GrantedAt)
	if err == sql.ErrNoRows {
		_, _ = s.db.Exec(`INSERT INTO tutor_trial_credits (user_id, credits) VALUES ($1,1) ON CONFLICT (user_id) DO NOTHING`, userID)
		err = s.db.QueryRow(`SELECT user_id, credits, updated_at, granted_at FROM tutor_trial_credits WHERE user_id=$1`, userID).Scan(&tc.UserID, &tc.Credits, &tc.UpdatedAt, &tc.GrantedAt)
	}
	if err != nil {
		return nil, err
	}
	if tc.Credits == 0 && time.Since(tc.GrantedAt) >= 30*24*time.Hour {
		_, _ = s.db.Exec(`UPDATE tutor_trial_credits SET credits=1, granted_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE user_id=$1 AND credits=0 AND granted_at <= NOW() - INTERVAL '30 days'`, userID)
		_ = s.db.QueryRow(`SELECT user_id, credits, updated_at, granted_at FROM tutor_trial_credits WHERE user_id=$1`, userID).Scan(&tc.UserID, &tc.Credits, &tc.UpdatedAt, &tc.GrantedAt)
	}
	return &tc, nil
}

func (s *TeacherService) GetTrialCreditDashboard(userID string) (*models.TrialCreditDashboard, error) {
	tc, err := s.GetTrialCredit(userID)
	if err != nil {
		return nil, err
	}
	var nextGrant *time.Time
	if tc.Credits == 0 {
		t := tc.GrantedAt.Add(30 * 24 * time.Hour)
		nextGrant = &t
	}
	rows, err := s.db.Query(`SELECT id, teacher_user_id, student_user_id, start_time, end_time, status, is_trial, COALESCE(note,''), COALESCE(review_notes,''), confirmed_at, completed_at, created_at, updated_at FROM tutor_bookings WHERE student_user_id=$1 AND is_trial=true ORDER BY created_at DESC LIMIT 50`, userID)
	if err != nil {
		return &models.TrialCreditDashboard{Credits: tc.Credits, GrantedAt: tc.GrantedAt, UpdatedAt: tc.UpdatedAt, NextGrantAt: nextGrant, History: []models.TutorBooking{}}, nil
	}
	defer rows.Close()
	var hist []models.TutorBooking
	for rows.Next() {
		var b models.TutorBooking
		if err := rows.Scan(&b.ID, &b.TeacherUserID, &b.StudentUserID, &b.StartTime, &b.EndTime, &b.Status, &b.IsTrial, &b.Note, &b.ReviewNotes, &b.ConfirmedAt, &b.CompletedAt, &b.CreatedAt, &b.UpdatedAt); err != nil {
			continue
		}
		hist = append(hist, b)
	}
	if hist == nil {
		hist = []models.TutorBooking{}
	}
	used := 0
	for _, h := range hist {
		if h.Status != "cancelled" {
			used++
		}
	}
	return &models.TrialCreditDashboard{Credits: tc.Credits, GrantedAt: tc.GrantedAt, UpdatedAt: tc.UpdatedAt, NextGrantAt: nextGrant, History: hist, TotalUsed: used, TotalTrial: len(hist)}, nil
}

func (s *TeacherService) ListReviews(teacherUserID string, limit, offset int) ([]models.TutorReview, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM tutor_reviews WHERE teacher_user_id=$1`, teacherUserID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Query(`SELECT id, teacher_user_id, student_user_id, rating, COALESCE(comment,''), created_at, COALESCE((SELECT display_name FROM users WHERE id=student_user_id), '') FROM tutor_reviews WHERE teacher_user_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, teacherUserID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []models.TutorReview
	for rows.Next() {
		var r models.TutorReview
		if err := rows.Scan(&r.ID, &r.TeacherUserID, &r.StudentUserID, &r.Rating, &r.Comment, &r.CreatedAt, &r.StudentName); err != nil {
			continue
		}
		out = append(out, r)
	}
	if out == nil {
		out = []models.TutorReview{}
	}
	return out, total, rows.Err()
}

func (s *TeacherService) AddReview(studentUserID, teacherUserID string, rating int, comment string) (*models.TutorReview, error) {
	if rating < 1 || rating > 5 {
		return nil, fmt.Errorf("rating must be 1-5")
	}
	comment = strings.TrimSpace(comment)
	if len(comment) > 1000 {
		return nil, fmt.Errorf("comment too long")
	}
	if studentUserID == teacherUserID {
		return nil, fmt.Errorf("cannot review yourself")
	}
	var exists string
	if err := s.db.QueryRow(`SELECT status FROM teacher_applications WHERE user_id=$1`, teacherUserID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("tutor not found")
	}
	if exists != "approved" {
		return nil, fmt.Errorf("tutor not approved")
	}
	var completed int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM tutor_bookings WHERE teacher_user_id=$1 AND student_user_id=$2 AND status IN ('confirmed','completed')`, teacherUserID, studentUserID).Scan(&completed)
	_ = completed
	var id string
	err := s.db.QueryRow(`INSERT INTO tutor_reviews (teacher_user_id, student_user_id, rating, comment) VALUES ($1,$2,$3,$4) ON CONFLICT (teacher_user_id, student_user_id) DO UPDATE SET rating=EXCLUDED.rating, comment=EXCLUDED.comment, created_at=CURRENT_TIMESTAMP RETURNING id`, teacherUserID, studentUserID, rating, comment).Scan(&id)
	if err != nil {
		return nil, err
	}
	var r models.TutorReview
	err = s.db.QueryRow(`SELECT id, teacher_user_id, student_user_id, rating, COALESCE(comment,''), created_at, COALESCE((SELECT display_name FROM users WHERE id=student_user_id), '') FROM tutor_reviews WHERE id=$1`, id).Scan(&r.ID, &r.TeacherUserID, &r.StudentUserID, &r.Rating, &r.Comment, &r.CreatedAt, &r.StudentName)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *TeacherService) GetAvailability(teacherUserID string) ([]models.TutorAvailability, error) {
	rows, err := s.db.Query(`SELECT id, teacher_user_id, start_time, end_time, created_at FROM tutor_availability WHERE teacher_user_id=$1 AND end_time > NOW() ORDER BY start_time ASC`, teacherUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.TutorAvailability
	for rows.Next() {
		var a models.TutorAvailability
		if err := rows.Scan(&a.ID, &a.TeacherUserID, &a.StartTime, &a.EndTime, &a.CreatedAt); err != nil {
			continue
		}
		out = append(out, a)
	}
	if out == nil {
		out = []models.TutorAvailability{}
	}
	return out, rows.Err()
}

func (s *TeacherService) AddAvailability(teacherUserID string, start, end time.Time) (*models.TutorAvailability, error) {
	if !end.After(start) {
		return nil, fmt.Errorf("end must be after start")
	}
	if start.Before(time.Now().Add(-time.Minute)) {
		return nil, fmt.Errorf("start must be in the future")
	}
	dur := end.Sub(start)
	if dur < 15*time.Minute || dur > 3*time.Hour {
		return nil, fmt.Errorf("slot duration must be 15m-3h")
	}
	var status string
	if err := s.db.QueryRow(`SELECT status FROM teacher_applications WHERE user_id=$1`, teacherUserID).Scan(&status); err != nil {
		return nil, fmt.Errorf("tutor not found")
	}
	if status != "approved" {
		return nil, fmt.Errorf("tutor not approved")
	}
	var overlap int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM tutor_availability WHERE teacher_user_id=$1 AND NOT (end_time <= $2 OR start_time >= $3)`, teacherUserID, start, end).Scan(&overlap); err != nil {
		return nil, err
	}
	if overlap > 0 {
		return nil, fmt.Errorf("availability overlaps existing slot")
	}
	var a models.TutorAvailability
	err := s.db.QueryRow(`INSERT INTO tutor_availability (teacher_user_id, start_time, end_time) VALUES ($1,$2,$3) RETURNING id, teacher_user_id, start_time, end_time, created_at`, teacherUserID, start, end).Scan(&a.ID, &a.TeacherUserID, &a.StartTime, &a.EndTime, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *TeacherService) RemoveAvailability(teacherUserID, availabilityID string) error {
	res, err := s.db.Exec(`DELETE FROM tutor_availability WHERE id=$1 AND teacher_user_id=$2`, availabilityID, teacherUserID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *TeacherService) CreateBooking(studentUserID, teacherUserID string, req models.CreateBookingRequest) (*models.TutorBooking, error) {
	if studentUserID == teacherUserID {
		return nil, fmt.Errorf("cannot book yourself")
	}
	if !req.EndTime.After(req.StartTime) {
		return nil, fmt.Errorf("end must be after start")
	}
	if req.StartTime.Before(time.Now().Add(-time.Minute)) {
		return nil, fmt.Errorf("start must be in the future")
	}
	dur := req.EndTime.Sub(req.StartTime)
	if dur < 15*time.Minute || dur > 3*time.Hour {
		return nil, fmt.Errorf("duration must be 15m-3h")
	}
	var status string
	if err := s.db.QueryRow(`SELECT status FROM teacher_applications WHERE user_id=$1`, teacherUserID).Scan(&status); err != nil {
		return nil, fmt.Errorf("tutor not found")
	}
	if status != "approved" {
		return nil, fmt.Errorf("tutor not approved")
	}
	var overlap int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM tutor_bookings WHERE teacher_user_id=$1 AND status IN ('pending','confirmed') AND NOT (end_time <= $2 OR start_time >= $3)`, teacherUserID, req.StartTime, req.EndTime).Scan(&overlap); err != nil {
		return nil, err
	}
	if overlap > 0 {
		return nil, fmt.Errorf("time slot already booked")
	}
	isTrial := false
	if req.IsTrial != nil {
		isTrial = *req.IsTrial
	}
	if isTrial {
		tc, err := s.GetTrialCredit(studentUserID)
		if err != nil {
			return nil, err
		}
		if tc.Credits <= 0 {
			return nil, fmt.Errorf("no trial credits remaining")
		}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var b models.TutorBooking
	note := strings.TrimSpace(req.Note)
	err = tx.QueryRow(`INSERT INTO tutor_bookings (teacher_user_id, student_user_id, start_time, end_time, status, is_trial, note) VALUES ($1,$2,$3,$4,'pending',$5,$6) RETURNING id, teacher_user_id, student_user_id, start_time, end_time, status, is_trial, COALESCE(note,''), COALESCE(review_notes,''), confirmed_at, completed_at, created_at, updated_at`, teacherUserID, studentUserID, req.StartTime, req.EndTime, isTrial, note).Scan(&b.ID, &b.TeacherUserID, &b.StudentUserID, &b.StartTime, &b.EndTime, &b.Status, &b.IsTrial, &b.Note, &b.ReviewNotes, &b.ConfirmedAt, &b.CompletedAt, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if isTrial {
		if _, err := tx.Exec(`UPDATE tutor_trial_credits SET credits = credits -1, updated_at=CURRENT_TIMESTAMP WHERE user_id=$1 AND credits>0`, studentUserID); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *TeacherService) ListBookings(userID string, role string, limit, offset int) ([]models.TutorBooking, int, error) {
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
	switch role {
	case "teacher":
		where = "teacher_user_id=$1"
	case "student":
		where = "student_user_id=$1"
	default:
		where = "(teacher_user_id=$1 OR student_user_id=$1)"
	}
	var total int
	if err := s.db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM tutor_bookings WHERE %s`, where), userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	q := fmt.Sprintf(`SELECT b.id, b.teacher_user_id, b.student_user_id, b.start_time, b.end_time, b.status, b.is_trial, COALESCE(b.note,''), COALESCE(b.review_notes,''), b.confirmed_at, b.completed_at, b.created_at, b.updated_at, COALESCE((SELECT display_name FROM users WHERE id=b.teacher_user_id),''), COALESCE((SELECT display_name FROM users WHERE id=b.student_user_id),'') FROM tutor_bookings b WHERE %s ORDER BY b.start_time DESC LIMIT $2 OFFSET $3`, where)
	rows, err := s.db.Query(q, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []models.TutorBooking
	for rows.Next() {
		var b models.TutorBooking
		if err := rows.Scan(&b.ID, &b.TeacherUserID, &b.StudentUserID, &b.StartTime, &b.EndTime, &b.Status, &b.IsTrial, &b.Note, &b.ReviewNotes, &b.ConfirmedAt, &b.CompletedAt, &b.CreatedAt, &b.UpdatedAt, &b.TeacherName, &b.StudentName); err != nil {
			continue
		}
		out = append(out, b)
	}
	if out == nil {
		out = []models.TutorBooking{}
	}
	return out, total, rows.Err()
}

func (s *TeacherService) UpdateBookingStatus(callerID, bookingID, newStatus string) (*models.TutorBooking, error) {
	var b models.TutorBooking
	err := s.db.QueryRow(`SELECT id, teacher_user_id, student_user_id, start_time, end_time, status, is_trial, COALESCE(note,''), COALESCE(review_notes,''), confirmed_at, completed_at, created_at, updated_at FROM tutor_bookings WHERE id=$1`, bookingID).Scan(&b.ID, &b.TeacherUserID, &b.StudentUserID, &b.StartTime, &b.EndTime, &b.Status, &b.IsTrial, &b.Note, &b.ReviewNotes, &b.ConfirmedAt, &b.CompletedAt, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if callerID != b.TeacherUserID && callerID != b.StudentUserID {
		return nil, fmt.Errorf("not authorized")
	}
	if newStatus == "confirmed" && callerID != b.TeacherUserID {
		return nil, fmt.Errorf("only teacher can confirm")
	}
	if newStatus == "confirmed" && b.Status != "pending" {
		return nil, fmt.Errorf("only pending bookings can be confirmed")
	}
	if newStatus == "cancelled" && b.Status == "cancelled" {
		return nil, fmt.Errorf("already cancelled")
	}
	if b.Status == "cancelled" || b.Status == "completed" {
		return nil, fmt.Errorf("booking already %s", b.Status)
	}
	var res sql.Result
	if newStatus == "confirmed" {
		res, err = s.db.Exec(`UPDATE tutor_bookings SET status=$1, confirmed_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=$2 AND status='pending'`, newStatus, bookingID)
	} else {
		res, err = s.db.Exec(`UPDATE tutor_bookings SET status=$1, updated_at=CURRENT_TIMESTAMP WHERE id=$2 AND status IN ('pending','confirmed')`, newStatus, bookingID)
	}
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("booking not updatable")
	}
	if newStatus == "cancelled" && b.IsTrial {
		_, _ = s.db.Exec(`UPDATE tutor_trial_credits SET credits = credits + 1, updated_at=CURRENT_TIMESTAMP WHERE user_id=$1`, b.StudentUserID)
	}
	err = s.db.QueryRow(`SELECT id, teacher_user_id, student_user_id, start_time, end_time, status, is_trial, COALESCE(note,''), COALESCE(review_notes,''), confirmed_at, completed_at, created_at, updated_at FROM tutor_bookings WHERE id=$1`, bookingID).Scan(&b.ID, &b.TeacherUserID, &b.StudentUserID, &b.StartTime, &b.EndTime, &b.Status, &b.IsTrial, &b.Note, &b.ReviewNotes, &b.ConfirmedAt, &b.CompletedAt, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *TeacherService) CompleteBooking(callerID, bookingID string) (*models.TutorBooking, error) {
	var b models.TutorBooking
	err := s.db.QueryRow(`SELECT id, teacher_user_id, student_user_id, start_time, end_time, status, is_trial, COALESCE(note,''), COALESCE(review_notes,''), confirmed_at, completed_at, created_at, updated_at FROM tutor_bookings WHERE id=$1`, bookingID).Scan(&b.ID, &b.TeacherUserID, &b.StudentUserID, &b.StartTime, &b.EndTime, &b.Status, &b.IsTrial, &b.Note, &b.ReviewNotes, &b.ConfirmedAt, &b.CompletedAt, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if callerID != b.TeacherUserID && callerID != b.StudentUserID {
		return nil, fmt.Errorf("not authorized")
	}
	if b.Status != "confirmed" {
		return nil, fmt.Errorf("only confirmed bookings can be completed")
	}
	res, err := s.db.Exec(`UPDATE tutor_bookings SET status='completed', completed_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=$1 AND status='confirmed'`, bookingID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("booking not updatable")
	}
	err = s.db.QueryRow(`SELECT id, teacher_user_id, student_user_id, start_time, end_time, status, is_trial, COALESCE(note,''), COALESCE(review_notes,''), confirmed_at, completed_at, created_at, updated_at FROM tutor_bookings WHERE id=$1`, bookingID).Scan(&b.ID, &b.TeacherUserID, &b.StudentUserID, &b.StartTime, &b.EndTime, &b.Status, &b.IsTrial, &b.Note, &b.ReviewNotes, &b.ConfirmedAt, &b.CompletedAt, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *TeacherService) UpdateReviewNotes(callerID, bookingID, notes string) (*models.TutorBooking, error) {
	notes = strings.TrimSpace(notes)
	if notes == "" {
		return nil, fmt.Errorf("notes required")
	}
	if len(notes) > 5000 {
		return nil, fmt.Errorf("notes too long")
	}
	var b models.TutorBooking
	err := s.db.QueryRow(`SELECT id, teacher_user_id, student_user_id, status FROM tutor_bookings WHERE id=$1`, bookingID).Scan(&b.ID, &b.TeacherUserID, &b.StudentUserID, &b.Status)
	if err != nil {
		return nil, err
	}
	if callerID != b.TeacherUserID {
		return nil, fmt.Errorf("only teacher can add review notes")
	}
	if b.Status != "completed" && b.Status != "confirmed" {
		return nil, fmt.Errorf("review notes only for completed or confirmed sessions")
	}
	_, err = s.db.Exec(`UPDATE tutor_bookings SET review_notes=$1, updated_at=CURRENT_TIMESTAMP WHERE id=$2`, notes, bookingID)
	if err != nil {
		return nil, err
	}
	err = s.db.QueryRow(`SELECT id, teacher_user_id, student_user_id, start_time, end_time, status, is_trial, COALESCE(note,''), COALESCE(review_notes,''), confirmed_at, completed_at, created_at, updated_at FROM tutor_bookings WHERE id=$1`, bookingID).Scan(&b.ID, &b.TeacherUserID, &b.StudentUserID, &b.StartTime, &b.EndTime, &b.Status, &b.IsTrial, &b.Note, &b.ReviewNotes, &b.ConfirmedAt, &b.CompletedAt, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *TeacherService) GetBooking(callerID, bookingID string) (*models.TutorBooking, error) {
	var b models.TutorBooking
	err := s.db.QueryRow(`SELECT b.id, b.teacher_user_id, b.student_user_id, b.start_time, b.end_time, b.status, b.is_trial, COALESCE(b.note,''), COALESCE(b.review_notes,''), b.confirmed_at, b.completed_at, b.created_at, b.updated_at, COALESCE((SELECT display_name FROM users WHERE id=b.teacher_user_id),''), COALESCE((SELECT display_name FROM users WHERE id=b.student_user_id),'') FROM tutor_bookings b WHERE b.id=$1`, bookingID).Scan(&b.ID, &b.TeacherUserID, &b.StudentUserID, &b.StartTime, &b.EndTime, &b.Status, &b.IsTrial, &b.Note, &b.ReviewNotes, &b.ConfirmedAt, &b.CompletedAt, &b.CreatedAt, &b.UpdatedAt, &b.TeacherName, &b.StudentName)
	if err != nil {
		return nil, err
	}
	if callerID != b.TeacherUserID && callerID != b.StudentUserID {
		return nil, fmt.Errorf("not authorized")
	}
	return &b, nil
}

func (s *TeacherService) GetDashboard(teacherUserID string) (*models.TeacherDashboard, error) {
	app, err := s.GetByUserID(teacherUserID)
	if err != nil {
		return nil, err
	}
	hasBio := len(strings.TrimSpace(app.Bio)) >= 10
	hasLanguages := len(app.Languages) > 0
	hasExpertise := len(strings.TrimSpace(app.Expertise)) > 0
	hasRate := app.RateCents > 0
	hasVideo := strings.TrimSpace(app.VideoURL) != ""
	hasCert := len(app.Certificates) > 0
	hasVerified := false
	for _, c := range app.Certificates {
		if c.Verified {
			hasVerified = true
			break
		}
	}
	isApproved := app.Status == "approved"
	done := 0
	total := 7
	if hasBio {
		done++
	}
	if hasLanguages {
		done++
	}
	if hasExpertise {
		done++
	}
	if hasRate {
		done++
	}
	if hasVideo {
		done++
	}
	if hasCert {
		done++
	}
	if isApproved {
		done++
	}
	pct := done * 100 / total
	checklist := models.TeacherChecklist{
		HasBio: hasBio, HasLanguages: hasLanguages, HasExpertise: hasExpertise,
		HasRate: hasRate, HasVideo: hasVideo, HasCertificate: hasCert,
		HasVerifiedCert: hasVerified, IsApproved: isApproved,
		Complete: done == total, CompletionPct: pct,
	}
	avail, _ := s.GetAvailability(teacherUserID)
	if avail == nil {
		avail = []models.TutorAvailability{}
	}
	var ratingAvg float64
	var ratingCount int
	_ = s.db.QueryRow(`SELECT COALESCE(AVG(rating)::float,0), COUNT(*) FROM tutor_reviews WHERE teacher_user_id=$1`, teacherUserID).Scan(&ratingAvg, &ratingCount)
	rows, err := s.db.Query(`SELECT status, start_time, end_time, is_trial FROM tutor_bookings WHERE teacher_user_id=$1`, teacherUserID)
	feePct := 15
	var verified bool
	_ = s.db.QueryRow(`SELECT COALESCE(EXISTS(SELECT 1 FROM teacher_certificates tc WHERE tc.application_id=ta.id AND tc.verified=true), false) FROM teacher_applications ta WHERE ta.user_id=$1`, teacherUserID).Scan(&verified)
	if verified {
		feePct = 10
	}
	earnings := models.TeacherEarnings{PlatformFeePct: feePct, RatingAvg: ratingAvg, RatingCount: ratingCount}
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var status string
			var st, et time.Time
			var isTrial bool
			if err := rows.Scan(&status, &st, &et, &isTrial); err != nil {
				continue
			}
			if isTrial {
				continue
			}
			dur := et.Sub(st).Hours()
			if dur <= 0 {
				dur = 1
			}
			gross := int(float64(app.RateCents) * dur)
			net := gross * (100 - feePct) / 100
			earnings.TotalBookings++
			switch status {
			case "completed":
				earnings.CompletedCount++
				earnings.TotalGrossCents += gross
				earnings.TotalNetCents += net
			case "confirmed", "pending":
				earnings.PendingCount++
				earnings.PendingGrossCents += gross
				earnings.PendingNetCents += net
			case "cancelled":
				earnings.CancelledCount++
			}
		}
	}
	_ = rows.Err()
	upcomingRows, err := s.db.Query(`SELECT b.id, b.teacher_user_id, b.student_user_id, b.start_time, b.end_time, b.status, b.is_trial, COALESCE(b.note,''), b.created_at, b.updated_at, COALESCE((SELECT display_name FROM users WHERE id=b.teacher_user_id),''), COALESCE((SELECT display_name FROM users WHERE id=b.student_user_id),'') FROM tutor_bookings b WHERE b.teacher_user_id=$1 AND b.start_time > NOW() AND b.status IN ('pending','confirmed') ORDER BY b.start_time ASC LIMIT 20`, teacherUserID)
	upcoming := []models.TutorBooking{}
	if err == nil {
		defer upcomingRows.Close()
		for upcomingRows.Next() {
			var b models.TutorBooking
			if err := upcomingRows.Scan(&b.ID, &b.TeacherUserID, &b.StudentUserID, &b.StartTime, &b.EndTime, &b.Status, &b.IsTrial, &b.Note, &b.CreatedAt, &b.UpdatedAt, &b.TeacherName, &b.StudentName); err != nil {
				continue
			}
			upcoming = append(upcoming, b)
		}
	}
	if upcoming == nil {
		upcoming = []models.TutorBooking{}
	}
	studentRows, err := s.db.Query(`SELECT b.student_user_id, COALESCE(u.display_name,''), COALESCE(u.avatar_url,''), COUNT(*) as cnt, MAX(b.start_time), SUM(CASE WHEN b.status='completed' THEN 1 ELSE 0 END) FROM tutor_bookings b JOIN users u ON u.id=b.student_user_id WHERE b.teacher_user_id=$1 GROUP BY b.student_user_id, u.display_name, u.avatar_url ORDER BY MAX(b.start_time) DESC LIMIT 100`, teacherUserID)
	students := []models.DashboardStudent{}
	if err == nil {
		defer studentRows.Close()
		for studentRows.Next() {
			var sid, dname string
			var avatar sql.NullString
			var cnt, completed int
			var last sql.NullTime
			if err := studentRows.Scan(&sid, &dname, &avatar, &cnt, &last, &completed); err != nil {
				continue
			}
			ds := models.DashboardStudent{StudentUserID: sid, DisplayName: dname, BookingsCount: cnt, CompletedCount: completed, AvatarColor: avatarColorFor(sid)}
			if avatar.Valid && avatar.String != "" {
				v := avatar.String
				ds.AvatarURL = &v
			}
			if last.Valid {
				t := last.Time
				ds.LastBookingAt = &t
			}
			students = append(students, ds)
		}
	}
	if students == nil {
		students = []models.DashboardStudent{}
	}
	return &models.TeacherDashboard{
		Application: app, Checklist: checklist, Earnings: earnings,
		Students: students, TotalStudents: len(students),
		UpcomingBookings: upcoming, UpcomingAvailability: avail,
	}, nil
}

func avatarColorFor(userID string) string {
	if userID == "" {
		return "#6b38d4"
	}
	h := 0
	for _, c := range userID {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	colors := []string{"#6b38d4", "#2563eb", "#007d55", "#ba1a1a", "#7c4dff", "#009688", "#ff6f00", "#455a64"}
	return colors[h%len(colors)]
}
