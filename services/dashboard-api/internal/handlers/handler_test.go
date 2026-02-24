package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	pollenPb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/pollen-provider/v1"
	weatherPb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/weather-provider/v1"
	"github.com/nickfang/personal-dashboard/services/shared"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// --- Weather mocks ---

type mockWeatherClient struct{}

func (m *mockWeatherClient) GetPressureStat(ctx context.Context, locationID string) (*weatherPb.PressureStat, error) {
	return &weatherPb.PressureStat{
		LocationId: locationID,
		Trend:      "rising",
		Delta_1H:   0.5,
	}, nil
}

func (m *mockWeatherClient) GetPressureStats(ctx context.Context) ([]*weatherPb.PressureStat, error) {
	return []*weatherPb.PressureStat{
		{
			LocationId: "house-nick",
			Trend:      "rising",
			Delta_1H:   0.5,
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

func keys(m map[string]interface{}) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}
