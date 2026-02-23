package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

var (
	ErrPixelNotFound = errors.New("pixel not found")
	ErrPixelNotOwned = errors.New("pixel not owned by customer")
)

type PixelService struct {
	pixelRepo repository.PixelRepository
}

func NewPixelService(pixelRepo repository.PixelRepository) *PixelService {
	return &PixelService{pixelRepo: pixelRepo}
}

type CreatePixelInput struct {
	FBPixelID     string `json:"fb_pixel_id" validate:"required"`
	FBAccessToken string `json:"fb_access_token" validate:"required"`
	Name          string `json:"name" validate:"required"`
}

type UpdatePixelInput struct {
	FBPixelID     *string `json:"fb_pixel_id,omitempty"`
	FBAccessToken *string `json:"fb_access_token,omitempty"`
	Name          *string `json:"name,omitempty"`
	IsActive      *bool   `json:"is_active,omitempty"`
}

func (s *PixelService) Create(ctx context.Context, customerID string, input CreatePixelInput) (*domain.Pixel, error) {
	pixel := &domain.Pixel{
		CustomerID:    customerID,
		FBPixelID:     input.FBPixelID,
		FBAccessToken: input.FBAccessToken,
		Name:          input.Name,
	}

	if err := s.pixelRepo.Create(ctx, pixel); err != nil {
		return nil, fmt.Errorf("create pixel: %w", err)
	}
	return pixel, nil
}

func (s *PixelService) GetByID(ctx context.Context, customerID, pixelID string) (*domain.Pixel, error) {
	pixel, err := s.pixelRepo.GetByID(ctx, pixelID)
	if err != nil {
		return nil, fmt.Errorf("get pixel: %w", err)
	}
	if pixel == nil {
		return nil, ErrPixelNotFound
	}
	if pixel.CustomerID != customerID {
		return nil, ErrPixelNotOwned
	}
	return pixel, nil
}

func (s *PixelService) List(ctx context.Context, customerID string) ([]*domain.Pixel, error) {
	pixels, err := s.pixelRepo.ListByCustomerID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("list pixels: %w", err)
	}
	if pixels == nil {
		pixels = []*domain.Pixel{}
	}
	return pixels, nil
}

func (s *PixelService) Update(ctx context.Context, customerID, pixelID string, input UpdatePixelInput) (*domain.Pixel, error) {
	pixel, err := s.pixelRepo.GetByID(ctx, pixelID)
	if err != nil {
		return nil, fmt.Errorf("get pixel: %w", err)
	}
	if pixel == nil {
		return nil, ErrPixelNotFound
	}
	if pixel.CustomerID != customerID {
		return nil, ErrPixelNotOwned
	}

	if input.FBPixelID != nil {
		pixel.FBPixelID = *input.FBPixelID
	}
	if input.FBAccessToken != nil {
		pixel.FBAccessToken = *input.FBAccessToken
	}
	if input.Name != nil {
		pixel.Name = *input.Name
	}
	if input.IsActive != nil {
		pixel.IsActive = *input.IsActive
		if !*input.IsActive {
			pixel.Status = "paused"
		} else {
			pixel.Status = "active"
		}
	}

	if err := s.pixelRepo.Update(ctx, pixel); err != nil {
		return nil, fmt.Errorf("update pixel: %w", err)
	}
	return pixel, nil
}

func (s *PixelService) Delete(ctx context.Context, customerID, pixelID string) error {
	pixel, err := s.pixelRepo.GetByID(ctx, pixelID)
	if err != nil {
		return fmt.Errorf("get pixel: %w", err)
	}
	if pixel == nil {
		return ErrPixelNotFound
	}
	if pixel.CustomerID != customerID {
		return ErrPixelNotOwned
	}
	return s.pixelRepo.Delete(ctx, pixelID)
}
