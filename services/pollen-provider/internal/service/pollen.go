package service

import (
	"context"

	"github.com/nickfang/personal-dashboard/services/pollen-provider/internal/repository"
)

type PollenService struct {
	repo repository.PollenReader
}

func NewPollenService(repo repository.PollenReader) *PollenService {
	return &PollenService{repo: repo}
}

func (s *PollenService) GetAllReports(ctx context.Context) ([]repository.CacheDoc, error) {
	return s.repo.GetAll(ctx)
}

func (s *PollenService) GetReportByID(ctx context.Context, id string) (*repository.CacheDoc, error) {
	return s.repo.GetByID(ctx, id)
}
