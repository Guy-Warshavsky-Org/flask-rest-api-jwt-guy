package router

import (
	"github.com/gin-gonic/gin"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/config"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/handler"
)

// SetupRouter creates and configures the Gin engine with all route groups.
// This function is used by both the main entrypoint and integration tests.
func SetupRouter(cfg *config.Config) *gin.Engine {
	// Set Gin mode based on environment
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else if cfg.AppEnv == "test" {
		gin.SetMode(gin.TestMode)
	}

	engine := gin.Default() // Includes Logger and Recovery middleware

	// Base route group (no prefix — all routes start from root)
	base := engine.Group("")

	// Register resource route groups
	handler.RegisterHealthRoutes(base)
	handler.RegisterUserRoutes(base, cfg)

	// Future milestones will register additional route groups here:
	// handler.RegisterStoreRoutes(base)
	// handler.RegisterItemRoutes(base)
	// handler.RegisterTagRoutes(base)

	return engine
}
