package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jaochai/pixlinks/backend/internal/cloudflare"
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
	salePageRepo := postgres.NewSalePageRepo(pool)
	customDomainRepo := postgres.NewCustomDomainRepo(pool)

	// Facebook CAPI client
	capiClient := facebook.NewCAPIClient(cfg.FBGraphAPIURL)

	// Cloudflare client (optional)
	var cfClient *cloudflare.Client
	if cfg.CFAPIToken != "" {
		cfClient = cloudflare.NewClient(cfg.CFAPIToken, cfg.CFAccountID, cfg.CFZoneID, cfg.CFKVNamespaceID)
	}

	// Services
	authService := service.NewAuthService(customerRepo, refreshTokenRepo, cfg)
	pixelService := service.NewPixelService(pixelRepo)
	eventService := service.NewEventService(eventRepo, pixelRepo, capiClient, logger)
	ruleService := service.NewRuleService(eventRuleRepo, pixelRepo)
	replayService := service.NewReplayService(replaySessionRepo, eventRepo, pixelRepo, capiClient, logger)
	analyticsService := service.NewAnalyticsService(pool)
	salePageService := service.NewSalePageService(salePageRepo, customerRepo, pixelRepo)
	customDomainService := service.NewCustomDomainService(customDomainRepo, salePageRepo, cfClient, cfg.CFCNAMETarget)

	// Storage
	storageService := service.NewStorageService(cfg)

	// Handlers
	healthHandler := handler.NewHealthHandler()
	authHandler := handler.NewAuthHandler(authService, logger)
	pixelHandler := handler.NewPixelHandler(pixelService)
	eventHandler := handler.NewEventHandler(eventService)
	ruleHandler := handler.NewRuleHandler(ruleService)
	replayHandler := handler.NewReplayHandler(replayService)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService)
	proxyHandler := handler.NewProxyHandler()
	salePageHandler := handler.NewSalePageHandler(salePageService, cfg.BaseURL, logger)
	customDomainHandler := handler.NewCustomDomainHandler(customDomainService, cfg.CFCNAMETarget, logger)
	uploadHandler := handler.NewUploadHandler(storageService)

	// Health check
	r.Get("/health", healthHandler.Health)

	// Public sale page route (rate limited)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RateLimit(cfg.RateLimitRPS))
		r.Get("/p/{slug}", salePageHandler.Serve)
	})

	// Public SDK route (no auth, with caching)
	sdkHandler := handler.NewSDKHandler()
	r.Group(func(r chi.Router) {
		r.Use(middleware.RateLimit(cfg.RateLimitRPS))
		r.Get("/sdk/pixlinks.min.js", sdkHandler.ServeSDK)
	})

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.RateLimit(cfg.RateLimitRPS))
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

			// Auth - get current user
			r.Get("/auth/me", authHandler.Me)

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

			// Sale pages routes
			r.Route("/sale-pages", func(r chi.Router) {
				r.Get("/", salePageHandler.List)
				r.Post("/", salePageHandler.Create)
				r.Put("/{id}", salePageHandler.Update)
				r.Delete("/{id}", salePageHandler.Delete)
				r.Get("/{id}/preview", salePageHandler.Preview)
			})

			// Custom domain routes
			r.Route("/domains", func(r chi.Router) {
				r.Get("/", customDomainHandler.List)
				r.Post("/", customDomainHandler.Create)
				r.Get("/{id}", customDomainHandler.GetByID)
				r.Post("/{id}/verify", customDomainHandler.Verify)
				r.Delete("/{id}", customDomainHandler.Delete)
			})

			// Analytics routes
			r.Get("/analytics/overview", analyticsHandler.Overview)
			r.Get("/analytics/events", analyticsHandler.EventChart)

			// Upload
			r.Post("/uploads/image", uploadHandler.UploadImage)

			// Proxy (for visual event setup)
			r.Get("/proxy", proxyHandler.Proxy)
		})
	})

	return r
}
