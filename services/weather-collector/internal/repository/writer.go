package repository

import "context"

// AnalyzeFunc is a callback that computes pressure statistics from history.
// This allows the service layer's business logic to run inside the repository's transaction.
type AnalyzeFunc func(history []PressurePoint) PressureStats

// Writer defines the interface for writing weather data to storage.
type Writer interface {
	SaveRaw(ctx context.Context, wp WeatherPoint) error
	UpdateCache(ctx context.Context, locationID string, wp WeatherPoint, analyze AnalyzeFunc) error
}
