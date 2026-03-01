package main

import (
	"fmt"
	"log"

	"flask-rest-api-jwt-guy/internal/config"
	"flask-rest-api-jwt-guy/internal/middleware"
	"flask-rest-api-jwt-guy/internal/model"
	"flask-rest-api-jwt-guy/internal/router"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting server in %s mode", cfg.Env)

	// Initialize database connection
	db, err := initDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run auto-migration
	if err := model.AutoMigrateAll(db); err != nil {
		log.Fatalf("Failed to run auto-migration: %v", err)
	}

	// Initialize token blacklist
	blacklist := middleware.NewTokenBlacklist()

	// Create router
	r := router.NewRouter(db, cfg, blacklist)

	// Start HTTP server
	addr := fmt.Sprintf(":%d", cfg.HTTPPort)
	log.Printf("Listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// initDB initializes the GORM database connection based on the configuration.
// Production uses PostgreSQL, development uses SQLite.
func initDB(cfg config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector

	if cfg.IsProduction() {
		dialector = postgres.Open(cfg.ActiveDSN())
	} else {
		dialector = sqlite.Open(cfg.ActiveDSN())
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return db, nil
}
