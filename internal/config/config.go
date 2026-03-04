package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config holds all configuration for the application.
// Values are loaded from environment variables, with defaults provided via struct tags.
type Config struct {
	AppEnv             string        `env:"APP_ENV" envDefault:"development"`
	Port               int           `env:"APP_PORT" envDefault:"8080"`
	LogLevel           string        `env:"LOG_LEVEL" envDefault:"debug"`
	DBPath             string        `env:"DB_PATH" envDefault:"./banana_farm.db"`
	CORSAllowedOrigins []string      `env:"CORS_ALLOWED_ORIGINS" envSeparator:"," envDefault:"*"`
	RateLimitMax       int           `env:"RATE_LIMIT_MAX" envDefault:"100"`
	RateLimitWindow    time.Duration `env:"RATE_LIMIT_WINDOW" envDefault:"1m"`
}

// IsDevelopment returns true if the app is running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

// IsProduction returns true if the app is running in production mode.
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// IsTest returns true if the app is running in test mode.
func (c *Config) IsTest() bool {
	return c.AppEnv == "test"
}

// Load reads the .env file (if it exists) and parses environment variables into a Config struct.
// The .env file is optional — in production, env vars are typically set by the host environment.
func Load() (*Config, error) {
	// Load .env file. Ignore error if file doesn't exist (production won't have one).
	_ = godotenv.Load()

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}
