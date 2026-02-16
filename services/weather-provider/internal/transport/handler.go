package transport

import (
	"context"
	"log/slog"

	pb "github.com/nickfang/personal-dashboard/services/weather-provider/internal/gen/go/weather-provider/v1"
	"github.com/nickfang/personal-dashboard/services/weather-provider/internal/repository"
	"github.com/nickfang/personal-dashboard/services/weather-provider/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GrpcHandler struct {
	pb.UnimplementedPressureStatsServiceServer
	svc *service.WeatherService
}

func NewGrpcHandler(svc *service.WeatherService) *GrpcHandler {
	return &GrpcHandler{svc: svc}
}

func (h *GrpcHandler) GetAllPressureStats(ctx context.Context, req *pb.GetAllPressureStatsRequest) (*pb.GetAllPressureStatsResponse, error) {
	docs, err := h.svc.GetAllStats(ctx)
	if err != nil {
		slog.Error("Failed to retrieve pressure data.", "error", err)
		return nil, status.Errorf(codes.Unknown, "Failed to retrieve pressure data: %v", err)
	}

	var stats []*pb.PressureStat
	for i := range docs {
		stats = append(stats, mapToProto(&docs[i]))
	}

	return &pb.GetAllPressureStatsResponse{Stats: stats}, nil
}

func (h *GrpcHandler) GetPressureStats(ctx context.Context, req *pb.GetPressureStatsRequest) (*pb.GetPressureStatsResponse, error) {
	doc, err := h.svc.GetStatsByID(ctx, req.LocationId)
	if err != nil {
		slog.Error("Failed to retrieve pressure data.", "error", err)
		return nil, status.Errorf(codes.Unknown, "Failed to retrieve pressure data: %v", err)
	}

	return &pb.GetPressureStatsResponse{Stat: mapToProto(doc)}, nil
}

// mapToProto converts the internal repository model to the gRPC message
func mapToProto(doc *repository.CacheDoc) *pb.PressureStat {
	stat := &pb.PressureStat{
		LocationId:  doc.LocationID,
		LastUpdated: timestamppb.New(doc.LastUpdated),
		Trend:       doc.Analysis.Trend,
	}

	// Safely map pointers (Deltas)
	if doc.Analysis.Delta1h != nil {
		stat.Delta_1H = *doc.Analysis.Delta1h
	}
	if doc.Analysis.Delta3h != nil {
		stat.Delta_3H = *doc.Analysis.Delta3h
	}
	if doc.Analysis.Delta6h != nil {
		stat.Delta_6H = *doc.Analysis.Delta6h
	}
	if doc.Analysis.Delta12h != nil {
		stat.Delta_12H = *doc.Analysis.Delta12h
	}
	if doc.Analysis.Delta24h != nil {
		stat.Delta_24H = *doc.Analysis.Delta24h
	}

	return stat
}
