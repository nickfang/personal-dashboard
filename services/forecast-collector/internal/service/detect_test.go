package service

import (
	"strings"
	"testing"
	"time"

	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/repository"
	"github.com/nickfang/personal-dashboard/services/shared"
)

var detectStart = time.Date(2026, 6, 12, 6, 0, 0, 0, time.UTC)

// hourlyPoints builds one forecast point per hour from the given pressures.
func hourlyPoints(pressures ...float64) []repository.ForecastPoint {
	points := make([]repository.ForecastPoint, len(pressures))
	for i, p := range pressures {
		points[i] = repository.ForecastPoint{
			ValidTime:  detectStart.Add(time.Duration(i) * time.Hour),
			PressureMb: p,
		}
	}
	return points
}

func detect(points []repository.ForecastPoint) []shared.Alert {
	return DetectPressureAlerts("house-nick", points, DefaultAlertConfig(), detectStart)
}

func TestDetect_MonotonicDropProducesOneWarning(t *testing.T) {
	// 6 mb drop over the first 3 hours, then flat.
	points := hourlyPoints(1013, 1011, 1009, 1007, 1007, 1007, 1007)

	alerts := detect(points)

	if len(alerts) != 1 {
		t.Fatalf("len = %d, want 1", len(alerts))
	}
	a := alerts[0]
	if a.Severity != shared.AlertSeverityWarning {
		t.Errorf("Severity = %q, want warning", a.Severity)
	}
	if a.Value != -6 {
		t.Errorf("Value = %v, want -6", a.Value)
	}
	if !a.WindowStart.Equal(detectStart) || !a.WindowEnd.Equal(detectStart.Add(3*time.Hour)) {
		t.Errorf("window = %v–%v, want %v–%v", a.WindowStart, a.WindowEnd, detectStart, detectStart.Add(3*time.Hour))
	}
	if a.Status != shared.AlertStatusActive {
		t.Errorf("Status = %q, want active", a.Status)
	}
	if a.Location != "house-nick" || a.Source != "pressure" || a.RuleID != "pressure-drop-3h" {
		t.Errorf("identity fields = %q/%q/%q", a.Location, a.Source, a.RuleID)
	}
	if !strings.Contains(a.Message, "-6.0 mb/3h") {
		t.Errorf("Message = %q, want it to contain the 3h drop", a.Message)
	}
	// 06:00 UTC anchor is 1 AM in America/Chicago (CDT) on a Friday.
	if !strings.Contains(a.Message, "Fri 1 AM") {
		t.Errorf("Message = %q, want anchor formatted in America/Chicago", a.Message)
	}
	if a.ID == "" || a.ID != a.ComputeID() {
		t.Error("ID should be the computed hash")
	}
}

func TestDetect_SmallDropIgnored(t *testing.T) {
	points := hourlyPoints(1013, 1012.5, 1011.7, 1011, 1011, 1011, 1011)

	if alerts := detect(points); len(alerts) != 0 {
		t.Fatalf("len = %d, want 0 (2 mb drop is below threshold)", len(alerts))
	}
}

func TestDetect_TwoSeparateDropsProduceTwoAlerts(t *testing.T) {
	points := hourlyPoints(
		1013, 1011, 1009, 1007, // drop 1: -6 over hours 0–3
		1007, 1007, 1007, 1007, 1007, 1007, // flat gap
		1007, 1005, 1003, 1001, // drop 2: -6 over hours 10–13
		1001, 1001, 1001,
	)

	alerts := detect(points)

	if len(alerts) != 2 {
		t.Fatalf("len = %d, want 2", len(alerts))
	}
	if alerts[0].ID == alerts[1].ID {
		t.Error("separate episodes must have distinct IDs")
	}
	if !alerts[1].WindowStart.Equal(detectStart.Add(10 * time.Hour)) {
		t.Errorf("second episode start = %v, want hour 10", alerts[1].WindowStart)
	}
}

func TestDetect_RisingPressureIgnored(t *testing.T) {
	points := hourlyPoints(1007, 1009, 1011, 1013, 1015, 1017, 1019)

	if alerts := detect(points); len(alerts) != 0 {
		t.Fatalf("len = %d, want 0 (rising pressure)", len(alerts))
	}
}

func TestDetect_EmptyAndSinglePoint(t *testing.T) {
	if alerts := detect(nil); len(alerts) != 0 {
		t.Fatal("nil points should produce no alerts")
	}
	if alerts := detect(hourlyPoints(1013)); len(alerts) != 0 {
		t.Fatal("single point should produce no alerts")
	}
}

func TestDetect_SlowSpreadDropIgnored(t *testing.T) {
	// 7.8 mb down over 6 hours, but never ≥5 mb within any 3h window.
	points := hourlyPoints(1013, 1011.7, 1010.4, 1009.1, 1007.8, 1006.5, 1005.2, 1005.2, 1005.2)

	if alerts := detect(points); len(alerts) != 0 {
		t.Fatalf("len = %d, want 0 (no single window crosses threshold)", len(alerts))
	}
}

func TestDetect_ContinuousFallIsOneEpisode(t *testing.T) {
	// 2 mb/hour for 8 hours: every 3h window qualifies; must coalesce to ONE
	// alert spanning the whole fall rather than one alert per window start.
	points := hourlyPoints(1013, 1011, 1009, 1007, 1005, 1003, 1001, 999, 997, 997, 997)

	alerts := detect(points)

	if len(alerts) != 1 {
		t.Fatalf("len = %d, want 1 coalesced episode", len(alerts))
	}
	a := alerts[0]
	if !a.WindowStart.Equal(detectStart) || !a.WindowEnd.Equal(detectStart.Add(8*time.Hour)) {
		t.Errorf("episode window = %v–%v, want hours 0–8", a.WindowStart, a.WindowEnd)
	}
	if a.Value != -6 {
		t.Errorf("Value = %v, want steepest single-window delta -6", a.Value)
	}
	if !strings.Contains(a.Message, "-12.0/6h") {
		t.Errorf("Message = %q, want 6h continuation of -12.0", a.Message)
	}
}

func TestDetect_SevereThreshold(t *testing.T) {
	points := hourlyPoints(1015, 1011, 1007, 1004, 1004, 1004, 1004)

	alerts := detect(points)

	if len(alerts) != 1 {
		t.Fatalf("len = %d, want 1", len(alerts))
	}
	if alerts[0].Severity != shared.AlertSeveritySevere {
		t.Errorf("Severity = %q, want severe (11 mb ≥ 10 mb)", alerts[0].Severity)
	}
}

func TestDetect_ExtendedDeltaOmittedAtHorizonEnd(t *testing.T) {
	// Only 4 points: the 3h window exists but anchor+6h is past the horizon.
	points := hourlyPoints(1013, 1011, 1009, 1007)

	alerts := detect(points)

	if len(alerts) != 1 {
		t.Fatalf("len = %d, want 1", len(alerts))
	}
	if strings.Contains(alerts[0].Message, "/6h") {
		t.Errorf("Message = %q, want no 6h part when horizon doesn't cover it", alerts[0].Message)
	}
}

func TestDetect_ToleratesMissingHour(t *testing.T) {
	// Hour 3 missing: the only complete 3h window starts at hour 1.
	points := []repository.ForecastPoint{
		{ValidTime: detectStart, PressureMb: 1013},
		{ValidTime: detectStart.Add(1 * time.Hour), PressureMb: 1012},
		{ValidTime: detectStart.Add(2 * time.Hour), PressureMb: 1010},
		{ValidTime: detectStart.Add(4 * time.Hour), PressureMb: 1006},
	}

	alerts := detect(points)

	if len(alerts) != 1 {
		t.Fatalf("len = %d, want 1 (window anchored at the hour with full coverage)", len(alerts))
	}
	if !alerts[0].WindowStart.Equal(detectStart.Add(1 * time.Hour)) {
		t.Errorf("WindowStart = %v, want hour 1", alerts[0].WindowStart)
	}
	if alerts[0].Value != -6 {
		t.Errorf("Value = %v, want -6", alerts[0].Value)
	}
}
