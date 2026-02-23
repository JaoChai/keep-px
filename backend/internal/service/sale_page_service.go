package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

var (
	ErrSalePageNotFound = errors.New("sale page not found")
	ErrSalePageNotOwned = errors.New("sale page not owned by customer")
	ErrSlugTaken        = errors.New("slug is already taken")
	ErrInvalidSlug      = errors.New("slug must contain only lowercase letters, numbers, and hyphens")
	ErrInvalidContent   = errors.New("invalid page content structure")
)

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

var reservedSlugs = map[string]bool{
	"api":       true,
	"admin":     true,
	"login":     true,
	"register":  true,
	"dashboard": true,
	"health":    true,
	"p":         true,
}

type SalePageService struct {
	salePageRepo repository.SalePageRepository
	customerRepo repository.CustomerRepository
}

func NewSalePageService(salePageRepo repository.SalePageRepository, customerRepo repository.CustomerRepository) *SalePageService {
	return &SalePageService{
		salePageRepo: salePageRepo,
		customerRepo: customerRepo,
	}
}

type CreateSalePageInput struct {
	Name         string          `json:"name" validate:"required"`
	Slug         string          `json:"slug" validate:"required,min=2,max=100"`
	PixelID      *string         `json:"pixel_id,omitempty"`
	TemplateName string          `json:"template_name" validate:"required"`
	Content      json.RawMessage `json:"content" validate:"required"`
}

type UpdateSalePageInput struct {
	Name         *string          `json:"name,omitempty"`
	Slug         *string          `json:"slug,omitempty"`
	PixelID      *string          `json:"pixel_id,omitempty"`
	TemplateName *string          `json:"template_name,omitempty"`
	Content      *json.RawMessage `json:"content,omitempty"`
	IsPublished  *bool            `json:"is_published,omitempty"`
}

type SalePagePublishData struct {
	Page   *domain.SalePage
	APIKey string
}

func validateSlug(slug string) error {
	if !slugRegex.MatchString(slug) {
		return ErrInvalidSlug
	}
	if reservedSlugs[slug] {
		return ErrSlugTaken
	}
	return nil
}

func (s *SalePageService) Create(ctx context.Context, customerID string, input CreateSalePageInput) (*domain.SalePage, error) {
	if err := validateSlug(input.Slug); err != nil {
		return nil, err
	}

	// Validate content structure
	var content domain.SimpleContent
	if err := json.Unmarshal(input.Content, &content); err != nil {
		return nil, ErrInvalidContent
	}

	exists, err := s.salePageRepo.SlugExists(ctx, input.Slug)
	if err != nil {
		return nil, fmt.Errorf("check slug: %w", err)
	}
	if exists {
		return nil, ErrSlugTaken
	}

	var pixelID *string
	if input.PixelID != nil && *input.PixelID != "" {
		pixelID = input.PixelID
	}

	page := &domain.SalePage{
		CustomerID:   customerID,
		PixelID:      pixelID,
		Name:         input.Name,
		Slug:         input.Slug,
		TemplateName: input.TemplateName,
		Content:      input.Content,
	}

	if err := s.salePageRepo.Create(ctx, page); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrSlugTaken
		}
		return nil, fmt.Errorf("create sale page: %w", err)
	}
	return page, nil
}

func (s *SalePageService) GetByID(ctx context.Context, customerID, pageID string) (*domain.SalePage, error) {
	page, err := s.salePageRepo.GetByID(ctx, pageID)
	if err != nil {
		return nil, fmt.Errorf("get sale page: %w", err)
	}
	if page == nil {
		return nil, ErrSalePageNotFound
	}
	if page.CustomerID != customerID {
		return nil, ErrSalePageNotOwned
	}
	return page, nil
}

func (s *SalePageService) List(ctx context.Context, customerID string) ([]*domain.SalePage, error) {
	pages, err := s.salePageRepo.ListByCustomerID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("list sale pages: %w", err)
	}
	if pages == nil {
		pages = []*domain.SalePage{}
	}
	return pages, nil
}

func (s *SalePageService) Update(ctx context.Context, customerID, pageID string, input UpdateSalePageInput) (*domain.SalePage, error) {
	page, err := s.salePageRepo.GetByID(ctx, pageID)
	if err != nil {
		return nil, fmt.Errorf("get sale page: %w", err)
	}
	if page == nil {
		return nil, ErrSalePageNotFound
	}
	if page.CustomerID != customerID {
		return nil, ErrSalePageNotOwned
	}

	if input.Name != nil {
		page.Name = *input.Name
	}
	if input.Slug != nil && *input.Slug != page.Slug {
		if err := validateSlug(*input.Slug); err != nil {
			return nil, err
		}
		exists, err := s.salePageRepo.SlugExists(ctx, *input.Slug)
		if err != nil {
			return nil, fmt.Errorf("check slug: %w", err)
		}
		if exists {
			return nil, ErrSlugTaken
		}
		page.Slug = *input.Slug
	}
	if input.PixelID != nil {
		if *input.PixelID == "" {
			page.PixelID = nil
		} else {
			page.PixelID = input.PixelID
		}
	}
	if input.TemplateName != nil {
		page.TemplateName = *input.TemplateName
	}
	if input.Content != nil {
		page.Content = *input.Content
	}
	if input.IsPublished != nil {
		page.IsPublished = *input.IsPublished
	}

	if err := s.salePageRepo.Update(ctx, page); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrSlugTaken
		}
		return nil, fmt.Errorf("update sale page: %w", err)
	}
	return page, nil
}

func (s *SalePageService) Delete(ctx context.Context, customerID, pageID string) error {
	page, err := s.salePageRepo.GetByID(ctx, pageID)
	if err != nil {
		return fmt.Errorf("get sale page: %w", err)
	}
	if page == nil {
		return ErrSalePageNotFound
	}
	if page.CustomerID != customerID {
		return ErrSalePageNotOwned
	}
	if err := s.salePageRepo.Delete(ctx, pageID); err != nil {
		return fmt.Errorf("delete sale page: %w", err)
	}
	return nil
}

func (s *SalePageService) GetBySlug(ctx context.Context, slug string) (*domain.SalePage, error) {
	page, err := s.salePageRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("get sale page by slug: %w", err)
	}
	if page == nil {
		return nil, ErrSalePageNotFound
	}
	if !page.IsPublished {
		return nil, ErrSalePageNotFound
	}
	return page, nil
}

func (s *SalePageService) GetPublishData(ctx context.Context, slug string) (*SalePagePublishData, error) {
	page, err := s.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	customer, err := s.customerRepo.GetByID(ctx, page.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}
	if customer == nil {
		return nil, fmt.Errorf("customer not found")
	}

	return &SalePagePublishData{
		Page:   page,
		APIKey: customer.APIKey,
	}, nil
}
