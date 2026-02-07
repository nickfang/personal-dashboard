package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	"github.com/joho/godotenv"
	pb "github.com/nickfang/personal-dashboard/services/weather-provider/gen/v1"
	"github.com/nickfang/personal-dashboard/services/weather-provider/internal/repository"
	"github.com/nickfang/personal-dashboard/services/weather-provider/internal/service"
	"github.com/nickfang/personal-dashboard/services/weather-provider/internal/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// 1. Setup Logging
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	if os.Getenv("DEBUG") == "true" {
		opts.Level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, opts))
	slog.SetDefault(logger)

	// Load .env file if it exists (local development)
	if err := godotenv.Load(); err != nil {
		slog.Debug("No .env file found, using system environment variables", "error", err)
	}

	// 2. Load Config
	projectID := os.Getenv("GCP_PROJECT_ID")
	port := os.Getenv("PORT")
	if port == "" {
		port = "50051"
	}
	if projectID == "" {
		slog.Error("Missing required env var: GCP_PROJECT_ID")
		os.Exit(1)
	}

	ctx := context.Background()

	// 3. Initialize Layers
	repo, err := repository.NewFirestoreRepository(ctx, projectID)
	if err != nil {
		slog.Error("Failed to initialize repository", "error", err)
		os.Exit(1)
	}
	defer repo.Close()

	svc := service.NewWeatherService(repo)
	handler := transport.NewGrpcHandler(svc)

	// 4. Start gRPC Server
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		slog.Error("Failed to listen", "port", port, "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterPressureStatsServiceServer(grpcServer, handler)
	
	// Enable reflection for debugging (e.g., using grpcurl)
	reflection.Register(grpcServer)

	slog.Info("Weather Provider Server listening", "port", port)
	if err := grpcServer.Serve(lis); err != nil {
		slog.Error("Failed to serve gRPC", "error", err)
		os.Exit(1)
	}
}
