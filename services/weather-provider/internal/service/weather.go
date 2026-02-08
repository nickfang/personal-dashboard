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

func (s *WeatherService) GetAllStats(ctx context.Context) ([]repository.CacheDoc, error) {
	return s.repo.GetAll(ctx)
}

func (s *WeatherService) GetStatsByID(ctx context.Context, id string) (*repository.CacheDoc, error) {
	return s.repo.GetByID(ctx, id)
}
