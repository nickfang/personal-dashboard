package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/nickfang/personal-dashboard/services/dashboard-api/internal/app"
	"github.com/nickfang/personal-dashboard/services/dashboard-api/internal/clients"
	"github.com/nickfang/personal-dashboard/services/dashboard-api/internal/handlers"
)

func main() {
	// 1. Setup Logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

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

	// 3. Initialize Clients
	weatherClient, err := clients.NewWeatherClient(weatherAddr)
	if err != nil {
		slog.Error("Failed to initialize weather client", "error", err)
		os.Exit(1)
	}
	defer weatherClient.Close()

	// 4. Initialize Handlers
	dashboardHandler := handlers.NewDashboardHandler(weatherClient)

	// 5. Initialize Router
	router := app.NewRouter(dashboardHandler)

	// 6. Start Server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	slog.Info("Dashboard API starting", "port", port, "weather_addr", weatherAddr)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
