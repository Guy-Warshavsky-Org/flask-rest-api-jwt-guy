package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Config holds all application configuration loaded from environment variables.
// Mirrors the Flask config.py with DevelopmentConfig and ProductionConfig defaults.
type Config struct {
	// AppEnv selects the environment: "development" (SQLite) or "production" (PostgreSQL).
	AppEnv string `env:"APP_ENV" envDefault:"development"`

	// Port the HTTP server listens on.
	Port string `env:"PORT" envDefault:"5000"`

	// DatabaseURL is the connection string.
	// Development default: SQLite file. Production default: PostgreSQL DSN.
	DatabaseURL string `env:"DATABASE_URL"`

	// SecretKey used for JWT signing. Mirrors Flask's SECRET_KEY.
	SecretKey string `env:"SECRET_KEY" envDefault:"your-secret-key-change-me"`

	// JWTAccessExpiry controls access token lifetime (default 15 minutes, matching Flask).
	JWTAccessExpiry time.Duration `env:"JWT_ACCESS_EXPIRY" envDefault:"15m"`

	// JWTRefreshExpiry controls refresh token lifetime (default 30 days, matching Flask).
	JWTRefreshExpiry time.Duration `env:"JWT_REFRESH_EXPIRY" envDefault:"720h"`

	// Debug enables verbose logging.
	Debug bool `env:"DEBUG" envDefault:"false"`
}

// Load parses environment variables into a Config struct.
func Load() (*Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Apply environment-specific defaults for DatabaseURL if not set.
	if cfg.DatabaseURL == "" {
		if cfg.AppEnv == "production" {
			cfg.DatabaseURL = "postgresql://user:password@db:5432/appdb"
		} else {
			cfg.DatabaseURL = "data-dev.sqlite"
		}
	}

	return &cfg, nil
}

// OpenDB opens a GORM database connection based on the DatabaseURL.
// It detects PostgreSQL URLs (starting with "postgres://", "postgresql://")
// and uses the appropriate driver.
func OpenDB(cfg *Config) (*gorm.DB, error) {
	var dialector gorm.Dialector

	if isPostgres(cfg.DatabaseURL) {
		dialector = postgres.Open(cfg.DatabaseURL)
	} else {
		// SQLite: strip "sqlite:///" prefix if present for compatibility.
		dsn := cfg.DatabaseURL
		dsn = strings.TrimPrefix(dsn, "sqlite:///")
		dsn = strings.TrimPrefix(dsn, "sqlite://")
		dialector = sqlite.Open(dsn)
	}

	gormCfg := &gorm.Config{}
	db, err := gorm.Open(dialector, gormCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}

// isPostgres checks if the URL indicates a PostgreSQL connection.
func isPostgres(url string) bool {
	return strings.HasPrefix(url, "postgres://") || strings.HasPrefix(url, "postgresql://")
}
