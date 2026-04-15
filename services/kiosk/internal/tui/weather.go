package tui

import (
	"fmt"
	"strings"

	"github.com/nickfang/personal-dashboard/services/kiosk/internal/client"
)

// renderWeather renders the weather section body (no border).
func renderWeather(w *client.Weather) string {
	if w == nil {
		return "  (no weather data)"
	}
	rows := []struct {
		label string
		value string
	}{
		{"Temp:", fmt.Sprintf("%.1f°F", w.TempF)},
		{"Feels like:", fmt.Sprintf("%.1f°F", w.TempFeelF)},
		{"Humidity:", fmt.Sprintf("%d%%", w.HumidityPercent)},
		{"Precipitation:", fmt.Sprintf("%d%%", w.PrecipitationPercent)},
	}
	var b strings.Builder
	for i, r := range rows {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("  ")
		b.WriteString(LabelStyle.Render(r.label))
		b.WriteString(ValueStyle.Render(r.value))
	}
	return b.String()
}
