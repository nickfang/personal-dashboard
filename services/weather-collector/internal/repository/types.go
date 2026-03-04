package repository

import "time"

const MaxHistoryPoints = 48

type WeatherPoint struct {
	Location  string    `firestore:"location"`
	Timestamp time.Time `firestore:"timestamp"`

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
	VisibilityKm float64 `firestore:"visibility_km"`

	TempF        float64 `firestore:"temp_f"`
	TempFeelF    float64 `firestore:"temp_feel_f"`
	WindSpeedMph float64 `firestore:"wind_speed_mph"`
	WindGustMph  float64 `firestore:"wind_gust_mph"`
	VisibilityM  float64 `firestore:"visibility_miles"`
	DewpointF    float64 `firestore:"dewpoint_f"`
}

type PressurePoint struct {
	TimeStamp       time.Time `firestore:"timestamp"`
	HumidityPercent int       `firestore:"humidity_pct"`
	PressureMb      float64   `firestore:"pressure_mb"`

	TempC     float64 `firestore:"temp_c"`
	TempFeelC float64 `firestore:"temp_feel_c"`
	DewpointC float64 `firestore:"dewpoint_c"`

	TempF     float64 `firestore:"temp_f"`
	TempFeelF float64 `firestore:"temp_feel_f"`
	DewpointF float64 `firestore:"dewpoint_f"`
}

type PressureStats struct {
	// Pointers are used for Delta fields to support a true "N/A" (nil) state.
	// This allows the dashboard to distinguish between a 0.0 change and missing data.
	Timestamp time.Time `firestore:"timestamp"`
	Delta1h   *float64  `firestore:"delta_01h"`
	Delta3h   *float64  `firestore:"delta_03h"`
	Delta6h   *float64  `firestore:"delta_06h"`
	Delta12h  *float64  `firestore:"delta_12h"`
	Delta24h  *float64  `firestore:"delta_24h"`
	Trend     string    `firestore:"trend"`
}

type CacheDoc struct {
	LastUpdated  time.Time       `firestore:"last_updated"`
	CurrentValue WeatherPoint    `firestore:"current"`
	Analysis     PressureStats   `firestore:"analysis"`
	History      []PressurePoint `firestore:"history"`
}
