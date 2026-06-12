package handlers

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	pollenPb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/pollen-provider/v1"
	pressurePb "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/gen/go/weather-provider/v1"
)

func formatPressureText(pressureStats []*pressurePb.PressureStat) map[string]string {
	pressureByLocation := make(map[string]string)
	sortedPressureStats := slices.Clone(pressureStats)
	slices.SortFunc(sortedPressureStats, func(a, b *pressurePb.PressureStat) int {
		return strings.Compare(a.LocationId, b.LocationId)
	})
	for _, pressureStat := range sortedPressureStats {
		location := pressureStat.LocationId
		var pressureText strings.Builder

		pressureText.WriteString(fmt.Sprintf("Pressure: %s\n", pressureStat.LastUpdated.AsTime().Local().Format("2006.01.02 15:04:05")))
		pressureText.WriteString(fmt.Sprintf("  %s\n", pressureStat.Trend))
		pressureText.WriteString(fmt.Sprintf("  Deltas: %.2f(1h), %.2f(3h), %.2f(6h) %.2f(12h) %.2f(24h)\n", pressureStat.Delta_1H, pressureStat.Delta_3H, pressureStat.Delta_6H, pressureStat.Delta_12H, pressureStat.Delta_24H))
		pressureByLocation[location] = pressureText.String()
	}
	return pressureByLocation
}

func formatWeatherText(weathers []*pressurePb.Weather) map[string]string {
	weatherByLocation := make(map[string]string)
	sortedWeathers := slices.Clone(weathers)
	slices.SortFunc(sortedWeathers, func(a, b *pressurePb.Weather) int {
		return strings.Compare(a.LocationId, b.LocationId)
	})
	for _, weather := range sortedWeathers {
		location := weather.LocationId
		var weatherText strings.Builder
		weatherText.WriteString(fmt.Sprintf("Weather: %s\n", weather.LastUpdated.AsTime().Local().Format("2006.01.02 15:04:05")))
		weatherText.WriteString(fmt.Sprintf("  Temp: %.2fF\n", weather.TempF))
		weatherText.WriteString(fmt.Sprintf("  Feels Like: %.2fF\n", weather.TempFeelF))
		weatherText.WriteString(fmt.Sprintf("  Humidity: %d%%\n", weather.HumidityPercent))
		weatherText.WriteString(fmt.Sprintf("  Precipitation: %d%%\n", weather.PrecipitationPercent))
		weatherText.WriteString(fmt.Sprintf("  Pressure: %.2fmb\n", weather.PressureMb))
		weatherByLocation[location] = weatherText.String()
	}
	return weatherByLocation
}

func formatPollenText(pollenReports []*pollenPb.PollenReport) map[string]string {
	pollenByLocation := make(map[string]string)
	sortedPollenReports := slices.Clone(pollenReports)
	slices.SortFunc(sortedPollenReports, func(a, b *pollenPb.PollenReport) int {
		return strings.Compare(a.LocationId, b.LocationId)
	})
	for _, pollenReport := range sortedPollenReports {
		if len(pollenReport.Plants) == 0 {
			pollenByLocation[pollenReport.LocationId] = "No pollen data available"
			continue
		}
		location := pollenReport.LocationId
		var pollenText strings.Builder
		pollenText.WriteString(fmt.Sprintf("Pollen: %s\n", pollenReport.CollectedAt.AsTime().Local().Format("2006.01.02 15:04:05")))
		sortedPlants := slices.Clone(pollenReport.Plants)
		slices.SortFunc(sortedPlants, func(a, b *pollenPb.PollenPlant) int {
			return int(b.Index) - int(a.Index)
		})
		first := true
		currentIndex := sortedPlants[0].Index + 1
		for _, plant := range sortedPlants {
			if plant.Index < 1 {
				break
			}
			if currentIndex != plant.Index {
				if first {
					first = false
				} else {
					pollenText.WriteString("\n")
				}
				pollenText.WriteString(fmt.Sprintf("  %-10s", plant.Category))
				currentIndex = plant.Index
			}
			inSeason := "Out of Season"
			if plant.InSeason {
				inSeason = "In Season"
			}
			pollenText.WriteString(fmt.Sprintf(" %s (%s)", plant.DisplayName, inSeason))
		}
		pollenText.WriteString("\n")
		pollenByLocation[location] = pollenText.String()
		// Don't show types.  It's not useful.
	}
	return pollenByLocation
}

// forecastPressureAt finds the pressure delta from the first point to the
// point nearest first+hours, within a 45-minute tolerance (resilient to
// missing forecast hours, like the collector's delta search).
func forecastDeltaAt(points []*pressurePb.ForecastPoint, hours int) (float64, bool) {
	if len(points) == 0 {
		return 0, false
	}
	const tolerance = 45 * time.Minute
	base := points[0]
	if base.ValidTime == nil {
		return 0, false
	}
	target := base.ValidTime.AsTime().Add(time.Duration(hours) * time.Hour)

	bestIdx := -1
	minDiff := tolerance + time.Second
	for i := 1; i < len(points); i++ {
		if points[i].ValidTime == nil {
			continue
		}
		diff := points[i].ValidTime.AsTime().Sub(target)
		if diff < 0 {
			diff = -diff
		}
		if diff <= tolerance && diff < minDiff {
			minDiff = diff
			bestIdx = i
		}
	}
	if bestIdx < 0 {
		return 0, false
	}
	return points[bestIdx].PressureMb - base.PressureMb, true
}

// forecastLow returns the lowest pressure within the given horizon from the
// first point.
func forecastLow(points []*pressurePb.ForecastPoint, hours int) (float64, bool) {
	if len(points) == 0 || points[0].ValidTime == nil {
		return 0, false
	}
	cutoff := points[0].ValidTime.AsTime().Add(time.Duration(hours) * time.Hour)
	low := points[0].PressureMb
	for _, p := range points[1:] {
		if p.ValidTime == nil {
			continue
		}
		if p.ValidTime.AsTime().After(cutoff) {
			break
		}
		if p.PressureMb < low {
			low = p.PressureMb
		}
	}
	return low, true
}

// forecastTrend mirrors the WMO 3h barometric-tendency bands used by the
// weather-collector's pressure analysis.
func forecastTrend(points []*pressurePb.ForecastPoint) string {
	delta3h, ok := forecastDeltaAt(points, 3)
	if !ok {
		return "unknown"
	}
	switch {
	case delta3h > 0.5:
		return "rising"
	case delta3h < -0.5:
		return "falling"
	default:
		return "stable"
	}
}

func formatForecastText(forecasts []*pressurePb.Forecast) map[string]string {
	forecastByLocation := make(map[string]string)
	sortedForecasts := slices.Clone(forecasts)
	slices.SortFunc(sortedForecasts, func(a, b *pressurePb.Forecast) int {
		return strings.Compare(a.LocationId, b.LocationId)
	})
	for _, forecast := range sortedForecasts {
		var text strings.Builder
		issuedAt := "unknown"
		if forecast.IssuedAt != nil {
			issuedAt = forecast.IssuedAt.AsTime().Local().Format("2006.01.02 15:04:05")
		}
		text.WriteString(fmt.Sprintf("Forecast: %s\n", issuedAt))

		line := "  " + forecastTrend(forecast.Points)
		if low, ok := forecastLow(forecast.Points, 48); ok {
			line += fmt.Sprintf(", low %.2f mb (48h)", low)
		}
		text.WriteString(line + "\n")

		var deltas []string
		for _, hours := range []int{3, 6, 12, 24, 48} {
			if delta, ok := forecastDeltaAt(forecast.Points, hours); ok {
				deltas = append(deltas, fmt.Sprintf("%+.2f(+%dh)", delta, hours))
			}
		}
		if len(deltas) > 0 {
			text.WriteString("  Deltas: " + strings.Join(deltas, ", ") + "\n")
		}

		for _, alert := range forecast.Alerts {
			if alert.Status == "active" {
				text.WriteString(fmt.Sprintf("  ⚠ %s\n", alert.Message))
			}
		}
		forecastByLocation[forecast.LocationId] = text.String()
	}
	return forecastByLocation
}

func formatDashboardText(pressureStats []*pressurePb.PressureStat, pollenReports []*pollenPb.PollenReport, lastWeathers []*pressurePb.Weather, forecasts []*pressurePb.Forecast) (string, error) {
	pressureByLocation := formatPressureText(pressureStats)
	pollenByLocation := formatPollenText(pollenReports)
	weatherByLocation := formatWeatherText(lastWeathers)
	forecastByLocation := formatForecastText(forecasts)

	locations := make(map[string]struct{})
	for location := range pressureByLocation {
		locations[location] = struct{}{}
	}
	for location := range pollenByLocation {
		locations[location] = struct{}{}
	}
	for location := range forecastByLocation {
		locations[location] = struct{}{}
	}

	var data strings.Builder
	for _, location := range slices.Sorted(maps.Keys(locations)) {
		data.WriteString(fmt.Sprintf("---------------- %s ----------------\n", location))
		data.WriteString(weatherByLocation[location])
		data.WriteString(pressureByLocation[location])
		data.WriteString(forecastByLocation[location])
		data.WriteString(pollenByLocation[location])
	}
	return data.String(), nil
}
