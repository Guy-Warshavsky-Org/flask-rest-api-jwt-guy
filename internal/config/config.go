package config

import (
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Env              string        `mapstructure:"ENV"`
	DatabaseURL      string        `mapstructure:"DATABASE_URL"`
	DevDatabaseURL   string        `mapstructure:"DEV_DATABASE_URL"`
	JWTSecret        string        `mapstructure:"JWT_SECRET_KEY"`
	AccessTokenTTL   time.Duration `mapstructure:"JWT_ACCESS_TOKEN_EXPIRES"`
	RefreshTokenTTL  time.Duration `mapstructure:"JWT_REFRESH_TOKEN_EXPIRES"`
	HTTPPort         string        `mapstructure:"PORT"`
	SecretKey        string        `mapstructure:"SECRET_KEY"`
}

// Load reads configuration from environment variables and optional config file.
func Load() (*Config, error) {
	v := viper.New()

	// Defaults mirroring Flask DevelopmentConfig
	v.SetDefault("ENV", "development")
	v.SetDefault("PORT", "5000")
	v.SetDefault("SECRET_KEY", "your-secret-key-change-me")
	v.SetDefault("JWT_SECRET_KEY", "")
	v.SetDefault("JWT_ACCESS_TOKEN_EXPIRES", 15*time.Minute)
	v.SetDefault("JWT_REFRESH_TOKEN_EXPIRES", 30*24*time.Hour) // 30 days
	v.SetDefault("DATABASE_URL", "")
	v.SetDefault("DEV_DATABASE_URL", "")

	// Bind environment variables
	v.AutomaticEnv()

	// Optional config file
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	_ = v.ReadInConfig() // ignore if not found

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	// If JWT_SECRET_KEY is not set, fall back to SECRET_KEY (Flask behavior)
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = cfg.SecretKey
	}

	return cfg, nil
}

// GetDatabaseURL returns the appropriate database URL based on environment.
func (c *Config) GetDatabaseURL() string {
	if c.Env == "production" {
		if c.DatabaseURL != "" {
			return c.DatabaseURL
		}
		return "postgresql://user:password@db:5432/appdb"
	}
	// Development
	if c.DevDatabaseURL != "" {
		return c.DevDatabaseURL
	}
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	return "sqlite://data-dev.sqlite"
}
