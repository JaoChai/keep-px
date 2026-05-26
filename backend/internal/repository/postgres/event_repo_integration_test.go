//go:build integration

package postgres

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCtx(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), 10*time.Second)
}

func TestEventRepo_Create_NewEvent(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	repo := NewEventRepo(pool)

	customer := testutil.CreateTestCustomer(t, pool)
	pixel := testutil.CreateTestPixel(t, pool, customer.ID)

	ctx, cancel := newCtx(t)
	defer cancel()

	e := &domain.PixelEvent{
		PixelID:   pixel.ID,
		EventName: "Purchase",
		EventData: json.RawMessage(`{"value":250,"currency":"THB"}`),
		UserData:  json.RawMessage(`{"em":["abc"]}`),
		SourceURL: "https://example.com/checkout",
		EventTime: time.Now(),
		EventID:   "evt_new_1",
	}

	inserted, err := repo.Create(ctx, e)
	require.NoError(t, err)
	assert.True(t, inserted, "first insert should return true")
	assert.NotEmpty(t, e.ID, "ID should be populated after insert")
	assert.False(t, e.ForwardedToCAPI, "new event must default to forwarded=false")
	assert.False(t, e.CreatedAt.IsZero(), "CreatedAt should be populated")
}

func TestEventRepo_Create_DuplicateEventID(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	repo := NewEventRepo(pool)

	customer := testutil.CreateTestCustomer(t, pool)
	pixel := testutil.CreateTestPixel(t, pool, customer.ID)

	ctx, cancel := newCtx(t)
	defer cancel()

	first := &domain.PixelEvent{
		PixelID:   pixel.ID,
		EventName: "Purchase",
		EventData: json.RawMessage(`{}`),
		UserData:  json.RawMessage(`{}`),
		SourceURL: "https://example.com",
		EventTime: time.Now(),
		EventID:   "evt_dup",
	}
	inserted, err := repo.Create(ctx, first)
	require.NoError(t, err)
	require.True(t, inserted)

	dup := &domain.PixelEvent{
		PixelID:   pixel.ID,
		EventName: "Purchase",
		EventData: json.RawMessage(`{}`),
		UserData:  json.RawMessage(`{}`),
		SourceURL: "https://example.com",
		EventTime: time.Now(),
		EventID:   "evt_dup",
	}
	inserted2, err := repo.Create(ctx, dup)
	require.NoError(t, err)
	assert.False(t, inserted2, "second insert with same (pixel_id, event_id) must return false")
}

func TestEventRepo_Create_EmptyEventID_BypassesDedup(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	repo := NewEventRepo(pool)

	customer := testutil.CreateTestCustomer(t, pool)
	pixel := testutil.CreateTestPixel(t, pool, customer.ID)

	ctx, cancel := newCtx(t)
	defer cancel()

	// Production behavior note: EventRepo.Create passes e.EventID (Go string) directly,
	// so an empty value is stored as '' (empty string), NOT NULL. The partial unique
	// index `WHERE event_id IS NOT NULL` therefore still applies to ''. This is a
	// latent behavior — clients that omit event_id will collide with each other.
	// This test pins the current behavior so any future change is intentional.
	first := &domain.PixelEvent{
		PixelID: pixel.ID, EventName: "PageView",
		EventData: json.RawMessage(`{}`), UserData: json.RawMessage(`{}`),
		SourceURL: "https://example.com", EventTime: time.Now(),
	}
	inserted, err := repo.Create(ctx, first)
	require.NoError(t, err)
	assert.True(t, inserted, "first insert with empty event_id succeeds")

	second := &domain.PixelEvent{
		PixelID: pixel.ID, EventName: "PageView",
		EventData: json.RawMessage(`{}`), UserData: json.RawMessage(`{}`),
		SourceURL: "https://example.com", EventTime: time.Now(),
	}
	inserted, err = repo.Create(ctx, second)
	require.NoError(t, err)
	assert.False(t, inserted, "second insert with same empty event_id collides — '' is treated as a dedup key")

	_, total, err := repo.ListByPixelID(ctx, pixel.ID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total, "only first insert persists due to '' dedup collision")
}

func TestEventRepo_GetByID_Found(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	repo := NewEventRepo(pool)

	customer := testutil.CreateTestCustomer(t, pool)
	pixel := testutil.CreateTestPixel(t, pool, customer.ID)
	created := testutil.CreateTestEvent(t, pool, pixel.ID, func(e *domain.PixelEvent) {
		e.EventID = "evt_lookup"
		e.ClientIP = "1.2.3.4"
		e.ClientUserAgent = "ua-test"
	})

	ctx, cancel := newCtx(t)
	defer cancel()

	got, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, pixel.ID, got.PixelID)
	assert.Equal(t, "Purchase", got.EventName)
	assert.Equal(t, "evt_lookup", got.EventID)
	assert.Equal(t, "1.2.3.4", got.ClientIP)
	assert.Equal(t, "ua-test", got.ClientUserAgent)
}

func TestEventRepo_GetByID_NotFound(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	repo := NewEventRepo(pool)

	ctx, cancel := newCtx(t)
	defer cancel()

	got, err := repo.GetByID(ctx, "00000000-0000-0000-0000-000000000000")
	require.NoError(t, err)
	assert.Nil(t, got, "missing id must return (nil, nil)")
}

func TestEventRepo_ListByPixelID_PaginationAndOrder(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	repo := NewEventRepo(pool)

	customer := testutil.CreateTestCustomer(t, pool)
	pixel := testutil.CreateTestPixel(t, pool, customer.ID)

	// insert 5 events with increasing event_time
	base := time.Now().Add(-5 * time.Hour)
	for i := 0; i < 5; i++ {
		i := i
		testutil.CreateTestEvent(t, pool, pixel.ID, func(e *domain.PixelEvent) {
			e.EventTime = base.Add(time.Duration(i) * time.Hour)
			e.EventName = "Purchase"
		})
	}

	ctx, cancel := newCtx(t)
	defer cancel()

	// page 1: limit 2 → newest 2 first
	page1, total, err := repo.ListByPixelID(ctx, pixel.ID, 2, 0)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	require.Len(t, page1, 2)
	assert.True(t, page1[0].EventTime.After(page1[1].EventTime) || page1[0].EventTime.Equal(page1[1].EventTime),
		"results must be ordered by event_time DESC")

	// page 2: offset 2 limit 2
	page2, _, err := repo.ListByPixelID(ctx, pixel.ID, 2, 2)
	require.NoError(t, err)
	require.Len(t, page2, 2)
	assert.True(t, page1[1].EventTime.After(page2[0].EventTime) || page1[1].EventTime.Equal(page2[0].EventTime),
		"page2 must be older than end of page1")

	// page 3: offset 4 limit 2 → only 1 left
	page3, _, err := repo.ListByPixelID(ctx, pixel.ID, 2, 4)
	require.NoError(t, err)
	assert.Len(t, page3, 1)
}

func TestEventRepo_ListByCustomerID_Filters(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	repo := NewEventRepo(pool)

	customer := testutil.CreateTestCustomer(t, pool)
	pixelA := testutil.CreateTestPixel(t, pool, customer.ID)
	pixelB := testutil.CreateTestPixel(t, pool, customer.ID)

	now := time.Now()
	// pixelA: 2 Purchase (in range), 1 PageView (in range), 1 Purchase (out of range)
	testutil.CreateTestEvent(t, pool, pixelA.ID, func(e *domain.PixelEvent) { e.EventName = "Purchase"; e.EventTime = now.Add(-1 * time.Hour) })
	testutil.CreateTestEvent(t, pool, pixelA.ID, func(e *domain.PixelEvent) { e.EventName = "Purchase"; e.EventTime = now.Add(-2 * time.Hour) })
	testutil.CreateTestEvent(t, pool, pixelA.ID, func(e *domain.PixelEvent) { e.EventName = "PageView"; e.EventTime = now.Add(-3 * time.Hour) })
	testutil.CreateTestEvent(t, pool, pixelA.ID, func(e *domain.PixelEvent) { e.EventName = "Purchase"; e.EventTime = now.Add(-48 * time.Hour) })
	// pixelB: 1 Purchase in range
	testutil.CreateTestEvent(t, pool, pixelB.ID, func(e *domain.PixelEvent) { e.EventName = "Purchase"; e.EventTime = now.Add(-1 * time.Hour) })

	ctx, cancel := newCtx(t)
	defer cancel()

	// no filters → 5 events total
	_, total, err := repo.ListByCustomerID(ctx, customer.ID, "", "", nil, nil, 100, 0)
	require.NoError(t, err)
	assert.Equal(t, 5, total)

	// filter by pixelID
	_, total, err = repo.ListByCustomerID(ctx, customer.ID, pixelB.ID, "", nil, nil, 100, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)

	// filter by event name
	_, total, err = repo.ListByCustomerID(ctx, customer.ID, "", "Purchase", nil, nil, 100, 0)
	require.NoError(t, err)
	assert.Equal(t, 4, total)

	// filter by date range (last 24h) → exclude the -48h event
	from := now.Add(-24 * time.Hour)
	to := now
	events, total, err := repo.ListByCustomerID(ctx, customer.ID, "", "", &from, &to, 100, 0)
	require.NoError(t, err)
	assert.Equal(t, 4, total)
	assert.Len(t, events, 4)

	// combined: pixelA + Purchase + last 24h → 2
	_, total, err = repo.ListByCustomerID(ctx, customer.ID, pixelA.ID, "Purchase", &from, &to, 100, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
}

func TestEventRepo_MarkForwarded(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	repo := NewEventRepo(pool)

	customer := testutil.CreateTestCustomer(t, pool)
	pixel := testutil.CreateTestPixel(t, pool, customer.ID)
	created := testutil.CreateTestEvent(t, pool, pixel.ID)

	ctx, cancel := newCtx(t)
	defer cancel()

	err := repo.MarkForwarded(ctx, created.ID, 200, 1)
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.ForwardedToCAPI, "forwarded_to_capi must be true")
	require.NotNil(t, got.CAPIResponseCode)
	assert.Equal(t, 200, *got.CAPIResponseCode)
	require.NotNil(t, got.CAPIEventsReceived)
	assert.Equal(t, 1, *got.CAPIEventsReceived)
}

func TestEventRepo_GetEventsForReplay_Filters(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	repo := NewEventRepo(pool)

	customer := testutil.CreateTestCustomer(t, pool)
	pixel := testutil.CreateTestPixel(t, pool, customer.ID)

	now := time.Now()
	testutil.CreateTestEvent(t, pool, pixel.ID, func(e *domain.PixelEvent) { e.EventName = "Purchase"; e.EventTime = now.Add(-1 * time.Hour) })
	testutil.CreateTestEvent(t, pool, pixel.ID, func(e *domain.PixelEvent) { e.EventName = "PageView"; e.EventTime = now.Add(-2 * time.Hour) })
	testutil.CreateTestEvent(t, pool, pixel.ID, func(e *domain.PixelEvent) { e.EventName = "Purchase"; e.EventTime = now.Add(-3 * time.Hour) })
	testutil.CreateTestEvent(t, pool, pixel.ID, func(e *domain.PixelEvent) { e.EventName = "Lead"; e.EventTime = now.Add(-48 * time.Hour) })

	ctx, cancel := newCtx(t)
	defer cancel()

	// filter by event types
	events, err := repo.GetEventsForReplay(ctx, pixel.ID, []string{"Purchase"}, nil, nil, nil)
	require.NoError(t, err)
	assert.Len(t, events, 2)
	for _, e := range events {
		assert.Equal(t, "Purchase", e.EventName)
	}

	// filter by date range (last 24h)
	from := now.Add(-24 * time.Hour)
	to := now
	events, err = repo.GetEventsForReplay(ctx, pixel.ID, nil, &from, &to, nil)
	require.NoError(t, err)
	assert.Len(t, events, 3, "should exclude the -48h Lead event")

	// ordering ASC
	if len(events) >= 2 {
		assert.True(t, events[0].EventTime.Before(events[1].EventTime) || events[0].EventTime.Equal(events[1].EventTime),
			"replay results must be ordered ASC by event_time")
	}

	// createdBefore in the past → exclude all (events were just created)
	past := now.Add(-1 * time.Hour)
	events, err = repo.GetEventsForReplay(ctx, pixel.ID, nil, nil, nil, &past)
	require.NoError(t, err)
	assert.Empty(t, events, "createdBefore in past should exclude freshly-inserted rows")
}

func TestEventRepo_GetDistinctEventTypes(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	repo := NewEventRepo(pool)

	customer := testutil.CreateTestCustomer(t, pool)
	pixel := testutil.CreateTestPixel(t, pool, customer.ID)

	testutil.CreateTestEvent(t, pool, pixel.ID, func(e *domain.PixelEvent) { e.EventName = "Purchase" })
	testutil.CreateTestEvent(t, pool, pixel.ID, func(e *domain.PixelEvent) { e.EventName = "Purchase" })
	testutil.CreateTestEvent(t, pool, pixel.ID, func(e *domain.PixelEvent) { e.EventName = "PageView" })
	testutil.CreateTestEvent(t, pool, pixel.ID, func(e *domain.PixelEvent) { e.EventName = "AddToCart" })

	ctx, cancel := newCtx(t)
	defer cancel()

	types, err := repo.GetDistinctEventTypes(ctx, pixel.ID)
	require.NoError(t, err)
	// ORDER BY event_name → AddToCart, PageView, Purchase
	assert.Equal(t, []string{"AddToCart", "PageView", "Purchase"}, types)
}

func TestEventRepo_DeleteOlderThan(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	repo := NewEventRepo(pool)

	customer := testutil.CreateTestCustomer(t, pool)
	pixel := testutil.CreateTestPixel(t, pool, customer.ID)

	// Insert 4 events, then back-date 2 of them via raw UPDATE
	old1 := testutil.CreateTestEvent(t, pool, pixel.ID)
	old2 := testutil.CreateTestEvent(t, pool, pixel.ID)
	_ = testutil.CreateTestEvent(t, pool, pixel.ID) // fresh
	_ = testutil.CreateTestEvent(t, pool, pixel.ID) // fresh

	ctx, cancel := newCtx(t)
	defer cancel()

	oldTime := time.Now().Add(-30 * 24 * time.Hour)
	_, err := pool.Exec(ctx, `UPDATE pixel_events SET created_at = $1 WHERE id IN ($2, $3)`,
		oldTime, old1.ID, old2.ID)
	require.NoError(t, err)

	// Delete with batchSize=1 → should remove only one row at a time
	before := time.Now().Add(-7 * 24 * time.Hour)
	deleted, err := repo.DeleteOlderThan(ctx, before, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted, "batchSize=1 must cap deletion at 1 row")

	// Delete the remaining old row
	deleted, err = repo.DeleteOlderThan(ctx, before, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// Third pass — nothing left to delete
	deleted, err = repo.DeleteOlderThan(ctx, before, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted)

	// fresh events untouched
	_, total, err := repo.ListByPixelID(ctx, pixel.ID, 100, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, total, "fresh events must remain")
}

func TestEventRepo_DeleteExpiredByRetention(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	repo := NewEventRepo(pool)

	customer := testutil.CreateTestCustomer(t, pool)
	pixel := testutil.CreateTestPixel(t, pool, customer.ID)

	ctx, cancel := newCtx(t)
	defer cancel()

	// Force retention_days = 1 for this customer
	_, err := pool.Exec(ctx, `UPDATE customers SET retention_days = 1 WHERE id = $1`, customer.ID)
	require.NoError(t, err)

	// Insert events; back-date 2 of them by 2 days (older than retention)
	old1 := testutil.CreateTestEvent(t, pool, pixel.ID)
	old2 := testutil.CreateTestEvent(t, pool, pixel.ID)
	_ = testutil.CreateTestEvent(t, pool, pixel.ID) // fresh, must survive

	twoDaysAgo := time.Now().Add(-2 * 24 * time.Hour)
	_, err = pool.Exec(ctx, `UPDATE pixel_events SET created_at = $1 WHERE id IN ($2, $3)`,
		twoDaysAgo, old1.ID, old2.ID)
	require.NoError(t, err)

	deleted, err := repo.DeleteExpiredByRetention(ctx, 100)
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted, "both back-dated rows should be deleted")

	_, total, err := repo.ListByPixelID(ctx, pixel.ID, 100, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total, "fresh event should remain")
}
