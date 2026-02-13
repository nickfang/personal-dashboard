package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	pb "github.com/nickfang/personal-dashboard/services/gen/go/weather-provider/v1"
)

// Mock implementation
type mockWeatherClient struct{}

func (m *mockWeatherClient) GetWeatherStat(ctx context.Context, locationID string) (*pb.PressureStat, error) {
	return &pb.PressureStat{
		LocationId: locationID,
		Trend:      "rising",
		Delta_1H:   0.5,
	}, nil
}

func (m *mockWeatherClient) GetWeatherStats(ctx context.Context) ([]*pb.PressureStat, error) {
	return []*pb.PressureStat{
		{
			LocationId: "house-nick",
			Trend:      "rising",
			Delta_1H:   0.5,
		},
	}, nil
}

func TestDashboardHandler_GetDashboard(t *testing.T) {
	// 1. Create Handler with Mock Client
	mockClient := &mockWeatherClient{}
	handler := NewDashboardHandler(mockClient)

	// 2. Create Request
	req, err := http.NewRequest("GET", "/api/v1/dashboard", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	// 3. Serve
	handler.GetDashboard(rr, req)

	// 4. Verify Status
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// 5. Verify JSON Structure
	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	// Check if we got the 'pressure' key
	pressure, ok := resp["pressure"].(map[string]interface{})
	if !ok {
		t.Fatal("Response missing 'pressure' object")
	}

	// Check if we got data for "house-nick" inside pressure
	data, ok := pressure["house-nick"].(map[string]interface{})
	if !ok {
		t.Fatal("Pressure object missing 'house-nick' entry")
	}

	if data["trend"] != "rising" {
		t.Errorf("Expected trend 'rising', got %v", data["trend"])
	}
}
