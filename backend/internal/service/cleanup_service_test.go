package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/mock"
)

func newTestCleanupService() (*CleanupService, *MockEventRepo) {
	repo := new(MockEventRepo)
	svc := NewCleanupService(repo, slog.Default())
	return svc, repo
}

func TestCleanup_CallsDeleteExpiredByRetention(t *testing.T) {
	svc, repo := newTestCleanupService()

	// Return fewer rows than batch size to indicate completion in one pass
	repo.On("DeleteExpiredByRetention", mock.Anything, cleanupBatchSize).
		Return(int64(42), nil).Once()

	svc.RunOnce(context.Background())

	repo.AssertExpectations(t)
}

func TestCleanup_BatchLoopStopsWhenDeletedLessThanBatchSize(t *testing.T) {
	svc, repo := newTestCleanupService()

	// First call returns a full batch, triggering another iteration
	repo.On("DeleteExpiredByRetention", mock.Anything, cleanupBatchSize).
		Return(int64(cleanupBatchSize), nil).Once()
	// Second call returns fewer than batch size, ending the loop
	repo.On("DeleteExpiredByRetention", mock.Anything, cleanupBatchSize).
		Return(int64(500), nil).Once()

	svc.RunOnce(context.Background())

	repo.AssertExpectations(t)
	repo.AssertNumberOfCalls(t, "DeleteExpiredByRetention", 2)
}

func TestCleanup_StopsOnContextCancellation(t *testing.T) {
	svc, repo := newTestCleanupService()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	svc.RunOnce(ctx)

	repo.AssertNotCalled(t, "DeleteExpiredByRetention")
}

func TestCleanup_StopsOnError(t *testing.T) {
	svc, repo := newTestCleanupService()

	repo.On("DeleteExpiredByRetention", mock.Anything, cleanupBatchSize).
		Return(int64(0), errors.New("db connection lost")).Once()

	svc.RunOnce(context.Background())

	repo.AssertExpectations(t)
	repo.AssertNumberOfCalls(t, "DeleteExpiredByRetention", 1)
}

func TestCleanup_MultipleBatchesDeleteCorrectTotal(t *testing.T) {
	svc, repo := newTestCleanupService()

	// Three full batches then a partial batch
	repo.On("DeleteExpiredByRetention", mock.Anything, cleanupBatchSize).
		Return(int64(cleanupBatchSize), nil).Times(3)
	repo.On("DeleteExpiredByRetention", mock.Anything, cleanupBatchSize).
		Return(int64(0), nil).Once()

	svc.RunOnce(context.Background())

	repo.AssertExpectations(t)
	repo.AssertNumberOfCalls(t, "DeleteExpiredByRetention", 4)
}
