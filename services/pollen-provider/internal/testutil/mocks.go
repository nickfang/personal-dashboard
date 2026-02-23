package testutil

import (
	"context"
	"fmt"

	"github.com/nickfang/personal-dashboard/services/pollen-provider/internal/repository"
)

// MockReader implements repository.PollenReader for testing.
// Unset function fields return a descriptive error instead of panicking.
type MockReader struct {
	GetAllFunc  func(ctx context.Context) ([]repository.CacheDoc, error)
	GetByIDFunc func(ctx context.Context, id string) (*repository.CacheDoc, error)
}

func (m *MockReader) GetAll(ctx context.Context) ([]repository.CacheDoc, error) {
	if m.GetAllFunc == nil {
		return nil, fmt.Errorf("GetAll not mocked")
	}
	return m.GetAllFunc(ctx)
}

func (m *MockReader) GetByID(ctx context.Context, id string) (*repository.CacheDoc, error) {
	if m.GetByIDFunc == nil {
		return nil, fmt.Errorf("GetByID not mocked")
	}
	return m.GetByIDFunc(ctx, id)
}
