package database

import (
	"fmt"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/flask-rest-api-jwt-guy/internal/config"
	"github.com/flask-rest-api-jwt-guy/internal/model"
)

// Init initializes and returns a GORM database connection based on configuration.
func Init(cfg *config.Config) (*gorm.DB, error) {
	dbURL := cfg.GetDatabaseURL()

	var db *gorm.DB
	var err error

	if strings.HasPrefix(dbURL, "sqlite://") {
		// SQLite: strip the sqlite:// prefix
		path := strings.TrimPrefix(dbURL, "sqlite://")
		db, err = gorm.Open(sqlite.Open(path), &gorm.Config{})
		if err == nil {
			// Enable foreign key enforcement for SQLite
			sqlDB, dbErr := db.DB()
			if dbErr == nil {
				sqlDB.Exec("PRAGMA foreign_keys = ON")
			}
		}
	} else if strings.HasPrefix(dbURL, "postgresql://") || strings.HasPrefix(dbURL, "postgres://") {
		db, err = gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	} else {
		return nil, fmt.Errorf("unsupported database URL scheme: %s", dbURL)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto-migrate all models
	if err := model.AutoMigrateAll(db); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate: %w", err)
	}

	return db, nil
}
