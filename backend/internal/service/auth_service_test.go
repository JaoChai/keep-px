package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jaochai/pixlinks/backend/internal/config"
	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository/mocks"
)

func newTestAuthService() (*AuthService, *mocks.MockCustomerRepo, *mocks.MockRefreshTokenRepo) {
	customerRepo := new(mocks.MockCustomerRepo)
	refreshTokenRepo := new(mocks.MockRefreshTokenRepo)
	cfg := &config.Config{
		JWTSecret:     "test-secret",
		JWTAccessTTL:  15 * time.Minute,
		JWTRefreshTTL: 168 * time.Hour,
	}
	svc := NewAuthService(customerRepo, refreshTokenRepo, cfg)
	return svc, customerRepo, refreshTokenRepo
}

func TestAuthService_RefreshTokens(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		setup     func(*mocks.MockCustomerRepo, *mocks.MockRefreshTokenRepo)
		wantErr   error
		wantToken bool
	}{
		{
			name:  "success",
			token: "valid-refresh-token",
			setup: func(cr *mocks.MockCustomerRepo, rt *mocks.MockRefreshTokenRepo) {
				rt.On("GetByTokenHash", mock.Anything, mock.AnythingOfType("string")).
					Return("cust-1", time.Now().Add(time.Hour), nil)
				rt.On("DeleteByTokenHash", mock.Anything, mock.AnythingOfType("string")).Return(nil)
				cr.On("GetByID", mock.Anything, "cust-1").Return(&domain.Customer{
					ID:    "cust-1",
					Email: "test@example.com",
				}, nil)
				rt.On("Create", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil)
			},
			wantErr:   nil,
			wantToken: true,
		},
		{
			name:  "invalid token",
			token: "invalid-token",
			setup: func(cr *mocks.MockCustomerRepo, rt *mocks.MockRefreshTokenRepo) {
				rt.On("GetByTokenHash", mock.Anything, mock.AnythingOfType("string")).
					Return("", time.Time{}, nil)
			},
			wantErr:   ErrInvalidRefreshToken,
			wantToken: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, customerRepo, refreshTokenRepo := newTestAuthService()
			tt.setup(customerRepo, refreshTokenRepo)

			tokens, err := svc.RefreshTokens(context.Background(), tt.token)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, tokens)
			} else {
				assert.NoError(t, err)
			}
			if tt.wantToken {
				assert.NotNil(t, tokens)
				assert.NotEmpty(t, tokens.AccessToken)
				assert.NotEmpty(t, tokens.RefreshToken)
			}
			customerRepo.AssertExpectations(t)
			refreshTokenRepo.AssertExpectations(t)
		})
	}
}

func TestAuthService_Logout(t *testing.T) {
	tests := []struct {
		name       string
		customerID string
		setup      func(*mocks.MockRefreshTokenRepo)
		wantErr    bool
	}{
		{
			name:       "success",
			customerID: "cust-1",
			setup: func(rt *mocks.MockRefreshTokenRepo) {
				rt.On("DeleteByCustomerID", mock.Anything, "cust-1").Return(nil)
			},
			wantErr: false,
		},
		{
			name:       "repo error",
			customerID: "cust-2",
			setup: func(rt *mocks.MockRefreshTokenRepo) {
				rt.On("DeleteByCustomerID", mock.Anything, "cust-2").Return(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, refreshTokenRepo := newTestAuthService()
			tt.setup(refreshTokenRepo)

			err := svc.Logout(context.Background(), tt.customerID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			refreshTokenRepo.AssertExpectations(t)
		})
	}
}

func TestProvisionCustomer(t *testing.T) {
	ctx := context.Background()

	t.Run("สร้างบัญชีใหม่เมื่อยังไม่เคยมี", func(t *testing.T) {
		svc, customerRepo, _ := newTestAuthService()
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
		svc, customerRepo, _ := newTestAuthService()
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
		svc, customerRepo, _ := newTestAuthService()
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
		svc, customerRepo, _ := newTestAuthService()
		customerRepo.On("GetByAuthUserID", mock.Anything, "neon-คนร้าย").Return(nil, nil)
		// ไม่ตั้ง GetByEmail — ต้อง error ก่อนถึง ไม่งั้นคนร้ายสวมบัญชีคนอื่นได้

		_, err := svc.ProvisionCustomer(ctx, ProvisionInput{
			AuthUserID: "neon-คนร้าย", Email: "victim@example.com", EmailVerified: false,
		})

		assert.ErrorIs(t, err, ErrEmailNotVerified)
		customerRepo.AssertExpectations(t)
	})

	t.Run("บัญชีถูกระงับต้องเข้าไม่ได้", func(t *testing.T) {
		svc, customerRepo, _ := newTestAuthService()
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
