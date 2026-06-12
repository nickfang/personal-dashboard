package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"golang.org/x/sync/errgroup"

	pollenPb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/pollen-provider/v1"
	pressurePb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/weather-provider/v1"
	"github.com/nickfang/personal-dashboard/services/shared"
	"google.golang.org/protobuf/encoding/protojson"
)

type WeatherFetcher interface {
	GetPressureStat(ctx context.Context, locationID string) (*pressurePb.PressureStat, error)
	GetPressureStats(ctx context.Context) ([]*pressurePb.PressureStat, error)
	GetLastWeather(ctx context.Context, locationID string) (*pressurePb.Weather, error)
	GetAllLastWeather(ctx context.Context) ([]*pressurePb.Weather, error)
}

type PollenFetcher interface {
	GetPollenReport(ctx context.Context, locationID string) (*pollenPb.PollenReport, error)
	GetPollenReports(ctx context.Context) ([]*pollenPb.PollenReport, error)
}

type ForecastFetcher interface {
	GetForecast(ctx context.Context, locationID string) (*pressurePb.Forecast, error)
	GetAllForecasts(ctx context.Context) ([]*pressurePb.Forecast, error)
}

type DashboardHandler struct {
	weatherClient  WeatherFetcher
	pollenClient   PollenFetcher
	forecastClient ForecastFetcher
}

func NewDashboardHandler(wc WeatherFetcher, pc PollenFetcher, fc ForecastFetcher) *DashboardHandler {
	return &DashboardHandler{
		weatherClient:  wc,
		pollenClient:   pc,
		forecastClient: fc,
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

func aggregateLastWeather(lastWeathers []*pressurePb.Weather) (map[string]json.RawMessage, error) {
	aggregatedData := make(map[string]json.RawMessage, len(lastWeathers))
	for _, weather := range lastWeathers {
		data, err := protoMarshaler.Marshal(weather)
		if err != nil {
			return nil, err
		}
		aggregatedData[weather.LocationId] = data
	}
	return aggregatedData, nil
}

// aggregateForecasts splits each forecast into the "forecast" map (points,
// no alerts) and the "alerts" map (per-location alert array), so alerts
// aren't shipped twice while still being addressable on their own.
func aggregateForecasts(forecasts []*pressurePb.Forecast) (map[string]json.RawMessage, map[string][]json.RawMessage, error) {
	forecastData := make(map[string]json.RawMessage, len(forecasts))
	alertData := make(map[string][]json.RawMessage, len(forecasts))
	for _, forecast := range forecasts {
		alerts := forecast.Alerts
		forecast.Alerts = nil
		data, err := protoMarshaler.Marshal(forecast)
		if err != nil {
			return nil, nil, err
		}
		forecastData[forecast.LocationId] = data

		raws := make([]json.RawMessage, 0, len(alerts))
		for _, alert := range alerts {
			raw, err := protoMarshaler.Marshal(alert)
			if err != nil {
				return nil, nil, err
			}
			raws = append(raws, raw)
		}
		alertData[forecast.LocationId] = raws
	}
	return forecastData, alertData, nil
}


func (h *DashboardHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	// 1. Fetch data from clients
	var pressureStats []*pressurePb.PressureStat
	var pollenReports []*pollenPb.PollenReport
	var allLastWeather []*pressurePb.Weather

	g, ctx := errgroup.WithContext(r.Context())

	g.Go(func() error {
		var err error
		rpcCtx, cancel := context.WithTimeout(ctx, shared.RPCClientTimeout)
		defer cancel()
		pressureStats, err = h.weatherClient.GetPressureStats(rpcCtx)
		return err
	})
	g.Go(func() error {
		var err error
		rpcCtx, cancel := context.WithTimeout(ctx, shared.RPCClientTimeout)
		defer cancel()
		allLastWeather, err = h.weatherClient.GetAllLastWeather(rpcCtx)
		return err
	})
	g.Go(func() error {
		var err error
		rpcCtx, cancel := context.WithTimeout(ctx, shared.RPCClientTimeout)
		defer cancel()
		pollenReports, err = h.pollenClient.GetPollenReports(rpcCtx)
		return err
	})

	var forecasts []*pressurePb.Forecast
	g.Go(func() error {
		var err error
		rpcCtx, cancel := context.WithTimeout(ctx, shared.RPCClientTimeout)
		defer cancel()
		forecasts, err = h.forecastClient.GetAllForecasts(rpcCtx)
		return err
	})

	if err := g.Wait(); err != nil {
		RespondWithGrpcError(w, err, "Failed to fetch dashboard data")
		return
	}

	// 2. Respond with text/plain if the user agent is curl
	if strings.Contains(r.Header.Get("User-Agent"), "curl") {
		body, err := formatDashboardText(pressureStats, pollenReports, allLastWeather, forecasts)
		if err != nil {
			http.Error(w, "Failed to format text dashboard", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(body))
		return
	}

	// 3. Aggregate with other future data
	aggregatedPressure, err := aggregatePressureStats(pressureStats)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	aggregatedPollen, err := aggregatePollenReports(pollenReports)
	if err != nil {
		http.Error(w, "Failed to encode pollen response", http.StatusInternalServerError)
		return
	}
	aggregatedLastWeather, err := aggregateLastWeather(allLastWeather)
	if err != nil {
		http.Error(w, "Failed to encode last weather response", http.StatusInternalServerError)
		return
	}
	aggregatedForecast, aggregatedAlerts, err := aggregateForecasts(forecasts)
	if err != nil {
		http.Error(w, "Failed to encode forecast response", http.StatusInternalServerError)
		return
	}

	// 4. Respond with JSON (encoding/json handles json.RawMessage values
	// by embedding them verbatim, so the protojson output passes through).
	// Marshal to buffer first so we can return a clean 500 if encoding fails.
	buf, err := json.Marshal(map[string]any{
		"weather":  aggregatedLastWeather,
		"pressure": aggregatedPressure,
		"pollen":   aggregatedPollen,
		"forecast": aggregatedForecast,
		"alerts":   aggregatedAlerts,
	})
	if err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func (h *DashboardHandler) GetDashboardByLocation(w http.ResponseWriter, r *http.Request) {
	// Chi's path matching guarantees a non-empty {locationID} segment
	// before the handler runs, so we don't re-validate emptiness here.
	locationID := chi.URLParam(r, "locationID")

	var pressureStat *pressurePb.PressureStat
	var pollenReport *pollenPb.PollenReport
	var lastWeather *pressurePb.Weather
	var forecast *pressurePb.Forecast

	g, ctx := errgroup.WithContext(r.Context())

	// Each goroutine swallows a NotFound error (treated as "no data for
	// this section, but the request as a whole is still valid") and lets
	// every other error propagate so errgroup can cancel its siblings and
	// fail the whole request. (nil, nil) returns from a misbehaving
	// provider also fall through as "section absent" — no panic, just no
	// data, same shape as a clean NotFound.
	g.Go(func() error {
		rpcCtx, cancel := context.WithTimeout(ctx, shared.RPCClientTimeout)
		defer cancel()
		s, err := h.weatherClient.GetPressureStat(rpcCtx, locationID)
		if isNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		pressureStat = s
		return nil
	})
	g.Go(func() error {
		rpcCtx, cancel := context.WithTimeout(ctx, shared.RPCClientTimeout)
		defer cancel()
		lw, err := h.weatherClient.GetLastWeather(rpcCtx, locationID)
		if isNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		lastWeather = lw
		return nil
	})
	g.Go(func() error {
		rpcCtx, cancel := context.WithTimeout(ctx, shared.RPCClientTimeout)
		defer cancel()
		p, err := h.pollenClient.GetPollenReport(rpcCtx, locationID)
		if isNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		pollenReport = p
		return nil
	})
	g.Go(func() error {
		rpcCtx, cancel := context.WithTimeout(ctx, shared.RPCClientTimeout)
		defer cancel()
		f, err := h.forecastClient.GetForecast(rpcCtx, locationID)
		if isNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		forecast = f
		return nil
	})

	if err := g.Wait(); err != nil {
		RespondWithGrpcError(w, err, "Failed to fetch dashboard data for location")
		return
	}

	// 404 only when the location is wholly absent across all sections.
	// Partial data (e.g. weather present but pollen NotFound) yields 200
	// with the corresponding maps left empty for the absent sections.
	if pressureStat == nil && lastWeather == nil && pollenReport == nil && forecast == nil {
		http.Error(w, "Location data not found", http.StatusNotFound)
		return
	}

	// Build aggregate inputs only from sections that actually returned data —
	// a one-element slice per present section, nil for absent ones. The
	// aggregate helpers iterate the slice, so nil-input maps come out empty,
	// which is exactly the partial-data shape we want.
	var pressureSlice []*pressurePb.PressureStat
	if pressureStat != nil {
		pressureSlice = []*pressurePb.PressureStat{pressureStat}
	}
	var weatherSlice []*pressurePb.Weather
	if lastWeather != nil {
		weatherSlice = []*pressurePb.Weather{lastWeather}
	}
	var pollenSlice []*pollenPb.PollenReport
	if pollenReport != nil {
		pollenSlice = []*pollenPb.PollenReport{pollenReport}
	}
	var forecastSlice []*pressurePb.Forecast
	if forecast != nil {
		forecastSlice = []*pressurePb.Forecast{forecast}
	}

	aggregatedPressure, err := aggregatePressureStats(pressureSlice)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	aggregatedPollen, err := aggregatePollenReports(pollenSlice)
	if err != nil {
		http.Error(w, "Failed to encode pollen response", http.StatusInternalServerError)
		return
	}
	aggregatedLastWeather, err := aggregateLastWeather(weatherSlice)
	if err != nil {
		http.Error(w, "Failed to encode last weather response", http.StatusInternalServerError)
		return
	}
	aggregatedForecast, aggregatedAlerts, err := aggregateForecasts(forecastSlice)
	if err != nil {
		http.Error(w, "Failed to encode forecast response", http.StatusInternalServerError)
		return
	}

	buf, err := json.Marshal(map[string]any{
		"weather":  aggregatedLastWeather,
		"pressure": aggregatedPressure,
		"pollen":   aggregatedPollen,
		"forecast": aggregatedForecast,
		"alerts":   aggregatedAlerts,
	})
	if err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}
