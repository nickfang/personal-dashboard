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
// #80, and the point of this job is to gather the evidence that decision
// needs rather than to bake in a rule that has not been validated.
type Observation struct {
	Location string
	Now      time.Time

	// Observed is nil when weather_cache has no document for this location.
	Observed *ObservedReading

	ForecastIssuedAt time.Time
	ForecastAgeMin   int

	// ForecastAtObservedMb is what the forecast predicted for the moment the
	// barometer was actually read, and ErrorMb is how far the reading is from
	// it. That difference is the forecast error accumulated since the
	// forecast was issued — the number that decides whether anchoring alerts
	// on observed pressure is worth building at all (#80).
	//
	// Both are nil without an observation: there is no forecast error to
	// measure, and the forecast's own value is already in forecast_raw.
	ForecastAtObservedMb *float64
	ErrorMb              *float64

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

	// Forecast error compares a prediction and a measurement of the *same*
	// instant, so the forecast is sampled at the observation's timestamp
	// rather than at now. weather-collector is partial-failure tolerant and
	// leaves the previous document in place when it skips a location, so an
	// observation can be hours old; sampling at now would fold that much real
	// pressure change into what this field calls forecast error.
	if obs.Observed != nil {
		if mb, ok := forecastAt(forecast.Points, obs.Observed.At); ok {
			obs.ForecastAtObservedMb = &mb
			err := obs.Observed.PressureMb - mb
			obs.ErrorMb = &err
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
