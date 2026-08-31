package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestIPRateLimiterRejectsRequestsOverLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(IPRateLimiter(1, time.Minute))
	router.POST("/waitlist", func(c *gin.Context) { c.Status(http.StatusCreated) })

	for _, expected := range []int{http.StatusCreated, http.StatusTooManyRequests} {
		request := httptest.NewRequest(http.MethodPost, "/waitlist", nil)
		request.RemoteAddr = "192.0.2.1:1234"
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != expected {
			t.Fatalf("expected %d, got %d", expected, response.Code)
		}
	}
}

func TestRateLimiterCustomKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RateLimiter(1, time.Minute, func(c *gin.Context) string {
		return c.GetHeader("X-Api-Key")
	}, 0))
	router.POST("/register", func(c *gin.Context) { c.Status(http.StatusCreated) })

	cases := []struct {
		key  string
		want int
	}{
		{"k1", http.StatusCreated},
		{"k1", http.StatusTooManyRequests},
		{"k2", http.StatusCreated},
	}
	for _, tc := range cases {
		request := httptest.NewRequest(http.MethodPost, "/register", nil)
		request.Header.Set("X-Api-Key", tc.key)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != tc.want {
			t.Fatalf("key %s: expected %d, got %d", tc.key, tc.want, response.Code)
		}
	}
}

func TestUserRateLimiterLimitsPerUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Simulate AuthMiddleware, which sets userID on the context.
	router.Use(func(c *gin.Context) {
		if uid := c.GetHeader("X-Test-User"); uid != "" {
			c.Set("userID", uid)
		}
		c.Next()
	})
	router.Use(UserRateLimiter(2, time.Minute))
	router.POST("/translate", func(c *gin.Context) { c.Status(http.StatusCreated) })

	cases := []struct {
		user string
		want int
	}{
		{"user-a", http.StatusCreated},
		{"user-a", http.StatusCreated},
		{"user-a", http.StatusTooManyRequests},
		{"user-b", http.StatusCreated},
	}
	for _, tc := range cases {
		request := httptest.NewRequest(http.MethodPost, "/translate", nil)
		request.RemoteAddr = "192.0.2.1:1234"
		request.Header.Set("X-Test-User", tc.user)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != tc.want {
			t.Fatalf("user %s: expected %d, got %d", tc.user, tc.want, response.Code)
		}
	}
}

func TestUserRateLimiterFallsBackToIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(UserRateLimiter(1, time.Minute))
	router.POST("/translate", func(c *gin.Context) { c.Status(http.StatusCreated) })

	first := httptest.NewRequest(http.MethodPost, "/translate", nil)
	first.RemoteAddr = "192.0.2.9:1234"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, first)
	if rec.Code != http.StatusCreated {
		t.Fatalf("first request expected %d, got %d", http.StatusCreated, rec.Code)
	}

	second := httptest.NewRequest(http.MethodPost, "/translate", nil)
	second.RemoteAddr = "192.0.2.9:9999" // Same IP, different source port
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, second)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request expected %d, got %d", http.StatusTooManyRequests, rec2.Code)
	}
}
