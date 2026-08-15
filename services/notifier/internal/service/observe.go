package service

import (
	"time"

	"github.com/nickfang/personal-dashboard/services/notifier/internal/repository"
	"github.com/nickfang/personal-dashboard/services/shared"
)

// Offsets are the forecast horizons this job reports on, mirroring the
// backward offsets weather-collector already computes.
var Offsets = []time.Duration{3 * time.Hour, 6 * time.Hour, 12 * time.Hour, 24 * time.Hour}

// matchTolerance is how far a forecast point may sit from a requested offset
// and still be used. This follows the display paths' 45 minutes rather than
// detection's 30: an observation is a report, not a trigger, and a missing
// forecast hour should not blank a whole row.
const matchTolerance = 45 * time.Minute

// Observation is one location's state at one evaluation: what the barometer
// reads now, what the forecast expects, and how far apart they are.
//
// It records facts, not verdicts. Deciding what is worth sending is issue
// #80, and the point of this job is to gather evidence for that decision
// rather than to bake in a rule that has not been validated.
//
// It deliberately does not record forecast error. Observations and forecasts
// both come from weather.googleapis.com and agree to ~0.01 mb for the same
// hour, so that measurement is zero by construction — see #80.
type Observation struct {
	Location string
	Now      time.Time

	// Observed is nil when weather_cache has no document for this location.
	Observed *ObservedReading

	ForecastIssuedAt time.Time
	ForecastAgeMin   int

	// Forward holds the change from the observed reading to the forecast at
	// each offset. Nil deltas mean no forecast point fell within tolerance.
	Forward []ForwardDelta

	Alerts []AlertState
}

type ObservedReading struct {
	PressureMb float64
	At         time.Time
	AgeMin     int
}

type ForwardDelta struct {
	Offset  time.Duration
	DeltaMb *float64

	// MatchedAt is the forecast point actually used, which can sit up to
	// matchTolerance away from the requested offset. Recorded because a
	// "+3h" delta measured over 2h15m is a different quantity, and later
	// analysis has no way to recover this.
	MatchedAt time.Time
}

type AlertState struct {
	ID          string
	RuleID      string
	Severity    string
	Status      string
	Value       float64
	WindowStart time.Time
	WindowEnd   time.Time
	NotifiedAt  time.Time

	// HoursToWindow is negative once the window has opened.
	HoursToWindow float64
}

// BuildObservation assembles one location's record. It is pure: every input
// is passed in, including now, so a run that straddles an hour boundary
// cannot split across two evaluations.
func BuildObservation(locationID string, observed *repository.WeatherCacheDoc, forecast *repository.ForecastCacheDoc, now time.Time) Observation {
	obs := Observation{
		Location:         locationID,
		Now:              now,
		ForecastIssuedAt: forecast.IssuedAt,
		ForecastAgeMin:   minutesSince(forecast.IssuedAt, now),
		Forward:          make([]ForwardDelta, 0, len(Offsets)),
		Alerts:           make([]AlertState, 0, len(forecast.Alerts)),
	}

	if observed != nil {
		obs.Observed = &ObservedReading{
			PressureMb: observed.Current.PressureMb,
			At:         observed.Current.Timestamp,
			AgeMin:     minutesSince(observed.Current.Timestamp, now),
		}
	}

	// Forward targets are anchored on now, not on the observation: the
	// question is what happens over the next N hours from here. A stale
	// observation therefore widens the true interval, which is what
	// ObservedAgeMin is for.
	for _, offset := range Offsets {
		d := ForwardDelta{Offset: offset}
		target := now.Add(offset)
		if i := nearestPoint(forecast.Points, target); i >= 0 {
			d.MatchedAt = forecast.Points[i].ValidTime
			if obs.Observed != nil {
				delta := forecast.Points[i].PressureMb - obs.Observed.PressureMb
				d.DeltaMb = &delta
			}
		}
		obs.Forward = append(obs.Forward, d)
	}

	for _, a := range forecast.Alerts {
		obs.Alerts = append(obs.Alerts, AlertState{
			ID:            a.ID,
			RuleID:        a.RuleID,
			Severity:      a.Severity,
			Status:        a.Status,
			Value:         a.Value,
			WindowStart:   a.WindowStart,
			WindowEnd:     a.WindowEnd,
			NotifiedAt:    a.NotifiedAt,
			HoursToWindow: a.WindowStart.Sub(now).Hours(),
		})
	}

	return obs
}

// forecastAt returns the forecast pressure nearest the given time.
func forecastAt(points []repository.ForecastPoint, target time.Time) (float64, bool) {
	if i := nearestPoint(points, target); i >= 0 {
		return points[i].PressureMb, true
	}
	return 0, false
}

func nearestPoint(points []repository.ForecastPoint, target time.Time) int {
	return shared.NearestIndex(points, func(p repository.ForecastPoint) time.Time {
		return p.ValidTime
	}, target, matchTolerance)
}

func minutesSince(t, now time.Time) int {
	return int(now.Sub(t).Round(time.Minute) / time.Minute)
}
