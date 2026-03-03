package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/auth"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/config"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/db"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/router"
	"gorm.io/gorm"
)

// testConfig returns a Config suitable for integration tests.
func testConfig() *config.Config {
	return &config.Config{
		AppEnv:            "test",
		Port:              "5000",
		DatabaseURL:       "",
		JWTSecretKey:      "test-secret-key",
		JWTAccessExpires:  15 * time.Minute,
		JWTRefreshExpires: 30 * 24 * time.Hour,
	}
}

// setupTestRouter creates a Gin engine backed by an in-memory SQLite database
// for integration testing. It returns the engine and the GORM DB instance.
// Each call creates a fresh database and resets the token blacklist to ensure
// test isolation.
func setupTestRouter() (*gin.Engine, *gorm.DB) {
	gin.SetMode(gin.TestMode)

	cfg := testConfig()

	database, err := db.InitTestDB()
	if err != nil {
		panic("failed to initialize test database: " + err.Error())
	}

	// Reset the global blacklist to avoid cross-test pollution
	auth.TokenBlacklist.Reset()

	r := router.SetupRouter(cfg)
	return r, database
}

// jsonBody creates a JSON request body from a map.
func jsonBody(data map[string]interface{}) *bytes.Buffer {
	body, _ := json.Marshal(data)
	return bytes.NewBuffer(body)
}

// performRequest executes an HTTP request against the router and returns the recorder.
func performRequest(r *gin.Engine, method, path string, body *bytes.Buffer, headers ...map[string]string) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		req, _ = http.NewRequest(method, path, body)
	} else {
		req, _ = http.NewRequest(method, path, nil)
	}
	req.Header.Set("Content-Type", "application/json")

	// Apply optional headers
	for _, h := range headers {
		for k, v := range h {
			req.Header.Set(k, v)
		}
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// parseJSON parses the response body into a map.
func parseJSON(w *httptest.ResponseRecorder) map[string]interface{} {
	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	return body
}

// authHeader creates an Authorization header map with a Bearer token.
func authHeader(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

// registerUser is a helper that registers a user and returns the response body.
func registerUser(r *gin.Engine, username, password string) (int, map[string]interface{}) {
	w := performRequest(r, "POST", "/user/register", jsonBody(map[string]interface{}{
		"username": username,
		"password": password,
	}))
	return w.Code, parseJSON(w)
}

// loginUser is a helper that logs in a user and returns the tokens.
func loginUser(r *gin.Engine, username, password string) (int, map[string]interface{}) {
	w := performRequest(r, "POST", "/user/login", jsonBody(map[string]interface{}{
		"username": username,
		"password": password,
	}))
	return w.Code, parseJSON(w)
}
