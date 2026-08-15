package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nickfang/personal-dashboard/services/notifier/internal/repository"
	"github.com/nickfang/personal-dashboard/services/shared"
	"time"
)

// NotifierService reads the cache documents for a location and records what
// it sees.
//
// It sends nothing. Delivery lives in forecast-collector and stays there
// until the gate is designed (#79, #80); this job exists to make that design
// answerable from measurements rather than argument. It holds no credentials,
// so sending is not merely disabled but structurally unavailable.
type NotifierService struct {
	store repository.Store
}

func NewNotifierService(store repository.Store) *NotifierService {
	return &NotifierService{store: store}
}

// Observe reads one location, logs its state, and returns the record it
// built. Returns an error only when the location could not be read at all.
//
// The Observation is returned rather than discarded so callers can assert on
// it, and so the delivery gate can be layered on here without restructuring
// when #79 and #80 land.
func (s *NotifierService) Observe(ctx context.Context, location shared.Location, now time.Time) (Observation, error) {
	forecast, err := s.store.ReadForecast(ctx, location.ID)
	if err != nil {
		return Observation{}, err
	}
	// The two Store reads have deliberately opposite nil conventions —
	// ReadObservation returns (nil, nil) for a missing document, ReadForecast
	// returns an error — and only a doc comment enforces the difference. Turn
	// a violation into this location's failure rather than a panic, which
	// would take down the other locations with it.
	if forecast == nil {
		return Observation{}, fmt.Errorf("store returned no forecast and no error for %s", location.ID)
	}
	// A missing observation is recorded, not fatal: weather-collector may not
	// have run for this location, and knowing that is itself a finding.
	observed, err := s.store.ReadObservation(ctx, location.ID)
	if err != nil {
		return Observation{}, err
	}

	obs := BuildObservation(location.ID, observed, forecast, now)
	logObservation(obs)
	return obs, nil
}

func logObservation(o Observation) {
	attrs := []any{
		"location", o.Location,
		"forecast_issued_at", o.ForecastIssuedAt,
		"forecast_age_min", o.ForecastAgeMin,
	}
	if o.Observed != nil {
		attrs = append(attrs,
			"observed_mb", o.Observed.PressureMb,
			"observed_at", o.Observed.At,
			"observed_age_min", o.Observed.AgeMin,
		)
	} else {
		attrs = append(attrs, "observed_missing", true)
	}
	if o.ForecastAtObservedMb != nil {
		attrs = append(attrs, "forecast_at_observed_mb", *o.ForecastAtObservedMb)
	}
	if o.ErrorMb != nil {
		attrs = append(attrs, "error_mb", *o.ErrorMb)
	}
	for _, d := range o.Forward {
		key := fmt.Sprintf("fwd_%02dh", int(d.Offset.Hours()))
		if d.DeltaMb != nil {
			attrs = append(attrs, key, *d.DeltaMb, key+"_matched", d.MatchedAt)
		} else {
			attrs = append(attrs, key, nil)
		}
	}
	slog.Info("observation", attrs...)

	for _, a := range o.Alerts {
		slog.Info("alert seen",
			"location", o.Location,
			"alert", a.ID,
			"rule", a.RuleID,
			"severity", a.Severity,
			"status", a.Status,
			"value", a.Value,
			"window_start", a.WindowStart,
			"window_end", a.WindowEnd,
			"notified", !a.NotifiedAt.IsZero(),
			"hours_to_window", a.HoursToWindow,
		)
	}
}
