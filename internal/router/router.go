package router

import (
	"flask-rest-api-jwt-guy/internal/config"
	"flask-rest-api-jwt-guy/internal/handler"
	"flask-rest-api-jwt-guy/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// NewRouter creates and configures a Gin engine with all route groups and middleware.
// This mirrors the Flask blueprint registration in app/__init__.py.
func NewRouter(db *gorm.DB, cfg config.Config, blacklist *middleware.TokenBlacklist) *gin.Engine {
	r := gin.Default()

	// JWT middleware instances
	accessAuth := middleware.JWTAuth(cfg.SecretKey, blacklist)
	refreshAuth := middleware.JWTRefresh(cfg.SecretKey, blacklist)

	// User routes — mirrors Flask users_bp with url_prefix='/user'
	userGroup := r.Group("/user")
	{
		userGroup.POST("/register", handler.RegisterUser(db))
		userGroup.POST("/login", handler.Login(db, cfg.SecretKey))
		userGroup.POST("/logout", accessAuth, handler.Logout(blacklist))
		userGroup.POST("/refresh", refreshAuth, handler.RefreshToken(cfg.SecretKey))
		userGroup.GET("/:id", accessAuth, handler.GetUser(db))
		userGroup.DELETE("/:id", accessAuth, handler.DeleteUser(db))
	}

	// Health route — mirrors Flask health_bp with url_prefix='/health'
	healthGroup := r.Group("/health")
	{
		healthGroup.GET("/", handler.HealthCheck(db))
	}

	// Store routes will be added in milestone 2
	// Item routes will be added in milestone 2
	// Tag routes will be added in milestone 3

	return r
}
