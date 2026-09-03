package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

func newOTPHandlerForTest(t *testing.T) (*OTPHandler, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	otp := services.NewOTPService(db, &services.LogWhatsAppSender{})
	auth := services.NewAuthService(db, "test-jwt-secret-for-qa-min-32-chars!!")
	userSvc := services.NewUserService(db)
	return NewOTPHandler(otp, auth, userSvc), mock, func() { db.Close() }
}

func serveOTP(t *testing.T, handler func(*gin.Context), method, pattern, path, userID string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	router := setupTestRouter()
	router.Handle(method, pattern, func(c *gin.Context) {
		if userID != "" {
			c.Set("userID", userID)
		}
		handler(c)
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, bytes.NewReader(body))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	router.ServeHTTP(w, req)
	return w
}

func TestSecurityQA_RequestOTP_InvalidPhone(t *testing.T) {
	h, _, cleanup := newOTPHandlerForTest(t)
	defer cleanup()
	body, _ := json.Marshal(models.OTPRequest{Phone: "not-a-phone"})
	w := serveOTP(t, h.RequestOTP, "POST", "/otp", "/otp", "u1", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSecurityQA_RequestOTP_RateLimited(t *testing.T) {
	h, mock, cleanup := newOTPHandlerForTest(t)
	defer cleanup()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM phone_otps WHERE user_id = $1 AND created_at > CURRENT_TIMESTAMP - INTERVAL '1 hour'`)).
		WithArgs("u1").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	body, _ := json.Marshal(models.OTPRequest{Phone: "+14155551234"})
	w := serveOTP(t, h.RequestOTP, "POST", "/otp", "/otp", "u1", body)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSecurityQA_VerifyPhone_TooManyAttempts(t *testing.T) {
	h, mock, cleanup := newOTPHandlerForTest(t)
	defer cleanup()
	phone := "+14155551234"
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, code_hash, attempts, expires_at FROM phone_otps WHERE user_id = $1 AND phone = $2 AND expires_at > CURRENT_TIMESTAMP ORDER BY created_at DESC LIMIT 1`)).
		WithArgs("u1", phone).WillReturnRows(sqlmock.NewRows([]string{"id", "code_hash", "attempts", "expires_at"}).AddRow("otp-1", "hash", 5, time.Now().Add(5*time.Minute)))
	body, _ := json.Marshal(models.OTPVerifyRequest{Phone: phone, Code: "123456"})
	w := serveOTP(t, h.VerifyPhone, "POST", "/verify", "/verify", "u1", body)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSecurityQA_VerifyPhone_InvalidCode(t *testing.T) {
	h, mock, cleanup := newOTPHandlerForTest(t)
	defer cleanup()
	phone := "+14155551234"
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, code_hash, attempts, expires_at FROM phone_otps WHERE user_id = $1 AND phone = $2 AND expires_at > CURRENT_TIMESTAMP ORDER BY created_at DESC LIMIT 1`)).
		WithArgs("u1", phone).WillReturnRows(sqlmock.NewRows([]string{"id", "code_hash", "attempts", "expires_at"}).AddRow("otp-1", "badhash", 0, time.Now().Add(5*time.Minute)))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE phone_otps SET attempts = attempts + 1 WHERE id = $1`)).
		WithArgs("otp-1").WillReturnResult(sqlmock.NewResult(0, 1))
	body, _ := json.Marshal(models.OTPVerifyRequest{Phone: phone, Code: "999999"})
	w := serveOTP(t, h.VerifyPhone, "POST", "/verify", "/verify", "u1", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSecurityQA_SetTwoFactor_RequiresVerified(t *testing.T) {
	h, mock, cleanup := newOTPHandlerForTest(t)
	defer cleanup()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT phone_verified FROM users WHERE id = $1`)).
		WithArgs("u1").WillReturnRows(sqlmock.NewRows([]string{"phone_verified"}).AddRow(false))
	enabled := true
	body, _ := json.Marshal(models.TwoFASetupRequest{Enabled: &enabled})
	w := serveOTP(t, h.SetTwoFactor, "PUT", "/2fa", "/2fa", "u1", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSecurityQA_SetTwoFactor_MissingField(t *testing.T) {
	h, _, cleanup := newOTPHandlerForTest(t)
	defer cleanup()
	w := serveOTP(t, h.SetTwoFactor, "PUT", "/2fa", "/2fa", "u1", []byte(`{}`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSecurityQA_Verify2FA_InvalidTempToken(t *testing.T) {
	h, _, cleanup := newOTPHandlerForTest(t)
	defer cleanup()
	body, _ := json.Marshal(models.TwoFAVerifyRequest{TempToken: "invalid.token.here", Code: "123456"})
	w := serveOTP(t, h.Verify2FA, "POST", "/2fa/verify", "/2fa/verify", "", body)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSecurityQA_Verify2FA_AccessTokenRejected(t *testing.T) {
	h, _, cleanup := newOTPHandlerForTest(t)
	defer cleanup()
	db, _, _ := sqlmock.New()
	defer db.Close()
	authSvc := services.NewAuthService(db, "test-jwt-secret-for-qa-min-32-chars!!")
	accessToken, _ := authSvc.GenerateAccessToken("u1")
	body, _ := json.Marshal(models.TwoFAVerifyRequest{TempToken: accessToken, Code: "123456"})
	w := serveOTP(t, h.Verify2FA, "POST", "/2fa/verify", "/2fa/verify", "", body)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for access token misuse, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSecurityQA_Verify2FA_WrongCode(t *testing.T) {
	h, mock, cleanup := newOTPHandlerForTest(t)
	defer cleanup()
	authSvc := services.NewAuthService(nil, "test-jwt-secret-for-qa-min-32-chars!!")
	_ = authSvc
	h2 := h
	db2, mock2, _ := sqlmock.New()
	defer db2.Close()
	otp2 := services.NewOTPService(db2, &services.LogWhatsAppSender{})
	auth2 := services.NewAuthService(db2, "test-jwt-secret-for-qa-min-32-chars!!")
	userSvc2 := services.NewUserService(db2)
	h2 = NewOTPHandler(otp2, auth2, userSvc2)
	tempToken, _ := auth2.Generate2FATempToken("u1")
	phone := "+14155551234"
	mock2.ExpectQuery(regexp.QuoteMeta(`SELECT phone, phone_verified FROM users WHERE id = $1`)).
		WithArgs("u1").WillReturnRows(sqlmock.NewRows([]string{"phone", "phone_verified"}).AddRow(phone, true))
	mock2.ExpectQuery(regexp.QuoteMeta(`SELECT id, code_hash, attempts, expires_at FROM phone_otps WHERE user_id = $1 AND phone = $2 AND expires_at > CURRENT_TIMESTAMP ORDER BY created_at DESC LIMIT 1`)).
		WithArgs("u1", phone).WillReturnRows(sqlmock.NewRows([]string{"id", "code_hash", "attempts", "expires_at"}).AddRow("otp-1", "badhash", 0, time.Now().Add(5*time.Minute)))
	mock2.ExpectExec(regexp.QuoteMeta(`UPDATE phone_otps SET attempts = attempts + 1 WHERE id = $1`)).
		WithArgs("otp-1").WillReturnResult(sqlmock.NewResult(0, 1))
	body, _ := json.Marshal(models.TwoFAVerifyRequest{TempToken: tempToken, Code: "999999"})
	router := setupTestRouter()
	router.POST("/2fa/verify", h2.Verify2FA)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/2fa/verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
	_ = mock
}

func TestSecurityQA_AuthMiddleware_MissingToken(t *testing.T) {
	router := setupTestRouter()
	db, _, _ := sqlmock.New()
	defer db.Close()
	authSvc := services.NewAuthService(db, "qa-secret-32chars-minimum-length!!")
	userSvc := services.NewUserService(db)
	router.GET("/protected", middleware.AuthMiddleware(authSvc, userSvc), func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSecurityQA_AuthMiddleware_InvalidHeaderFormat(t *testing.T) {
	router := setupTestRouter()
	db, _, _ := sqlmock.New()
	defer db.Close()
	authSvc := services.NewAuthService(db, "qa-secret-32chars-minimum-length!!")
	userSvc := services.NewUserService(db)
	router.GET("/protected", middleware.AuthMiddleware(authSvc, userSvc), func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "NotBearer token123")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSecurityQA_RateLimiter_BlocksBurst(t *testing.T) {
	router := setupTestRouter()
	router.Use(middleware.RateLimiter(2, time.Minute, middleware.IPKey, 0))
	router.GET("/limited", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/limited", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("request %d should pass, got %d", i, w.Code)
		}
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/limited", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	router.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
}

func TestSecurityQA_Block_SelfIsRejected(t *testing.T) {
	h, _, cleanup := newModerationHandlerForTest(t)
	defer cleanup()
	w := serveModeration(t, h.Block, http.MethodPost, "/blocks", "/blocks", "u1",
		mustJSONBody(t, map[string]string{"blockedUserId": "u1"}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 self-block, got %d", w.Code)
	}
}

func TestSecurityQA_Report_SelfIsRejected(t *testing.T) {
	h, mock, cleanup := newModerationHandlerForTest(t)
	defer cleanup()
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM users WHERE id = \$1 AND deleted_at IS NULL\)`).
		WithArgs("u1").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	w := serveModeration(t, h.Report, http.MethodPost, "/reports", "/reports", "u1",
		mustJSONBody(t, models.ReportRequest{Type: "user", ReportedUserID: "u1", Reason: "spam"}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 self-report, got %d: %s", w.Code, w.Body.String())
	}
}
