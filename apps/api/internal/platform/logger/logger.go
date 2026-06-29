package logger

import (
	"log/slog"
	"os"
)

// New builds a slog.Logger configured with the given level, format, service
// name, and environment. format is "json" or "text"; level is one of
// debug|info|warn|error.
func New(level, format, service, env string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}

	var handler slog.Handler
	if format == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler).With(
		slog.String("service", service),
		slog.String("env", env),
	)
}
