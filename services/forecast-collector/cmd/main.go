package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/api"
	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/repository"
	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/service"
	"github.com/nickfang/personal-dashboard/services/shared"
)

const defaultHorizonHours = 72

func main() {
	shared.InitLogging()

	if err := godotenv.Load(); err != nil {
		slog.Debug("No .env file found, using system environment variables", "error", err)
	}
	apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	projectID := os.Getenv("GCP_PROJECT_ID")
	if apiKey == "" || projectID == "" {
		slog.Error("Missing required env vars", "vars", "GOOGLE_MAPS_API_KEY, GCP_PROJECT_ID")
		os.Exit(1)
	}
	horizonHours := envInt("FORECAST_HORIZON_HOURS", defaultHorizonHours)
	alertCfg := service.DefaultAlertConfig()
	alertCfg.DropThresholdMb = envFloat("PRESSURE_DROP_MB", alertCfg.DropThresholdMb)
	alertCfg.SevereThresholdMb = envFloat("PRESSURE_SEVERE_MB", alertCfg.SevereThresholdMb)
	alertCfg.WindowHours = envInt("PRESSURE_WINDOW_HOURS", alertCfg.WindowHours)

	ctx := context.Background()
	writer, err := repository.NewFirestoreWriter(ctx, projectID)
	if err != nil {
		slog.Error("Failed to create firestore writer", "error", err)
		os.Exit(1)
	}
	defer writer.Close()

	httpClient := &http.Client{Timeout: 15 * time.Second}
	fetcher := api.New(httpClient)
	collector := service.NewCollectorService(fetcher, writer, horizonHours, alertCfg)
	if err := collectAll(ctx, apiKey, collector, shared.Locations); err != nil {
		slog.Error("Collection failed", "error", err)
		os.Exit(1)
	}
}

// envInt reads an integer env var, falling back to a default when unset or invalid.
func envInt(name string, fallback int) int {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		slog.Warn("Invalid env var, using default", "name", name, "value", raw, "default", fallback)
		return fallback
	}
	return v
}

// envFloat reads a float env var, falling back to a default when unset or invalid.
func envFloat(name string, fallback float64) float64 {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 {
		slog.Warn("Invalid env var, using default", "name", name, "value", raw, "default", fallback)
		return fallback
	}
	return v
}

func collectAll(ctx context.Context, apiKey string, collector *service.CollectorService, locations []shared.Location) error {
	if len(locations) == 0 {
		return fmt.Errorf("no locations provided")
	}
	successCount := 0
	for _, loc := range locations {
		err := collector.Collect(ctx, apiKey, loc)
		if err != nil {
			slog.Error("Failed to collect forecast", "location", loc.ID, "error", err)
			continue
		}
		successCount++
		slog.Info("Processed forecast", "location", loc.ID)
	}
	if successCount == 0 {
		return fmt.Errorf("all %d locations failed", len(locations))
	}
	slog.Info("Forecast collection complete", "succeeded", successCount, "total", len(locations))
	return nil
}
