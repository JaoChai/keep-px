package router

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	keyfunc "github.com/MicahParks/keyfunc/v3"
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

func New(cfg *config.Config, logger *slog.Logger, pool *pgxpool.Pool, shutdownCtx context.Context) (http.Handler, CleanupFunc, error) {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.Recoverer)
	// Cloudflare sets CF-Connecting-IP to the real client address at the edge and
	// overwrites any client-supplied value, so it is the only header we trust.
	// The Worker strips X-Real-IP / X-Forwarded-For / True-Client-IP before
	// forwarding — see worker/client-ip.ts.
	r.Use(chimiddleware.ClientIPFromHeader("CF-Connecting-IP"))
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger(logger))
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))
	r.Use(chimiddleware.Compress(5))
	r.Use(middleware.MaxBodySize(1 << 20)) // 1 MB request body limit

	// Token encryption (optional in dev, required in production)
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
	if encryptor == nil && cfg.Env == "production" {
		logger.Error("TOKEN_ENCRYPTION_KEY is required in production")
		os.Exit(1)
	}

	// Repositories
	customerRepo := postgres.NewCustomerRepo(pool)
	pixelRepo := postgres.NewPixelRepo(pool, encryptor)
	eventRepo := postgres.NewEventRepo(pool)
	replaySessionRepo := postgres.NewReplaySessionRepo(pool)
	salePageRepo := postgres.NewSalePageRepo(pool)
	purchaseRepo := postgres.NewPurchaseRepo(pool)
	creditRepo := postgres.NewReplayCreditRepo(pool)
	subRepo := postgres.NewSubscriptionRepo(pool)
	usageRepo := postgres.NewEventUsageRepo(pool)
	webhookEventRepo := postgres.NewWebhookEventRepo(pool)

	// Admin repository
	adminRepo := postgres.NewAdminRepo(pool)

	// Facebook CAPI client
	capiClient := facebook.NewCAPIClient(cfg.FBGraphAPIURL)

	// โหลดกุญแจสาธารณะของ Neon Auth จาก JWKS endpoint
	// NoErrorReturnFirstHTTPReq=false: บังคับให้ fail ตอน startup ถ้าดึง JWKS ไม่สำเร็จ
	// ค่า default ของ library (true) จะกลืน error แรกแล้วขึ้นระบบเงียบๆ ทำให้ login พังทุกคน
	// จนกว่า background refresh goroutine จะสำเร็จ (นานถึง 1 ชม.) — ไม่ใช่พฤติกรรมที่ต้องการ
	noSwallow := false
	jwks, err := keyfunc.NewDefaultOverrideCtx(context.Background(),
		[]string{cfg.NeonAuthURL + "/.well-known/jwks.json"},
		keyfunc.Override{NoErrorReturnFirstHTTPReq: &noSwallow},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("โหลดกุญแจของ Neon Auth ไม่ได้: %w", err)
	}
	// issuer = ส่วน origin ของ NeonAuthURL (scheme://host) ตัด path ทิ้ง
	issuer := cfg.NeonAuthURL
	if i := strings.Index(cfg.NeonAuthURL, "://"); i >= 0 {
		schemeEnd := i + len("://")
		if j := strings.Index(cfg.NeonAuthURL[schemeEnd:], "/"); j >= 0 {
			issuer = cfg.NeonAuthURL[:schemeEnd+j]
		}
	}

	// Services
	authService := service.NewAuthService(customerRepo)
	billingService := service.NewBillingService(purchaseRepo, creditRepo, subRepo, customerRepo, webhookEventRepo, pool, cfg)
	quotaService := service.NewQuotaService(creditRepo, subRepo, usageRepo, pixelRepo, salePageRepo, customerRepo)
	pixelService := service.NewPixelService(pixelRepo, capiClient, logger, quotaService)
	eventService := service.NewEventService(eventRepo, pixelRepo, capiClient, logger, quotaService)
	replayService := service.NewReplayService(shutdownCtx, replaySessionRepo, eventRepo, pixelRepo, capiClient, logger, 5, quotaService)
	analyticsService := service.NewAnalyticsService(pool)
	salePageService := service.NewSalePageService(shutdownCtx, salePageRepo, customerRepo, pixelRepo, quotaService, cfg.SalePageCacheTTL)
	adminService := service.NewAdminService(adminRepo, customerRepo, creditRepo, replaySessionRepo, pool, logger)

	// Storage
	storageService := service.NewStorageService(cfg)

	// Handlers
	healthHandler := handler.NewHealthHandler(pool)
	authHandler := handler.NewAuthHandler(authService, logger, jwks.Keyfunc, issuer)
	pixelHandler := handler.NewPixelHandler(pixelService, logger)
	eventHandler := handler.NewEventHandler(eventService, logger)
	replayHandler := handler.NewReplayHandler(replayService, logger)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService, logger)
	salePageHandler := handler.NewSalePageHandler(salePageService, cfg.BaseURL, logger)
	uploadHandler := handler.NewUploadHandler(storageService)
	billingHandler := handler.NewBillingHandler(billingService, quotaService, cfg, logger)
	adminHandler := handler.NewAdminHandler(adminService, logger)

	// DB query timeout for all routes except Stripe webhook
	dbTimeout := middleware.DBTimeout(cfg.DBQueryTimeout)

	// Health check
	r.Get("/health", healthHandler.Health)
	r.Get("/ready", healthHandler.Ready)

	// Rate limiter with context for proper goroutine cleanup
	rateLimiter := middleware.RateLimitWithContext(shutdownCtx, cfg.RateLimitRPS)

	// Public sale page route (rate limited)
	r.Group(func(r chi.Router) {
		r.Use(rateLimiter)
		r.Use(dbTimeout)
		r.Get("/p/{slug}", salePageHandler.Serve)
	})

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(rateLimiter)

		// Stripe webhook (no auth, no DB timeout - Stripe retries can take longer)
		r.Post("/billing/webhook", billingHandler.Webhook)

		// All other API routes get DB query timeout
		r.Group(func(r chi.Router) {
			r.Use(dbTimeout)

			// Auth routes (public)
			r.Route("/auth", func(r chi.Router) {
				r.Post("/session", authHandler.Session)
			})

			// Event ingestion (API key auth)
			r.Route("/events", func(r chi.Router) {
				r.Use(middleware.APIKeyAuthWithContext(shutdownCtx, customerRepo))
				r.With(middleware.RateLimitByAPIKey(cfg.RateLimitAPIKeyRPS, cfg.RateLimitAPIKeyBurst)).Post("/ingest", eventHandler.Ingest)
			})

			// Dashboard routes (JWT auth)
			r.Group(func(r chi.Router) {
				r.Use(middleware.JWTAuth(jwks.Keyfunc, issuer, customerRepo.GetByAuthUserID))

				// Auth - get current user
				r.Get("/auth/me", authHandler.Me)
				r.Post("/auth/regenerate-api-key", authHandler.RegenerateAPIKey)

				// Pixel routes - Phase 3
				r.Route("/pixels", func(r chi.Router) {
					r.Get("/", pixelHandler.List)
					r.Post("/", pixelHandler.Create)
					r.Get("/{id}", pixelHandler.Get)
					r.Put("/{id}", pixelHandler.Update)
					r.Delete("/{id}", pixelHandler.Delete)
					r.Post("/{id}/test", pixelHandler.Test)
				})

				// Event log routes - Phase 4
				r.Get("/events", eventHandler.List)
				r.Get("/events/event-types", eventHandler.EventTypes)
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

				// Billing routes
				r.Route("/billing", func(r chi.Router) {
					r.Get("/", billingHandler.GetBillingOverview)
					r.Get("/quota", billingHandler.GetQuota)
					r.Post("/checkout", billingHandler.CreateCheckout)
					r.Put("/slots", billingHandler.UpdateSlots)
					r.Post("/portal", billingHandler.CreatePortalSession)
				})

				// Analytics routes
				r.Get("/analytics/overview", analyticsHandler.Overview)
				r.Get("/analytics/events", analyticsHandler.EventChart)

				// Upload
				r.Post("/uploads/image", uploadHandler.UploadImage)

				// Admin routes (JWT + AdminOnly)
				r.Route("/admin", func(r chi.Router) {
					r.Use(middleware.AdminOnly)

					r.Get("/customers", adminHandler.ListCustomers)
					r.Get("/customers/{id}", adminHandler.GetCustomerDetail)
					r.Put("/customers/{id}/plan", adminHandler.ChangePlan)
					r.Post("/customers/{id}/suspend", adminHandler.SuspendCustomer)
					r.Post("/customers/{id}/activate", adminHandler.ActivateCustomer)
					r.Post("/customers/{id}/credits", adminHandler.GrantCredits)

					r.Get("/analytics/overview", adminHandler.GetPlatformOverview)
					r.Get("/analytics/revenue", adminHandler.GetRevenueChart)
					r.Get("/analytics/growth", adminHandler.GetGrowthChart)

					r.Get("/purchases", adminHandler.ListPurchases)
					r.Get("/subscriptions", adminHandler.ListSubscriptions)
					r.Get("/credit-grants", adminHandler.ListCreditGrants)

					// F1: Sale Pages
					r.Get("/sale-pages", adminHandler.ListSalePages)
					r.Get("/sale-pages/{id}", adminHandler.GetSalePageDetail)
					r.Post("/sale-pages/{id}/disable", adminHandler.DisableSalePage)
					r.Post("/sale-pages/{id}/enable", adminHandler.EnableSalePage)
					r.Delete("/sale-pages/{id}", adminHandler.DeleteSalePage)

					// F2: Pixels
					r.Get("/pixels", adminHandler.ListPixels)
					r.Get("/pixels/{id}", adminHandler.GetPixelDetail)
					r.Post("/pixels/{id}/disable", adminHandler.DisablePixel)
					r.Post("/pixels/{id}/enable", adminHandler.EnablePixel)

					// F3: Replays
					r.Get("/replays", adminHandler.ListReplays)
					r.Get("/replays/{id}", adminHandler.GetReplayDetail)
					r.Post("/replays/{id}/cancel", adminHandler.CancelReplay)

					// F4: Events
					r.Get("/events/stats", adminHandler.GetEventStats)
					r.Get("/events", adminHandler.ListEvents)

					// F5: Audit Log
					r.Get("/audit-log", adminHandler.ListAuditLog)
				})

			})

		}) // end dbTimeout group
	})

	return r, func() { replayService.Shutdown() }, nil
}
