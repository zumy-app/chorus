package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/chorus/messenger/internal/models"
	"github.com/google/uuid"
)

// Supported attachment media types. These map 1:1 onto the media_attachments
// table CHECK constraint (task 6.3/6.5), so a document share writes a
// 'document' row that the media gallery and search already read.
var (
	typeImage    = "image"
	typeVideo    = "video"
	typeAudio    = "audio"
	typeDocument = "document"

	// allowedTypes is the set of media types an upload may resolve to.
	allowedTypes = map[string]bool{
		typeImage:    true,
		typeVideo:    true,
		typeAudio:    true,
		typeDocument: true,
	}

	// maxUploadBytes is the global upload cap. It is overridden by the service
	// constructor field (wired from MAX_UPLOAD_MB).
	maxUploadBytes int64 = 50 << 20 // 50 MB
)

// knownDocumentExtensions are the extensions treated as documents (the task
// 6.6 PDF/doc/xlsx class plus office/legacy/plain-text formats). When a client
// uploads without a reliable Content-Type, we still classify the file by name.
var knownDocumentExtensions = map[string]bool{
	// PDF
	".pdf": true,
	// MS Word (legacy + OOXML)
	".doc": true, ".docx": true, ".dot": true, ".rtf": true,
	// MS Excel (legacy + OOXML)
	".xls": true, ".xlsx": true, ".xlsm": true, ".ods": true,
	// MS PowerPoint (legacy + OOXML)
	".ppt": true, ".pptx": true, ".odp": true,
	// Plain text / markup
	".txt": true, ".csv": true, ".md": true, ".log": true,
	// OpenDocument
	".odt": true, ".dotx": true,
}

// fileForbiddenChars replaces characters that are unsafe in a file name
// (path separators, control chars, quotes) so a stored name can never escape
// the upload directory or be confused by a URL.
var fileForbiddenChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// AttachmentService persists file/document shares (task 6.6): it writes the
// uploaded bytes to disk, creates the parent message, records a
// media_attachments row, and returns the message with the attachment populated
// so callers can broadcast it exactly like a text message. The read surface
// (media gallery, media search) already understands these rows.
type AttachmentService struct {
	db *sql.DB
	// uploadDir is the root directory on disk where attachments are stored.
	uploadDir string
	// mediaBaseURL is the public URL prefix for attachments (e.g. "/media").
	mediaBaseURL string
	// maxBytes is the per-upload size cap.
	maxBytes int64
}

// NewAttachmentService creates an AttachmentService. uploadDir is created
// (with 0755) if it does not already exist.
func NewAttachmentService(db *sql.DB, uploadDir, mediaBaseURL string, maxBytes int64) (*AttachmentService, error) {
	if strings.TrimSpace(uploadDir) == "" {
		uploadDir = "./uploads"
	}
	if strings.TrimSpace(mediaBaseURL) == "" {
		mediaBaseURL = "/media"
	}
	if maxBytes <= 0 {
		maxBytes = maxUploadBytes
	}
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create upload dir %q: %w", uploadDir, err)
	}
	return &AttachmentService{
		db:           db,
		uploadDir:    uploadDir,
		mediaBaseURL: strings.TrimRight(mediaBaseURL, "/"),
		maxBytes:     maxBytes,
	}, nil
}

// MaxBytes returns the configured upload size cap.
func (s *AttachmentService) MaxBytes() int64 { return s.maxBytes }

// SupportedType reports whether a media type is accepted on upload.
func (s *AttachmentService) SupportedType(mediaType string) bool {
	return allowedTypes[strings.ToLower(strings.TrimSpace(mediaType))]
}

// SendFile persists a shared file and returns the created message (with the
// attachment attached). The caller:
//
//   - must be a participant of chatID (checked by the handler),
//   - supplies the raw bytes via src, the original file name, and size,
//   - may pass an explicit media type (from the multipart "type" field) or an
//     empty string to have it inferred from the file name / MIME type,
//   - may pass a caption; when empty the file name is used as the message text.
//
// It validates size and type, stores the bytes under
// <uploadDir>/<attachmentID>/<sanitizedFileName>, then persists a row in
// media_attachments tied to a fresh message row in the same transaction so a
// partial write can never leave a media_attachments row without a message (and
// the returned message carries Media[0] so realtime fan-out and the send
// response include it).
func (s *AttachmentService) SendFile(ctx context.Context, chatID, userID, caption, fileName, mediaType string, src io.Reader, size int64) (*models.Message, error) {
	if size <= 0 {
		return nil, errors.New("uploaded file is empty")
	}
	if size > s.maxBytes {
		return nil, fmt.Errorf("file exceeds the %d byte upload limit", s.maxBytes)
	}

	// Resolve support: explicit type wins; otherwise infer from the name/MIME.
	resolvedType, err := s.resolveType(mediaType, fileName)
	if err != nil {
		return nil, err
	}

	// Sanitize the stored file name so it can be served and downloaded safely.
	storedName := sanitizeFileName(fileName)
	if storedName == "" {
		storedName = "attachment"
	}

	attachmentID := uuid.NewString()
	relPath := fmt.Sprintf("%s/%s", attachmentID, storedName)
	absPath := filepath.Join(s.uploadDir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create attachment dir: %w", err)
	}

	// Stream bytes to disk first (outside the DB tx so we only roll back the
	// row, never hold a DB lock while writing a large file).
	if err := writeFile(absPath, src); err != nil {
		return nil, fmt.Errorf("failed to store attachment: %w", err)
	}

	text := strings.TrimSpace(caption)
	if text == "" {
		text = storedName
	}

	mimeType := detectMIME(fileName, mediaType)

	// Persist the message + attachment atomically.
	message := &models.Message{}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	msg, msgErr := createMessageTx(ctx, tx, chatID, userID, text)
	if msgErr != nil {
		return nil, msgErr
	}
	message = msg

	att := models.MediaAttachment{
		ID:        attachmentID,
		MessageID: message.ID,
		ChatID:    chatID,
		Type:      resolvedType,
		FileName:  storedName,
		FileSize:  size,
		MimeType:  mimeType,
		URL:       s.publicURL(relPath),
		CreatedAt: time.Now().UTC(),
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO media_attachments (id, message_id, type, file_name, file_size, mime_type, url)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, att.ID, att.MessageID, att.Type, att.FileName, att.FileSize, att.MimeType, att.URL); err != nil {
		return nil, fmt.Errorf("failed to record attachment: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	message.Media = []models.MediaAttachment{att}
	return message, nil
}

// publicURL builds the client-facing URL for an attachment's storage path. The
// path is joined to the configured base URL (a local prefix like /media, or a
// CDN origin + bucket prefix in production).
func (s *AttachmentService) publicURL(relPath string) string {
	relPath = strings.ReplaceAll(relPath, "\\", "/")
	return s.mediaBaseURL + "/" + relPath
}

// resolveType returns the canonical media type for an upload. An explicit
// type (from the multipart "type" field) is validated against the allowed set;
// otherwise the type is inferred from the file extension, MIME type, and the
// known document extension table.
func (s *AttachmentService) resolveType(explicitType, fileName string) (string, error) {
	if explicitType != "" {
		t := strings.ToLower(strings.TrimSpace(explicitType))
		if !allowedTypes[t] {
			return "", fmt.Errorf("unsupported media type %q", explicitType)
		}
		return t, nil
	}

	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg", ".heic", ".avif":
		return typeImage, nil
	case ".mp4", ".mov", ".mkv", ".webm", ".avi", ".m4v":
		return typeVideo, nil
	case ".mp3", ".wav", ".ogg", ".m4a", ".aac", ".flac", ".opus":
		return typeAudio, nil
	}

	// The task 6.6 document class (PDF/doc/xlsx + office/text/legacy) and any
	// MIME type that is clearly a document.
	if knownDocumentExtensions[ext] {
		return typeDocument, nil
	}

	// Some uploads carry a meaningful extension-less Content-Type (e.g. a
	// blob marked application/pdf); use it as a last hint. mime.TypeByExtension
	// needs a leading dot.
	mimeType := detectMIME(fileName, "")
	if strings.HasPrefix(mimeType, "image/") {
		return typeImage, nil
	}
	if strings.HasPrefix(mimeType, "video/") {
		return typeVideo, nil
	}
	if strings.HasPrefix(mimeType, "audio/") {
		return typeAudio, nil
	}
	if isDocumentMIME(mimeType) || knownDocumentExtensions[ext] {
		return typeDocument, nil
	}

	return "", fmt.Errorf("could not determine a supported media type for %q", fileName)
}

// createMessageTx inserts a message row inside the caller's transaction and
// returns the persisted message.
func createMessageTx(ctx context.Context, tx *sql.Tx, chatID, userID, text string) (*models.Message, error) {
	message := &models.Message{}
	err := tx.QueryRowContext(ctx, `
		INSERT INTO messages (chat_id, sender_id, text, delivery_status)
		VALUES ($1, $2, $3, 'sent')
		RETURNING id, chat_id, sender_id, text, COALESCE(original_language, ''),
		          COALESCE(translations, '{}'::jsonb), delivery_status, reply_to_id, created_at
	`, chatID, userID, text).Scan(
		&message.ID,
		&message.ChatID,
		&message.SenderID,
		&message.Text,
		&message.OriginalLanguage,
		new([]byte),
		&message.DeliveryStatus,
		&message.ReplyToID,
		&message.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}
	return message, nil
}

// writeFile streams src into a new file at absPath, closing it on completion.
func writeFile(absPath string, src io.Reader) error {
	f, err := os.Create(absPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, src); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// sanitizeFileName reduces a client-supplied name to a safe basename: path
// separators and unsafe characters are replaced so the stored path can never
// traverse the upload directory.
func sanitizeFileName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = fileForbiddenChars.ReplaceAllString(name, "_")
	name = strings.Trim(name, "._")
	if len(name) > 255 {
		ext := filepath.Ext(name)
		name = strings.TrimSuffix(name, ext)
		if len(name) > 240 {
			name = name[:240]
		}
		name += ext
	}
	return name
}

// detectMIME returns a best-effort MIME type for an attachment. It uses the
// client-supplied media type hint when given, then the standard library's
// extension map, and finally a sniff-based fallback.
func detectMIME(fileName, mediaTypeHint string) string {
	if hint := strings.TrimSpace(mediaTypeHint); hint != "" && strings.Contains(hint, "/") {
		return strings.ToLower(hint)
	}
	if mt := mime.TypeByExtension(strings.ToLower(filepath.Ext(fileName))); mt != "" {
		// TypeByExtension may include charset params; drop them.
		if i := strings.IndexByte(mt, ';'); i >= 0 {
			mt = mt[:i]
		}
		return strings.ToLower(strings.TrimSpace(mt))
	}
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx", ".dotx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx", ".xlsm":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".ppt":
		return "application/vnd.ms-powerpoint"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".txt", ".md", ".log", ".csv":
		return "text/plain"
	}
	return "application/octet-stream"
}

// isDocumentMIME reports whether a MIME type names a document/file type.
func isDocumentMIME(m string) bool {
	m = strings.ToLower(strings.TrimSpace(m))
	if strings.HasPrefix(m, "application/") && m != "application/octet-stream" {
		return true
	}
	if strings.HasPrefix(m, "text/") {
		return true
	}
	return false
}
