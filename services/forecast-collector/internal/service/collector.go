package service

import (
	"context"
	"fmt"
	"time"

	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/api"
	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/repository"
	"github.com/nickfang/personal-dashboard/services/shared"
)

// CollectorService orchestrates the forecast collection flow.
type CollectorService struct {
	fetcher      api.Fetcher
	writer       repository.Writer
	horizonHours int
}

// NewCollectorService creates a new CollectorService with injected dependencies.
func NewCollectorService(fetcher api.Fetcher, writer repository.Writer, horizonHours int) *CollectorService {
	return &CollectorService{fetcher: fetcher, writer: writer, horizonHours: horizonHours}
}

// Collect fetches the forecast for a location, maps it, and writes to storage.
func (s *CollectorService) Collect(ctx context.Context, apiKey string, location shared.Location) error {
	hours, err := s.fetcher.Fetch(apiKey, location, s.horizonHours)
	if err != nil {
		return fmt.Errorf("fetching forecast for %s: %w", location.ID, err)
	}
	points := MapRun(hours)
	if len(points) == 0 {
		return fmt.Errorf("no valid forecast points for %s (%d hours fetched)", location.ID, len(hours))
	}
	run := repository.ForecastRun{
		Location:     location.ID,
		IssuedAt:     time.Now(),
		HorizonHours: s.horizonHours,
		Points:       points,
	}
	if err := s.writer.SaveRaw(ctx, run); err != nil {
		return fmt.Errorf("saving forecast run for %s: %w", location.ID, err)
	}
	if err := s.writer.UpdateCache(ctx, location.ID, run); err != nil {
		return fmt.Errorf("updating forecast cache for %s: %w", location.ID, err)
	}
	return nil
}
