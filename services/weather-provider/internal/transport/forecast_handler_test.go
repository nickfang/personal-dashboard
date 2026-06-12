package transport

import (
	"context"
	"testing"
	"time"

	"github.com/nickfang/personal-dashboard/services/shared"
	pb "github.com/nickfang/personal-dashboard/services/weather-provider/internal/gen/go/weather-provider/v1"
	"github.com/nickfang/personal-dashboard/services/weather-provider/internal/repository"
	"github.com/nickfang/personal-dashboard/services/weather-provider/internal/service"
	"github.com/nickfang/personal-dashboard/services/weather-provider/internal/testutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetForecast_Mapping(t *testing.T) {
	now := time.Date(2026, 6, 12, 6, 0, 0, 0, time.UTC)
	windowEnd := now.Add(3 * time.Hour)

	mockRepo := &testutil.MockReader{
		GetForecastFunc: func(ctx context.Context, id string) (*repository.ForecastCacheDoc, error) {
			return &repository.ForecastCacheDoc{
				LocationID: id,
				IssuedAt:   now,
				Points: []repository.ForecastPoint{
					{ValidTime: now, PressureMb: 1012.65, TempF: 81.5, HumidityPercent: 82},
					{ValidTime: now.Add(time.Hour), PressureMb: 1013.13},
				},
				Alerts: []shared.Alert{
					{
						ID:          "alert-1",
						Location:    id,
						RuleID:      "pressure-drop-3h",
						Severity:    shared.AlertSeverityWarning,
						Value:       -6.2,
						Threshold:   5,
						WindowStart: now,
						WindowEnd:   windowEnd,
						Message:     "Fri 1 AM  -6.2 mb/3h",
						Status:      shared.AlertStatusActive,
						IssuedAt:    now,
					},
				},
			}, nil
		},
	}

	svc := service.NewWeatherService(mockRepo)
	handler := NewGrpcHandler(svc)

	resp, err := handler.GetForecast(context.Background(), &pb.GetForecastRequest{LocationId: "test-loc"})
	if err != nil {
		t.Fatalf("failed to call handler: %v", err)
	}

	f := resp.Forecast
	if f.LocationId != "test-loc" {
		t.Errorf("LocationId = %q, want test-loc", f.LocationId)
	}
	if !f.IssuedAt.AsTime().Equal(now) {
		t.Errorf("IssuedAt = %v, want %v", f.IssuedAt.AsTime(), now)
	}
	if len(f.Points) != 2 {
		t.Fatalf("len(Points) = %d, want 2", len(f.Points))
	}
	if f.Points[0].PressureMb != 1012.65 || f.Points[0].TempF != 81.5 || f.Points[0].HumidityPercent != 82 {
		t.Errorf("Points[0] = %v, fields not mapped", f.Points[0])
	}
	if !f.Points[1].ValidTime.AsTime().Equal(now.Add(time.Hour)) {
		t.Errorf("Points[1].ValidTime = %v, want +1h", f.Points[1].ValidTime.AsTime())
	}
	if len(f.Alerts) != 1 {
		t.Fatalf("len(Alerts) = %d, want 1", len(f.Alerts))
	}
	a := f.Alerts[0]
	if a.Id != "alert-1" || a.LocationId != "test-loc" || a.RuleId != "pressure-drop-3h" {
		t.Errorf("alert identity = %q/%q/%q", a.Id, a.LocationId, a.RuleId)
	}
	if a.Value != -6.2 || a.Threshold != 5 || a.Severity != "warning" || a.Status != "active" {
		t.Errorf("alert values = %v/%v/%q/%q", a.Value, a.Threshold, a.Severity, a.Status)
	}
	if a.Message != "Fri 1 AM  -6.2 mb/3h" {
		t.Errorf("alert message = %q", a.Message)
	}
	if !a.WindowEnd.AsTime().Equal(windowEnd) {
		t.Errorf("WindowEnd = %v, want %v", a.WindowEnd.AsTime(), windowEnd)
	}
}

func TestGetAllForecasts(t *testing.T) {
	now := time.Date(2026, 6, 12, 6, 0, 0, 0, time.UTC)
	mockRepo := &testutil.MockReader{
		GetAllForecastsFunc: func(ctx context.Context) ([]repository.ForecastCacheDoc, error) {
			return []repository.ForecastCacheDoc{
				{LocationID: "house-nick", IssuedAt: now, Points: []repository.ForecastPoint{{ValidTime: now, PressureMb: 1012.65}}},
				{LocationID: "house-nita", IssuedAt: now},
			}, nil
		},
	}

	svc := service.NewWeatherService(mockRepo)
	handler := NewGrpcHandler(svc)

	resp, err := handler.GetAllForecasts(context.Background(), &pb.GetAllForecastsRequest{})
	if err != nil {
		t.Fatalf("failed to call handler: %v", err)
	}

	if len(resp.Forecasts) != 2 {
		t.Fatalf("len(Forecasts) = %d, want 2", len(resp.Forecasts))
	}
	if resp.Forecasts[0].LocationId != "house-nick" || resp.Forecasts[1].LocationId != "house-nita" {
		t.Errorf("unexpected location IDs in response")
	}
	if resp.Forecasts[0].Points[0].PressureMb != 1012.65 {
		t.Errorf("Points[0].PressureMb = %v, want 1012.65", resp.Forecasts[0].Points[0].PressureMb)
	}
}

func TestGetForecast_NotFound(t *testing.T) {
	mockRepo := &testutil.MockReader{
		GetForecastFunc: func(ctx context.Context, id string) (*repository.ForecastCacheDoc, error) {
			return nil, status.Errorf(codes.NotFound, "no doc")
		},
	}

	svc := service.NewWeatherService(mockRepo)
	handler := NewGrpcHandler(svc)

	_, err := handler.GetForecast(context.Background(), &pb.GetForecastRequest{LocationId: "missing"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("error code = %v, want NotFound", status.Code(err))
	}
}

func TestGetForecast_EmptyLocationID(t *testing.T) {
	svc := service.NewWeatherService(&testutil.MockReader{})
	handler := NewGrpcHandler(svc)

	_, err := handler.GetForecast(context.Background(), &pb.GetForecastRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("error code = %v, want InvalidArgument", status.Code(err))
	}
}
