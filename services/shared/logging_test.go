package shared

import (
	"log/slog"
	"os"
	"testing"
)

func TestInitLogging_DoesNotPanic(t *testing.T) {
	// InitLogging should not panic regardless of DEBUG env var state.
	InitLogging()
}

func TestInitLogging_SetsGlobalLogger(t *testing.T) {
	before := slog.Default()
	InitLogging()
	after := slog.Default()

	// The global logger should have been replaced.
	if before == after {
		t.Error("InitLogging() did not replace the default slog logger")
	}
}

func TestInitLogging_DebugMode(t *testing.T) {
	os.Setenv("DEBUG", "true")
	defer os.Unsetenv("DEBUG")

	InitLogging()

	// Debug-level messages should be enabled.
	if !slog.Default().Enabled(nil, slog.LevelDebug) {
		t.Error("expected debug level to be enabled when DEBUG=true")
	}
}

func TestInitLogging_DefaultInfoLevel(t *testing.T) {
	os.Unsetenv("DEBUG")

	InitLogging()

	// Debug should be disabled at default info level.
	if slog.Default().Enabled(nil, slog.LevelDebug) {
		t.Error("expected debug level to be disabled when DEBUG is not set")
	}
	// Info should be enabled.
	if !slog.Default().Enabled(nil, slog.LevelInfo) {
		t.Error("expected info level to be enabled at default")
	}
}
