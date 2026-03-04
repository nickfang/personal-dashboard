package testutil

import (
	"context"

	"github.com/nickfang/personal-dashboard/services/shared"
	"github.com/nickfang/personal-dashboard/services/weather-collector/internal/api"
	"github.com/nickfang/personal-dashboard/services/weather-collector/internal/repository"
)

// MockFetcher implements api.Fetcher for testing.
type MockFetcher struct {
	FetchFn func(apiKey string, location shared.Location) (*api.WeatherAPIResponse, error)
}

func (m *MockFetcher) Fetch(apiKey string, location shared.Location) (*api.WeatherAPIResponse, error) {
	return m.FetchFn(apiKey, location)
}

// MockWriter implements repository.Writer for testing.
type MockWriter struct {
	SaveRawFn     func(ctx context.Context, wp repository.WeatherPoint) error
	UpdateCacheFn func(ctx context.Context, locationID string, wp repository.WeatherPoint, analyze repository.AnalyzeFunc) error
}

func (m *MockWriter) SaveRaw(ctx context.Context, wp repository.WeatherPoint) error {
	return m.SaveRawFn(ctx, wp)
}

func (m *MockWriter) UpdateCache(ctx context.Context, locationID string, wp repository.WeatherPoint, analyze repository.AnalyzeFunc) error {
	return m.UpdateCacheFn(ctx, locationID, wp, analyze)
}
