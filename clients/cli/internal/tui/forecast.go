package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/nickfang/personal-dashboard/clients/cli/internal/client"
)

// forecastDeltaHours are the forward-looking offsets shown in the FORECAST
// delta block, mirroring the PRESSURE section's backward-looking deltas.
var forecastDeltaHours = []int{3, 6, 12, 24, 48}

// forecastLowHorizonHours bounds the "low N mb" lookup so it describes the
// same span as the widest delta column.
const forecastLowHorizonHours = 48

// parsedPoint is a forecast point with its timestamp parsed; points with
// unparseable timestamps are dropped before analysis.
type parsedPoint struct {
	at         time.Time
	pressureMb float64
}

func parsePoints(points []client.ForecastPoint) []parsedPoint {
	parsed := make([]parsedPoint, 0, len(points))
	for _, p := range points {
		at, err := time.Parse(time.RFC3339, p.ValidTime)
		if err != nil {
			continue
		}
		parsed = append(parsed, parsedPoint{at: at, pressureMb: p.PressureMb})
	}
	return parsed
}

// deltaAt returns the pressure change from the first point to the point
// nearest first+hours, within a 45-minute tolerance.
func deltaAt(points []parsedPoint, hours int) (float64, bool) {
	if len(points) < 2 {
		return 0, false
	}
	const tolerance = 45 * time.Minute
	target := points[0].at.Add(time.Duration(hours) * time.Hour)

	bestIdx := -1
	minDiff := tolerance + time.Second
	for i := 1; i < len(points); i++ {
		diff := points[i].at.Sub(target)
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
	return points[bestIdx].pressureMb - points[0].pressureMb, true
}

// forecastTrendArrow classifies the next-3h delta using the same WMO ±0.5 mb
// bands as the PRESSURE section's trend.
func forecastTrendArrow(points []parsedPoint) (string, string) {
	delta3h, ok := deltaAt(points, 3)
	if !ok {
		return "·", "Unknown"
	}
	switch {
	case delta3h > 0.5:
		return "▲", "Rising"
	case delta3h < -0.5:
		return "▼", "Falling"
	default:
		return "→", "Steady"
	}
}

// forecastLowMb returns the lowest pressure within the horizon.
func forecastLowMb(points []parsedPoint, hours int) (float64, bool) {
	if len(points) == 0 {
		return 0, false
	}
	cutoff := points[0].at.Add(time.Duration(hours) * time.Hour)
	low := points[0].pressureMb
	for _, p := range points[1:] {
		if p.at.After(cutoff) {
			break
		}
		if p.pressureMb < low {
			low = p.pressureMb
		}
	}
	return low, true
}

// forecastDeltaBlock formats the forward deltas as a two-row table (labels
// above values), matching the PRESSURE section's deltaBlock. Horizons not
// covered by the forecast render as "--".
func forecastDeltaBlock(points []parsedPoint) string {
	labels := make([]string, 0, len(forecastDeltaHours))
	values := make([]string, 0, len(forecastDeltaHours))
	for _, hours := range forecastDeltaHours {
		labels = append(labels, fmt.Sprintf("%6s", fmt.Sprintf("Δ+%dh", hours)))
		if delta, ok := deltaAt(points, hours); ok {
			values = append(values, fmt.Sprintf("%+6.2f", delta))
		} else {
			values = append(values, fmt.Sprintf("%6s", "--"))
		}
	}
	return strings.Join(labels, "") + "\n" + strings.Join(values, "")
}

// renderForecast renders the FORECAST section body: a trend line with the
// 48h low, then the forward delta block.
func renderForecast(f *client.Forecast) string {
	if f == nil {
		return "  (no forecast data)"
	}
	points := parsePoints(f.Points)
	if len(points) == 0 {
		return "  (no forecast data)"
	}

	arrow, label := forecastTrendArrow(points)
	line1 := fmt.Sprintf("  %s %s", arrow, label)
	if low, ok := forecastLowMb(points, forecastLowHorizonHours); ok {
		line1 += fmt.Sprintf("   low %.0f mb (%dh)", low, forecastLowHorizonHours)
	}
	return line1 + "\n" + forecastDeltaBlock(points)
}

// alertBanner renders one line per active alert, styled by severity. Returns
// an empty string when there is nothing to show — resolved alerts never
// render. An alert stays active, and so keeps rendering, after the user has
// been notified about it; delivery is tracked separately server-side.
func alertBanner(alerts []client.Alert) string {
	var lines []string
	for _, a := range alerts {
		if a.Status != "active" {
			continue
		}
		line := "⚠ " + a.Message
		if a.Severity == "severe" {
			line = ErrorStyle.Render(line)
		} else {
			line = WarnStyle.Render(line)
		}
		lines = append(lines, " "+line)
	}
	return strings.Join(lines, "\n")
}
