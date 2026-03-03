package tests

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/auth"
)

// =============================================================================
// Registration Tests
// =============================================================================

func TestRegisterSuccess(t *testing.T) {
	r, _ := setupTestRouter()

	code, body := registerUser(r, "testuser", "password123")

	if code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", code)
	}

	// Verify response matches Flask's user_schema.dump(user): {"id": N, "username": "..."}
	if body["username"] != "testuser" {
		t.Errorf("expected username 'testuser', got '%v'", body["username"])
	}
	if _, ok := body["id"]; !ok {
		t.Error("expected 'id' field in response")
	}
	// password_hash should NOT be in response
	if _, ok := body["password_hash"]; ok {
		t.Error("password_hash should not be in response")
	}
}

func TestRegisterDuplicateUsername(t *testing.T) {
	r, _ := setupTestRouter()

	// Register first user
	registerUser(r, "testuser", "password123")

	// Attempt duplicate registration
	code, body := registerUser(r, "testuser", "different_password")

	if code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", code)
	}
	if body["message"] != "User exists" {
		t.Errorf("expected message 'User exists', got '%v'", body["message"])
	}
}

func TestRegisterMissingFields(t *testing.T) {
	r, _ := setupTestRouter()

	// Missing password
	w := performRequest(r, "POST", "/user/register", jsonBody(map[string]interface{}{
		"username": "testuser",
	}))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestRegisterResponseShape(t *testing.T) {
	r, _ := setupTestRouter()

	code, body := registerUser(r, "testuser", "password123")
	if code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", code)
	}

	// Should have exactly 2 keys: id and username (matching Flask UserSchema)
	if len(body) != 2 {
		t.Errorf("expected 2 keys in response, got %d: %v", len(body), body)
	}
	if _, ok := body["id"]; !ok {
		t.Error("missing 'id' key")
	}
	if _, ok := body["username"]; !ok {
		t.Error("missing 'username' key")
	}
}

// =============================================================================
// Login Tests
// =============================================================================

func TestLoginSuccess(t *testing.T) {
	r, _ := setupTestRouter()

	registerUser(r, "testuser", "password123")
	code, body := loginUser(r, "testuser", "password123")

	if code != http.StatusOK {
		t.Errorf("expected status 200, got %d", code)
	}

	// Verify response matches Flask: {"access_token": "...", "refresh_token": "..."}
	if _, ok := body["access_token"]; !ok {
		t.Error("expected 'access_token' in response")
	}
	if _, ok := body["refresh_token"]; !ok {
		t.Error("expected 'refresh_token' in response")
	}

	// Tokens should be non-empty strings
	if body["access_token"] == "" {
		t.Error("access_token should not be empty")
	}
	if body["refresh_token"] == "" {
		t.Error("refresh_token should not be empty")
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	r, _ := setupTestRouter()

	registerUser(r, "testuser", "password123")

	// Wrong password
	code, body := loginUser(r, "testuser", "wrong_password")

	if code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", code)
	}
	if body["message"] != "Invalid credentials" {
		t.Errorf("expected message 'Invalid credentials', got '%v'", body["message"])
	}
}

func TestLoginNonexistentUser(t *testing.T) {
	r, _ := setupTestRouter()

	code, body := loginUser(r, "nobody", "password123")

	if code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", code)
	}
	if body["message"] != "Invalid credentials" {
		t.Errorf("expected message 'Invalid credentials', got '%v'", body["message"])
	}
}

func TestLoginResponseShape(t *testing.T) {
	r, _ := setupTestRouter()

	registerUser(r, "testuser", "password123")
	code, body := loginUser(r, "testuser", "password123")

	if code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", code)
	}

	// Should have exactly 2 keys matching Flask response
	if len(body) != 2 {
		t.Errorf("expected 2 keys in response, got %d: %v", len(body), body)
	}
}

// =============================================================================
// Logout Tests
// =============================================================================

func TestLogoutSuccess(t *testing.T) {
	r, _ := setupTestRouter()

	registerUser(r, "testuser", "password123")
	_, loginBody := loginUser(r, "testuser", "password123")
	accessToken := loginBody["access_token"].(string)

	// Logout
	w := performRequest(r, "POST", "/user/logout", nil, authHeader(accessToken))

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := parseJSON(w)
	if body["message"] != "Logged out" {
		t.Errorf("expected message 'Logged out', got '%v'", body["message"])
	}
}

func TestLogoutRevokesToken(t *testing.T) {
	r, _ := setupTestRouter()

	registerUser(r, "testuser", "password123")
	_, loginBody := loginUser(r, "testuser", "password123")
	accessToken := loginBody["access_token"].(string)

	// Logout (blacklists the token)
	performRequest(r, "POST", "/user/logout", nil, authHeader(accessToken))

	// Attempt to use the revoked token to access a protected endpoint
	w := performRequest(r, "GET", "/user/1", nil, authHeader(accessToken))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 after token revocation, got %d", w.Code)
	}

	body := parseJSON(w)
	if body["msg"] != "Token has been revoked" {
		t.Errorf("expected revocation message, got '%v'", body["msg"])
	}
}

func TestLogoutWithoutToken(t *testing.T) {
	r, _ := setupTestRouter()

	w := performRequest(r, "POST", "/user/logout", nil)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

// =============================================================================
// Refresh Tests
// =============================================================================

func TestRefreshSuccess(t *testing.T) {
	r, _ := setupTestRouter()

	registerUser(r, "testuser", "password123")
	_, loginBody := loginUser(r, "testuser", "password123")
	refreshToken := loginBody["refresh_token"].(string)

	// Use refresh token to get a new access token
	w := performRequest(r, "POST", "/user/refresh", nil, authHeader(refreshToken))

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := parseJSON(w)
	if _, ok := body["access_token"]; !ok {
		t.Error("expected 'access_token' in response")
	}
	if body["access_token"] == "" {
		t.Error("access_token should not be empty")
	}
}

func TestRefreshWithAccessTokenRejected(t *testing.T) {
	r, _ := setupTestRouter()

	registerUser(r, "testuser", "password123")
	_, loginBody := loginUser(r, "testuser", "password123")
	accessToken := loginBody["access_token"].(string)

	// Attempt to use access token for refresh (should be rejected)
	w := performRequest(r, "POST", "/user/refresh", nil, authHeader(accessToken))

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422, got %d", w.Code)
	}
}

func TestRefreshNewTokenWorks(t *testing.T) {
	r, _ := setupTestRouter()

	// Register and login
	_, regBody := registerUser(r, "testuser", "password123")
	userID := regBody["id"]

	_, loginBody := loginUser(r, "testuser", "password123")
	refreshToken := loginBody["refresh_token"].(string)

	// Get new access token
	w := performRequest(r, "POST", "/user/refresh", nil, authHeader(refreshToken))
	body := parseJSON(w)
	newAccessToken := body["access_token"].(string)

	// Use the new access token to access a protected endpoint
	w = performRequest(r, "GET", fmt.Sprintf("/user/%v", userID), nil, authHeader(newAccessToken))

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 with refreshed token, got %d", w.Code)
	}
}

// =============================================================================
// Get User Tests
// =============================================================================

func TestGetUserSuccess(t *testing.T) {
	r, _ := setupTestRouter()

	_, regBody := registerUser(r, "testuser", "password123")
	userID := regBody["id"]

	_, loginBody := loginUser(r, "testuser", "password123")
	accessToken := loginBody["access_token"].(string)

	w := performRequest(r, "GET", fmt.Sprintf("/user/%v", userID), nil, authHeader(accessToken))

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := parseJSON(w)
	if body["username"] != "testuser" {
		t.Errorf("expected username 'testuser', got '%v'", body["username"])
	}
	if body["id"] != userID {
		t.Errorf("expected id %v, got %v", userID, body["id"])
	}
	// password_hash should NOT be exposed
	if _, ok := body["password_hash"]; ok {
		t.Error("password_hash should not be in response")
	}
}

func TestGetUserUnauthorizedDifferentUser(t *testing.T) {
	r, _ := setupTestRouter()

	// Register two users
	registerUser(r, "user1", "password123")
	_, regBody2 := registerUser(r, "user2", "password123")
	user2ID := regBody2["id"]

	// Login as user1
	_, loginBody := loginUser(r, "user1", "password123")
	accessToken := loginBody["access_token"].(string)

	// Attempt to get user2's data with user1's token
	w := performRequest(r, "GET", fmt.Sprintf("/user/%v", user2ID), nil, authHeader(accessToken))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	body := parseJSON(w)
	if body["message"] != "Unauthorized" {
		t.Errorf("expected message 'Unauthorized', got '%v'", body["message"])
	}
}

func TestGetUserNotFound(t *testing.T) {
	r, _ := setupTestRouter()

	registerUser(r, "testuser", "password123")
	_, loginBody := loginUser(r, "testuser", "password123")
	accessToken := loginBody["access_token"].(string)

	// Request non-existent user
	w := performRequest(r, "GET", "/user/9999", nil, authHeader(accessToken))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestGetUserWithoutToken(t *testing.T) {
	r, _ := setupTestRouter()

	w := performRequest(r, "GET", "/user/1", nil)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestGetUserResponseShape(t *testing.T) {
	r, _ := setupTestRouter()

	_, regBody := registerUser(r, "testuser", "password123")
	userID := regBody["id"]

	_, loginBody := loginUser(r, "testuser", "password123")
	accessToken := loginBody["access_token"].(string)

	w := performRequest(r, "GET", fmt.Sprintf("/user/%v", userID), nil, authHeader(accessToken))

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := parseJSON(w)
	// Should have exactly 2 keys: id and username (matching Flask UserSchema)
	if len(body) != 2 {
		t.Errorf("expected 2 keys in response, got %d: %v", len(body), body)
	}
}

// =============================================================================
// Delete User Tests
// =============================================================================

func TestDeleteUserSuccess(t *testing.T) {
	r, _ := setupTestRouter()

	_, regBody := registerUser(r, "testuser", "password123")
	userID := regBody["id"]

	_, loginBody := loginUser(r, "testuser", "password123")
	accessToken := loginBody["access_token"].(string)

	w := performRequest(r, "DELETE", fmt.Sprintf("/user/%v", userID), nil, authHeader(accessToken))

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := parseJSON(w)
	if body["message"] != "Deleted" {
		t.Errorf("expected message 'Deleted', got '%v'", body["message"])
	}
}

func TestDeleteUserVerifyRemoved(t *testing.T) {
	r, _ := setupTestRouter()

	_, regBody := registerUser(r, "testuser", "password123")
	userID := regBody["id"]

	_, loginBody := loginUser(r, "testuser", "password123")
	accessToken := loginBody["access_token"].(string)

	// Delete the user
	performRequest(r, "DELETE", fmt.Sprintf("/user/%v", userID), nil, authHeader(accessToken))

	// Attempt to get the deleted user (should be not found)
	w := performRequest(r, "GET", fmt.Sprintf("/user/%v", userID), nil, authHeader(accessToken))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404 after deletion, got %d", w.Code)
	}
}

func TestDeleteUserUnauthorizedDifferentUser(t *testing.T) {
	r, _ := setupTestRouter()

	// Register two users
	registerUser(r, "user1", "password123")
	_, regBody2 := registerUser(r, "user2", "password123")
	user2ID := regBody2["id"]

	// Login as user1
	_, loginBody := loginUser(r, "user1", "password123")
	accessToken := loginBody["access_token"].(string)

	// Attempt to delete user2 with user1's token
	w := performRequest(r, "DELETE", fmt.Sprintf("/user/%v", user2ID), nil, authHeader(accessToken))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	body := parseJSON(w)
	if body["message"] != "Unauthorized" {
		t.Errorf("expected message 'Unauthorized', got '%v'", body["message"])
	}
}

func TestDeleteUserNotFound(t *testing.T) {
	r, _ := setupTestRouter()

	registerUser(r, "testuser", "password123")
	_, loginBody := loginUser(r, "testuser", "password123")
	accessToken := loginBody["access_token"].(string)

	w := performRequest(r, "DELETE", "/user/9999", nil, authHeader(accessToken))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestDeleteUserWithoutToken(t *testing.T) {
	r, _ := setupTestRouter()

	w := performRequest(r, "DELETE", "/user/1", nil)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

// =============================================================================
// Authentication Flow (End-to-End) Tests
// =============================================================================

func TestFullAuthFlow(t *testing.T) {
	r, _ := setupTestRouter()

	// Step 1: Register
	regCode, regBody := registerUser(r, "flowuser", "secret123")
	if regCode != http.StatusCreated {
		t.Fatalf("register failed: %d", regCode)
	}
	userID := regBody["id"]

	// Step 2: Login
	loginCode, loginBody := loginUser(r, "flowuser", "secret123")
	if loginCode != http.StatusOK {
		t.Fatalf("login failed: %d", loginCode)
	}
	accessToken := loginBody["access_token"].(string)
	refreshToken := loginBody["refresh_token"].(string)

	// Step 3: Access protected endpoint with access token
	w := performRequest(r, "GET", fmt.Sprintf("/user/%v", userID), nil, authHeader(accessToken))
	if w.Code != http.StatusOK {
		t.Errorf("expected protected endpoint access to succeed, got %d", w.Code)
	}

	// Step 4: Refresh to get a new access token
	w = performRequest(r, "POST", "/user/refresh", nil, authHeader(refreshToken))
	if w.Code != http.StatusOK {
		t.Errorf("expected refresh to succeed, got %d", w.Code)
	}
	newAccessToken := parseJSON(w)["access_token"].(string)

	// Step 5: Use the new access token
	w = performRequest(r, "GET", fmt.Sprintf("/user/%v", userID), nil, authHeader(newAccessToken))
	if w.Code != http.StatusOK {
		t.Errorf("expected new access token to work, got %d", w.Code)
	}

	// Step 6: Logout (revoke the new access token)
	w = performRequest(r, "POST", "/user/logout", nil, authHeader(newAccessToken))
	if w.Code != http.StatusOK {
		t.Errorf("expected logout to succeed, got %d", w.Code)
	}

	// Step 7: Verify revoked token is rejected
	w = performRequest(r, "GET", fmt.Sprintf("/user/%v", userID), nil, authHeader(newAccessToken))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected revoked token to be rejected, got %d", w.Code)
	}

	// Step 8: Original access token should still work (only new one was revoked)
	w = performRequest(r, "GET", fmt.Sprintf("/user/%v", userID), nil, authHeader(accessToken))
	if w.Code != http.StatusOK {
		t.Errorf("expected original access token to still work, got %d", w.Code)
	}
}

func TestInvalidTokenRejected(t *testing.T) {
	r, _ := setupTestRouter()

	// Use a completely invalid token
	w := performRequest(r, "GET", "/user/1", nil, authHeader("not-a-valid-jwt"))

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422, got %d", w.Code)
	}
}

func TestExpiredTokenRejected(t *testing.T) {
	r, _ := setupTestRouter()

	cfg := testConfig()

	// Generate a token that expired 1 hour ago
	token, err := auth.GenerateAccessToken("1", cfg.JWTSecretKey, -1*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	w := performRequest(r, "GET", "/user/1", nil, authHeader(token))

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422 for expired token, got %d", w.Code)
	}
}

func TestMissingAuthorizationHeader(t *testing.T) {
	r, _ := setupTestRouter()

	w := performRequest(r, "GET", "/user/1", nil)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	body := parseJSON(w)
	if body["msg"] != "Missing Authorization Header" {
		t.Errorf("expected missing auth header message, got '%v'", body["msg"])
	}
}

func TestBadAuthorizationFormat(t *testing.T) {
	r, _ := setupTestRouter()

	w := performRequest(r, "GET", "/user/1", nil, map[string]string{
		"Authorization": "Basic some-token",
	})

	// Should reject non-Bearer auth
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422, got %d", w.Code)
	}
}
