# Phase 1: Foundation

Set up the project skeleton, configuration, logging, database connection, and server with graceful shutdown.

By the end of this phase you'll have a running Go server that loads config from `.env`, logs with colors, connects to SQLite, and shuts down gracefully.

---

## Step 1: Initialize the Go module

Run from the project root:

```bash
go mod init github.com/justincordova/banana-farm-api
```

This creates `go.mod`. The module path can be anything, but using your GitHub path is convention.

---

## Step 2: Install all dependencies

Run each command. Go will update `go.mod` and create `go.sum` automatically.

```bash
# Router
go get github.com/go-chi/chi/v5
go get github.com/go-chi/cors
go get github.com/go-chi/httprate

# ORM + Database
go get gorm.io/gorm
go get gorm.io/driver/sqlite

# Config
go get github.com/caarlos0/env/v11
go get github.com/joho/godotenv

# Validation
go get github.com/go-playground/validator/v10

# Logging
go get github.com/lmittmann/tint
go get gopkg.in/lumberjack.v2

# Testing
go get github.com/stretchr/testify

# Hot reload (dev tool, install globally)
go install github.com/air-verse/air@latest
```

---

## Step 3: Create `.env` and `.env.example`

### Create `.env`

```env
APP_ENV=development
APP_PORT=8080
LOG_LEVEL=debug
DB_PATH=./banana_farm.db
CORS_ALLOWED_ORIGINS=http://localhost:3000
RATE_LIMIT_MAX=100
RATE_LIMIT_WINDOW=1m
```

### Create `.env.example`

Identical to `.env` but with placeholder values. This gets committed to git so other devs know what env vars are needed.

```env
APP_ENV=development
APP_PORT=8080
LOG_LEVEL=debug
DB_PATH=./banana_farm.db
CORS_ALLOWED_ORIGINS=http://localhost:3000
RATE_LIMIT_MAX=100
RATE_LIMIT_WINDOW=1m
```

### Create `.gitignore`

```gitignore
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
banana-farm-api

# Database
*.db
*.db-journal
*.db-shm
*.db-wal

# Environment
.env

# Logs
logs/
*.log

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Air
tmp/
```

---

## Step 4: Create `config/config.go`

Create the `config/` directory and the config file.

```go
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
```

### What's happening here:
- `godotenv.Load()` reads the `.env` file and sets the values as environment variables
- `env.Parse(cfg)` reads those environment variables and populates the struct
- The `envDefault` tags provide fallback values if an env var is missing
- `CORSAllowedOrigins` uses `envSeparator:","` so you can pass multiple origins: `http://localhost:3000,http://localhost:5173`
- The `Is*()` methods give you clean conditionals: `if cfg.IsDevelopment() { ... }`

---

## Step 5: Create `config/logger.go`

```go
package config

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"gopkg.in/lumberjack.v2"
)

// SetupLogger configures the global slog logger based on the application environment.
//
// - Development: colored text output to stdout via tint
// - Production: JSON output to stdout + rotating log files via lumberjack
// - Test: discards all output
func SetupLogger(cfg *Config) {
	level := parseLogLevel(cfg.LogLevel)

	var handler slog.Handler

	switch {
	case cfg.IsTest():
		// Silent logger for tests — no output
		handler = slog.NewTextHandler(io.Discard, nil)

	case cfg.IsProduction():
		// JSON handler that writes to both stdout and a rotating log file
		logFile := &lumberjack.Logger{
			Filename:   "logs/app.log",
			MaxSize:    10, // MB
			MaxBackups: 10,
			MaxAge:     30, // days
		}

		// MultiWriter sends output to both stdout and the log file
		multiWriter := io.MultiWriter(os.Stdout, logFile)

		handler = slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
			Level: level,
		})

	default:
		// Development: colored text output via tint
		handler = tint.NewHandler(os.Stdout, &tint.Options{
			Level:      level,
			TimeFormat: time.DateTime, // "2006-01-02 15:04:05"
		})
	}

	// Set the configured handler as the global default
	slog.SetDefault(slog.New(handler))
}

// parseLogLevel converts a string log level to a slog.Level.
// Defaults to slog.LevelDebug if the string is not recognized.
func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelDebug
	}
}
```

### What's happening here:
- **tint** replaces the default text handler with colored output. In your terminal you'll see colored log levels (green INFO, yellow WARN, red ERROR)
- **lumberjack** handles log file rotation — when `app.log` hits 10MB, it rotates to `app-2024-01-15T10-30-00.log` and starts a new file. Keeps 10 backups max, deletes files older than 30 days
- **`io.MultiWriter`** is a stdlib trick that duplicates writes to multiple destinations (stdout + file)
- **`slog.SetDefault`** sets the global logger so you can use `slog.Info(...)` anywhere without passing a logger around
- The `parseLogLevel` function maps your `LOG_LEVEL` env var string to a `slog.Level` constant

---

## Step 6: Create `database/database.go`

Create the `database/` directory and the database file.

```go
package database

import (
	"fmt"
	"log/slog"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect opens a SQLite database connection and returns the GORM DB instance.
// The dbPath is the file path to the SQLite database (e.g., "./banana_farm.db").
// For tests, pass a unique in-memory DSN: "file:name?mode=memory&cache=private"
func Connect(dbPath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		// Silent mode — we use slog for logging, not GORM's built-in logger
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// SQLite does not enforce foreign key constraints by default.
	// Enable them so ON DELETE CASCADE works correctly.
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	slog.Info("database connected", "path", dbPath)
	return db, nil
}

// Migrate runs auto-migration for all models.
// Pass all your model structs here. GORM will create/update tables as needed.
// NOTE: models will be added in Phase 2 and 3. For now this is a placeholder.
func Migrate(db *gorm.DB) error {
	slog.Info("running database migrations")

	err := db.AutoMigrate(
	// Models will be added here as we create them:
	// &models.Farm{},
	// &models.BananaTree{},
	// &models.Bunch{},
	// &models.Banana{},
	// &models.Tool{},
	// &models.Worker{},
	)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	slog.Info("database migrations completed")
	return nil
}

// Close gracefully closes the database connection.
// Call this during server shutdown.
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	return sqlDB.Close()
}
```

### What's happening here:
- **`gorm.Open`** opens the SQLite database. If the file doesn't exist, SQLite creates it automatically
- **`logger.Silent`** disables GORM's built-in query logging. We'll use slog for all logging to keep things consistent
- **`AutoMigrate`** creates tables if they don't exist and adds new columns if you add fields to your structs. It does NOT delete columns or drop tables (safe for development)
- **`Close`** gets the underlying `*sql.DB` from GORM and closes it. Important for graceful shutdown so connections don't leak
- The `Migrate` function is a placeholder — we'll uncomment models as we create them in Phase 2 and 3

---

## Step 7: Create `main.go`

This is the entry point. It wires everything together and handles graceful shutdown.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/justincordova/banana-farm-api/config"
	"github.com/justincordova/banana-farm-api/database"
)

func main() {
	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize logger
	config.SetupLogger(cfg)

	slog.Info("starting banana farm api",
		"env", cfg.AppEnv,
		"port", cfg.Port,
		"pid", os.Getpid(),
	)

	// 3. Connect to database
	db, err := database.Connect(cfg.DBPath)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	// 4. Run migrations
	if err := database.Migrate(db); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// 5. Build router
	// TODO: Replace with routes.Setup(cfg, db) in Phase 2
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "Banana Farm API"}`))
	})

	// 6. Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
	}

	// 7. Start server in a goroutine
	go func() {
		slog.Info("server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// 8. Wait for interrupt signal (Ctrl+C or kill)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	slog.Info("shutdown signal received", "signal", sig.String())

	// 9. Graceful shutdown with 30 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	// 10. Close database connection
	if err := database.Close(db); err != nil {
		slog.Error("failed to close database", "error", err)
	}

	slog.Info("server exited gracefully")
}
```

### What's happening here:

**Lines 1-5: Startup sequence**
- Load config from `.env` → initialize slog → connect DB → run migrations
- If any step fails, log the error and exit with code 1
- Uses `fmt.Fprintf` for the config error because slog isn't set up yet at that point

**Lines 6-7: Server startup**
- The server is started in a **goroutine** (a lightweight thread) so the main function can continue to the signal handler
- The `ListenAndServe` error check ignores `http.ErrServerClosed` because that's the expected error when `Shutdown` is called
- If the port is already in use, you'll get `bind: address already in use` here and the process exits

**Lines 8-10: Graceful shutdown**
- `signal.Notify` tells Go to send `SIGINT` (Ctrl+C) and `SIGTERM` (kill/docker stop) to the `quit` channel
- `<-quit` blocks until a signal is received — this is the main goroutine just waiting
- `srv.Shutdown(ctx)` stops accepting new connections and waits up to 30 seconds for in-flight requests to complete
- After the server is shut down, close the DB connection
- If shutdown takes longer than 30 seconds, the context expires and `Shutdown` returns an error

**Temporary route:** The `GET /` handler is a placeholder. In Phase 2, we'll replace `http.NewServeMux()` with our Chi router from `routes.Setup()`.

---

## Step 8: Create Air config for hot reload

Create `.air.toml` in the project root:

```toml
root = "."
tmp_dir = "tmp"

[build]
  cmd = "go build -o ./tmp/main ."
  bin = "./tmp/main"
  include_ext = ["go", "toml", "env"]
  exclude_dir = ["tmp", "vendor", "docs"]
  delay = 1000 # ms

[log]
  time = false

[misc]
  clean_on_exit = true
```

### How to use Air:
```bash
# Instead of `go run .`, use:
air

# Air watches for file changes and automatically rebuilds + restarts
# You'll see your colored slog output in the terminal
```

---

## Verify Phase 1 is complete

Run the server:

```bash
go run .
```

You should see colored output like:

```
2024-01-15 10:30:00 INFO starting banana farm api env=development port=8080 pid=12345
2024-01-15 10:30:00 INFO database connected path=./banana_farm.db
2024-01-15 10:30:00 INFO running database migrations
2024-01-15 10:30:00 INFO database migrations completed
2024-01-15 10:30:00 INFO server listening addr=:8080
```

Test the placeholder route:

```bash
curl http://localhost:8080/
# {"message": "Banana Farm API"}
```

Test graceful shutdown by pressing `Ctrl+C`:

```
2024-01-15 10:31:00 INFO shutdown signal received signal=interrupt
2024-01-15 10:31:00 INFO server exited gracefully
```

Check that `banana_farm.db` was created in your project root.

---

## Phase 1 Checklist

- [ ] `go.mod` and `go.sum` exist with all dependencies
- [ ] `.env`, `.env.example`, and `.gitignore` created
- [ ] `config/config.go` loads env vars into a typed struct
- [ ] `config/logger.go` sets up slog with tint (colored dev output)
- [ ] `database/database.go` connects to SQLite and has migration placeholder
- [ ] `main.go` starts server, logs startup info, shuts down gracefully
- [ ] `.air.toml` configured for hot reload
- [ ] Server runs, responds to requests, and shuts down cleanly on Ctrl+C
