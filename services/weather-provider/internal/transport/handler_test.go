package transport

import (
	"context"
	"testing"
	"time"

	pb "github.com/nickfang/personal-dashboard/services/gen/go/weather-provider/v1"
	"github.com/nickfang/personal-dashboard/services/weather-provider/internal/repository"
	"github.com/nickfang/personal-dashboard/services/weather-provider/internal/service"
)

// MockReader is used by the Service in this test
type MockReader struct {
	GetByIDFunc func(ctx context.Context, id string) (*repository.CacheDoc, error)
	GetAllFunc  func(ctx context.Context) ([]repository.CacheDoc, error)
}

func (m *MockReader) GetAll(ctx context.Context) ([]repository.CacheDoc, error) {
	return m.GetAllFunc(ctx)
}
func (m *MockReader) GetByID(ctx context.Context, id string) (*repository.CacheDoc, error) {
	return m.GetByIDFunc(ctx, id)
}

func TestGetPressureStats_Mapping(t *testing.T) {
	deltaValue := 1.5
	now := time.Now()

	mockRepo := &MockReader{
		GetByIDFunc: func(ctx context.Context, id string) (*repository.CacheDoc, error) {
			return &repository.CacheDoc{
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
	mockRepo := &MockReader{
		GetAllFunc: func(ctx context.Context) ([]repository.CacheDoc, error) {
			return []repository.CacheDoc{
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
