package services

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
)

func TestKeyServiceRegisterDeviceKeysStoresOwnPublicKeyMaterial(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	service := NewKeyService(db)
	req := models.RegisterDeviceKeysRequest{
		DeviceID:              "device-1",
		DeviceName:            "Safari on Mac",
		DeviceType:            "web",
		IdentityPublicKey:     "identity-public",
		SignedPreKey:          "signed-pre-key",
		SignedPreKeySignature: "signed-pre-key-signature",
		OneTimePreKeys:        []string{"one-time-a", "one-time-b"},
	}

	mock.ExpectQuery(`INSERT INTO user_devices`).
		WithArgs("device-1", "user-1", "Safari on Mac", "web", "identity-public", "signed-pre-key", "signed-pre-key-signature", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "device_name", "device_type", "identity_public_key",
			"signed_pre_key", "signed_pre_key_signature", "one_time_pre_keys",
			"key_version", "created_at", "updated_at",
		}).AddRow(
			"device-1", "user-1", "Safari on Mac", "web", "identity-public",
			"signed-pre-key", "signed-pre-key-signature", `["one-time-a","one-time-b"]`,
			1, time.Now(), time.Now(),
		))

	bundle, err := service.RegisterDeviceKeys("user-1", req)
	if err != nil {
		t.Fatalf("RegisterDeviceKeys failed: %v", err)
	}
	if bundle.UserID != "user-1" || bundle.DeviceID != "device-1" {
		t.Fatalf("unexpected bundle identifiers: %#v", bundle)
	}
	if len(bundle.OneTimePreKeys) != 2 {
		t.Fatalf("expected two one-time prekeys, got %d", len(bundle.OneTimePreKeys))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestKeyServiceStoreChatRecipientKeysStoresEncryptedEnvelopes(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	service := NewKeyService(db)
	envelopes := []models.EncryptedRecipientKey{{
		ChatID:             "chat-1",
		UserID:             "user-1",
		DeviceID:           "device-1",
		Algorithm:          "ECDH-P256-AES-GCM",
		Nonce:              "nonce",
		Ciphertext:         "wrapped-key",
		EphemeralPublicKey: "ephemeral-public",
	}}

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO chat_recipient_keys`).
		WithArgs("chat-1", "user-1", "device-1", "ECDH-P256-AES-GCM", "nonce", "wrapped-key", "ephemeral-public").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := service.StoreChatRecipientKeys("chat-1", envelopes); err != nil {
		t.Fatalf("StoreChatRecipientKeys failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
