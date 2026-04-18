package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

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
