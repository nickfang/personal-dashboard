package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/nickfang/personal-dashboard/services/shared"
	"github.com/nickfang/personal-dashboard/services/weather-collector/internal/api"
	"github.com/nickfang/personal-dashboard/services/weather-collector/internal/repository"
	"github.com/nickfang/personal-dashboard/services/weather-collector/internal/service"
	"github.com/nickfang/personal-dashboard/services/weather-collector/internal/testutil"
)

func happyFetcher() *testutil.MockFetcher {
	return &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location) (*api.WeatherAPIResponse, error) {
			resp := &api.WeatherAPIResponse{}
			resp.AirPressure.MeanSeaLevelMillibars = 1013.25
			resp.Temperature.Degrees = 25.0
			return resp, nil
		},
	}
}

func happyWriter() *testutil.MockWriter {
	return &testutil.MockWriter{
		SaveRawFn: func(ctx context.Context, wp repository.WeatherPoint) error {
			return nil
		},
		UpdateCacheFn: func(ctx context.Context, locationID string, wp repository.WeatherPoint, analyze repository.AnalyzeFunc) error {
			return nil
		},
	}
}

func failingFetcher() *testutil.MockFetcher {
	return &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location) (*api.WeatherAPIResponse, error) {
			return nil, fmt.Errorf("API unavailable")
		},
	}
}

var testLocations = []shared.Location{
	{ID: "house-nick", Lat: 30.0, Long: -97.0},
	{ID: "house-nita", Lat: 31.0, Long: -98.0},
}

func TestCollectAll_AllLocationsSucceed(t *testing.T) {
	collector := service.NewCollectorService(happyFetcher(), happyWriter())

	err := collectAll(context.Background(), "test-key", collector, testLocations)
	if err != nil {
		t.Fatalf("collectAll() returned error: %v", err)
	}
}

func TestCollectAll_PartialFailure(t *testing.T) {
	callCount := 0
	fetcher := &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location) (*api.WeatherAPIResponse, error) {
			callCount++
			if callCount == 1 {
				return nil, fmt.Errorf("API unavailable")
			}
			resp := &api.WeatherAPIResponse{}
			resp.AirPressure.MeanSeaLevelMillibars = 1013.25
			resp.Temperature.Degrees = 25.0
			return resp, nil
		},
	}

	collector := service.NewCollectorService(fetcher, happyWriter())

	err := collectAll(context.Background(), "test-key", collector, testLocations)
	if err != nil {
		t.Fatalf("collectAll() should succeed with partial failures, got: %v", err)
	}
}

func TestCollectAll_AllLocationsFail(t *testing.T) {
	collector := service.NewCollectorService(failingFetcher(), happyWriter())

	err := collectAll(context.Background(), "test-key", collector, testLocations)
	if err == nil {
		t.Fatal("collectAll() should return error when all locations fail")
	}
}

func TestCollectAll_EmptyLocations(t *testing.T) {
	collector := service.NewCollectorService(happyFetcher(), happyWriter())

	err := collectAll(context.Background(), "test-key", collector, []shared.Location{})
	if err == nil {
		t.Fatal("collectAll() should return error when no locations are provided")
	}
}
