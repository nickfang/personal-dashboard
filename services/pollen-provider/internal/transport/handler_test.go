package transport

import (
	"context"
	"fmt"
	"testing"
	"time"

	pb "github.com/nickfang/personal-dashboard/services/pollen-provider/internal/gen/go/pollen-provider/v1"
	"github.com/nickfang/personal-dashboard/services/pollen-provider/internal/repository"
	"github.com/nickfang/personal-dashboard/services/pollen-provider/internal/service"
)

// MockReader implements repository.PollenReader for handler tests
type MockReader struct {
	GetByIDFunc func(ctx context.Context, id string) (*repository.CacheDoc, error)
	GetAllFunc  func(ctx context.Context) ([]repository.CacheDoc, error)
}

func (m *MockReader) GetAll(ctx context.Context) ([]repository.CacheDoc, error) {
	return m.GetAllFunc(ctx)
}

func (m *MockReader) GetByID(ctx context.Context, id string) (*repository.CacheDoc, error) {
	return m.GetByIDFunc(ctx, id)
}

func TestGetPollenReport_Mapping(t *testing.T) {
	now := time.Now()
	mockRepo := &MockReader{
		GetByIDFunc: func(ctx context.Context, id string) (*repository.CacheDoc, error) {
			return &repository.CacheDoc{
				LocationID:  id,
				LastUpdated: now,
				CurrentValue: repository.PollenSnapshot{
					CollectedAt:     now,
					OverallIndex:    4,
					OverallCategory: "High",
					DominantType:    "TREE",
					Types: []repository.StorePollenType{
						{Code: "TREE", Index: 4, Category: "High", InSeason: true},
						{Code: "GRASS", Index: 1, Category: "Very Low", InSeason: false},
						{Code: "WEED", Index: 0, Category: "None", InSeason: false},
					},
					Plants: []repository.StorePollenPlant{
						{Code: "JUNIPER", DisplayName: "Juniper", Index: 4, Category: "High", InSeason: true},
						{Code: "OAK", DisplayName: "Oak", Index: 0, Category: "None", InSeason: false},
					},
				},
			}, nil
		},
	}

	svc := service.NewPollenService(mockRepo)
	handler := NewGrpcHandler(svc)

	req := &pb.GetPollenReportRequest{LocationId: "house-nick"}
	resp, err := handler.GetPollenReport(context.Background(), req)

	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Verify overall summary mapping
	if resp.Report.LocationId != "house-nick" {
		t.Errorf("LocationId = %s, want house-nick", resp.Report.LocationId)
	}
	if resp.Report.OverallIndex != 4 {
		t.Errorf("OverallIndex = %d, want 4", resp.Report.OverallIndex)
	}
	if resp.Report.OverallCategory != "High" {
		t.Errorf("OverallCategory = %s, want High", resp.Report.OverallCategory)
	}
	if resp.Report.DominantType != "TREE" {
		t.Errorf("DominantType = %s, want TREE", resp.Report.DominantType)
	}

	// Verify types mapping
	if len(resp.Report.Types) != 3 {
		t.Fatalf("len(Types) = %d, want 3", len(resp.Report.Types))
	}
	if resp.Report.Types[0].Code != "TREE" {
		t.Errorf("Types[0].Code = %s, want TREE", resp.Report.Types[0].Code)
	}
	if resp.Report.Types[0].Index != 4 {
		t.Errorf("Types[0].Index = %d, want 4", resp.Report.Types[0].Index)
	}
	if resp.Report.Types[0].InSeason != true {
		t.Errorf("Types[0].InSeason = %v, want true", resp.Report.Types[0].InSeason)
	}

	// Verify plants mapping
	if len(resp.Report.Plants) != 2 {
		t.Fatalf("len(Plants) = %d, want 2", len(resp.Report.Plants))
	}
	if resp.Report.Plants[0].Code != "JUNIPER" {
		t.Errorf("Plants[0].Code = %s, want JUNIPER", resp.Report.Plants[0].Code)
	}
	if resp.Report.Plants[0].DisplayName != "Juniper" {
		t.Errorf("Plants[0].DisplayName = %s, want Juniper", resp.Report.Plants[0].DisplayName)
	}
	if resp.Report.Plants[0].Index != 4 {
		t.Errorf("Plants[0].Index = %d, want 4", resp.Report.Plants[0].Index)
	}
	if resp.Report.Plants[0].InSeason != true {
		t.Errorf("Plants[0].InSeason = %v, want true", resp.Report.Plants[0].InSeason)
	}

	// Verify CollectedAt is set (non-zero timestamp)
	if resp.Report.CollectedAt == nil {
		t.Fatal("CollectedAt should not be nil")
	}
}

func TestGetAllPollenReports(t *testing.T) {
	now := time.Now()
	mockRepo := &MockReader{
		GetAllFunc: func(ctx context.Context) ([]repository.CacheDoc, error) {
			return []repository.CacheDoc{
				{
					LocationID:  "house-nick",
					LastUpdated: now,
					CurrentValue: repository.PollenSnapshot{
						CollectedAt:     now,
						OverallIndex:    4,
						OverallCategory: "High",
						DominantType:    "TREE",
					},
				},
				{
					LocationID:  "house-nita",
					LastUpdated: now,
					CurrentValue: repository.PollenSnapshot{
						CollectedAt:     now,
						OverallIndex:    1,
						OverallCategory: "Very Low",
						DominantType:    "GRASS",
					},
				},
			}, nil
		},
	}

	svc := service.NewPollenService(mockRepo)
	handler := NewGrpcHandler(svc)

	resp, err := handler.GetAllPollenReports(context.Background(), &pb.GetAllPollenReportsRequest{})

	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(resp.Reports) != 2 {
		t.Fatalf("expected 2 reports, got %d", len(resp.Reports))
	}
	if resp.Reports[0].LocationId != "house-nick" {
		t.Errorf("Reports[0].LocationId = %s, want house-nick", resp.Reports[0].LocationId)
	}
	if resp.Reports[0].OverallIndex != 4 {
		t.Errorf("Reports[0].OverallIndex = %d, want 4", resp.Reports[0].OverallIndex)
	}
	if resp.Reports[1].LocationId != "house-nita" {
		t.Errorf("Reports[1].LocationId = %s, want house-nita", resp.Reports[1].LocationId)
	}
	if resp.Reports[1].OverallIndex != 1 {
		t.Errorf("Reports[1].OverallIndex = %d, want 1", resp.Reports[1].OverallIndex)
	}
}

func TestGetPollenReport_Error(t *testing.T) {
	mockRepo := &MockReader{
		GetByIDFunc: func(ctx context.Context, id string) (*repository.CacheDoc, error) {
			return nil, fmt.Errorf("firestore unavailable")
		},
	}

	svc := service.NewPollenService(mockRepo)
	handler := NewGrpcHandler(svc)

	_, err := handler.GetPollenReport(context.Background(), &pb.GetPollenReportRequest{LocationId: "house-nick"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
