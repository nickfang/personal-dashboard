package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/nickfang/personal-dashboard/services/dashboard-api/internal/app"
	"github.com/nickfang/personal-dashboard/services/dashboard-api/internal/clients"
	"github.com/nickfang/personal-dashboard/services/dashboard-api/internal/handlers"
	"github.com/nickfang/personal-dashboard/services/shared"
)

func main() {
	// 1. Setup Logging
	shared.InitLogging()

	// 2. Load Config
	if err := godotenv.Load(); err != nil {
		slog.Debug("No .env file found, utilizing environment variables")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	weatherAddr := os.Getenv("WEATHER_PROVIDER_ADDR")
	if weatherAddr == "" {
		weatherAddr = "localhost:50051"
	}

	pollenAddr := os.Getenv("POLLEN_PROVIDER_ADDR")
	if pollenAddr == "" {
		pollenAddr = "localhost:50052"
	}

	// 3. Initialize Clients
	weatherClient, err := clients.NewWeatherClient(context.Background(), weatherAddr)
	if err != nil {
		slog.Error("Failed to initialize weather client", "error", err)
		os.Exit(1)
	}
	pollenClient, err := clients.NewPollenClient(context.Background(), pollenAddr)
	if err != nil {
		slog.Error("Failed to initialize pollen client", "error", err)
		os.Exit(1)
	}
	defer weatherClient.Close()
	defer pollenClient.Close()

	// 4. Initialize Handlers
	dashboardHandler := handlers.NewDashboardHandler(weatherClient, pollenClient)

	// 5. Initialize Router
	router := app.NewRouter(dashboardHandler)

	// 6. Start Server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("Dashboard API starting", "port", port, "weather_addr", weatherAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server gracefully...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Shutdown error", "error", err)
		os.Exit(1)
	}
	slog.Info("Server stopped")
}
