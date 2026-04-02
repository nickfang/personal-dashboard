package service

import (
	"context"

	"github.com/nickfang/personal-dashboard/services/weather-provider/internal/repository"
)

type WeatherService struct {
	repo repository.WeatherReader
}

func NewWeatherService(repo repository.WeatherReader) *WeatherService {
	return &WeatherService{repo: repo}
}

func (s *WeatherService) GetStatsByID(ctx context.Context, id string) (*repository.PressureCacheDoc, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *WeatherService) GetAllStats(ctx context.Context) ([]repository.PressureCacheDoc, error) {
	return s.repo.GetAll(ctx)
}

func (s *WeatherService) GetLastWeather(ctx context.Context, id string) (*repository.WeatherCacheDoc, error) {
	return s.repo.GetLastWeather(ctx, id)
}

func (s *WeatherService) GetAllLastWeather(ctx context.Context) ([]repository.WeatherCacheDoc, error) {
	return s.repo.GetAllLastWeather(ctx)
}

func (s *WeatherService) GetAllRaw(ctx context.Context) ([]repository.WeatherPoint, error) {
	return s.repo.GetAllRaw(ctx)
}
