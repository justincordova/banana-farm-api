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
