package logging

import (
	"log/slog"
	"os"
	"strings"
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

	logLevel = strings.ToLower(logLevel)

	// Default to info level logs
	level := slog.LevelInfo

	// Override if debug, warning or error logs are requested
	if len(logLevel) > 0 {
		switch logLevel[0] {
		case 'd':
			level = slog.LevelDebug
		case 'w':
			level = slog.LevelWarn
		case 'e':
			level = slog.LevelError
		}
	}

	// Select handler based on LOG_FORMAT
	format := os.Getenv("LOG_FORMAT") // supported: "json", "text" (default)

	var handler slog.Handler
	switch strings.ToLower(format) {

	case "json":
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	default:
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	}

	Logger = slog.New(handler)
}
