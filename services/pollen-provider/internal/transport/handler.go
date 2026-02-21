package transport

import (
	"context"
	"log/slog"

	pb "github.com/nickfang/personal-dashboard/services/pollen-provider/internal/gen/go/pollen-provider/v1"
	"github.com/nickfang/personal-dashboard/services/pollen-provider/internal/repository"
	"github.com/nickfang/personal-dashboard/services/pollen-provider/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GrpcHandler struct {
	pb.UnimplementedPollenServiceServer
	svc *service.PollenService
}

func NewGrpcHandler(svc *service.PollenService) *GrpcHandler {
	return &GrpcHandler{svc: svc}
}

func (h *GrpcHandler) GetAllPollenReports(ctx context.Context, req *pb.GetAllPollenReportsRequest) (*pb.GetAllPollenReportsResponse, error) {
	docs, err := h.svc.GetAllReports(ctx)
	if err != nil {
		slog.Error("Failed to retrieve pollen data", "error", err)
		return nil, status.Errorf(codes.Unknown, "Failed to retrieve pollen data: %v", err)
	}

	var reports []*pb.PollenReport
	for i := range docs {
		reports = append(reports, mapToProto(&docs[i]))
	}
	return &pb.GetAllPollenReportsResponse{Reports: reports}, nil
}

func (h *GrpcHandler) GetPollenReport(ctx context.Context, req *pb.GetPollenReportRequest) (*pb.GetPollenReportResponse, error) {
	doc, err := h.svc.GetReportByID(ctx, req.LocationId)
	if err != nil {
		slog.Error("Failed to retrieve pollen data", "error", err)
		return nil, status.Errorf(codes.Unknown, "Failed to retrieve pollen data: %v", err)
	}

	return &pb.GetPollenReportResponse{Report: mapToProto(doc)}, nil
}

func mapToProto(doc *repository.CacheDoc) *pb.PollenReport {
	report := &pb.PollenReport{
		LocationId:      doc.LocationID,
		CollectedAt:     timestamppb.New(doc.CurrentValue.CollectedAt),
		OverallIndex:    int32(doc.CurrentValue.OverallIndex),
		OverallCategory: doc.CurrentValue.OverallCategory,
		DominantType:    doc.CurrentValue.DominantType,
	}

	for _, t := range doc.CurrentValue.Types {
		report.Types = append(report.Types, &pb.PollenType{
			Code:     t.Code,
			Index:    int32(t.Index),
			Category: t.Category,
			InSeason: t.InSeason,
		})
	}

	for _, p := range doc.CurrentValue.Plants {
		report.Plants = append(report.Plants, &pb.PollenPlant{
			Code:        p.Code,
			DisplayName: p.DisplayName,
			Index:       int32(p.Index),
			Category:    p.Category,
			InSeason:    p.InSeason,
		})
	}

	return report
}
