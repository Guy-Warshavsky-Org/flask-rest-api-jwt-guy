package handler_test

import (
	"net/http"
	"testing"
)

func TestHealthCheck_Healthy(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupTestRouter(db)

	w := performRequest(r, http.MethodGet, "/health/", nil, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	result := parseJSON(t, w)
	if result["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got '%v'", result["status"])
	}
	if result["database"] != "healthy" {
		t.Errorf("expected database 'healthy', got '%v'", result["database"])
	}
}

func TestHealthCheck_Unhealthy(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupTestRouter(db)

	// Close the underlying SQL DB to simulate failure
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying DB: %v", err)
	}
	sqlDB.Close()

	w := performRequest(r, http.MethodGet, "/health/", nil, nil)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", w.Code, w.Body.String())
	}

	result := parseJSON(t, w)
	if result["status"] != "unhealthy" {
		t.Errorf("expected status 'unhealthy', got '%v'", result["status"])
	}
	if result["database"] != "unhealthy" {
		t.Errorf("expected database 'unhealthy', got '%v'", result["database"])
	}
}
