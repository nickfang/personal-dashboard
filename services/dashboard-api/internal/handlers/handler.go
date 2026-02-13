package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	pb "github.com/nickfang/personal-dashboard/services/gen/go/weather-provider/v1"
)

// WeatherFetcher defines the dependency on the weather client
type WeatherFetcher interface {
	GetWeatherStat(ctx context.Context, locationID string) (*pb.PressureStat, error)
	GetWeatherStats(ctx context.Context) ([]*pb.PressureStat, error)
}

type DashboardHandler struct {
	weatherClient WeatherFetcher
}

func NewDashboardHandler(wc WeatherFetcher) *DashboardHandler {
	return &DashboardHandler{
		weatherClient: wc,
	}
}

func aggregatePressureStats(pressureStats []*pb.PressureStat) map[string]interface{} {
	aggregatedData := make(map[string]interface{})
	for _, stat := range pressureStats {
		aggregatedData[stat.LocationId] = stat
	}
	return aggregatedData
}

func (h *DashboardHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	// 1. Fetch data from weather client
	pressureStats, err := h.weatherClient.GetWeatherStats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// 2. Aggregate with other future data
	aggregatedData := aggregatePressureStats(pressureStats)

	// 3. Respond with JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"pressure": aggregatedData,
	}); err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
	}
}
