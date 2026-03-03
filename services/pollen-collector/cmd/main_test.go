package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/nickfang/personal-dashboard/services/pollen-collector/internal/api"
	"github.com/nickfang/personal-dashboard/services/pollen-collector/internal/repository"
	"github.com/nickfang/personal-dashboard/services/pollen-collector/internal/service"
	"github.com/nickfang/personal-dashboard/services/pollen-collector/internal/testutil"
	"github.com/nickfang/personal-dashboard/services/shared"
)

func happyFetcher() *testutil.MockFetcher {
	return &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location) (*api.PollenAPIResponse, error) {
			return &api.PollenAPIResponse{
				DailyInfo: []api.DailyInfo{{
					PollenTypeInfo: []api.PollenTypeInfo{
						{Code: "TREE", InSeason: true, IndexInfo: api.IndexInfo{Value: 3, Category: "Moderate"}},
					},
				}},
			}, nil
		},
	}
}

func happyWriter() *testutil.MockWriter {
	return &testutil.MockWriter{
		SaveRawFn: func(ctx context.Context, snapshot repository.PollenSnapshot) error {
			return nil
		},
		UpdateCacheFn: func(ctx context.Context, locationID string, snapshot repository.PollenSnapshot) error {
			return nil
		},
	}
}

func failingFetcher() *testutil.MockFetcher {
	return &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location) (*api.PollenAPIResponse, error) {
			return nil, fmt.Errorf("API unavailable")
		},
	}
}

var testLocations = []shared.Location{
	{ID: "house-nick", Lat: 30.0, Long: -97.0},
	{ID: "house-nita", Lat: 31.0, Long: -98.0},
}

func TestRun_AllLocationsSucceed(t *testing.T) {
	collector := service.NewCollectorService(happyFetcher(), happyWriter())

	err := collectAll(context.Background(), "test-key", collector, testLocations)
	if err != nil {
		t.Fatalf("run() returned error: %v", err)
	}
}

func TestRun_PartialFailure(t *testing.T) {
	// First location fails, second succeeds — should still be ok
	callCount := 0
	fetcher := &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location) (*api.PollenAPIResponse, error) {
			callCount++
			if callCount == 1 {
				return nil, fmt.Errorf("API unavailable")
			}
			return &api.PollenAPIResponse{
				DailyInfo: []api.DailyInfo{{
					PollenTypeInfo: []api.PollenTypeInfo{
						{Code: "TREE", InSeason: true, IndexInfo: api.IndexInfo{Value: 1, Category: "Very Low"}},
					},
				}},
			}, nil
		},
	}

	collector := service.NewCollectorService(fetcher, happyWriter())

	err := collectAll(context.Background(), "test-key", collector, testLocations)
	if err != nil {
		t.Fatalf("collectAll() should succeed with partial failures, got: %v", err)
	}
}

func TestRun_AllLocationsFail(t *testing.T) {
	collector := service.NewCollectorService(failingFetcher(), happyWriter())

	err := collectAll(context.Background(), "test-key", collector, testLocations)

	if err == nil {
		t.Fatal("collectAll() should return error when all locations fail")
	}
}

func TestRun_EmptyLocations(t *testing.T) {
	collector := service.NewCollectorService(happyFetcher(), happyWriter())

	err := collectAll(context.Background(), "test-key", collector, []shared.Location{})

	if err == nil {
		t.Fatal("collectAll() should return error when no locations are provided")
	}
}
