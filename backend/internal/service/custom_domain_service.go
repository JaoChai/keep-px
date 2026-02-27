package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jaochai/pixlinks/backend/internal/cloudflare"
	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

var (
	ErrCustomDomainNotFound    = errors.New("custom domain not found")
	ErrCustomDomainNotOwned    = errors.New("custom domain not owned by customer")
	ErrDomainAlreadyExists     = errors.New("domain already exists")
	ErrInvalidDomain           = errors.New("invalid domain name")
	ErrSalePageNotPublished    = errors.New("sale page must be published")
	ErrCloudflareNotConfigured = errors.New("cloudflare integration not configured")
)

type CustomDomainService struct {
	domainRepo   repository.CustomDomainRepository
	salePageRepo repository.SalePageRepository
	cfClient     *cloudflare.Client
	cnameTarget  string
}

func NewCustomDomainService(domainRepo repository.CustomDomainRepository, salePageRepo repository.SalePageRepository, cfClient *cloudflare.Client, cnameTarget string) *CustomDomainService {
	return &CustomDomainService{
		domainRepo:   domainRepo,
		salePageRepo: salePageRepo,
		cfClient:     cfClient,
		cnameTarget:  cnameTarget,
	}
}

type CreateCustomDomainInput struct {
	Domain     string `json:"domain" validate:"required,fqdn"`
	SalePageID string `json:"sale_page_id" validate:"required,uuid"`
}

func (s *CustomDomainService) Create(ctx context.Context, customerID string, input CreateCustomDomainInput) (*domain.CustomDomain, error) {
	// Validate domain format: no protocol, no path, just hostname
	d := input.Domain
	if strings.Contains(d, "://") || strings.Contains(d, "/") || strings.Contains(d, " ") {
		return nil, ErrInvalidDomain
	}

	// Get sale page and check ownership
	page, err := s.salePageRepo.GetByID(ctx, input.SalePageID)
	if err != nil {
		return nil, fmt.Errorf("get sale page: %w", err)
	}
	if page == nil {
		return nil, ErrSalePageNotFound
	}
	if page.CustomerID != customerID {
		return nil, ErrSalePageNotOwned
	}
	if !page.IsPublished {
		return nil, ErrSalePageNotPublished
	}

	// Check domain not already taken
	existing, err := s.domainRepo.GetByDomain(ctx, d)
	if err != nil {
		return nil, fmt.Errorf("check domain: %w", err)
	}
	if existing != nil {
		return nil, ErrDomainAlreadyExists
	}

	// Generate verification token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("generate verification token: %w", err)
	}
	verificationToken := hex.EncodeToString(tokenBytes)

	customDomain := &domain.CustomDomain{
		CustomerID:        customerID,
		SalePageID:        input.SalePageID,
		Domain:            d,
		VerificationToken: verificationToken,
		DNSVerified:       false,
		SSLActive:         false,
	}

	// If Cloudflare is configured, create custom hostname and store KV mapping
	if s.cfClient != nil {
		cfResp, err := s.cfClient.CreateCustomHostname(ctx, d)
		if err != nil {
			return nil, fmt.Errorf("cloudflare create hostname: %w", err)
		}
		customDomain.CFHostnameID = &cfResp.ID

		// Store domain→slug mapping in KV
		kvValue, err := json.Marshal(map[string]string{
			"slug":         page.Slug,
			"sale_page_id": page.ID,
			"customer_id":  customerID,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal kv value: %w", err)
		}
		if err := s.cfClient.PutKVValue(ctx, d, kvValue); err != nil {
			return nil, fmt.Errorf("cloudflare put kv: %w", err)
		}
	}

	// Save to DB
	if err := s.domainRepo.Create(ctx, customDomain); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrDomainAlreadyExists
		}
		return nil, fmt.Errorf("create custom domain: %w", err)
	}

	return customDomain, nil
}

func (s *CustomDomainService) List(ctx context.Context, customerID string) ([]*domain.CustomDomain, error) {
	domains, err := s.domainRepo.ListByCustomerID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("list custom domains: %w", err)
	}
	if domains == nil {
		domains = []*domain.CustomDomain{}
	}
	return domains, nil
}

func (s *CustomDomainService) GetByID(ctx context.Context, customerID, id string) (*domain.CustomDomain, error) {
	d, err := s.domainRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get custom domain: %w", err)
	}
	if d == nil {
		return nil, ErrCustomDomainNotFound
	}
	if d.CustomerID != customerID {
		return nil, ErrCustomDomainNotOwned
	}
	return d, nil
}

func (s *CustomDomainService) Verify(ctx context.Context, customerID, id string) (*domain.CustomDomain, error) {
	d, err := s.domainRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get custom domain: %w", err)
	}
	if d == nil {
		return nil, ErrCustomDomainNotFound
	}
	if d.CustomerID != customerID {
		return nil, ErrCustomDomainNotOwned
	}

	if s.cfClient == nil {
		return nil, ErrCloudflareNotConfigured
	}

	if d.CFHostnameID == nil || *d.CFHostnameID == "" {
		return nil, ErrCloudflareNotConfigured
	}

	// Check status with Cloudflare
	cfResp, err := s.cfClient.GetCustomHostname(ctx, *d.CFHostnameID)
	if err != nil {
		return nil, fmt.Errorf("cloudflare get hostname: %w", err)
	}

	// Update DNS and SSL status based on CF response
	d.DNSVerified = cfResp.Status == "active"
	d.SSLActive = cfResp.SSL.Status == "active"

	if d.DNSVerified && d.VerifiedAt == nil {
		now := time.Now()
		d.VerifiedAt = &now
	}

	if err := s.domainRepo.Update(ctx, d); err != nil {
		return nil, fmt.Errorf("update custom domain: %w", err)
	}

	return d, nil
}

func (s *CustomDomainService) Delete(ctx context.Context, customerID, id string) error {
	d, err := s.domainRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get custom domain: %w", err)
	}
	if d == nil {
		return ErrCustomDomainNotFound
	}
	if d.CustomerID != customerID {
		return ErrCustomDomainNotOwned
	}

	// Clean up Cloudflare resources if configured
	if s.cfClient != nil {
		if d.CFHostnameID != nil && *d.CFHostnameID != "" {
			if err := s.cfClient.DeleteCustomHostname(ctx, *d.CFHostnameID); err != nil {
				return fmt.Errorf("cloudflare delete hostname: %w", err)
			}
		}
		if err := s.cfClient.DeleteKVValue(ctx, d.Domain); err != nil {
			return fmt.Errorf("cloudflare delete kv: %w", err)
		}
	}

	if err := s.domainRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete custom domain: %w", err)
	}
	return nil
}
