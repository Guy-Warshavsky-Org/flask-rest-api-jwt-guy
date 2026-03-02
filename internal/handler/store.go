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

// StoreHandler contains handlers for store endpoints.
// Mirrors Flask's app/resources/stores.py.
type StoreHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

// NewStoreHandler creates a new StoreHandler.
func NewStoreHandler(db *gorm.DB, cfg *config.Config) *StoreHandler {
	return &StoreHandler{DB: db, Cfg: cfg}
}

// createStoreRequest represents the JSON body for POST /store/.
type createStoreRequest struct {
	Name string `json:"name" binding:"required"`
}

// storeResponse mirrors the Marshmallow StoreSchema output (include_fk = True).
type storeResponse struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	UserID int    `json:"user_id"`
}

// checkOwnership compares the resource owner's user ID with the authenticated user.
// If they do not match, it writes a 401 JSON response and returns false.
// Handlers should return early when this returns false.
func checkOwnership(c *gin.Context, ownerID int) bool {
	userID, _ := middleware.GetUserID(c)
	if userID != ownerID {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return false
	}
	return true
}

// CreateStore handles POST /store/.
// Mirrors Flask: creates a store owned by the authenticated user, returns 201.
func (h *StoreHandler) CreateStore(c *gin.Context) {
	var req createStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request payload"})
		return
	}

	userID, _ := middleware.GetUserID(c)

	store := model.Store{
		Name:   req.Name,
		UserID: userID,
	}

	if err := h.DB.Create(&store).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create store"})
		return
	}

	c.JSON(http.StatusCreated, storeResponse{
		ID:     store.ID,
		Name:   store.Name,
		UserID: store.UserID,
	})
}

// GetStore handles GET /store/:id.
// Mirrors Flask: returns store JSON if the authenticated user owns it.
func (h *StoreHandler) GetStore(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	var store model.Store
	if err := h.DB.First(&store, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		return
	}

	// Ownership check (mirrors: if store.user_id != int(get_jwt_identity())).
	if !checkOwnership(c, store.UserID) {
		return
	}

	c.JSON(http.StatusOK, storeResponse{
		ID:     store.ID,
		Name:   store.Name,
		UserID: store.UserID,
	})
}

// GetAllStores handles GET /store/s.
// Mirrors Flask: returns all stores belonging to the authenticated user.
func (h *StoreHandler) GetAllStores(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var stores []model.Store
	if err := h.DB.Where("user_id = ?", userID).Find(&stores).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		return
	}

	// Build response slice matching Marshmallow StoreSchema(many=True) output.
	resp := make([]storeResponse, len(stores))
	for i, s := range stores {
		resp[i] = storeResponse{
			ID:     s.ID,
			Name:   s.Name,
			UserID: s.UserID,
		}
	}

	c.JSON(http.StatusOK, resp)
}

// DeleteStore handles DELETE /store/:id.
// Mirrors Flask: deletes a store (with cascade to items and tags) if the user owns it.
func (h *StoreHandler) DeleteStore(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	var store model.Store
	if err := h.DB.First(&store, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		return
	}

	// Ownership check.
	if !checkOwnership(c, store.UserID) {
		return
	}

	// Delete store. GORM cascade constraints handle related items and tags.
	// Use Select to trigger GORM cascade for has-many relationships.
	if err := h.DB.Select("Items", "Tags").Delete(&store).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete store"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}
