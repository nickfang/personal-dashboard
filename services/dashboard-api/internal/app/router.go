package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nickfang/personal-dashboard/services/dashboard-api/internal/handlers"
	customMiddleware "github.com/nickfang/personal-dashboard/services/dashboard-api/internal/middleware"
)

func NewRouter(dashboardHandler *handlers.DashboardHandler) *chi.Mux {
	r := chi.NewRouter()

	// Global Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(customMiddleware.SlogLogger) // Use our custom slog middleware

	// API Routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/dashboard", dashboardHandler.GetDashboard)
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Dashboard API is healthy"))
	})

	return r
}
