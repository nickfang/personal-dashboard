package service

import (
	"testing"
	"time"

	"github.com/nickfang/personal-dashboard/services/notifier/internal/repository"
	"github.com/nickfang/personal-dashboard/services/shared"
)

var testNow = time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)

// forecastPoints builds hourly points starting at testNow, one per pressure.
func forecastPoints(pressures ...float64) []repository.ForecastPoint {
	pts := make([]repository.ForecastPoint, len(pressures))
	for i, p := range pressures {
		pts[i] = repository.ForecastPoint{
			ValidTime:  testNow.Add(time.Duration(i) * time.Hour),
			PressureMb: p,
		}
	}
	return pts
}

func observedAt(pressure float64, ago time.Duration) *repository.WeatherCacheDoc {
	return &repository.WeatherCacheDoc{
		LastUpdated: testNow.Add(-ago),
		Current: repository.ObservedPoint{
			Timestamp:  testNow.Add(-ago),
			PressureMb: pressure,
		},
	}
}

func deltaAt(t *testing.T, o Observation, offset time.Duration) *float64 {
	t.Helper()
	for _, d := range o.Forward {
		if d.Offset == offset {
			return d.DeltaMb
		}
	}
	t.Fatalf("no forward delta recorded for offset %v", offset)
	return nil
}

func TestBuildObservation_ForwardDeltasAnchorOnObserved(t *testing.T) {
	// 25 hourly points so every offset resolves. Pressure falls 1 mb/hour.
	pressures := make([]float64, 25)
	for i := range pressures {
		pressures[i] = 1013 - float64(i)
	}
	forecast := &repository.ForecastCacheDoc{IssuedAt: testNow, Points: forecastPoints(pressures...)}

	// Observed sits 2 mb below the forecast's own value for now (1013).
	o := BuildObservation("house-nick", observedAt(1011, 10*time.Minute), forecast, testNow)

	// +3h forecast is 1010; from observed 1011 that is -1, not the -3 a
	// forecast-internal delta would report. That difference is the point.
	if got := deltaAt(t, o, 3*time.Hour); got == nil || *got != -1 {
		t.Errorf("fwd_03h = %v, want -1 (anchored on observed 1011, not forecast 1013)", got)
	}
	if got := deltaAt(t, o, 24*time.Hour); got == nil || *got != -22 {
		t.Errorf("fwd_24h = %v, want -22", got)
	}
}

func TestBuildObservation_RecordsForecastError(t *testing.T) {
	forecast := &repository.ForecastCacheDoc{IssuedAt: testNow.Add(-4 * time.Hour), Points: forecastPoints(1013, 1012, 1011)}

	o := BuildObservation("house-nick", observedAt(1009.5, 5*time.Minute), forecast, testNow)

	if o.ForecastAtObservedMb == nil || *o.ForecastAtObservedMb != 1013 {
		t.Fatalf("ForecastAtObservedMb = %v, want 1013", o.ForecastAtObservedMb)
	}
	// The barometer is 3.5 mb below where the forecast said it would be —
	// this is the number that decides whether the anchor is worth building.
	if o.ErrorMb == nil || *o.ErrorMb != -3.5 {
		t.Errorf("ErrorMb = %v, want -3.5", o.ErrorMb)
	}
	if o.ForecastAgeMin != 240 {
		t.Errorf("ForecastAgeMin = %d, want 240", o.ForecastAgeMin)
	}
}

func TestBuildObservation_NilDeltaWhenHorizonFallsShort(t *testing.T) {
	// Only 4 hours of forecast: +3h resolves, +6h and beyond cannot.
	forecast := &repository.ForecastCacheDoc{IssuedAt: testNow, Points: forecastPoints(1013, 1012, 1011, 1010)}

	o := BuildObservation("house-nick", observedAt(1013, 0), forecast, testNow)

	if got := deltaAt(t, o, 3*time.Hour); got == nil {
		t.Error("fwd_03h should resolve within the 4-hour horizon")
	}
	for _, offset := range []time.Duration{6 * time.Hour, 12 * time.Hour, 24 * time.Hour} {
		if got := deltaAt(t, o, offset); got != nil {
			t.Errorf("fwd_%v = %v, want nil beyond the horizon", offset, *got)
		}
	}
	if len(o.Forward) != len(Offsets) {
		t.Errorf("recorded %d offsets, want %d — an unresolved offset is still a row", len(o.Forward), len(Offsets))
	}
}

func TestBuildObservation_RecordsWhichPointWasMatched(t *testing.T) {
	// Points on the half hour: the +3h target at 15:00 matches 14:30, which
	// is 30 minutes off. A delta labelled "+3h" measured over 2h30m is a
	// different quantity, and nothing downstream can recover that.
	forecast := &repository.ForecastCacheDoc{IssuedAt: testNow}
	for i := range 8 {
		forecast.Points = append(forecast.Points, repository.ForecastPoint{
			ValidTime:  testNow.Add(time.Duration(i)*time.Hour + 30*time.Minute),
			PressureMb: 1013 - float64(i),
		})
	}

	o := BuildObservation("house-nick", observedAt(1013, 0), forecast, testNow)

	want := testNow.Add(2*time.Hour + 30*time.Minute)
	for _, d := range o.Forward {
		if d.Offset != 3*time.Hour {
			continue
		}
		if !d.MatchedAt.Equal(want) {
			t.Errorf("fwd_03h matched %v, want %v", d.MatchedAt, want)
		}
	}
}

func TestBuildObservation_MissingObservationIsRecordedNotFatal(t *testing.T) {
	forecast := &repository.ForecastCacheDoc{IssuedAt: testNow, Points: forecastPoints(1013, 1012, 1011, 1010)}

	o := BuildObservation("house-nick", nil, forecast, testNow)

	if o.Observed != nil {
		t.Error("Observed should be nil when weather_cache has no document")
	}
	if o.ErrorMb != nil {
		t.Error("ErrorMb needs an observation to be meaningful")
	}
	// Without a reading there is no forecast error to measure, and the
	// forecast's own value is already retained in forecast_raw.
	if o.ForecastAtObservedMb != nil {
		t.Errorf("ForecastAtObservedMb = %v, want nil without an observation", *o.ForecastAtObservedMb)
	}
	if got := deltaAt(t, o, 3*time.Hour); got != nil {
		t.Errorf("fwd_03h = %v, want nil — there is nothing to anchor on", *got)
	}
}

func TestBuildObservation_ObservedAge(t *testing.T) {
	forecast := &repository.ForecastCacheDoc{IssuedAt: testNow, Points: forecastPoints(1013)}

	o := BuildObservation("house-nick", observedAt(1013, 75*time.Minute), forecast, testNow)

	if o.Observed == nil || o.Observed.AgeMin != 75 {
		t.Errorf("ObservedAgeMin = %v, want 75", o.Observed)
	}
}

func TestBuildObservation_MapsAlertsWithLeadTime(t *testing.T) {
	notified := testNow.Add(-3 * time.Hour)
	forecast := &repository.ForecastCacheDoc{
		IssuedAt: testNow,
		Points:   forecastPoints(1013, 1012),
		Alerts: []shared.Alert{
			{
				ID: "a1", RuleID: "pressure-drop-3h", Severity: shared.AlertSeverityWarning,
				Status: shared.AlertStatusActive, Value: -6.2,
				WindowStart: testNow.Add(11 * time.Hour), WindowEnd: testNow.Add(14 * time.Hour),
			},
			{
				ID: "a2", RuleID: "pressure-drop-3h", Severity: shared.AlertSeveritySevere,
				Status: shared.AlertStatusActive, Value: -11.0, NotifiedAt: notified,
				WindowStart: testNow.Add(-2 * time.Hour), WindowEnd: testNow.Add(time.Hour),
			},
		},
	}

	o := BuildObservation("house-nick", observedAt(1013, 0), forecast, testNow)

	if len(o.Alerts) != 2 {
		t.Fatalf("recorded %d alerts, want 2", len(o.Alerts))
	}
	if o.Alerts[0].HoursToWindow != 11 {
		t.Errorf("a1 HoursToWindow = %v, want 11", o.Alerts[0].HoursToWindow)
	}
	// Negative once the window has opened — that distinction is the whole
	// point of recording it.
	if o.Alerts[1].HoursToWindow != -2 {
		t.Errorf("a2 HoursToWindow = %v, want -2", o.Alerts[1].HoursToWindow)
	}
	if !o.Alerts[1].NotifiedAt.Equal(notified) {
		t.Errorf("a2 NotifiedAt = %v, want %v", o.Alerts[1].NotifiedAt, notified)
	}
}

func TestBuildObservation_ErrorUsesTheObservationsOwnHour(t *testing.T) {
	// weather-collector is partial-failure tolerant: when it skips a
	// location, weather_cache keeps the previous document. Sampling the
	// forecast at now rather than at the reading's timestamp would fold that
	// gap's real pressure change into what this field calls forecast error —
	// here it would flip the sign.
	var points []repository.ForecastPoint
	for i, mb := range []float64{1020, 1018, 1016, 1014, 1012, 1010, 1008} {
		points = append(points, repository.ForecastPoint{
			ValidTime:  testNow.Add(time.Duration(i-3) * time.Hour),
			PressureMb: mb,
		})
	}
	forecast := &repository.ForecastCacheDoc{IssuedAt: testNow.Add(-3 * time.Hour), Points: points}

	// Read two hours ago at 1017, against a forecast of 1018 for that hour.
	o := BuildObservation("house-nick", observedAt(1017, 2*time.Hour), forecast, testNow)

	if o.ForecastAtObservedMb == nil || *o.ForecastAtObservedMb != 1018 {
		t.Fatalf("ForecastAtObservedMb = %v, want 1018 (the forecast for the reading's hour, not for now)", o.ForecastAtObservedMb)
	}
	if o.ErrorMb == nil || *o.ErrorMb != -1 {
		t.Errorf("ErrorMb = %v, want -1; sampling the forecast at now would give +3", o.ErrorMb)
	}
	if o.Observed.AgeMin != 120 {
		t.Errorf("ObservedAgeMin = %d, want 120 — the caveat this measurement carries", o.Observed.AgeMin)
	}
}
