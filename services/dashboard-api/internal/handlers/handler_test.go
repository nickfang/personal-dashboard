package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	pb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/weather-provider/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Mock implementation that returns successful responses
type mockWeatherClient struct{}

func (m *mockWeatherClient) GetPressureStat(ctx context.Context, locationID string) (*pb.PressureStat, error) {
	return &pb.PressureStat{
		LocationId: locationID,
		Trend:      "rising",
		Delta_1H:   0.5,
	}, nil
}

func (m *mockWeatherClient) GetPressureStats(ctx context.Context) ([]*pb.PressureStat, error) {
	return []*pb.PressureStat{
		{
			LocationId: "house-nick",
			Trend:      "rising",
			Delta_1H:   0.5,
		},
	}, nil
}

// Mock implementation that returns errors
type errorWeatherClient struct {
	err error
}

func (m *errorWeatherClient) GetPressureStat(ctx context.Context, locationID string) (*pb.PressureStat, error) {
	return nil, m.err
}

func (m *errorWeatherClient) GetPressureStats(ctx context.Context) ([]*pb.PressureStat, error) {
	return nil, m.err
}

func TestDashboardHandler_GetDashboard(t *testing.T) {
	mockClient := &mockWeatherClient{}
	handler := NewDashboardHandler(mockClient)

	req, err := http.NewRequest("GET", "/api/v1/dashboard", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	handler.GetDashboard(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	pressure, ok := resp["pressure"].(map[string]interface{})
	if !ok {
		t.Fatal("Response missing 'pressure' object")
	}

	data, ok := pressure["house-nick"].(map[string]interface{})
	if !ok {
		t.Fatal("Pressure object missing 'house-nick' entry")
	}

	if data["trend"] != "rising" {
		t.Errorf("Expected trend 'rising', got %v", data["trend"])
	}
}

func TestDashboardHandler_GetDashboard_ProtojsonFormat(t *testing.T) {
	mockClient := &mockWeatherClient{}
	handler := NewDashboardHandler(mockClient)

	req, err := http.NewRequest("GET", "/api/v1/dashboard", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	handler.GetDashboard(rr, req)

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	pressure := resp["pressure"].(map[string]interface{})
	data := pressure["house-nick"].(map[string]interface{})

	// protojson uses camelCase field names, not snake_case
	if _, ok := data["locationId"]; !ok {
		t.Errorf("Expected camelCase 'locationId' from protojson, got keys: %v", keys(data))
	}
	if _, ok := data["delta1h"]; !ok {
		t.Errorf("Expected camelCase 'delta1h' from protojson, got keys: %v", keys(data))
	}
}

func TestDashboardHandler_GetDashboard_GrpcError(t *testing.T) {
	tests := []struct {
		name           string
		grpcErr        error
		expectedStatus int
	}{
		{
			name:           "Unavailable returns 503",
			grpcErr:        status.Error(codes.Unavailable, "weather-provider down"),
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "DeadlineExceeded returns 504",
			grpcErr:        status.Error(codes.DeadlineExceeded, "timeout"),
			expectedStatus: http.StatusGatewayTimeout,
		},
		{
			name:           "Unknown returns 500",
			grpcErr:        status.Error(codes.Unknown, "unknown"),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "Non-gRPC error returns 500",
			grpcErr:        fmt.Errorf("connection refused"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewDashboardHandler(&errorWeatherClient{err: tt.grpcErr})

			req, err := http.NewRequest("GET", "/api/v1/dashboard", nil)
			if err != nil {
				t.Fatal(err)
			}
			rr := httptest.NewRecorder()

			handler.GetDashboard(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func keys(m map[string]interface{}) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}
