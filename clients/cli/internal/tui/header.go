package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// renderHeader renders the top "PERSONAL DASHBOARD" bar across the full width.
func renderHeader(width int) string {
	if width <= 0 {
		width = 40
	}
	title := "PERSONAL DASHBOARD"
	return BorderStyle.
		Width(width - 2). // account for border chars
		Align(lipgloss.Center).
		Render(TitleStyle.Render(title))
}
