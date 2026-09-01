package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

// newGalleryTestHandler builds a GalleryHandler backed by a sqlmock DB.
func newGalleryTestHandler(t *testing.T) (*GalleryHandler, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	galleryService := services.NewGalleryService(db)
	return NewGalleryHandler(galleryService), mock, func() { db.Close() }
}

// TestGetChatGallery_Success verifies the handler returns the gallery payload
// for an authenticated participant (task 6.5).
func TestGetChatGallery_Success(t *testing.T) {
	h, mock, cleanup := newGalleryTestHandler(t)
	defer cleanup()

	now := time.Now()

	mock.ExpectQuery(`SELECT ma\.type, COUNT\(\*\) FROM media_attachments`).
		WithArgs("user-1", "chat-1").
		WillReturnRows(sqlmock.NewRows([]string{"type", "count"}).
			AddRow("image", 1).
			AddRow("link", 1))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM media_attachments`).
		WithArgs("user-1", "chat-1", pq.Array([]string{"image", "video"})).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT ma\.id, ma\.message_id, ma\.type`).
		WithArgs("user-1", "chat-1", pq.Array([]string{"image", "video"}), 30, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "message_id", "type", "file_name", "file_size", "mime_type",
			"url", "thumbnail_url", "created_at", "chat_id",
			"latitude", "longitude", "location_name",
			"sender_id", "sender_name", "sender_username",
		}).
			AddRow("att-1", "msg-1", "image", "pic.jpg", 100, "image/jpeg", "http://cdn/pic.jpg", "http://cdn/pic-t.jpg", now, "chat-1", nil, nil, "", "u-maria", "Maria", "maria"))

	w := serveGallery(t, h.GetChatGallery, "/chats/:chatId/gallery", "/chats/chat-1/gallery?type=media", "user-1")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if body := w.Body.String(); body == "" {
		t.Fatal("expected a JSON body")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestGetChatGallery_Unauthorized ensures a missing userID is rejected before
// any DB access.
func TestGetChatGallery_Unauthorized(t *testing.T) {
	h, _, cleanup := newGalleryTestHandler(t)
	defer cleanup()

	w := serveGallery(t, h.GetChatGallery, "/chats/:chatId/gallery", "/chats/chat-1/gallery", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// TestGetChatGallery_MissingChatID ensures a missing chatId is a validation error.
func TestGetChatGallery_MissingChatID(t *testing.T) {
	h, _, cleanup := newGalleryTestHandler(t)
	defer cleanup()

	// Mount the handler on a route with no :chatId param so the guard branch
	// (rather than Gin's own 404 for an empty segment) is exercised.
	w := serveGallery(t, h.GetChatGallery, "/gallery", "/gallery", "user-1")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func serveGallery(t *testing.T, handler func(*gin.Context), routePattern, path, userID string) *httptest.ResponseRecorder {
	t.Helper()
	router := setupTestRouter()
	router.Handle(http.MethodGet, routePattern, func(c *gin.Context) {
		c.Set("userID", userID)
		handler(c)
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, path, nil)
	router.ServeHTTP(w, req)
	return w
}
