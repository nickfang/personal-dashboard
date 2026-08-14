package repository

import (
	"time"

	"github.com/nickfang/personal-dashboard/services/shared"
)

// The structs below mirror documents owned by other services:
// weather_cache is written by weather-collector, forecast_cache by
// forecast-collector. Firestore ignores document fields absent from the
// destination struct, so these declare only what this job reads rather than
// duplicating the owners' full schemas. weather-provider does the same thing
// for the same reason.
//
// The cost of mirroring is that a shape change upstream fails at decode time
// rather than compile time. Keep these minimal so there is less to drift.

// ObservedPoint is the subset of weather-collector's WeatherPoint this job
// uses.
type ObservedPoint struct {
	Timestamp  time.Time `firestore:"timestamp"`
	PressureMb float64   `firestore:"pressure_mb"`
}

// WeatherCacheDoc mirrors weather_cache/{locationID}.
type WeatherCacheDoc struct {
	LastUpdated time.Time     `firestore:"last_updated"`
	Current     ObservedPoint `firestore:"current"`
}

// ForecastPoint is the subset of forecast-collector's ForecastPoint this job
// uses.
type ForecastPoint struct {
	ValidTime  time.Time `firestore:"valid_time"`
	PressureMb float64   `firestore:"pressure_mb"`
}

// ForecastCacheDoc mirrors forecast_cache/{locationID}.
type ForecastCacheDoc struct {
	Location string          `firestore:"location"`
	IssuedAt time.Time       `firestore:"issued_at"`
	Points   []ForecastPoint `firestore:"points"`
	Alerts   []shared.Alert  `firestore:"alerts"`
}
