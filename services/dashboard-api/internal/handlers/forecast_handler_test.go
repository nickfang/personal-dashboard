package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	weatherPb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/weather-provider/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// --- Forecast mocks ---

type mockForecastClient struct{}

func (m *mockForecastClient) GetForecast(ctx context.Context, locationID string) (*weatherPb.Forecast, error) {
	base := time.Date(2026, 6, 12, 6, 0, 0, 0, time.UTC)
	points := make([]*weatherPb.ForecastPoint, 0, 4)
	for i, mb := range []float64{1013.25, 1011.0, 1009.0, 1007.25} {
		points = append(points, &weatherPb.ForecastPoint{
			ValidTime:  timestamppb.New(base.Add(time.Duration(i) * time.Hour)),
			PressureMb: mb,
			TempF:      81.5,
		})
	}
	return &weatherPb.Forecast{
		LocationId:  locationID,
		IssuedAt:    timestamppb.New(base),
		LastUpdated: timestamppb.New(base),
		Points:      points,
		Alerts: []*weatherPb.Alert{
			{
				Id:          "alert-1",
				LocationId:  locationID,
				Source:      "pressure",
				RuleId:      "pressure-drop-3h",
				Severity:    "warning",
				Metric:      "pressure_mb_delta",
				Value:       -6.0,
				Threshold:   5,
				WindowStart: timestamppb.New(base),
				WindowEnd:   timestamppb.New(base.Add(3 * time.Hour)),
				Message:     "Fri 1 AM  -6.0 mb/3h",
				Status:      "active",
				IssuedAt:    timestamppb.New(base),
			},
		},
	}, nil
}

type errorForecastClient struct {
	err error
}

func (m *errorForecastClient) GetForecast(ctx context.Context, locationID string) (*weatherPb.Forecast, error) {
	return nil, m.err
}

// --- Forecast aggregation tests ---

func TestDashboardHandler_GetDashboard_IncludesForecastAndAlerts(t *testing.T) {
	handler := NewDashboardHandler(&mockWeatherClient{}, &mockPollenClient{}, &mockForecastClient{})

	req, err := http.NewRequest("GET", "/api/v1/dashboard", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	handler.GetDashboard(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	forecast, ok := resp["forecast"].(map[string]interface{})
	if !ok {
		t.Fatal("Response missing 'forecast' object")
	}
	forecastData, ok := forecast["house-nick"].(map[string]interface{})
	if !ok {
		t.Fatalf("Forecast object missing 'house-nick' entry (keys: %v)", keys(forecast))
	}
	if _, ok := forecastData["locationId"]; !ok {
		t.Errorf("Expected camelCase 'locationId' from protojson, got keys: %v", keys(forecastData))
	}
	points, ok := forecastData["points"].([]interface{})
	if !ok || len(points) != 4 {
		t.Fatalf("Expected 4 forecast points, got %v", forecastData["points"])
	}
	if _, ok := forecastData["alerts"]; ok {
		t.Error("forecast entry should not embed alerts; they have their own top-level map")
	}

	alerts, ok := resp["alerts"].(map[string]interface{})
	if !ok {
		t.Fatal("Response missing 'alerts' object")
	}
	alertList, ok := alerts["house-nick"].([]interface{})
	if !ok || len(alertList) != 1 {
		t.Fatalf("Expected 1 alert for house-nick, got %v", alerts["house-nick"])
	}
	alert := alertList[0].(map[string]interface{})
	if alert["ruleId"] != "pressure-drop-3h" {
		t.Errorf("Expected camelCase ruleId 'pressure-drop-3h', got %v (keys: %v)", alert["ruleId"], keys(alert))
	}
	if alert["status"] != "active" {
		t.Errorf("Expected status 'active', got %v", alert["status"])
	}
	if alert["value"] != -6.0 {
		t.Errorf("Expected value -6.0, got %v", alert["value"])
	}
	if _, ok := alert["windowStart"]; !ok {
		t.Errorf("Expected camelCase 'windowStart', got keys: %v", keys(alert))
	}
}

func TestDashboardHandler_GetDashboard_ForecastNotFound_Tolerated(t *testing.T) {
	handler := NewDashboardHandler(
		&mockWeatherClient{},
		&mockPollenClient{},
		&errorForecastClient{err: status.Error(codes.NotFound, "no forecast yet")},
	)

	req, err := http.NewRequest("GET", "/api/v1/dashboard", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	handler.GetDashboard(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 when forecast is NotFound, got %d", rr.Code)
	}

	var body map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	var forecast map[string]json.RawMessage
	if err := json.Unmarshal(body["forecast"], &forecast); err != nil {
		t.Fatalf("decode forecast map: %v", err)
	}
	if len(forecast) != 0 {
		t.Errorf("forecast map should be empty when no forecasts exist, got %d entries", len(forecast))
	}
}

func TestDashboardHandler_GetDashboard_ForecastUnavailable_Fatal(t *testing.T) {
	handler := NewDashboardHandler(
		&mockWeatherClient{},
		&mockPollenClient{},
		&errorForecastClient{err: status.Error(codes.Unavailable, "weather-provider down")},
	)

	req, err := http.NewRequest("GET", "/api/v1/dashboard", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	handler.GetDashboard(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when forecast RPC is Unavailable, got %d", rr.Code)
	}
}

func TestDashboardHandler_GetDashboardByLocation_IncludesForecast(t *testing.T) {
	handler := NewDashboardHandler(&mockWeatherClient{}, &mockPollenClient{}, &mockForecastClient{})
	req := requestWithLocationID(t, "house-nick")
	rr := httptest.NewRecorder()

	handler.GetDashboardByLocation(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	forecast, ok := resp["forecast"].(map[string]interface{})
	if !ok {
		t.Fatal("Response missing 'forecast' object")
	}
	if len(forecast) != 1 {
		t.Errorf("expected exactly one forecast entry, got %d (keys: %v)", len(forecast), keys(forecast))
	}
	if _, ok := forecast["house-nick"]; !ok {
		t.Errorf("forecast missing 'house-nick' entry")
	}

	alerts, ok := resp["alerts"].(map[string]interface{})
	if !ok {
		t.Fatal("Response missing 'alerts' object")
	}
	alertList, ok := alerts["house-nick"].([]interface{})
	if !ok || len(alertList) != 1 {
		t.Fatalf("Expected 1 alert for house-nick, got %v", alerts["house-nick"])
	}
}

func TestDashboardHandler_GetDashboard_CurlText_IncludesForecast(t *testing.T) {
	handler := NewDashboardHandler(&mockWeatherClient{}, &mockPollenClient{}, &mockForecastClient{})

	req, err := http.NewRequest("GET", "/api/v1/dashboard", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", "curl/8.7.1")
	rr := httptest.NewRecorder()

	handler.GetDashboard(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	body := rr.Body.String()

	if !strings.Contains(body, "Forecast:") {
		t.Errorf("expected Forecast section in text response, got:\n%s", body)
	}
	if !strings.Contains(body, "falling") {
		t.Errorf("expected falling trend in forecast text, got:\n%s", body)
	}
	if !strings.Contains(body, "⚠ Fri 1 AM  -6.0 mb/3h") {
		t.Errorf("expected alert banner in text response, got:\n%s", body)
	}
}
