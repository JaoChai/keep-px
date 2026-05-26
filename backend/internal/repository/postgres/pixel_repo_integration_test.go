//go:build integration

package postgres

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaochai/pixlinks/backend/internal/crypto"
	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newEncryptor builds a TokenEncryptor backed by a random 32-byte key.
func newEncryptor(t *testing.T) *crypto.TokenEncryptor {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	enc, err := crypto.NewTokenEncryptor(key)
	require.NoError(t, err)
	return enc
}

// rawTokenFromDB reads fb_access_token directly via SQL, bypassing the repo.
func rawTokenFromDB(t *testing.T, pool *pgxpool.Pool, pixelID string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var token string
	err := pool.QueryRow(ctx,
		`SELECT fb_access_token FROM pixels WHERE id = $1`, pixelID,
	).Scan(&token)
	require.NoError(t, err)
	return token
}

func TestPixelRepo_Create_WithoutEncryptor(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)

	customer := testutil.CreateTestCustomer(t, pool)
	repo := NewPixelRepo(pool, nil)

	ctx := context.Background()
	p := &domain.Pixel{
		CustomerID:    customer.ID,
		FBPixelID:     "9999999999",
		FBAccessToken: "plaintext-token",
		Name:          "Plain Pixel",
	}
	require.NoError(t, repo.Create(ctx, p))

	assert.NotEmpty(t, p.ID, "ID should be populated")
	assert.True(t, p.IsActive, "IsActive should default to true")
	assert.False(t, p.CreatedAt.IsZero(), "CreatedAt should be populated")
	assert.False(t, p.UpdatedAt.IsZero(), "UpdatedAt should be populated")

	// Verify stored value is plaintext (no enc: prefix).
	raw := rawTokenFromDB(t, pool, p.ID)
	assert.Equal(t, "plaintext-token", raw)
	assert.False(t, crypto.IsEncrypted(raw))
}

func TestPixelRepo_Create_WithEncryptor(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)

	customer := testutil.CreateTestCustomer(t, pool)
	enc := newEncryptor(t)
	repo := NewPixelRepo(pool, enc)

	ctx := context.Background()
	p := &domain.Pixel{
		CustomerID:    customer.ID,
		FBPixelID:     "1111111111",
		FBAccessToken: "super-secret-token",
		Name:          "Encrypted Pixel",
	}
	require.NoError(t, repo.Create(ctx, p))
	require.NotEmpty(t, p.ID)

	raw := rawTokenFromDB(t, pool, p.ID)
	assert.True(t, crypto.IsEncrypted(raw), "stored token should have enc: prefix, got %q", raw)
	assert.NotEqual(t, "super-secret-token", raw)
}

func TestPixelRepo_GetByID_WithEncryptor_DecryptsToken(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)

	customer := testutil.CreateTestCustomer(t, pool)
	enc := newEncryptor(t)
	repo := NewPixelRepo(pool, enc)

	ctx := context.Background()
	original := "round-trip-token-xyz"
	p := &domain.Pixel{
		CustomerID:    customer.ID,
		FBPixelID:     "2222222222",
		FBAccessToken: original,
		Name:          "Round Trip",
	}
	require.NoError(t, repo.Create(ctx, p))

	// Sanity: stored encrypted.
	raw := rawTokenFromDB(t, pool, p.ID)
	require.True(t, crypto.IsEncrypted(raw))

	got, err := repo.GetByID(ctx, p.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, original, got.FBAccessToken, "token should be decrypted back to plaintext")
	assert.Equal(t, p.ID, got.ID)
	assert.Equal(t, customer.ID, got.CustomerID)
	assert.Equal(t, "2222222222", got.FBPixelID)
}

func TestPixelRepo_GetByID_NotFound_ReturnsNilNil(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)

	repo := NewPixelRepo(pool, nil)
	ctx := context.Background()

	// Use a syntactically valid UUID that does not exist.
	got, err := repo.GetByID(ctx, "00000000-0000-0000-0000-000000000000")
	assert.NoError(t, err)
	assert.Nil(t, got)
}

func TestPixelRepo_GetByIDs_Batch(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)

	customer := testutil.CreateTestCustomer(t, pool)
	enc := newEncryptor(t)
	repo := NewPixelRepo(pool, enc)

	ctx := context.Background()
	pixels := make([]*domain.Pixel, 3)
	ids := make([]string, 3)
	for i := 0; i < 3; i++ {
		p := &domain.Pixel{
			CustomerID:    customer.ID,
			FBPixelID:     "33333333" + string(rune('0'+i)),
			FBAccessToken: "tok-" + string(rune('a'+i)),
			Name:          "Batch Pixel",
		}
		require.NoError(t, repo.Create(ctx, p))
		pixels[i] = p
		ids[i] = p.ID
	}

	got, err := repo.GetByIDs(ctx, ids)
	require.NoError(t, err)
	require.Len(t, got, 3)

	// Verify each token was decrypted.
	tokens := map[string]string{}
	for _, p := range got {
		tokens[p.ID] = p.FBAccessToken
	}
	for i, p := range pixels {
		assert.Equal(t, "tok-"+string(rune('a'+i)), tokens[p.ID])
	}
}

func TestPixelRepo_GetByIDs_EmptyInput(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)

	repo := NewPixelRepo(pool, nil)
	got, err := repo.GetByIDs(context.Background(), nil)
	assert.NoError(t, err)
	assert.Nil(t, got)

	got, err = repo.GetByIDs(context.Background(), []string{})
	assert.NoError(t, err)
	assert.Nil(t, got)
}

func TestPixelRepo_ListByCustomerID_OrderedAndScoped(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)

	customerA := testutil.CreateTestCustomer(t, pool)
	customerB := testutil.CreateTestCustomer(t, pool)
	repo := NewPixelRepo(pool, nil)
	ctx := context.Background()

	// Create 3 pixels for A with deliberate delay so created_at differs.
	var aIDs []string
	for i := 0; i < 3; i++ {
		p := &domain.Pixel{
			CustomerID:    customerA.ID,
			FBPixelID:     "44444444" + string(rune('0'+i)),
			FBAccessToken: "a-tok",
			Name:          "A Pixel",
		}
		require.NoError(t, repo.Create(ctx, p))
		aIDs = append(aIDs, p.ID)
		time.Sleep(10 * time.Millisecond)
	}
	// And 1 pixel for B (should not leak into A's list).
	bPixel := &domain.Pixel{
		CustomerID:    customerB.ID,
		FBPixelID:     "5555555555",
		FBAccessToken: "b-tok",
		Name:          "B Pixel",
	}
	require.NoError(t, repo.Create(ctx, bPixel))

	got, err := repo.ListByCustomerID(ctx, customerA.ID)
	require.NoError(t, err)
	require.Len(t, got, 3)

	// All belong to customer A.
	for _, p := range got {
		assert.Equal(t, customerA.ID, p.CustomerID)
	}

	// Ordered created_at DESC → newest (last created) first.
	assert.Equal(t, aIDs[2], got[0].ID)
	assert.Equal(t, aIDs[1], got[1].ID)
	assert.Equal(t, aIDs[0], got[2].ID)

	// Sanity: B sees just its own.
	gotB, err := repo.ListByCustomerID(ctx, customerB.ID)
	require.NoError(t, err)
	require.Len(t, gotB, 1)
	assert.Equal(t, bPixel.ID, gotB[0].ID)
}

func TestPixelRepo_CountByCustomerID(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)

	customer := testutil.CreateTestCustomer(t, pool)
	repo := NewPixelRepo(pool, nil)
	ctx := context.Background()

	// Initially zero.
	count, err := repo.CountByCustomerID(ctx, customer.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Insert 4 pixels.
	for i := 0; i < 4; i++ {
		testutil.CreateTestPixel(t, pool, customer.ID)
	}

	count, err = repo.CountByCustomerID(ctx, customer.ID)
	require.NoError(t, err)
	assert.Equal(t, 4, count)

	// Unrelated customer still sees zero.
	other := testutil.CreateTestCustomer(t, pool)
	count, err = repo.CountByCustomerID(ctx, other.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestPixelRepo_Update_ReEncryptsToken(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)

	customer := testutil.CreateTestCustomer(t, pool)
	enc := newEncryptor(t)
	repo := NewPixelRepo(pool, enc)
	ctx := context.Background()

	p := &domain.Pixel{
		CustomerID:    customer.ID,
		FBPixelID:     "6666666666",
		FBAccessToken: "old-token",
		Name:          "Before",
	}
	require.NoError(t, repo.Create(ctx, p))

	// Mutate fields and update.
	p.FBAccessToken = "brand-new-token"
	p.Name = "After"
	p.IsActive = false
	require.NoError(t, repo.Update(ctx, p))

	// Raw stored value must be encrypted (not the plaintext).
	raw := rawTokenFromDB(t, pool, p.ID)
	assert.True(t, crypto.IsEncrypted(raw), "updated token should be encrypted, got %q", raw)
	assert.NotEqual(t, "brand-new-token", raw)

	// And decrypt back via GetByID.
	got, err := repo.GetByID(ctx, p.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "brand-new-token", got.FBAccessToken)
	assert.Equal(t, "After", got.Name)
	assert.False(t, got.IsActive)
}

func TestPixelRepo_Delete_CascadesToEvents(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)

	customer := testutil.CreateTestCustomer(t, pool)
	repo := NewPixelRepo(pool, nil)
	ctx := context.Background()

	pixel := testutil.CreateTestPixel(t, pool, customer.ID)
	// Insert two events for that pixel.
	testutil.CreateTestEvent(t, pool, pixel.ID)
	testutil.CreateTestEvent(t, pool, pixel.ID)

	// Sanity: events exist.
	var before int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM pixel_events WHERE pixel_id = $1`, pixel.ID,
	).Scan(&before))
	require.Equal(t, 2, before)

	// Delete the pixel.
	require.NoError(t, repo.Delete(ctx, pixel.ID))

	// Pixel is gone.
	got, err := repo.GetByID(ctx, pixel.ID)
	require.NoError(t, err)
	assert.Nil(t, got)

	// Events were cascaded.
	var after int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM pixel_events WHERE pixel_id = $1`, pixel.ID,
	).Scan(&after))
	assert.Equal(t, 0, after, "pixel_events should be cascade-deleted")
}
