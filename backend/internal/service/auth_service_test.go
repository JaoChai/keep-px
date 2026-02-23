package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"

	"github.com/jaochai/pixlinks/backend/internal/config"
	"github.com/jaochai/pixlinks/backend/internal/domain"
)

func newTestAuthService() (*AuthService, *MockCustomerRepo, *MockRefreshTokenRepo) {
	customerRepo := new(MockCustomerRepo)
	refreshTokenRepo := new(MockRefreshTokenRepo)
	cfg := &config.Config{
		JWTSecret:     "test-secret",
		JWTAccessTTL:  15 * time.Minute,
		JWTRefreshTTL: 168 * time.Hour,
	}
	svc := NewAuthService(customerRepo, refreshTokenRepo, cfg)
	return svc, customerRepo, refreshTokenRepo
}

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name      string
		input     RegisterInput
		setup     func(*MockCustomerRepo, *MockRefreshTokenRepo)
		wantErr   error
		wantToken bool
	}{
		{
			name: "success",
			input: RegisterInput{
				Email:    "test@example.com",
				Password: "password123",
				Name:     "Test User",
			},
			setup: func(cr *MockCustomerRepo, rt *MockRefreshTokenRepo) {
				cr.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, nil)
				cr.On("Create", mock.Anything, mock.AnythingOfType("*domain.Customer")).Return(nil)
				rt.On("Create", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil)
			},
			wantErr:   nil,
			wantToken: true,
		},
		{
			name: "duplicate email",
			input: RegisterInput{
				Email:    "existing@example.com",
				Password: "password123",
				Name:     "Existing User",
			},
			setup: func(cr *MockCustomerRepo, rt *MockRefreshTokenRepo) {
				cr.On("GetByEmail", mock.Anything, "existing@example.com").Return(&domain.Customer{
					ID:    "existing-id",
					Email: "existing@example.com",
				}, nil)
			},
			wantErr:   ErrEmailAlreadyExists,
			wantToken: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, customerRepo, refreshTokenRepo := newTestAuthService()
			tt.setup(customerRepo, refreshTokenRepo)

			tokens, err := svc.Register(context.Background(), tt.input)

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
				assert.NotNil(t, tokens.Customer)
			}
			customerRepo.AssertExpectations(t)
			refreshTokenRepo.AssertExpectations(t)
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	hashedPw, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	tests := []struct {
		name      string
		input     LoginInput
		setup     func(*MockCustomerRepo, *MockRefreshTokenRepo)
		wantErr   error
		wantToken bool
	}{
		{
			name: "success",
			input: LoginInput{
				Email:    "test@example.com",
				Password: "password123",
			},
			setup: func(cr *MockCustomerRepo, rt *MockRefreshTokenRepo) {
				cr.On("GetByEmail", mock.Anything, "test@example.com").Return(&domain.Customer{
					ID:           "cust-1",
					Email:        "test@example.com",
					PasswordHash: string(hashedPw),
				}, nil)
				rt.On("Create", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil)
			},
			wantErr:   nil,
			wantToken: true,
		},
		{
			name: "wrong password",
			input: LoginInput{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			setup: func(cr *MockCustomerRepo, rt *MockRefreshTokenRepo) {
				cr.On("GetByEmail", mock.Anything, "test@example.com").Return(&domain.Customer{
					ID:           "cust-1",
					Email:        "test@example.com",
					PasswordHash: string(hashedPw),
				}, nil)
			},
			wantErr:   ErrInvalidCredentials,
			wantToken: false,
		},
		{
			name: "customer not found",
			input: LoginInput{
				Email:    "nobody@example.com",
				Password: "password123",
			},
			setup: func(cr *MockCustomerRepo, rt *MockRefreshTokenRepo) {
				cr.On("GetByEmail", mock.Anything, "nobody@example.com").Return(nil, nil)
			},
			wantErr:   ErrInvalidCredentials,
			wantToken: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, customerRepo, refreshTokenRepo := newTestAuthService()
			tt.setup(customerRepo, refreshTokenRepo)

			tokens, err := svc.Login(context.Background(), tt.input)

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

func TestAuthService_RefreshTokens(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		setup     func(*MockCustomerRepo, *MockRefreshTokenRepo)
		wantErr   error
		wantToken bool
	}{
		{
			name:  "success",
			token: "valid-refresh-token",
			setup: func(cr *MockCustomerRepo, rt *MockRefreshTokenRepo) {
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
			setup: func(cr *MockCustomerRepo, rt *MockRefreshTokenRepo) {
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
