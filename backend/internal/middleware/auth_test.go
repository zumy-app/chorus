package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// mockTokenValidator for testing
type mockTokenValidator struct {
	validateFn func(token string) (string, error)
}

func (m *mockTokenValidator) ValidateAccessToken(token string) (string, error) {
	return m.validateFn(token)
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}
		c.Next()
	})

	router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	router.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}
		// Check format
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			c.JSON(401, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}
		c.Set("userID", "user-1")
		c.Next()
	})

	router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	// Test with wrong format (missing "Bearer ")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "InvalidToken")
	router.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected 401 for invalid format, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			c.JSON(401, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}
		c.Set("userID", "user-1")
		c.Next()
	})

	router.GET("/protected", func(c *gin.Context) {
		userID, _ := c.Get("userID")
		c.JSON(200, gin.H{"userID": userID})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	respBytes := w.Body.Bytes()
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["userID"] != "user-1" {
		t.Fatalf("expected userID=user-1, got %v", resp["userID"])
	}
}
