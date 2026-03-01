package handler_test

import (
	"fmt"
	"net/http"
	"testing"
)

func TestRegisterUser_Success(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupTestRouter(db)

	w := registerUser(r, "testuser", "password123")

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}

	result := parseJSON(t, w)
	if result["username"] != "testuser" {
		t.Errorf("expected username 'testuser', got '%v'", result["username"])
	}
	if _, ok := result["id"]; !ok {
		t.Error("expected 'id' field in response")
	}
	// Ensure password_hash is NOT in response (security)
	if _, ok := result["password_hash"]; ok {
		t.Error("password_hash should not be in response")
	}
}

func TestRegisterUser_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupTestRouter(db)

	// Register first time
	w := registerUser(r, "testuser", "password123")
	if w.Code != http.StatusCreated {
		t.Fatalf("first registration failed: %d", w.Code)
	}

	// Register duplicate
	w = registerUser(r, "testuser", "password456")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for duplicate, got %d", w.Code)
	}

	result := parseJSON(t, w)
	if result["message"] != "User exists" {
		t.Errorf("expected message 'User exists', got '%v'", result["message"])
	}
}

func TestLogin_Success(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupTestRouter(db)

	registerUser(r, "testuser", "password123")

	w := performRequest(r, http.MethodPost, "/user/login", map[string]string{
		"username": "testuser",
		"password": "password123",
	}, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	result := parseJSON(t, w)
	if _, ok := result["access_token"]; !ok {
		t.Error("expected 'access_token' in response")
	}
	if _, ok := result["refresh_token"]; !ok {
		t.Error("expected 'refresh_token' in response")
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupTestRouter(db)

	registerUser(r, "testuser", "password123")

	w := performRequest(r, http.MethodPost, "/user/login", map[string]string{
		"username": "testuser",
		"password": "wrongpassword",
	}, nil)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}

	result := parseJSON(t, w)
	if result["message"] != "Invalid credentials" {
		t.Errorf("expected message 'Invalid credentials', got '%v'", result["message"])
	}
}

func TestLogin_NonexistentUser(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupTestRouter(db)

	w := performRequest(r, http.MethodPost, "/user/login", map[string]string{
		"username": "nonexistent",
		"password": "password123",
	}, nil)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}

	result := parseJSON(t, w)
	if result["message"] != "Invalid credentials" {
		t.Errorf("expected message 'Invalid credentials', got '%v'", result["message"])
	}
}

func TestRefreshToken(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupTestRouter(db)

	registerUser(r, "testuser", "password123")
	_, refreshToken := loginUser(t, r, "testuser", "password123")

	w := performRequest(r, http.MethodPost, "/user/refresh", nil, authHeader(refreshToken))

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	result := parseJSON(t, w)
	if _, ok := result["access_token"]; !ok {
		t.Error("expected 'access_token' in refresh response")
	}
}

func TestRefreshToken_WithAccessToken_Rejected(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupTestRouter(db)

	registerUser(r, "testuser", "password123")
	accessToken, _ := loginUser(t, r, "testuser", "password123")

	// Try to use access token on refresh endpoint — should be rejected
	w := performRequest(r, http.MethodPost, "/user/refresh", nil, authHeader(accessToken))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLogout(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupTestRouter(db)

	registerUser(r, "testuser", "password123")
	accessToken, _ := loginUser(t, r, "testuser", "password123")

	// Logout
	w := performRequest(r, http.MethodPost, "/user/logout", nil, authHeader(accessToken))

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	result := parseJSON(t, w)
	if result["message"] != "Logged out" {
		t.Errorf("expected message 'Logged out', got '%v'", result["message"])
	}

	// Subsequent request with same token should fail (blacklisted)
	w = performRequest(r, http.MethodGet, "/user/1", nil, authHeader(accessToken))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after logout, got %d: %s", w.Code, w.Body.String())
	}

	result = parseJSON(t, w)
	if result["msg"] != "Token has been revoked" {
		t.Errorf("expected msg 'Token has been revoked', got '%v'", result["msg"])
	}
}

func TestGetUser_Authorized(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupTestRouter(db)

	w := registerUser(r, "testuser", "password123")
	result := parseJSON(t, w)
	userID := result["id"].(float64)

	accessToken, _ := loginUser(t, r, "testuser", "password123")

	w = performRequest(r, http.MethodGet, fmt.Sprintf("/user/%d", int(userID)), nil, authHeader(accessToken))

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	result = parseJSON(t, w)
	if result["username"] != "testuser" {
		t.Errorf("expected username 'testuser', got '%v'", result["username"])
	}
	// Ensure password_hash is not leaked
	if _, ok := result["password_hash"]; ok {
		t.Error("password_hash should not be in response")
	}
}

func TestGetUser_Unauthorized(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupTestRouter(db)

	// Register two users
	registerUser(r, "user1", "password123")
	registerUser(r, "user2", "password123")

	// Login as user2
	accessToken, _ := loginUser(t, r, "user2", "password123")

	// Try to get user1's profile — should be unauthorized
	w := performRequest(r, http.MethodGet, "/user/1", nil, authHeader(accessToken))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", w.Code, w.Body.String())
	}

	result := parseJSON(t, w)
	if result["message"] != "Unauthorized" {
		t.Errorf("expected message 'Unauthorized', got '%v'", result["message"])
	}
}

func TestGetUser_NotFound(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupTestRouter(db)

	registerUser(r, "testuser", "password123")
	accessToken, _ := loginUser(t, r, "testuser", "password123")

	w := performRequest(r, http.MethodGet, "/user/999", nil, authHeader(accessToken))

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetUser_NoAuth(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupTestRouter(db)

	w := performRequest(r, http.MethodGet, "/user/1", nil, nil)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestDeleteUser(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupTestRouter(db)

	w := registerUser(r, "testuser", "password123")
	result := parseJSON(t, w)
	userID := result["id"].(float64)

	accessToken, _ := loginUser(t, r, "testuser", "password123")

	w = performRequest(r, http.MethodDelete, fmt.Sprintf("/user/%d", int(userID)), nil, authHeader(accessToken))

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	result = parseJSON(t, w)
	if result["message"] != "Deleted" {
		t.Errorf("expected message 'Deleted', got '%v'", result["message"])
	}

	// Verify user is gone (login should fail)
	w = performRequest(r, http.MethodPost, "/user/login", map[string]string{
		"username": "testuser",
		"password": "password123",
	}, nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 after deletion, got %d", w.Code)
	}
}

func TestDeleteUser_Unauthorized(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupTestRouter(db)

	registerUser(r, "user1", "password123")
	registerUser(r, "user2", "password123")

	accessToken, _ := loginUser(t, r, "user2", "password123")

	// Try to delete user1 as user2
	w := performRequest(r, http.MethodDelete, "/user/1", nil, authHeader(accessToken))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", w.Code, w.Body.String())
	}

	result := parseJSON(t, w)
	if result["message"] != "Unauthorized" {
		t.Errorf("expected message 'Unauthorized', got '%v'", result["message"])
	}
}
