package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/config"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/db"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/handler"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting server in %s mode", cfg.AppEnv)

	// Initialize database
	if _, err := db.InitDB(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Set up Gin router
	router := SetupRouter(cfg)

	// Start server
	addr := ":" + cfg.Port
	log.Printf("Server listening on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// SetupRouter creates and configures the Gin engine with all route groups.
// This function is exported so it can be used by integration tests.
func SetupRouter(cfg *config.Config) *gin.Engine {
	// Set Gin mode based on environment
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default() // Includes Logger and Recovery middleware

	// Base route group (no prefix — all routes start from root)
	base := router.Group("")

	// Register resource route groups
	handler.RegisterHealthRoutes(base)

	// Future milestones will register additional route groups here:
	// handler.RegisterUserRoutes(base, cfg)
	// handler.RegisterStoreRoutes(base)
	// handler.RegisterItemRoutes(base)
	// handler.RegisterTagRoutes(base)

	return router
}
