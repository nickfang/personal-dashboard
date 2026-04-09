package repository

import "context"

// WeatherReader defines the interface for fetching weather data.
// This allows the Service layer to be tested using a mock repository.
type WeatherReader interface {
	GetByID(ctx context.Context, id string) (*PressureCacheDoc, error)
	GetAll(ctx context.Context) ([]PressureCacheDoc, error)
	GetLastWeather(ctx context.Context, id string) (*WeatherCacheDoc, error)
	GetAllLastWeather(ctx context.Context) ([]WeatherCacheDoc, error)
	GetAllRaw(ctx context.Context) ([]WeatherPoint, error)
}
