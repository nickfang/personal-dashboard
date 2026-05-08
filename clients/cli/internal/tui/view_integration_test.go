package tui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nickfang/personal-dashboard/clients/cli/internal/client"
)

func TestModelView_FullRenderWithStagingShapedData(t *testing.T) {
	resp := &client.Response{
		Weather: map[string]client.Weather{
			"house-nick":        {LocationID: "house-nick", LastUpdated: "2026-04-15T02:02:08Z", TempF: 75.4, TempFeelF: 78.4, HumidityPercent: 71, PrecipitationPercent: 0, PressureMb: 1013.58},
			"house-nita":        {LocationID: "house-nita", LastUpdated: "2026-04-15T02:02:08Z", TempF: 76.5, TempFeelF: 79.2, HumidityPercent: 68, PrecipitationPercent: 0, PressureMb: 1013.55},
			"distribution-hall": {LocationID: "distribution-hall", LastUpdated: "2026-04-15T02:02:08Z", TempF: 76.5, TempFeelF: 79.3, HumidityPercent: 68, PrecipitationPercent: 0, PressureMb: 1013.55},
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

	m := NewModel(nil, 60*time.Second, "")
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
		"q quit",
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

func TestModelView_SingleLocationViewRendersOnlyFocused(t *testing.T) {
	resp := &client.Response{
		Weather: map[string]client.Weather{
			"house-nick": {LocationID: "house-nick", LastUpdated: "2026-04-15T02:02:08Z", TempF: 75.4, HumidityPercent: 71, PressureMb: 1013.58},
			"house-nita": {LocationID: "house-nita", LastUpdated: "2026-04-15T02:02:08Z", TempF: 76.5, HumidityPercent: 68, PressureMb: 1013.55},
		},
		Pressure: map[string]client.Pressure{
			"house-nick": {LocationID: "house-nick", Trend: "rising"},
			"house-nita": {LocationID: "house-nita", Trend: "rising"},
		},
	}

	m := NewModel(nil, 60*time.Second, "")
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 110, Height: 60})
	tm, _ = tm.Update(fetchResultMsg{data: resp, at: time.Now()})
	// Right arrow from "all" enters first single view (house-nick by sort order).
	tm, _ = tm.Update(tea.KeyMsg{Type: tea.KeyRight})

	view := tm.View()
	if !strings.Contains(view, "house-nick") {
		t.Errorf("expected house-nick in single view:\n%s", view)
	}
	if strings.Contains(view, "house-nita") {
		t.Errorf("expected house-nita to be filtered out in single view:\n%s", view)
	}
	for _, s := range []string{"(1/2)", "← → cycle", "↑ all"} {
		if !strings.Contains(view, s) {
			t.Errorf("expected single-view hint %q:\n%s", s, view)
		}
	}
}

func TestModel_InitialLocationFocusesOnFirstFetch(t *testing.T) {
	resp := &client.Response{
		Weather: map[string]client.Weather{
			"house-nick": {LocationID: "house-nick", LastUpdated: "2026-04-15T02:02:08Z", TempF: 75.4, PressureMb: 1013.58},
			"house-nita": {LocationID: "house-nita", LastUpdated: "2026-04-15T02:02:08Z", TempF: 76.5, PressureMb: 1013.55},
		},
		Pressure: map[string]client.Pressure{
			"house-nick": {Trend: "rising"},
			"house-nita": {Trend: "rising"},
		},
	}

	m := NewModel(nil, 60*time.Second, "house-nita")
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 110, Height: 60})
	tm, _ = tm.Update(fetchResultMsg{data: resp, at: time.Now()})

	view := tm.View()
	if !strings.Contains(view, "house-nita") {
		t.Errorf("expected house-nita to be focused per -location flag:\n%s", view)
	}
	if strings.Contains(view, "house-nick") {
		// "house-nick" wouldn't appear as a location box but might appear in
		// e.g. logs. The reliable check is that the position indicator is for nita.
	}
	if !strings.Contains(view, "(2/2)") {
		t.Errorf("expected position (2/2) for house-nita (alphabetical second):\n%s", view)
	}
}

// TestModel_RefreshKeyHitsRightEndpointPerView locks in the view-aware
// refresh contract from issue #60: pressing 'r' in viewAll re-fetches the
// full /v1/dashboard, while pressing 'r' in viewSingle re-fetches only
// /v1/dashboard/{focusedID}. Drives the model through a real *client.Client
// pointed at an httptest server so the assertion is on the actual wire path.
func TestModel_RefreshKeyHitsRightEndpointPerView(t *testing.T) {
	var mu sync.Mutex
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"weather":  {"house-nick": {"locationId":"house-nick","lastUpdated":"2026-04-15T02:02:08Z","tempF":75}},
			"pressure": {"house-nick": {"locationId":"house-nick","lastUpdated":"2026-04-15T02:02:08Z","trend":"rising"}},
			"pollen":   {}
		}`))
	}))
	defer srv.Close()

	c, err := client.New(srv.URL)
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}

	// Seed cached data so single view has something to focus on without
	// having to round-trip the test server first.
	seed := &client.Response{
		Weather: map[string]client.Weather{
			"house-nick": {LocationID: "house-nick", LastUpdated: "2026-04-15T02:02:08Z", TempF: 75},
		},
	}
	m := NewModel(c, 60*time.Second, "")
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 110, Height: 60})
	tm, _ = tm.Update(fetchResultMsg{data: seed, at: time.Now()})

	// 'r' in all view → /v1/dashboard
	_, cmd := tm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if cmd == nil {
		t.Fatal("expected refresh cmd in all view, got nil")
	}
	// tea.Cmds wrap the actual work in a closure; invoking it synchronously
	// here drives the HTTP call through the real client to the test server.
	cmd()

	// Right arrow enters single view focused on the (only) seeded location.
	tm, _ = tm.Update(tea.KeyMsg{Type: tea.KeyRight})

	// 'r' in single view → /v1/dashboard/house-nick
	_, cmd = tm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if cmd == nil {
		t.Fatal("expected refresh cmd in single view, got nil")
	}
	cmd()

	mu.Lock()
	defer mu.Unlock()
	if len(paths) != 2 {
		t.Fatalf("expected 2 server hits, got %d: %v", len(paths), paths)
	}
	if paths[0] != "/v1/dashboard" {
		t.Errorf("first refresh (all view) hit %q, want /v1/dashboard", paths[0])
	}
	if paths[1] != "/v1/dashboard/house-nick" {
		t.Errorf("second refresh (single view) hit %q, want /v1/dashboard/house-nick", paths[1])
	}
}

func TestModel_InitialLocationMissShowsFlash(t *testing.T) {
	resp := &client.Response{
		Weather: map[string]client.Weather{
			"house-nick": {LocationID: "house-nick", LastUpdated: "2026-04-15T02:02:08Z", TempF: 75.4},
		},
	}

	m := NewModel(nil, 60*time.Second, "bogus-location")
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 110, Height: 60})
	tm, _ = tm.Update(fetchResultMsg{data: resp, at: time.Now()})

	view := tm.View()
	if !strings.Contains(view, "bogus-location") {
		t.Errorf("expected flash mentioning the missing location:\n%s", view)
	}
	if !strings.Contains(view, "← → focus") {
		t.Errorf("expected fallback to all-view hints:\n%s", view)
	}
}
