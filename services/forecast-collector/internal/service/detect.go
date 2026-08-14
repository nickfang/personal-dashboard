package service

import (
	"fmt"
	"time"

	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/repository"
	"github.com/nickfang/personal-dashboard/services/shared"
)

// windowTolerance allows the end-of-window lookup to survive a missing
// or shifted forecast hour, like the weather-collector's delta search.
const windowTolerance = 30 * time.Minute

// displayLocation is the timezone used in alert messages. Cloud Run runs in
// UTC, so the local zone must be loaded explicitly (tzdata in the image).
var displayLocation = func() *time.Location {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// DetectionConfig holds the pressure-drop detection thresholds, sourced from env
// so they can be tuned without a code change.
type DetectionConfig struct {
	DropThresholdMb   float64 // PRESSURE_DROP_MB: warning at this drop per window
	SevereThresholdMb float64 // PRESSURE_SEVERE_MB: severe at this drop per window
	WindowHours       int     // PRESSURE_WINDOW_HOURS: detection window length
}

// DefaultDetectionConfig matches the values deployed via Terraform.
func DefaultDetectionConfig() DetectionConfig {
	return DetectionConfig{DropThresholdMb: 5, SevereThresholdMb: 10, WindowHours: 3}
}

func (c DetectionConfig) ruleID() string {
	return fmt.Sprintf("pressure-drop-%dh", c.WindowHours)
}

// qualifyingWindow is one window position whose pressure delta crossed the
// drop threshold.
type qualifyingWindow struct {
	startIdx, endIdx int
	delta            float64
}

// DetectPressureAlerts slides a WindowHours window across the forecast and
// coalesces overlapping qualifying windows into one alert per continuous
// drop episode. The alert's Value is the steepest single-window delta and
// its window spans the whole episode.
func DetectPressureAlerts(locationID string, points []repository.ForecastPoint, cfg DetectionConfig, now time.Time) []shared.Alert {
	if len(points) < 2 || cfg.WindowHours <= 0 || cfg.DropThresholdMb <= 0 {
		return nil
	}

	var wins []qualifyingWindow
	for i := range points {
		j := pointNearOffset(points, i, time.Duration(cfg.WindowHours)*time.Hour)
		if j < 0 {
			continue
		}
		delta := points[j].PressureMb - points[i].PressureMb
		if delta <= -cfg.DropThresholdMb {
			wins = append(wins, qualifyingWindow{startIdx: i, endIdx: j, delta: delta})
		}
	}
	if len(wins) == 0 {
		return nil
	}

	var alerts []shared.Alert
	episode := []qualifyingWindow{wins[0]}
	for _, w := range wins[1:] {
		last := episode[len(episode)-1]
		if !points[w.startIdx].ValidTime.After(points[last.endIdx].ValidTime) {
			episode = append(episode, w)
			continue
		}
		alerts = append(alerts, buildAlert(locationID, points, episode, cfg, now))
		episode = []qualifyingWindow{w}
	}
	alerts = append(alerts, buildAlert(locationID, points, episode, cfg, now))
	return alerts
}

// pointNearOffset finds the index of the point closest to
// points[i].ValidTime+offset within the tolerance, or -1 if none exists.
func pointNearOffset(points []repository.ForecastPoint, i int, offset time.Duration) int {
	target := points[i].ValidTime.Add(offset)
	best := -1
	minDiff := windowTolerance + time.Second
	for j := i + 1; j < len(points); j++ {
		diff := points[j].ValidTime.Sub(target)
		if diff < 0 {
			diff = -diff
		}
		if diff <= windowTolerance && diff < minDiff {
			minDiff = diff
			best = j
		}
		if points[j].ValidTime.Sub(target) > windowTolerance {
			break
		}
	}
	return best
}

// buildAlert collapses one episode's qualifying windows into a single Alert.
func buildAlert(locationID string, points []repository.ForecastPoint, episode []qualifyingWindow, cfg DetectionConfig, now time.Time) shared.Alert {
	steepest := episode[0]
	for _, w := range episode[1:] {
		if w.delta < steepest.delta {
			steepest = w
		}
	}

	severity := shared.AlertSeverityWarning
	if steepest.delta <= -cfg.SevereThresholdMb {
		severity = shared.AlertSeveritySevere
	}

	alert := shared.Alert{
		Location:    locationID,
		RuleID:      cfg.ruleID(),
		Severity:    severity,
		Value:       steepest.delta,
		Threshold:   cfg.DropThresholdMb,
		WindowStart: points[episode[0].startIdx].ValidTime,
		WindowEnd:   points[episode[len(episode)-1].endIdx].ValidTime,
		Message:     buildMessage(points, steepest, cfg),
		Status:      shared.AlertStatusActive,
		IssuedAt:    now,
	}
	alert.ID = alert.ComputeID()
	return alert
}

// buildMessage renders the client-facing alert text, anchored on the hour
// where the steepest drop begins: "Thu 2 PM  -6.2 mb/3h  -8.1/6h". The
// extended part covers twice the window and is omitted when the forecast
// horizon doesn't reach that far.
func buildMessage(points []repository.ForecastPoint, steepest qualifyingWindow, cfg DetectionConfig) string {
	anchor := points[steepest.startIdx]
	msg := fmt.Sprintf("%s  %+.1f mb/%dh",
		anchor.ValidTime.In(displayLocation).Format("Mon 3 PM"), steepest.delta, cfg.WindowHours)

	extendedHours := 2 * cfg.WindowHours
	if k := pointNearOffset(points, steepest.startIdx, time.Duration(extendedHours)*time.Hour); k >= 0 {
		extendedDelta := points[k].PressureMb - anchor.PressureMb
		msg += fmt.Sprintf("  %+.1f/%dh", extendedDelta, extendedHours)
	}
	return msg
}
