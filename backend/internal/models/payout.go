package models

import "time"

type PayoutMethod struct {
	ID            string    `json:"id" db:"id"`
	TeacherUserID string    `json:"teacherUserId" db:"teacher_user_id"`
	Type          string    `json:"type" db:"type"`
	Label         string    `json:"label" db:"label"`
	Details       string    `json:"details" db:"details"`
	IsDefault     bool      `json:"isDefault" db:"is_default"`
	CreatedAt     time.Time `json:"createdAt" db:"created_at"`
}

type CreatePayoutMethodRequest struct {
	Type      string `json:"type" binding:"required,oneof=paypal bank"`
	Label     string `json:"label" binding:"required,min=1,max=100"`
	Details   string `json:"details" binding:"required,min=2,max=255"`
	IsDefault *bool  `json:"isDefault"`
}

type PayoutRecord struct {
	ID            string     `json:"id" db:"id"`
	TeacherUserID string     `json:"teacherUserId" db:"teacher_user_id"`
	AmountCents   int        `json:"amountCents" db:"amount_cents"`
	FeeCents      int        `json:"feeCents" db:"fee_cents"`
	GrossCents    int        `json:"grossCents" db:"gross_cents"`
	MethodID      *string    `json:"methodId,omitempty" db:"method_id"`
	Destination   string     `json:"destination" db:"destination"`
	Status        string     `json:"status" db:"status"`
	Reference     string     `json:"reference" db:"reference"`
	PaypalBatchID *string    `json:"paypalBatchId,omitempty" db:"paypal_batch_id"`
	CreatedAt     time.Time  `json:"createdAt" db:"created_at"`
	CompletedAt   *time.Time `json:"completedAt,omitempty" db:"completed_at"`
}

type CreatePayoutRequest struct {
	AmountCents int     `json:"amountCents" binding:"required,min=1"`
	MethodID    *string `json:"methodId"`
}

type PayoutTransaction struct {
	StudentName string    `json:"studentName"`
	Initials    string    `json:"initials"`
	Minutes     int       `json:"minutes"`
	AmountCents int       `json:"amountCents"`
	GrossCents  int       `json:"grossCents"`
	FeeCents    int       `json:"feeCents"`
	Date        time.Time `json:"date"`
	Status      string    `json:"status"`
}

type PayoutOverview struct {
	AvailableCents     int                 `json:"availableCents"`
	PendingCents       int                 `json:"pendingCents"`
	PendingGrossCents  int                 `json:"pendingGrossCents"`
	TotalGrossCents    int                 `json:"totalGrossCents"`
	TotalNetCents      int                 `json:"totalNetCents"`
	LifetimeGross      int                 `json:"lifetimeGross"`
	LifetimeNet        int                 `json:"lifetimeNet"`
	TotalPaidCents     int                 `json:"totalPaidCents"`
	PlatformFeePct     int                 `json:"platformFeePct"`
	CompletedCount     int                 `json:"completedCount"`
	PendingCount       int                 `json:"pendingCount"`
	CancelledCount     int                 `json:"cancelledCount"`
	TotalBookings      int                 `json:"totalBookings"`
	NextPayoutDate     *time.Time          `json:"nextPayoutDate,omitempty"`
	HoursTaught        float64             `json:"hoursTaught"`
	ActiveStudents     int                 `json:"activeStudents"`
	RecentTransactions []PayoutTransaction `json:"recentTransactions"`
}
