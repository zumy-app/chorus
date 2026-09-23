package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// flagColumns is the SELECT projection used by FeatureFlagService.resolveFromDB.
func expectFlagResolve(mock sqlmock.Sqlmock, userID uuid.UUID, rows []services.FeatureFlagRow) {
	mock.ExpectQuery(`SELECT key, default_state, admin_only, beta_access, stable\s+FROM feature_flags`).
		WillReturnRows(flagRows(rows))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT flag_key, enabled FROM feature_flag_overrides WHERE user_id = $1`)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"flag_key", "enabled"}))
}

func flagRows(rows []services.FeatureFlagRow) *sqlmock.Rows {
	r := sqlmock.NewRows([]string{"key", "default_state", "admin_only", "beta_access", "stable"})
	for _, f := range rows {
		r.AddRow(f.Key, f.DefaultState, f.AdminOnly, f.BetaAccess, f.Stable)
	}
	return r
}

func TestFeatureFlagHandler_GetMyFlags_GeneralUser(t *testing.T) {
	router := setupTestRouter()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	uid := uuid.New()
	svc := services.NewFeatureFlagService(db, true)
	h := NewFeatureFlagHandler(svc)

	expectFlagResolve(mock, uid, []services.FeatureFlagRow{
		{Key: "grammar_insights", DefaultState: false, AdminOnly: false, BetaAccess: true, Stable: true},
		{Key: "video_calls", DefaultState: false, AdminOnly: true, BetaAccess: false, Stable: false},
	})

	router.GET("/users/me/flags", func(c *gin.Context) {
		c.Set("userID", uid.String())
		c.Set("userRole", "member")
		c.Set("userBetaAccess", false)
		h.GetMyFlags(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users/me/flags", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Flags map[string]bool `json:"flags"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Flags["grammar_insights"] != true {
		t.Fatalf("expected grammar_insights=true for general user, got %v", resp.Flags["grammar_insights"])
	}
	if resp.Flags["video_calls"] != false {
		t.Fatalf("expected video_calls=false for general user, got %v", resp.Flags["video_calls"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestFeatureFlagHandler_GetMyFlags_AdminSeesAll(t *testing.T) {
	router := setupTestRouter()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	uid := uuid.New()
	svc := services.NewFeatureFlagService(db, true)
	h := NewFeatureFlagHandler(svc)

	expectFlagResolve(mock, uid, []services.FeatureFlagRow{
		{Key: "video_calls", DefaultState: false, AdminOnly: true, BetaAccess: false, Stable: false},
	})

	router.GET("/users/me/flags", func(c *gin.Context) {
		c.Set("userID", uid.String())
		c.Set("userRole", services.RoleAdmin)
		c.Set("userBetaAccess", false)
		h.GetMyFlags(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users/me/flags", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Flags map[string]bool `json:"flags"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Flags["video_calls"] != true {
		t.Fatalf("expected video_calls=true for admin, got %v", resp.Flags["video_calls"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestFeatureFlagHandler_GetMyFlags_InvalidUserID(t *testing.T) {
	router := setupTestRouter()

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	svc := services.NewFeatureFlagService(db, true)
	h := NewFeatureFlagHandler(svc)

	router.GET("/users/me/flags", func(c *gin.Context) {
		c.Set("userID", "not-a-uuid")
		h.GetMyFlags(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users/me/flags", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid user id, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFeatureFlagHandler_UpdateFlagTiers(t *testing.T) {
	router := setupTestRouter()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	svc := services.NewFeatureFlagService(db, true)
	h := NewFeatureFlagHandler(svc)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE feature_flags SET admin_only = $1, beta_access = $2, stable = $3, updated_at = NOW() WHERE key = $4`)).
		WithArgs(false, false, true, "video_calls").
		WillReturnResult(sqlmock.NewResult(0, 1))

	router.PUT("/admin/feature-flags/:key", func(c *gin.Context) {
		c.Set("userRole", services.RoleAdmin)
		h.UpdateFlagTiers(c)
	})

	body := bytes.NewBufferString(`{"adminOnly":false,"betaAccess":false,"stable":true}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/admin/feature-flags/video_calls", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestFeatureFlagHandler_UpdateFlagTiers_NotFound(t *testing.T) {
	router := setupTestRouter()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	svc := services.NewFeatureFlagService(db, true)
	h := NewFeatureFlagHandler(svc)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE feature_flags SET admin_only = $1, beta_access = $2, stable = $3, updated_at = NOW() WHERE key = $4`)).
		WithArgs(false, false, true, "missing_flag").
		WillReturnResult(sqlmock.NewResult(0, 0))

	router.PUT("/admin/feature-flags/:key", h.UpdateFlagTiers)

	body := bytes.NewBufferString(`{"adminOnly":false,"betaAccess":false,"stable":true}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/admin/feature-flags/missing_flag", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when flag missing, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFeatureFlagHandler_SetUserOverride(t *testing.T) {
	router := setupTestRouter()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	svc := services.NewFeatureFlagService(db, true)
	h := NewFeatureFlagHandler(svc)

	target := uuid.New()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO feature_flag_overrides (user_id, flag_key, enabled, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW()) ON CONFLICT (user_id, flag_key) DO UPDATE SET enabled = EXCLUDED.enabled, updated_at = NOW()`)).
		WithArgs(target, "video_calls", true).
		WillReturnResult(sqlmock.NewResult(1, 1))

	router.POST("/admin/feature-flags/:key/overrides/:userId", h.SetUserOverride)

	body := bytes.NewBufferString(`{"enabled":true}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/admin/feature-flags/video_calls/overrides/"+target.String(), body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestFeatureFlagHandler_SetUserOverride_InvalidUserID(t *testing.T) {
	router := setupTestRouter()

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	svc := services.NewFeatureFlagService(db, true)
	h := NewFeatureFlagHandler(svc)

	router.POST("/admin/feature-flags/:key/overrides/:userId", h.SetUserOverride)

	body := bytes.NewBufferString(`{"enabled":true}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/admin/feature-flags/video_calls/overrides/not-a-uuid", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid user id, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFeatureFlagHandler_PreviewUserFlags_ResolvesAsGeneral(t *testing.T) {
	router := setupTestRouter()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	svc := services.NewFeatureFlagService(db, true)
	h := NewFeatureFlagHandler(svc)

	target := uuid.New()
	expectFlagResolve(mock, target, []services.FeatureFlagRow{
		{Key: "video_calls", DefaultState: false, AdminOnly: true, BetaAccess: false, Stable: false},
		{Key: "grammar_insights", DefaultState: false, AdminOnly: false, BetaAccess: true, Stable: true},
	})

	router.GET("/admin/feature-flags/preview/:userId", h.PreviewUserFlags)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/feature-flags/preview/"+target.String(), nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Flags map[string]bool `json:"flags"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Flags["video_calls"] != false {
		t.Fatalf("preview must resolve as general user: video_calls should be false, got %v", resp.Flags["video_calls"])
	}
	if resp.Flags["grammar_insights"] != true {
		t.Fatalf("preview: grammar_insights should be true (stable), got %v", resp.Flags["grammar_insights"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestFeatureFlagHandler_ListFlags(t *testing.T) {
	router := setupTestRouter()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	svc := services.NewFeatureFlagService(db, true)
	h := NewFeatureFlagHandler(svc)

	mock.ExpectQuery(`SELECT key, description, default_state, admin_only, beta_access, stable, created_at, updated_at\s+FROM feature_flags\s+ORDER BY key`).
		WillReturnRows(sqlmock.NewRows([]string{"key", "description", "default_state", "admin_only", "beta_access", "stable", "created_at", "updated_at"}).
			AddRow("grammar_insights", "Grammar analysis", false, false, true, true, time.Now(), time.Now()))

	router.GET("/admin/feature-flags", h.ListFlags)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/feature-flags", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Flags []services.FeatureFlagRow `json:"flags"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(resp.Flags) != 1 || resp.Flags[0].Key != "grammar_insights" {
		t.Fatalf("expected 1 flag grammar_insights, got %+v", resp.Flags)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
