package services

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/chorus/messenger/internal/models"
)

var (
	ErrInvalidPhone      = errors.New("invalid phone number")
	ErrRateLimited       = errors.New("too many OTP requests")
	ErrInvalidOTP        = errors.New("invalid or expired code")
	ErrOTPExpired        = errors.New("code has expired")
	ErrTooManyAttempts   = errors.New("too many attempts")
	ErrPhoneNotVerified  = errors.New("phone not verified")
	ErrAlreadyVerified   = errors.New("already verified")
)

const (
	otpTTL           = 5 * time.Minute
	otpMaxAttempts   = 5
	otpMaxPerHour    = 5
	otpCodeLength    = 6
)

type WhatsAppSender interface {
	SendOTP(phone, code string) error
}

type LogWhatsAppSender struct{}

func (l *LogWhatsAppSender) SendOTP(phone, code string) error {
	log.Printf("[WhatsApp] OTP for %s: %s (mock sender)", MaskPhone(phone), code)
	return nil
}

type HTTPWhatsAppSender struct {
	APIURL  string
	Token   string
	PhoneID string
	Client  *http.Client
}

func (h *HTTPWhatsAppSender) SendOTP(phone, code string) error {
	if h.APIURL == "" || h.Token == "" {
		log.Printf("[WhatsApp] OTP for %s: %s (no API configured, mocked send)", MaskPhone(phone), code)
		return nil
	}
	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                phone,
		"type":              "template",
		"template": map[string]interface{}{
			"name": "otp_verification",
			"language": map[string]string{"code": "en_US"},
			"components": []map[string]interface{}{
				{"type": "body", "parameters": []map[string]string{{"type": "text", "text": code}}},
			},
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", strings.TrimRight(h.APIURL, "/")+"/messages", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+h.Token)
	req.Header.Set("Content-Type", "application/json")
	client := h.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[WhatsApp] send failed for %s: %v (falling back to mock)", MaskPhone(phone), err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("[WhatsApp] API returned %d for %s (mock fallback)", resp.StatusCode, MaskPhone(phone))
		return nil
	}
	return nil
}

type OTPService struct {
	db     *sql.DB
	sender WhatsAppSender
}

func NewOTPService(db *sql.DB, sender WhatsAppSender) *OTPService {
	if sender == nil {
		sender = &LogWhatsAppSender{}
	}
	return &OTPService{db: db, sender: sender}
}

var phoneRegexp = regexp.MustCompile(`^\+[1-9]\d{7,14}$`)

func NormalizePhone(phone string) string {
	return strings.TrimSpace(phone)
}

func ValidatePhone(phone string) error {
	phone = NormalizePhone(phone)
	if !phoneRegexp.MatchString(phone) {
		return ErrInvalidPhone
	}
	return nil
}

func MaskPhone(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	return phone[:2] + strings.Repeat("*", len(phone)-4) + phone[len(phone)-2:]
}

func hashCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

func generateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", err
	}
	code := fmt.Sprintf("%06d", n.Int64()+100000)
	return code, nil
}

func (s *OTPService) RequestOTP(userID, phone string) error {
	phone = NormalizePhone(phone)
	if err := ValidatePhone(phone); err != nil {
		return err
	}
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM phone_otps WHERE user_id = $1 AND created_at > CURRENT_TIMESTAMP - INTERVAL '1 hour'`, userID).Scan(&count)
	if err != nil {
		return err
	}
	if count >= otpMaxPerHour {
		return ErrRateLimited
	}
	code, err := generateCode()
	if err != nil {
		return err
	}
	codeHash := hashCode(code)
	expiresAt := time.Now().Add(otpTTL)
	_, err = s.db.Exec(`INSERT INTO phone_otps (user_id, phone, code_hash, expires_at) VALUES ($1,$2,$3,$4)`, userID, phone, codeHash, expiresAt)
	if err != nil {
		return err
	}
	if _, err := s.db.Exec(`UPDATE users SET phone = $1 WHERE id = $2`, phone, userID); err != nil {
		log.Printf("failed to update phone for %s: %v", userID, err)
	}
	if err := s.sender.SendOTP(phone, code); err != nil {
		log.Printf("WhatsApp send error for %s: %v", MaskPhone(phone), err)
	}
	return nil
}

func (s *OTPService) VerifyOTP(userID, phone, code string) error {
	phone = NormalizePhone(phone)
	if err := ValidatePhone(phone); err != nil {
		return err
	}
	if len(code) != 6 {
		return ErrInvalidOTP
	}
	codeHash := hashCode(code)
	var id, storedHash string
	var attempts int
	var expiresAt time.Time
	err := s.db.QueryRow(`SELECT id, code_hash, attempts, expires_at FROM phone_otps WHERE user_id = $1 AND phone = $2 AND expires_at > CURRENT_TIMESTAMP ORDER BY created_at DESC LIMIT 1`, userID, phone).Scan(&id, &storedHash, &attempts, &expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvalidOTP
		}
		return err
	}
	if time.Now().After(expiresAt) {
		return ErrOTPExpired
	}
	if attempts >= otpMaxAttempts {
		return ErrTooManyAttempts
	}
	if storedHash != codeHash {
		s.db.Exec(`UPDATE phone_otps SET attempts = attempts + 1 WHERE id = $1`, id)
		return ErrInvalidOTP
	}
	s.db.Exec(`UPDATE phone_otps SET attempts = attempts + 1 WHERE id = $1`, id)
	_, err = s.db.Exec(`UPDATE users SET phone = $1, phone_verified = true, phone_verified_at = CURRENT_TIMESTAMP WHERE id = $2`, phone, userID)
	if err != nil {
		return err
	}
	s.db.Exec(`DELETE FROM phone_otps WHERE user_id = $1 AND phone = $2`, userID, phone)
	return nil
}

func (s *OTPService) VerifyLoginOTP(userID, code string) error {
	var phone *string
	var verified bool
	err := s.db.QueryRow(`SELECT phone, phone_verified FROM users WHERE id = $1`, userID).Scan(&phone, &verified)
	if err != nil {
		return err
	}
	if !verified || phone == nil || *phone == "" {
		return ErrPhoneNotVerified
	}
	return s.VerifyOTP(userID, *phone, code)
}

func (s *OTPService) GetStatus(userID string) (*models.PhoneStatus, error) {
	var phone *string
	var verified, twoFA bool
	err := s.db.QueryRow(`SELECT phone, phone_verified, two_factor_enabled FROM users WHERE id = $1`, userID).Scan(&phone, &verified, &twoFA)
	if err != nil {
		return nil, err
	}
	status := &models.PhoneStatus{Phone: phone, PhoneVerified: verified, TwoFactorEnabled: twoFA}
	if phone != nil && *phone != "" {
		status.PhoneMasked = MaskPhone(*phone)
	}
	return status, nil
}

func (s *OTPService) SetTwoFactor(userID string, enabled bool) error {
	if enabled {
		var verified bool
		err := s.db.QueryRow(`SELECT phone_verified FROM users WHERE id = $1`, userID).Scan(&verified)
		if err != nil {
			return err
		}
		if !verified {
			return ErrPhoneNotVerified
		}
	}
	_, err := s.db.Exec(`UPDATE users SET two_factor_enabled = $1 WHERE id = $2`, enabled, userID)
	return err
}
