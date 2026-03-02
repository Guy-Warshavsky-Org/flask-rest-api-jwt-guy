package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/modelcode-ai/flask-rest-api-jwt-guy/internal/config"
	"github.com/modelcode-ai/flask-rest-api-jwt-guy/internal/middleware"
	"github.com/modelcode-ai/flask-rest-api-jwt-guy/internal/model"
)

// ItemHandler contains handlers for item endpoints.
// Mirrors Flask's app/resources/items.py.
type ItemHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

// NewItemHandler creates a new ItemHandler.
func NewItemHandler(db *gorm.DB, cfg *config.Config) *ItemHandler {
	return &ItemHandler{DB: db, Cfg: cfg}
}

// createItemRequest represents the JSON body for POST /item/.
type createItemRequest struct {
	Name    string  `json:"name" binding:"required"`
	Price   float64 `json:"price" binding:"required"`
	StoreID int     `json:"store_id" binding:"required"`
}

// updateItemRequest represents the JSON body for PUT /item/:id.
// Pointer fields distinguish "absent" from "present with value" (Approach 1),
// mirroring Flask's data.get('field', default) pattern.
type updateItemRequest struct {
	Name    *string  `json:"name"`
	Price   *float64 `json:"price"`
	StoreID *int     `json:"store_id"`
}

// itemResponse mirrors the Marshmallow ItemSchema output (include_fk = True).
type itemResponse struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Price   float64 `json:"price"`
	StoreID int     `json:"store_id"`
}

// CreateItem handles POST /item/.
// Mirrors Flask: creates an item in a store, verifying the user owns the target store.
func (h *ItemHandler) CreateItem(c *gin.Context) {
	var req createItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request payload"})
		return
	}

	// Verify the target store exists and the user owns it.
	// Mirrors Flask: store = Store.query.get_or_404(data['store_id'])
	var store model.Store
	if err := h.DB.First(&store, req.StoreID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		return
	}

	// Ownership check on the store.
	if !checkOwnership(c, store.UserID) {
		return
	}

	item := model.Item{
		Name:    req.Name,
		Price:   req.Price,
		StoreID: req.StoreID,
	}

	if err := h.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create item"})
		return
	}

	c.JSON(http.StatusCreated, itemResponse{
		ID:      item.ID,
		Name:    item.Name,
		Price:   item.Price,
		StoreID: item.StoreID,
	})
}

// GetAllItems handles GET /item/s.
// Mirrors Flask: returns all items across the authenticated user's stores.
func (h *ItemHandler) GetAllItems(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	// Get all stores belonging to the user (mirrors Flask pattern).
	var stores []model.Store
	if err := h.DB.Where("user_id = ?", userID).Find(&stores).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		return
	}

	storeIDs := make([]int, len(stores))
	for i, s := range stores {
		storeIDs[i] = s.ID
	}

	var items []model.Item
	if len(storeIDs) > 0 {
		// Mirrors Flask: Item.query.filter(Item.store_id.in_(store_ids)).all()
		if err := h.DB.Where("store_id IN ?", storeIDs).Find(&items).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
			return
		}
	}

	// Build response slice matching Marshmallow ItemSchema(many=True) output.
	resp := make([]itemResponse, len(items))
	for i, item := range items {
		resp[i] = itemResponse{
			ID:      item.ID,
			Name:    item.Name,
			Price:   item.Price,
			StoreID: item.StoreID,
		}
	}

	c.JSON(http.StatusOK, resp)
}

// GetItem handles GET /item/:id.
// Mirrors Flask: returns item JSON, verifying user owns the item's store.
func (h *ItemHandler) GetItem(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	// Fetch item with its store preloaded for ownership check.
	// Mirrors Flask's item.store.user_id access (which triggers lazy load in SQLAlchemy).
	var item model.Item
	if err := h.DB.Preload("Store").First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		return
	}

	// Indirect ownership check through the store.
	if !checkOwnership(c, item.Store.UserID) {
		return
	}

	c.JSON(http.StatusOK, itemResponse{
		ID:      item.ID,
		Name:    item.Name,
		Price:   item.Price,
		StoreID: item.StoreID,
	})
}

// UpdateItem handles PUT /item/:id.
// Mirrors Flask: updates item fields (name, price, store_id) with ownership verification.
// Supports partial updates and cross-store transfer.
func (h *ItemHandler) UpdateItem(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	// Fetch item with its store preloaded.
	var item model.Item
	if err := h.DB.Preload("Store").First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		return
	}

	// Ownership check on the current store.
	if !checkOwnership(c, item.Store.UserID) {
		return
	}

	var req updateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request payload"})
		return
	}

	// Apply partial updates (mirrors Flask's data.get('field', item.field) pattern).
	if req.Name != nil {
		item.Name = *req.Name
	}
	if req.Price != nil {
		item.Price = *req.Price
	}

	// Handle cross-store transfer (mirrors Flask's 'store_id' in data check).
	if req.StoreID != nil {
		// Verify the destination store exists and the user owns it.
		var newStore model.Store
		if err := h.DB.First(&newStore, *req.StoreID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
			return
		}

		if !checkOwnership(c, newStore.UserID) {
			return
		}

		item.StoreID = *req.StoreID
	}

	if err := h.DB.Save(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update item"})
		return
	}

	c.JSON(http.StatusOK, itemResponse{
		ID:      item.ID,
		Name:    item.Name,
		Price:   item.Price,
		StoreID: item.StoreID,
	})
}

// DeleteItem handles DELETE /item/:id.
// Mirrors Flask: deletes an item, verifying user owns the item's store.
func (h *ItemHandler) DeleteItem(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	// Fetch item with its store preloaded for ownership check.
	var item model.Item
	if err := h.DB.Preload("Store").First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		return
	}

	// Indirect ownership check through the store.
	if !checkOwnership(c, item.Store.UserID) {
		return
	}

	if err := h.DB.Delete(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}
