package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/nickfang/personal-dashboard/clients/cli/internal/client"
)

// formatTimestamp parses an RFC3339 string and returns "MM.DD HH:MM:SS".
// On parse error, returns the raw input (possibly truncated).
func formatTimestamp(s string) string {
	if s == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		if len(s) > 19 {
			return s[:19]
		}
		return s
	}
	return t.Format("01.02 15:04:05")
}

// sectionBar renders a horizontal bar like "── WEATHER ────── 04.11 14:30:05 ──"
// with the title left-aligned and timestamp right-aligned, padded to innerWidth.
func sectionBar(title, timestamp string, innerWidth int) string {
	if innerWidth < 20 {
		innerWidth = 20
	}
	left := "── " + SectionTitleStyle.Render(title) + " "
	right := ""
	if timestamp != "" {
		right = " " + timestamp + " ──"
	} else {
		right = "──"
	}
	// Compute visible (unstyled) widths.
	leftVis := "── " + title + " "
	rightVis := right
	fillLen := innerWidth - len(leftVis) - len(rightVis)
	if fillLen < 1 {
		fillLen = 1
	}
	fill := strings.Repeat("─", fillLen)
	return SectionBarStyle.Render(left + fill + right)
}

// renderLocation renders a single location's full block (border + three sections).
func renderLocation(id string, w *client.Weather, p *client.Pressure, pol *client.Pollen, innerWidth int) string {
	if innerWidth < 30 {
		innerWidth = 30
	}

	var weatherTs, pressureTs, pollenTs string
	if w != nil {
		weatherTs = formatTimestamp(w.LastUpdated)
	}
	if p != nil {
		pressureTs = formatTimestamp(p.LastUpdated)
	}
	if pol != nil {
		pollenTs = formatTimestamp(pol.CollectedAt)
	}

	// Title bar across the top with location ID.
	titleBar := sectionBar(id, "", innerWidth)

	var b strings.Builder
	b.WriteString(titleBar)
	b.WriteString("\n")

	b.WriteString(sectionBar("WEATHER", weatherTs, innerWidth))
	b.WriteString("\n")
	b.WriteString(renderWeather(w))
	b.WriteString("\n")

	b.WriteString(sectionBar("PRESSURE", pressureTs, innerWidth))
	b.WriteString("\n")
	var currentMb float64
	haveCurrent := false
	if w != nil && w.PressureMb != 0 {
		currentMb = w.PressureMb
		haveCurrent = true
	}
	b.WriteString(renderPressureWithCurrent(p, currentMb, haveCurrent))
	b.WriteString("\n")

	b.WriteString(sectionBar("POLLEN", pollenTs, innerWidth))
	b.WriteString("\n")
	b.WriteString(renderPollen(pol))

	return BorderStyle.Width(innerWidth).Render(b.String())
}

// Ensure lipgloss import stays used even if border API changes upstream.
var _ = lipgloss.RoundedBorder
