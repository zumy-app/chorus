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
