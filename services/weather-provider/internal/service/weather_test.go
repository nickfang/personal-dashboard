package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nickfang/personal-dashboard/services/weather-provider/internal/repository"
)

// MockRepository implements repository.WeatherReader
type MockRepository struct {
	GetAllFunc  func(ctx context.Context) ([]repository.CacheDoc, error)
	GetByIDFunc func(ctx context.Context, id string) (*repository.CacheDoc, error)
}

func (m *MockRepository) GetAll(ctx context.Context) ([]repository.CacheDoc, error) {
	return m.GetAllFunc(ctx)
}

func (m *MockRepository) GetByID(ctx context.Context, id string) (*repository.CacheDoc, error) {
	return m.GetByIDFunc(ctx, id)
}

func TestGetStatsByID(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := &MockRepository{
			GetByIDFunc: func(ctx context.Context, id string) (*repository.CacheDoc, error) {
				return &repository.CacheDoc{
					LocationID:  id,
					LastUpdated: time.Now(),
				}, nil
			},
		}
		svc := NewWeatherService(mockRepo)

		res, err := svc.GetStatsByID(context.Background(), "test-loc")

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if res.LocationID != "test-loc" {
			t.Errorf("expected location test-loc, got %s", res.LocationID)
		}
	})

	t.Run("Error", func(t *testing.T) {
		mockRepo := &MockRepository{
			GetByIDFunc: func(ctx context.Context, id string) (*repository.CacheDoc, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewWeatherService(mockRepo)

		_, err := svc.GetStatsByID(context.Background(), "test-loc")

		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}
