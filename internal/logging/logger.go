package logging

import (
	"log/slog"
	"os"
)

// Logger is the global logger instance used throughout the application.
var Logger *slog.Logger

func init() {
	initLogger()
}

// initLogger initializes the global logger with appropriate settings.
// Debug level is enabled when DEBUG environment variable is set.
func initLogger() {
	level := slog.LevelInfo
	if os.Getenv("DEBUG") != "" {
		level = slog.LevelDebug
	}

	Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
}

// ReinitLogger reinitializes the logger. This is useful when the DEBUG
// environment variable is set after package initialization (e.g., via --verbose flag).
func ReinitLogger() {
	initLogger()
}
