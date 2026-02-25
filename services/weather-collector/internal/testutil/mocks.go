package testutil

import (
	"context"

	"github.com/nickfang/personal-dashboard/services/weather-collector/internal/client"
	"github.com/nickfang/personal-dashboard/services/weather-collector/internal/repository"
)

// MockFetcher implements client.Fetcher for testing.
type MockFetcher struct {
	FetchFn func(ctx context.Context, apiKey string, locationID string, lat, long float64) (*client.WeatherAPIResponse, error)
}

func (m *MockFetcher) Fetch(ctx context.Context, apiKey string, locationID string, lat, long float64) (*client.WeatherAPIResponse, error) {
	return m.FetchFn(ctx, apiKey, locationID, lat, long)
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
