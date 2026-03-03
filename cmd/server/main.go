package main

import (
	"log"

	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/config"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/db"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/router"
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
	r := router.SetupRouter(cfg)

	// Start server
	addr := ":" + cfg.Port
	log.Printf("Server listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
