package service

import (
	"context"

	"github.com/nickfang/personal-dashboard/services/pollen-collector/internal/client"
	"github.com/nickfang/personal-dashboard/services/pollen-collector/internal/repository"
)

// CollectorService orchestrates the pollen collection flow.
type CollectorService struct {
	fetcher client.Fetcher
	writer  repository.Writer
}

// NewCollectorService creates a new CollectorService with injected dependencies.
func NewCollectorService(fetcher client.Fetcher, writer repository.Writer) *CollectorService {
	return &CollectorService{fetcher: fetcher, writer: writer}
}

// Collect fetches pollen data for a location, maps it, and writes to storage.
// TODO: implement — fetch → map → save raw → update cache.
func (s *CollectorService) Collect(ctx context.Context, apiKey, locationID string, lat, long float64) error {
	return nil
}

// MapToSnapshot converts an API response into a PollenSnapshot for storage.
// It computes the overall summary (highest UPI across the 3 pollen types).
// TODO: implement.
func MapToSnapshot(locationID string, apiResp *client.PollenAPIResponse) repository.PollenSnapshot {
	return repository.PollenSnapshot{}
}
