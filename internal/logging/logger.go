package logging

import (
	"context"
	"io"
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
// InitLogger initializes global Logger based on provided or env log level & format.
func InitLogger(logLevel string) {
	if logLevel == "" {
		if logLevel = os.Getenv("LOG_LEVEL"); logLevel == "" {
			logLevel = "info"
		}
	}

	ll := strings.ToLower(logLevel)
	level := slog.LevelInfo
	if len(ll) > 0 {
		switch ll[0] {
		case 'd':
			level = slog.LevelDebug
		case 'w':
			level = slog.LevelWarn
		case 'e':
			level = slog.LevelError
		}
	}

	format := strings.ToLower(os.Getenv("LOG_FORMAT")) // "text"|"json"|"color"
	if format == "" && os.Getenv("NO_COLOR") == "" {
		// Default to color if terminal supports it (NO_COLOR not set)
		format = "color"
	}

	var handler slog.Handler
	switch format {
	case "json":
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	case "color":
		handler = newColorHandler(os.Stderr, level, os.Getenv("NO_COLOR") != "")
	default:
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	}

	Logger = slog.New(handler)
}

// customColorHandler implements a minimal pretty colored handler.
// Output format: LEVEL: message [k=v ...]\n
type customColorHandler struct {
	w       io.Writer
	level   slog.Level
	noColor bool
	attrs   []slog.Attr
}

func newColorHandler(w io.Writer, lvl slog.Level, noColor bool) *customColorHandler {
	return &customColorHandler{w: w, level: lvl, noColor: noColor}
}

func (h *customColorHandler) Enabled(_ context.Context, lvl slog.Level) bool {
	return lvl >= h.level
}

func (h *customColorHandler) Handle(_ context.Context, r slog.Record) error {
	// Color level label
	lvl := strings.ToUpper(r.Level.String())
	if !h.noColor {
		switch lvl {
		case "DEBUG":
			lvl = "\x1b[90m" + lvl + "\x1b[0m"
		case "INFO":
			lvl = "\x1b[36m" + lvl + "\x1b[0m"
		case "WARN", "WARNING":
			lvl = "\x1b[33m" + lvl + "\x1b[0m"
		case "ERROR":
			lvl = "\x1b[31m" + lvl + "\x1b[0m"
		}
	}

	var b strings.Builder
	b.WriteString(lvl)
	b.WriteString(": ")
	b.WriteString(r.Message)

	// Combine stored attrs + record attrs
	writeAttr := func(a slog.Attr) {
		// Skip level/time; already represented
		if a.Key == slog.LevelKey || a.Key == slog.TimeKey || a.Key == "msg" {
			return
		}
		b.WriteString(" ")
		b.WriteString(a.Key)
		b.WriteString("=")
		b.WriteString(valueString(a.Value))
	}
	for _, a := range h.attrs {
		writeAttr(a)
	}
	r.Attrs(func(a slog.Attr) bool { writeAttr(a); return true })

	b.WriteByte('\n')
	_, err := io.WriteString(h.w, b.String())
	return err
}

func (h *customColorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Copy existing slice
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)
	return &customColorHandler{w: h.w, level: h.level, noColor: h.noColor, attrs: newAttrs}
}

func (h *customColorHandler) WithGroup(name string) slog.Handler {
	// Groups ignored for simplicity; could prefix keys if desired.
	return h
}

// valueString renders slog.Value succinctly (no quotes around plain strings to enable color codes in message if present).
func valueString(v slog.Value) string {
	if v.Kind() == slog.KindString {
		return v.String()
	}
	return v.String()
}
