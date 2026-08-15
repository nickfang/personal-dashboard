package testutil

import (
	"context"

	"github.com/nickfang/personal-dashboard/services/notifier/internal/repository"
)

// MockStore implements repository.Store for testing. Unset funcs return
// zero values, so a test supplies only the reads it cares about.
type MockStore struct {
	ReadObservationFn func(ctx context.Context, locationID string) (*repository.WeatherCacheDoc, error)
	ReadForecastFn    func(ctx context.Context, locationID string) (*repository.ForecastCacheDoc, error)
}

func (m *MockStore) ReadObservation(ctx context.Context, locationID string) (*repository.WeatherCacheDoc, error) {
	if m.ReadObservationFn == nil {
		return nil, nil
	}
	return m.ReadObservationFn(ctx, locationID)
}

func (m *MockStore) ReadForecast(ctx context.Context, locationID string) (*repository.ForecastCacheDoc, error) {
	if m.ReadForecastFn == nil {
		return &repository.ForecastCacheDoc{}, nil
	}
	return m.ReadForecastFn(ctx, locationID)
}
