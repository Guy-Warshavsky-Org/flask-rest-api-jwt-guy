package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/flask-rest-api-jwt-guy/internal/config"
	"github.com/flask-rest-api-jwt-guy/internal/handler"
	"github.com/flask-rest-api-jwt-guy/internal/middleware"
	"github.com/flask-rest-api-jwt-guy/internal/model"
)

// testEnv holds shared test dependencies.
type testEnv struct {
	Router    *gin.Engine
	DB        *gorm.DB
	Config    *config.Config
	Blacklist *middleware.TokenBlacklist
}

// setupTestEnv creates a fresh in-memory SQLite database and Gin router for tests.
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file::memory:?_foreign_keys=on"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Enable foreign key enforcement for SQLite
	sqlDB, dbErr := db.DB()
	if dbErr != nil {
		t.Fatalf("Failed to get underlying DB: %v", dbErr)
	}
	if _, err := sqlDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	if err := model.AutoMigrateAll(db); err != nil {
		t.Fatalf("Failed to auto-migrate: %v", err)
	}

	cfg := &config.Config{
		Env:             "test",
		JWTSecret:       "test-secret-key",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 30 * 24 * time.Hour,
		HTTPPort:        "5000",
	}

	blacklist := middleware.NewTokenBlacklist()

	router := gin.New()

	userHandler := &handler.UserHandler{
		DB:        db,
		Config:    cfg,
		Blacklist: blacklist,
	}
	userGroup := router.Group("/user")
	userHandler.RegisterUserRoutes(userGroup)

	healthHandler := &handler.HealthHandler{DB: db}
	healthGroup := router.Group("/health")
	healthHandler.RegisterHealthRoutes(healthGroup)

	return &testEnv{
		Router:    router,
		DB:        db,
		Config:    cfg,
		Blacklist: blacklist,
	}
}

// doRequest executes an HTTP request against the test router and returns the response.
func (te *testEnv) doRequest(method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBytes, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBytes)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, _ := http.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	te.Router.ServeHTTP(w, req)
	return w
}

// parseJSON parses the response body into a map.
func parseJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON response: %v, body: %s", err, w.Body.String())
	}
	return result
}

// registerUser registers a user and returns the response.
func (te *testEnv) registerUser(t *testing.T, username, password string) *httptest.ResponseRecorder {
	t.Helper()
	return te.doRequest("POST", "/user/register", map[string]string{
		"username": username,
		"password": password,
	}, "")
}

// loginUser logs in and returns access and refresh tokens.
func (te *testEnv) loginUser(t *testing.T, username, password string) (accessToken, refreshToken string) {
	t.Helper()
	w := te.doRequest("POST", "/user/login", map[string]string{
		"username": username,
		"password": password,
	}, "")
	if w.Code != http.StatusOK {
		t.Fatalf("Login failed with status %d: %s", w.Code, w.Body.String())
	}
	result := parseJSON(t, w)
	return result["access_token"].(string), result["refresh_token"].(string)
}
