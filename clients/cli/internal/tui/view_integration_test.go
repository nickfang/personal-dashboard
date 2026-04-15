package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nickfang/personal-dashboard/clients/cli/internal/client"
)

func TestModelView_FullRenderWithStagingShapedData(t *testing.T) {
	resp := &client.Response{
		Weather: map[string]client.Weather{
			"house-nick":         {LocationID: "house-nick", LastUpdated: "2026-04-15T02:02:08Z", TempF: 75.4, TempFeelF: 78.4, HumidityPercent: 71, PrecipitationPercent: 0, PressureMb: 1013.58},
			"house-nita":         {LocationID: "house-nita", LastUpdated: "2026-04-15T02:02:08Z", TempF: 76.5, TempFeelF: 79.2, HumidityPercent: 68, PrecipitationPercent: 0, PressureMb: 1013.55},
			"distribution-hall":  {LocationID: "distribution-hall", LastUpdated: "2026-04-15T02:02:08Z", TempF: 76.5, TempFeelF: 79.3, HumidityPercent: 68, PrecipitationPercent: 0, PressureMb: 1013.55},
		},
		Pressure: map[string]client.Pressure{
			"house-nick":        {LocationID: "house-nick", LastUpdated: "2026-04-15T02:02:08Z", Delta1h: 0.82, Delta3h: 1.24, Delta6h: -0.76, Delta12h: -3.39, Delta24h: -2.42, Trend: "rising"},
			"house-nita":        {LocationID: "house-nita", LastUpdated: "2026-04-15T02:02:08Z", Delta1h: 0.82, Delta3h: 1.24, Delta6h: -0.77, Delta12h: -3.43, Delta24h: -2.42, Trend: "rising"},
			"distribution-hall": {LocationID: "distribution-hall", LastUpdated: "2026-04-15T02:02:08Z", Delta1h: 0.82, Delta3h: 1.24, Delta6h: -0.76, Delta12h: -3.42, Delta24h: -2.43, Trend: "rising"},
		},
		Pollen: map[string]client.Pollen{
			"house-nick": {LocationID: "house-nick", CollectedAt: "2026-04-14T19:02:22Z", OverallIndex: 2, OverallCategory: "Low", DominantType: "GRASS", Plants: []client.PollenPlant{
				{Code: "OAK", DisplayName: "Oak", Index: 2, Category: "Low", InSeason: true},
				{Code: "MAPLE", DisplayName: "Maple", Index: 1, Category: "Very Low"},
				{Code: "COTTONWOOD", DisplayName: "Cottonwood", Category: "None", InSeason: true},
			}},
		},
	}

	m := NewModel(nil, 60*time.Second)
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 110, Height: 60})
	tm, _ = tm.Update(fetchResultMsg{data: resp, at: time.Now()})

	view := tm.View()
	if view == "" {
		t.Fatal("View() returned empty string after data load")
	}

	mustContain := []string{
		"PERSONAL DASHBOARD",
		"house-nick", "house-nita", "distribution-hall",
		"75.4°F", "71%",
		"Rising",
		"Oak (In Season)",
		"Refreshing every",
		"q: quit",
	}
	for _, s := range mustContain {
		if !strings.Contains(view, s) {
			t.Errorf("view missing %q\n---\n%s\n---", s, view)
		}
	}

	if strings.Contains(view, "Cottonwood") {
		t.Errorf("Cottonwood should be filtered (index 0) but appeared in view")
	}
}
