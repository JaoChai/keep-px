package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountSuspended   = errors.New("account suspended")
	ErrEmailNotVerified   = errors.New("email not verified")
)

type AuthService struct {
	customerRepo repository.CustomerRepository
}

func NewAuthService(customerRepo repository.CustomerRepository) *AuthService {
	return &AuthService{
		customerRepo: customerRepo,
	}
}

type ProvisionInput struct {
	AuthUserID    string
	Email         string
	Name          string
	EmailVerified bool
}

func (s *AuthService) GetCustomerByID(ctx context.Context, id string) (*domain.Customer, error) {
	customer, err := s.customerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}
	if customer == nil {
		return nil, ErrInvalidCredentials
	}
	return customer, nil
}

// ProvisionCustomer หา customer ที่ผูกกับ user ของ Neon อยู่แล้ว
// ถ้ายังไม่มี ให้ผูกกับบัญชีเดิมที่ email ตรงกัน หรือสร้างใหม่
func (s *AuthService) ProvisionCustomer(ctx context.Context, in ProvisionInput) (*domain.Customer, error) {
	customer, err := s.customerRepo.GetByAuthUserID(ctx, in.AuthUserID)
	if err != nil {
		return nil, fmt.Errorf("get by auth user id: %w", err)
	}
	if customer != nil {
		if customer.SuspendedAt != nil {
			return nil, ErrAccountSuspended
		}
		return customer, nil
	}

	if !in.EmailVerified {
		return nil, ErrEmailNotVerified
	}

	customer, err = s.customerRepo.GetByEmail(ctx, in.Email)
	if err != nil {
		return nil, fmt.Errorf("get by email: %w", err)
	}
	if customer != nil {
		if customer.SuspendedAt != nil {
			return nil, ErrAccountSuspended
		}
		customer.AuthUserID = &in.AuthUserID
		if err := s.customerRepo.Update(ctx, customer); err != nil {
			return nil, fmt.Errorf("link auth user id: %w", err)
		}
		return customer, nil
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("generate api key: %w", err)
	}

	customer = &domain.Customer{
		Email:         in.Email,
		AuthUserID:    &in.AuthUserID,
		Name:          in.Name,
		APIKey:        apiKey,
		Plan:          domain.PlanSandbox,
		RetentionDays: 7,
	}
	if err := s.customerRepo.Create(ctx, customer); err != nil {
		return nil, fmt.Errorf("create customer: %w", err)
	}
	return customer, nil
}

func (s *AuthService) RegenerateAPIKey(ctx context.Context, customerID string) (*domain.Customer, error) {
	newKey, err := generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("generate api key: %w", err)
	}

	customer, err := s.customerRepo.RegenerateAPIKey(ctx, customerID, newKey)
	if err != nil {
		return nil, fmt.Errorf("regenerate api key: %w", err)
	}
	if customer == nil {
		return nil, ErrInvalidCredentials
	}

	return customer, nil
}

func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "pk_" + hex.EncodeToString(bytes), nil
}
