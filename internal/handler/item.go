package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/config"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/db"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/middleware"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/model"
)

// createItemRequest represents the JSON body for POST /item/.
type createItemRequest struct {
	Name    string  `json:"name" binding:"required"`
	Price   float64 `json:"price" binding:"required"`
	StoreID int     `json:"store_id" binding:"required"`
}

// itemResponse mirrors Flask's ItemSchema with include_fk=True.
// Fields: id, name, price, store_id — matching the Marshmallow auto-schema output.
type itemResponse struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Price   float64 `json:"price"`
	StoreID int     `json:"store_id"`
}

// toItemResponse converts an Item model to the API response DTO.
func toItemResponse(i *model.Item) itemResponse {
	return itemResponse{
		ID:      i.ID,
		Name:    i.Name,
		Price:   i.Price,
		StoreID: i.StoreID,
	}
}

// toItemResponses converts a slice of Item models to response DTOs.
func toItemResponses(items []model.Item) []itemResponse {
	result := make([]itemResponse, len(items))
	for i, item := range items {
		result[i] = toItemResponse(&item)
	}
	return result
}

// RegisterItemRoutes registers all item-related endpoints on the given router group.
// Routes mirror the Flask items_bp Blueprint with url_prefix='/item'.
func RegisterItemRoutes(rg *gin.RouterGroup, cfg *config.Config) {
	item := rg.Group("/item")
	{
		// All item routes require access token (mirrors @jwt_required())
		item.POST("/", middleware.RequireAccessToken(cfg.JWTSecretKey), handleCreateItem)
		item.GET("/s", middleware.RequireAccessToken(cfg.JWTSecretKey), handleGetAllItems)
		item.GET("/:id", middleware.RequireAccessToken(cfg.JWTSecretKey), handleGetItem)
		item.PUT("/:id", middleware.RequireAccessToken(cfg.JWTSecretKey), handleUpdateItem)
		item.DELETE("/:id", middleware.RequireAccessToken(cfg.JWTSecretKey), handleDeleteItem)
	}
}

// handleCreateItem implements POST /item/.
// Creates a new item in a specified store if the store is owned by the authenticated user.
// Response: 201 with item JSON, 404 if store not found, 401 if store not owned.
func handleCreateItem(c *gin.Context) {
	var req createItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Missing required fields"})
		return
	}

	// Look up the target store (mirrors Flask: Store.query.get_or_404(data['store_id']))
	store, found := findStoreOrNotFound(c, req.StoreID)
	if !found {
		return
	}

	// Ownership check: store.user_id != int(get_jwt_identity())
	if !checkOwnership(c, store.UserID) {
		return
	}

	// Create the item
	// Mirrors Flask: Item(name=data['name'], price=data['price'], store_id=data['store_id'])
	item := model.Item{
		Name:    req.Name,
		Price:   req.Price,
		StoreID: req.StoreID,
	}

	if err := db.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create item"})
		return
	}

	// Return item response matching Flask's item_schema.dump(item)
	c.JSON(http.StatusCreated, toItemResponse(&item))
}

// handleGetAllItems implements GET /item/s.
// Lists all items across all stores owned by the authenticated user.
// Uses the Flask two-query approach: first query stores, then items by store IDs.
// Response: 200 with array of item objects.
func handleGetAllItems(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	// Step 1: Get all stores for this user
	// Mirrors Flask: stores = Store.query.filter_by(user_id=int(get_jwt_identity())).all()
	var stores []model.Store
	if err := db.DB.Where("user_id = ?", userID).Find(&stores).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch stores"})
		return
	}

	// Step 2: Extract store IDs
	// Mirrors Flask: store_ids = [s.id for s in stores]
	storeIDs := make([]int, len(stores))
	for i, s := range stores {
		storeIDs[i] = s.ID
	}

	// Step 3: Query items by store IDs
	// Mirrors Flask: items = Item.query.filter(Item.store_id.in_(store_ids)).all()
	var items []model.Item
	if len(storeIDs) > 0 {
		if err := db.DB.Where("store_id IN ?", storeIDs).Find(&items).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch items"})
			return
		}
	}

	// Return empty array if no items (not null)
	if items == nil {
		items = []model.Item{}
	}

	// Return array of items matching Flask's items_schema.dump(items)
	c.JSON(http.StatusOK, toItemResponses(items))
}

// handleGetItem implements GET /item/:id.
// Retrieves an item by ID if its parent store is owned by the authenticated user.
// Response: 200 with item JSON, 404 if not found, 401 if not owned.
func handleGetItem(c *gin.Context) {
	// Parse item ID from URL parameter
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	// Find the item (mirrors Flask: Item.query.get_or_404(id))
	var item model.Item
	if err := db.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	// Look up parent store to check ownership
	// Mirrors Flask: item.store.user_id != int(get_jwt_identity())
	var store model.Store
	if err := db.DB.First(&store, item.StoreID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	if !checkOwnership(c, store.UserID) {
		return
	}

	// Return item data matching Flask's item_schema.dump(item)
	c.JSON(http.StatusOK, toItemResponse(&item))
}

// handleUpdateItem implements PUT /item/:id.
// Updates an item's name, price, and/or store_id with partial update semantics.
// If store_id is changed, the new store must also be owned by the authenticated user.
// Response: 200 with updated item JSON, 404 if not found, 401 if not owned.
func handleUpdateItem(c *gin.Context) {
	// Parse item ID from URL parameter
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	// Find the item (mirrors Flask: Item.query.get_or_404(id))
	var item model.Item
	if err := db.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	// Look up parent store to check ownership
	var store model.Store
	if err := db.DB.First(&store, item.StoreID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	if !checkOwnership(c, store.UserID) {
		return
	}

	// Parse the update request body as a map to support partial updates.
	// Flask uses data.get('name', item.name) which preserves existing values for missing fields.
	// Using a map lets us distinguish "field not provided" from "field set to zero value".
	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request body"})
		return
	}

	// Update name if provided — mirrors Flask: item.name = data.get('name', item.name)
	if name, ok := data["name"]; ok {
		if nameStr, ok := name.(string); ok {
			item.Name = nameStr
		}
	}

	// Update price if provided — mirrors Flask: item.price = data.get('price', item.price)
	if price, ok := data["price"]; ok {
		if priceFloat, ok := price.(float64); ok {
			item.Price = priceFloat
		}
	}

	// Update store_id if provided — mirrors Flask:
	//   if 'store_id' in data:
	//       new_store = Store.query.get_or_404(data['store_id'])
	//       if new_store.user_id != int(get_jwt_identity()):
	//           return jsonify({'message': 'Unauthorized'}), 401
	//       item.store_id = data['store_id']
	if storeIDRaw, ok := data["store_id"]; ok {
		storeIDFloat, ok := storeIDRaw.(float64)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid store_id"})
			return
		}
		newStoreID := int(storeIDFloat)

		// Look up the new store
		newStore, found := findStoreOrNotFound(c, newStoreID)
		if !found {
			return
		}

		// Check ownership of the new store
		if !checkOwnership(c, newStore.UserID) {
			return
		}

		item.StoreID = newStoreID
	}

	// Save changes — mirrors Flask: db.session.commit()
	if err := db.DB.Save(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update item"})
		return
	}

	// Return updated item matching Flask's item_schema.dump(item)
	c.JSON(http.StatusOK, toItemResponse(&item))
}

// handleDeleteItem implements DELETE /item/:id.
// Deletes an item if its parent store is owned by the authenticated user.
// Response: 200 with {"message": "Deleted"}, 404 if not found, 401 if not owned.
func handleDeleteItem(c *gin.Context) {
	// Parse item ID from URL parameter
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	// Find the item (mirrors Flask: Item.query.get_or_404(id))
	var item model.Item
	if err := db.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	// Look up parent store to check ownership
	// Mirrors Flask: item.store.user_id != int(get_jwt_identity())
	var store model.Store
	if err := db.DB.First(&store, item.StoreID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	if !checkOwnership(c, store.UserID) {
		return
	}

	// Delete the item — mirrors Flask: db.session.delete(item); db.session.commit()
	if err := db.DB.Delete(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete item"})
		return
	}

	// Matches Flask: return jsonify({'message': 'Deleted'})
	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}
