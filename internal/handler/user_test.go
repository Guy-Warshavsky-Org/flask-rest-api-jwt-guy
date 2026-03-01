package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/modelcode-ai/flask-rest-api-jwt-guy/internal/config"
	"github.com/modelcode-ai/flask-rest-api-jwt-guy/internal/middleware"
	"github.com/modelcode-ai/flask-rest-api-jwt-guy/internal/model"
	"github.com/modelcode-ai/flask-rest-api-jwt-guy/internal/router"
)

// testConfig returns a test configuration.
func testConfig() *config.Config {
	return &config.Config{
		AppEnv:           "test",
		Port:             "8080",
		DatabaseURL:      ":memory:",
		SecretKey:        "test-secret-key",
		JWTAccessExpiry:  15 * time.Minute,
		JWTRefreshExpiry: 720 * time.Hour,
		Debug:            false,
	}
}

// setupTestDB creates a fresh in-memory SQLite DB and runs migrations.
// Each call returns an independent database.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Using a unique DSN per test via file::memory: ensures isolation.
	// The connection_pool=1 prevents reuse issues.
	dsn := fmt.Sprintf("file:testdb_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "Failed to open test database")

	err = model.AutoMigrateAll(db)
	require.NoError(t, err, "Failed to run migrations")

	// Enable foreign keys for SQLite (required for cascade deletes).
	db.Exec("PRAGMA foreign_keys = ON")

	return db
}

// setupTestRouter creates a full Gin engine configured for testing.
// Each test gets its own isolated DB and clean blacklist.
func setupTestRouter(t *testing.T) (*gin.Engine, *gorm.DB, *config.Config) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	middleware.ResetBlacklist()

	cfg := testConfig()
	db := setupTestDB(t)
	r := router.Setup(db, cfg)

	return r, db, cfg
}

// doRequest performs an HTTP request against the test router and returns the response.
func doRequest(r *gin.Engine, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBytes, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBytes)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// parseJSON parses the response body into a map.
func parseJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err, "Failed to parse response JSON: %s", w.Body.String())
	return result
}

// getUserID extracts the user ID from a response body as an integer string.
func getUserID(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	body := parseJSON(t, w)
	// JSON numbers come as float64.
	id := body["id"].(float64)
	return fmt.Sprintf("%d", int(id))
}

// registerUser is a helper to register a user and return the response.
func registerUser(r *gin.Engine, username, password string) *httptest.ResponseRecorder {
	return doRequest(r, "POST", "/user/register", map[string]string{
		"username": username,
		"password": password,
	}, "")
}

// loginUser is a helper to login and return the response.
func loginUser(r *gin.Engine, username, password string) *httptest.ResponseRecorder {
	return doRequest(r, "POST", "/user/login", map[string]string{
		"username": username,
		"password": password,
	}, "")
}

// registerAndLogin is a helper that registers a user and returns access and refresh tokens.
func registerAndLogin(t *testing.T, r *gin.Engine, username, password string) (userID string, accessToken string, refreshToken string) {
	t.Helper()
	regResp := registerUser(r, username, password)
	require.Equal(t, http.StatusCreated, regResp.Code, "Registration failed")
	userID = getUserID(t, regResp)

	loginResp := loginUser(r, username, password)
	require.Equal(t, http.StatusOK, loginResp.Code, "Login failed")
	tokens := parseJSON(t, loginResp)
	accessToken = tokens["access_token"].(string)
	refreshToken = tokens["refresh_token"].(string)
	return
}

// ============================================================
// Registration Tests
// ============================================================

func TestRegister_Success(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	w := registerUser(r, "testuser", "testpass")

	assert.Equal(t, http.StatusCreated, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, "testuser", body["username"])
	assert.NotNil(t, body["id"])
	// Ensure password_hash is NOT in response (json:"-" on model).
	assert.Nil(t, body["password_hash"])
}

func TestRegister_DuplicateUser(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	// First registration should succeed.
	w1 := registerUser(r, "testuser", "testpass")
	assert.Equal(t, http.StatusCreated, w1.Code)

	// Second registration with same username should fail with 400.
	w2 := registerUser(r, "testuser", "testpass")
	assert.Equal(t, http.StatusBadRequest, w2.Code)
	body := parseJSON(t, w2)
	assert.Equal(t, "User exists", body["message"])
}

func TestRegister_InvalidPayload(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	// Missing password field.
	w := doRequest(r, "POST", "/user/register", map[string]string{
		"username": "testuser",
	}, "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================
// Login Tests
// ============================================================

func TestLogin_Success(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	registerUser(r, "testuser", "testpass")

	w := loginUser(r, "testuser", "testpass")
	assert.Equal(t, http.StatusOK, w.Code)

	body := parseJSON(t, w)
	assert.NotEmpty(t, body["access_token"])
	assert.NotEmpty(t, body["refresh_token"])
}

func TestLogin_WrongPassword(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	registerUser(r, "testuser", "testpass")

	w := loginUser(r, "testuser", "wrongpass")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, "Invalid credentials", body["message"])
}

func TestLogin_NonExistentUser(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	w := loginUser(r, "nonexistent", "testpass")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, "Invalid credentials", body["message"])
}

// ============================================================
// Logout Tests
// ============================================================

func TestLogout_Success(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")

	w := doRequest(r, "POST", "/user/logout", nil, accessToken)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, "Logged out", body["message"])
}

func TestLogout_BlacklistedToken(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	userID, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")

	// Logout to blacklist the token.
	doRequest(r, "POST", "/user/logout", nil, accessToken)

	// Try to use the blacklisted token — should be rejected.
	w := doRequest(r, "GET", "/user/"+userID, nil, accessToken)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, "Token has been revoked", body["message"])
}

func TestLogout_NoToken(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	w := doRequest(r, "POST", "/user/logout", nil, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================
// Token Refresh Tests
// ============================================================

func TestRefresh_Success(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, _, refreshToken := registerAndLogin(t, r, "testuser", "testpass")

	w := doRequest(r, "POST", "/user/refresh", nil, refreshToken)
	assert.Equal(t, http.StatusOK, w.Code)

	body := parseJSON(t, w)
	assert.NotEmpty(t, body["access_token"])
}

func TestRefresh_WithAccessToken(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")

	// Attempting to refresh with an access token should fail.
	w := doRequest(r, "POST", "/user/refresh", nil, accessToken)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================
// Get User Tests
// ============================================================

func TestGetUser_Success(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	userID, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")

	w := doRequest(r, "GET", "/user/"+userID, nil, accessToken)
	assert.Equal(t, http.StatusOK, w.Code)

	body := parseJSON(t, w)
	assert.Equal(t, "testuser", body["username"])
	assert.NotNil(t, body["id"])
	// Ensure password_hash is NOT in response.
	assert.Nil(t, body["password_hash"])
}

func TestGetUser_Unauthorized(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	// Register two users.
	user1ID, _, _ := registerAndLogin(t, r, "user1", "pass1")
	_, user2AccessToken, _ := registerAndLogin(t, r, "user2", "pass2")

	// user2 tries to get user1 — should be unauthorized.
	w := doRequest(r, "GET", "/user/"+user1ID, nil, user2AccessToken)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, "Unauthorized", body["message"])
}

func TestGetUser_NotFound(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")

	w := doRequest(r, "GET", "/user/999", nil, accessToken)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetUser_NoToken(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	w := doRequest(r, "GET", "/user/1", nil, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================
// Delete User Tests
// ============================================================

func TestDeleteUser_Success(t *testing.T) {
	r, db, _ := setupTestRouter(t)

	userID, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")

	w := doRequest(r, "DELETE", "/user/"+userID, nil, accessToken)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, "Deleted", body["message"])

	// Verify user is gone from DB.
	var count int64
	db.Model(&model.User{}).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestDeleteUser_Unauthorized(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	// Register two users.
	user1ID, _, _ := registerAndLogin(t, r, "user1", "pass1")
	_, user2AccessToken, _ := registerAndLogin(t, r, "user2", "pass2")

	// user2 tries to delete user1 — should be unauthorized.
	w := doRequest(r, "DELETE", "/user/"+user1ID, nil, user2AccessToken)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, "Unauthorized", body["message"])
}

func TestDeleteUser_NotFound(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")

	w := doRequest(r, "DELETE", "/user/999", nil, accessToken)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ============================================================
// Cascade Delete Tests
// ============================================================

func TestDeleteUser_CascadeDeletesStores(t *testing.T) {
	r, db, _ := setupTestRouter(t)

	userID, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")

	// Get user ID as int for DB operations.
	var userIDInt int
	fmt.Sscanf(userID, "%d", &userIDInt)

	// Create a store directly in DB.
	store := model.Store{Name: "Test Store", UserID: userIDInt}
	require.NoError(t, db.Create(&store).Error, "Failed to create store")

	// Create an item and tag under the store.
	item := model.Item{Name: "Test Item", Price: 9.99, StoreID: store.ID}
	tag := model.Tag{Name: "Test Tag", StoreID: store.ID}
	require.NoError(t, db.Create(&item).Error, "Failed to create item")
	require.NoError(t, db.Create(&tag).Error, "Failed to create tag")

	// Verify they exist.
	var storeCount, itemCount, tagCount int64
	db.Model(&model.Store{}).Count(&storeCount)
	db.Model(&model.Item{}).Count(&itemCount)
	db.Model(&model.Tag{}).Count(&tagCount)
	assert.Equal(t, int64(1), storeCount)
	assert.Equal(t, int64(1), itemCount)
	assert.Equal(t, int64(1), tagCount)

	// Delete user — should cascade delete store, item, and tag.
	w := doRequest(r, "DELETE", "/user/"+userID, nil, accessToken)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify all related records are gone.
	db.Model(&model.Store{}).Count(&storeCount)
	db.Model(&model.Item{}).Count(&itemCount)
	db.Model(&model.Tag{}).Count(&tagCount)
	assert.Equal(t, int64(0), storeCount, "Stores should be cascade deleted")
	assert.Equal(t, int64(0), itemCount, "Items should be cascade deleted")
	assert.Equal(t, int64(0), tagCount, "Tags should be cascade deleted")
}

// ============================================================
// Full Auth Flow Test
// ============================================================

func TestFullAuthFlow(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	// 1. Register.
	w := registerUser(r, "flowuser", "flowpass")
	assert.Equal(t, http.StatusCreated, w.Code)
	userID := getUserID(t, w)

	// 2. Login.
	w = loginUser(r, "flowuser", "flowpass")
	assert.Equal(t, http.StatusOK, w.Code)
	tokens := parseJSON(t, w)
	accessToken := tokens["access_token"].(string)
	refreshToken := tokens["refresh_token"].(string)

	// 3. Get user (with access token).
	w = doRequest(r, "GET", "/user/"+userID, nil, accessToken)
	assert.Equal(t, http.StatusOK, w.Code)

	// 4. Refresh token to get new access token.
	w = doRequest(r, "POST", "/user/refresh", nil, refreshToken)
	assert.Equal(t, http.StatusOK, w.Code)
	newTokens := parseJSON(t, w)
	newAccessToken := newTokens["access_token"].(string)

	// 5. Use new access token.
	w = doRequest(r, "GET", "/user/"+userID, nil, newAccessToken)
	assert.Equal(t, http.StatusOK, w.Code)

	// 6. Logout (blacklist new access token).
	w = doRequest(r, "POST", "/user/logout", nil, newAccessToken)
	assert.Equal(t, http.StatusOK, w.Code)

	// 7. Verify blacklisted token is rejected.
	w = doRequest(r, "GET", "/user/"+userID, nil, newAccessToken)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 8. Original access token should still work (only the new one was blacklisted).
	w = doRequest(r, "GET", "/user/"+userID, nil, accessToken)
	assert.Equal(t, http.StatusOK, w.Code)
}
