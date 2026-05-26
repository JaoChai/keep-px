//go:build integration

package postgres

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newReplayCredit(customerID string) *domain.ReplayCredit {
	return &domain.ReplayCredit{
		CustomerID:         customerID,
		PackType:           "pack_10",
		TotalReplays:       10,
		UsedReplays:        0,
		MaxEventsPerReplay: 0,
		ExpiresAt:          time.Now().Add(30 * 24 * time.Hour),
	}
}

func TestReplayCreditRepo_CreateAndGetByID(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	ctx := context.Background()

	repo := NewReplayCreditRepo(pool)
	customer := testutil.CreateTestCustomer(t, pool)

	c := newReplayCredit(customer.ID)
	require.NoError(t, repo.Create(ctx, c))
	assert.NotEmpty(t, c.ID, "ID should be populated")
	assert.False(t, c.CreatedAt.IsZero(), "CreatedAt should be populated")

	got, err := repo.GetByID(ctx, c.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, c.ID, got.ID)
	assert.Equal(t, customer.ID, got.CustomerID)
	assert.Equal(t, "pack_10", got.PackType)
	assert.Equal(t, 10, got.TotalReplays)
	assert.Equal(t, 0, got.UsedReplays)
	assert.Equal(t, 0, got.MaxEventsPerReplay)
	assert.WithinDuration(t, c.ExpiresAt, got.ExpiresAt, time.Second)
}

func TestReplayCreditRepo_GetByID_NotFound(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	ctx := context.Background()

	repo := NewReplayCreditRepo(pool)
	got, err := repo.GetByID(ctx, "00000000-0000-0000-0000-000000000000")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestReplayCreditRepo_GetActiveByCustomerID_FiltersExhaustedAndExpired(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	ctx := context.Background()

	repo := NewReplayCreditRepo(pool)
	customer := testutil.CreateTestCustomer(t, pool)

	// A: active — still has replays and not expired
	active := newReplayCredit(customer.ID)
	active.PackType = "active"
	active.ExpiresAt = time.Now().Add(30 * 24 * time.Hour)
	require.NoError(t, repo.Create(ctx, active))

	// B: exhausted — used == total
	exhausted := newReplayCredit(customer.ID)
	exhausted.PackType = "exhausted"
	exhausted.TotalReplays = 5
	exhausted.UsedReplays = 5
	require.NoError(t, repo.Create(ctx, exhausted))

	// C: expired — expires_at in the past
	expired := newReplayCredit(customer.ID)
	expired.PackType = "expired"
	expired.ExpiresAt = time.Now().Add(-24 * time.Hour)
	require.NoError(t, repo.Create(ctx, expired))

	credits, err := repo.GetActiveByCustomerID(ctx, customer.ID)
	require.NoError(t, err)
	require.Len(t, credits, 1)
	assert.Equal(t, active.ID, credits[0].ID)
	assert.Equal(t, "active", credits[0].PackType)
}

func TestReplayCreditRepo_GetActiveByCustomerID_OrderedByExpiresAtASC(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	ctx := context.Background()

	repo := NewReplayCreditRepo(pool)
	customer := testutil.CreateTestCustomer(t, pool)

	later := newReplayCredit(customer.ID)
	later.PackType = "later"
	later.ExpiresAt = time.Now().Add(60 * 24 * time.Hour)
	require.NoError(t, repo.Create(ctx, later))

	sooner := newReplayCredit(customer.ID)
	sooner.PackType = "sooner"
	sooner.ExpiresAt = time.Now().Add(30 * 24 * time.Hour)
	require.NoError(t, repo.Create(ctx, sooner))

	credits, err := repo.GetActiveByCustomerID(ctx, customer.ID)
	require.NoError(t, err)
	require.Len(t, credits, 2)
	assert.Equal(t, sooner.ID, credits[0].ID, "earliest expiry should be first")
	assert.Equal(t, later.ID, credits[1].ID)
}

func TestReplayCreditRepo_IncrementUsed_Normal(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	ctx := context.Background()

	repo := NewReplayCreditRepo(pool)
	customer := testutil.CreateTestCustomer(t, pool)

	c := newReplayCredit(customer.ID)
	require.NoError(t, repo.Create(ctx, c))

	require.NoError(t, repo.IncrementUsed(ctx, c.ID))

	got, err := repo.GetByID(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, got.UsedReplays)
}

func TestReplayCreditRepo_IncrementUsed_Exhausted(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	ctx := context.Background()

	repo := NewReplayCreditRepo(pool)
	customer := testutil.CreateTestCustomer(t, pool)

	c := newReplayCredit(customer.ID)
	c.TotalReplays = 3
	c.UsedReplays = 3
	require.NoError(t, repo.Create(ctx, c))

	err := repo.IncrementUsed(ctx, c.ID)
	assert.ErrorIs(t, err, pgx.ErrNoRows)

	got, err := repo.GetByID(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, got.UsedReplays, "used should not change on exhausted increment")
}

func TestReplayCreditRepo_IncrementUsed_Unlimited(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	ctx := context.Background()

	repo := NewReplayCreditRepo(pool)
	customer := testutil.CreateTestCustomer(t, pool)

	c := newReplayCredit(customer.ID)
	c.TotalReplays = -1 // unlimited
	c.UsedReplays = 1000
	require.NoError(t, repo.Create(ctx, c))

	// Should be able to increment many times without hitting a limit.
	for i := 0; i < 5; i++ {
		require.NoError(t, repo.IncrementUsed(ctx, c.ID))
	}

	got, err := repo.GetByID(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, 1005, got.UsedReplays)
}

func TestReplayCreditRepo_RefundCredit(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	ctx := context.Background()

	repo := NewReplayCreditRepo(pool)
	customer := testutil.CreateTestCustomer(t, pool)

	c := newReplayCredit(customer.ID)
	c.UsedReplays = 3
	require.NoError(t, repo.Create(ctx, c))

	require.NoError(t, repo.RefundCredit(ctx, c.ID))
	got, err := repo.GetByID(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, got.UsedReplays)
}

func TestReplayCreditRepo_RefundCredit_ZeroUsed(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	ctx := context.Background()

	repo := NewReplayCreditRepo(pool)
	customer := testutil.CreateTestCustomer(t, pool)

	c := newReplayCredit(customer.ID)
	c.UsedReplays = 0
	require.NoError(t, repo.Create(ctx, c))

	err := repo.RefundCredit(ctx, c.ID)
	assert.ErrorIs(t, err, pgx.ErrNoRows)

	got, err := repo.GetByID(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, got.UsedReplays)
}

func TestReplayCreditRepo_ConsumeOneCredit_HappyPath(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	ctx := context.Background()

	repo := NewReplayCreditRepo(pool)
	customer := testutil.CreateTestCustomer(t, pool)

	c := newReplayCredit(customer.ID)
	require.NoError(t, repo.Create(ctx, c))

	consumed, err := repo.ConsumeOneCredit(ctx, customer.ID, 50)
	require.NoError(t, err)
	require.NotNil(t, consumed)
	assert.Equal(t, c.ID, consumed.ID)
	assert.Equal(t, 1, consumed.UsedReplays)

	// Verify DB matches the returned value.
	got, err := repo.GetByID(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, got.UsedReplays)
}

func TestReplayCreditRepo_ConsumeOneCredit_NoCredits(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	ctx := context.Background()

	repo := NewReplayCreditRepo(pool)
	customer := testutil.CreateTestCustomer(t, pool)

	consumed, err := repo.ConsumeOneCredit(ctx, customer.ID, 50)
	require.NoError(t, err)
	assert.Nil(t, consumed)
}

func TestReplayCreditRepo_ConsumeOneCredit_RespectsMaxEventsPerReplay(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	ctx := context.Background()

	repo := NewReplayCreditRepo(pool)
	customer := testutil.CreateTestCustomer(t, pool)

	c := newReplayCredit(customer.ID)
	c.MaxEventsPerReplay = 100
	require.NoError(t, repo.Create(ctx, c))

	// Asking for more than the credit allows -> not eligible.
	consumed, err := repo.ConsumeOneCredit(ctx, customer.ID, 200)
	require.NoError(t, err)
	assert.Nil(t, consumed, "credit with max=100 should not satisfy request for 200")

	// Confirm used was not incremented.
	got, err := repo.GetByID(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, got.UsedReplays)

	// Within limit -> should succeed.
	consumed, err = repo.ConsumeOneCredit(ctx, customer.ID, 50)
	require.NoError(t, err)
	require.NotNil(t, consumed)
	assert.Equal(t, c.ID, consumed.ID)
	assert.Equal(t, 1, consumed.UsedReplays)
}

func TestReplayCreditRepo_ConsumeOneCredit_PicksEarliestExpiry(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	ctx := context.Background()

	repo := NewReplayCreditRepo(pool)
	customer := testutil.CreateTestCustomer(t, pool)

	later := newReplayCredit(customer.ID)
	later.PackType = "later"
	later.ExpiresAt = time.Now().Add(60 * 24 * time.Hour)
	require.NoError(t, repo.Create(ctx, later))

	sooner := newReplayCredit(customer.ID)
	sooner.PackType = "sooner"
	sooner.ExpiresAt = time.Now().Add(30 * 24 * time.Hour)
	require.NoError(t, repo.Create(ctx, sooner))

	consumed, err := repo.ConsumeOneCredit(ctx, customer.ID, 10)
	require.NoError(t, err)
	require.NotNil(t, consumed)
	assert.Equal(t, sooner.ID, consumed.ID, "should consume the credit expiring soonest")

	// Verify the other credit was not touched.
	gotLater, err := repo.GetByID(ctx, later.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, gotLater.UsedReplays)
}

func TestReplayCreditRepo_ConsumeOneCredit_ConcurrentSafety(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	ctx := context.Background()

	repo := NewReplayCreditRepo(pool)
	customer := testutil.CreateTestCustomer(t, pool)

	const totalReplays = 10
	const goroutines = 20

	c := newReplayCredit(customer.ID)
	c.TotalReplays = totalReplays
	require.NoError(t, repo.Create(ctx, c))

	var (
		wg           sync.WaitGroup
		successCount int64
		nilCount     int64
	)

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			consumed, err := repo.ConsumeOneCredit(ctx, customer.ID, 10)
			if err != nil {
				t.Errorf("ConsumeOneCredit returned error: %v", err)
				return
			}
			if consumed != nil {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&nilCount, 1)
			}
		}()
	}
	wg.Wait()

	// FOR UPDATE SKIP LOCKED: when one goroutine holds the row lock, others skip
	// and return immediately. Each goroutine attempts ONCE — there is no retry loop.
	// So successCount may be anywhere from 1 to totalReplays depending on scheduling.
	// The invariants we MUST verify:
	//   1. successCount + nilCount == goroutines (no goroutine lost or errored)
	//   2. successCount <= totalReplays (never over-consume)
	//   3. DB used_replays == successCount (no lost updates)
	successes := atomic.LoadInt64(&successCount)
	nils := atomic.LoadInt64(&nilCount)
	assert.Equal(t, int64(goroutines), successes+nils, "every goroutine accounted for")
	assert.LessOrEqual(t, successes, int64(totalReplays), "must never exceed total_replays")
	assert.Positive(t, successes, "at least one goroutine should succeed")

	got, err := repo.GetByID(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, int(successes), got.UsedReplays,
		"DB used_replays must equal observed success count (no lost updates)")
	assert.LessOrEqual(t, got.UsedReplays, totalReplays,
		"used_replays must not exceed total_replays under concurrency")
}
