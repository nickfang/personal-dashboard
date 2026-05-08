package tui

import (
	"fmt"
	"strings"
	"time"
)

// renderStatus renders the bottom status bar. The hints adapt to the current
// view: in viewAll the hints suggest entering single-view; in viewSingle they
// show the focused location, position indicator, and how to escape.
//
// err and flash, when present, render as their own lines stacked above the
// contextual bar. The contextual bar (with nav hints) is always rendered, so
// the user can always see how to navigate even while an error is on screen.
func renderStatus(refresh time.Duration, lastFetch time.Time, err error,
	mode viewMode, focusID string, idx, total int, flash string) string {
	// "Last fetch" reflects the wall-clock time of the most recent
	// *successful* fetch, not the most recent attempt. Failed fetches set
	// m.err but leave lastFetch untouched (see model.go's fetchResultMsg
	// and fetchLocationResultMsg handlers), so this timestamp is what the
	// user can trust the rendered data to be from.
	last := "never"
	if !lastFetch.IsZero() {
		last = lastFetch.Format("15:04:05")
	}

	var hint string
	switch mode {
	case viewSingle:
		if total > 0 && focusID != "" {
			hint = fmt.Sprintf("%s (%d/%d)  ← → cycle  ↑ all  r refresh  q quit",
				focusID, idx+1, total)
		} else {
			hint = "← → cycle  ↑ all  r refresh  q quit"
		}
	default:
		hint = "← → focus  r refresh  q quit"
	}

	bar := StatusBarStyle.Render(fmt.Sprintf(
		"Refreshing every %s │ Last fetch: %s │ %s",
		refresh.String(), last, hint,
	))

	var lines []string
	if err != nil {
		lines = append(lines, ErrorStyle.Render(fmt.Sprintf("Error: %s", err.Error())))
	}
	if flash != "" {
		lines = append(lines, StatusBarStyle.Render(flash))
	}
	lines = append(lines, bar)
	return strings.Join(lines, "\n")
}
