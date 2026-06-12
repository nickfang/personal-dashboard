package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/api"
	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/repository"
	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/testutil"
	"github.com/nickfang/personal-dashboard/services/shared"
)

var testLocation = shared.Location{ID: "test-loc", Lat: 30.0, Long: -97.0}

func happyHours() []api.ForecastHour {
	return []api.ForecastHour{validHour()}
}

func TestCollect_SavesRawThenCache(t *testing.T) {
	var savedRun *repository.ForecastRun
	var cachedLocation string
	var cachedRun *repository.ForecastRun

	fetcher := &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location, horizonHours int) ([]api.ForecastHour, error) {
			if horizonHours != 72 {
				t.Errorf("Fetch horizonHours = %d, want 72", horizonHours)
			}
			return happyHours(), nil
		},
	}
	writer := &testutil.MockWriter{
		SaveRawFn: func(ctx context.Context, run repository.ForecastRun) error {
			savedRun = &run
			return nil
		},
		UpdateCacheFn: func(ctx context.Context, locationID string, run repository.ForecastRun) error {
			cachedLocation = locationID
			cachedRun = &run
			return nil
		},
	}

	collector := NewCollectorService(fetcher, writer, 72)
	if err := collector.Collect(context.Background(), "test-key", testLocation); err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}

	if savedRun == nil {
		t.Fatal("SaveRaw was not called")
	}
	if savedRun.Location != "test-loc" || savedRun.HorizonHours != 72 {
		t.Errorf("saved run = %q/%d, want test-loc/72", savedRun.Location, savedRun.HorizonHours)
	}
	if len(savedRun.Points) != 1 {
		t.Fatalf("saved run has %d points, want 1", len(savedRun.Points))
	}
	if cachedLocation != "test-loc" {
		t.Errorf("UpdateCache locationID = %q, want test-loc", cachedLocation)
	}
	if cachedRun == nil || len(cachedRun.Points) != 1 {
		t.Fatal("UpdateCache did not receive the mapped run")
	}
}

func TestCollect_FetchErrorPropagates(t *testing.T) {
	fetcher := &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location, horizonHours int) ([]api.ForecastHour, error) {
			return nil, fmt.Errorf("API unavailable")
		},
	}
	writer := &testutil.MockWriter{
		SaveRawFn: func(ctx context.Context, run repository.ForecastRun) error {
			t.Fatal("SaveRaw should not be called when fetch fails")
			return nil
		},
		UpdateCacheFn: func(ctx context.Context, locationID string, run repository.ForecastRun) error {
			return nil
		},
	}

	collector := NewCollectorService(fetcher, writer, 72)
	if err := collector.Collect(context.Background(), "test-key", testLocation); err == nil {
		t.Fatal("Collect() should propagate fetch errors")
	}
}

func TestCollect_AllPointsInvalidFails(t *testing.T) {
	bad := validHour()
	bad.AirPressure.MeanSeaLevelMillibars = 0
	fetcher := &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location, horizonHours int) ([]api.ForecastHour, error) {
			return []api.ForecastHour{bad}, nil
		},
	}
	writer := &testutil.MockWriter{
		SaveRawFn: func(ctx context.Context, run repository.ForecastRun) error {
			t.Fatal("SaveRaw should not be called when no points are valid")
			return nil
		},
		UpdateCacheFn: func(ctx context.Context, locationID string, run repository.ForecastRun) error {
			return nil
		},
	}

	collector := NewCollectorService(fetcher, writer, 72)
	if err := collector.Collect(context.Background(), "test-key", testLocation); err == nil {
		t.Fatal("Collect() should fail when every forecast hour is invalid")
	}
}

func TestCollect_SaveRawErrorPropagates(t *testing.T) {
	fetcher := &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location, horizonHours int) ([]api.ForecastHour, error) {
			return happyHours(), nil
		},
	}
	writer := &testutil.MockWriter{
		SaveRawFn: func(ctx context.Context, run repository.ForecastRun) error {
			return fmt.Errorf("firestore unavailable")
		},
		UpdateCacheFn: func(ctx context.Context, locationID string, run repository.ForecastRun) error {
			t.Fatal("UpdateCache should not be called when SaveRaw fails")
			return nil
		},
	}

	collector := NewCollectorService(fetcher, writer, 72)
	if err := collector.Collect(context.Background(), "test-key", testLocation); err == nil {
		t.Fatal("Collect() should propagate SaveRaw errors")
	}
}
