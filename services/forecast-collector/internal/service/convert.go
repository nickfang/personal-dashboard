package service

import (
	"fmt"
	"log/slog"

	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/api"
	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/repository"
)

// CtoF converts Celsius to Fahrenheit.
func CtoF(c float64) float64 {
	return (c * 1.8) + 32
}

// MapToForecastPoint converts one API forecast hour into storage form.
// Returns an error if the data is invalid (e.g., 0.0 pressure), matching the
// strict validation in weather-collector: a 0.0 pressure reading would ruin
// delta analysis and alert detection.
func MapToForecastPoint(h api.ForecastHour) (repository.ForecastPoint, error) {
	if h.AirPressure.MeanSeaLevelMillibars == 0 {
		return repository.ForecastPoint{}, fmt.Errorf("invalid pressure data (0.0) for hour %s", h.Interval.StartTime)
	}
	return repository.ForecastPoint{
		ValidTime:            h.Interval.StartTime,
		TempC:                h.Temperature.Degrees,
		TempF:                CtoF(h.Temperature.Degrees),
		TempFeelC:            h.FeelsLikeTemperature.Degrees,
		TempFeelF:            CtoF(h.FeelsLikeTemperature.Degrees),
		DewpointC:            h.DewPoint.Degrees,
		DewpointF:            CtoF(h.DewPoint.Degrees),
		HumidityPercent:      h.RelativeHumidityPercent,
		UVIndex:              h.UVIndex,
		PrecipitationPercent: h.Precipitation.Probability.Percent,
		PressureMb:           h.AirPressure.MeanSeaLevelMillibars,
		WindDirDeg:           h.Wind.Direction.Degrees,
		WindSpeedKph:         h.Wind.Speed.Value,
		WindGustKph:          h.Wind.Gust.Value,
	}, nil
}

// MapRun converts all API forecast hours into storage points, skipping
// invalid hours so one bad entry doesn't discard the whole run.
func MapRun(hours []api.ForecastHour) []repository.ForecastPoint {
	points := make([]repository.ForecastPoint, 0, len(hours))
	for _, h := range hours {
		p, err := MapToForecastPoint(h)
		if err != nil {
			slog.Warn("Skipping invalid forecast hour", "error", err)
			continue
		}
		points = append(points, p)
	}
	return points
}
