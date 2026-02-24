package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/flask-rest-api-jwt-guy/internal/config"
	"github.com/flask-rest-api-jwt-guy/internal/database"
	"github.com/flask-rest-api-jwt-guy/internal/handler"
	"github.com/flask-rest-api-jwt-guy/internal/middleware"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := database.Init(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Create token blacklist (in-memory, matching Flask behavior)
	blacklist := middleware.NewTokenBlacklist()

	// Setup router
	router := SetupRouter(cfg, db, blacklist)

	// Start server
	addr := ":" + cfg.HTTPPort
	log.Printf("Starting server on %s (env=%s)", addr, cfg.Env)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// SetupRouter creates and configures the Gin router with all route groups.
// Exported so it can be reused in tests.
func SetupRouter(cfg *config.Config, db *gorm.DB, blacklist *middleware.TokenBlacklist) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// User routes: /user
	userHandler := &handler.UserHandler{
		DB:        db,
		Config:    cfg,
		Blacklist: blacklist,
	}
	userGroup := router.Group("/user")
	userHandler.RegisterUserRoutes(userGroup)

	// Health routes: /health
	healthHandler := &handler.HealthHandler{DB: db}
	healthGroup := router.Group("/health")
	healthHandler.RegisterHealthRoutes(healthGroup)

	return router
}
