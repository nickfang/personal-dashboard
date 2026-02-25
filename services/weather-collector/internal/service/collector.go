package service

import (
	"context"
	"time"

	"github.com/nickfang/personal-dashboard/services/weather-collector/internal/client"
	"github.com/nickfang/personal-dashboard/services/weather-collector/internal/repository"
)

const (
	DeltaTolerance      = 45 * time.Minute
	DeltaNoiseThreshold = 0.5 // mb
)

// CollectorService orchestrates the weather collection flow.
type CollectorService struct {
	fetcher client.Fetcher
	writer  repository.Writer
}

// NewCollectorService creates a new CollectorService with injected dependencies.
func NewCollectorService(fetcher client.Fetcher, writer repository.Writer) *CollectorService {
	return &CollectorService{fetcher: fetcher, writer: writer}
}

// Collect fetches weather data for a location, maps it, and writes to storage.
// TODO: implement — fetch → validate → map → save raw → update cache with AnalyzeFunc callback.
func (s *CollectorService) Collect(ctx context.Context, apiKey, locationID string, lat, long float64) error {
	return nil
}

// MapToWeatherPoint converts an API response into a WeatherPoint for storage.
// Returns an error if the data is invalid (e.g., 0.0 pressure).
// TODO: implement.
func MapToWeatherPoint(locationID string, data client.WeatherAPIResponse) (*repository.WeatherPoint, error) {
	return nil, nil
}

// CalculatePressureStats computes barometric pressure deltas and trend from history.
// Used as the AnalyzeFunc callback passed to repository.Writer.UpdateCache.
// TODO: implement.
func CalculatePressureStats(history []repository.PressurePoint) repository.PressureStats {
	return repository.PressureStats{}
}
