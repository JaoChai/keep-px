package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jaochai/pixlinks/backend/internal/config"
	"github.com/jaochai/pixlinks/backend/internal/crypto"
	"github.com/jaochai/pixlinks/backend/internal/repository/postgres"
	"github.com/jaochai/pixlinks/backend/internal/router"
	"github.com/jaochai/pixlinks/backend/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if cfg.Env == "development" {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}

	// Database connection pool
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to parse database URL", "error", err)
		os.Exit(1)
	}
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = 1 * time.Hour
	poolConfig.MaxConnIdleTime = 15 * time.Minute
	poolConfig.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		logger.Error("failed to create connection pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		logger.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	logger.Info("connected to database")

	// Run database migrations
	if err := runMigrations(cfg.DatabaseURL, logger); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Recover orphaned replay sessions from previous crash/restart
	replaySessionRepo := postgres.NewReplaySessionRepo(pool)
	recovered, err := replaySessionRepo.RecoverOrphanedSessions(context.Background())
	if err != nil {
		logger.Error("failed to recover orphaned replay sessions", "error", err)
	} else if recovered > 0 {
		logger.Warn("recovered orphaned replay sessions", "count", recovered)
	}

	// Migrate plaintext tokens to encrypted (one-time, idempotent)
	if cfg.TokenEncryptionKey != "" {
		keyBytes, err := hex.DecodeString(cfg.TokenEncryptionKey)
		if err != nil {
			logger.Error("invalid TOKEN_ENCRYPTION_KEY for migration", "error", err)
		} else {
			enc, err := crypto.NewTokenEncryptor(keyBytes)
			if err != nil {
				logger.Error("failed to create encryptor for migration", "error", err)
			} else {
				migrateTokens(context.Background(), pool, enc, logger)
			}
		}
	}

	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	handler, cleanupReplay := router.New(cfg, logger, pool, shutdownCtx)

	// Start event retention cleanup service (daily, deletes events older than 365 days)
	eventRepo := postgres.NewEventRepo(pool)
	cleanupService := service.NewCleanupService(eventRepo, logger)
	cleanupService.Start(shutdownCtx)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("server starting", "port", cfg.Port, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// Signal background goroutines to stop
	shutdownCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced shutdown", "error", err)
	}

	// Wait for background replay goroutines to finish
	cleanupReplay()

	// Wait for cleanup service to finish
	cleanupService.Stop()

	logger.Info("server stopped")
}

func runMigrations(databaseURL string, logger *slog.Logger) error {
	logger.Info("running database migrations...")

	m, err := migrate.New("file://db/migrations", databaseURL)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Info("migrations: already up to date")
			return nil
		}
		// Check if dirty — try to recover
		version, dirty, verErr := m.Version()
		if verErr == nil && dirty {
			logger.Warn("dirty migration detected, forcing version", "version", version)
			if forceErr := m.Force(int(version)); forceErr != nil {
				return fmt.Errorf("force migration version: %w", forceErr)
			}
			// Retry Up after force
			if retryErr := m.Up(); retryErr != nil && !errors.Is(retryErr, migrate.ErrNoChange) {
				return fmt.Errorf("run migrations after force: %w", retryErr)
			}
			logger.Info("migrations: recovered from dirty state")
			return nil
		}
		return fmt.Errorf("run migrations: %w", err)
	}

	logger.Info("migrations: applied successfully")
	return nil
}

func migrateTokens(ctx context.Context, pool *pgxpool.Pool, encryptor *crypto.TokenEncryptor, logger *slog.Logger) {
	rows, err := pool.Query(ctx, "SELECT id, fb_access_token FROM pixels WHERE fb_access_token != ''")
	if err != nil {
		logger.Error("token migration: failed to query pixels", "error", err)
		return
	}
	defer rows.Close()

	var migrated int
	for rows.Next() {
		var id, token string
		if err := rows.Scan(&id, &token); err != nil {
			logger.Error("token migration: failed to scan row", "error", err)
			continue
		}
		if crypto.IsEncrypted(token) {
			continue // already encrypted
		}
		encrypted, err := encryptor.Encrypt(token)
		if err != nil {
			logger.Error("token migration: failed to encrypt", "pixel_id", id, "error", err)
			continue
		}
		if _, err := pool.Exec(ctx, "UPDATE pixels SET fb_access_token = $1 WHERE id = $2", encrypted, id); err != nil {
			logger.Error("token migration: failed to update", "pixel_id", id, "error", err)
			continue
		}
		migrated++
	}
	if err := rows.Err(); err != nil {
		logger.Error("token migration: rows iteration error", "error", err)
	}
	if migrated > 0 {
		logger.Info("token migration: encrypted plaintext tokens", "count", migrated)
	} else {
		logger.Info("token migration: no plaintext tokens to migrate")
	}
}
