package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jaochai/pixlinks/backend/internal/cloudflare"
	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

const cfStatusActive = "active"

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

// validateCustomDomain checks that the domain is a valid, non-platform hostname.
func validateCustomDomain(d string) error {
	// Reject protocol, path, or whitespace
	if strings.Contains(d, "://") || strings.Contains(d, "/") || strings.Contains(d, " ") {
		return ErrInvalidDomain
	}

	// Must have at least one dot (no single-label domains)
	if !strings.Contains(d, ".") {
		return ErrInvalidDomain
	}

	// Reject platform domains
	if d == "pixlinks.xyz" || d == "app.pixlinks.xyz" || d == "www.pixlinks.xyz" ||
		strings.HasSuffix(d, ".pixlinks.xyz") {
		return ErrInvalidDomain
	}

	// Reject localhost
	if d == "localhost" || strings.HasSuffix(d, ".localhost") {
		return ErrInvalidDomain
	}

	// Reject IP addresses
	if net.ParseIP(d) != nil {
		return ErrInvalidDomain
	}

	return nil
}

func (s *CustomDomainService) Create(ctx context.Context, customerID string, input CreateCustomDomainInput) (*domain.CustomDomain, error) {
	// 1. Validate domain
	d := strings.ToLower(strings.TrimSpace(input.Domain))
	if err := validateCustomDomain(d); err != nil {
		return nil, err
	}

	// 2. Get sale page and check ownership + published
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

	// 3. Check domain not already taken
	existing, err := s.domainRepo.GetByDomain(ctx, d)
	if err != nil {
		return nil, fmt.Errorf("check domain: %w", err)
	}
	if existing != nil {
		return nil, ErrDomainAlreadyExists
	}

	// 4. Generate verification token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("generate verification token: %w", err)
	}
	verificationToken := hex.EncodeToString(tokenBytes)

	// 5. Create domain in DB FIRST (catches race condition via UNIQUE constraint)
	customDomain := &domain.CustomDomain{
		CustomerID:        customerID,
		SalePageID:        input.SalePageID,
		Domain:            d,
		VerificationToken: verificationToken,
		DNSVerified:       false,
		SSLActive:         false,
	}

	if err := s.domainRepo.Create(ctx, customDomain); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrDomainAlreadyExists
		}
		return nil, fmt.Errorf("create custom domain: %w", err)
	}

	// 6. THEN create CF resources (if configured)
	if s.cfClient != nil {
		cfResp, err := s.cfClient.CreateCustomHostname(ctx, d)
		if err != nil {
			// Rollback: delete DB record
			_ = s.domainRepo.Delete(ctx, customDomain.ID)
			return nil, fmt.Errorf("cloudflare create hostname: %w", err)
		}
		customDomain.CFHostnameID = &cfResp.ID

		kvData, _ := json.Marshal(map[string]string{
			"slug":         page.Slug,
			"sale_page_id": page.ID,
			"customer_id":  customerID,
		})
		if err := s.cfClient.PutKVValue(ctx, d, kvData); err != nil {
			// Rollback: delete CF hostname + DB record
			_ = s.cfClient.DeleteCustomHostname(ctx, cfResp.ID)
			_ = s.domainRepo.Delete(ctx, customDomain.ID)
			return nil, fmt.Errorf("cloudflare put kv: %w", err)
		}

		// Update DB with CF hostname ID
		if err := s.domainRepo.Update(ctx, customDomain); err != nil {
			// Best-effort cleanup
			_ = s.cfClient.DeleteCustomHostname(ctx, cfResp.ID)
			_ = s.cfClient.DeleteKVValue(ctx, d)
			_ = s.domainRepo.Delete(ctx, customDomain.ID)
			return nil, fmt.Errorf("update custom domain: %w", err)
		}
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
	d.DNSVerified = cfResp.Status == cfStatusActive
	d.SSLActive = cfResp.SSL.Status == cfStatusActive

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
