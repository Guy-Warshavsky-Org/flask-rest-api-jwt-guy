package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds application configuration values.
type Config struct {
	Env            string // "development" or "production"
	SecretKey      string
	DatabaseURL    string
	DevDatabaseURL string
	HTTPPort       int
}

// Load reads configuration from environment variables using Viper
// and returns a populated Config struct.
func Load() (Config, error) {
	v := viper.New()

	// Bind environment variables
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("SECRET_KEY", "your-secret-key-change-me")
	v.SetDefault("PORT", 5000)

	v.BindEnv("APP_ENV")
	v.BindEnv("SECRET_KEY")
	v.BindEnv("DATABASE_URL")
	v.BindEnv("DEV_DATABASE_URL")
	v.BindEnv("PORT")

	env := v.GetString("APP_ENV")
	secretKey := v.GetString("SECRET_KEY")
	databaseURL := v.GetString("DATABASE_URL")
	devDatabaseURL := v.GetString("DEV_DATABASE_URL")
	httpPort := v.GetInt("PORT")

	cfg := Config{
		Env:            env,
		SecretKey:      secretKey,
		DatabaseURL:    databaseURL,
		DevDatabaseURL: devDatabaseURL,
		HTTPPort:       httpPort,
	}

	// Resolve the active database URL based on environment
	if env == "production" {
		if cfg.DatabaseURL == "" {
			return cfg, fmt.Errorf("DATABASE_URL is required in production mode")
		}
	} else {
		// Development mode: use DEV_DATABASE_URL or default SQLite
		if cfg.DevDatabaseURL == "" {
			// Default to SQLite file in current working directory
			cwd, err := os.Getwd()
			if err != nil {
				cwd = "."
			}
			cfg.DevDatabaseURL = filepath.Join(cwd, "data-dev.sqlite")
		}
	}

	return cfg, nil
}

// ActiveDSN returns the database connection string to use based on the environment.
// For production, it returns DatabaseURL (PostgreSQL).
// For development, it returns DevDatabaseURL (SQLite path).
func (c Config) ActiveDSN() string {
	if c.Env == "production" {
		return c.DatabaseURL
	}
	return c.DevDatabaseURL
}

// IsProduction returns true if the environment is production.
func (c Config) IsProduction() bool {
	return c.Env == "production"
}
