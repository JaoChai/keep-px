package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/idtoken"

	"github.com/jaochai/pixlinks/backend/internal/config"
	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrEmailAlreadyExists  = errors.New("email already exists")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrInvalidGoogleToken  = errors.New("invalid google token")
	ErrAccountSuspended    = errors.New("account suspended")
)

type AuthService struct {
	customerRepo     repository.CustomerRepository
	refreshTokenRepo repository.RefreshTokenRepository
	cfg              *config.Config
}

func NewAuthService(
	customerRepo repository.CustomerRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	cfg *config.Config,
) *AuthService {
	return &AuthService{
		customerRepo:     customerRepo,
		refreshTokenRepo: refreshTokenRepo,
		cfg:              cfg,
	}
}

type AuthTokens struct {
	AccessToken  string           `json:"access_token"`
	RefreshToken string           `json:"refresh_token"`
	Customer     *domain.Customer `json:"customer"`
}

type RegisterInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Name     string `json:"name" validate:"required"`
}

type LoginInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type GoogleAuthInput struct {
	IDToken string `json:"id_token" validate:"required"`
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

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*AuthTokens, error) {
	existing, err := s.customerRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("generate api key: %w", err)
	}

	customer := &domain.Customer{
		Email:        input.Email,
		PasswordHash: string(hashedPassword),
		Name:         input.Name,
		APIKey:       apiKey,
		Plan:         "free",
	}

	if err := s.customerRepo.Create(ctx, customer); err != nil {
		return nil, fmt.Errorf("create customer: %w", err)
	}

	return s.generateTokens(ctx, customer)
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (*AuthTokens, error) {
	customer, err := s.customerRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}
	if customer == nil {
		return nil, ErrInvalidCredentials
	}

	if customer.PasswordHash == "" {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(customer.PasswordHash), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	if customer.SuspendedAt != nil {
		return nil, ErrAccountSuspended
	}

	return s.generateTokens(ctx, customer)
}

func (s *AuthService) GoogleAuth(ctx context.Context, input GoogleAuthInput) (*AuthTokens, error) {
	payload, err := idtoken.Validate(ctx, input.IDToken, s.cfg.GoogleClientID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidGoogleToken, err)
	}

	googleID, _ := payload.Claims["sub"].(string)
	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	emailVerified, _ := payload.Claims["email_verified"].(bool)

	if googleID == "" || email == "" || !emailVerified {
		return nil, ErrInvalidGoogleToken
	}

	// Check if user already linked by Google ID
	customer, err := s.customerRepo.GetByGoogleID(ctx, googleID)
	if err != nil {
		return nil, fmt.Errorf("get by google id: %w", err)
	}
	if customer != nil {
		if customer.SuspendedAt != nil {
			return nil, ErrAccountSuspended
		}
		return s.generateTokens(ctx, customer)
	}

	// Check if user exists by email — link Google ID
	customer, err = s.customerRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("get by email: %w", err)
	}
	if customer != nil {
		if customer.SuspendedAt != nil {
			return nil, ErrAccountSuspended
		}
		customer.GoogleID = &googleID
		if err := s.customerRepo.Update(ctx, customer); err != nil {
			return nil, fmt.Errorf("link google id: %w", err)
		}
		return s.generateTokens(ctx, customer)
	}

	// New user — create without password
	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("generate api key: %w", err)
	}

	customer = &domain.Customer{
		Email:    email,
		GoogleID: &googleID,
		Name:     name,
		APIKey:   apiKey,
		Plan:     domain.PlanSandbox,
	}

	if err := s.customerRepo.Create(ctx, customer); err != nil {
		return nil, fmt.Errorf("create customer: %w", err)
	}

	return s.generateTokens(ctx, customer)
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

func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string) (*AuthTokens, error) {
	tokenHash := hashToken(refreshToken)
	customerID, _, err := s.refreshTokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	if customerID == "" {
		return nil, ErrInvalidRefreshToken
	}

	// Delete used refresh token (rotation)
	if err := s.refreshTokenRepo.DeleteByTokenHash(ctx, tokenHash); err != nil {
		return nil, fmt.Errorf("delete refresh token: %w", err)
	}

	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil || customer == nil {
		return nil, ErrInvalidRefreshToken
	}

	if customer.SuspendedAt != nil {
		return nil, ErrAccountSuspended
	}

	return s.generateTokens(ctx, customer)
}

func (s *AuthService) generateTokens(ctx context.Context, customer *domain.Customer) (*AuthTokens, error) {
	now := time.Now()

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      customer.ID,
		"email":    customer.Email,
		"is_admin": customer.IsAdmin,
		"iat":      now.Unix(),
		"exp":      now.Add(s.cfg.JWTAccessTTL).Unix(),
	})

	accessTokenStr, err := accessToken.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	refreshTokenBytes := make([]byte, 32)
	if _, err := rand.Read(refreshTokenBytes); err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	refreshTokenStr := hex.EncodeToString(refreshTokenBytes)

	expiresAt := now.Add(s.cfg.JWTRefreshTTL)
	if err := s.refreshTokenRepo.Create(ctx, customer.ID, hashToken(refreshTokenStr), expiresAt); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &AuthTokens{
		AccessToken:  accessTokenStr,
		RefreshToken: refreshTokenStr,
		Customer:     customer,
	}, nil
}

func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "pk_" + hex.EncodeToString(bytes), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
