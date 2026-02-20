package shared

import (
	"log/slog"
	"os"
)

// InitLogging configures the global slog logger with JSON output.
// Set DEBUG=true env var for debug-level logging.
func InitLogging() {
	level := slog.LevelInfo
	if os.Getenv("DEBUG") == "true" {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)
}
