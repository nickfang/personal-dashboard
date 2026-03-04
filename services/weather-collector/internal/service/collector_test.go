package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nickfang/personal-dashboard/services/shared"
	"github.com/nickfang/personal-dashboard/services/weather-collector/internal/api"
	"github.com/nickfang/personal-dashboard/services/weather-collector/internal/repository"
	"github.com/nickfang/personal-dashboard/services/weather-collector/internal/testutil"
)

// --- Pressure stats tests (migrated from main_test.go) ---

func TestCalculatePressureStats(t *testing.T) {
	now := time.Now()

	mkPoint := func(hoursAgo int, pressure float64) repository.PressurePoint {
		return repository.PressurePoint{
			TimeStamp:  now.Add(time.Duration(-hoursAgo) * time.Hour),
			PressureMb: pressure,
		}
	}

	tests := []struct {
		name         string
		history      []repository.PressurePoint
		wantTrend    string
		wantDelta3h  *float64
		wantDelta24h *float64
	}{
		{
			name:      "Empty History",
			history:   []repository.PressurePoint{},
			wantTrend: "unknown",
		},
		{
			name:      "Single Point",
			history:   []repository.PressurePoint{mkPoint(0, 1013.0)},
			wantTrend: "unknown",
		},
		{
			name: "Stable Pressure",
			history: []repository.PressurePoint{
				mkPoint(3, 1013.0),
				mkPoint(2, 1013.1),
				mkPoint(1, 1013.0),
				mkPoint(0, 1013.2),
			},
			wantTrend:   "stable",
			wantDelta3h: floatPtr(0.2), // 1013.2 - 1013.0
		},
		{
			name: "Rising Pressure",
			history: []repository.PressurePoint{
				mkPoint(4, 1010.0),
				mkPoint(3, 1011.0),
				mkPoint(2, 1012.0),
				mkPoint(1, 1013.0),
				mkPoint(0, 1014.0), // 1014 - 1011 = 3.0 increase
			},
			wantTrend:   "rising",
			wantDelta3h: floatPtr(3.0),
		},
		{
			name: "Falling Pressure",
			history: []repository.PressurePoint{
				mkPoint(3, 1020.0),
				mkPoint(2, 1019.0),
				mkPoint(0, 1018.0),
			},
			wantTrend:   "falling",
			wantDelta3h: floatPtr(-2.0), // 1018 - 1020
		},
		{
			name: "Long History with Gap",
			history: []repository.PressurePoint{
				mkPoint(24, 1000.0),
				mkPoint(12, 1005.0),
				mkPoint(0, 1010.0),
			},
			wantTrend:    "unknown",      // 3h delta is missing
			wantDelta24h: floatPtr(10.0), // 1010 - 1000
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := CalculatePressureStats(tt.history)

			if stats.Trend != tt.wantTrend {
				t.Errorf("Trend = %s, want %s", stats.Trend, tt.wantTrend)
			}

			if !compareFloatPtr(stats.Delta3h, tt.wantDelta3h) {
				t.Errorf("Delta3h = %v, want %v", formatFloatPtr(stats.Delta3h), formatFloatPtr(tt.wantDelta3h))
			}

			if !compareFloatPtr(stats.Delta24h, tt.wantDelta24h) {
				t.Errorf("Delta24h = %v, want %v", formatFloatPtr(stats.Delta24h), formatFloatPtr(tt.wantDelta24h))
			}
		})
	}
}

// --- Mapping tests (new) ---

func TestMapToWeatherPoint_InvalidPressure(t *testing.T) {
	data := api.WeatherAPIResponse{} // pressure defaults to 0.0

	_, err := MapToWeatherPoint("house-nick", data, time.Now())
	if err == nil {
		t.Fatal("MapToWeatherPoint should return error for 0.0 pressure")
	}
}

// --- Orchestration tests (new) ---

func TestCollect_Success(t *testing.T) {
	var savedWP repository.WeatherPoint
	var cachedLocationID string

	fetcher := &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location) (*api.WeatherAPIResponse, error) {
			resp := &api.WeatherAPIResponse{}
			resp.AirPressure.MeanSeaLevelMillibars = 1013.25
			resp.Temperature.Degrees = 25.0
			return resp, nil
		},
	}

	writer := &testutil.MockWriter{
		SaveRawFn: func(ctx context.Context, wp repository.WeatherPoint) error {
			savedWP = wp
			return nil
		},
		UpdateCacheFn: func(ctx context.Context, locationID string, wp repository.WeatherPoint, analyze repository.AnalyzeFunc) error {
			cachedLocationID = locationID
			return nil
		},
	}

	svc := NewCollectorService(fetcher, writer)
	loc := shared.Location{ID: "house-nick", Lat: 30.0, Long: -97.0}
	err := svc.Collect(context.Background(), "test-key", loc)
	if err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}

	if savedWP.Location != "house-nick" {
		t.Errorf("SaveRaw WeatherPoint Location = %q, want %q", savedWP.Location, "house-nick")
	}

	if cachedLocationID != "house-nick" {
		t.Errorf("UpdateCache locationID = %q, want %q", cachedLocationID, "house-nick")
	}
}

func TestCollect_FetchError(t *testing.T) {
	writerCalled := false

	fetcher := &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location) (*api.WeatherAPIResponse, error) {
			return nil, fmt.Errorf("API unavailable")
		},
	}

	writer := &testutil.MockWriter{
		SaveRawFn: func(ctx context.Context, wp repository.WeatherPoint) error {
			writerCalled = true
			return nil
		},
		UpdateCacheFn: func(ctx context.Context, locationID string, wp repository.WeatherPoint, analyze repository.AnalyzeFunc) error {
			writerCalled = true
			return nil
		},
	}

	svc := NewCollectorService(fetcher, writer)
	loc := shared.Location{ID: "house-nick", Lat: 30.0, Long: -97.0}
	err := svc.Collect(context.Background(), "test-key", loc)

	if err == nil {
		t.Fatal("Collect() should return error when fetch fails")
	}

	if writerCalled {
		t.Error("Writer should not be called when fetch fails")
	}
}

func TestCollect_SaveRawError(t *testing.T) {
	cacheCalled := false

	fetcher := &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location) (*api.WeatherAPIResponse, error) {
			resp := &api.WeatherAPIResponse{}
			resp.AirPressure.MeanSeaLevelMillibars = 1013.25
			return resp, nil
		},
	}

	writer := &testutil.MockWriter{
		SaveRawFn: func(ctx context.Context, wp repository.WeatherPoint) error {
			return fmt.Errorf("firestore write failed")
		},
		UpdateCacheFn: func(ctx context.Context, locationID string, wp repository.WeatherPoint, analyze repository.AnalyzeFunc) error {
			cacheCalled = true
			return nil
		},
	}

	svc := NewCollectorService(fetcher, writer)
	loc := shared.Location{ID: "house-nick", Lat: 30.0, Long: -97.0}
	err := svc.Collect(context.Background(), "test-key", loc)

	if err == nil {
		t.Fatal("Collect() should return error when SaveRaw fails")
	}
	if cacheCalled {
		t.Error("UpdateCache should not be called when SaveRaw fails")
	}
}

func TestCollect_UpdateCacheError(t *testing.T) {
	fetcher := &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location) (*api.WeatherAPIResponse, error) {
			resp := &api.WeatherAPIResponse{}
			resp.AirPressure.MeanSeaLevelMillibars = 1013.25
			return resp, nil
		},
	}

	writer := &testutil.MockWriter{
		SaveRawFn: func(ctx context.Context, wp repository.WeatherPoint) error {
			return nil
		},
		UpdateCacheFn: func(ctx context.Context, locationID string, wp repository.WeatherPoint, analyze repository.AnalyzeFunc) error {
			return fmt.Errorf("cache update failed")
		},
	}

	svc := NewCollectorService(fetcher, writer)
	loc := shared.Location{ID: "house-nick", Lat: 30.0, Long: -97.0}
	err := svc.Collect(context.Background(), "test-key", loc)

	if err == nil {
		t.Fatal("Collect() should return error when UpdateCache fails")
	}
}

// --- Test helpers ---

func floatPtr(f float64) *float64 {
	return &f
}

func compareFloatPtr(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	diff := *a - *b
	if diff < 0 {
		diff = -diff
	}
	return diff < 0.001
}

func formatFloatPtr(p *float64) string {
	if p == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%.4f", *p)
}
