package router

import (
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jaochai/pixlinks/backend/internal/config"
	"github.com/jaochai/pixlinks/backend/internal/facebook"
	"github.com/jaochai/pixlinks/backend/internal/handler"
	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/repository/postgres"
	"github.com/jaochai/pixlinks/backend/internal/service"
)

func New(cfg *config.Config, logger *slog.Logger, pool *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger(logger))
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))

	// Repositories
	customerRepo := postgres.NewCustomerRepo(pool)
	refreshTokenRepo := postgres.NewRefreshTokenRepo(pool)
	pixelRepo := postgres.NewPixelRepo(pool)
	eventRepo := postgres.NewEventRepo(pool)
	eventRuleRepo := postgres.NewEventRuleRepo(pool)
	replaySessionRepo := postgres.NewReplaySessionRepo(pool)

	// Facebook CAPI client
	capiClient := facebook.NewCAPIClient(cfg.FBGraphAPIURL)

	// Services
	authService := service.NewAuthService(customerRepo, refreshTokenRepo, cfg)
	pixelService := service.NewPixelService(pixelRepo)
	eventService := service.NewEventService(eventRepo, pixelRepo, capiClient, logger)
	ruleService := service.NewRuleService(eventRuleRepo, pixelRepo)
	replayService := service.NewReplayService(replaySessionRepo, eventRepo, pixelRepo, capiClient, logger)
	analyticsService := service.NewAnalyticsService(pool)

	// Handlers
	healthHandler := handler.NewHealthHandler()
	authHandler := handler.NewAuthHandler(authService, logger)
	pixelHandler := handler.NewPixelHandler(pixelService)
	eventHandler := handler.NewEventHandler(eventService)
	ruleHandler := handler.NewRuleHandler(ruleService)
	replayHandler := handler.NewReplayHandler(replayService)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService)
	proxyHandler := handler.NewProxyHandler()

	// Health check
	r.Get("/health", healthHandler.Health)

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Auth routes (public)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
		})

		// SDK routes (API key auth)
		r.Route("/events", func(r chi.Router) {
			r.Use(middleware.APIKeyAuth(customerRepo))
			r.Post("/ingest", eventHandler.Ingest)
		})

		// Dashboard routes (JWT auth)
		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuth(cfg.JWTSecret))

			// Pixel routes - Phase 3
			r.Route("/pixels", func(r chi.Router) {
				r.Get("/", pixelHandler.List)
				r.Post("/", pixelHandler.Create)
				r.Put("/{id}", pixelHandler.Update)
				r.Delete("/{id}", pixelHandler.Delete)
			})

			// Event log routes - Phase 4
			r.Get("/events", eventHandler.List)
			r.Get("/events/{id}", eventHandler.GetByID)

			// Event rules - Phase 5
			r.Route("/pixels/{pixelId}/rules", func(r chi.Router) {
				r.Get("/", ruleHandler.ListByPixel)
				r.Post("/", ruleHandler.Create)
			})
			r.Route("/rules", func(r chi.Router) {
				r.Put("/{id}", ruleHandler.Update)
				r.Delete("/{id}", ruleHandler.Delete)
			})

			// Replay routes - Phase 6
			r.Route("/replays", func(r chi.Router) {
				r.Post("/", replayHandler.Create)
				r.Get("/", replayHandler.List)
				r.Get("/{id}", replayHandler.GetByID)
			})

			// Analytics routes
			r.Get("/analytics/overview", analyticsHandler.Overview)
			r.Get("/analytics/events", analyticsHandler.EventChart)

			// Proxy (for visual event setup)
			r.Get("/proxy", proxyHandler.Proxy)
		})
	})

	// SPA static file serving
	spaHandler := spaFileServer("./public")
	r.NotFound(spaHandler)

	return r
}

// spaFileServer serves static files from the given directory.
// If the file exists, it serves it with immutable cache headers.
// Otherwise, it serves index.html with no-cache for client-side routing.
func spaFileServer(publicDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Clean the path and prevent directory traversal
		path := filepath.Clean(r.URL.Path)
		if path == "/" {
			path = "/index.html"
		}

		// Don't serve API paths as static files
		if strings.HasPrefix(path, "/api/") {
			http.NotFound(w, r)
			return
		}

		fullPath := filepath.Join(publicDir, path)

		// Check if the file exists and is not a directory
		info, err := os.Stat(fullPath)
		if err == nil && !info.IsDir() {
			// File exists — serve with immutable cache
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			http.ServeFile(w, r, fullPath)
			return
		}

		// File doesn't exist — serve index.html for SPA routing
		indexPath := filepath.Join(publicDir, "index.html")
		if _, err := os.Stat(indexPath); errors.Is(err, fs.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, indexPath)
	}
}
