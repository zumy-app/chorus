package services

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/chorus/messenger/internal/models"
)

type PayoutService struct {
	db     *sql.DB
	paypal *PayPalClient
}

func NewPayoutService(db *sql.DB, paypal *PayPalClient) *PayoutService {
	return &PayoutService{db: db, paypal: paypal}
}

const payoutPlanPremium = "premium"

func platformFeePct(verified bool) int {
	if verified {
		return 10
	}
	return 15
}

func (s *PayoutService) feePctForTeacher(teacherID string) int {
	var appID string
	var verified bool
	err := s.db.QueryRow(`SELECT ta.id, COALESCE(EXISTS(SELECT 1 FROM teacher_certificates tc WHERE tc.application_id=ta.id AND tc.verified=true), false) FROM teacher_applications ta WHERE ta.user_id=$1`, teacherID).Scan(&appID, &verified)
	if err != nil {
		return 15
	}
	return platformFeePct(verified)
}

func (s *PayoutService) studentFeePct(plan string, graceUntil *time.Time) int {
	if plan == payoutPlanPremium {
		return 10
	}
	if graceUntil != nil && time.Now().Before(*graceUntil) {
		return 10
	}
	return 15
}

func (s *PayoutService) computeEarnings(teacherID string) (gross, net, pendingGross, pendingNet int, completedCount, pendingCount, cancelledCount, totalBookings int, feePct int, err error) {
	feePct = s.feePctForTeacher(teacherID)
	verifiedFee := feePct == 10
	var rateCents int
	err = s.db.QueryRow(`SELECT rate_cents FROM teacher_applications WHERE user_id=$1`, teacherID).Scan(&rateCents)
	if err != nil {
		if err == sql.ErrNoRows {
			err = fmt.Errorf("teacher not found")
		}
		return
	}
	rows, qerr := s.db.Query(`SELECT tb.status, tb.start_time, tb.end_time, tb.is_trial, COALESCE(u.plan,'free'), u.plan_grace_until FROM tutor_bookings tb LEFT JOIN users u ON u.id=tb.student_user_id WHERE tb.teacher_user_id=$1`, teacherID)
	if qerr != nil {
		err = qerr
		return
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var st, et time.Time
		var isTrial bool
		var studentPlan string
		var studentGrace *time.Time
		if serr := rows.Scan(&status, &st, &et, &isTrial, &studentPlan, &studentGrace); serr != nil {
			continue
		}
		if isTrial {
			continue
		}
		dur := et.Sub(st).Hours()
		if dur <= 0 {
			dur = 1
		}
		g := int(float64(rateCents) * dur)
		pct := 15
		if verifiedFee || s.studentFeePct(studentPlan, studentGrace) == 10 {
			pct = 10
		}
		n := g * (100 - pct) / 100
		totalBookings++
		switch status {
		case "completed":
			completedCount++
			gross += g
			net += n
		case "confirmed", "pending":
			pendingCount++
			pendingGross += g
			pendingNet += n
		case "cancelled":
			cancelledCount++
		}
	}
	err = rows.Err()
	return
}

func (s *PayoutService) GetOverview(teacherID string) (*models.PayoutOverview, error) {
	var app models.TeacherApplication
	err := s.db.QueryRow(`SELECT id, status FROM teacher_applications WHERE user_id=$1`, teacherID).Scan(&app.ID, &app.Status)
	if err != nil {
		return nil, err
	}
	gross, net, pendingGross, pendingNet, completedCount, pendingCount, cancelledCount, totalBookings, feePct, err := s.computeEarnings(teacherID)
	if err != nil {
		return nil, err
	}
	var paidCents int
	_ = s.db.QueryRow(`SELECT COALESCE(SUM(amount_cents),0) FROM teacher_payouts WHERE teacher_user_id=$1 AND status IN ('pending','completed','processing')`, teacherID).Scan(&paidCents)
	available := net - paidCents
	if available < 0 {
		available = 0
	}
	var totalPaid int
	_ = s.db.QueryRow(`SELECT COALESCE(SUM(amount_cents),0) FROM teacher_payouts WHERE teacher_user_id=$1 AND status='completed'`, teacherID).Scan(&totalPaid)
	lifetimeGross := gross
	lifetimeNet := net
	var pendingPayouts int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM teacher_payouts WHERE teacher_user_id=$1 AND status='pending'`, teacherID).Scan(&pendingPayouts)
	_ = pendingPayouts
	nextPayout := time.Now().AddDate(0, 0, 15)
	nextPayout = time.Date(nextPayout.Year(), nextPayout.Month(), 15, 0, 0, 0, 0, time.UTC)
	if nextPayout.Before(time.Now()) {
		nextPayout = nextPayout.AddDate(0, 1, 0)
	}
	var hours float64
	var activeStudents int
	_ = s.db.QueryRow(`SELECT COALESCE(SUM(EXTRACT(EPOCH FROM (end_time-start_time))/3600),0) FROM tutor_bookings WHERE teacher_user_id=$1 AND status='completed' AND is_trial=false`, teacherID).Scan(&hours)
	_ = s.db.QueryRow(`SELECT COUNT(DISTINCT student_user_id) FROM tutor_bookings WHERE teacher_user_id=$1 AND status IN ('completed','confirmed','pending')`, teacherID).Scan(&activeStudents)
	txRows, _ := s.db.Query(`SELECT tb.start_time, tb.end_time, tb.status, COALESCE(u.display_name,''), tb.is_trial FROM tutor_bookings tb JOIN users u ON u.id=tb.student_user_id WHERE tb.teacher_user_id=$1 AND tb.status='completed' AND tb.is_trial=false ORDER BY tb.completed_at DESC NULLS LAST, tb.start_time DESC LIMIT 5`, teacherID)
	var recent []models.PayoutTransaction
	if txRows != nil {
		defer txRows.Close()
		for txRows.Next() {
			var st, et time.Time
			var status, studentName string
			var isTrial bool
			if err := txRows.Scan(&st, &et, &status, &studentName, &isTrial); err != nil {
				continue
			}
			dur := et.Sub(st).Hours()
			if dur <= 0 {
				dur = 1
			}
			var rate int
			_ = s.db.QueryRow(`SELECT rate_cents FROM teacher_applications WHERE user_id=$1`, teacherID).Scan(&rate)
			g := int(float64(rate) * dur)
			n := g * (100 - feePct) / 100
			initials := "S"
			if len(studentName) > 0 {
				initials = strings.ToUpper(string([]rune(studentName)[0]))
			}
			recent = append(recent, models.PayoutTransaction{
				StudentName: studentName,
				Initials:    initials,
				Minutes:     int(dur * 60),
				AmountCents: n,
				GrossCents:  g,
				FeeCents:    g - n,
				Date:        st,
				Status:      status,
			})
		}
	}
	if recent == nil {
		recent = []models.PayoutTransaction{}
	}
	ov := &models.PayoutOverview{
		AvailableCents:    available,
		PendingCents:      pendingNet,
		PendingGrossCents: pendingGross,
		TotalGrossCents:   lifetimeGross,
		TotalNetCents:     lifetimeNet,
		LifetimeGross:     lifetimeGross,
		LifetimeNet:       lifetimeNet,
		TotalPaidCents:    totalPaid,
		PlatformFeePct:    feePct,
		CompletedCount:    completedCount,
		PendingCount:      pendingCount,
		CancelledCount:    cancelledCount,
		TotalBookings:     totalBookings,
		NextPayoutDate:    &nextPayout,
		HoursTaught:       hours,
		ActiveStudents:    activeStudents,
		RecentTransactions: recent,
	}
	return ov, nil
}

func (s *PayoutService) ListMethods(teacherID string) ([]models.PayoutMethod, error) {
	rows, err := s.db.Query(`SELECT id, teacher_user_id, type, label, details, is_default, created_at FROM teacher_payout_methods WHERE teacher_user_id=$1 ORDER BY is_default DESC, created_at ASC`, teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.PayoutMethod
	for rows.Next() {
		var m models.PayoutMethod
		if err := rows.Scan(&m.ID, &m.TeacherUserID, &m.Type, &m.Label, &m.Details, &m.IsDefault, &m.CreatedAt); err != nil {
			continue
		}
		out = append(out, m)
	}
	if out == nil {
		out = []models.PayoutMethod{}
	}
	return out, rows.Err()
}

func (s *PayoutService) AddMethod(teacherID string, req models.CreatePayoutMethodRequest) (*models.PayoutMethod, error) {
	req.Label = strings.TrimSpace(req.Label)
	req.Details = strings.TrimSpace(req.Details)
	if req.Label == "" {
		return nil, fmt.Errorf("label required")
	}
	if req.Type != "paypal" && req.Type != "bank" {
		return nil, fmt.Errorf("type must be paypal or bank")
	}
	if req.Details == "" {
		return nil, fmt.Errorf("details required")
	}
	if req.Type == "paypal" && !strings.Contains(req.Details, "@") {
		return nil, fmt.Errorf("paypal email invalid")
	}
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM teacher_payout_methods WHERE teacher_user_id=$1`, teacherID).Scan(&count)
	isDefault := count == 0
	if req.IsDefault != nil && *req.IsDefault {
		isDefault = true
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if isDefault {
		_, _ = tx.Exec(`UPDATE teacher_payout_methods SET is_default=false WHERE teacher_user_id=$1`, teacherID)
	}
	var m models.PayoutMethod
	err = tx.QueryRow(`INSERT INTO teacher_payout_methods (teacher_user_id, type, label, details, is_default) VALUES ($1,$2,$3,$4,$5) RETURNING id, teacher_user_id, type, label, details, is_default, created_at`, teacherID, req.Type, req.Label, req.Details, isDefault).Scan(&m.ID, &m.TeacherUserID, &m.Type, &m.Label, &m.Details, &m.IsDefault, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *PayoutService) RemoveMethod(teacherID, methodID string) error {
	res, err := s.db.Exec(`DELETE FROM teacher_payout_methods WHERE id=$1 AND teacher_user_id=$2`, methodID, teacherID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *PayoutService) SetDefaultMethod(teacherID, methodID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var exists int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM teacher_payout_methods WHERE id=$1 AND teacher_user_id=$2`, methodID, teacherID).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return sql.ErrNoRows
	}
	if _, err := tx.Exec(`UPDATE teacher_payout_methods SET is_default=false WHERE teacher_user_id=$1`, teacherID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE teacher_payout_methods SET is_default=true WHERE id=$1`, methodID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *PayoutService) ListPayouts(teacherID string, limit, offset int) ([]models.PayoutRecord, int, error) {
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
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM teacher_payouts WHERE teacher_user_id=$1`, teacherID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Query(`SELECT id, teacher_user_id, amount_cents, fee_cents, gross_cents, COALESCE(method_id::text,''), COALESCE(destination,''), status, reference, COALESCE(paypal_batch_id,''), created_at, completed_at FROM teacher_payouts WHERE teacher_user_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, teacherID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []models.PayoutRecord
	for rows.Next() {
		var p models.PayoutRecord
		var methodID, destination, batchID string
		if err := rows.Scan(&p.ID, &p.TeacherUserID, &p.AmountCents, &p.FeeCents, &p.GrossCents, &methodID, &destination, &p.Status, &p.Reference, &batchID, &p.CreatedAt, &p.CompletedAt); err != nil {
			continue
		}
		if methodID != "" {
			p.MethodID = &methodID
		}
		p.Destination = destination
		if batchID != "" {
			p.PaypalBatchID = &batchID
		}
		out = append(out, p)
	}
	if out == nil {
		out = []models.PayoutRecord{}
	}
	return out, total, rows.Err()
}

func (s *PayoutService) RequestPayout(teacherID string, req models.CreatePayoutRequest) (*models.PayoutRecord, error) {
	if req.AmountCents <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	if req.AmountCents < 100 {
		return nil, fmt.Errorf("minimum payout is $1.00")
	}
	ov, err := s.GetOverview(teacherID)
	if err != nil {
		return nil, err
	}
	if req.AmountCents > ov.AvailableCents {
		return nil, fmt.Errorf("insufficient available balance: %d available", ov.AvailableCents)
	}
	var method *models.PayoutMethod
	if req.MethodID != nil && *req.MethodID != "" {
		var m models.PayoutMethod
		err := s.db.QueryRow(`SELECT id, teacher_user_id, type, label, details, is_default, created_at FROM teacher_payout_methods WHERE id=$1 AND teacher_user_id=$2`, *req.MethodID, teacherID).Scan(&m.ID, &m.TeacherUserID, &m.Type, &m.Label, &m.Details, &m.IsDefault, &m.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("payout method not found")
		}
		method = &m
	} else {
		err := s.db.QueryRow(`SELECT id, teacher_user_id, type, label, details, is_default, created_at FROM teacher_payout_methods WHERE teacher_user_id=$1 AND is_default=true LIMIT 1`, teacherID).Scan(&struct{ID string}{})
		// fallback: pick first method if no default flagged (legacy rows)
		var m models.PayoutMethod
		qerr := s.db.QueryRow(`SELECT id, teacher_user_id, type, label, details, is_default, created_at FROM teacher_payout_methods WHERE teacher_user_id=$1 ORDER BY is_default DESC, created_at ASC LIMIT 1`, teacherID).Scan(&m.ID, &m.TeacherUserID, &m.Type, &m.Label, &m.Details, &m.IsDefault, &m.CreatedAt)
		if qerr != nil {
			return nil, fmt.Errorf("no payout method configured; add one first")
		}
		_ = err
		method = &m
	}
	gross := req.AmountCents * 100 / (100 - ov.PlatformFeePct)
	if gross < req.AmountCents {
		gross = req.AmountCents
	}
	fee := gross - req.AmountCents
	destination := method.Label + " " + method.Details
	if method.Type == "bank" && len(method.Details) >= 4 {
		destination = method.Label + " **** " + method.Details[len(method.Details)-4:]
	}
	ref := fmt.Sprintf("TRX-%X", time.Now().UnixNano()&0xFFFFFF)
	status := "pending"
	var batchID *string
	if method.Type == "paypal" && s.paypal != nil && s.paypal.Enabled() {
		batch, err := s.paypal.CreatePayout(s.db, method.Details, req.AmountCents)
		if err == nil && batch != "" {
			b := batch
			batchID = &b
			status = "processing"
		}
	}
	// Mock immediate completion when paypal not configured: keep pending then mark completed for demo
	if s.paypal == nil || !s.paypal.Enabled() {
		if method.Type == "paypal" {
			status = "pending"
		}
	}
	var p models.PayoutRecord
	var methodIDVal interface{} = nil
	if method != nil {
		methodIDVal = method.ID
	}
	err = s.db.QueryRow(`INSERT INTO teacher_payouts (teacher_user_id, amount_cents, fee_cents, gross_cents, method_id, destination, status, reference, paypal_batch_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id, teacher_user_id, amount_cents, fee_cents, gross_cents, COALESCE(method_id::text,''), COALESCE(destination,''), status, reference, COALESCE(paypal_batch_id,''), created_at, completed_at`, teacherID, req.AmountCents, fee, gross, methodIDVal, destination, status, ref, batchID).Scan(&p.ID, &p.TeacherUserID, &p.AmountCents, &p.FeeCents, &p.GrossCents, &methodIDVal, &p.Destination, &p.Status, &p.Reference, &batchID, &p.CreatedAt, &p.CompletedAt)
	if err != nil {
		return nil, err
	}
	if method != nil {
		mid := method.ID
		p.MethodID = &mid
	}
	if batchID != nil {
		p.PaypalBatchID = batchID
	}
	return &p, nil
}
