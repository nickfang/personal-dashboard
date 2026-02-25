package testutil

import (
	"context"

	"github.com/nickfang/personal-dashboard/services/pollen-collector/internal/client"
	"github.com/nickfang/personal-dashboard/services/pollen-collector/internal/repository"
)

// MockFetcher implements client.Fetcher for testing.
type MockFetcher struct {
	FetchFn func(ctx context.Context, apiKey string, locationID string, lat, long float64) (*client.PollenAPIResponse, error)
}

func (m *MockFetcher) Fetch(ctx context.Context, apiKey string, locationID string, lat, long float64) (*client.PollenAPIResponse, error) {
	return m.FetchFn(ctx, apiKey, locationID, lat, long)
}

// MockWriter implements repository.Writer for testing.
type MockWriter struct {
	SaveRawFn     func(ctx context.Context, snapshot repository.PollenSnapshot) error
	UpdateCacheFn func(ctx context.Context, locationID string, snapshot repository.PollenSnapshot) error
}

func (m *MockWriter) SaveRaw(ctx context.Context, snapshot repository.PollenSnapshot) error {
	return m.SaveRawFn(ctx, snapshot)
}

func (m *MockWriter) UpdateCache(ctx context.Context, locationID string, snapshot repository.PollenSnapshot) error {
	return m.UpdateCacheFn(ctx, locationID, snapshot)
}
