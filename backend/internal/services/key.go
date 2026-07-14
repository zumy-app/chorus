package services

import (
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/chorus/messenger/internal/models"
)

type KeyService struct {
	db *sql.DB
}

func NewKeyService(db *sql.DB) *KeyService {
	return &KeyService{db: db}
}

func (s *KeyService) RegisterDeviceKeys(userID string, req models.RegisterDeviceKeysRequest) (*models.DeviceKeyBundle, error) {
	oneTimePreKeys, err := json.Marshal(req.OneTimePreKeys)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO user_devices (
			id, user_id, device_name, device_type, identity_public_key,
			signed_pre_key, signed_pre_key_signature, one_time_pre_keys
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			device_name = EXCLUDED.device_name,
			device_type = EXCLUDED.device_type,
			identity_public_key = EXCLUDED.identity_public_key,
			signed_pre_key = EXCLUDED.signed_pre_key,
			signed_pre_key_signature = EXCLUDED.signed_pre_key_signature,
			one_time_pre_keys = EXCLUDED.one_time_pre_keys,
			key_version = user_devices.key_version + 1,
			updated_at = CURRENT_TIMESTAMP
		WHERE user_devices.user_id = EXCLUDED.user_id
		RETURNING id, user_id, device_name, device_type, identity_public_key,
			signed_pre_key, signed_pre_key_signature, one_time_pre_keys,
			key_version, created_at, updated_at
	`

	return scanDeviceKeyBundle(s.db.QueryRow(query,
		req.DeviceID,
		userID,
		req.DeviceName,
		req.DeviceType,
		req.IdentityPublicKey,
		req.SignedPreKey,
		req.SignedPreKeySignature,
		oneTimePreKeys,
	))
}

func (s *KeyService) GetUserDeviceKeys(userID string) ([]models.DeviceKeyBundle, error) {
	query := `
		SELECT id, user_id, device_name, device_type, identity_public_key,
			signed_pre_key, signed_pre_key_signature, one_time_pre_keys,
			key_version, created_at, updated_at
		FROM user_devices
		WHERE user_id = $1
		ORDER BY created_at ASC
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bundles []models.DeviceKeyBundle
	for rows.Next() {
		bundle, err := scanDeviceKeyBundle(rows)
		if err != nil {
			return nil, err
		}
		bundles = append(bundles, *bundle)
	}
	return bundles, rows.Err()
}

func (s *KeyService) StoreChatRecipientKeys(chatID string, envelopes []models.EncryptedRecipientKey) error {
	if len(envelopes) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO chat_recipient_keys (chat_id, user_id, device_id, algorithm, nonce, ciphertext, ephemeral_public_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (chat_id, user_id, device_id) DO UPDATE SET
			algorithm = EXCLUDED.algorithm,
			nonce = EXCLUDED.nonce,
			ciphertext = EXCLUDED.ciphertext,
			ephemeral_public_key = EXCLUDED.ephemeral_public_key,
			updated_at = CURRENT_TIMESTAMP
	`

	for _, envelope := range envelopes {
		if envelope.UserID == "" || envelope.DeviceID == "" || envelope.Ciphertext == "" {
			return errors.New("recipient key envelope missing required fields")
		}
		if _, err := tx.Exec(query, chatID, envelope.UserID, envelope.DeviceID, envelope.Algorithm, envelope.Nonce, envelope.Ciphertext, envelope.EphemeralPublicKey); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *KeyService) GetChatRecipientKey(chatID, userID, deviceID string) (*models.EncryptedRecipientKey, error) {
	query := `
		SELECT chat_id, user_id, device_id, algorithm, nonce, ciphertext, COALESCE(ephemeral_public_key, '')
		FROM chat_recipient_keys
		WHERE chat_id = $1 AND user_id = $2 AND device_id = $3
	`

	envelope := &models.EncryptedRecipientKey{}
	err := s.db.QueryRow(query, chatID, userID, deviceID).Scan(
		&envelope.ChatID,
		&envelope.UserID,
		&envelope.DeviceID,
		&envelope.Algorithm,
		&envelope.Nonce,
		&envelope.Ciphertext,
		&envelope.EphemeralPublicKey,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return envelope, nil
}

type deviceKeyScanner interface {
	Scan(dest ...interface{}) error
}

func scanDeviceKeyBundle(scanner deviceKeyScanner) (*models.DeviceKeyBundle, error) {
	bundle := &models.DeviceKeyBundle{}
	var oneTimePreKeys []byte
	err := scanner.Scan(
		&bundle.DeviceID,
		&bundle.UserID,
		&bundle.DeviceName,
		&bundle.DeviceType,
		&bundle.IdentityPublicKey,
		&bundle.SignedPreKey,
		&bundle.SignedPreKeySignature,
		&oneTimePreKeys,
		&bundle.KeyVersion,
		&bundle.CreatedAt,
		&bundle.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(oneTimePreKeys) > 0 {
		if err := json.Unmarshal(oneTimePreKeys, &bundle.OneTimePreKeys); err != nil {
			return nil, err
		}
	}
	return bundle, nil
}
