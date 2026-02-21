package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	pb "github.com/nickfang/personal-dashboard/services/pollen-provider/internal/gen/go/pollen-provider/v1"
	"github.com/nickfang/personal-dashboard/services/pollen-provider/internal/repository"
	"github.com/nickfang/personal-dashboard/services/pollen-provider/internal/service"
	"github.com/nickfang/personal-dashboard/services/pollen-provider/internal/transport"
	"github.com/nickfang/personal-dashboard/services/shared"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	// 1. Setup Logging
	shared.InitLogging()

	slog.Info("Pollen Provider starting", "version", "1.0.0", "debug", os.Getenv("DEBUG"))

	if err := godotenv.Load(); err != nil {
		slog.Debug("No .env file found, using system environment variables", "error", err)
	}

	// 2. Load Config
	projectID := os.Getenv("GCP_PROJECT_ID")
	port := os.Getenv("PORT")
	if port == "" {
		port = "50052"
	}
	if projectID == "" {
		slog.Error("Missing required env var: GCP_PROJECT_ID", "env", os.Environ())
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

	svc := service.NewPollenService(repo)
	handler := transport.NewGrpcHandler(svc)

	// 4. Start gRPC Server
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		slog.Error("Failed to listen", "port", port, "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterPollenServiceServer(grpcServer, handler)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	if os.Getenv("DEBUG") == "true" {
		reflection.Register(grpcServer)
	}

	// 5. Graceful Shutdown
	go func() {
		slog.Info("Pollen Provider Server listening", "port", port)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("Failed to serve gRPC", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server gracefully...")
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	grpcServer.GracefulStop()
	slog.Info("Server stopped")
}
