package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/jaochai/pixlinks/backend/internal/repository"
)

const (
	cleanupInterval   = 24 * time.Hour
	cleanupStartDelay = 5 * time.Minute
	cleanupBatchSize  = 10000
	cleanupBatchPause = 200 * time.Millisecond
)

// CleanupService periodically deletes old events that exceed the retention period.
type CleanupService struct {
	eventRepo repository.EventRepository
	logger    *slog.Logger
	cancel    context.CancelFunc
	done      chan struct{}
}

func NewCleanupService(eventRepo repository.EventRepository, logger *slog.Logger) *CleanupService {
	return &CleanupService{
		eventRepo: eventRepo,
		logger:    logger,
		done:      make(chan struct{}),
	}
}

// Start begins the background cleanup loop. It runs once immediately, then
// repeats on a daily ticker. The loop respects context cancellation.
func (s *CleanupService) Start(ctx context.Context) {
	ctx, s.cancel = context.WithCancel(ctx)
	go s.run(ctx)
}

// Stop signals the cleanup loop to stop and waits for it to finish.
func (s *CleanupService) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	<-s.done
}

func (s *CleanupService) run(ctx context.Context) {
	defer close(s.done)

	// Delay first run to let the server fully start
	select {
	case <-time.After(cleanupStartDelay):
	case <-ctx.Done():
		return
	}
	s.cleanup(ctx)

	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanup(ctx)
		case <-ctx.Done():
			s.logger.Info("cleanup service stopped")
			return
		}
	}
}

func (s *CleanupService) cleanup(ctx context.Context) {
	var totalDeleted int64

	s.logger.Info("per-plan event cleanup started")

	for {
		if ctx.Err() != nil {
			s.logger.Info("event cleanup interrupted by shutdown", "deleted_so_far", totalDeleted)
			return
		}

		deleted, err := s.eventRepo.DeleteExpiredByPlan(ctx, cleanupBatchSize)
		if err != nil {
			s.logger.Error("event cleanup batch failed", "error", err, "deleted_so_far", totalDeleted)
			return
		}

		totalDeleted += deleted

		// If fewer rows deleted than batch size, we're done
		if deleted < int64(cleanupBatchSize) {
			break
		}

		s.logger.Info("event cleanup batch completed", "batch_deleted", deleted, "total_deleted", totalDeleted)

		// Pause between batches to avoid saturating DB write IOPS
		select {
		case <-time.After(cleanupBatchPause):
		case <-ctx.Done():
			s.logger.Info("event cleanup interrupted by shutdown", "deleted_so_far", totalDeleted)
			return
		}
	}

	s.logger.Info("per-plan event cleanup completed", "total_deleted", totalDeleted)
}
