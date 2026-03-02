package router

import (
	"context"
	"log/slog"
	"net/http"

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

type CleanupFunc func()

func New(cfg *config.Config, logger *slog.Logger, pool *pgxpool.Pool, shutdownCtx context.Context) (http.Handler, CleanupFunc) {
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
	replaySessionRepo := postgres.NewReplaySessionRepo(pool)
	salePageRepo := postgres.NewSalePageRepo(pool)
	notifRepo := postgres.NewNotificationRepo(pool)
	// Facebook CAPI client
	capiClient := facebook.NewCAPIClient(cfg.FBGraphAPIURL)

	// Services
	authService := service.NewAuthService(customerRepo, refreshTokenRepo, cfg)
	pixelService := service.NewPixelService(pixelRepo, capiClient, logger)
	eventService := service.NewEventService(eventRepo, pixelRepo, capiClient, logger)
	notifService := service.NewNotificationService(notifRepo)
	replayService := service.NewReplayService(shutdownCtx, replaySessionRepo, eventRepo, pixelRepo, capiClient, logger, 5, notifService)
	analyticsService := service.NewAnalyticsService(pool)
	salePageService := service.NewSalePageService(salePageRepo, customerRepo, pixelRepo)

	// Storage
	storageService := service.NewStorageService(cfg)

	// Handlers
	healthHandler := handler.NewHealthHandler()
	authHandler := handler.NewAuthHandler(authService, logger)
	pixelHandler := handler.NewPixelHandler(pixelService)
	eventHandler := handler.NewEventHandler(eventService)
	replayHandler := handler.NewReplayHandler(replayService)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService)
	notifHandler := handler.NewNotificationHandler(notifService)
	salePageHandler := handler.NewSalePageHandler(salePageService, cfg.BaseURL, logger)
	uploadHandler := handler.NewUploadHandler(storageService)

	// Health check
	r.Get("/health", healthHandler.Health)

	// Public sale page route (rate limited)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RateLimit(cfg.RateLimitRPS))
		r.Get("/p/{slug}", salePageHandler.Serve)
	})

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.RateLimit(cfg.RateLimitRPS))
		// Auth routes (public)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/google", authHandler.GoogleAuth)
			r.Post("/refresh", authHandler.Refresh)
		})

		// Event ingestion (API key auth)
		r.Route("/events", func(r chi.Router) {
			r.Use(middleware.APIKeyAuth(customerRepo))
			r.Post("/ingest", eventHandler.Ingest)
		})

		// Dashboard routes (JWT auth)
		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuth(cfg.JWTSecret))

			// Auth - get current user
			r.Get("/auth/me", authHandler.Me)

			// Pixel routes - Phase 3
			r.Route("/pixels", func(r chi.Router) {
				r.Get("/", pixelHandler.List)
				r.Post("/", pixelHandler.Create)
				r.Put("/{id}", pixelHandler.Update)
				r.Delete("/{id}", pixelHandler.Delete)
				r.Post("/{id}/test", pixelHandler.Test)
			})

			// Event log routes - Phase 4
			r.Get("/events", eventHandler.List)
			r.Get("/events/recent", eventHandler.ListRecent)
			r.Get("/events/{id}", eventHandler.GetByID)

			// Replay routes - Phase 6
			r.Route("/replays", func(r chi.Router) {
				r.Post("/", replayHandler.Create)
				r.Get("/", replayHandler.List)
				r.Post("/preview", replayHandler.Preview)
				r.Get("/event-types", replayHandler.EventTypes)
				r.Get("/{id}", replayHandler.GetByID)
				r.Post("/{id}/cancel", replayHandler.Cancel)
				r.Post("/{id}/retry", replayHandler.Retry)
			})

			// Sale pages routes
			r.Route("/sale-pages", func(r chi.Router) {
				r.Get("/", salePageHandler.List)
				r.Post("/", salePageHandler.Create)
				r.Put("/{id}", salePageHandler.Update)
				r.Delete("/{id}", salePageHandler.Delete)
				r.Get("/{id}/preview", salePageHandler.Preview)
			})

			// Notification routes
			r.Route("/notifications", func(r chi.Router) {
				r.Get("/", notifHandler.List)
				r.Get("/unread-count", notifHandler.UnreadCount)
				r.Post("/{id}/read", notifHandler.MarkRead)
				r.Post("/read-all", notifHandler.MarkAllRead)
			})

			// Analytics routes
			r.Get("/analytics/overview", analyticsHandler.Overview)
			r.Get("/analytics/events", analyticsHandler.EventChart)

			// Upload
			r.Post("/uploads/image", uploadHandler.UploadImage)

		})
	})

	return r, func() { replayService.Shutdown() }
}
