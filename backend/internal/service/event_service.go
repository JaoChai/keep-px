package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/facebook"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

type EventService struct {
	eventRepo  repository.EventRepository
	pixelRepo  repository.PixelRepository
	capiClient *facebook.CAPIClient
	logger     *slog.Logger
}

func NewEventService(
	eventRepo repository.EventRepository,
	pixelRepo repository.PixelRepository,
	capiClient *facebook.CAPIClient,
	logger *slog.Logger,
) *EventService {
	return &EventService{
		eventRepo:  eventRepo,
		pixelRepo:  pixelRepo,
		capiClient: capiClient,
		logger:     logger,
	}
}

type IngestEventInput struct {
	PixelID   string          `json:"pixel_id" validate:"required"`
	EventName string          `json:"event_name" validate:"required"`
	EventData json.RawMessage `json:"event_data"`
	UserData  json.RawMessage `json:"user_data,omitempty"`
	SourceURL string          `json:"source_url,omitempty"`
	EventTime string          `json:"event_time,omitempty"`
	EventID   string          `json:"event_id,omitempty"`
}

type IngestBatchInput struct {
	Events []IngestEventInput `json:"events" validate:"required,min=1,max=100"`
}

func (s *EventService) Ingest(ctx context.Context, customerID string, input IngestBatchInput, clientIP, clientUA string) (int, error) {
	created := 0
	for _, eventInput := range input.Events {
		// Verify pixel belongs to customer
		pixel, err := s.pixelRepo.GetByID(ctx, eventInput.PixelID)
		if err != nil {
			s.logger.Error("get pixel failed", "error", err, "pixel_id", eventInput.PixelID)
			continue
		}
		if pixel == nil || pixel.CustomerID != customerID || !pixel.IsActive {
			s.logger.Warn("pixel not found or inactive", "pixel_id", eventInput.PixelID)
			continue
		}

		eventTime := time.Now()
		if eventInput.EventTime != "" {
			if parsed, err := time.Parse(time.RFC3339, eventInput.EventTime); err == nil {
				eventTime = parsed
			}
		}

		eventData := eventInput.EventData
		if eventData == nil {
			eventData = json.RawMessage(`{}`)
		}

		// Strip port from clientIP
		ip := clientIP
		if host, _, err := net.SplitHostPort(clientIP); err == nil {
			ip = host
		}

		// Generate event_id if not provided
		eventID := eventInput.EventID
		if eventID == "" {
			eventID = uuid.New().String()
		}

		event := &domain.PixelEvent{
			PixelID:         eventInput.PixelID,
			EventName:       eventInput.EventName,
			EventData:       eventData,
			UserData:        eventInput.UserData,
			SourceURL:       eventInput.SourceURL,
			EventTime:       eventTime,
			EventID:         eventID,
			ClientIP:        ip,
			ClientUserAgent: clientUA,
		}

		if err := s.eventRepo.Create(ctx, event); err != nil {
			s.logger.Error("create event failed", "error", err)
			continue
		}

		// Forward to Facebook CAPI asynchronously
		go s.forwardToCAPI(context.Background(), event, pixel)

		created++
	}
	return created, nil
}

func (s *EventService) forwardToCAPI(ctx context.Context, event *domain.PixelEvent, pixel *domain.Pixel) {
	var customData map[string]interface{}
	if event.EventData != nil {
		_ = json.Unmarshal(event.EventData, &customData)
	}

	var userData map[string]interface{}
	if event.UserData != nil {
		_ = json.Unmarshal(event.UserData, &userData)
	}
	if userData == nil {
		userData = make(map[string]interface{})
	}

	// Add client context for EMQ score
	if event.ClientIP != "" {
		userData["client_ip_address"] = event.ClientIP
	}
	if event.ClientUserAgent != "" {
		userData["client_user_agent"] = event.ClientUserAgent
	}

	// Hash PII fields
	userData = facebook.HashUserData(userData)

	capiEvent := facebook.CAPIEvent{
		EventName:             event.EventName,
		EventTime:             event.EventTime.Unix(),
		UserData:              userData,
		CustomData:            customData,
		EventSourceURL:        event.SourceURL,
		ActionSource:          "website",
		EventID:               event.EventID,
		DataProcessingOptions: []string{},
	}

	resp, err := s.capiClient.SendEvent(ctx, pixel.FBPixelID, pixel.FBAccessToken, capiEvent)
	if err != nil {
		s.logger.Error("forward to CAPI failed", "error", err, "event_id", event.ID)
		return
	}

	responseCode := 200
	if resp != nil {
		_ = s.eventRepo.MarkForwarded(ctx, event.ID, responseCode)
		s.logger.Info("forwarded to CAPI", "event_id", event.ID, "events_received", resp.EventsReceived)
	}
}

func (s *EventService) ListByCustomerID(ctx context.Context, customerID string, page, perPage int) ([]*domain.PixelEvent, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}
	offset := (page - 1) * perPage

	events, total, err := s.eventRepo.ListByCustomerID(ctx, customerID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list events: %w", err)
	}
	if events == nil {
		events = []*domain.PixelEvent{}
	}
	return events, total, nil
}

const realtimeEventLimit = 50

func (s *EventService) ListRecent(ctx context.Context, customerID string, since time.Time, pixelID string) ([]*domain.RealtimeEvent, error) {
	events, err := s.eventRepo.ListRecentByCustomerID(ctx, customerID, since, pixelID, realtimeEventLimit)
	if err != nil {
		return nil, fmt.Errorf("list recent events: %w", err)
	}
	if events == nil {
		return []*domain.RealtimeEvent{}, nil
	}
	return events, nil
}

func (s *EventService) GetByID(ctx context.Context, id string) (*domain.PixelEvent, error) {
	event, err := s.eventRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}
	return event, nil
}
