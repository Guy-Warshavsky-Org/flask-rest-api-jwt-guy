package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/modelcode-ai/flask-rest-api-jwt-guy/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// Item Test Helpers
// ============================================================

// createStoreHelper creates a store via API and returns its ID as int and string.
func createStoreHelper(t *testing.T, r *gin.Engine, name, token string) (int, string) {
	t.Helper()
	w := doRequest(r, "POST", "/store/", map[string]string{"name": name}, token)
	require.Equal(t, http.StatusCreated, w.Code, "Failed to create store: %s", w.Body.String())
	body := parseJSON(t, w)
	id := int(body["id"].(float64))
	return id, fmt.Sprintf("%d", id)
}

// createItemHelper creates an item via API and returns its response body.
func createItemHelper(t *testing.T, r *gin.Engine, name string, price float64, storeID int, token string) map[string]interface{} {
	t.Helper()
	w := doRequest(r, "POST", "/item/", map[string]interface{}{
		"name":     name,
		"price":    price,
		"store_id": storeID,
	}, token)
	require.Equal(t, http.StatusCreated, w.Code, "Failed to create item: %s", w.Body.String())
	return parseJSON(t, w)
}

// ============================================================
// Create Item Tests
// ============================================================

func TestCreateItem_Success(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")
	storeID, _ := createStoreHelper(t, r, "My Store", accessToken)

	w := doRequest(r, "POST", "/item/", map[string]interface{}{
		"name":     "Widget",
		"price":    9.99,
		"store_id": storeID,
	}, accessToken)

	assert.Equal(t, http.StatusCreated, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, "Widget", body["name"])
	assert.Equal(t, 9.99, body["price"])
	assert.Equal(t, float64(storeID), body["store_id"])
	assert.NotNil(t, body["id"])
}

func TestCreateItem_UnauthorizedStore(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	// User1 creates a store.
	_, user1Token, _ := registerAndLogin(t, r, "user1", "pass1")
	storeID, _ := createStoreHelper(t, r, "User1 Store", user1Token)

	// User2 tries to create an item in user1's store — should be unauthorized.
	_, user2Token, _ := registerAndLogin(t, r, "user2", "pass2")
	w := doRequest(r, "POST", "/item/", map[string]interface{}{
		"name":     "Widget",
		"price":    9.99,
		"store_id": storeID,
	}, user2Token)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, "Unauthorized", body["message"])
}

func TestCreateItem_StoreNotFound(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")

	w := doRequest(r, "POST", "/item/", map[string]interface{}{
		"name":     "Widget",
		"price":    9.99,
		"store_id": 999,
	}, accessToken)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreateItem_InvalidPayload(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")

	// Missing required fields.
	w := doRequest(r, "POST", "/item/", map[string]string{
		"name": "Widget",
	}, accessToken)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateItem_NoToken(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	w := doRequest(r, "POST", "/item/", map[string]interface{}{
		"name":     "Widget",
		"price":    9.99,
		"store_id": 1,
	}, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================
// Get All Items Tests
// ============================================================

func TestGetAllItems_Success(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")
	storeID, _ := createStoreHelper(t, r, "My Store", accessToken)

	// Create two items.
	createItemHelper(t, r, "Item A", 5.00, storeID, accessToken)
	createItemHelper(t, r, "Item B", 10.00, storeID, accessToken)

	w := doRequest(r, "GET", "/item/s", nil, accessToken)
	assert.Equal(t, http.StatusOK, w.Code)

	var items []map[string]interface{}
	err := parseJSONArray(t, w, &items)
	require.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestGetAllItems_AcrossMultipleStores(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")
	storeID1, _ := createStoreHelper(t, r, "Store A", accessToken)
	storeID2, _ := createStoreHelper(t, r, "Store B", accessToken)

	// Create items in different stores.
	createItemHelper(t, r, "Item A", 5.00, storeID1, accessToken)
	createItemHelper(t, r, "Item B", 10.00, storeID2, accessToken)

	w := doRequest(r, "GET", "/item/s", nil, accessToken)
	assert.Equal(t, http.StatusOK, w.Code)

	var items []map[string]interface{}
	err := parseJSONArray(t, w, &items)
	require.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestGetAllItems_OnlyOwnedItems(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	// User1 creates a store and item.
	_, user1Token, _ := registerAndLogin(t, r, "user1", "pass1")
	storeID1, _ := createStoreHelper(t, r, "User1 Store", user1Token)
	createItemHelper(t, r, "User1 Item", 5.00, storeID1, user1Token)

	// User2 creates a store and item.
	_, user2Token, _ := registerAndLogin(t, r, "user2", "pass2")
	storeID2, _ := createStoreHelper(t, r, "User2 Store", user2Token)
	createItemHelper(t, r, "User2 Item", 10.00, storeID2, user2Token)

	// User1 should only see their own items.
	w := doRequest(r, "GET", "/item/s", nil, user1Token)
	assert.Equal(t, http.StatusOK, w.Code)

	var items []map[string]interface{}
	err := parseJSONArray(t, w, &items)
	require.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "User1 Item", items[0]["name"])
}

func TestGetAllItems_EmptyList(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")

	w := doRequest(r, "GET", "/item/s", nil, accessToken)
	assert.Equal(t, http.StatusOK, w.Code)

	var items []map[string]interface{}
	err := parseJSONArray(t, w, &items)
	require.NoError(t, err)
	assert.Len(t, items, 0)
}

func TestGetAllItems_NoToken(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	w := doRequest(r, "GET", "/item/s", nil, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================
// Get Item Tests
// ============================================================

func TestGetItem_Success(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")
	storeID, _ := createStoreHelper(t, r, "My Store", accessToken)
	itemData := createItemHelper(t, r, "Widget", 9.99, storeID, accessToken)
	itemID := fmt.Sprintf("%d", int(itemData["id"].(float64)))

	w := doRequest(r, "GET", "/item/"+itemID, nil, accessToken)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, "Widget", body["name"])
	assert.Equal(t, 9.99, body["price"])
	assert.Equal(t, float64(storeID), body["store_id"])
}

func TestGetItem_Unauthorized(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	// User1 creates a store and item.
	_, user1Token, _ := registerAndLogin(t, r, "user1", "pass1")
	storeID, _ := createStoreHelper(t, r, "User1 Store", user1Token)
	itemData := createItemHelper(t, r, "Widget", 9.99, storeID, user1Token)
	itemID := fmt.Sprintf("%d", int(itemData["id"].(float64)))

	// User2 tries to get user1's item — should be unauthorized.
	_, user2Token, _ := registerAndLogin(t, r, "user2", "pass2")
	w := doRequest(r, "GET", "/item/"+itemID, nil, user2Token)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, "Unauthorized", body["message"])
}

func TestGetItem_NotFound(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")

	w := doRequest(r, "GET", "/item/999", nil, accessToken)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetItem_NoToken(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	w := doRequest(r, "GET", "/item/1", nil, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================
// Update Item Tests
// ============================================================

func TestUpdateItem_Success(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")
	storeID, _ := createStoreHelper(t, r, "My Store", accessToken)
	itemData := createItemHelper(t, r, "Widget", 9.99, storeID, accessToken)
	itemID := fmt.Sprintf("%d", int(itemData["id"].(float64)))

	// Update name and price.
	w := doRequest(r, "PUT", "/item/"+itemID, map[string]interface{}{
		"name":  "Updated Widget",
		"price": 19.99,
	}, accessToken)

	assert.Equal(t, http.StatusOK, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, "Updated Widget", body["name"])
	assert.Equal(t, 19.99, body["price"])
	assert.Equal(t, float64(storeID), body["store_id"])
}

func TestUpdateItem_PartialUpdate_NameOnly(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")
	storeID, _ := createStoreHelper(t, r, "My Store", accessToken)
	itemData := createItemHelper(t, r, "Widget", 9.99, storeID, accessToken)
	itemID := fmt.Sprintf("%d", int(itemData["id"].(float64)))

	// Update only name — price should remain unchanged.
	w := doRequest(r, "PUT", "/item/"+itemID, map[string]interface{}{
		"name": "New Name",
	}, accessToken)

	assert.Equal(t, http.StatusOK, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, "New Name", body["name"])
	assert.Equal(t, 9.99, body["price"], "Price should remain unchanged")
}

func TestUpdateItem_PartialUpdate_PriceOnly(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")
	storeID, _ := createStoreHelper(t, r, "My Store", accessToken)
	itemData := createItemHelper(t, r, "Widget", 9.99, storeID, accessToken)
	itemID := fmt.Sprintf("%d", int(itemData["id"].(float64)))

	// Update only price — name should remain unchanged.
	w := doRequest(r, "PUT", "/item/"+itemID, map[string]interface{}{
		"price": 29.99,
	}, accessToken)

	assert.Equal(t, http.StatusOK, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, "Widget", body["name"], "Name should remain unchanged")
	assert.Equal(t, 29.99, body["price"])
}

func TestUpdateItem_CrossStoreTransfer_Success(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")
	storeID1, _ := createStoreHelper(t, r, "Store A", accessToken)
	storeID2, _ := createStoreHelper(t, r, "Store B", accessToken)
	itemData := createItemHelper(t, r, "Widget", 9.99, storeID1, accessToken)
	itemID := fmt.Sprintf("%d", int(itemData["id"].(float64)))

	// Transfer item from Store A to Store B.
	w := doRequest(r, "PUT", "/item/"+itemID, map[string]interface{}{
		"store_id": storeID2,
	}, accessToken)

	assert.Equal(t, http.StatusOK, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, float64(storeID2), body["store_id"], "Item should be in Store B now")
	assert.Equal(t, "Widget", body["name"], "Name should remain unchanged")
	assert.Equal(t, 9.99, body["price"], "Price should remain unchanged")
}

func TestUpdateItem_CrossStoreTransfer_Unauthorized(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	// User1 creates two stores; user2 creates a store.
	_, user1Token, _ := registerAndLogin(t, r, "user1", "pass1")
	storeID1, _ := createStoreHelper(t, r, "User1 Store", user1Token)
	itemData := createItemHelper(t, r, "Widget", 9.99, storeID1, user1Token)
	itemID := fmt.Sprintf("%d", int(itemData["id"].(float64)))

	_, user2Token, _ := registerAndLogin(t, r, "user2", "pass2")
	storeID2, _ := createStoreHelper(t, r, "User2 Store", user2Token)

	// User1 tries to transfer item to user2's store — should be unauthorized.
	w := doRequest(r, "PUT", "/item/"+itemID, map[string]interface{}{
		"store_id": storeID2,
	}, user1Token)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, "Unauthorized", body["message"])
}

func TestUpdateItem_Unauthorized(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	// User1 creates a store and item.
	_, user1Token, _ := registerAndLogin(t, r, "user1", "pass1")
	storeID, _ := createStoreHelper(t, r, "User1 Store", user1Token)
	itemData := createItemHelper(t, r, "Widget", 9.99, storeID, user1Token)
	itemID := fmt.Sprintf("%d", int(itemData["id"].(float64)))

	// User2 tries to update user1's item — should be unauthorized.
	_, user2Token, _ := registerAndLogin(t, r, "user2", "pass2")
	w := doRequest(r, "PUT", "/item/"+itemID, map[string]interface{}{
		"name": "Hacked Widget",
	}, user2Token)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, "Unauthorized", body["message"])
}

func TestUpdateItem_NotFound(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")

	w := doRequest(r, "PUT", "/item/999", map[string]interface{}{
		"name": "Updated",
	}, accessToken)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateItem_NoToken(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	w := doRequest(r, "PUT", "/item/1", map[string]interface{}{
		"name": "Updated",
	}, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================
// Delete Item Tests
// ============================================================

func TestDeleteItem_Success(t *testing.T) {
	r, db, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")
	storeID, _ := createStoreHelper(t, r, "My Store", accessToken)
	itemData := createItemHelper(t, r, "Widget", 9.99, storeID, accessToken)
	itemID := fmt.Sprintf("%d", int(itemData["id"].(float64)))

	w := doRequest(r, "DELETE", "/item/"+itemID, nil, accessToken)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, "Deleted", body["message"])

	// Verify item is gone from DB.
	var count int64
	db.Model(&model.Item{}).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestDeleteItem_Unauthorized(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	// User1 creates a store and item.
	_, user1Token, _ := registerAndLogin(t, r, "user1", "pass1")
	storeID, _ := createStoreHelper(t, r, "User1 Store", user1Token)
	itemData := createItemHelper(t, r, "Widget", 9.99, storeID, user1Token)
	itemID := fmt.Sprintf("%d", int(itemData["id"].(float64)))

	// User2 tries to delete user1's item — should be unauthorized.
	_, user2Token, _ := registerAndLogin(t, r, "user2", "pass2")
	w := doRequest(r, "DELETE", "/item/"+itemID, nil, user2Token)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, "Unauthorized", body["message"])
}

func TestDeleteItem_NotFound(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")

	w := doRequest(r, "DELETE", "/item/999", nil, accessToken)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteItem_NoToken(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	w := doRequest(r, "DELETE", "/item/1", nil, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================
// Item Response Shape Test
// ============================================================

func TestItemResponse_MatchesFlaskSchema(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")
	storeID, _ := createStoreHelper(t, r, "My Store", accessToken)

	// Create an item and verify the response shape matches Flask's ItemSchema.
	w := doRequest(r, "POST", "/item/", map[string]interface{}{
		"name":     "Test Item",
		"price":    15.50,
		"store_id": storeID,
	}, accessToken)

	assert.Equal(t, http.StatusCreated, w.Code)
	body := parseJSON(t, w)

	// Verify all expected fields exist.
	assert.Contains(t, body, "id")
	assert.Contains(t, body, "name")
	assert.Contains(t, body, "price")
	assert.Contains(t, body, "store_id")

	// Verify no extra fields (Flask's ItemSchema includes id, name, price, store_id).
	assert.Len(t, body, 4, "Response should have exactly 4 fields matching Flask ItemSchema")

	// Verify types.
	assert.IsType(t, float64(0), body["id"])   // JSON number
	assert.IsType(t, "", body["name"])          // string
	assert.IsType(t, float64(0), body["price"]) // JSON number (float)
	assert.IsType(t, float64(0), body["store_id"]) // JSON number
}
