package tui

import (
	"fmt"
	"time"
)

// renderStatus renders the bottom status bar.
func renderStatus(refresh time.Duration, lastFetch time.Time, err error) string {
	if err != nil {
		return ErrorStyle.Render(fmt.Sprintf("Error: %s │ q: quit", err.Error()))
	}
	last := "never"
	if !lastFetch.IsZero() {
		last = lastFetch.Format("15:04:05")
	}
	return StatusBarStyle.Render(fmt.Sprintf(
		"Refreshing every %s │ Last fetch: %s │ q: quit",
		refresh.String(), last,
	))
}
