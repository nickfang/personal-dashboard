# CLI Kiosk Dashboard — Design Document

## Overview

A terminal-based kiosk application that displays weather, pressure, and pollen data using ASCII art with box-drawing characters. It runs continuously in a terminal, auto-refreshes on an interval, and renders a retro-terminal-styled dashboard.

**Phase 1** is a read-only auto-refreshing display.
**Phase 2** adds interactivity: selectable locations, view switching (single location vs. cross-location comparison for a specific data type like pollen-only or pressure-only).

## Architecture

```
┌─────────────┐       HTTP GET        ┌─────────────────┐
│   CLI Kiosk │ ────────────────────> │  Dashboard API   │
│  (bubbletea)│ <──────────────────── │  (port 8080)     │
│             │     JSON response      │                  │
└─────────────┘                        └──────┬──────┬───┘
                                         gRPC │      │ gRPC
                                              v      v
                                     Weather     Pollen
                                     Provider    Provider
                                    (50051)      (50052)
```

The kiosk is a pure display client. It calls `GET /v1/dashboard` on the dashboard-api, which already aggregates weather, pressure, and pollen data from the gRPC providers in parallel. The kiosk parses the JSON response and renders it.

### Why HTTP from dashboard-api (not direct gRPC)

- No proto codegen needed in the kiosk module — lean `go.mod`
- No gRPC auth/TLS complexity
- Dashboard-api already does parallel aggregation via `errgroup`
- Decoupled from internal service topology — if a third provider is added, only dashboard-api changes
- The kiosk works identically pointing at localhost or a Cloud Run URL

## Technology

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Language | Go | Fits existing monorepo (go.work, Makefile patterns, shared module) |
| TUI framework | [Bubbletea](https://github.com/charmbracelet/bubbletea) | Elm architecture (Model/Update/View), first-class terminal resize via `WindowSizeMsg`, periodic refresh via `tea.Tick`, future interactivity support |
| Styling | [Lipgloss](https://github.com/charmbracelet/lipgloss) | Declarative styles, box-drawing borders, centering via `lipgloss.Place()`, color support |
| HTTP client | `net/http` (stdlib) | No additional dependency needed |
| JSON parsing | `encoding/json` (stdlib) | Dashboard-api returns protojson (camelCase, RFC 3339 timestamps) |

### Why Bubbletea over alternatives

- **vs. Bash script**: No state management, no resize handling, brittle layout code, `jq` dependency for JSON parsing
- **vs. Pure Go (no library)**: Manual ANSI codes for every layout operation; reimplements what Lipgloss provides for free
- **vs. tcell/tview**: More imperative, less flexible styling, single maintainer
- **vs. Python rich/textual**: Architectural outlier in a Go monorepo
- **vs. Rust ratatui**: Wrong language for this monorepo

## Data Source

`GET /v1/dashboard` returns:

```json
{
  "weather": {
    "<location-id>": {
      "locationId": "string",
      "lastUpdated": "RFC 3339 timestamp",
      "tempC": 0.0, "tempF": 0.0,
      "tempFeelC": 0.0, "tempFeelF": 0.0,
      "humidityPercent": 0,
      "precipitationPercent": 0,
      "pressureMb": 0.0
    }
  },
  "pressure": {
    "<location-id>": {
      "locationId": "string",
      "lastUpdated": "RFC 3339 timestamp",
      "delta1h": 0.0, "delta3h": 0.0,
      "delta6h": 0.0, "delta12h": 0.0, "delta24h": 0.0,
      "trend": "rising|falling|steady"
    }
  },
  "pollen": {
    "<location-id>": {
      "locationId": "string",
      "collectedAt": "RFC 3339 timestamp",
      "overallIndex": 0,
      "overallCategory": "string",
      "dominantType": "TREE|GRASS|WEED",
      "types": [{ "code": "", "index": 0, "category": "", "inSeason": false }],
      "plants": [{ "code": "", "displayName": "", "index": 0, "category": "", "inSeason": false }]
    }
  }
}
```

**Locations:** `house-nick`, `house-nita`, `distribution-hall`

## Layout (Phase 1 — Read-Only)

The dashboard displays all three locations, each showing weather, pressure, and pollen sections. The layout centers within the terminal width and adapts on resize.

```
┌──────────────────────────────────────────────────────────────────┐
│                       PERSONAL DASHBOARD                         │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─ house-nick ────────────────────────────────────────────────┐ │
│  ├─ WEATHER ──────────────────────────────── 04.11 14:30:05 ──┤ │
│  │  Temp:          85.2°F                                     │ │
│  │  Feels like:    89.1°F                                     │ │
│  │  Humidity:      62%                                        │ │
│  │  Precipitation: 10%                                        │ │
│  ├─ PRESSURE ─────────────────────────────── 04.11 14:30:05 ──┤ │
│  │  1013.25 mb  ▲ Rising                                      │ │
│  │  Δ1h: +0.30  Δ3h: +0.80  Δ6h: +1.20  Δ12h: +2.00  Δ24h: +3.10 │
│  ├─ POLLEN ────────────────── Overall: 4 ─── 04.11 06:00:00 ──┤ │
│  │  High       Juniper (In Season)  Elm (In Season)            │ │
│  │  Moderate   Oak (In Season)                                 │ │
│  │  Low        Maple (Out)                                     │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                  │
│  ┌─ house-nita ────────────────────────────────────────────────┐ │
│  │  ...                                                        │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                  │
│  ┌─ distribution-hall ─────────────────────────────────────────┐ │
│  │  ...                                                        │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                  │
│  Refreshing every 60s │ Last fetch: 14:30:05 │ q: quit          │
└──────────────────────────────────────────────────────────────────┘
```

### Layout rules

- Outer frame centers horizontally in the terminal using `lipgloss.Place()`
- Each location is a bordered box with the location ID in the top border
- All three sections (Weather, Pressure, Pollen) use the same format: labeled header in the border with the section's collected-at timestamp on the right
- Weather and Pressure `lastUpdated` come from the same collector but are stored independently; Pollen `collectedAt` is from a separate collector on a different schedule (2x daily)
- Weather section: one metric per line (temp °F, feels-like, humidity %, precipitation %) with aligned label column
- Pressure line: current mb, trend arrow (▲/▼/→), deltas
- Pollen section: plants grouped by category (descending index), only index >= 1 shown, with in-season status
- Status bar at bottom: refresh interval, last fetch time (when kiosk called the API), quit key
- On narrow terminals (< 60 cols), sections may stack vertically instead of fitting on one line

## View Modes (Phase 2 — Interactive, future)

These are not in scope for Phase 1 but inform the architecture:

1. **All locations** (default) — shows all locations stacked vertically (Phase 1 layout)
2. **Single location** — shows one location with expanded detail (e.g., all pollen plants, full pressure history)
3. **Cross-location comparison** — shows one data type (e.g., pollen) across all locations side-by-side

Navigation would use keyboard: `Tab`/arrow keys to switch locations, `1`/`2`/`3` for view modes, `q` to quit.

## Terminal Width Handling

- Bubbletea sends `WindowSizeMsg` on startup and every resize
- The `Update` function stores width/height in the Model
- The `View` function uses the stored dimensions for `lipgloss.Place()` centering and to choose layout variants
- Minimum usable width: ~60 columns (below this, truncate or abbreviate)
- No horizontal scrolling — layout adapts to fit

## Refresh Mechanism

- `tea.Tick(refreshInterval, func(t time.Time) tea.Msg { return RefreshMsg(t) })`
- On `RefreshMsg`: fire an async `tea.Cmd` that calls dashboard-api via HTTP
- On success: update model with new data, re-render
- On failure: show error in status bar, keep displaying last successful data
- Default refresh interval: 60 seconds (configurable via env var or flag)

## Module Structure

The CLI lives under `clients/cli/` (not `services/`) because it is a display-only consumer of the dashboard-api, not a backend service. It is a standalone Go module with no local dependencies on any backend module — the REST endpoint is its only integration point.

```
clients/cli/
├── cmd/
│   └── main.go                  # Entry point, parse flags, start bubbletea
├── internal/
│   ├── tui/
│   │   ├── model.go             # Bubbletea Model, Init, Update, View (composes sections)
│   │   ├── styles.go            # Shared lipgloss styles (borders, colors, label column)
│   │   ├── header.go            # Title bar renderer
│   │   ├── weather.go           # Weather section renderer
│   │   ├── pressure.go          # Pressure section renderer
│   │   ├── pollen.go            # Pollen section renderer
│   │   └── status.go            # Bottom status bar renderer
│   └── client/
│       ├── dashboard.go         # HTTP client for GET /v1/dashboard
│       └── types.go             # JSON response structs matching protojson output
├── go.mod
└── Dockerfile
```

### Integration with monorepo

- **Not added to `go.work`** — the CLI is intentionally outside the Go workspace so it cannot import any backend module. Build/test targets pass `GOWORK=off` to respect that boundary.
- Makefile targets: `cli-dev`, `cli-build`, `cli-test`
- No import of `services/shared` or any other `services/*` package. The CLI inlines its own logging setup and derives render order from the REST response (sorted location IDs).
- Not added to `docker-compose.yml` for Phase 1 — a TUI in a detached container is awkward; run the binary locally against the compose stack or a staging URL.

## Configuration

| Setting | Source | Default |
|---------|--------|---------|
| Dashboard API URL | `DASHBOARD_API_URL` env var or `-url` flag | `http://localhost:8080` |
| Refresh interval | `REFRESH_INTERVAL` env var or `-refresh` flag | `60s` |

## Phase 1 Scope (this issue)

- [ ] Scaffold `clients/cli/` with go.mod, cmd/main.go
- [ ] HTTP client to fetch and parse `/v1/dashboard` JSON
- [ ] Bubbletea model with `WindowSizeMsg` handling and `tea.Tick` refresh
- [ ] Lipgloss styles: retro terminal aesthetic (dark bg, box-drawing borders)
- [ ] Render components: header, weather section, pressure section, pollen section, status bar
- [ ] Center layout in terminal
- [ ] Handle fetch errors gracefully (show in status bar)
- [ ] Makefile targets (CLI stays out of `go.work`)
- [ ] `q` key to quit

## Phase 2 Scope (future)

- [ ] Location selection (keyboard navigation)
- [ ] View mode switching (all locations / single location / cross-location comparison)
- [ ] Sparkline charts for pressure history (if data is available)
- [ ] Color-coded pollen severity
- [ ] Docker compose integration
