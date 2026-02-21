package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nickfang/personal-dashboard/services/pollen-provider/internal/repository"
)

// MockRepository implements repository.PollenReader
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

func TestGetReportByID(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		mockRepo := &MockRepository{
			GetByIDFunc: func(ctx context.Context, id string) (*repository.CacheDoc, error) {
				return &repository.CacheDoc{
					LocationID:  id,
					LastUpdated: now,
					CurrentValue: repository.PollenSnapshot{
						CollectedAt:     now,
						OverallIndex:    4,
						OverallCategory: "High",
						DominantType:    "TREE",
					},
				}, nil
			},
		}
		svc := NewPollenService(mockRepo)

		res, err := svc.GetReportByID(context.Background(), "house-nick")

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if res.LocationID != "house-nick" {
			t.Errorf("expected location house-nick, got %s", res.LocationID)
		}
		if res.CurrentValue.OverallIndex != 4 {
			t.Errorf("expected OverallIndex 4, got %d", res.CurrentValue.OverallIndex)
		}
	})

	t.Run("Error", func(t *testing.T) {
		mockRepo := &MockRepository{
			GetByIDFunc: func(ctx context.Context, id string) (*repository.CacheDoc, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewPollenService(mockRepo)

		_, err := svc.GetReportByID(context.Background(), "house-nick")

		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestGetAllReports(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		mockRepo := &MockRepository{
			GetAllFunc: func(ctx context.Context) ([]repository.CacheDoc, error) {
				return []repository.CacheDoc{
					{LocationID: "house-nick", LastUpdated: now},
					{LocationID: "house-nita", LastUpdated: now},
					{LocationID: "distribution-hall", LastUpdated: now},
				}, nil
			},
		}
		svc := NewPollenService(mockRepo)

		results, err := svc.GetAllReports(context.Background())

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(results) != 3 {
			t.Errorf("expected 3 reports, got %d", len(results))
		}
		if results[0].LocationID != "house-nick" {
			t.Errorf("expected first location house-nick, got %s", results[0].LocationID)
		}
	})

	t.Run("Error", func(t *testing.T) {
		mockRepo := &MockRepository{
			GetAllFunc: func(ctx context.Context) ([]repository.CacheDoc, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewPollenService(mockRepo)

		_, err := svc.GetAllReports(context.Background())

		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}
