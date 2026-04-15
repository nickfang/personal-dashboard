package tui

import "github.com/charmbracelet/lipgloss"

// LabelColWidth is the right-padded width of label columns in the weather section.
const LabelColWidth = 15

var (
	// dimFg is a neutral foreground that works on both light and dark backgrounds.
	dimFg = lipgloss.AdaptiveColor{Light: "240", Dark: "245"}
	// accentFg is used for titles/headers.
	accentFg = lipgloss.AdaptiveColor{Light: "22", Dark: "42"}
	// errorFg is for errors.
	errorFg = lipgloss.AdaptiveColor{Light: "124", Dark: "203"}

	// BorderStyle is a rounded box-drawing border with a dim foreground.
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(dimFg)

	// TitleStyle is bold, used for the dashboard title bar.
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentFg).
			Align(lipgloss.Center)

	// SectionTitleStyle is bold for section header bars (WEATHER/PRESSURE/POLLEN).
	SectionTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(accentFg)

	// LabelStyle right-aligns weather labels.
	LabelStyle = lipgloss.NewStyle().
			Width(LabelColWidth)

	// ValueStyle is used for metric values.
	ValueStyle = lipgloss.NewStyle().Bold(true)

	// ErrorStyle is used for error messages.
	ErrorStyle = lipgloss.NewStyle().Foreground(errorFg).Bold(true)

	// StatusBarStyle is used for the bottom status bar.
	StatusBarStyle = lipgloss.NewStyle().Foreground(dimFg)

	// SectionBarStyle is dim, used for the section separator bars inside a location.
	SectionBarStyle = lipgloss.NewStyle().Foreground(dimFg)
)
