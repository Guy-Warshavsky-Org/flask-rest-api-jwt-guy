package handler_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/flask-rest-api-jwt-guy/internal/middleware"
	"github.com/flask-rest-api-jwt-guy/internal/model"
)

// ======================================================
// Registration Tests
// ======================================================

func TestRegister_Success(t *testing.T) {
	te := setupTestEnv(t)
	w := te.registerUser(t, "alice", "password123")

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	result := parseJSON(t, w)
	if result["username"] != "alice" {
		t.Errorf("Expected username 'alice', got %v", result["username"])
	}
	if result["id"] == nil {
		t.Error("Expected id to be present")
	}
	// Ensure password_hash is not in response
	if _, ok := result["password_hash"]; ok {
		t.Error("password_hash should not be in response")
	}
}

func TestRegister_DuplicateUser(t *testing.T) {
	te := setupTestEnv(t)
	te.registerUser(t, "alice", "password123")

	// Try to register same username again
	w := te.registerUser(t, "alice", "different")
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for duplicate user, got %d", w.Code)
	}

	result := parseJSON(t, w)
	if result["message"] != "User exists" {
		t.Errorf("Expected 'User exists' message, got %v", result["message"])
	}
}

func TestRegister_MissingFields(t *testing.T) {
	te := setupTestEnv(t)

	// Missing password
	w := te.doRequest("POST", "/user/register", map[string]string{
		"username": "alice",
	}, "")
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	// Missing username
	w = te.doRequest("POST", "/user/register", map[string]string{
		"password": "pass",
	}, "")
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// ======================================================
// Login Tests
// ======================================================

func TestLogin_Success(t *testing.T) {
	te := setupTestEnv(t)
	te.registerUser(t, "alice", "password123")

	w := te.doRequest("POST", "/user/login", map[string]string{
		"username": "alice",
		"password": "password123",
	}, "")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	result := parseJSON(t, w)
	if result["access_token"] == nil {
		t.Error("Expected access_token in response")
	}
	if result["refresh_token"] == nil {
		t.Error("Expected refresh_token in response")
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	te := setupTestEnv(t)
	te.registerUser(t, "alice", "password123")

	// Wrong password
	w := te.doRequest("POST", "/user/login", map[string]string{
		"username": "alice",
		"password": "wrongpassword",
	}, "")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	result := parseJSON(t, w)
	if result["message"] != "Invalid credentials" {
		t.Errorf("Expected 'Invalid credentials', got %v", result["message"])
	}
}

func TestLogin_NonexistentUser(t *testing.T) {
	te := setupTestEnv(t)

	w := te.doRequest("POST", "/user/login", map[string]string{
		"username": "nonexistent",
		"password": "password",
	}, "")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

// ======================================================
// Logout Tests
// ======================================================

func TestLogout_Success(t *testing.T) {
	te := setupTestEnv(t)
	te.registerUser(t, "alice", "password123")
	accessToken, _ := te.loginUser(t, "alice", "password123")

	w := te.doRequest("POST", "/user/logout", nil, accessToken)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	result := parseJSON(t, w)
	if result["message"] != "Logged out" {
		t.Errorf("Expected 'Logged out', got %v", result["message"])
	}

	// Verify token is now blacklisted - subsequent use should fail
	w = te.doRequest("POST", "/user/logout", nil, accessToken)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for revoked token, got %d", w.Code)
	}
}

func TestLogout_NoToken(t *testing.T) {
	te := setupTestEnv(t)

	w := te.doRequest("POST", "/user/logout", nil, "")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

// ======================================================
// Refresh Tests
// ======================================================

func TestRefresh_Success(t *testing.T) {
	te := setupTestEnv(t)
	te.registerUser(t, "alice", "password123")
	_, refreshToken := te.loginUser(t, "alice", "password123")

	w := te.doRequest("POST", "/user/refresh", nil, refreshToken)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	result := parseJSON(t, w)
	if result["access_token"] == nil {
		t.Error("Expected access_token in response")
	}
}

func TestRefresh_WithAccessToken_Fails(t *testing.T) {
	te := setupTestEnv(t)
	te.registerUser(t, "alice", "password123")
	accessToken, _ := te.loginUser(t, "alice", "password123")

	// Using access token for refresh should fail
	w := te.doRequest("POST", "/user/refresh", nil, accessToken)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 when using access token for refresh, got %d", w.Code)
	}
}

// ======================================================
// Get User Tests
// ======================================================

func TestGetUser_Success(t *testing.T) {
	te := setupTestEnv(t)
	w := te.registerUser(t, "alice", "password123")
	regResult := parseJSON(t, w)
	userID := regResult["id"]

	accessToken, _ := te.loginUser(t, "alice", "password123")

	w = te.doRequest("GET", fmt.Sprintf("/user/%v", userID), nil, accessToken)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	result := parseJSON(t, w)
	if result["username"] != "alice" {
		t.Errorf("Expected username 'alice', got %v", result["username"])
	}
	// Ensure no password_hash
	if _, ok := result["password_hash"]; ok {
		t.Error("password_hash should not be in response")
	}
}

func TestGetUser_Unauthorized_DifferentUser(t *testing.T) {
	te := setupTestEnv(t)
	te.registerUser(t, "alice", "password123")
	w := te.registerUser(t, "bob", "password456")
	bobResult := parseJSON(t, w)
	bobID := bobResult["id"]

	// Login as alice
	aliceToken, _ := te.loginUser(t, "alice", "password123")

	// Try to get bob's profile as alice
	w = te.doRequest("GET", fmt.Sprintf("/user/%v", bobID), nil, aliceToken)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	result := parseJSON(t, w)
	if result["message"] != "Unauthorized" {
		t.Errorf("Expected 'Unauthorized', got %v", result["message"])
	}
}

func TestGetUser_NotFound(t *testing.T) {
	te := setupTestEnv(t)
	te.registerUser(t, "alice", "password123")
	accessToken, _ := te.loginUser(t, "alice", "password123")

	w := te.doRequest("GET", "/user/9999", nil, accessToken)
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestGetUser_NoAuth(t *testing.T) {
	te := setupTestEnv(t)

	w := te.doRequest("GET", "/user/1", nil, "")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

// ======================================================
// Delete User Tests
// ======================================================

func TestDeleteUser_Success(t *testing.T) {
	te := setupTestEnv(t)
	w := te.registerUser(t, "alice", "password123")
	regResult := parseJSON(t, w)
	userID := regResult["id"]

	accessToken, _ := te.loginUser(t, "alice", "password123")

	w = te.doRequest("DELETE", fmt.Sprintf("/user/%v", userID), nil, accessToken)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	result := parseJSON(t, w)
	if result["message"] != "Deleted" {
		t.Errorf("Expected 'Deleted', got %v", result["message"])
	}

	// Verify user is gone
	var user model.User
	err := te.DB.First(&user, userID).Error
	if err == nil {
		t.Error("User should have been deleted")
	}
}

func TestDeleteUser_Unauthorized_DifferentUser(t *testing.T) {
	te := setupTestEnv(t)
	te.registerUser(t, "alice", "password123")
	w := te.registerUser(t, "bob", "password456")
	bobResult := parseJSON(t, w)
	bobID := bobResult["id"]

	aliceToken, _ := te.loginUser(t, "alice", "password123")

	// Try to delete bob as alice
	w = te.doRequest("DELETE", fmt.Sprintf("/user/%v", bobID), nil, aliceToken)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

// ======================================================
// Cascade Delete Tests
// ======================================================

func TestDeleteUser_CascadeDeletesStoresItemsTags(t *testing.T) {
	te := setupTestEnv(t)

	// Register and login
	w := te.registerUser(t, "alice", "password123")
	regResult := parseJSON(t, w)
	userID := uint(regResult["id"].(float64))
	accessToken, _ := te.loginUser(t, "alice", "password123")

	// Create a store directly in DB
	store := model.Store{Name: "Alice's Store", UserID: userID}
	te.DB.Create(&store)

	// Create items and tags in the store
	item := model.Item{Name: "Widget", Price: 9.99, StoreID: store.ID}
	te.DB.Create(&item)
	tag := model.Tag{Name: "sale", StoreID: store.ID}
	te.DB.Create(&tag)

	// Verify they exist
	var storeCount, itemCount, tagCount int64
	te.DB.Model(&model.Store{}).Count(&storeCount)
	te.DB.Model(&model.Item{}).Count(&itemCount)
	te.DB.Model(&model.Tag{}).Count(&tagCount)

	if storeCount != 1 || itemCount != 1 || tagCount != 1 {
		t.Fatalf("Expected 1 store, 1 item, 1 tag before delete; got %d, %d, %d", storeCount, itemCount, tagCount)
	}

	// Delete user
	w = te.doRequest("DELETE", fmt.Sprintf("/user/%d", userID), nil, accessToken)
	if w.Code != http.StatusOK {
		t.Fatalf("Delete user failed with status %d: %s", w.Code, w.Body.String())
	}

	// Verify cascade: all related records should be gone
	te.DB.Model(&model.Store{}).Count(&storeCount)
	te.DB.Model(&model.Item{}).Count(&itemCount)
	te.DB.Model(&model.Tag{}).Count(&tagCount)

	if storeCount != 0 {
		t.Errorf("Expected 0 stores after cascade delete, got %d", storeCount)
	}
	if itemCount != 0 {
		t.Errorf("Expected 0 items after cascade delete, got %d", itemCount)
	}
	if tagCount != 0 {
		t.Errorf("Expected 0 tags after cascade delete, got %d", tagCount)
	}
}

// ======================================================
// JWT Full Lifecycle Test
// ======================================================

func TestJWT_FullLifecycle(t *testing.T) {
	te := setupTestEnv(t)

	// 1. Register
	w := te.registerUser(t, "alice", "password123")
	if w.Code != http.StatusCreated {
		t.Fatalf("Registration failed: %d", w.Code)
	}
	regResult := parseJSON(t, w)
	userID := regResult["id"]

	// 2. Login
	accessToken, refreshToken := te.loginUser(t, "alice", "password123")

	// 3. Use access token to get user profile
	w = te.doRequest("GET", fmt.Sprintf("/user/%v", userID), nil, accessToken)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 with valid token, got %d", w.Code)
	}

	// 4. Refresh to get new access token
	w = te.doRequest("POST", "/user/refresh", nil, refreshToken)
	if w.Code != http.StatusOK {
		t.Fatalf("Refresh failed: %d", w.Code)
	}
	newResult := parseJSON(t, w)
	newAccessToken := newResult["access_token"].(string)

	// 5. Use new access token
	w = te.doRequest("GET", fmt.Sprintf("/user/%v", userID), nil, newAccessToken)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 with refreshed token, got %d", w.Code)
	}

	// 6. Logout (revoke access token)
	w = te.doRequest("POST", "/user/logout", nil, newAccessToken)
	if w.Code != http.StatusOK {
		t.Errorf("Logout failed: %d", w.Code)
	}

	// 7. Verify revoked token is rejected
	w = te.doRequest("GET", fmt.Sprintf("/user/%v", userID), nil, newAccessToken)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 with revoked token, got %d", w.Code)
	}
}

// ======================================================
// Expired Token Test
// ======================================================

func TestExpiredAccessToken(t *testing.T) {
	te := setupTestEnv(t)
	te.registerUser(t, "alice", "password123")

	// Generate a token with very short TTL
	te.Config.AccessTokenTTL = 1 * time.Millisecond
	accessToken, _ := middleware.GenerateAccessToken(1, te.Config)
	te.Config.AccessTokenTTL = 15 * time.Minute // restore

	// Wait for it to expire
	time.Sleep(10 * time.Millisecond)

	w := te.doRequest("GET", "/user/1", nil, accessToken)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for expired token, got %d", w.Code)
	}
}
