package tests

import (
	"fmt"
	"net/http"
	"testing"
)

// =============================================================================
// Create Item Tests
// =============================================================================

func TestCreateItemSuccess(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")
	_, storeBody := createStore(r, "My Store", token)
	storeID := int(storeBody["id"].(float64))

	code, body := createItem(r, "Widget", 9.99, storeID, token)

	if code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", code)
	}

	if body["name"] != "Widget" {
		t.Errorf("expected name 'Widget', got '%v'", body["name"])
	}
	if body["price"] != 9.99 {
		t.Errorf("expected price 9.99, got %v", body["price"])
	}
	if int(body["store_id"].(float64)) != storeID {
		t.Errorf("expected store_id %d, got %v", storeID, body["store_id"])
	}
	if _, ok := body["id"]; !ok {
		t.Error("expected 'id' field in response")
	}
}

func TestCreateItemResponseShape(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")
	_, storeBody := createStore(r, "My Store", token)
	storeID := int(storeBody["id"].(float64))

	code, body := createItem(r, "Widget", 9.99, storeID, token)
	if code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", code)
	}

	// Should have exactly 4 keys: id, name, price, store_id (matching Flask ItemSchema)
	if len(body) != 4 {
		t.Errorf("expected 4 keys in response, got %d: %v", len(body), body)
	}
	for _, key := range []string{"id", "name", "price", "store_id"} {
		if _, ok := body[key]; !ok {
			t.Errorf("missing '%s' key in response", key)
		}
	}
}

func TestCreateItemStoreNotFound(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")

	code, _ := createItem(r, "Widget", 9.99, 9999, token)

	if code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", code)
	}
}

func TestCreateItemUnauthorizedStore(t *testing.T) {
	r, _ := setupTestRouter()

	// User1 creates a store
	token1, _ := registerAndLogin(r, "user1", "password123")
	_, storeBody := createStore(r, "User1 Store", token1)
	storeID := int(storeBody["id"].(float64))

	// User2 tries to create an item in user1's store
	token2, _ := registerAndLogin(r, "user2", "password123")
	code, body := createItem(r, "Widget", 9.99, storeID, token2)

	if code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", code)
	}
	if body["message"] != "Unauthorized" {
		t.Errorf("expected message 'Unauthorized', got '%v'", body["message"])
	}
}

func TestCreateItemMissingFields(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")
	_, storeBody := createStore(r, "My Store", token)
	storeID := int(storeBody["id"].(float64))

	// Missing name
	w := performRequest(r, "POST", "/item/", jsonBody(map[string]interface{}{
		"price":    9.99,
		"store_id": storeID,
	}), authHeader(token))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestCreateItemWithoutToken(t *testing.T) {
	r, _ := setupTestRouter()

	w := performRequest(r, "POST", "/item/", jsonBody(map[string]interface{}{
		"name":     "Widget",
		"price":    9.99,
		"store_id": 1,
	}))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

// =============================================================================
// Get All Items Tests
// =============================================================================

func TestGetAllItemsSuccess(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")
	_, storeBody := createStore(r, "My Store", token)
	storeID := int(storeBody["id"].(float64))

	createItem(r, "Widget", 9.99, storeID, token)
	createItem(r, "Gadget", 19.99, storeID, token)

	w := performRequest(r, "GET", "/item/s", nil, authHeader(token))

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := parseJSONArray(w)
	if len(body) != 2 {
		t.Errorf("expected 2 items, got %d", len(body))
	}
}

func TestGetAllItemsEmptyList(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")

	w := performRequest(r, "GET", "/item/s", nil, authHeader(token))

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := parseJSONArray(w)
	if len(body) != 0 {
		t.Errorf("expected 0 items, got %d", len(body))
	}
}

func TestGetAllItemsAcrossMultipleStores(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")
	_, store1Body := createStore(r, "Store 1", token)
	storeID1 := int(store1Body["id"].(float64))
	_, store2Body := createStore(r, "Store 2", token)
	storeID2 := int(store2Body["id"].(float64))

	createItem(r, "Widget", 9.99, storeID1, token)
	createItem(r, "Gadget", 19.99, storeID2, token)

	w := performRequest(r, "GET", "/item/s", nil, authHeader(token))

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := parseJSONArray(w)
	if len(body) != 2 {
		t.Errorf("expected 2 items across stores, got %d", len(body))
	}
}

func TestGetAllItemsOnlyOwned(t *testing.T) {
	r, _ := setupTestRouter()

	// User1 creates items
	token1, _ := registerAndLogin(r, "user1", "password123")
	_, store1Body := createStore(r, "User1 Store", token1)
	storeID1 := int(store1Body["id"].(float64))
	createItem(r, "User1 Widget", 9.99, storeID1, token1)
	createItem(r, "User1 Gadget", 19.99, storeID1, token1)

	// User2 creates items
	token2, _ := registerAndLogin(r, "user2", "password123")
	_, store2Body := createStore(r, "User2 Store", token2)
	storeID2 := int(store2Body["id"].(float64))
	createItem(r, "User2 Widget", 29.99, storeID2, token2)

	// User1 should see only their 2 items
	w := performRequest(r, "GET", "/item/s", nil, authHeader(token1))
	body := parseJSONArray(w)
	if len(body) != 2 {
		t.Errorf("expected 2 items for user1, got %d", len(body))
	}

	// User2 should see only their 1 item
	w = performRequest(r, "GET", "/item/s", nil, authHeader(token2))
	body = parseJSONArray(w)
	if len(body) != 1 {
		t.Errorf("expected 1 item for user2, got %d", len(body))
	}
}

func TestGetAllItemsWithoutToken(t *testing.T) {
	r, _ := setupTestRouter()

	w := performRequest(r, "GET", "/item/s", nil)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

// =============================================================================
// Get Item Tests
// =============================================================================

func TestGetItemSuccess(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")
	_, storeBody := createStore(r, "My Store", token)
	storeID := int(storeBody["id"].(float64))
	_, itemBody := createItem(r, "Widget", 9.99, storeID, token)
	itemID := itemBody["id"]

	w := performRequest(r, "GET", fmt.Sprintf("/item/%v", itemID), nil, authHeader(token))

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := parseJSON(w)
	if body["name"] != "Widget" {
		t.Errorf("expected name 'Widget', got '%v'", body["name"])
	}
	if body["price"] != 9.99 {
		t.Errorf("expected price 9.99, got %v", body["price"])
	}
}

func TestGetItemResponseShape(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")
	_, storeBody := createStore(r, "My Store", token)
	storeID := int(storeBody["id"].(float64))
	_, itemBody := createItem(r, "Widget", 9.99, storeID, token)
	itemID := itemBody["id"]

	w := performRequest(r, "GET", fmt.Sprintf("/item/%v", itemID), nil, authHeader(token))

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := parseJSON(w)
	// Should have exactly 4 keys: id, name, price, store_id
	if len(body) != 4 {
		t.Errorf("expected 4 keys in response, got %d: %v", len(body), body)
	}
}

func TestGetItemNotFound(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")

	w := performRequest(r, "GET", "/item/9999", nil, authHeader(token))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestGetItemUnauthorizedDifferentUser(t *testing.T) {
	r, _ := setupTestRouter()

	// User1 creates a store and item
	token1, _ := registerAndLogin(r, "user1", "password123")
	_, storeBody := createStore(r, "User1 Store", token1)
	storeID := int(storeBody["id"].(float64))
	_, itemBody := createItem(r, "Widget", 9.99, storeID, token1)
	itemID := itemBody["id"]

	// User2 tries to access the item
	token2, _ := registerAndLogin(r, "user2", "password123")
	w := performRequest(r, "GET", fmt.Sprintf("/item/%v", itemID), nil, authHeader(token2))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	body := parseJSON(w)
	if body["message"] != "Unauthorized" {
		t.Errorf("expected message 'Unauthorized', got '%v'", body["message"])
	}
}

func TestGetItemWithoutToken(t *testing.T) {
	r, _ := setupTestRouter()

	w := performRequest(r, "GET", "/item/1", nil)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

// =============================================================================
// Update Item Tests
// =============================================================================

func TestUpdateItemFullUpdate(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")
	_, storeBody := createStore(r, "My Store", token)
	storeID := int(storeBody["id"].(float64))
	_, itemBody := createItem(r, "Widget", 9.99, storeID, token)
	itemID := itemBody["id"]

	// Update all fields
	w := performRequest(r, "PUT", fmt.Sprintf("/item/%v", itemID), jsonBody(map[string]interface{}{
		"name":  "Updated Widget",
		"price": 19.99,
	}), authHeader(token))

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := parseJSON(w)
	if body["name"] != "Updated Widget" {
		t.Errorf("expected name 'Updated Widget', got '%v'", body["name"])
	}
	if body["price"] != 19.99 {
		t.Errorf("expected price 19.99, got %v", body["price"])
	}
}

func TestUpdateItemPartialNameOnly(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")
	_, storeBody := createStore(r, "My Store", token)
	storeID := int(storeBody["id"].(float64))
	_, itemBody := createItem(r, "Widget", 9.99, storeID, token)
	itemID := itemBody["id"]

	// Update only name — price should remain unchanged
	w := performRequest(r, "PUT", fmt.Sprintf("/item/%v", itemID), jsonBody(map[string]interface{}{
		"name": "New Name",
	}), authHeader(token))

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := parseJSON(w)
	if body["name"] != "New Name" {
		t.Errorf("expected name 'New Name', got '%v'", body["name"])
	}
	if body["price"] != 9.99 {
		t.Errorf("expected price to remain 9.99, got %v", body["price"])
	}
}

func TestUpdateItemPartialPriceOnly(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")
	_, storeBody := createStore(r, "My Store", token)
	storeID := int(storeBody["id"].(float64))
	_, itemBody := createItem(r, "Widget", 9.99, storeID, token)
	itemID := itemBody["id"]

	// Update only price — name should remain unchanged
	w := performRequest(r, "PUT", fmt.Sprintf("/item/%v", itemID), jsonBody(map[string]interface{}{
		"price": 29.99,
	}), authHeader(token))

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := parseJSON(w)
	if body["name"] != "Widget" {
		t.Errorf("expected name to remain 'Widget', got '%v'", body["name"])
	}
	if body["price"] != 29.99 {
		t.Errorf("expected price 29.99, got %v", body["price"])
	}
}

func TestUpdateItemChangeStoreID(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")
	_, store1Body := createStore(r, "Store 1", token)
	storeID1 := int(store1Body["id"].(float64))
	_, store2Body := createStore(r, "Store 2", token)
	storeID2 := int(store2Body["id"].(float64))

	_, itemBody := createItem(r, "Widget", 9.99, storeID1, token)
	itemID := itemBody["id"]

	// Move item to store 2
	w := performRequest(r, "PUT", fmt.Sprintf("/item/%v", itemID), jsonBody(map[string]interface{}{
		"store_id": storeID2,
	}), authHeader(token))

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := parseJSON(w)
	if int(body["store_id"].(float64)) != storeID2 {
		t.Errorf("expected store_id %d, got %v", storeID2, body["store_id"])
	}
}

func TestUpdateItemChangeToUnauthorizedStore(t *testing.T) {
	r, _ := setupTestRouter()

	// User1 creates a store and item
	token1, _ := registerAndLogin(r, "user1", "password123")
	_, store1Body := createStore(r, "User1 Store", token1)
	storeID1 := int(store1Body["id"].(float64))
	_, itemBody := createItem(r, "Widget", 9.99, storeID1, token1)
	itemID := itemBody["id"]

	// User2 creates a store
	token2, _ := registerAndLogin(r, "user2", "password123")
	_, store2Body := createStore(r, "User2 Store", token2)
	storeID2 := int(store2Body["id"].(float64))

	// User1 tries to move item to user2's store
	w := performRequest(r, "PUT", fmt.Sprintf("/item/%v", itemID), jsonBody(map[string]interface{}{
		"store_id": storeID2,
	}), authHeader(token1))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestUpdateItemNotFound(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")

	w := performRequest(r, "PUT", "/item/9999", jsonBody(map[string]interface{}{
		"name": "New Name",
	}), authHeader(token))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestUpdateItemUnauthorizedDifferentUser(t *testing.T) {
	r, _ := setupTestRouter()

	// User1 creates a store and item
	token1, _ := registerAndLogin(r, "user1", "password123")
	_, storeBody := createStore(r, "User1 Store", token1)
	storeID := int(storeBody["id"].(float64))
	_, itemBody := createItem(r, "Widget", 9.99, storeID, token1)
	itemID := itemBody["id"]

	// User2 tries to update the item
	token2, _ := registerAndLogin(r, "user2", "password123")
	w := performRequest(r, "PUT", fmt.Sprintf("/item/%v", itemID), jsonBody(map[string]interface{}{
		"name": "Hacked",
	}), authHeader(token2))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestUpdateItemWithoutToken(t *testing.T) {
	r, _ := setupTestRouter()

	w := performRequest(r, "PUT", "/item/1", jsonBody(map[string]interface{}{
		"name": "New Name",
	}))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestUpdateItemResponseShape(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")
	_, storeBody := createStore(r, "My Store", token)
	storeID := int(storeBody["id"].(float64))
	_, itemBody := createItem(r, "Widget", 9.99, storeID, token)
	itemID := itemBody["id"]

	w := performRequest(r, "PUT", fmt.Sprintf("/item/%v", itemID), jsonBody(map[string]interface{}{
		"name": "Updated",
	}), authHeader(token))

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := parseJSON(w)
	// Should have exactly 4 keys: id, name, price, store_id
	if len(body) != 4 {
		t.Errorf("expected 4 keys in response, got %d: %v", len(body), body)
	}
}

// =============================================================================
// Delete Item Tests
// =============================================================================

func TestDeleteItemSuccess(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")
	_, storeBody := createStore(r, "My Store", token)
	storeID := int(storeBody["id"].(float64))
	_, itemBody := createItem(r, "Widget", 9.99, storeID, token)
	itemID := itemBody["id"]

	w := performRequest(r, "DELETE", fmt.Sprintf("/item/%v", itemID), nil, authHeader(token))

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := parseJSON(w)
	if body["message"] != "Deleted" {
		t.Errorf("expected message 'Deleted', got '%v'", body["message"])
	}
}

func TestDeleteItemVerifyRemoved(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")
	_, storeBody := createStore(r, "My Store", token)
	storeID := int(storeBody["id"].(float64))
	_, itemBody := createItem(r, "Widget", 9.99, storeID, token)
	itemID := itemBody["id"]

	// Delete the item
	performRequest(r, "DELETE", fmt.Sprintf("/item/%v", itemID), nil, authHeader(token))

	// Verify it's gone
	w := performRequest(r, "GET", fmt.Sprintf("/item/%v", itemID), nil, authHeader(token))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404 after deletion, got %d", w.Code)
	}
}

func TestDeleteItemNotFound(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")

	w := performRequest(r, "DELETE", "/item/9999", nil, authHeader(token))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestDeleteItemUnauthorizedDifferentUser(t *testing.T) {
	r, _ := setupTestRouter()

	// User1 creates a store and item
	token1, _ := registerAndLogin(r, "user1", "password123")
	_, storeBody := createStore(r, "User1 Store", token1)
	storeID := int(storeBody["id"].(float64))
	_, itemBody := createItem(r, "Widget", 9.99, storeID, token1)
	itemID := itemBody["id"]

	// User2 tries to delete the item
	token2, _ := registerAndLogin(r, "user2", "password123")
	w := performRequest(r, "DELETE", fmt.Sprintf("/item/%v", itemID), nil, authHeader(token2))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	body := parseJSON(w)
	if body["message"] != "Unauthorized" {
		t.Errorf("expected message 'Unauthorized', got '%v'", body["message"])
	}
}

func TestDeleteItemWithoutToken(t *testing.T) {
	r, _ := setupTestRouter()

	w := performRequest(r, "DELETE", "/item/1", nil)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestDeleteItemRemovedFromList(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")
	_, storeBody := createStore(r, "My Store", token)
	storeID := int(storeBody["id"].(float64))
	_, item1Body := createItem(r, "Widget", 9.99, storeID, token)
	item1ID := item1Body["id"]
	createItem(r, "Gadget", 19.99, storeID, token)

	// Delete item 1
	performRequest(r, "DELETE", fmt.Sprintf("/item/%v", item1ID), nil, authHeader(token))

	// Verify only 1 item remains
	w := performRequest(r, "GET", "/item/s", nil, authHeader(token))
	items := parseJSONArray(w)
	if len(items) != 1 {
		t.Errorf("expected 1 item remaining, got %d", len(items))
	}
	if items[0]["name"] != "Gadget" {
		t.Errorf("expected remaining item 'Gadget', got '%v'", items[0]["name"])
	}
}

// =============================================================================
// Store Delete Cascade to Items Test
// =============================================================================

func TestStoreDeleteCascadesToItems(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")

	// Create two stores with items
	_, store1Body := createStore(r, "Store 1", token)
	storeID1 := int(store1Body["id"].(float64))
	_, store2Body := createStore(r, "Store 2", token)
	storeID2 := int(store2Body["id"].(float64))

	createItem(r, "S1 Widget", 9.99, storeID1, token)
	createItem(r, "S1 Gadget", 19.99, storeID1, token)
	createItem(r, "S2 Widget", 29.99, storeID2, token)

	// Delete store 1
	w := performRequest(r, "DELETE", fmt.Sprintf("/store/%d", storeID1), nil, authHeader(token))
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for store delete, got %d", w.Code)
	}

	// Verify only store 2's item remains
	w = performRequest(r, "GET", "/item/s", nil, authHeader(token))
	items := parseJSONArray(w)
	if len(items) != 1 {
		t.Errorf("expected 1 item remaining after cascade, got %d", len(items))
	}
	if len(items) > 0 && items[0]["name"] != "S2 Widget" {
		t.Errorf("expected remaining item 'S2 Widget', got '%v'", items[0]["name"])
	}
}

func TestItemPersistsAfterGetRetrieve(t *testing.T) {
	r, _ := setupTestRouter()

	token, _ := registerAndLogin(r, "testuser", "password123")
	_, storeBody := createStore(r, "My Store", token)
	storeID := int(storeBody["id"].(float64))

	// Create and then retrieve the item to verify persistence
	_, itemBody := createItem(r, "Widget", 9.99, storeID, token)
	itemID := itemBody["id"]

	// Get the item to verify it matches
	w := performRequest(r, "GET", fmt.Sprintf("/item/%v", itemID), nil, authHeader(token))
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := parseJSON(w)
	if body["name"] != "Widget" {
		t.Errorf("expected name 'Widget', got '%v'", body["name"])
	}
	if body["price"] != 9.99 {
		t.Errorf("expected price 9.99, got %v", body["price"])
	}
	if int(body["store_id"].(float64)) != storeID {
		t.Errorf("expected store_id %d, got %v", storeID, body["store_id"])
	}
}
