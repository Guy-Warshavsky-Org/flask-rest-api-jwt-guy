package router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/modelcode-ai/flask-rest-api-jwt-guy/internal/config"
	"github.com/modelcode-ai/flask-rest-api-jwt-guy/internal/handler"
	"github.com/modelcode-ai/flask-rest-api-jwt-guy/internal/middleware"
)

// Setup creates and configures the Gin engine with all route groups.
// This replaces Flask's Blueprint registration in app/__init__.py.
func Setup(db *gorm.DB, cfg *config.Config) *gin.Engine {
	if !cfg.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Shared middleware for access-token-protected routes.
	accessAuth := middleware.JWTAuth(cfg.SecretKey, middleware.AccessToken)

	// Initialize handlers.
	userHandler := handler.NewUserHandler(db, cfg)
	storeHandler := handler.NewStoreHandler(db, cfg)
	itemHandler := handler.NewItemHandler(db, cfg)

	// User routes (mirrors Flask's users_bp with url_prefix='/user').
	userGroup := r.Group("/user")
	{
		// Public routes (no JWT required).
		userGroup.POST("/register", userHandler.Register)
		userGroup.POST("/login", userHandler.Login)

		// Protected routes (access token required).
		userGroup.POST("/logout", accessAuth, userHandler.Logout)
		userGroup.GET("/:id", accessAuth, userHandler.GetUser)
		userGroup.DELETE("/:id", accessAuth, userHandler.DeleteUser)

		// Refresh token route.
		refreshAuth := middleware.JWTAuth(cfg.SecretKey, middleware.RefreshToken)
		userGroup.POST("/refresh", refreshAuth, userHandler.Refresh)
	}

	// Store routes (mirrors Flask's stores_bp with url_prefix='/store').
	storeGroup := r.Group("/store")
	storeGroup.Use(accessAuth)
	{
		storeGroup.POST("/", storeHandler.CreateStore)
		storeGroup.GET("/:id", storeHandler.GetStore)
		storeGroup.GET("/s", storeHandler.GetAllStores)
		storeGroup.DELETE("/:id", storeHandler.DeleteStore)
	}

	// Item routes (mirrors Flask's items_bp with url_prefix='/item').
	itemGroup := r.Group("/item")
	itemGroup.Use(accessAuth)
	{
		itemGroup.POST("/", itemHandler.CreateItem)
		itemGroup.GET("/s", itemHandler.GetAllItems)
		itemGroup.GET("/:id", itemHandler.GetItem)
		itemGroup.PUT("/:id", itemHandler.UpdateItem)
		itemGroup.DELETE("/:id", itemHandler.DeleteItem)
	}

	// Additional route groups (tags, health) will be added
	// in subsequent milestones.

	return r
}
