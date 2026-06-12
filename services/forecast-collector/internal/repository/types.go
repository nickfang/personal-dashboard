package repository

import (
	"time"

	"github.com/nickfang/personal-dashboard/services/shared"
)

// ForecastPoint is one forecast hour in storage form.
type ForecastPoint struct {
	ValidTime time.Time `firestore:"valid_time"`

	HumidityPercent      int     `firestore:"humidity_pct"`
	PrecipitationPercent int     `firestore:"precipitation_pct"`
	UVIndex              int     `firestore:"uv_index"`
	PressureMb           float64 `firestore:"pressure_mb"`
	WindDirDeg           int     `firestore:"wind_dir_deg"`

	TempC        float64 `firestore:"temp_c"`
	TempFeelC    float64 `firestore:"temp_feel_c"`
	DewpointC    float64 `firestore:"dewpoint_c"`
	WindSpeedKph float64 `firestore:"wind_speed_kph"`
	WindGustKph  float64 `firestore:"wind_gust_kph"`

	TempF     float64 `firestore:"temp_f"`
	TempFeelF float64 `firestore:"temp_feel_f"`
	DewpointF float64 `firestore:"dewpoint_f"`
}

// ForecastRun is one append-only audit record in forecast_raw: the full
// forecast snapshot produced by a single collector run for one location.
type ForecastRun struct {
	Location     string          `firestore:"location"`
	IssuedAt     time.Time       `firestore:"issued_at"`
	HorizonHours int             `firestore:"horizon_hours"`
	Points       []ForecastPoint `firestore:"points"`
}

// ForecastCacheDoc is the latest forecast for a location, stored at
// forecast_cache/{locationID} and fully replaced on each run.
type ForecastCacheDoc struct {
	Location    string          `firestore:"location"`
	LastUpdated time.Time       `firestore:"last_updated"`
	IssuedAt    time.Time       `firestore:"issued_at"`
	Points      []ForecastPoint `firestore:"points"`
	Alerts      []shared.Alert  `firestore:"alerts"`
}
