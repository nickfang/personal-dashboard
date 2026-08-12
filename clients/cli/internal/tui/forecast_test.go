package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/nickfang/personal-dashboard/clients/cli/internal/client"
)

// testForecast builds a forecast with hourly points from the given pressures,
// starting at a fixed base time.
func testForecast(pressures ...float64) *client.Forecast {
	base := time.Date(2026, 6, 12, 6, 0, 0, 0, time.UTC)
	points := make([]client.ForecastPoint, len(pressures))
	for i, p := range pressures {
		points[i] = client.ForecastPoint{
			ValidTime:  base.Add(time.Duration(i) * time.Hour).Format(time.RFC3339),
			PressureMb: p,
		}
	}
	return &client.Forecast{
		LocationID: "house-nick",
		IssuedAt:   base.Format(time.RFC3339),
		Points:     points,
	}
}

func TestRenderForecast_FallingTrendLowAndDeltas(t *testing.T) {
	// 49 hourly points: down 2 mb/h for 4h, then flat at 1005.
	pressures := make([]float64, 49)
	p := 1013.0
	for i := range pressures {
		if i >= 1 && i <= 4 {
			p -= 2
		}
		pressures[i] = p
	}

	got := renderForecast(testForecast(pressures...))

	for _, s := range []string{
		"▼ Falling",
		"low 1005 mb (48h)",
		"Δ+3h", "Δ+6h", "Δ+12h", "Δ+24h", "Δ+48h",
		"-6.00", "-8.00",
	} {
		if !strings.Contains(got, s) {
			t.Errorf("missing %q in forecast render:\n%s", s, got)
		}
	}
}

func TestRenderForecast_RisingTrend(t *testing.T) {
	got := renderForecast(testForecast(1007, 1009, 1011, 1013, 1013))
	if !strings.Contains(got, "▲ Rising") {
		t.Errorf("expected rising arrow, got:\n%s", got)
	}
}

func TestRenderForecast_MissingHorizonsRenderPlaceholder(t *testing.T) {
	// Only 7 hours of data: Δ+12h/Δ+24h/Δ+48h are uncovered.
	got := renderForecast(testForecast(1013, 1012, 1011, 1010, 1009, 1008, 1007))
	if !strings.Contains(got, "--") {
		t.Errorf("expected -- placeholders for uncovered horizons, got:\n%s", got)
	}
	if !strings.Contains(got, "-3.00") {
		t.Errorf("expected covered Δ+3h value, got:\n%s", got)
	}
}

func TestRenderForecast_NilAndEmpty(t *testing.T) {
	if got := renderForecast(nil); got != "  (no forecast data)" {
		t.Errorf("nil forecast = %q", got)
	}
	if got := renderForecast(&client.Forecast{}); got != "  (no forecast data)" {
		t.Errorf("empty forecast = %q", got)
	}
}

func TestAlertBanner_ActiveOnly(t *testing.T) {
	alerts := []client.Alert{
		{Message: "Thu 2 PM  -6.2 mb/3h  -8.1/6h", Severity: "warning", Status: "active"},
		{Message: "old drop", Severity: "warning", Status: "resolved"},
		{Message: "unknown status", Severity: "warning", Status: "something-else"},
	}

	got := alertBanner(alerts)

	if !strings.Contains(got, "⚠ Thu 2 PM  -6.2 mb/3h  -8.1/6h") {
		t.Errorf("active alert missing from banner: %q", got)
	}
	if strings.Contains(got, "old drop") || strings.Contains(got, "unknown status") {
		t.Errorf("non-active alerts must not render: %q", got)
	}
}

func TestAlertBanner_EmptyWhenNoActive(t *testing.T) {
	if got := alertBanner(nil); got != "" {
		t.Errorf("nil alerts should render nothing, got %q", got)
	}
	if got := alertBanner([]client.Alert{{Message: "x", Status: "resolved"}}); got != "" {
		t.Errorf("resolved-only alerts should render nothing, got %q", got)
	}
}

func TestAlertBanner_SevereUsesErrorStyle(t *testing.T) {
	warning := alertBanner([]client.Alert{{Message: "drop", Severity: "warning", Status: "active"}})
	severe := alertBanner([]client.Alert{{Message: "drop", Severity: "severe", Status: "active"}})
	// Both render the same text; the styling differs. We can't assert on
	// ANSI codes (lipgloss strips them in tests without a TTY), but both
	// must at least carry the banner glyph.
	for name, got := range map[string]string{"warning": warning, "severe": severe} {
		if !strings.Contains(got, "⚠ drop") {
			t.Errorf("%s banner missing content: %q", name, got)
		}
	}
}
