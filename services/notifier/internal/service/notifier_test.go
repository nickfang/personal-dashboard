package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nickfang/personal-dashboard/services/notifier/internal/repository"
	"github.com/nickfang/personal-dashboard/services/notifier/internal/testutil"
	"github.com/nickfang/personal-dashboard/services/shared"
)

var testLocation = shared.Location{ID: "house-nick", Lat: 30.0, Long: -97.0}

func TestObserve_ReadsBothCaches(t *testing.T) {
	var forecastFor, observationFor string
	store := &testutil.MockStore{
		ReadForecastFn: func(ctx context.Context, id string) (*repository.ForecastCacheDoc, error) {
			forecastFor = id
			return &repository.ForecastCacheDoc{IssuedAt: testNow, Points: forecastPoints(1013, 1012)}, nil
		},
		ReadObservationFn: func(ctx context.Context, id string) (*repository.WeatherCacheDoc, error) {
			observationFor = id
			return observedAt(1013, time.Minute), nil
		},
	}

	if _, err := NewNotifierService(store).Observe(context.Background(), testLocation, testNow); err != nil {
		t.Fatalf("Observe() returned error: %v", err)
	}
	if forecastFor != "house-nick" || observationFor != "house-nick" {
		t.Errorf("read forecast for %q and observation for %q, want both house-nick", forecastFor, observationFor)
	}
}

func TestObserve_MissingObservationSucceeds(t *testing.T) {
	// weather-collector may simply not have run for this location. That is a
	// finding worth recording, not a failure.
	store := &testutil.MockStore{
		ReadForecastFn: func(ctx context.Context, id string) (*repository.ForecastCacheDoc, error) {
			return &repository.ForecastCacheDoc{IssuedAt: testNow, Points: forecastPoints(1013)}, nil
		},
		ReadObservationFn: func(ctx context.Context, id string) (*repository.WeatherCacheDoc, error) {
			return nil, nil
		},
	}

	if _, err := NewNotifierService(store).Observe(context.Background(), testLocation, testNow); err != nil {
		t.Fatalf("Observe() should tolerate a missing observation, got: %v", err)
	}
}

func TestObserve_MissingForecastFails(t *testing.T) {
	store := &testutil.MockStore{
		ReadForecastFn: func(ctx context.Context, id string) (*repository.ForecastCacheDoc, error) {
			return nil, fmt.Errorf("no forecast cache document for %s", id)
		},
		ReadObservationFn: func(ctx context.Context, id string) (*repository.WeatherCacheDoc, error) {
			t.Fatal("observation should not be read when the forecast is missing")
			return nil, nil
		},
	}

	if _, err := NewNotifierService(store).Observe(context.Background(), testLocation, testNow); err == nil {
		t.Fatal("Observe() should fail when there is no forecast to observe against")
	}
}

func TestObserve_ObservationReadErrorPropagates(t *testing.T) {
	store := &testutil.MockStore{
		ReadForecastFn: func(ctx context.Context, id string) (*repository.ForecastCacheDoc, error) {
			return &repository.ForecastCacheDoc{IssuedAt: testNow}, nil
		},
		ReadObservationFn: func(ctx context.Context, id string) (*repository.WeatherCacheDoc, error) {
			return nil, fmt.Errorf("firestore unavailable")
		},
	}

	// A read failure is different from an absent document: the first means
	// something is broken, the second means nothing has run yet.
	if _, err := NewNotifierService(store).Observe(context.Background(), testLocation, testNow); err == nil {
		t.Fatal("Observe() should propagate a failed observation read")
	}
}
