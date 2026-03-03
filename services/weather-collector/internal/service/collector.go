package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nickfang/personal-dashboard/services/shared"
	"github.com/nickfang/personal-dashboard/services/weather-collector/internal/api"
	"github.com/nickfang/personal-dashboard/services/weather-collector/internal/repository"
)

const (
	DeltaTolerance      = 45 * time.Minute
	DeltaNoiseThreshold = 0.5 // mb
)

// CollectorService orchestrates the weather collection flow.
type CollectorService struct {
	fetcher api.Fetcher
	writer  repository.Writer
}

// NewCollectorService creates a new CollectorService with injected dependencies.
func NewCollectorService(fetcher api.Fetcher, writer repository.Writer) *CollectorService {
	return &CollectorService{fetcher: fetcher, writer: writer}
}

// Collect fetches weather data for a location, maps it, and writes to storage.
func (s *CollectorService) Collect(ctx context.Context, apiKey string, location shared.Location) error {
	apiResp, err := s.fetcher.Fetch(apiKey, location)
	if err != nil {
		return fmt.Errorf("fetching weather data for %s: %w", location.ID, err)
	}
	wp, err := MapToWeatherPoint(location.ID, *apiResp)
	if err != nil {
		return fmt.Errorf("mapping weather data for %s: %w", location.ID, err)
	}
	if err := s.writer.SaveRaw(ctx, *wp); err != nil {
		return fmt.Errorf("saving weather data for %s: %w", location.ID, err)
	}
	if err := s.writer.UpdateCache(ctx, location.ID, *wp, CalculatePressureStats); err != nil {
		return fmt.Errorf("updating weather cache for %s: %w", location.ID, err)
	}
	return nil
}

// MapToWeatherPoint converts an API response into a WeatherPoint for storage.
// Returns an error if the data is invalid (e.g., 0.0 pressure).
func MapToWeatherPoint(locationID string, data api.WeatherAPIResponse) (*repository.WeatherPoint, error) {
	// Strict Data Validation:
	// If pressure is 0.0, we assume the API response is incomplete or corrupted.
	// Saving a 0.0 pressure reading ruins statistical analysis (deltas).
	if data.AirPressure.MeanSeaLevelMillibars == 0 {
		return nil, fmt.Errorf("invalid pressure data (0.0) received for %s. Full payload: %+v", locationID, data)
	}

	wp := &repository.WeatherPoint{
		Location:             locationID,
		Timestamp:            time.Now(),
		TempC:                data.Temperature.Degrees,
		TempF:                CtoF(data.Temperature.Degrees),
		TempFeelC:            data.FeelsLikeTemperature.Degrees,
		TempFeelF:            CtoF(data.FeelsLikeTemperature.Degrees),
		HumidityPercent:      data.RelativeHumidityPercent,
		UVIndex:              data.UVIndex,
		PressureMb:           data.AirPressure.MeanSeaLevelMillibars,
		WindDirDeg:           data.Wind.Direction.Degrees,
		WindSpeedKph:         data.Wind.Speed.Value,
		WindSpeedMph:         KtoM(data.Wind.Speed.Value),
		WindGustKph:          data.Wind.Gust.Value,
		WindGustMph:          KtoM(data.Wind.Gust.Value),
		VisibilityKm:         data.Visibility.Distance,
		VisibilityM:          KtoM(data.Visibility.Distance),
		DewpointC:            data.DewPoint.Degrees,
		DewpointF:            CtoF(data.DewPoint.Degrees),
		PrecipitationPercent: data.Precipitation.Probability.Percent,
	}

	// Structured Debug Log - Contains all mapping info for troubleshooting
	slog.Debug("Mapped Weather Data [DB Format]",
		"location", locationID,
		"timestamp", wp.Timestamp,
		"temp_c", wp.TempC,
		"feels_like_c", wp.TempFeelC,
		"humidity_pct", wp.HumidityPercent,
		"uv_index", wp.UVIndex,
		"pressure_mb", wp.PressureMb,
		"wind_dir_deg", wp.WindDirDeg,
		"wind_speed_kph", wp.WindSpeedKph,
		"wind_gust_kph", wp.WindGustKph,
		"visibility_km", wp.VisibilityKm,
		"dewpoint_c", wp.DewpointC,
		"precipitation_pct", wp.PrecipitationPercent,
	)

	return wp, nil
}

// CalculatePressureStats computes barometric pressure deltas and trend from history.
// Used as the AnalyzeFunc callback passed to repository.Writer.UpdateCache.
func CalculatePressureStats(history []repository.PressurePoint) repository.PressureStats {
	stats := repository.PressureStats{Trend: "unknown"}

	if len(history) < 2 {
		return stats
	}

	current := history[len(history)-1]

	// Log audit info once per location
	type deltaAudit struct {
		Target string   `json:"target"`
		Found  string   `json:"found,omitempty"`
		Delta  *float64 `json:"delta,omitempty"`
	}
	audit := make(map[string]deltaAudit)

	// getDelta uses a Time-Window Search instead of array indices.
	// This decouples logic from the sampling rate and makes it resilient to
	// missing data points or job scheduling jitter.
	getDelta := func(hoursAgo int) *float64 {
		targetTime := current.TimeStamp.Add(time.Duration(-hoursAgo) * time.Hour)
		// 45 minute tolerance allows us to find the closest point even if
		// some cycles were missed or delayed.
		tolerance := DeltaTolerance

		var bestMatch *repository.PressurePoint
		minDiff := tolerance + (1 * time.Second)

		for i := len(history) - 2; i >= 0; i-- {
			p := &history[i]

			diff := p.TimeStamp.Sub(targetTime)
			if diff < 0 {
				diff = -diff
			}

			if diff <= tolerance {
				if diff < minDiff {
					minDiff = diff
					bestMatch = p
				}
			}

			// Optimization: History is sorted ascending; if we are way before
			// the target window, we can safely stop searching.
			if targetTime.Sub(p.TimeStamp) > tolerance {
				break
			}
		}

		key := fmt.Sprintf("%dh", hoursAgo)
		entry := deltaAudit{Target: targetTime.Format(time.RFC3339)}

		if bestMatch != nil {
			entry.Found = bestMatch.TimeStamp.Format(time.RFC3339)
			res := current.PressureMb - bestMatch.PressureMb
			entry.Delta = &res
			audit[key] = entry
			return &res
		}

		audit[key] = entry
		return nil
	}

	stats.Delta1h = getDelta(1)
	stats.Delta3h = getDelta(3)
	stats.Delta6h = getDelta(6)
	stats.Delta12h = getDelta(12)
	stats.Delta24h = getDelta(24)

	// Log the timestamp and value audit info at INFO level
	slog.Info("Pressure Analysis Diagnostics",
		"current_time", current.TimeStamp.Format(time.RFC3339),
		"analysis", audit,
	)

	// Simple trend logic with noise threshold
	// NOTE: We rely exclusively on the 3-Hour Delta (Delta3h) to define the "Trend" string.
	// This adheres to the World Meteorological Organization (WMO) definition of "Barometric Tendency".
	if stats.Delta3h != nil {
		d3 := *stats.Delta3h
		if d3 > DeltaNoiseThreshold {
			stats.Trend = "rising"
		} else if d3 < -DeltaNoiseThreshold {
			stats.Trend = "falling"
		} else {
			stats.Trend = "stable"
		}
	}

	return stats
}
