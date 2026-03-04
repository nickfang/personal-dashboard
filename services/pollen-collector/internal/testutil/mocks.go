package testutil

import (
	"context"

	"github.com/nickfang/personal-dashboard/services/pollen-collector/internal/api"
	"github.com/nickfang/personal-dashboard/services/pollen-collector/internal/repository"
	"github.com/nickfang/personal-dashboard/services/shared"
)

// MockFetcher implements api.Fetcher for testing.
type MockFetcher struct {
	FetchFn func(apiKey string, location shared.Location) (*api.PollenAPIResponse, error)
}

func (m *MockFetcher) Fetch(apiKey string, location shared.Location) (*api.PollenAPIResponse, error) {
	return m.FetchFn(apiKey, location)
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
