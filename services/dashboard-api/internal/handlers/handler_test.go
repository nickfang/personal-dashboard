package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	pollenPb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/pollen-provider/v1"
	weatherPb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/weather-provider/v1"
	"github.com/nickfang/personal-dashboard/services/shared"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// requestWithLocationID returns an *http.Request whose chi RouteContext has
// the {locationID} URL param set to the given value, simulating chi's router
// without routing through chi.NewRouter. Pass an empty string to simulate a
// request whose param was never populated.
func requestWithLocationID(t *testing.T, locationID string) *http.Request {
	t.Helper()
	req, err := http.NewRequest("GET", "/api/v1/dashboard/"+locationID, nil)
	if err != nil {
		t.Fatal(err)
	}
	rctx := chi.NewRouteContext()
	if locationID != "" {
		rctx.URLParams.Add("locationID", locationID)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// --- Weather mocks ---

type mockWeatherClient struct{}

func (m *mockWeatherClient) GetPressureStat(ctx context.Context, locationID string) (*weatherPb.PressureStat, error) {
	return &weatherPb.PressureStat{
		LocationId:  locationID,
		Trend:       "rising",
		Delta_1H:    0.5,
		LastUpdated: timestamppb.Now(),
	}, nil
}

func (m *mockWeatherClient) GetPressureStats(ctx context.Context) ([]*weatherPb.PressureStat, error) {
	return []*weatherPb.PressureStat{
		{
			LocationId:  "house-nick",
			Trend:       "rising",
			Delta_1H:    0.5,
			LastUpdated: timestamppb.Now(),
		},
	}, nil
}

func (m *mockWeatherClient) GetLastWeather(ctx context.Context, locationID string) (*weatherPb.Weather, error) {
	return &weatherPb.Weather{
		LocationId:           locationID,
		TempC:                22.5,
		TempF:                72.5,
		TempFeelC:            21.0,
		TempFeelF:            69.8,
		HumidityPercent:      65,
		PressureMb:           1013.25,
		PrecipitationPercent: 10,
		LastUpdated:          timestamppb.Now(),
	}, nil
}

func (m *mockWeatherClient) GetAllLastWeather(ctx context.Context) ([]*weatherPb.Weather, error) {
	return []*weatherPb.Weather{
		{
			LocationId:           "house-nick",
			TempC:                22.5,
			TempF:                72.5,
			TempFeelC:            21.0,
			TempFeelF:            69.8,
			HumidityPercent:      65,
			PressureMb:           1013.25,
			PrecipitationPercent: 10,
			LastUpdated:          timestamppb.Now(),
		},
	}, nil
}

type errorWeatherClient struct {
	err error
}

func (m *errorWeatherClient) GetPressureStat(ctx context.Context, locationID string) (*weatherPb.PressureStat, error) {
	return nil, m.err
}

func (m *errorWeatherClient) GetPressureStats(ctx context.Context) ([]*weatherPb.PressureStat, error) {
	return nil, m.err
}

func (m *errorWeatherClient) GetLastWeather(ctx context.Context, locationID string) (*weatherPb.Weather, error) {
	return nil, m.err
}

func (m *errorWeatherClient) GetAllLastWeather(ctx context.Context) ([]*weatherPb.Weather, error) {
	return nil, m.err
}

// --- Pollen mocks ---

type mockPollenClient struct{}

func (m *mockPollenClient) GetPollenReport(ctx context.Context, locationID string) (*pollenPb.PollenReport, error) {
	return &pollenPb.PollenReport{
		LocationId:      locationID,
		CollectedAt:     timestamppb.Now(),
		OverallIndex:    4,
		OverallCategory: "High",
		DominantType:    "TREE",
		Types: []*pollenPb.PollenType{
			{Code: "TREE", Index: 4, Category: "High", InSeason: true},
			{Code: "GRASS", Index: 1, Category: "Very Low", InSeason: false},
		},
		Plants: []*pollenPb.PollenPlant{
			{Code: "JUNIPER", DisplayName: "Juniper", Index: 4, Category: "High", InSeason: true},
		},
	}, nil
}

func (m *mockPollenClient) GetPollenReports(ctx context.Context) ([]*pollenPb.PollenReport, error) {
	return []*pollenPb.PollenReport{
		{
			LocationId:      "house-nick",
			CollectedAt:     timestamppb.Now(),
			OverallIndex:    4,
			OverallCategory: "High",
			DominantType:    "TREE",
			Types: []*pollenPb.PollenType{
				{Code: "TREE", Index: 4, Category: "High", InSeason: true},
				{Code: "GRASS", Index: 1, Category: "Very Low", InSeason: false},
			},
			Plants: []*pollenPb.PollenPlant{
				{Code: "JUNIPER", DisplayName: "Juniper", Index: 4, Category: "High", InSeason: true},
			},
		},
	}, nil
}

type errorPollenClient struct {
	err error
}

func (m *errorPollenClient) GetPollenReport(ctx context.Context, locationID string) (*pollenPb.PollenReport, error) {
	return nil, m.err
}

func (m *errorPollenClient) GetPollenReports(ctx context.Context) ([]*pollenPb.PollenReport, error) {
	return nil, m.err
}

// --- Existing weather tests (updated to pass both mocks) ---

func TestDashboardHandler_GetDashboard(t *testing.T) {
	handler := NewDashboardHandler(&mockWeatherClient{}, &mockPollenClient{})

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
	handler := NewDashboardHandler(&mockWeatherClient{}, &mockPollenClient{})

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
			handler := NewDashboardHandler(&errorWeatherClient{err: tt.grpcErr}, &mockPollenClient{})

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

// --- Weather (last weather) integration tests ---

func TestDashboardHandler_GetDashboard_IncludesWeather(t *testing.T) {
	handler := NewDashboardHandler(&mockWeatherClient{}, &mockPollenClient{})

	req, err := http.NewRequest("GET", "/api/v1/dashboard", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	handler.GetDashboard(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	if _, ok := resp["weather"]; !ok {
		t.Fatal("Response missing 'weather' key")
	}

	weather, ok := resp["weather"].(map[string]interface{})
	if !ok {
		t.Fatal("'weather' is not an object")
	}

	weatherData, ok := weather["house-nick"].(map[string]interface{})
	if !ok {
		t.Fatal("Weather object missing 'house-nick' entry")
	}

	if weatherData["tempC"] != 22.5 {
		t.Errorf("Expected tempC 22.5, got %v", weatherData["tempC"])
	}
	if weatherData["tempF"] != 72.5 {
		t.Errorf("Expected tempF 72.5, got %v", weatherData["tempF"])
	}
}

func TestDashboardHandler_GetDashboard_WeatherProtojsonFormat(t *testing.T) {
	handler := NewDashboardHandler(&mockWeatherClient{}, &mockPollenClient{})

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

	weather := resp["weather"].(map[string]interface{})
	data := weather["house-nick"].(map[string]interface{})

	// protojson uses camelCase field names
	if _, ok := data["locationId"]; !ok {
		t.Errorf("Expected camelCase 'locationId' from protojson, got keys: %v", keys(data))
	}
	if _, ok := data["tempC"]; !ok {
		t.Errorf("Expected camelCase 'tempC' from protojson, got keys: %v", keys(data))
	}
	if _, ok := data["humidityPercent"]; !ok {
		t.Errorf("Expected camelCase 'humidityPercent' from protojson, got keys: %v", keys(data))
	}
}

// --- New pollen integration tests ---

func TestDashboardHandler_GetDashboard_IncludesPollen(t *testing.T) {
	handler := NewDashboardHandler(&mockWeatherClient{}, &mockPollenClient{})

	req, err := http.NewRequest("GET", "/api/v1/dashboard", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	handler.GetDashboard(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	// Verify both keys exist
	if _, ok := resp["pressure"]; !ok {
		t.Fatal("Response missing 'pressure' key")
	}
	if _, ok := resp["pollen"]; !ok {
		t.Fatal("Response missing 'pollen' key")
	}

	// Verify pollen data structure
	pollen, ok := resp["pollen"].(map[string]interface{})
	if !ok {
		t.Fatal("'pollen' is not an object")
	}

	pollenData, ok := pollen["house-nick"].(map[string]interface{})
	if !ok {
		t.Fatal("Pollen object missing 'house-nick' entry")
	}

	if pollenData["dominantType"] != "TREE" {
		t.Errorf("Expected dominantType 'TREE', got %v", pollenData["dominantType"])
	}

	// protojson renders int32 as number
	overallIndex, ok := pollenData["overallIndex"].(float64)
	if !ok {
		t.Fatalf("Expected overallIndex to be a number, got %T", pollenData["overallIndex"])
	}
	if overallIndex != 4 {
		t.Errorf("Expected overallIndex 4, got %v", overallIndex)
	}
}

func TestDashboardHandler_GetDashboard_PollenProtojsonFormat(t *testing.T) {
	handler := NewDashboardHandler(&mockWeatherClient{}, &mockPollenClient{})

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

	pollen := resp["pollen"].(map[string]interface{})
	data := pollen["house-nick"].(map[string]interface{})

	// protojson uses camelCase
	if _, ok := data["locationId"]; !ok {
		t.Errorf("Expected camelCase 'locationId' from protojson, got keys: %v", keys(data))
	}
	if _, ok := data["overallIndex"]; !ok {
		t.Errorf("Expected camelCase 'overallIndex' from protojson, got keys: %v", keys(data))
	}
	if _, ok := data["dominantType"]; !ok {
		t.Errorf("Expected camelCase 'dominantType' from protojson, got keys: %v", keys(data))
	}
}

func TestDashboardHandler_GetDashboard_PollenGrpcError(t *testing.T) {
	tests := []struct {
		name           string
		grpcErr        error
		expectedStatus int
	}{
		{
			name:           "Unavailable returns 503",
			grpcErr:        status.Error(codes.Unavailable, "pollen-provider down"),
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
			handler := NewDashboardHandler(&mockWeatherClient{}, &errorPollenClient{err: tt.grpcErr})

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

func TestDashboardHandler_GetDashboard_BothServicesFail(t *testing.T) {
	handler := NewDashboardHandler(
		&errorWeatherClient{err: status.Error(codes.Unavailable, "weather down")},
		&errorPollenClient{err: status.Error(codes.DeadlineExceeded, "pollen timeout")},
	)

	req, err := http.NewRequest("GET", "/api/v1/dashboard", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	handler.GetDashboard(rr, req)

	if rr.Code == http.StatusOK {
		t.Errorf("expected error status when both services fail, got 200")
	}
}

// --- Per-RPC deadline tests ---

// slowWeatherClient simulates a provider that takes longer than the per-RPC timeout.
type slowWeatherClient struct {
	delay time.Duration
}

func (m *slowWeatherClient) GetPressureStat(ctx context.Context, locationID string) (*weatherPb.PressureStat, error) {
	timer := time.NewTimer(m.delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return &weatherPb.PressureStat{LocationId: locationID}, nil
	case <-ctx.Done():
		return nil, status.Error(codes.DeadlineExceeded, ctx.Err().Error())
	}
}

func (m *slowWeatherClient) GetPressureStats(ctx context.Context) ([]*weatherPb.PressureStat, error) {
	timer := time.NewTimer(m.delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return []*weatherPb.PressureStat{{LocationId: "house-nick"}}, nil
	case <-ctx.Done():
		return nil, status.Error(codes.DeadlineExceeded, ctx.Err().Error())
	}
}

func (m *slowWeatherClient) GetLastWeather(ctx context.Context, locationID string) (*weatherPb.Weather, error) {
	timer := time.NewTimer(m.delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return &weatherPb.Weather{LocationId: "house-nick"}, nil
	case <-ctx.Done():
		return nil, status.Error(codes.DeadlineExceeded, ctx.Err().Error())
	}
}

func (m *slowWeatherClient) GetAllLastWeather(ctx context.Context) ([]*weatherPb.Weather, error) {
	timer := time.NewTimer(m.delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return []*weatherPb.Weather{{LocationId: "house-nick"}}, nil
	case <-ctx.Done():
		return nil, status.Error(codes.DeadlineExceeded, ctx.Err().Error())
	}
}

// slowPollenClient simulates a provider that takes longer than the per-RPC timeout.
type slowPollenClient struct {
	delay time.Duration
}

func (m *slowPollenClient) GetPollenReport(ctx context.Context, locationID string) (*pollenPb.PollenReport, error) {
	timer := time.NewTimer(m.delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return &pollenPb.PollenReport{LocationId: locationID}, nil
	case <-ctx.Done():
		return nil, status.Error(codes.DeadlineExceeded, ctx.Err().Error())
	}
}

func (m *slowPollenClient) GetPollenReports(ctx context.Context) ([]*pollenPb.PollenReport, error) {
	timer := time.NewTimer(m.delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return []*pollenPb.PollenReport{{LocationId: "house-nick"}}, nil
	case <-ctx.Done():
		return nil, status.Error(codes.DeadlineExceeded, ctx.Err().Error())
	}
}

func TestDashboardHandler_GetDashboard_SlowWeatherTimesOut(t *testing.T) {
	handler := NewDashboardHandler(
		&slowWeatherClient{delay: 10 * time.Second},
		&mockPollenClient{},
	)

	req, err := http.NewRequest("GET", "/api/v1/dashboard", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	start := time.Now()
	handler.GetDashboard(rr, req)
	elapsed := time.Since(start)

	if rr.Code != http.StatusGatewayTimeout {
		t.Errorf("expected status 504, got %d", rr.Code)
	}
	if elapsed > shared.RPCClientTimeout+1*time.Second {
		t.Errorf("expected per-RPC timeout to fire within 5s, but took %s", elapsed)
	}
}

func TestDashboardHandler_GetDashboard_SlowPollenTimesOut(t *testing.T) {
	handler := NewDashboardHandler(
		&mockWeatherClient{},
		&slowPollenClient{delay: 10 * time.Second},
	)

	req, err := http.NewRequest("GET", "/api/v1/dashboard", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	start := time.Now()
	handler.GetDashboard(rr, req)
	elapsed := time.Since(start)

	if rr.Code != http.StatusGatewayTimeout {
		t.Errorf("expected status 504, got %d", rr.Code)
	}
	if elapsed > shared.RPCClientTimeout+1*time.Second {
		t.Errorf("expected per-RPC timeout to fire within 5s, but took %s", elapsed)
	}
}

func TestDashboardHandler_GetDashboard_CurlUserAgent_ReturnsText(t *testing.T) {
	handler := NewDashboardHandler(&mockWeatherClient{}, &mockPollenClient{})

	req, err := http.NewRequest("GET", "/api/v1/dashboard", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", "curl/8.7.1")
	rr := httptest.NewRecorder()

	handler.GetDashboard(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "text/plain; charset=utf-8" {
		t.Errorf("expected Content-Type 'text/plain; charset=utf-8', got '%s'", contentType)
	}

	body := rr.Body.String()

	// Should contain weather data
	if !strings.Contains(body, "Weather:") {
		t.Errorf("expected Weather section in text response, got:\n%s", body)
	}
	if !strings.Contains(body, "Temp:") {
		t.Errorf("expected Temp in text response, got:\n%s", body)
	}

	// Should contain pressure data
	if !strings.Contains(body, "Pressure:") {
		t.Errorf("expected Pressure section in text response, got:\n%s", body)
	}
	if !strings.Contains(body, "rising") {
		t.Errorf("expected trend in text response, got:\n%s", body)
	}

	// Should contain pollen data
	if !strings.Contains(body, "Pollen:") {
		t.Errorf("expected Pollen section in text response, got:\n%s", body)
	}

	// Should NOT be valid JSON
	var jsonCheck map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &jsonCheck); err == nil {
		t.Error("expected non-JSON response for curl user agent, but got valid JSON")
	}
}

func keys(m map[string]interface{}) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}

// --- GetDashboardByLocation tests ---

func TestDashboardHandler_GetDashboardByLocation_Success(t *testing.T) {
	handler := NewDashboardHandler(&mockWeatherClient{}, &mockPollenClient{})
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

	for _, key := range []string{"weather", "pressure", "pollen"} {
		section, ok := resp[key].(map[string]interface{})
		if !ok {
			t.Fatalf("Response missing %q object", key)
		}
		if len(section) != 1 {
			t.Errorf("expected exactly one entry under %q, got %d (keys: %v)", key, len(section), keys(section))
		}
		if _, ok := section["house-nick"]; !ok {
			t.Errorf("Response %q missing 'house-nick' entry (keys: %v)", key, keys(section))
		}
	}

	weather := resp["weather"].(map[string]interface{})["house-nick"].(map[string]interface{})
	if weather["tempC"] != 22.5 {
		t.Errorf("Expected tempC 22.5, got %v", weather["tempC"])
	}

	pressure := resp["pressure"].(map[string]interface{})["house-nick"].(map[string]interface{})
	if pressure["trend"] != "rising" {
		t.Errorf("Expected trend 'rising', got %v", pressure["trend"])
	}
	if _, ok := pressure["locationId"]; !ok {
		t.Errorf("expected camelCase 'locationId' from protojson, got keys: %v", keys(pressure))
	}

	pollen := resp["pollen"].(map[string]interface{})["house-nick"].(map[string]interface{})
	if pollen["dominantType"] != "TREE" {
		t.Errorf("Expected dominantType 'TREE', got %v", pollen["dominantType"])
	}
}

// TestDashboardHandler_GetDashboardByLocation_GrpcError covers the *fatal*
// gRPC error codes — the ones that should fail the whole request rather
// than be tolerated as "section absent." NotFound is intentionally absent
// from this table; per the partial-data contract, NotFound from any single
// provider is non-fatal (see the PartialData / AllSectionsAbsent tests).
func TestDashboardHandler_GetDashboardByLocation_GrpcError(t *testing.T) {
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
			handler := NewDashboardHandler(&errorWeatherClient{err: tt.grpcErr}, &mockPollenClient{})
			req := requestWithLocationID(t, "house-nick")
			rr := httptest.NewRecorder()

			handler.GetDashboardByLocation(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

// nilReturnWeatherClient returns (nil, nil) for every call — simulating a
// misbehaving provider that responds with no error but also no data. Without
// a defensive nil-check, the aggregate helpers would panic on the nil proto.
type nilReturnWeatherClient struct{}

func (m *nilReturnWeatherClient) GetPressureStat(ctx context.Context, locationID string) (*weatherPb.PressureStat, error) {
	return nil, nil
}
func (m *nilReturnWeatherClient) GetPressureStats(ctx context.Context) ([]*weatherPb.PressureStat, error) {
	return nil, nil
}
func (m *nilReturnWeatherClient) GetLastWeather(ctx context.Context, locationID string) (*weatherPb.Weather, error) {
	return nil, nil
}
func (m *nilReturnWeatherClient) GetAllLastWeather(ctx context.Context) ([]*weatherPb.Weather, error) {
	return nil, nil
}

// TestDashboardHandler_GetDashboardByLocation_PartialData_WeatherMissingPollenPresent
// locks in the partial-data contract: when a provider returns (nil, nil)
// for a location (weather/pressure here), we don't fail the request — we
// return 200 with the absent sections rendered as empty JSON maps and the
// present sections populated normally. This is the shape the CLI's
// Response.Merge depends on (empty src maps are no-ops on the dst).
func TestDashboardHandler_GetDashboardByLocation_PartialData_WeatherMissingPollenPresent(t *testing.T) {
	handler := NewDashboardHandler(&nilReturnWeatherClient{}, &mockPollenClient{})
	req := requestWithLocationID(t, "house-nick")
	rr := httptest.NewRecorder()

	handler.GetDashboardByLocation(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 with partial data, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var body map[string]map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := len(body["weather"]); got != 0 {
		t.Errorf("weather map should be empty (nil-provider section), got %d entries", got)
	}
	if got := len(body["pressure"]); got != 0 {
		t.Errorf("pressure map should be empty (nil-provider section), got %d entries", got)
	}
	if _, ok := body["pollen"]["house-nick"]; !ok {
		t.Errorf("pollen[house-nick] should be present, body: %s", rr.Body.String())
	}
}

// TestDashboardHandler_GetDashboardByLocation_AllSectionsAbsent_Returns404
// is the "wholly absent" boundary: when every provider signals NotFound,
// there's nothing partial to return, so we surface 404 rather than a
// confusingly-empty 200.
func TestDashboardHandler_GetDashboardByLocation_AllSectionsAbsent_Returns404(t *testing.T) {
	notFound := status.Error(codes.NotFound, "no data for location")
	handler := NewDashboardHandler(
		&errorWeatherClient{err: notFound},
		&errorPollenClient{err: notFound},
	)
	req := requestWithLocationID(t, "house-nick")
	rr := httptest.NewRecorder()

	handler.GetDashboardByLocation(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 when every section is NotFound, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

// TestDashboardHandler_GetDashboardByLocation_PollenNotFound_PartialData
// covers the per-section partial-data path from the other side: pollen is
// the missing section this time, weather and pressure are populated.
// Pairs with the WeatherMissingPollenPresent test above to lock in that
// any single-section NotFound is non-fatal regardless of which one.
func TestDashboardHandler_GetDashboardByLocation_PollenNotFound_PartialData(t *testing.T) {
	handler := NewDashboardHandler(
		&mockWeatherClient{},
		&errorPollenClient{err: status.Error(codes.NotFound, "no pollen for location")},
	)
	req := requestWithLocationID(t, "house-nick")
	rr := httptest.NewRecorder()

	handler.GetDashboardByLocation(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 (pollen NotFound is non-fatal), got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var body map[string]map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := len(body["pollen"]); got != 0 {
		t.Errorf("pollen map should be empty (NotFound), got %d entries", got)
	}
	if _, ok := body["weather"]["house-nick"]; !ok {
		t.Errorf("weather[house-nick] should be present, body: %s", rr.Body.String())
	}
	if _, ok := body["pressure"]["house-nick"]; !ok {
		t.Errorf("pressure[house-nick] should be present, body: %s", rr.Body.String())
	}
}

// TestDashboardHandler_GetDashboardByLocation_PollenUnavailable_Fatal
// is the negative pair to PollenNotFound_PartialData: a non-NotFound
// pollen error (Unavailable here) is *not* tolerated and should fail the
// whole request with the appropriate 5xx, even though weather and
// pressure would otherwise be returnable. This is what protects callers
// from silently consuming a partial response when an upstream is broken
// rather than empty.
func TestDashboardHandler_GetDashboardByLocation_PollenUnavailable_Fatal(t *testing.T) {
	handler := NewDashboardHandler(
		&mockWeatherClient{},
		&errorPollenClient{err: status.Error(codes.Unavailable, "pollen-provider down")},
	)
	req := requestWithLocationID(t, "house-nick")
	rr := httptest.NewRecorder()

	handler.GetDashboardByLocation(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when pollen returns Unavailable, got %d", rr.Code)
	}
}

func TestDashboardHandler_GetDashboardByLocation_SlowProviderTimesOut(t *testing.T) {
	handler := NewDashboardHandler(
		&slowWeatherClient{delay: 10 * time.Second},
		&mockPollenClient{},
	)
	req := requestWithLocationID(t, "house-nick")
	rr := httptest.NewRecorder()

	start := time.Now()
	handler.GetDashboardByLocation(rr, req)
	elapsed := time.Since(start)

	if rr.Code != http.StatusGatewayTimeout {
		t.Errorf("expected status 504, got %d", rr.Code)
	}
	if elapsed > shared.RPCClientTimeout+1*time.Second {
		t.Errorf("expected per-RPC timeout to fire within budget, but took %s", elapsed)
	}
}
