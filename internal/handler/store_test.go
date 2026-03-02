package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/modelcode-ai/flask-rest-api-jwt-guy/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// Create Store Tests
// ============================================================

func TestCreateStore_Success(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")

	w := doRequest(r, "POST", "/store/", map[string]string{
		"name": "My Store",
	}, accessToken)

	assert.Equal(t, http.StatusCreated, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, "My Store", body["name"])
	assert.NotNil(t, body["id"])
	assert.NotNil(t, body["user_id"])
}

func TestCreateStore_InvalidPayload(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")

	// Missing name field.
	w := doRequest(r, "POST", "/store/", map[string]string{}, accessToken)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateStore_NoToken(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	w := doRequest(r, "POST", "/store/", map[string]string{
		"name": "My Store",
	}, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================
// Get Store Tests
// ============================================================

func TestGetStore_Success(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")

	// Create a store.
	createResp := doRequest(r, "POST", "/store/", map[string]string{
		"name": "My Store",
	}, accessToken)
	require.Equal(t, http.StatusCreated, createResp.Code)
	created := parseJSON(t, createResp)
	storeID := fmt.Sprintf("%d", int(created["id"].(float64)))

	// Get the store.
	w := doRequest(r, "GET", "/store/"+storeID, nil, accessToken)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, "My Store", body["name"])
	assert.Equal(t, created["id"], body["id"])
	assert.NotNil(t, body["user_id"])
}

func TestGetStore_Unauthorized(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	// User1 creates a store.
	_, user1Token, _ := registerAndLogin(t, r, "user1", "pass1")
	createResp := doRequest(r, "POST", "/store/", map[string]string{
		"name": "User1 Store",
	}, user1Token)
	require.Equal(t, http.StatusCreated, createResp.Code)
	created := parseJSON(t, createResp)
	storeID := fmt.Sprintf("%d", int(created["id"].(float64)))

	// User2 tries to get user1's store — should be unauthorized.
	_, user2Token, _ := registerAndLogin(t, r, "user2", "pass2")
	w := doRequest(r, "GET", "/store/"+storeID, nil, user2Token)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, "Unauthorized", body["message"])
}

func TestGetStore_NotFound(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")

	w := doRequest(r, "GET", "/store/999", nil, accessToken)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetStore_NoToken(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	w := doRequest(r, "GET", "/store/1", nil, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================
// Get All Stores Tests
// ============================================================

func TestGetAllStores_Success(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")

	// Create two stores.
	doRequest(r, "POST", "/store/", map[string]string{"name": "Store A"}, accessToken)
	doRequest(r, "POST", "/store/", map[string]string{"name": "Store B"}, accessToken)

	// Get all stores.
	w := doRequest(r, "GET", "/store/s", nil, accessToken)
	assert.Equal(t, http.StatusOK, w.Code)

	var stores []map[string]interface{}
	err := parseJSONArray(t, w, &stores)
	require.NoError(t, err)
	assert.Len(t, stores, 2)
}

func TestGetAllStores_OnlyOwned(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	// User1 creates a store.
	_, user1Token, _ := registerAndLogin(t, r, "user1", "pass1")
	doRequest(r, "POST", "/store/", map[string]string{"name": "User1 Store"}, user1Token)

	// User2 creates a store.
	_, user2Token, _ := registerAndLogin(t, r, "user2", "pass2")
	doRequest(r, "POST", "/store/", map[string]string{"name": "User2 Store"}, user2Token)

	// User1 should only see their own store.
	w := doRequest(r, "GET", "/store/s", nil, user1Token)
	assert.Equal(t, http.StatusOK, w.Code)

	var stores []map[string]interface{}
	err := parseJSONArray(t, w, &stores)
	require.NoError(t, err)
	assert.Len(t, stores, 1)
	assert.Equal(t, "User1 Store", stores[0]["name"])
}

func TestGetAllStores_EmptyList(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")

	w := doRequest(r, "GET", "/store/s", nil, accessToken)
	assert.Equal(t, http.StatusOK, w.Code)

	var stores []map[string]interface{}
	err := parseJSONArray(t, w, &stores)
	require.NoError(t, err)
	assert.Len(t, stores, 0)
}

func TestGetAllStores_NoToken(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	w := doRequest(r, "GET", "/store/s", nil, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================
// Delete Store Tests
// ============================================================

func TestDeleteStore_Success(t *testing.T) {
	r, db, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")

	// Create a store.
	createResp := doRequest(r, "POST", "/store/", map[string]string{
		"name": "My Store",
	}, accessToken)
	require.Equal(t, http.StatusCreated, createResp.Code)
	created := parseJSON(t, createResp)
	storeID := fmt.Sprintf("%d", int(created["id"].(float64)))

	// Delete the store.
	w := doRequest(r, "DELETE", "/store/"+storeID, nil, accessToken)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, "Deleted", body["message"])

	// Verify store is gone from DB.
	var count int64
	db.Model(&model.Store{}).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestDeleteStore_Unauthorized(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	// User1 creates a store.
	_, user1Token, _ := registerAndLogin(t, r, "user1", "pass1")
	createResp := doRequest(r, "POST", "/store/", map[string]string{
		"name": "User1 Store",
	}, user1Token)
	require.Equal(t, http.StatusCreated, createResp.Code)
	created := parseJSON(t, createResp)
	storeID := fmt.Sprintf("%d", int(created["id"].(float64)))

	// User2 tries to delete user1's store — should be unauthorized.
	_, user2Token, _ := registerAndLogin(t, r, "user2", "pass2")
	w := doRequest(r, "DELETE", "/store/"+storeID, nil, user2Token)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, "Unauthorized", body["message"])
}

func TestDeleteStore_NotFound(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")

	w := doRequest(r, "DELETE", "/store/999", nil, accessToken)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteStore_NoToken(t *testing.T) {
	r, _, _ := setupTestRouter(t)

	w := doRequest(r, "DELETE", "/store/1", nil, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================
// Store Cascade Delete Tests
// ============================================================

func TestDeleteStore_CascadeDeletesItemsAndTags(t *testing.T) {
	r, db, _ := setupTestRouter(t)

	_, accessToken, _ := registerAndLogin(t, r, "testuser", "testpass")

	// Create a store via API.
	createResp := doRequest(r, "POST", "/store/", map[string]string{
		"name": "My Store",
	}, accessToken)
	require.Equal(t, http.StatusCreated, createResp.Code)
	storeData := parseJSON(t, createResp)
	storeID := int(storeData["id"].(float64))
	storeIDStr := fmt.Sprintf("%d", storeID)

	// Create items and tags directly in DB under the store.
	item1 := model.Item{Name: "Item 1", Price: 9.99, StoreID: storeID}
	item2 := model.Item{Name: "Item 2", Price: 19.99, StoreID: storeID}
	tag1 := model.Tag{Name: "Tag 1", StoreID: storeID}
	require.NoError(t, db.Create(&item1).Error)
	require.NoError(t, db.Create(&item2).Error)
	require.NoError(t, db.Create(&tag1).Error)

	// Verify they exist.
	var itemCount, tagCount int64
	db.Model(&model.Item{}).Count(&itemCount)
	db.Model(&model.Tag{}).Count(&tagCount)
	assert.Equal(t, int64(2), itemCount)
	assert.Equal(t, int64(1), tagCount)

	// Delete the store.
	w := doRequest(r, "DELETE", "/store/"+storeIDStr, nil, accessToken)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify all related records are gone.
	db.Model(&model.Store{}).Count(&itemCount)
	assert.Equal(t, int64(0), itemCount, "Store should be deleted")

	db.Model(&model.Item{}).Count(&itemCount)
	assert.Equal(t, int64(0), itemCount, "Items should be cascade deleted")

	db.Model(&model.Tag{}).Count(&tagCount)
	assert.Equal(t, int64(0), tagCount, "Tags should be cascade deleted")
}
