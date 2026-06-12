package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/api"
	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/repository"
	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/service"
	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/testutil"
	"github.com/nickfang/personal-dashboard/services/shared"
)

func happyHour() api.ForecastHour {
	var h api.ForecastHour
	h.Interval.StartTime = time.Now().Truncate(time.Hour)
	h.AirPressure.MeanSeaLevelMillibars = 1013.25
	h.Temperature.Degrees = 25.0
	return h
}

func happyFetcher() *testutil.MockFetcher {
	return &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location, horizonHours int) ([]api.ForecastHour, error) {
			return []api.ForecastHour{happyHour()}, nil
		},
	}
}

func happyWriter() *testutil.MockWriter {
	return &testutil.MockWriter{
		SaveRawFn: func(ctx context.Context, run repository.ForecastRun) error {
			return nil
		},
		UpdateCacheFn: func(ctx context.Context, locationID string, run repository.ForecastRun, merge repository.MergeFunc) error {
			return nil
		},
	}
}

func failingFetcher() *testutil.MockFetcher {
	return &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location, horizonHours int) ([]api.ForecastHour, error) {
			return nil, fmt.Errorf("API unavailable")
		},
	}
}

var testLocations = []shared.Location{
	{ID: "house-nick", Lat: 30.0, Long: -97.0},
	{ID: "house-nita", Lat: 31.0, Long: -98.0},
}

func TestCollectAll_AllLocationsSucceed(t *testing.T) {
	collector := service.NewCollectorService(happyFetcher(), happyWriter(), 72, service.DefaultAlertConfig())

	err := collectAll(context.Background(), "test-key", collector, testLocations)
	if err != nil {
		t.Fatalf("collectAll() returned error: %v", err)
	}
}

func TestCollectAll_PartialFailure(t *testing.T) {
	callCount := 0
	fetcher := &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location, horizonHours int) ([]api.ForecastHour, error) {
			callCount++
			if callCount == 1 {
				return nil, fmt.Errorf("API unavailable")
			}
			return []api.ForecastHour{happyHour()}, nil
		},
	}

	collector := service.NewCollectorService(fetcher, happyWriter(), 72, service.DefaultAlertConfig())

	err := collectAll(context.Background(), "test-key", collector, testLocations)
	if err != nil {
		t.Fatalf("collectAll() should succeed with partial failures, got: %v", err)
	}
}

func TestCollectAll_AllLocationsFail(t *testing.T) {
	collector := service.NewCollectorService(failingFetcher(), happyWriter(), 72, service.DefaultAlertConfig())

	err := collectAll(context.Background(), "test-key", collector, testLocations)
	if err == nil {
		t.Fatal("collectAll() should return error when all locations fail")
	}
}

func TestCollectAll_EmptyLocations(t *testing.T) {
	collector := service.NewCollectorService(happyFetcher(), happyWriter(), 72, service.DefaultAlertConfig())

	err := collectAll(context.Background(), "test-key", collector, []shared.Location{})
	if err == nil {
		t.Fatal("collectAll() should return error when no locations are provided")
	}
}

func TestEnvFloat(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  float64
	}{
		{"unset", "", 5.0},
		{"valid", "7.5", 7.5},
		{"invalid", "abc", 5.0},
		{"negative", "-2", 5.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				t.Setenv("TEST_THRESHOLD", tt.value)
			}
			if got := envFloat("TEST_THRESHOLD", 5.0); got != tt.want {
				t.Errorf("envFloat(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestEnvInt(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int
	}{
		{"unset", "", 72},
		{"valid", "120", 120},
		{"invalid", "abc", 72},
		{"negative", "-5", 72},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				t.Setenv("TEST_HORIZON", tt.value)
			}
			if got := envInt("TEST_HORIZON", 72); got != tt.want {
				t.Errorf("envInt(%q) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}
}
