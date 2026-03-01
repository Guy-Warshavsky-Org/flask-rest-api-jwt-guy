package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"flask-rest-api-jwt-guy/internal/config"
	"flask-rest-api-jwt-guy/internal/middleware"
	"flask-rest-api-jwt-guy/internal/model"
	"flask-rest-api-jwt-guy/internal/router"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const testSecretKey = "test-secret-key-for-testing"

// setupTestDB creates an in-memory SQLite database for testing
// and runs auto-migration on all models.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := model.AutoMigrateAll(db); err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}

	return db
}

// setupTestRouter creates a test Gin router with all routes registered.
func setupTestRouter(db *gorm.DB) (*gin.Engine, *middleware.TokenBlacklist) {
	gin.SetMode(gin.TestMode)
	blacklist := middleware.NewTokenBlacklist()
	cfg := config.Config{
		Env:       "development",
		SecretKey: testSecretKey,
		HTTPPort:  5000,
	}
	r := router.NewRouter(db, cfg, blacklist)
	return r, blacklist
}

// performRequest is a helper to execute an HTTP request against the test router.
func performRequest(r *gin.Engine, method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	var reqBody *bytes.Reader
	if body != nil {
		jsonBytes, _ := json.Marshal(body)
		reqBody = bytes.NewReader(jsonBytes)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// parseJSON parses the response body into a map.
func parseJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON response: %v, body: %s", err, w.Body.String())
	}
	return result
}

// registerUser is a helper that registers a user and returns the response.
func registerUser(r *gin.Engine, username, password string) *httptest.ResponseRecorder {
	return performRequest(r, http.MethodPost, "/user/register", map[string]string{
		"username": username,
		"password": password,
	}, nil)
}

// loginUser is a helper that logs in a user and returns access and refresh tokens.
func loginUser(t *testing.T, r *gin.Engine, username, password string) (string, string) {
	t.Helper()
	w := performRequest(r, http.MethodPost, "/user/login", map[string]string{
		"username": username,
		"password": password,
	}, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("login failed with status %d: %s", w.Code, w.Body.String())
	}
	result := parseJSON(t, w)
	accessToken, ok := result["access_token"].(string)
	if !ok {
		t.Fatal("access_token not found in login response")
	}
	refreshToken, ok := result["refresh_token"].(string)
	if !ok {
		t.Fatal("refresh_token not found in login response")
	}
	return accessToken, refreshToken
}

// authHeader returns headers with Bearer token.
func authHeader(token string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + token,
	}
}
