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
		stats = append(stats, mapToProtoPressureStat(&docs[i]))
	}

	return &pb.GetAllPressureStatsResponse{Stats: stats}, nil
}

func (h *GrpcHandler) GetPressureStats(ctx context.Context, req *pb.GetPressureStatsRequest) (*pb.GetPressureStatsResponse, error) {
	doc, err := h.svc.GetStatsByID(ctx, req.LocationId)
	if err != nil {
		slog.Error("Failed to retrieve pressure data.", "error", err)
		return nil, status.Errorf(codes.Unknown, "Failed to retrieve pressure data: %v", err)
	}

	return &pb.GetPressureStatsResponse{Stat: mapToProtoPressureStat(doc)}, nil
}

func (h *GrpcHandler) GetLastWeather(ctx context.Context, req *pb.GetLastWeatherRequest) (*pb.GetLastWeatherResponse, error) {
	doc, err := h.svc.GetLastWeather(ctx, req.LocationId)
	if err != nil {
		slog.Error("Failed to retrieve last weather data.", "error", err)
		return nil, status.Errorf(codes.Unknown, "Failed to retrieve last weather data: %v", err)
	}
	return &pb.GetLastWeatherResponse{Weather: mapToProtoWeather(doc)}, nil
}

func (h *GrpcHandler) GetAllLastWeather(ctx context.Context, req *pb.GetAllLastWeatherRequest) (*pb.GetAllLastWeatherResponse, error) {
	docs, err := h.svc.GetAllLastWeather(ctx)
	if err != nil {
		slog.Error("Failed to retrieve last weather data.", "error", err)
		return nil, status.Errorf(codes.Unknown, "Failed to retrieve last weather data: %v", err)
	}
	var weathers []*pb.Weather
	for i := range docs {
		weathers = append(weathers, mapToProtoWeather(&docs[i]))
	}
	return &pb.GetAllLastWeatherResponse{Weather: weathers}, nil
}

// mapToProto converts the internal repository model to the gRPC message
func mapToProtoPressureStat(doc *repository.PressureCacheDoc) *pb.PressureStat {
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

func mapToProtoWeather(doc *repository.WeatherCacheDoc) *pb.Weather {
	return &pb.Weather{
		LocationId:           doc.LocationID,
		LastUpdated:          timestamppb.New(doc.CurrentValue.Timestamp),
		TempC:                doc.CurrentValue.TempC,
		TempF:                doc.CurrentValue.TempF,
		TempFeelC:            doc.CurrentValue.TempFeelC,
		TempFeelF:            doc.CurrentValue.TempFeelF,
		HumidityPercent:      int32(doc.CurrentValue.HumidityPercent),
		PressureMb:           doc.CurrentValue.PressureMb,
		PrecipitationPercent: int32(doc.CurrentValue.PrecipitationPercent),
	}
}
