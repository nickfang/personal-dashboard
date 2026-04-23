package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nickfang/personal-dashboard/clients/cli/internal/client"
)

func TestRenderWeather(t *testing.T) {
	tests := []struct {
		name    string
		w       *client.Weather
		want    []string
		notWant []string
	}{
		{
			name: "full data",
			w: &client.Weather{
				TempF: 85.2, TempFeelF: 89.1, HumidityPercent: 62, PrecipitationPercent: 10,
			},
			want: []string{"85.2°F", "89.1°F", "62%", "10%", "Temp:", "Feels like:", "Humidity:", "Precipitation:"},
		},
		{
			name: "nil",
			w:    nil,
			want: []string{"(no weather data)"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := renderWeather(tc.w)
			if got == "" {
				t.Fatalf("empty output")
			}
			for _, s := range tc.want {
				if !strings.Contains(got, s) {
					t.Errorf("missing %q in output:\n%s", s, got)
				}
			}
		})
	}
}

func TestRenderPressure(t *testing.T) {
	p := &client.Pressure{
		Delta1h: 0.30, Delta3h: 0.80, Delta6h: 1.20, Delta12h: 2.00, Delta24h: 3.10,
		Trend: "rising",
	}
	got := renderPressureWithCurrent(p, 1013.25, true)
	for _, s := range []string{"1013.25 mb", "▲", "Rising", "+0.30", "+0.80", "+1.20", "+2.00", "+3.10"} {
		if !strings.Contains(got, s) {
			t.Errorf("missing %q:\n%s", s, got)
		}
	}

	if got := renderPressure(nil); !strings.Contains(got, "(no pressure data)") {
		t.Errorf("nil pressure fallback missing: %q", got)
	}

	// Falling and steady arrows.
	if got := renderPressure(&client.Pressure{Trend: "falling"}); !strings.Contains(got, "▼") {
		t.Errorf("expected ▼ for falling: %q", got)
	}
	if got := renderPressure(&client.Pressure{Trend: "steady"}); !strings.Contains(got, "→") {
		t.Errorf("expected → for steady: %q", got)
	}
}

func TestRenderPollen(t *testing.T) {
	tests := []struct {
		name    string
		p       *client.Pollen
		want    []string
		notWant []string
		ordered []string // substrings that must appear in this order
	}{
		{
			name: "title case categories",
			p: &client.Pollen{
				Plants: []client.PollenPlant{
					{DisplayName: "Juniper", Index: 4, Category: "High", InSeason: true},
					{DisplayName: "Elm", Index: 4, Category: "High", InSeason: true},
					{DisplayName: "Oak", Index: 3, Category: "Moderate", InSeason: true},
					{DisplayName: "Maple", Index: 2, Category: "Low", InSeason: false},
					{DisplayName: "Skipped", Index: 0, Category: "Low"},
				},
			},
			want:    []string{"High", "Juniper", "(In Season)", "Moderate", "Oak", "Low", "Maple", "(Out)"},
			notWant: []string{"Skipped"},
			ordered: []string{"High", "Moderate", "Low"},
		},
		{
			name: "realistic staging payload",
			p: &client.Pollen{
				OverallCategory: "Low",
				Plants: []client.PollenPlant{
					{Code: "MAPLE", DisplayName: "Maple", Index: 1, Category: "Very Low"},
					{Code: "OAK", DisplayName: "Oak", Index: 2, Category: "Low", InSeason: true},
					{Code: "COTTONWOOD", DisplayName: "Cottonwood", Category: "None", InSeason: true},
					{Code: "ELM", DisplayName: "Elm"},
				},
			},
			want:    []string{"Low", "Oak", "(In Season)", "Very Low", "Maple", "(Out)"},
			notWant: []string{"Cottonwood", "Elm", "None"},
			ordered: []string{"Low", "Very Low"},
		},
		{
			name: "very high ranks above high",
			p: &client.Pollen{
				Plants: []client.PollenPlant{
					{DisplayName: "Oak", Index: 4, Category: "High", InSeason: true},
					{DisplayName: "Juniper", Index: 5, Category: "Very High", InSeason: true},
				},
			},
			want:    []string{"Very High", "Juniper", "High", "Oak"},
			ordered: []string{"Very High", "High"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := renderPollen(tc.p)
			for _, s := range tc.want {
				if !strings.Contains(got, s) {
					t.Errorf("missing %q:\n%s", s, got)
				}
			}
			for _, s := range tc.notWant {
				if strings.Contains(got, s) {
					t.Errorf("should not contain %q:\n%s", s, got)
				}
			}
			last := -1
			for _, s := range tc.ordered {
				idx := strings.Index(got, s)
				if idx < 0 {
					t.Errorf("ordered token %q missing:\n%s", s, got)
					continue
				}
				if idx < last {
					t.Errorf("token %q appeared out of expected order:\n%s", s, got)
				}
				last = idx
			}
		})
	}

	if got := renderPollen(nil); !strings.Contains(got, "(no pollen data)") {
		t.Errorf("nil fallback missing: %q", got)
	}
	if got := renderPollen(&client.Pollen{}); !strings.Contains(got, "(no pollen data)") {
		t.Errorf("empty pollen fallback missing: %q", got)
	}
}

func TestRenderStatus(t *testing.T) {
	ts := time.Date(2026, 4, 11, 14, 30, 5, 0, time.UTC)
	got := renderStatus(60*time.Second, ts, nil, 80)
	for _, s := range []string{"1m0s", "14:30:05", "q: quit", "Refreshing"} {
		if !strings.Contains(got, s) {
			t.Errorf("missing %q:\n%s", s, got)
		}
	}
	got = renderStatus(60*time.Second, ts, errors.New("boom"), 80)
	if !strings.Contains(got, "Error:") || !strings.Contains(got, "boom") {
		t.Errorf("expected error status, got: %q", got)
	}
}

func TestRenderHeader(t *testing.T) {
	got := renderHeader(80)
	if !strings.Contains(got, "PERSONAL DASHBOARD") {
		t.Errorf("missing title: %q", got)
	}
}

func TestRenderLocation(t *testing.T) {
	weatherTs := parseLocal(t, "2026-04-11T14:30:05Z").Format("01.02 15:04:05")
	pollenTs := parseLocal(t, "2026-04-11T06:00:00Z").Format("01.02 15:04:05")

	got := renderLocation("house-nick",
		&client.Weather{TempF: 85.2, TempFeelF: 89.1, HumidityPercent: 62, PrecipitationPercent: 10, LastUpdated: "2026-04-11T14:30:05Z", PressureMb: 1013.25},
		&client.Pressure{Trend: "rising", Delta1h: 0.3, LastUpdated: "2026-04-11T14:30:05Z"},
		&client.Pollen{Plants: []client.PollenPlant{{DisplayName: "Oak", Index: 3, Category: "Moderate", InSeason: true}}, CollectedAt: "2026-04-11T06:00:00Z"},
		70,
	)
	for _, s := range []string{"house-nick", "WEATHER", "PRESSURE", "POLLEN", "85.2°F", "Oak", weatherTs, pollenTs} {
		if !strings.Contains(got, s) {
			t.Errorf("missing %q in location render:\n%s", s, got)
		}
	}
}

func TestFormatTimestamp(t *testing.T) {
	const input = "2026-04-11T14:30:05Z"
	parsed, err := time.Parse(time.RFC3339, input)
	if err != nil {
		t.Fatal(err)
	}
	want := parsed.Local().Format("01.02 15:04:05")

	if got := formatTimestamp(input); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := formatTimestamp(""); got != "" {
		t.Errorf("empty should return empty, got %q", got)
	}
	if got := formatTimestamp("not-a-time"); got == "" {
		t.Errorf("bad parse should return raw, got empty")
	}
}

func TestModelUpdateQuit(t *testing.T) {
	m := NewModel(nil, time.Minute)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected Quit cmd on q")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestModelUpdateWindowSize(t *testing.T) {
	m := NewModel(nil, time.Minute)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m2 := updated.(Model)
	if m2.width != 120 || m2.height != 40 {
		t.Errorf("dims not stored: %d x %d", m2.width, m2.height)
	}
}

func TestModelViewEmptyWhenNoSize(t *testing.T) {
	m := NewModel(nil, time.Minute)
	if got := m.View(); got != "" {
		t.Errorf("expected empty view pre-WindowSizeMsg, got %q", got)
	}
}

func TestModelFetchResult(t *testing.T) {
	m := NewModel(nil, time.Minute)
	now := time.Now()
	updated, _ := m.Update(fetchResultMsg{data: &client.Response{}, err: nil, at: now})
	m2 := updated.(Model)
	if m2.data == nil {
		t.Error("expected data set")
	}
	if !m2.lastFetch.Equal(now) {
		t.Errorf("lastFetch not updated")
	}
	updated, _ = m2.Update(fetchResultMsg{err: errors.New("x"), at: now})
	m3 := updated.(Model)
	if m3.err == nil {
		t.Error("expected err set")
	}
}

// parseLocal parses an RFC3339 timestamp and returns it in the host's local
// timezone. Used to compute expected render values that match whatever zone
// the test is running in.
func parseLocal(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return parsed.Local()
}
