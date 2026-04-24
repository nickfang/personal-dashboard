package tui

import (
	"context"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nickfang/personal-dashboard/clients/cli/internal/client"
)

// refreshMsg is emitted on each tick — triggers a new fetch.
type refreshMsg time.Time

// fetchResultMsg carries the outcome of a dashboard fetch.
type fetchResultMsg struct {
	data *client.Response
	err  error
	at   time.Time
}

// Model is the bubbletea model for the kiosk TUI.
type Model struct {
	client    *client.Client
	refresh   time.Duration
	width     int
	height    int
	data      *client.Response
	err       error
	lastFetch time.Time
}

// NewModel constructs a Model ready to run under tea.NewProgram.
func NewModel(c *client.Client, refresh time.Duration) Model {
	return Model{client: c, refresh: refresh}
}

// Init returns the initial commands: an immediate fetch and a periodic tick.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.fetchCmd(), m.tickCmd())
}

// fetchCmd returns a tea.Cmd that performs a fetch (with a 10s timeout) in the background.
func (m Model) fetchCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		data, err := m.client.Fetch(ctx)
		return fetchResultMsg{data: data, err: err, at: time.Now()}
	}
}

// tickCmd waits the refresh interval, then emits a refreshMsg.
func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(m.refresh, func(t time.Time) tea.Msg {
		return refreshMsg(t)
	})
}

// Update processes incoming messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
		return m, nil

	case refreshMsg:
		// Kick off a fetch and schedule the next tick.
		return m, tea.Batch(m.fetchCmd(), m.tickCmd())

	case fetchResultMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.err = nil
			m.data = msg.data
			m.lastFetch = msg.at
		}
		return m, nil
	}
	return m, nil
}

// View renders the full TUI.
func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	// Compute frame width: cap to something sane, leave margin.
	frameWidth := m.width - 4
	if frameWidth < 40 {
		frameWidth = m.width
	}
	if frameWidth > 100 {
		frameWidth = 100
	}

	var b strings.Builder
	b.WriteString(renderHeader(frameWidth))
	b.WriteString("\n\n")

	// Inner width inside each location box: frameWidth minus the box's own border padding.
	innerWidth := frameWidth - 4
	if innerWidth < 30 {
		innerWidth = 30
	}

	ids := locationIDs(m.data)
	if len(ids) == 0 && m.err == nil {
		b.WriteString("  Loading dashboard data...\n\n")
	}

	for _, id := range ids {
		var w *client.Weather
		var p *client.Pressure
		var pol *client.Pollen
		if v, ok := m.data.Weather[id]; ok {
			w = &v
		}
		if v, ok := m.data.Pressure[id]; ok {
			p = &v
		}
		if v, ok := m.data.Pollen[id]; ok {
			pol = &v
		}
		b.WriteString(renderLocation(id, w, p, pol, innerWidth))
		b.WriteString("\n\n")
	}

	b.WriteString(renderStatus(m.refresh, m.lastFetch, m.err))

	// Center horizontally.
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Top, b.String())
}

// locationIDs returns the union of location IDs present in the response,
// sorted alphabetically for deterministic render order. Returns nil if no data.
func locationIDs(r *client.Response) []string {
	if r == nil {
		return nil
	}
	seen := make(map[string]struct{})
	for id := range r.Weather {
		seen[id] = struct{}{}
	}
	for id := range r.Pressure {
		seen[id] = struct{}{}
	}
	for id := range r.Pollen {
		seen[id] = struct{}{}
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
