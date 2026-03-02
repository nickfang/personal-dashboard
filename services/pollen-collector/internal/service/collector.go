package service

import (
	"context"
	"fmt"
	"time"

	"github.com/nickfang/personal-dashboard/services/pollen-collector/internal/api"
	"github.com/nickfang/personal-dashboard/services/pollen-collector/internal/repository"
	"github.com/nickfang/personal-dashboard/services/shared"
)

// CollectorService orchestrates the pollen collection flow.
type CollectorService struct {
	fetcher api.Fetcher
	writer  repository.Writer
}

// NewCollectorService creates a new CollectorService with injected dependencies.
func NewCollectorService(fetcher api.Fetcher, writer repository.Writer) *CollectorService {
	return &CollectorService{fetcher: fetcher, writer: writer}
}

// Collect fetches pollen data for a location, maps it, and writes to storage.
func (s *CollectorService) Collect(ctx context.Context, apiKey string, location shared.Location) error {
	apiResp, err := s.fetcher.Fetch(apiKey, location)
	if err != nil {
		return fmt.Errorf("fetching pollen data for %s: %w", location.ID, err)
	}

	snapshot := MapToSnapshot(location.ID, apiResp)

	if err := s.writer.SaveRaw(ctx, snapshot); err != nil {
		return fmt.Errorf("saving raw pollen data for %s: %w", location.ID, err)
	}

	if err := s.writer.UpdateCache(ctx, location.ID, snapshot); err != nil {
		return fmt.Errorf("updating pollen cache for %s: %w", location.ID, err)
	}

	return nil
}

// MapToSnapshot converts an API response into a PollenSnapshot for storage.
// It computes the overall summary (highest UPI across the 3 pollen types).
func MapToSnapshot(locationID string, apiResp *api.PollenAPIResponse) repository.PollenSnapshot {
	today := apiResp.DailyInfo[0]

	snapshot := repository.PollenSnapshot{
		LocationID:  locationID,
		CollectedAt: time.Now(),
	}

	// Map pollen types
	for _, t := range today.PollenTypeInfo {
		snapshot.Types = append(snapshot.Types, repository.StoredPollenType{
			Code:     t.Code,
			Index:    t.IndexInfo.Value,
			Category: t.IndexInfo.Category,
			InSeason: t.InSeason,
		})
	}

	// Map plants
	for _, p := range today.PlantInfo {
		snapshot.Plants = append(snapshot.Plants, repository.StoredPollenPlant{
			Code:        p.Code,
			DisplayName: p.DisplayName,
			Index:       p.IndexInfo.Value,
			Category:    p.IndexInfo.Category,
			InSeason:    p.InSeason,
		})
	}

	// Compute overall summary: find the highest UPI across the 3 types
	for _, t := range snapshot.Types {
		if t.Index > snapshot.OverallIndex {
			snapshot.OverallIndex = t.Index
			snapshot.OverallCategory = t.Category
			snapshot.DominantType = t.Code
		}
	}

	return snapshot
}
