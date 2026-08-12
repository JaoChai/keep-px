//go:build integration

package postgres

import (
	"context"
	"testing"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomerRepo_GetByAuthUserID(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	repo := NewCustomerRepo(pool)
	ctx := context.Background()

	authUserID := "neon-user-abc123"
	c := &domain.Customer{
		Email:         "authuser@example.com",
		Name:          "Auth User",
		APIKey:        "pk_test_getbyauthuserid",
		Plan:          domain.PlanSandbox,
		RetentionDays: 7,
		AuthUserID:    &authUserID,
	}
	require.NoError(t, repo.Create(ctx, c))

	t.Run("เจอเมื่อมีอยู่จริง", func(t *testing.T) {
		got, err := repo.GetByAuthUserID(ctx, authUserID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, c.ID, got.ID)
		require.NotNil(t, got.AuthUserID, "ถ้า scanCustomer ลืมคอลัมน์ใหม่ ตรงนี้จะจับได้")
		assert.Equal(t, authUserID, *got.AuthUserID)
	})

	t.Run("คืน nil เมื่อไม่มี ไม่ใช่ error", func(t *testing.T) {
		got, err := repo.GetByAuthUserID(ctx, "neon-user-ไม่มีอยู่จริง")
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}
