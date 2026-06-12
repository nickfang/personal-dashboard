package testutil

import (
	"context"

	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/api"
	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/repository"
	"github.com/nickfang/personal-dashboard/services/shared"
)

// MockFetcher implements api.Fetcher for testing.
type MockFetcher struct {
	FetchFn func(apiKey string, location shared.Location, horizonHours int) ([]api.ForecastHour, error)
}

func (m *MockFetcher) Fetch(apiKey string, location shared.Location, horizonHours int) ([]api.ForecastHour, error) {
	return m.FetchFn(apiKey, location, horizonHours)
}

// MockWriter implements repository.Writer for testing.
type MockWriter struct {
	SaveRawFn     func(ctx context.Context, run repository.ForecastRun) error
	UpdateCacheFn func(ctx context.Context, locationID string, run repository.ForecastRun) error
}

func (m *MockWriter) SaveRaw(ctx context.Context, run repository.ForecastRun) error {
	return m.SaveRawFn(ctx, run)
}

func (m *MockWriter) UpdateCache(ctx context.Context, locationID string, run repository.ForecastRun) error {
	return m.UpdateCacheFn(ctx, locationID, run)
}
