package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository/mocks"
)

func newTestAuthService() (*AuthService, *mocks.MockCustomerRepo) {
	customerRepo := new(mocks.MockCustomerRepo)
	svc := NewAuthService(customerRepo)
	return svc, customerRepo
}

func TestProvisionCustomer(t *testing.T) {
	ctx := context.Background()

	t.Run("สร้างบัญชีใหม่เมื่อยังไม่เคยมี", func(t *testing.T) {
		svc, customerRepo := newTestAuthService()
		customerRepo.On("GetByAuthUserID", mock.Anything, "neon-1").Return(nil, nil)
		customerRepo.On("GetByEmail", mock.Anything, "new@example.com").Return(nil, nil)
		customerRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

		got, err := svc.ProvisionCustomer(ctx, ProvisionInput{
			AuthUserID: "neon-1", Email: "new@example.com", Name: "คนใหม่", EmailVerified: true,
		})

		require.NoError(t, err)
		require.NotNil(t, got.AuthUserID)
		assert.Equal(t, "neon-1", *got.AuthUserID)
		assert.Equal(t, domain.PlanSandbox, got.Plan)
		assert.True(t, strings.HasPrefix(got.APIKey, "pk_"), "ต้องได้ API key ตั้งแต่แรก")
		assert.Equal(t, 7, got.RetentionDays)
		customerRepo.AssertExpectations(t)
	})

	t.Run("ผูกกับบัญชีเดิมที่ email ตรงกัน ไม่สร้างซ้ำ", func(t *testing.T) {
		svc, customerRepo := newTestAuthService()
		existing := &domain.Customer{
			ID: "cust-เดิม", Email: "old@example.com", APIKey: "pk_ของเดิม", Plan: domain.PlanPaid,
		}
		customerRepo.On("GetByAuthUserID", mock.Anything, "neon-2").Return(nil, nil)
		customerRepo.On("GetByEmail", mock.Anything, "old@example.com").Return(existing, nil)
		customerRepo.On("Update", mock.Anything, existing).Return(nil)
		// ไม่ตั้ง Create expectation — testify จะ fail เองถ้าถูกเรียก

		got, err := svc.ProvisionCustomer(ctx, ProvisionInput{
			AuthUserID: "neon-2", Email: "old@example.com", Name: "คนเดิม", EmailVerified: true,
		})

		require.NoError(t, err)
		assert.Equal(t, "cust-เดิม", got.ID, "ต้องเป็นบัญชีเดิม ไม่ใช่บัญชีใหม่")
		assert.Equal(t, "pk_ของเดิม", got.APIKey, "API key เดิมต้องไม่หาย")
		assert.Equal(t, domain.PlanPaid, got.Plan, "แพลนเดิมต้องไม่ถูกรีเซ็ต")
		customerRepo.AssertExpectations(t)
	})

	t.Run("เรียกซ้ำได้ผลเดิม ไม่สร้างซ้ำ", func(t *testing.T) {
		svc, customerRepo := newTestAuthService()
		second := &domain.Customer{ID: "cust-3", Email: "a@example.com"}
		customerRepo.On("GetByAuthUserID", mock.Anything, "neon-3").Return(nil, nil).Once()
		customerRepo.On("GetByAuthUserID", mock.Anything, "neon-3").Return(second, nil).Once()
		customerRepo.On("GetByEmail", mock.Anything, "a@example.com").Return(nil, nil)
		customerRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			args.Get(1).(*domain.Customer).ID = "cust-3"
		})
		in := ProvisionInput{AuthUserID: "neon-3", Email: "a@example.com", EmailVerified: true}

		first, err := svc.ProvisionCustomer(ctx, in)
		require.NoError(t, err)
		gotSecond, err := svc.ProvisionCustomer(ctx, in)
		require.NoError(t, err)

		assert.Equal(t, first.ID, gotSecond.ID)
		customerRepo.AssertNumberOfCalls(t, "Create", 1)
	})

	t.Run("email ที่ยังไม่ยืนยันต้องไม่ผูกกับบัญชีเดิม", func(t *testing.T) {
		svc, customerRepo := newTestAuthService()
		customerRepo.On("GetByAuthUserID", mock.Anything, "neon-คนร้าย").Return(nil, nil)
		// ไม่ตั้ง GetByEmail — ต้อง error ก่อนถึง ไม่งั้นคนร้ายสวมบัญชีคนอื่นได้

		_, err := svc.ProvisionCustomer(ctx, ProvisionInput{
			AuthUserID: "neon-คนร้าย", Email: "victim@example.com", EmailVerified: false,
		})

		assert.ErrorIs(t, err, ErrEmailNotVerified)
		customerRepo.AssertExpectations(t)
	})

	t.Run("บัญชีถูกระงับต้องเข้าไม่ได้", func(t *testing.T) {
		svc, customerRepo := newTestAuthService()
		suspended := time.Now()
		customerRepo.On("GetByAuthUserID", mock.Anything, "neon-4").Return(&domain.Customer{
			ID: "cust-ระงับ", Email: "s@example.com", SuspendedAt: &suspended,
		}, nil)

		_, err := svc.ProvisionCustomer(ctx, ProvisionInput{
			AuthUserID: "neon-4", Email: "s@example.com", EmailVerified: true,
		})

		assert.ErrorIs(t, err, ErrAccountSuspended)
		customerRepo.AssertExpectations(t)
	})
}
