package handler_test

import (
	"net/http"
	"testing"
)

func TestHealthCheck_Success(t *testing.T) {
	te := setupTestEnv(t)

	w := te.doRequest("GET", "/health/", nil, "")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	result := parseJSON(t, w)
	if result["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", result["status"])
	}
	if result["database"] != "healthy" {
		t.Errorf("Expected database 'healthy', got %v", result["database"])
	}
}

func TestHealthCheck_NoAuth_Required(t *testing.T) {
	te := setupTestEnv(t)

	// Health endpoint should be accessible without authentication
	w := te.doRequest("GET", "/health/", nil, "")
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 without auth, got %d", w.Code)
	}
}
