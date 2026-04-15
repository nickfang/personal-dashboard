package tui

import (
	"fmt"
	"strings"

	"github.com/nickfang/personal-dashboard/services/kiosk/internal/client"
)

// trendArrow maps a trend string to a symbol + title-cased label.
func trendArrow(trend string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(trend)) {
	case "rising":
		return "▲", "Rising"
	case "falling":
		return "▼", "Falling"
	case "steady":
		return "→", "Steady"
	default:
		if trend == "" {
			return "·", "Unknown"
		}
		return "·", trend
	}
}

// renderPressure renders the pressure section body. The current mb reading
// comes from the weather payload (same collector), so renderLocation passes
// a pre-formatted current-mb string via a sibling function.
func renderPressure(p *client.Pressure) string {
	if p == nil {
		return "  (no pressure data)"
	}
	arrow, label := trendArrow(p.Trend)
	line1 := fmt.Sprintf("  %s %s", arrow, label)
	line2 := fmt.Sprintf("  Δ1h: %+.2f  Δ3h: %+.2f  Δ6h: %+.2f  Δ12h: %+.2f  Δ24h: %+.2f",
		p.Delta1h, p.Delta3h, p.Delta6h, p.Delta12h, p.Delta24h)
	return line1 + "\n" + line2
}

// renderPressureWithCurrent renders the pressure block with current mb from weather.
func renderPressureWithCurrent(p *client.Pressure, currentMb float64, haveCurrent bool) string {
	if p == nil {
		return "  (no pressure data)"
	}
	arrow, label := trendArrow(p.Trend)
	var line1 string
	if haveCurrent {
		line1 = fmt.Sprintf("  %.2f mb  %s %s", currentMb, arrow, label)
	} else {
		line1 = fmt.Sprintf("  %s %s", arrow, label)
	}
	line2 := fmt.Sprintf("  Δ1h: %+.2f  Δ3h: %+.2f  Δ6h: %+.2f  Δ12h: %+.2f  Δ24h: %+.2f",
		p.Delta1h, p.Delta3h, p.Delta6h, p.Delta12h, p.Delta24h)
	return line1 + "\n" + line2
}
