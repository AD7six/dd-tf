package logging

import (
	"log/slog"
	"os"
)

// Logger is the global logger instance used throughout the application.
var Logger *slog.Logger

func init() {
	InitLogger("")
}

// initLogger initializes the global logger with appropriate settings.
// Log level is controlled by LOG_LEVEL environment variable (debug, info, warn, error).
// If LOG_LEVEL is not set, defaults to info level.
func InitLogger(logLevel string) {
	if logLevel == "" {
		if logLevel = os.Getenv("LOG_LEVEL"); logLevel == "" {
			logLevel = "info"
		}
	}

	level := slog.LevelInfo

	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
}
