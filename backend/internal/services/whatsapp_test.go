package services

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestValidatePhone(t *testing.T) {
	if err := ValidatePhone("+14155551234"); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
	if err := ValidatePhone("14155551234"); err == nil {
		t.Fatal("expected invalid without +")
	}
	if err := ValidatePhone("+123"); err == nil {
		t.Fatal("expected invalid too short")
	}
}

func TestMaskPhone(t *testing.T) {
	if MaskPhone("+14155551234") != "+1********34" {
		t.Fatalf("unexpected mask %s", MaskPhone("+14155551234"))
	}
}

func TestOTP_RequestAndVerify(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	sender := &LogWhatsAppSender{}
	svc := NewOTPService(db, sender)

	phone := "+14155551234"
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM phone_otps WHERE user_id = $1 AND created_at > CURRENT_TIMESTAMP - INTERVAL '1 hour'`)).
		WithArgs("user-1").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO phone_otps (user_id, phone, code_hash, expires_at) VALUES ($1,$2,$3,$4)`)).
		WithArgs("user-1", phone, sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET phone = $1 WHERE id = $2`)).
		WithArgs(phone, "user-1").WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.RequestOTP("user-1", phone); err != nil {
		t.Fatalf("RequestOTP failed: %v", err)
	}

	code := "123456"
	hash := func(c string) string { s := sha256.Sum256([]byte(c)); return hex.EncodeToString(s[:]) }
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, code_hash, attempts, expires_at FROM phone_otps WHERE user_id = $1 AND phone = $2 AND expires_at > CURRENT_TIMESTAMP ORDER BY created_at DESC LIMIT 1`)).
		WithArgs("user-1", phone).WillReturnRows(sqlmock.NewRows([]string{"id", "code_hash", "attempts", "expires_at"}).AddRow("otp-1", hash(code), 0, time.Now().Add(5*time.Minute)))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE phone_otps SET attempts = attempts + 1 WHERE id = $1`)).
		WithArgs("otp-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET phone = $1, phone_verified = true, phone_verified_at = CURRENT_TIMESTAMP WHERE id = $2`)).
		WithArgs(phone, "user-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM phone_otps WHERE user_id = $1 AND phone = $2`)).
		WithArgs("user-1", phone).WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.VerifyOTP("user-1", phone, code); err != nil {
		t.Fatalf("VerifyOTP failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestOTP_RateLimited(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	svc := NewOTPService(db, &LogWhatsAppSender{})
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM phone_otps WHERE user_id = $1 AND created_at > CURRENT_TIMESTAMP - INTERVAL '1 hour'`)).
		WithArgs("user-1").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	if err := svc.RequestOTP("user-1", "+14155551234"); err != ErrRateLimited {
		t.Fatalf("expected rate limited, got %v", err)
	}
}

func TestOTP_WrongCode(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	svc := NewOTPService(db, &LogWhatsAppSender{})
	phone := "+14155551234"
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, code_hash, attempts, expires_at FROM phone_otps WHERE user_id = $1 AND phone = $2 AND expires_at > CURRENT_TIMESTAMP ORDER BY created_at DESC LIMIT 1`)).
		WithArgs("user-1", phone).WillReturnRows(sqlmock.NewRows([]string{"id", "code_hash", "attempts", "expires_at"}).AddRow("otp-1", "badhash", 0, time.Now().Add(5*time.Minute)))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE phone_otps SET attempts = attempts + 1 WHERE id = $1`)).
		WithArgs("otp-1").WillReturnResult(sqlmock.NewResult(0, 1))
	if err := svc.VerifyOTP("user-1", phone, "999999"); err != ErrInvalidOTP {
		t.Fatalf("expected invalid otp, got %v", err)
	}
}

func TestOTP_TwoFactorRequiresVerified(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	svc := NewOTPService(db, &LogWhatsAppSender{})
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT phone_verified FROM users WHERE id = $1`)).
		WithArgs("user-1").WillReturnRows(sqlmock.NewRows([]string{"phone_verified"}).AddRow(false))
	if err := svc.SetTwoFactor("user-1", true); err != ErrPhoneNotVerified {
		t.Fatalf("expected phone not verified, got %v", err)
	}
	var _ = sql.ErrNoRows
}
