package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/facebook"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

var fbPixelIDRegex = regexp.MustCompile(`^\d{15,16}$`)

var (
	ErrPixelNotFound        = errors.New("pixel not found")
	ErrPixelNotOwned        = errors.New("pixel not owned by customer")
	ErrPixelNoAccessToken   = errors.New("pixel has no access token configured")
	ErrBackupPixelSelf      = errors.New("cannot set pixel as its own backup")
	ErrBackupPixelNotFound  = errors.New("backup pixel not found")
	ErrBackupPixelNotOwned  = errors.New("backup pixel not owned by customer")
	ErrInvalidFBPixelID     = errors.New("invalid Facebook Pixel ID format: must be 15-16 digits")
)

type PixelService struct {
	pixelRepo    repository.PixelRepository
	capiClient   *facebook.CAPIClient
	logger       *slog.Logger
	quotaService *QuotaService
}

func NewPixelService(pixelRepo repository.PixelRepository, capiClient *facebook.CAPIClient, logger *slog.Logger, quotaService *QuotaService) *PixelService {
	return &PixelService{
		pixelRepo:    pixelRepo,
		capiClient:   capiClient,
		logger:       logger,
		quotaService: quotaService,
	}
}

type CreatePixelInput struct {
	FBPixelID     string  `json:"fb_pixel_id" validate:"required"`
	FBAccessToken string  `json:"fb_access_token" validate:"required"`
	Name          string  `json:"name" validate:"required"`
	TestEventCode *string `json:"test_event_code,omitempty"`
	BackupPixelID *string `json:"backup_pixel_id,omitempty"`
}

type UpdatePixelInput struct {
	FBPixelID     *string `json:"fb_pixel_id,omitempty"`
	FBAccessToken *string `json:"fb_access_token,omitempty"`
	Name          *string `json:"name,omitempty"`
	IsActive      *bool   `json:"is_active,omitempty"`
	BackupPixelID *string `json:"backup_pixel_id,omitempty"`
	TestEventCode *string `json:"test_event_code,omitempty"`
}

func validateFBPixelID(id string) error {
	if !fbPixelIDRegex.MatchString(id) {
		return ErrInvalidFBPixelID
	}
	return nil
}

func (s *PixelService) validateBackupPixel(ctx context.Context, backupID, customerID string) error {
	backupPixel, err := s.pixelRepo.GetByID(ctx, backupID)
	if err != nil {
		return fmt.Errorf("get backup pixel: %w", err)
	}
	if backupPixel == nil {
		return ErrBackupPixelNotFound
	}
	if backupPixel.CustomerID != customerID {
		return ErrBackupPixelNotOwned
	}
	return nil
}

func (s *PixelService) Create(ctx context.Context, customerID string, input CreatePixelInput) (*domain.Pixel, error) {
	if err := validateFBPixelID(input.FBPixelID); err != nil {
		return nil, err
	}

	// Validate backup pixel before quota check (fail fast on invalid input)
	var backupID *string
	if input.BackupPixelID != nil && *input.BackupPixelID != "" {
		if err := s.validateBackupPixel(ctx, *input.BackupPixelID, customerID); err != nil {
			return nil, err
		}
		backupID = input.BackupPixelID
	}

	// Check pixel creation quota
	if s.quotaService != nil {
		if err := s.quotaService.CheckPixelCreationQuota(ctx, customerID); err != nil {
			return nil, err
		}
	}

	pixel := &domain.Pixel{
		CustomerID:    customerID,
		FBPixelID:     input.FBPixelID,
		FBAccessToken: input.FBAccessToken,
		Name:          input.Name,
		TestEventCode: input.TestEventCode,
		BackupPixelID: backupID,
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
		if err := validateFBPixelID(*input.FBPixelID); err != nil {
			return nil, err
		}
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
	}

	if input.TestEventCode != nil {
		if *input.TestEventCode == "" {
			pixel.TestEventCode = nil
		} else {
			pixel.TestEventCode = input.TestEventCode
		}
	}

	if input.BackupPixelID != nil {
		if *input.BackupPixelID == "" {
			// Clear backup
			pixel.BackupPixelID = nil
		} else {
			if *input.BackupPixelID == pixelID {
				return nil, ErrBackupPixelSelf
			}
			if err := s.validateBackupPixel(ctx, *input.BackupPixelID, customerID); err != nil {
				return nil, err
			}
			pixel.BackupPixelID = input.BackupPixelID
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

func (s *PixelService) TestConnection(ctx context.Context, customerID, pixelID string) (*facebook.CAPIResponse, error) {
	pixel, err := s.GetByID(ctx, customerID, pixelID)
	if err != nil {
		return nil, err
	}

	if pixel.FBAccessToken == "" {
		return nil, ErrPixelNoAccessToken
	}

	event := facebook.CAPIEvent{
		EventName:             "PageView",
		EventTime:             time.Now().Unix(),
		ActionSource:          "website",
		EventID:               uuid.NewString(),
		EventSourceURL:        "https://keepx.io/test",
		DataProcessingOptions: []string{},
		UserData: map[string]interface{}{
			"client_user_agent": "KeepPX/1.0 (Connection Test)",
			"external_id":       facebook.HashValue("keepx-test-" + pixel.FBPixelID),
		},
	}

	testEventCode := ""
	if pixel.TestEventCode != nil {
		testEventCode = *pixel.TestEventCode
	}

	resp, err := s.capiClient.SendEvent(ctx, pixel.FBPixelID, pixel.FBAccessToken, testEventCode, event)
	if err != nil {
		s.logger.WarnContext(ctx, "pixel test connection failed",
			slog.String("pixel_id", pixelID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("test connection: %w", err)
	}

	s.logger.InfoContext(ctx, "pixel test connection succeeded",
		slog.String("pixel_id", pixelID),
		slog.Int("events_received", resp.EventsReceived),
	)

	return resp, nil
}
