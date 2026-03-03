package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application.
type Config struct {
	AppEnv            string        `mapstructure:"APP_ENV"`
	Port              string        `mapstructure:"PORT"`
	DatabaseURL       string        `mapstructure:"DATABASE_URL"`
	JWTSecretKey      string        `mapstructure:"JWT_SECRET_KEY"`
	JWTAccessExpires  time.Duration `mapstructure:"JWT_ACCESS_EXPIRES"`
	JWTRefreshExpires time.Duration `mapstructure:"JWT_REFRESH_EXPIRES"`
}

// IsSQLite returns true if the configured database URL indicates a SQLite database.
func (c *Config) IsSQLite() bool {
	return c.DatabaseURL == "" ||
		strings.HasPrefix(c.DatabaseURL, "sqlite://") ||
		strings.HasPrefix(c.DatabaseURL, "file:")
}

// SQLiteDSN returns the DSN string suitable for the GORM SQLite driver.
// It strips the "sqlite:///" or "sqlite://" prefix if present and ensures
// foreign key enforcement is enabled.
func (c *Config) SQLiteDSN() string {
	dsn := c.DatabaseURL

	if dsn == "" {
		dsn = "file:data-dev.sqlite?_foreign_keys=on"
		return dsn
	}

	// Strip sqlite:/// or sqlite://
	dsn = strings.TrimPrefix(dsn, "sqlite:///")
	dsn = strings.TrimPrefix(dsn, "sqlite://")

	// If it already has the file: prefix, ensure _foreign_keys param
	if strings.HasPrefix(dsn, "file:") {
		if !strings.Contains(dsn, "_foreign_keys") {
			if strings.Contains(dsn, "?") {
				dsn += "&_foreign_keys=on"
			} else {
				dsn += "?_foreign_keys=on"
			}
		}
		return dsn
	}

	// Plain file path — wrap with file: prefix and add foreign keys
	if strings.Contains(dsn, "?") {
		dsn = "file:" + dsn + "&_foreign_keys=on"
	} else {
		dsn = "file:" + dsn + "?_foreign_keys=on"
	}
	return dsn
}

// PostgresDSN returns the DSN for GORM's PostgreSQL driver.
func (c *Config) PostgresDSN() string {
	return c.DatabaseURL
}

// LoadConfig reads configuration from environment variables with defaults.
func LoadConfig() (*Config, error) {
	v := viper.New()

	// Set defaults matching the Flask app
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("PORT", "5000")
	v.SetDefault("DATABASE_URL", "")
	v.SetDefault("JWT_SECRET_KEY", "your-secret-key-change-me")
	v.SetDefault("JWT_ACCESS_EXPIRES", 15*time.Minute)
	v.SetDefault("JWT_REFRESH_EXPIRES", 30*24*time.Hour) // 30 days

	// Read from environment variables
	v.AutomaticEnv()

	cfg := &Config{}
	cfg.AppEnv = v.GetString("APP_ENV")
	cfg.Port = v.GetString("PORT")
	cfg.DatabaseURL = v.GetString("DATABASE_URL")
	cfg.JWTSecretKey = v.GetString("JWT_SECRET_KEY")
	cfg.JWTAccessExpires = v.GetDuration("JWT_ACCESS_EXPIRES")
	cfg.JWTRefreshExpires = v.GetDuration("JWT_REFRESH_EXPIRES")

	return cfg, nil
}
