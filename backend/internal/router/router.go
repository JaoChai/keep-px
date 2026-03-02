package router

import (
	"context"
	"encoding/hex"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jaochai/pixlinks/backend/internal/config"
	"github.com/jaochai/pixlinks/backend/internal/crypto"
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

	// Token encryption (optional)
	var encryptor *crypto.TokenEncryptor
	if cfg.TokenEncryptionKey != "" {
		keyBytes, err := hex.DecodeString(cfg.TokenEncryptionKey)
		if err != nil {
			logger.Error("invalid TOKEN_ENCRYPTION_KEY (must be hex-encoded)", "error", err)
		} else {
			enc, err := crypto.NewTokenEncryptor(keyBytes)
			if err != nil {
				logger.Error("failed to create token encryptor", "error", err)
			} else {
				encryptor = enc
				logger.Info("token encryption enabled")
			}
		}
	}

	// Repositories
	customerRepo := postgres.NewCustomerRepo(pool)
	refreshTokenRepo := postgres.NewRefreshTokenRepo(pool)
	pixelRepo := postgres.NewPixelRepo(pool, encryptor)
	eventRepo := postgres.NewEventRepo(pool)
	replaySessionRepo := postgres.NewReplaySessionRepo(pool)
	salePageRepo := postgres.NewSalePageRepo(pool)
	notifRepo := postgres.NewNotificationRepo(pool)
	purchaseRepo := postgres.NewPurchaseRepo(pool)
	creditRepo := postgres.NewReplayCreditRepo(pool)
	subRepo := postgres.NewSubscriptionRepo(pool)
	usageRepo := postgres.NewEventUsageRepo(pool)
	webhookEventRepo := postgres.NewWebhookEventRepo(pool)

	// Facebook CAPI client
	capiClient := facebook.NewCAPIClient(cfg.FBGraphAPIURL)

	// Services
	authService := service.NewAuthService(customerRepo, refreshTokenRepo, cfg)
	billingService := service.NewBillingService(purchaseRepo, creditRepo, subRepo, customerRepo, webhookEventRepo, cfg)
	quotaService := service.NewQuotaService(creditRepo, subRepo, usageRepo, pixelRepo, salePageRepo)
	pixelService := service.NewPixelService(pixelRepo, capiClient, logger, quotaService)
	eventService := service.NewEventService(eventRepo, pixelRepo, capiClient, logger, quotaService)
	notifService := service.NewNotificationService(notifRepo)
	replayService := service.NewReplayService(shutdownCtx, replaySessionRepo, eventRepo, pixelRepo, capiClient, logger, 5, notifService, quotaService)
	analyticsService := service.NewAnalyticsService(pool)
	salePageService := service.NewSalePageService(salePageRepo, customerRepo, pixelRepo, quotaService)

	// Storage
	storageService := service.NewStorageService(cfg)

	// Handlers
	healthHandler := handler.NewHealthHandler(pool)
	authHandler := handler.NewAuthHandler(authService, logger)
	pixelHandler := handler.NewPixelHandler(pixelService)
	eventHandler := handler.NewEventHandler(eventService)
	replayHandler := handler.NewReplayHandler(replayService)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService)
	notifHandler := handler.NewNotificationHandler(notifService)
	salePageHandler := handler.NewSalePageHandler(salePageService, cfg.BaseURL, logger)
	uploadHandler := handler.NewUploadHandler(storageService)
	billingHandler := handler.NewBillingHandler(billingService, quotaService, cfg, logger)

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

		// Stripe webhook (no auth - uses Stripe signature verification)
		r.Post("/billing/webhook", billingHandler.Webhook)

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
			r.Post("/auth/regenerate-api-key", authHandler.RegenerateAPIKey)

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

			// Billing routes
			r.Route("/billing", func(r chi.Router) {
				r.Get("/", billingHandler.GetBillingOverview)
				r.Get("/quota", billingHandler.GetQuota)
				r.Post("/checkout", billingHandler.CreateCheckout)
				r.Post("/portal", billingHandler.CreatePortalSession)
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
