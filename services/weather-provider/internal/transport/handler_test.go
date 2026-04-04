package transport

import (
	"context"
	"testing"
	"time"

	pb "github.com/nickfang/personal-dashboard/services/weather-provider/internal/gen/go/weather-provider/v1"
	"github.com/nickfang/personal-dashboard/services/weather-provider/internal/repository"
	"github.com/nickfang/personal-dashboard/services/weather-provider/internal/service"
	"github.com/nickfang/personal-dashboard/services/weather-provider/internal/testutil"
)

func TestGetPressureStats_Mapping(t *testing.T) {
	deltaValue := 1.5
	now := time.Now()

	mockRepo := &testutil.MockReader{
		GetByIDFunc: func(ctx context.Context, id string) (*repository.PressureCacheDoc, error) {
			return &repository.PressureCacheDoc{
				LocationID:  id,
				LastUpdated: now,
				Analysis: repository.PressureStats{
					Delta3h: &deltaValue,
					Trend:   "rising",
				},
			}, nil
		},
	}

	svc := service.NewWeatherService(mockRepo)
	handler := NewGrpcHandler(svc)

	req := &pb.GetPressureStatsRequest{LocationId: "test-loc"}
	resp, err := handler.GetPressureStats(context.Background(), req)
	if err != nil {
		t.Fatalf("failed to call handler: %v", err)
	}

	// Verify Mapping
	if resp.Stat.LocationId != "test-loc" {
		t.Errorf("expected LocationId test-loc, got %s", resp.Stat.LocationId)
	}
	if resp.Stat.Delta_3H != 1.5 {
		t.Errorf("expected Delta3h 1.5, got %f", resp.Stat.Delta_3H)
	}
	if resp.Stat.Trend != "rising" {
		t.Errorf("expected Trend rising, got %s", resp.Stat.Trend)
	}
	// Verify that unmapped fields remain default
	if resp.Stat.Delta_1H != 0 {
		t.Errorf("expected Delta1h 0 (unset), got %f", resp.Stat.Delta_1H)
	}
}

func TestGetAllPressureStats(t *testing.T) {
	now := time.Now()
	mockRepo := &testutil.MockReader{
		GetAllFunc: func(ctx context.Context) ([]repository.PressureCacheDoc, error) {
			return []repository.PressureCacheDoc{
				{LocationID: "loc-1", LastUpdated: now},
				{LocationID: "loc-2", LastUpdated: now},
			}, nil
		},
	}

	svc := service.NewWeatherService(mockRepo)
	handler := NewGrpcHandler(svc)

	resp, err := handler.GetAllPressureStats(context.Background(), &pb.GetAllPressureStatsRequest{})
	if err != nil {
		t.Fatalf("failed to call handler: %v", err)
	}

	if len(resp.Stats) != 2 {
		t.Errorf("expected 2 stats, got %d", len(resp.Stats))
	}
	if resp.Stats[0].LocationId != "loc-1" || resp.Stats[1].LocationId != "loc-2" {
		t.Errorf("unexpected location IDs in response")
	}
}

func TestGetLastWeather_Mapping(t *testing.T) {
	now := time.Now()

	mockRepo := &testutil.MockReader{
		GetLastWeatherFunc: func(ctx context.Context, id string) (*repository.WeatherCacheDoc, error) {
			return &repository.WeatherCacheDoc{
				LocationID: id,
				CurrentValue: repository.WeatherPoint{
					LocationID:           id,
					Timestamp:            now,
					TempC:                22.5,
					TempF:                72.5,
					TempFeelC:            21.0,
					TempFeelF:            69.8,
					HumidityPercent:      65,
					PressureMb:           1013.25,
					PrecipitationPercent: 10,
				},
			}, nil
		},
	}

	svc := service.NewWeatherService(mockRepo)
	handler := NewGrpcHandler(svc)

	req := &pb.GetLastWeatherRequest{LocationId: "test-loc"}
	resp, err := handler.GetLastWeather(context.Background(), req)
	if err != nil {
		t.Fatalf("failed to call handler: %v", err)
	}

	w := resp.Weather

	// Verify Mapping
	if w.LocationId != "test-loc" {
		t.Errorf("expected LocationId test-loc, got %s", w.LocationId)
	}
	if w.TempC != 22.5 {
		t.Errorf("expected TempC 22.5, got %f", w.TempC)
	}
	if w.TempF != 72.5 {
		t.Errorf("expected TempF 72.5, got %f", w.TempF)
	}
	if w.TempFeelC != 21.0 {
		t.Errorf("expected TempFeelC 21.0, got %f", w.TempFeelC)
	}
	if w.TempFeelF != 69.8 {
		t.Errorf("expected TempFeelF 69.8, got %f", w.TempFeelF)
	}
	if w.HumidityPercent != 65 {
		t.Errorf("expected HumidityPercent 65, got %d", w.HumidityPercent)
	}
	if w.PressureMb != 1013.25 {
		t.Errorf("expected PressureMb 1013.25, got %f", w.PressureMb)
	}
	if w.PrecipitationPercent != 10 {
		t.Errorf("expected PrecipitationPercent 10, got %d", w.PrecipitationPercent)
	}
}

func TestGetAllLastWeather(t *testing.T) {
	now := time.Now()
	mockRepo := &testutil.MockReader{
		GetAllLastWeatherFunc: func(ctx context.Context) ([]repository.WeatherCacheDoc, error) {
			return []repository.WeatherCacheDoc{
				{
					LocationID:  "loc-1",
					LastUpdated: now,
					CurrentValue: repository.WeatherPoint{
						LocationID: "loc-1",
						Timestamp:  now,
						TempC:      20.0,
						TempF:      68.0,
					},
				},
				{
					LocationID:  "loc-2",
					LastUpdated: now,
					CurrentValue: repository.WeatherPoint{
						LocationID: "loc-2",
						Timestamp:  now,
						TempC:      25.0,
						TempF:      77.0,
					},
				},
			}, nil
		},
	}

	svc := service.NewWeatherService(mockRepo)
	handler := NewGrpcHandler(svc)

	resp, err := handler.GetAllLastWeather(context.Background(), &pb.GetAllLastWeatherRequest{})
	if err != nil {
		t.Fatalf("failed to call handler: %v", err)
	}

	if len(resp.Weather) != 2 {
		t.Fatalf("expected 2 weather entries, got %d", len(resp.Weather))
	}
	if resp.Weather[0].LocationId != "loc-1" || resp.Weather[1].LocationId != "loc-2" {
		t.Errorf("unexpected location IDs in response")
	}
	if resp.Weather[0].TempC != 20.0 {
		t.Errorf("expected loc-1 TempC 20.0, got %f", resp.Weather[0].TempC)
	}
	if resp.Weather[1].TempC != 25.0 {
		t.Errorf("expected loc-2 TempC 25.0, got %f", resp.Weather[1].TempC)
	}
}
