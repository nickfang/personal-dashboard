package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/api"
	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/repository"
	"github.com/nickfang/personal-dashboard/services/shared"
	"github.com/nickfang/personal-dashboard/services/shared/notify"
)

// Config holds the collector's tuning, kept separate from the interfaces it
// depends on so the constructor's parameter list distinguishes what the
// service talks to from how it is configured.
type Config struct {
	HorizonHours int         // FORECAST_HORIZON_HOURS: how many forecast hours to request
	Alert        AlertConfig // Pressure-drop detection thresholds
}

// CollectorService orchestrates the forecast collection flow.
type CollectorService struct {
	fetcher api.Fetcher
	writer  repository.Writer
	sender  notify.Sender
	cfg     Config
}

// NewCollectorService creates a new CollectorService with injected dependencies.
func NewCollectorService(fetcher api.Fetcher, writer repository.Writer, sender notify.Sender, cfg Config) *CollectorService {
	return &CollectorService{fetcher: fetcher, writer: writer, sender: sender, cfg: cfg}
}

// Collect fetches the forecast for a location, maps it, and writes to storage.
func (s *CollectorService) Collect(ctx context.Context, apiKey string, location shared.Location) error {
	hours, err := s.fetcher.Fetch(apiKey, location, s.cfg.HorizonHours)
	if err != nil {
		return fmt.Errorf("fetching forecast for %s: %w", location.ID, err)
	}
	points := MapRun(hours)
	if len(points) == 0 {
		return fmt.Errorf("no valid forecast points for %s (%d hours fetched)", location.ID, len(hours))
	}
	run := repository.ForecastRun{
		Location: location.ID,
		IssuedAt: time.Now(),
		Points:   points,
	}
	if err := s.writer.SaveRaw(ctx, run); err != nil {
		return fmt.Errorf("saving forecast run for %s: %w", location.ID, err)
	}
	// Detection is pure and runs once; the merge against stored alerts runs
	// inside the repository's transaction (it may retry on contention).
	detected := DetectPressureAlerts(location.ID, points, s.cfg.Alert, run.IssuedAt)
	merge := func(prev []shared.Alert) []shared.Alert {
		return shared.MergeAlerts(prev, detected, run.IssuedAt)
	}
	committed, err := s.writer.UpdateCache(ctx, location.ID, run, merge)
	if err != nil {
		return fmt.Errorf("updating forecast cache for %s: %w", location.ID, err)
	}
	// Delivery runs after the cache write commits, and its failures are
	// logged rather than returned: a notification problem should not make a
	// successful collection look like a failed one.
	s.deliver(ctx, location.ID, committed, run.IssuedAt)
	return nil
}

// deliver sends every alert that is currently in the forecast and has not
// been delivered, then records which ones went out.
//
// Order matters: deliver, then mark. If marking fails the alert re-delivers
// on the next run, whereas marking first risks recording a delivery that
// never happened — the worse failure for an alerting system.
func (s *CollectorService) deliver(ctx context.Context, locationID string, alerts []shared.Alert, at time.Time) {
	var delivered []string
	for _, a := range alerts {
		if a.Status != shared.AlertStatusActive || !a.NotifiedAt.IsZero() {
			continue
		}
		if err := s.sender.Send(ctx, notify.FromAlert(a)); err != nil {
			slog.Error("Failed to deliver alert", "location", locationID, "alert", a.ID, "error", err)
			continue
		}
		slog.Info("Delivered alert", "location", locationID, "alert", a.ID, "severity", a.Severity)
		delivered = append(delivered, a.ID)
	}
	if len(delivered) == 0 {
		return
	}
	if err := s.writer.MarkNotified(ctx, locationID, delivered, at); err != nil {
		// The alerts will re-deliver on the next run.
		slog.Error("Failed to mark alerts as notified", "location", locationID, "alerts", delivered, "error", err)
	}
}
