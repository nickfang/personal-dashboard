package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/nickfang/personal-dashboard/services/shared"
	"github.com/nickfang/personal-dashboard/services/weather-collector/internal/api"
	"github.com/nickfang/personal-dashboard/services/weather-collector/internal/repository"
	"github.com/nickfang/personal-dashboard/services/weather-collector/internal/service"
)

func main() {
	// Setup Structured Logging
	shared.InitLogging()

	// Load .env file if it exists (local development)
	if err := godotenv.Load(); err != nil {
		slog.Debug("No .env file found, using system environment variables", "error", err)
	}
	apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	projectID := os.Getenv("GCP_PROJECT_ID")
	if apiKey == "" || projectID == "" {
		slog.Error("Missing required env vars", "vars", "GOOGLE_MAPS_API_KEY, GCP_PROJECT_ID")
		os.Exit(1)
	}

	ctx := context.Background()
	writer, err := repository.NewFirestoreWriter(ctx, projectID)
	if err != nil {
		slog.Error("Failed to create firestore writer", "error", err)
		os.Exit(1)
	}
	defer writer.Close()

	httpClient := &http.Client{Timeout: 15 * time.Second}
	fetcher := api.New(httpClient)
	collector := service.NewCollectorService(fetcher, writer)
	if err := collectAll(ctx, apiKey, collector, shared.Locations); err != nil {
		slog.Error("Collection failed", "error", err)
		os.Exit(1)
	}
}

func collectAll(ctx context.Context, apiKey string, collector *service.CollectorService, locations []shared.Location) error {
	if len(locations) == 0 {
		return fmt.Errorf("no locations provided")
	}
	successCount := 0
	for _, loc := range locations {
		err := collector.Collect(ctx, apiKey, loc)
		if err != nil {
			slog.Error("Failed to collect weather", "location", loc.ID, "error", err)
			continue
		}
		successCount++
		slog.Info("Processed weather", "location", loc.ID)
	}
	if successCount == 0 {
		return fmt.Errorf("all locations failed")
	}
	slog.Info("Weather collection complete", "succeeded", successCount, "total", len(locations))
	return nil
}
