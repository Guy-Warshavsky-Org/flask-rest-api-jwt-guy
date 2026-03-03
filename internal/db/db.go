package db

import (
	"fmt"
	"log"

	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/config"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// DB is the global database instance.
var DB *gorm.DB

// InitDB initializes the database connection based on the provided configuration.
// It automatically selects between SQLite and PostgreSQL drivers based on DATABASE_URL,
// runs AutoMigrate for all models, and stores the connection in the global DB variable.
func InitDB(cfg *config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector

	if cfg.IsSQLite() {
		dsn := cfg.SQLiteDSN()
		log.Printf("Using SQLite database: %s", dsn)
		dialector = sqlite.Open(dsn)
	} else {
		dsn := cfg.PostgresDSN()
		log.Printf("Using PostgreSQL database")
		dialector = postgres.Open(dsn)
	}

	database, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run AutoMigrate for all models
	if err := database.AutoMigrate(
		&model.User{},
		&model.Store{},
		&model.Item{},
		&model.Tag{},
	); err != nil {
		return nil, fmt.Errorf("failed to run database migrations: %w", err)
	}

	DB = database
	return database, nil
}

// InitTestDB creates an in-memory SQLite database for testing.
// Foreign key enforcement is enabled.
func InitTestDB() (*gorm.DB, error) {
	database, err := gorm.Open(sqlite.Open("file::memory:?_foreign_keys=on"), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to test database: %w", err)
	}

	if err := database.AutoMigrate(
		&model.User{},
		&model.Store{},
		&model.Item{},
		&model.Tag{},
	); err != nil {
		return nil, fmt.Errorf("failed to run test database migrations: %w", err)
	}

	DB = database
	return database, nil
}
