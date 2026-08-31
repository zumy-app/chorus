package services

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/chorus/messenger/internal/models"
	"github.com/lib/pq"
)

// MaxContactScanHashes bounds how many hashed identifiers a single contact scan
// may submit. It is both an abuse guard and a work-bounding limit so the scan
// stays cheap for a P0 launch.
const MaxContactScanHashes = 1000

var ErrTooManyContactHashes = errors.New("too many contact hashes")

// HashIdentifier returns the canonical, privacy-preserving identifier used for
// on-platform detection. Raw contacts must never be transmitted — clients send
// only the output of this function over the same normalized (trim + lower)
// identifier, so the server can match without ever seeing the raw value.
func HashIdentifier(identifier string) string {
	normalized := strings.ToLower(strings.TrimSpace(identifier))
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

// ContactService implements the permission-gated contact scan (FR-22/23 REQ
// 2.4): given a set of hashed identifiers from the device's address book, it
// detects which of them belong to existing on-platform users. The server never
// receives or stores raw contact data.
type ContactService struct {
	db *sql.DB
}

func NewContactService(db *sql.DB) *ContactService {
	return &ContactService{db: db}
}

// ScanHashed matches the client-supplied hashes against registered users'
// emails. It returns only on-platform matches (excluding the requester and any
// suspended/deleted accounts) with the matched hash echoed back so the client
// can correlate the result with its address-book entry.
func (s *ContactService) ScanHashed(userID string, hashes []string) ([]models.ContactMatch, error) {
	if len(hashes) == 0 {
		return []models.ContactMatch{}, nil
	}
	if len(hashes) > MaxContactScanHashes {
		return nil, ErrTooManyContactHashes
	}

	// Normalize the client set (defensive: hashes come lowercased hex already,
	// but trim/lower defensively) and dedupe.
	want := make(map[string]struct{}, len(hashes))
	for _, h := range hashes {
		h = strings.ToLower(strings.TrimSpace(h))
		if h != "" {
			want[h] = struct{}{}
		}
	}

	rows, err := s.db.Query(`
		SELECT id, username, email, display_name, native_language, target_languages
		FROM users
		WHERE id != $1 AND deleted_at IS NULL AND suspended_at IS NULL`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matches := []models.ContactMatch{}
	for rows.Next() {
		var userID, username, email, displayName, nativeLanguage string
		var targetLanguages []string
		if err := rows.Scan(&userID, &username, &email, &displayName, &nativeLanguage, pq.Array(&targetLanguages)); err != nil {
			return nil, err
		}
		emailHash := HashIdentifier(email)
		if _, ok := want[emailHash]; !ok {
			continue
		}
		matches = append(matches, models.ContactMatch{
			UserID:          userID,
			Username:        username,
			DisplayName:     displayName,
			Email:           email,
			EmailHash:       emailHash,
			NativeLanguage:  nativeLanguage,
			TargetLanguages: targetLanguages,
		})
	}
	return matches, rows.Err()
}
