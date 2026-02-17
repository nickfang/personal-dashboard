package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	pb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/weather-provider/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

// WeatherFetcher defines the dependency on the weather client
type WeatherFetcher interface {
	GetPressureStat(ctx context.Context, locationID string) (*pb.PressureStat, error)
	GetPressureStats(ctx context.Context) ([]*pb.PressureStat, error)
}

type DashboardHandler struct {
	weatherClient WeatherFetcher
}

func NewDashboardHandler(wc WeatherFetcher) *DashboardHandler {
	return &DashboardHandler{
		weatherClient: wc,
	}
}

// protojson produces camelCase field names and RFC 3339 timestamps,
// which is what the frontend expects. encoding/json would produce
// snake_case and raw {seconds, nanos} objects from proto structs.
var protoMarshaler = protojson.MarshalOptions{}

func aggregatePressureStats(pressureStats []*pb.PressureStat) (map[string]json.RawMessage, error) {
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

func (h *DashboardHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	// 1. Fetch data from weather client
	pressureStats, err := h.weatherClient.GetPressureStats(r.Context())
	if err != nil {
		RespondWithGrpcError(w, err, "Failed to fetch weather statistics")
		return
	}

	// 2. Aggregate with other future data
	aggregatedData, err := aggregatePressureStats(pressureStats)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	// 3. Respond with JSON (encoding/json handles json.RawMessage values
	// by embedding them verbatim, so the protojson output passes through)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"pressure": aggregatedData,
	}); err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
	}
}
