package repository

import "context"

// WeatherReader defines the interface for fetching weather data.
// This allows the Service layer to be tested using a mock repository.
type WeatherReader interface {
	GetAll(ctx context.Context) ([]CacheDoc, error)
	GetByID(ctx context.Context, id string) (*CacheDoc, error)
}
