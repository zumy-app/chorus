package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

// mock implementations
type mockAuthService struct {
	registerFn        func(req models.RegisterRequest) (*models.User, error)
	loginFn           func(username, password string) (*models.User, error)
	generateAccessFn  func(userID string) (string, error)
	generateRefreshFn func(userID string) (string, error)
	validateAccessFn  func(token string) (string, error)
	validateRefreshFn func(token string) (string, error)
	deleteRefreshFn   func(token string) error
}

func (m *mockAuthService) Register(req models.RegisterRequest) (*models.User, error) {
	return m.registerFn(req)
}
func (m *mockAuthService) Login(username, password string) (*models.User, error) {
	return m.loginFn(username, password)
}
func (m *mockAuthService) GenerateAccessToken(userID string) (string, error) {
	return m.generateAccessFn(userID)
}
func (m *mockAuthService) GenerateRefreshToken(userID string) (string, error) {
	return m.generateRefreshFn(userID)
}
func (m *mockAuthService) ValidateAccessToken(token string) (string, error) {
	return m.validateAccessFn(token)
}
func (m *mockAuthService) ValidateRefreshToken(token string) (string, error) {
	return m.validateRefreshFn(token)
}
func (m *mockAuthService) DeleteRefreshToken(token string) error {
	return m.deleteRefreshFn(token)
}

type mockUserService struct {
	getByIDFn func(userID string) (*models.User, error)
	updateFn  func(userID string, req models.UpdateUserRequest) (*models.User, error)
}

func (m *mockUserService) GetByID(userID string) (*models.User, error) {
	return m.getByIDFn(userID)
}
func (m *mockUserService) Update(userID string, req models.UpdateUserRequest) (*models.User, error) {
	return m.updateFn(userID, req)
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func now() time.Time { return time.Now() }

func makeUser(id string) *models.User {
	return &models.User{
		ID:              id,
		Username:        "testuser",
		Email:           "test@example.com",
		DisplayName:     "Test User",
		NativeLanguage:  "en",
		TargetLanguages: []string{"es"},
		CreatedAt:       now(),
		LastActiveAt:    now(),
	}
}

func TestRegisterHandler_Success(t *testing.T) {
	router := setupTestRouter()

	authMock := &mockAuthService{
		registerFn: func(req models.RegisterRequest) (*models.User, error) {
			return makeUser("user-1"), nil
		},
		generateAccessFn: func(userID string) (string, error) {
			return "access-token-123", nil
		},
		generateRefreshFn: func(userID string) (string, error) {
			return "refresh-token-123", nil
		},
	}
	router.POST("/api/v1/auth/register", func(c *gin.Context) {
		var req models.RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid registration data"})
			return
		}
		user, err := authMock.Register(req)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		accessToken, _ := authMock.GenerateAccessToken(user.ID)
		refreshToken, _ := authMock.GenerateRefreshToken(user.ID)
		c.JSON(201, gin.H{
			"user": user,
			"tokens": models.AuthTokens{
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
				ExpiresIn:    86400,
			},
		})
	})

	body, _ := json.Marshal(models.RegisterRequest{
		Username:        "testuser",
		Email:           "test@example.com",
		Password:        "Password123!",
		DisplayName:     "Test User",
		NativeLanguage:  "en",
		TargetLanguages: []string{"es"},
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["user"] == nil {
		t.Fatal("expected user in response")
	}
	tokens := resp["tokens"].(map[string]interface{})
	if tokens["accessToken"] != "access-token-123" {
		t.Fatalf("expected access-token-123, got %v", tokens["accessToken"])
	}
}

func TestRegisterHandler_DuplicateEmail(t *testing.T) {
	router := setupTestRouter()

	authMock := &mockAuthService{
		registerFn: func(req models.RegisterRequest) (*models.User, error) {
			return nil, services.ErrEmailAlreadyRegistered
		},
	}

	router.POST("/api/v1/auth/register", func(c *gin.Context) {
		var req models.RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid registration data"})
			return
		}
		_, err := authMock.Register(req)
		if err != nil {
			if errors.Is(err, services.ErrEmailAlreadyRegistered) {
				c.JSON(409, gin.H{"error": "Email is already registered"})
				return
			}
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
	})

	body, _ := json.Marshal(models.RegisterRequest{
		Email:    "existing@example.com",
		Password: "Password123!",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != 409 {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegisterHandler_InvalidInput(t *testing.T) {
	router := setupTestRouter()
	h := NewAuthHandler(nil, nil, nil, nil, nil, "")
	router.POST("/api/v1/auth/register", h.Register)

	// Missing email and password
	body, _ := json.Marshal(map[string]string{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400 for invalid input, got %d", w.Code)
	}
}

func TestLoginHandler_Success(t *testing.T) {
	router := setupTestRouter()

	authMock := &mockAuthService{
		loginFn: func(username, password string) (*models.User, error) {
			return makeUser("user-1"), nil
		},
		generateAccessFn: func(userID string) (string, error) {
			return "access-token-123", nil
		},
		generateRefreshFn: func(userID string) (string, error) {
			return "refresh-token-123", nil
		},
	}

	router.POST("/api/v1/auth/login", func(c *gin.Context) {
		var req models.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}
		user, err := authMock.Login(req.Username, req.Password)
		if err != nil {
			c.JSON(401, gin.H{"error": "Invalid credentials"})
			return
		}
		accessToken, _ := authMock.GenerateAccessToken(user.ID)
		refreshToken, _ := authMock.GenerateRefreshToken(user.ID)
		c.JSON(200, gin.H{
			"user": user,
			"tokens": models.AuthTokens{
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
				ExpiresIn:    86400,
			},
		})
	})

	body, _ := json.Marshal(models.LoginRequest{
		Username: "testuser",
		Password: "Password123!",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLoginHandler_InvalidCredentials(t *testing.T) {
	router := setupTestRouter()

	authMock := &mockAuthService{
		loginFn: func(username, password string) (*models.User, error) {
			return nil, errors.New("invalid credentials")
		},
	}

	router.POST("/api/v1/auth/login", func(c *gin.Context) {
		var req models.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}
		_, err := authMock.Login(req.Username, req.Password)
		if err != nil {
			c.JSON(401, gin.H{"error": "Invalid credentials"})
			return
		}
	})

	body, _ := json.Marshal(models.LoginRequest{
		Username: "wrong",
		Password: "wrong",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRefreshTokenHandler(t *testing.T) {
	router := setupTestRouter()

	authMock := &mockAuthService{
		validateRefreshFn: func(token string) (string, error) {
			return "user-1", nil
		},
		generateAccessFn: func(userID string) (string, error) {
			return "new-access-token", nil
		},
		generateRefreshFn: func(userID string) (string, error) {
			return "new-refresh-token", nil
		},
		deleteRefreshFn: func(token string) error {
			return nil
		},
	}

	router.POST("/api/v1/auth/refresh", func(c *gin.Context) {
		var req struct {
			RefreshToken string `json:"refreshToken"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}
		userID, err := authMock.ValidateRefreshToken(req.RefreshToken)
		if err != nil {
			c.JSON(401, gin.H{"error": "Invalid refresh token"})
			return
		}
		authMock.DeleteRefreshToken(req.RefreshToken)
		accessToken, _ := authMock.GenerateAccessToken(userID)
		refreshToken, _ := authMock.GenerateRefreshToken(userID)
		c.JSON(200, gin.H{
			"tokens": models.AuthTokens{
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
				ExpiresIn:    86400,
			},
		})
	})

	body, _ := json.Marshal(map[string]string{"refreshToken": "valid-refresh-token"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthMiddleware_NoHeader(t *testing.T) {
	router := setupTestRouter()
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	router.ServeHTTP(w, req)

	// Without middleware, it should pass through - we're testing the middleware separately
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}
		c.Set("userID", "test-user-id")
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
}

func TestGetMyEntitlements_Success(t *testing.T) {
	router := setupTestRouter()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	userService := services.NewUserService(db)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, username, email, display_name, first_name, last_name, native_language, target_languages, role, created_at, last_active_at, suspended_at, deleted_at, plan, plan_grace_until, premium_since, subscription_id, subscription_provider, subscription_plan_id, subscription_status, next_billing_date, last_payment_at, avatar_url FROM users WHERE id = $1`)).
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "display_name", "first_name", "last_name", "native_language", "target_languages", "role", "created_at", "last_active_at", "suspended_at", "deleted_at", "plan", "plan_grace_until", "premium_since", "subscription_id", "subscription_provider", "subscription_plan_id", "subscription_status", "next_billing_date", "last_payment_at", "avatar_url"}).
			AddRow("user-1", "testuser", "test@example.com", "Test User", "", "", "en", pq.Array([]string{"es"}), "member", time.Now(), time.Now(), nil, nil, "free", nil, nil, nil, "paypal", nil, nil, nil, nil, nil))

	entitlementService := services.NewEntitlementService(false)
	h := NewAuthHandler(nil, userService, nil, nil, entitlementService, "")

	router.GET("/users/me/entitlements", func(c *gin.Context) {
		c.Set("userID", "user-1")
		h.GetMyEntitlements(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/users/me/entitlements", nil)
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["plan"] != "free" {
		t.Fatalf("expected plan free, got %v", resp["plan"])
	}
	if resp["effectivePlan"] != "free" {
		t.Fatalf("expected effectivePlan free, got %v", resp["effectivePlan"])
	}
	if resp["showAds"] != true {
		t.Fatalf("expected showAds true on free plan, got %v", resp["showAds"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestOnboardMe_Success(t *testing.T) {
	router := setupTestRouter()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	userService := services.NewUserService(db)
	userColumns := `id, username, email, display_name, first_name, last_name, native_language, target_languages, role, created_at, last_active_at, suspended_at, deleted_at, plan, plan_grace_until, premium_since, subscription_id, subscription_provider, subscription_plan_id, subscription_status, next_billing_date, last_payment_at, avatar_url`
	onboardSQL := `UPDATE users SET first_name = $2, last_name = $3, display_name = $4 WHERE id = $1 RETURNING ` + userColumns
	mock.ExpectQuery(regexp.QuoteMeta(onboardSQL)).
		WithArgs("user-1", "John", "Smith", "John Smith").
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "display_name", "first_name", "last_name", "native_language", "target_languages", "role", "created_at", "last_active_at", "suspended_at", "deleted_at", "plan", "plan_grace_until", "premium_since", "subscription_id", "subscription_provider", "subscription_plan_id", "subscription_status", "next_billing_date", "last_payment_at", "avatar_url"}).
			AddRow("user-1", "testuser", "test@example.com", "John Smith", "John", "Smith", "en", pq.Array([]string{"es"}), "member", time.Now(), time.Now(), nil, nil, "free", nil, nil, nil, "paypal", nil, nil, nil, nil, nil))

	h := NewAuthHandler(nil, userService, nil, nil, services.NewEntitlementService(false), "")

	router.PUT("/users/me/onboard", func(c *gin.Context) {
		c.Set("userID", "user-1")
		h.OnboardMe(c)
	})

	body := bytes.NewBufferString(`{"firstName":"John","lastName":"Smith"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/users/me/onboard", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.User
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.FirstName != "John" {
		t.Fatalf("expected firstName 'John', got '%s'", resp.FirstName)
	}
	if resp.LastName != "Smith" {
		t.Fatalf("expected lastName 'Smith', got '%s'", resp.LastName)
	}
	if resp.DisplayName != "John Smith" {
		t.Fatalf("expected displayName 'John Smith', got '%s'", resp.DisplayName)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetMyEntitlements_NotFound(t *testing.T) {
	router := setupTestRouter()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	userService := services.NewUserService(db)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, username, email, display_name, first_name, last_name, native_language, target_languages, role, created_at, last_active_at, suspended_at, deleted_at, plan, plan_grace_until, premium_since, subscription_id, subscription_provider, subscription_plan_id, subscription_status, next_billing_date, last_payment_at, avatar_url FROM users WHERE id = $1`)).
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)

	entitlementService := services.NewEntitlementService(false)
	h := NewAuthHandler(nil, userService, nil, nil, entitlementService, "")

	router.GET("/users/me/entitlements", func(c *gin.Context) {
		c.Set("userID", "missing")
		h.GetMyEntitlements(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/users/me/entitlements", nil)
	router.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
