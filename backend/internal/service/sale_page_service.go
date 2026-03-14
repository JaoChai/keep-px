package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"regexp"
	"strings"
	"time"

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
var hexColorRegex = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

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
	pixelRepo    repository.PixelRepository
	quotaService *QuotaService
	cache        *salePageCache
}

func NewSalePageService(ctx context.Context, salePageRepo repository.SalePageRepository, customerRepo repository.CustomerRepository, pixelRepo repository.PixelRepository, quotaService *QuotaService, cacheTTL time.Duration) *SalePageService {
	return &SalePageService{
		salePageRepo: salePageRepo,
		customerRepo: customerRepo,
		pixelRepo:    pixelRepo,
		quotaService: quotaService,
		cache:        newSalePageCache(ctx, cacheTTL),
	}
}

type CreateSalePageInput struct {
	Name         string          `json:"name" validate:"required"`
	Slug         string          `json:"slug" validate:"omitempty,min=2,max=100"`
	PixelIDs     []string        `json:"pixel_ids,omitempty"`
	TemplateName string          `json:"template_name" validate:"required"`
	Content      json.RawMessage `json:"content" validate:"required"`
	IsPublished  bool            `json:"is_published"`
}

type UpdateSalePageInput struct {
	Name         *string          `json:"name,omitempty"`
	Slug         *string          `json:"slug,omitempty"`
	PixelIDs     *[]string        `json:"pixel_ids,omitempty"`
	TemplateName *string          `json:"template_name,omitempty"`
	Content      *json.RawMessage `json:"content,omitempty"`
	IsPublished  *bool            `json:"is_published,omitempty"`
}

type PixelPublishInfo struct {
	PixelID   string
	FBPixelID string
}

type SalePagePublishData struct {
	Page   *domain.SalePage
	APIKey string
	Pixels []PixelPublishInfo
}

func generateRandomSlug() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	max := big.NewInt(int64(len(charset)))
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = charset[n.Int64()]
	}
	return "p-" + string(b), nil
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

func (s *SalePageService) validatePixelOwnership(ctx context.Context, customerID string, pixelIDs []string) error {
	if len(pixelIDs) == 0 {
		return nil
	}
	pixels, err := s.pixelRepo.GetByIDs(ctx, pixelIDs)
	if err != nil {
		return fmt.Errorf("get pixels: %w", err)
	}
	found := make(map[string]*domain.Pixel, len(pixels))
	for _, p := range pixels {
		found[p.ID] = p
	}
	for _, pid := range pixelIDs {
		pixel, ok := found[pid]
		if !ok {
			return ErrPixelNotFound
		}
		if pixel.CustomerID != customerID {
			return ErrPixelNotOwned
		}
	}
	return nil
}

func (s *SalePageService) Create(ctx context.Context, customerID string, input CreateSalePageInput) (*domain.SalePage, error) {
	// Check sale page creation quota
	if s.quotaService != nil {
		if err := s.quotaService.CheckSalePageCreationQuota(ctx, customerID); err != nil {
			return nil, err
		}
	}

	if err := validateContent(input.Content); err != nil {
		return nil, err
	}

	if input.Slug == "" {
		// Auto-generate random slug
		for i := 0; i < 5; i++ {
			generated, err := generateRandomSlug()
			if err != nil {
				return nil, fmt.Errorf("generate slug: %w", err)
			}
			exists, err := s.salePageRepo.SlugExists(ctx, generated)
			if err != nil {
				return nil, fmt.Errorf("check slug: %w", err)
			}
			if !exists {
				input.Slug = generated
				break
			}
		}
		if input.Slug == "" {
			return nil, fmt.Errorf("failed to generate unique slug")
		}
	} else {
		if err := validateSlug(input.Slug); err != nil {
			return nil, err
		}
		exists, err := s.salePageRepo.SlugExists(ctx, input.Slug)
		if err != nil {
			return nil, fmt.Errorf("check slug: %w", err)
		}
		if exists {
			return nil, ErrSlugTaken
		}
	}

	if input.PixelIDs == nil {
		input.PixelIDs = []string{}
	}
	if len(input.PixelIDs) > 0 {
		if err := s.validatePixelOwnership(ctx, customerID, input.PixelIDs); err != nil {
			return nil, err
		}
	}

	page := &domain.SalePage{
		CustomerID:   customerID,
		PixelIDs:     input.PixelIDs,
		Name:         input.Name,
		Slug:         input.Slug,
		TemplateName: input.TemplateName,
		Content:      input.Content,
		IsPublished:  input.IsPublished,
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

func (s *SalePageService) List(ctx context.Context, customerID string, page, perPage int) ([]*domain.SalePage, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage
	pages, total, err := s.salePageRepo.ListByCustomerID(ctx, customerID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list sale pages: %w", err)
	}
	if pages == nil {
		pages = []*domain.SalePage{}
	}
	return pages, total, nil
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

	oldSlug := page.Slug

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
	if input.PixelIDs != nil {
		if len(*input.PixelIDs) > 0 {
			if err := s.validatePixelOwnership(ctx, customerID, *input.PixelIDs); err != nil {
				return nil, err
			}
		}
		page.PixelIDs = *input.PixelIDs
	}
	if input.TemplateName != nil {
		page.TemplateName = *input.TemplateName
	}
	if input.Content != nil {
		if err := validateContent(*input.Content); err != nil {
			return nil, err
		}
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

	s.cache.Invalidate(oldSlug)
	if page.Slug != oldSlug {
		s.cache.Invalidate(page.Slug)
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
	s.cache.Invalidate(page.Slug)
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
	if cached, ok := s.cache.Get(slug); ok {
		return cached, nil
	}

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

	var pixels []PixelPublishInfo
	if len(page.PixelIDs) > 0 {
		fetched, err := s.pixelRepo.GetByIDs(ctx, page.PixelIDs)
		if err != nil {
			return nil, fmt.Errorf("get pixels: %w", err)
		}
		for _, p := range fetched {
			pixels = append(pixels, PixelPublishInfo{
				PixelID:   p.ID,
				FBPixelID: p.FBPixelID,
			})
		}
	}

	data := &SalePagePublishData{
		Page:   page,
		APIKey: customer.APIKey,
		Pixels: pixels,
	}
	s.cache.Set(slug, data)
	return data, nil
}

func validateSafeURL(raw string) error {
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%w: invalid URL", ErrInvalidContent)
	}
	if u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: URL must use http or https scheme", ErrInvalidContent)
	}
	return nil
}

func validatePageStyle(style domain.PageStyle) error {
	if style.BgColor != "" && !hexColorRegex.MatchString(style.BgColor) {
		return fmt.Errorf("%w: bg_color must be a valid hex color (e.g. #ffffff)", ErrInvalidContent)
	}
	if style.AccentColor != "" && !hexColorRegex.MatchString(style.AccentColor) {
		return fmt.Errorf("%w: accent_color must be a valid hex color (e.g. #667eea)", ErrInvalidContent)
	}
	if style.TextColor != "" && !hexColorRegex.MatchString(style.TextColor) {
		return fmt.Errorf("%w: text_color must be a valid hex color (e.g. #1a1a2e)", ErrInvalidContent)
	}
	if style.BgImageURL != "" && !strings.HasPrefix(style.BgImageURL, "https://") {
		return fmt.Errorf("%w: bg_image_url must start with https://", ErrInvalidContent)
	}
	return nil
}

func validateContent(raw json.RawMessage) error {
	var peek struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(raw, &peek); err != nil {
		return ErrInvalidContent
	}

	if peek.Version == 2 {
		var content domain.BlocksContent
		if err := json.Unmarshal(raw, &content); err != nil {
			return ErrInvalidContent
		}
		if len(content.Blocks) == 0 {
			return fmt.Errorf("%w: blocks content must have at least one block", ErrInvalidContent)
		}
		if len(content.Blocks) > 100 {
			return fmt.Errorf("%w: too many blocks (max 100)", ErrInvalidContent)
		}
		for i, b := range content.Blocks {
			if b.ID == "" {
				return fmt.Errorf("%w: block at index %d missing ID", ErrInvalidContent, i)
			}
			switch b.Type {
			case domain.BlockTypeImage:
				if err := validateSafeURL(b.ImageURL); err != nil {
					return err
				}
				if err := validateSafeURL(b.LinkURL); err != nil {
					return err
				}
			case domain.BlockTypeText:
				// text can be empty (draft)
			case domain.BlockTypeButton:
				if b.ButtonStyle != "line" && b.ButtonStyle != "website" && b.ButtonStyle != "custom" {
					return fmt.Errorf("%w: block at index %d has invalid button_style", ErrInvalidContent, i)
				}
				if b.ButtonStyle == "website" || b.ButtonStyle == "custom" {
					if err := validateSafeURL(b.ButtonURL); err != nil {
						return err
					}
				}
			default:
				return fmt.Errorf("%w: block at index %d has unknown type %q", ErrInvalidContent, i, b.Type)
			}
		}
		if err := validatePageStyle(content.Style); err != nil {
			return err
		}
		return nil
	}

	// Default: v1 SimpleContent
	var content domain.SimpleContent
	if err := json.Unmarshal(raw, &content); err != nil {
		return ErrInvalidContent
	}
	if err := validatePageStyle(content.Style); err != nil {
		return err
	}
	return nil
}
