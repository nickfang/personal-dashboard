package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nickfang/personal-dashboard/clients/cli/internal/client"
	tui "github.com/nickfang/personal-dashboard/clients/cli/internal/tui"
)

const (
	appName        = "pd-cli"
	defaultURL     = "http://localhost:8080"
	defaultRefresh = 300 * time.Second
	envURL         = "DASHBOARD_API_URL"
	envRefresh     = "REFRESH_INTERVAL"
)

func main() {
	// Defaults come from env vars; flags override.
	urlDefault := defaultURL
	if v := os.Getenv(envURL); v != "" {
		urlDefault = v
	}
	refreshDefault := defaultRefresh
	if v := os.Getenv(envRefresh); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			refreshDefault = d
		} else {
			fmt.Fprintf(os.Stderr, "invalid %s=%q: %v (using default %s)\n", envRefresh, v, err, defaultRefresh)
		}
	}

	urlFlag := flag.String("url", urlDefault, "Dashboard API base URL (env: DASHBOARD_API_URL)")
	refreshFlag := flag.Duration("refresh", refreshDefault, "Refresh interval (env: REFRESH_INTERVAL)")
	flag.Parse()

	initLogging()
	slog.Info(appName+" starting", "url", *urlFlag, "refresh", refreshFlag.String())

	apiClient := client.New(*urlFlag)

	m := tui.NewModel(apiClient, *refreshFlag)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		slog.Error(appName+" exited with error", "err", err)
		os.Exit(1)
	}
}

func initLogging() {
	level := slog.LevelInfo
	if os.Getenv("DEBUG") == "true" {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
}
