package main

import (
	"fmt"
	"log"

	"github.com/modelcode-ai/flask-rest-api-jwt-guy/internal/config"
	"github.com/modelcode-ai/flask-rest-api-jwt-guy/internal/model"
	"github.com/modelcode-ai/flask-rest-api-jwt-guy/internal/router"
)

func main() {
	// Load configuration from environment variables.
	// Mirrors Flask's app.config.from_object(config[config_name]).
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting server in %s mode", cfg.AppEnv)

	// Connect to database.
	// Mirrors Flask-SQLAlchemy's db.init_app(app).
	db, err := config.OpenDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run auto-migration.
	// Mirrors Flask's create_db() / db.create_all().
	if err := model.AutoMigrateAll(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Printf("Database connected and migrated successfully")

	// Setup router and start server.
	// Mirrors Flask's app.run().
	r := router.Setup(db, cfg)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Server listening on %s", addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
