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

// createStoreRequest represents the JSON body for POST /store/.
type createStoreRequest struct {
	Name string `json:"name" binding:"required"`
}

// storeResponse mirrors Flask's StoreSchema with include_fk=True.
// Fields: id, name, user_id — matching the Marshmallow auto-schema output.
type storeResponse struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	UserID int    `json:"user_id"`
}

// toStoreResponse converts a Store model to the API response DTO.
func toStoreResponse(s *model.Store) storeResponse {
	return storeResponse{
		ID:     s.ID,
		Name:   s.Name,
		UserID: s.UserID,
	}
}

// toStoreResponses converts a slice of Store models to response DTOs.
func toStoreResponses(stores []model.Store) []storeResponse {
	result := make([]storeResponse, len(stores))
	for i, s := range stores {
		result[i] = toStoreResponse(&s)
	}
	return result
}

// getUserIDFromContext extracts the authenticated user's ID from the Gin context
// (set by the JWT auth middleware). Returns the user ID as int and true on success,
// or writes a 401 response and returns 0, false on failure.
func getUserIDFromContext(c *gin.Context) (int, bool) {
	currentUser := c.GetString(middleware.ContextKeyUserID)
	if currentUser == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return 0, false
	}
	userID, err := strconv.Atoi(currentUser)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return 0, false
	}
	return userID, true
}

// checkOwnership compares the authenticated user's ID against the resource owner's ID.
// Returns true if they match. If they don't match, writes a 401 JSON response and returns false.
func checkOwnership(c *gin.Context, ownerID int) bool {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return false
	}
	if userID != ownerID {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return false
	}
	return true
}

// RegisterStoreRoutes registers all store-related endpoints on the given router group.
// Routes mirror the Flask stores_bp Blueprint with url_prefix='/store'.
func RegisterStoreRoutes(rg *gin.RouterGroup, cfg *config.Config) {
	store := rg.Group("/store")
	{
		// All store routes require access token (mirrors @jwt_required())
		store.POST("/", middleware.RequireAccessToken(cfg.JWTSecretKey), handleCreateStore)
		store.GET("/:id", middleware.RequireAccessToken(cfg.JWTSecretKey), handleGetStore)
		store.GET("/s", middleware.RequireAccessToken(cfg.JWTSecretKey), handleGetAllStores)
		store.DELETE("/:id", middleware.RequireAccessToken(cfg.JWTSecretKey), handleDeleteStore)
	}
}

// handleCreateStore implements POST /store/.
// Creates a new store for the authenticated user.
// Response: 201 with {"id": N, "name": "...", "user_id": N}.
func handleCreateStore(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	var req createStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Missing required fields"})
		return
	}

	// Create store with authenticated user's ID
	// Mirrors Flask: Store(name=data['name'], user_id=int(get_jwt_identity()))
	store := model.Store{
		Name:   req.Name,
		UserID: userID,
	}

	if err := db.DB.Create(&store).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create store"})
		return
	}

	// Return store response matching Flask's store_schema.dump(store)
	c.JSON(http.StatusCreated, toStoreResponse(&store))
}

// handleGetStore implements GET /store/:id.
// Retrieves a store by ID if owned by the authenticated user.
// Response: 200 with store JSON, 404 if not found, 401 if not owned.
func handleGetStore(c *gin.Context) {
	// Parse store ID from URL parameter
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	// Find the store (mirrors Flask: Store.query.get_or_404(id))
	var store model.Store
	if err := db.DB.First(&store, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	// Ownership check: store.user_id != int(get_jwt_identity())
	if !checkOwnership(c, store.UserID) {
		return
	}

	// Return store data matching Flask's store_schema.dump(store)
	c.JSON(http.StatusOK, toStoreResponse(&store))
}

// handleGetAllStores implements GET /store/s.
// Lists all stores owned by the authenticated user.
// Response: 200 with array of store objects.
func handleGetAllStores(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	// Mirrors Flask: Store.query.filter_by(user_id=int(get_jwt_identity())).all()
	var stores []model.Store
	if err := db.DB.Where("user_id = ?", userID).Find(&stores).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch stores"})
		return
	}

	// Return array of stores matching Flask's stores_schema.dump(stores)
	c.JSON(http.StatusOK, toStoreResponses(stores))
}

// handleDeleteStore implements DELETE /store/:id.
// Deletes a store (cascading to items and tags) if owned by the authenticated user.
// Response: 200 with {"message": "Deleted"}, 404 if not found, 401 if not owned.
func handleDeleteStore(c *gin.Context) {
	// Parse store ID from URL parameter
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	// Find the store (mirrors Flask: Store.query.get_or_404(id))
	var store model.Store
	if err := db.DB.First(&store, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	// Ownership check
	if !checkOwnership(c, store.UserID) {
		return
	}

	// Delete associated items and tags first (for SQLite cascade support)
	// then delete the store itself. Mirrors Flask:
	//   db.session.delete(store)  # cascade='all, delete-orphan' handles children
	//   db.session.commit()
	if err := db.DB.Where("store_id = ?", store.ID).Delete(&model.Item{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete store items"})
		return
	}
	if err := db.DB.Where("store_id = ?", store.ID).Delete(&model.Tag{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete store tags"})
		return
	}
	if err := db.DB.Delete(&store).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete store"})
		return
	}

	// Matches Flask: return jsonify({'message': 'Deleted'})
	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}

// findStoreOrNotFound looks up a store by ID. Returns the store and true if found,
// or writes a 404 JSON response and returns nil, false. Used by item handlers too.
func findStoreOrNotFound(c *gin.Context, storeID int) (*model.Store, bool) {
	var store model.Store
	if err := db.DB.First(&store, storeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return nil, false
	}
	return &store, true
}
