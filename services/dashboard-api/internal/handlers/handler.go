package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	pollenPb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/pollen-provider/v1"
	pressurePb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/weather-provider/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

// WeatherFetcher defines the dependency on the weather client
type WeatherFetcher interface {
	GetPressureStat(ctx context.Context, locationID string) (*pressurePb.PressureStat, error)
	GetPressureStats(ctx context.Context) ([]*pressurePb.PressureStat, error)
}

type PollenFetcher interface {
	GetPollenReport(ctx context.Context, locationID string) (*pollenPb.PollenReport, error)
	GetPollenReports(ctx context.Context) ([]*pollenPb.PollenReport, error)
}

type DashboardHandler struct {
	weatherClient WeatherFetcher
	pollenClient  PollenFetcher
}

func NewDashboardHandler(wc WeatherFetcher, pc PollenFetcher) *DashboardHandler {
	return &DashboardHandler{
		weatherClient: wc,
		pollenClient:  pc,
	}
}

// protojson produces camelCase field names and RFC 3339 timestamps,
// which is what the frontend expects. encoding/json would produce
// snake_case and raw {seconds, nanos} objects from proto structs.
var protoMarshaler = protojson.MarshalOptions{}

func aggregatePressureStats(pressureStats []*pressurePb.PressureStat) (map[string]json.RawMessage, error) {
	aggregatedData := make(map[string]json.RawMessage, len(pressureStats))
	for _, stat := range pressureStats {
		data, err := protoMarshaler.Marshal(stat)
		if err != nil {
			return nil, err
		}
		aggregatedData[stat.LocationId] = data
	}
	return aggregatedData, nil
}

func aggregatePollenReports(pollenReports []*pollenPb.PollenReport) (map[string]json.RawMessage, error) {
	aggregatedData := make(map[string]json.RawMessage, len(pollenReports))
	for _, report := range pollenReports {
		data, err := protoMarshaler.Marshal(report)
		if err != nil {
			return nil, err
		}
		aggregatedData[report.LocationId] = data
	}
	return aggregatedData, nil
}

func (h *DashboardHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	// 1. Fetch data from clients
	pressureStats, err := h.weatherClient.GetPressureStats(r.Context())
	if err != nil {
		RespondWithGrpcError(w, err, "Failed to fetch weather statistics")
		return
	}
	pollenReports, err := h.pollenClient.GetPollenReports(r.Context())
	if err != nil {
		RespondWithGrpcError(w, err, "Failed to fetch pollen reports")
		return
	}

	// 2. Aggregate with other future data
	aggregatedPressure, err := aggregatePressureStats(pressureStats)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	aggregatedPollen, err := aggregatePollenReports(pollenReports)
	if err != nil {
		http.Error(w, "Failed to encode pollen response", http.StatusInternalServerError)
	}

	// 3. Respond with JSON (encoding/json handles json.RawMessage values
	// by embedding them verbatim, so the protojson output passes through)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"pressure": aggregatedPressure,
		"pollen":   aggregatedPollen,
	}); err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
	}
}
