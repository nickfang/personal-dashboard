package weather_provider

import (
	"context"
	"log/slog"
	"net"
	"os"

	weather_provider "github.com/nickfang/personal-dashboard/services/gen/go/weather-provider/v1"
	"google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type server struct {
	weather_provider.UnimplementedPressureStatsServiceServer
}

func (s *server) GetAllPressureStats(ctx context.Context, req *weather_provider.GetAllPressureStatsRequest) (*weather_provider.GetAllPressureStatsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetAllPressureStats not implemented")
}

func (s *server) GetPressureStats(ctx context.Context, req *weather_provider.GetPressureStatsRequest) (*weather_provider.GetPressureStatsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPressureStats not implemented")
}

func main() {
	// Setup Structured Logging
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	if os.Getenv("DEBUG") == "true" {
		opts.Level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, opts))
	slog.SetDefault(logger)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		slog.Error("failed to listen: %v", err)
		os.Exit(1)
	}
	defer lis.Close()

	grpcServer := grpc.NewServer()
	weather_provider.RegisterPressureStatsServiceServer(grpcServer, &server{})
	slog.Info("Server listening on port 50051")
	if err := grpcServer.Serve(lis); err != nil {
		slog.Error("failed to serve: %v", err)
	}

}
