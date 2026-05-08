package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nickfang/personal-dashboard/clients/cli/internal/client"
)

// statusFlashTTL is how long a transient status-bar flash message stays
// visible before being cleared on the next render.
const statusFlashTTL = 5 * time.Second

// viewMode describes what the TUI is currently rendering.
type viewMode int

const (
	viewAll viewMode = iota
	viewSingle
)

// refreshMsg is emitted on each tick — triggers a new full-dashboard fetch.
type refreshMsg time.Time

// fetchResultMsg carries the outcome of a full-dashboard fetch.
type fetchResultMsg struct {
	data *client.Response
	err  error
	at   time.Time
}

// fetchLocationResultMsg carries the outcome of a single-location fetch.
// The result is merged into the cached Response rather than replacing it.
type fetchLocationResultMsg struct {
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

	mode         viewMode
	selectedIdx  int
	initialLocID string // -location flag value; consumed on first successful fetch

	statusFlash   string
	statusFlashAt time.Time
}

// NewModel constructs a Model ready to run under tea.NewProgram. If
// initialLocID is non-empty, the model will switch to single-location view
// for that ID after the first successful fetch (or surface a flash message
// if the ID isn't present in the response).
func NewModel(c *client.Client, refresh time.Duration, initialLocID string) Model {
	return Model{client: c, refresh: refresh, initialLocID: initialLocID}
}

// Init returns the initial commands: an immediate fetch and a periodic tick.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.fetchCmd(), m.tickCmd())
}

// fetchCmd returns a tea.Cmd that performs a full-dashboard fetch in the
// background. Timeout is owned by the client (client.DefaultTimeout) so
// the tea.Cmd doesn't duplicate that policy here.
func (m Model) fetchCmd() tea.Cmd {
	return func() tea.Msg {
		data, err := m.client.Fetch(context.Background())
		return fetchResultMsg{data: data, err: err, at: time.Now()}
	}
}

// fetchLocationCmd returns a tea.Cmd that fetches just one location.
func (m Model) fetchLocationCmd(locationID string) tea.Cmd {
	return func() tea.Msg {
		data, err := m.client.FetchLocation(context.Background(), locationID)
		return fetchLocationResultMsg{data: data, err: err, at: time.Now()}
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
		return m.handleKey(msg)

	case refreshMsg:
		// Tick refresh always pulls the full dashboard so navigation stays
		// instant from cache regardless of which view we're in.
		return m, tea.Batch(m.fetchCmd(), m.tickCmd())

	case fetchResultMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.data = msg.data
		m.lastFetch = msg.at
		m.consumeInitialLocation()
		m.clampSelection()
		return m, nil

	case fetchLocationResultMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		if m.data == nil {
			m.data = msg.data
		} else {
			m.data.Merge(msg.data)
		}
		m.lastFetch = msg.at
		return m, nil
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit

	case "left", "right":
		ids := locationIDs(m.data)
		if len(ids) == 0 {
			return m, nil
		}
		if m.mode == viewAll {
			m.mode = viewSingle
			if msg.String() == "right" {
				m.selectedIdx = 0
			} else {
				m.selectedIdx = len(ids) - 1
			}
			return m, nil
		}
		if msg.String() == "right" {
			m.selectedIdx = (m.selectedIdx + 1) % len(ids)
		} else {
			m.selectedIdx = (m.selectedIdx - 1 + len(ids)) % len(ids)
		}
		return m, nil

	case "up":
		if m.mode == viewSingle {
			m.mode = viewAll
		}
		return m, nil

	case "r":
		if m.mode == viewAll {
			return m, m.fetchCmd()
		}
		ids := locationIDs(m.data)
		if len(ids) == 0 {
			return m, m.fetchCmd()
		}
		if m.selectedIdx >= len(ids) {
			m.selectedIdx = len(ids) - 1
		}
		return m, m.fetchLocationCmd(ids[m.selectedIdx])
	}
	return m, nil
}

// consumeInitialLocation runs after the first successful full-dashboard
// fetch. If the user supplied -location=ID and that ID is present in the
// response, switch to single-location view for it. Otherwise emit a
// non-fatal flash message and stay on "all".
func (m *Model) consumeInitialLocation() {
	if m.initialLocID == "" {
		return
	}
	ids := locationIDs(m.data)
	for i, id := range ids {
		if id == m.initialLocID {
			m.mode = viewSingle
			m.selectedIdx = i
			m.initialLocID = ""
			return
		}
	}
	m.statusFlash = fmt.Sprintf("location %q not found, showing all", m.initialLocID)
	m.statusFlashAt = time.Now()
	m.initialLocID = ""
}

// clampSelection keeps selectedIdx in range when the set of locations
// changes between fetches (e.g. a provider drops a location).
func (m *Model) clampSelection() {
	ids := locationIDs(m.data)
	if len(ids) == 0 {
		m.selectedIdx = 0
		return
	}
	if m.selectedIdx >= len(ids) {
		m.selectedIdx = len(ids) - 1
	}
	if m.selectedIdx < 0 {
		m.selectedIdx = 0
	}
}

// View renders the full TUI.
func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

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

	innerWidth := frameWidth - 4
	if innerWidth < 30 {
		innerWidth = 30
	}

	ids := locationIDs(m.data)
	if len(ids) == 0 && m.err == nil {
		b.WriteString("  Loading dashboard data...\n\n")
	}

	switch m.mode {
	case viewSingle:
		if len(ids) > 0 {
			idx := m.selectedIdx
			if idx >= len(ids) {
				idx = len(ids) - 1
			}
			id := ids[idx]
			b.WriteString(renderLocation(id, weatherFor(m.data, id), pressureFor(m.data, id), pollenFor(m.data, id), innerWidth))
			b.WriteString("\n\n")
		}
	default:
		for _, id := range ids {
			b.WriteString(renderLocation(id, weatherFor(m.data, id), pressureFor(m.data, id), pollenFor(m.data, id), innerWidth))
			b.WriteString("\n\n")
		}
	}

	flash := m.statusFlash
	if flash != "" && time.Since(m.statusFlashAt) > statusFlashTTL {
		flash = ""
	}
	focusID := ""
	idx, total := 0, len(ids)
	if m.mode == viewSingle && total > 0 {
		i := m.selectedIdx
		if i >= total {
			i = total - 1
		}
		focusID = ids[i]
		idx = i
	}
	b.WriteString(renderStatus(m.refresh, m.lastFetch, m.err, m.mode, focusID, idx, total, flash))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Top, b.String())
}

// weatherFor returns a pointer to the weather entry for id, or nil if absent.
func weatherFor(r *client.Response, id string) *client.Weather {
	if r == nil {
		return nil
	}
	if v, ok := r.Weather[id]; ok {
		return &v
	}
	return nil
}

// pressureFor returns a pointer to the pressure entry for id, or nil if absent.
func pressureFor(r *client.Response, id string) *client.Pressure {
	if r == nil {
		return nil
	}
	if v, ok := r.Pressure[id]; ok {
		return &v
	}
	return nil
}

// pollenFor returns a pointer to the pollen entry for id, or nil if absent.
func pollenFor(r *client.Response, id string) *client.Pollen {
	if r == nil {
		return nil
	}
	if v, ok := r.Pollen[id]; ok {
		return &v
	}
	return nil
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
