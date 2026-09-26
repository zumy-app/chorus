package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// newFlagTestService returns a FeatureFlagService backed by sqlmock with a
// single flag row: video_calls -> (default=false, adminOnly=true).
func newFlagTestService(t *testing.T, userID uuid.UUID) (*services.FeatureFlagService, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	mock.ExpectQuery(`SELECT key, default_state, admin_only, beta_access, stable\s+FROM feature_flags`).
		WillReturnRows(sqlmock.NewRows([]string{"key", "default_state", "admin_only", "beta_access", "stable"}).
			AddRow("video_calls", false, true, false, false).
			AddRow("grammar_insights", false, false, true, true))
	mock.ExpectQuery(`SELECT flag_key, enabled\s+FROM feature_flag_overrides\s+WHERE user_id = \$1`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"flag_key", "enabled"}))

	return services.NewFeatureFlagService(db, true), mock
}

func flagRouter(svc *services.FeatureFlagService, key string, setContext func(c *gin.Context)) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { setContext(c); c.Next() })
	r.GET("/gated", RequireFlag(svc, key), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestRequireFlag_AdminSeesAdminOnlyFlag(t *testing.T) {
	uid := uuid.New()
	svc, mock := newFlagTestService(t, uid)

	router := flagRouter(svc, "video_calls", func(c *gin.Context) {
		c.Set("userID", uid.String())
		c.Set("userRole", services.RoleAdmin)
		c.Set("userBetaAccess", false)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/gated", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestRequireFlag_GeneralUserDeniedAdminOnlyFlag(t *testing.T) {
	uid := uuid.New()
	svc, mock := newFlagTestService(t, uid)

	router := flagRouter(svc, "video_calls", func(c *gin.Context) {
		c.Set("userID", uid.String())
		c.Set("userRole", "member")
		c.Set("userBetaAccess", false)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/gated", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for general user, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestRequireFlag_GeneralUserAllowedStableFlag(t *testing.T) {
	uid := uuid.New()
	svc, mock := newFlagTestService(t, uid)

	router := flagRouter(svc, "grammar_insights", func(c *gin.Context) {
		c.Set("userID", uid.String())
		c.Set("userRole", "member")
		c.Set("userBetaAccess", false)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/gated", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for stable flag, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestRequireFlag_Unauthenticated(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	svc := services.NewFeatureFlagService(db, true)

	router := flagRouter(svc, "video_calls", func(c *gin.Context) {
		// No userID in context
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/gated", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRequireFlag_UnknownFlagDeniedWhenDefaultOff(t *testing.T) {
	uid := uuid.New()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	mock.ExpectQuery(`SELECT key, default_state, admin_only, beta_access, stable\s+FROM feature_flags`).
		WillReturnRows(sqlmock.NewRows([]string{"key", "default_state", "admin_only", "beta_access", "stable"}))
	mock.ExpectQuery(`SELECT flag_key, enabled\s+FROM feature_flag_overrides\s+WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"flag_key", "enabled"}))

	svc := services.NewFeatureFlagService(db, true) // defaultOff = true (production mode)
	router := flagRouter(svc, "never_existed_flag", func(c *gin.Context) {
		c.Set("userID", uid.String())
		c.Set("userRole", services.RoleAdmin)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/gated", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for unknown flag with defaultOff, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestFlagContextMiddleware_InjectsResolvedMap(t *testing.T) {
	uid := uuid.New()
	svc, mock := newFlagTestService(t, uid)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uid.String())
		c.Set("userRole", services.RoleAdmin)
		c.Next()
	})
	r.Use(FlagContextMiddleware(svc))
	r.GET("/check", func(c *gin.Context) {
		got := FeatureFlagFromContext(c, "video_calls")
		c.JSON(http.StatusOK, gin.H{"videoCalls": got})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/check", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if body := w.Body.String(); body != `{"videoCalls":true}` {
		t.Fatalf("expected videoCalls=true for admin, got %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestFeatureFlagFromContext_MissingMap(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/check", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"flag": FeatureFlagFromContext(c, "video_calls")})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/check", nil)
	r.ServeHTTP(w, req)

	if body := w.Body.String(); body != `{"flag":false}` {
		t.Fatalf("expected false with no context map, got %s", body)
	}
}

