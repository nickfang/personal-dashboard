package tui

import (
	"sort"
	"strings"

	"github.com/nickfang/personal-dashboard/services/kiosk/internal/client"
)

// categoryRank maps a pollen category (Title Case, as returned by the API) to a
// descending sort rank (higher = more severe). Unknown/empty categories sort last.
func categoryRank(cat string) int {
	switch strings.TrimSpace(cat) {
	case "Very High":
		return 6
	case "High":
		return 5
	case "Moderate":
		return 4
	case "Low":
		return 3
	case "Very Low":
		return 2
	case "None":
		return 1
	default:
		return 0
	}
}

// renderPollen renders the pollen section body.
func renderPollen(p *client.Pollen) string {
	if p == nil {
		return "  (no pollen data)"
	}
	// Filter plants with index >= 1 and group by category.
	groups := map[string][]client.PollenPlant{}
	for _, pl := range p.Plants {
		if pl.Index < 1 {
			continue
		}
		groups[pl.Category] = append(groups[pl.Category], pl)
	}
	if len(groups) == 0 {
		return "  (no pollen data)"
	}

	// Sort categories by rank, descending.
	cats := make([]string, 0, len(groups))
	for k := range groups {
		cats = append(cats, k)
	}
	sort.Slice(cats, func(i, j int) bool {
		return categoryRank(cats[i]) > categoryRank(cats[j])
	})

	var b strings.Builder
	for i, cat := range cats {
		if i > 0 {
			b.WriteString("\n")
		}
		plants := groups[cat]
		// Sort plants within category by index descending for stability.
		sort.SliceStable(plants, func(a, b int) bool {
			return plants[a].Index > plants[b].Index
		})
		parts := make([]string, 0, len(plants))
		for _, pl := range plants {
			name := pl.DisplayName
			if name == "" {
				name = pl.Code
			}
			season := "Out"
			if pl.InSeason {
				season = "In Season"
			}
			parts = append(parts, name+" ("+season+")")
		}
		// Pad category label to a stable column width.
		catLabel := cat
		pad := 10
		if len(catLabel) < pad {
			catLabel = catLabel + strings.Repeat(" ", pad-len(catLabel))
		}
		b.WriteString("  ")
		b.WriteString(catLabel)
		b.WriteString(" ")
		b.WriteString(strings.Join(parts, "  "))
	}
	return b.String()
}
