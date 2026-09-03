package services

import (
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
)

func TestSecurity_PhoneValidation_RejectsInjection(t *testing.T) {
	cases := []string{"+1415'; DROP TABLE users;--", "+1415<script>", "++14155551234", "+0 141555", "+1415 555 1234", "", " ", "+123"}
	for _, c := range cases {
		if ValidatePhone(c) == nil {
			t.Fatalf("expected invalid for %q", c)
		}
	}
	if err := ValidatePhone(" +14155551234 "); err != nil {
		t.Fatalf("trimmed valid should pass: %v", err)
	}
}

func TestSecurity_NormalizePhone_TrimsSpaces(t *testing.T) {
	if NormalizePhone("  +14155551234  ") != "+14155551234" {
		t.Fatal("normalize failed")
	}
}

func TestSecurity_MaskPhone_EdgeCases(t *testing.T) {
	if MaskPhone("") != "****" {
		t.Fatalf("empty mask %s", MaskPhone(""))
	}
	if MaskPhone("+1") != "****" {
		t.Fatalf("short mask %s", MaskPhone("+1"))
	}
	if got := MaskPhone("+14155551234"); got == "+14155551234" {
		t.Fatal("should mask")
	}
}

func TestSecurity_OTP_ExpiredCode(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	svc := NewOTPService(db, &LogWhatsAppSender{})
	phone := "+14155551234"
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, code_hash, attempts, expires_at FROM phone_otps WHERE user_id = $1 AND phone = $2 AND expires_at > CURRENT_TIMESTAMP ORDER BY created_at DESC LIMIT 1`)).
		WithArgs("u1", phone).WillReturnRows(sqlmock.NewRows([]string{"id", "code_hash", "attempts", "expires_at"}).AddRow("otp-1", "hash", 0, time.Now().Add(-1*time.Minute)))
	if err := svc.VerifyOTP("u1", phone, "123456"); err == nil {
		t.Fatal("expected error for expired")
	}
}

func TestSecurity_OTP_TooManyAttempts(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	svc := NewOTPService(db, &LogWhatsAppSender{})
	phone := "+14155551234"
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, code_hash, attempts, expires_at FROM phone_otps WHERE user_id = $1 AND phone = $2 AND expires_at > CURRENT_TIMESTAMP ORDER BY created_at DESC LIMIT 1`)).
		WithArgs("u1", phone).WillReturnRows(sqlmock.NewRows([]string{"id", "code_hash", "attempts", "expires_at"}).AddRow("otp-1", "hash", 5, time.Now().Add(5*time.Minute)))
	if err := svc.VerifyOTP("u1", phone, "123456"); err != ErrTooManyAttempts {
		t.Fatalf("expected too many attempts, got %v", err)
	}
}

func TestSecurity_OTP_NoRows_ReturnsInvalid(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	svc := NewOTPService(db, &LogWhatsAppSender{})
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, code_hash, attempts, expires_at FROM phone_otps WHERE user_id = $1 AND phone = $2 AND expires_at > CURRENT_TIMESTAMP ORDER BY created_at DESC LIMIT 1`)).
		WithArgs("u1", "+14155551234").WillReturnError(sql.ErrNoRows)
	if err := svc.VerifyOTP("u1", "+14155551234", "123456"); err != ErrInvalidOTP {
		t.Fatalf("expected invalid otp, got %v", err)
	}
}

func TestSecurity_VerifyLoginOTP_NotVerified(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	svc := NewOTPService(db, &LogWhatsAppSender{})
	phone := "+14155551234"
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT phone, phone_verified FROM users WHERE id = $1`)).
		WithArgs("u1").WillReturnRows(sqlmock.NewRows([]string{"phone", "phone_verified"}).AddRow(phone, false))
	if err := svc.VerifyLoginOTP("u1", "123456"); err != ErrPhoneNotVerified {
		t.Fatalf("expected phone not verified, got %v", err)
	}
}

func TestSecurity_Privacy_CanView_Matrix(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	svc := NewPrivacyService(db)
	if !svc.CanView("a", "a", models.PrivacyNobody) {
		t.Fatal("self should always view")
	}
	if !svc.CanView("viewer", "owner", models.PrivacyEveryone) {
		t.Fatal("everyone should allow")
	}
	if svc.CanView("viewer", "owner", models.PrivacyNobody) {
		t.Fatal("nobody should deny")
	}
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM chat_participants`)).
		WithArgs("viewer", "owner").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	if svc.CanView("viewer", "owner", models.PrivacyContacts) {
		t.Fatal("non-contact should deny")
	}
}

func TestSecurity_Privacy_FilterUser_Nobody(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	svc := NewPrivacyService(db)
	avatar := "https://cdn/a.jpg"
	u := &models.User{ID: "owner", AvatarURL: &avatar, CreatedAt: time.Now().Add(-24 * time.Hour), LastActiveAt: time.Now()}
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT profile_photo_visibility FROM user_settings WHERE user_id = $1`)).
		WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"profile_photo_visibility"}).AddRow("nobody"))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT last_seen_visibility FROM user_settings WHERE user_id = $1`)).
		WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"last_seen_visibility"}).AddRow("nobody"))
	svc.FilterUser("viewer", u)
	if u.AvatarURL != nil {
		t.Fatal("avatar should be blanked")
	}
	if !u.LastActiveAt.Equal(u.CreatedAt) {
		t.Fatal("last active should be reset to created_at")
	}
}

func TestSecurity_Privacy_FilterUser_Everyone_Passthrough(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	svc := NewPrivacyService(db)
	avatar := "https://cdn/a.jpg"
	u := &models.User{ID: "owner", AvatarURL: &avatar, CreatedAt: time.Now().Add(-24 * time.Hour), LastActiveAt: time.Now()}
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT profile_photo_visibility FROM user_settings WHERE user_id = $1`)).
		WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"profile_photo_visibility"}).AddRow("everyone"))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT last_seen_visibility FROM user_settings WHERE user_id = $1`)).
		WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"last_seen_visibility"}).AddRow("everyone"))
	svc.FilterUser("viewer", u)
	if u.AvatarURL == nil || *u.AvatarURL != avatar {
		t.Fatal("avatar should remain")
	}
}

func TestSecurity_TwoFATempToken_InvalidPurpose(t *testing.T) {
	svc := NewAuthService(nil, "test-secret-for-qa-1234567890")
	token, err := svc.Generate2FATempToken("user-1")
	if err != nil {
		t.Fatalf("gen: %v", err)
	}
	if _, err := svc.Validate2FATempToken(token); err != nil {
		t.Fatalf("valid 2fa token should pass: %v", err)
	}
	access, _ := svc.GenerateAccessToken("user-1")
	if _, err := svc.Validate2FATempToken(access); err == nil {
		t.Fatal("access token should not validate as 2fa")
	}
	if _, err := svc.Validate2FATempToken("garbage.token.here"); err == nil {
		t.Fatal("garbage should fail")
	}
}
