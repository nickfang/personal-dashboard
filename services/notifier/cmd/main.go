package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/nickfang/personal-dashboard/services/notifier/internal/repository"
	"github.com/nickfang/personal-dashboard/services/notifier/internal/service"
	"github.com/nickfang/personal-dashboard/services/shared"
)

func main() {
	shared.InitLogging()

	if err := godotenv.Load(); err != nil {
		slog.Debug("No .env file found, using system environment variables", "error", err)
	}
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		slog.Error("Missing required env vars", "vars", "GCP_PROJECT_ID")
		os.Exit(1)
	}

	ctx := context.Background()
	store, err := repository.NewFirestoreStore(ctx, projectID)
	if err != nil {
		slog.Error("Failed to create firestore store", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	// Pinned once so every location in a run shares an evaluation time; a run
	// that straddles the hour must not split across two.
	now := time.Now()

	notifier := service.NewNotifierService(store)
	if err := observeAll(ctx, notifier, shared.Locations, now); err != nil {
		slog.Error("Observation failed", "error", err)
		os.Exit(1)
	}
}

// observeAll is partial-failure tolerant, matching the collectors: a location
// that cannot be read is logged and skipped, and the run only fails when
// every location does.
func observeAll(ctx context.Context, notifier *service.NotifierService, locations []shared.Location, now time.Time) error {
	if len(locations) == 0 {
		return fmt.Errorf("no locations provided")
	}
	successCount := 0
	for _, loc := range locations {
		if _, err := notifier.Observe(ctx, loc, now); err != nil {
			slog.Error("Failed to observe location", "location", loc.ID, "error", err)
			continue
		}
		successCount++
	}
	if successCount == 0 {
		return fmt.Errorf("all %d locations failed", len(locations))
	}
	slog.Info("Observation complete", "succeeded", successCount, "total", len(locations))
	return nil
}
