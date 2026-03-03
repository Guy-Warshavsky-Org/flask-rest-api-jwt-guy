package tests

import (
	"fmt"
	"net/http"
	"testing"
)

// =============================================================================
// Create Store Tests
// =============================================================================

func TestCreateStoreSuccess(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")

	code, body := createStore(r, "My Store", token)

	if code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", code)
	}

	// Verify response matches Flask's store_schema.dump(store)
	if body["name"] != "My Store" {
		t.Errorf("expected name 'My Store', got '%v'", body["name"])
	}
	if _, ok := body["id"]; !ok {
		t.Error("expected 'id' field in response")
	}
	if _, ok := body["user_id"]; !ok {
		t.Error("expected 'user_id' field in response")
	}
}

func TestCreateStoreResponseShape(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")

	code, body := createStore(r, "My Store", token)
	if code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", code)
	}

	// Should have exactly 3 keys: id, name, user_id (matching Flask StoreSchema)
	if len(body) != 3 {
		t.Errorf("expected 3 keys in response, got %d: %v", len(body), body)
	}
	for _, key := range []string{"id", "name", "user_id"} {
		if _, ok := body[key]; !ok {
			t.Errorf("missing '%s' key in response", key)
		}
	}
}

func TestCreateStoreAssignsCurrentUser(t *testing.T) {
	r, _ := setupTestRouter()

	token, userID := registerAndLogin(r, "testuser", "password123")

	_, body := createStore(r, "My Store", token)

	// Verify the store is assigned to the authenticated user
	if body["user_id"] != userID {
		t.Errorf("expected user_id %v, got %v", userID, body["user_id"])
	}
}

func TestCreateStoreMissingFields(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")

	// Missing name field
	w := performRequest(r, "POST", "/store/", jsonBody(map[string]interface{}{}), authHeader(token))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestCreateStoreWithoutToken(t *testing.T) {
	r, _ := setupTestRouter()

	w := performRequest(r, "POST", "/store/", jsonBody(map[string]interface{}{
		"name": "My Store",
	}))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

// =============================================================================
// Get Store Tests
// =============================================================================

func TestGetStoreSuccess(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")
	_, storeBody := createStore(r, "My Store", token)
	storeID := storeBody["id"]

	w := performRequest(r, "GET", fmt.Sprintf("/store/%v", storeID), nil, authHeader(token))

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := parseJSON(w)
	if body["name"] != "My Store" {
		t.Errorf("expected name 'My Store', got '%v'", body["name"])
	}
	if body["id"] != storeID {
		t.Errorf("expected id %v, got %v", storeID, body["id"])
	}
}

func TestGetStoreResponseShape(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")
	_, storeBody := createStore(r, "My Store", token)
	storeID := storeBody["id"]

	w := performRequest(r, "GET", fmt.Sprintf("/store/%v", storeID), nil, authHeader(token))

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := parseJSON(w)
	// Should have exactly 3 keys: id, name, user_id
	if len(body) != 3 {
		t.Errorf("expected 3 keys in response, got %d: %v", len(body), body)
	}
}

func TestGetStoreNotFound(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")

	w := performRequest(r, "GET", "/store/9999", nil, authHeader(token))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestGetStoreUnauthorizedDifferentUser(t *testing.T) {
	r, _ := setupTestRouter()

	// User1 creates a store
	token1, _ := registerAndLogin(r, "user1", "password123")
	_, storeBody := createStore(r, "User1 Store", token1)
	storeID := storeBody["id"]

	// User2 tries to access it
	token2, _ := registerAndLogin(r, "user2", "password123")
	w := performRequest(r, "GET", fmt.Sprintf("/store/%v", storeID), nil, authHeader(token2))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	body := parseJSON(w)
	if body["message"] != "Unauthorized" {
		t.Errorf("expected message 'Unauthorized', got '%v'", body["message"])
	}
}

func TestGetStoreWithoutToken(t *testing.T) {
	r, _ := setupTestRouter()

	w := performRequest(r, "GET", "/store/1", nil)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

// =============================================================================
// Get All Stores Tests
// =============================================================================

func TestGetAllStoresSuccess(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")
	createStore(r, "Store 1", token)
	createStore(r, "Store 2", token)

	w := performRequest(r, "GET", "/store/s", nil, authHeader(token))

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := parseJSONArray(w)
	if len(body) != 2 {
		t.Errorf("expected 2 stores, got %d", len(body))
	}
}

func TestGetAllStoresEmptyList(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")

	w := performRequest(r, "GET", "/store/s", nil, authHeader(token))

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := parseJSONArray(w)
	if len(body) != 0 {
		t.Errorf("expected 0 stores, got %d", len(body))
	}
}

func TestGetAllStoresOnlyOwned(t *testing.T) {
	r, _ := setupTestRouter()

	// User1 creates 2 stores
	token1, _ := registerAndLogin(r, "user1", "password123")
	createStore(r, "User1 Store 1", token1)
	createStore(r, "User1 Store 2", token1)

	// User2 creates 1 store
	token2, _ := registerAndLogin(r, "user2", "password123")
	createStore(r, "User2 Store", token2)

	// User1 should see only their 2 stores
	w := performRequest(r, "GET", "/store/s", nil, authHeader(token1))
	body := parseJSONArray(w)
	if len(body) != 2 {
		t.Errorf("expected 2 stores for user1, got %d", len(body))
	}

	// User2 should see only their 1 store
	w = performRequest(r, "GET", "/store/s", nil, authHeader(token2))
	body = parseJSONArray(w)
	if len(body) != 1 {
		t.Errorf("expected 1 store for user2, got %d", len(body))
	}
}

func TestGetAllStoresWithoutToken(t *testing.T) {
	r, _ := setupTestRouter()

	w := performRequest(r, "GET", "/store/s", nil)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

// =============================================================================
// Delete Store Tests
// =============================================================================

func TestDeleteStoreSuccess(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")
	_, storeBody := createStore(r, "My Store", token)
	storeID := storeBody["id"]

	w := performRequest(r, "DELETE", fmt.Sprintf("/store/%v", storeID), nil, authHeader(token))

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := parseJSON(w)
	if body["message"] != "Deleted" {
		t.Errorf("expected message 'Deleted', got '%v'", body["message"])
	}
}

func TestDeleteStoreVerifyRemoved(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")
	_, storeBody := createStore(r, "My Store", token)
	storeID := storeBody["id"]

	// Delete the store
	performRequest(r, "DELETE", fmt.Sprintf("/store/%v", storeID), nil, authHeader(token))

	// Verify it's gone
	w := performRequest(r, "GET", fmt.Sprintf("/store/%v", storeID), nil, authHeader(token))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404 after deletion, got %d", w.Code)
	}
}

func TestDeleteStoreNotFound(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")

	w := performRequest(r, "DELETE", "/store/9999", nil, authHeader(token))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestDeleteStoreUnauthorizedDifferentUser(t *testing.T) {
	r, _ := setupTestRouter()

	// User1 creates a store
	token1, _ := registerAndLogin(r, "user1", "password123")
	_, storeBody := createStore(r, "User1 Store", token1)
	storeID := storeBody["id"]

	// User2 tries to delete it
	token2, _ := registerAndLogin(r, "user2", "password123")
	w := performRequest(r, "DELETE", fmt.Sprintf("/store/%v", storeID), nil, authHeader(token2))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	body := parseJSON(w)
	if body["message"] != "Unauthorized" {
		t.Errorf("expected message 'Unauthorized', got '%v'", body["message"])
	}
}

func TestDeleteStoreWithoutToken(t *testing.T) {
	r, _ := setupTestRouter()

	w := performRequest(r, "DELETE", "/store/1", nil)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestDeleteStoreCascadesItems(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")

	// Create store with items
	_, storeBody := createStore(r, "My Store", token)
	storeID := int(storeBody["id"].(float64))
	createItem(r, "Item 1", 9.99, storeID, token)
	createItem(r, "Item 2", 19.99, storeID, token)

	// Verify items exist
	w := performRequest(r, "GET", "/item/s", nil, authHeader(token))
	items := parseJSONArray(w)
	if len(items) != 2 {
		t.Fatalf("expected 2 items before delete, got %d", len(items))
	}

	// Delete the store
	w = performRequest(r, "DELETE", fmt.Sprintf("/store/%d", storeID), nil, authHeader(token))
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for delete, got %d", w.Code)
	}

	// Verify items are gone (cascade delete)
	w = performRequest(r, "GET", "/item/s", nil, authHeader(token))
	items = parseJSONArray(w)
	if len(items) != 0 {
		t.Errorf("expected 0 items after cascade delete, got %d", len(items))
	}
}

func TestDeleteStoreRemovedFromList(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")
	_, store1Body := createStore(r, "Store 1", token)
	store1ID := store1Body["id"]
	createStore(r, "Store 2", token)

	// Delete store 1
	performRequest(r, "DELETE", fmt.Sprintf("/store/%v", store1ID), nil, authHeader(token))

	// Verify only 1 store remains
	w := performRequest(r, "GET", "/store/s", nil, authHeader(token))
	stores := parseJSONArray(w)
	if len(stores) != 1 {
		t.Errorf("expected 1 store remaining, got %d", len(stores))
	}
	if stores[0]["name"] != "Store 2" {
		t.Errorf("expected remaining store 'Store 2', got '%v'", stores[0]["name"])
	}
}
