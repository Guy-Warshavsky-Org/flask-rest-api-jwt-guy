package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthCheckOK(t *testing.T) {
	r, _ := setupTestRouter()

	req, err := http.NewRequest(http.MethodGet, "/health/", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Verify status code
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Parse response body
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}

	// Verify JSON structure and values
	if body["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got '%s'", body["status"])
	}
	if body["database"] != "healthy" {
		t.Errorf("expected database 'healthy', got '%s'", body["database"])
	}
}

func TestHealthCheckResponseKeys(t *testing.T) {
	r, _ := setupTestRouter()

	req, err := http.NewRequest(http.MethodGet, "/health/", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}

	// Verify exact keys present (matching Flask response shape)
	expectedKeys := []string{"status", "database"}
	for _, key := range expectedKeys {
		if _, ok := body[key]; !ok {
			t.Errorf("expected key '%s' in response, but it was missing", key)
		}
	}

	// Verify no extra keys
	if len(body) != len(expectedKeys) {
		t.Errorf("expected %d keys in response, got %d", len(expectedKeys), len(body))
	}
}

func TestHealthCheckContentType(t *testing.T) {
	r, _ := setupTestRouter()

	req, err := http.NewRequest(http.MethodGet, "/health/", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	expected := "application/json; charset=utf-8"
	if contentType != expected {
		t.Errorf("expected Content-Type '%s', got '%s'", expected, contentType)
	}
}
