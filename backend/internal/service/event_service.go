package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/facebook"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

type EventService struct {
	eventRepo    repository.EventRepository
	pixelRepo    repository.PixelRepository
	capiClient   *facebook.CAPIClient
	logger       *slog.Logger
	quotaService *QuotaService
	capiSem      chan struct{} // limits concurrent CAPI goroutines
}

func NewEventService(
	eventRepo repository.EventRepository,
	pixelRepo repository.PixelRepository,
	capiClient *facebook.CAPIClient,
	logger *slog.Logger,
	quotaService *QuotaService,
) *EventService {
	return &EventService{
		eventRepo:    eventRepo,
		pixelRepo:    pixelRepo,
		capiClient:   capiClient,
		logger:       logger,
		quotaService: quotaService,
		capiSem:      make(chan struct{}, 50),
	}
}

// ClientContext holds HTTP request data for enriching CAPI user_data.
type ClientContext struct {
	IP        string
	UserAgent string
	FBC       string // _fbc cookie (Facebook Click ID)
	FBP       string // _fbp cookie (Facebook Browser ID)
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

func (s *EventService) Ingest(ctx context.Context, customerID string, input IngestBatchInput, client ClientContext) (int, error) {
	// Atomically check quota and increment usage counter
	if s.quotaService != nil {
		if err := s.quotaService.CheckAndIncrementEventQuota(ctx, customerID, int64(len(input.Events))); err != nil {
			return 0, err
		}
	}

	// Batch fetch all unique pixel IDs to avoid N+1 queries
	uniqueIDs := make(map[string]struct{})
	for _, e := range input.Events {
		uniqueIDs[e.PixelID] = struct{}{}
	}
	ids := make([]string, 0, len(uniqueIDs))
	for id := range uniqueIDs {
		ids = append(ids, id)
	}
	pixels, err := s.pixelRepo.GetByIDs(ctx, ids)
	if err != nil {
		s.logger.Error("batch get pixels failed", "error", err)
		return 0, fmt.Errorf("batch get pixels: %w", err)
	}
	pixelMap := make(map[string]*domain.Pixel, len(pixels))
	for _, p := range pixels {
		pixelMap[p.ID] = p
	}

	created := 0
	for _, eventInput := range input.Events {
		// Verify pixel belongs to customer via pre-fetched map
		pixel := pixelMap[eventInput.PixelID]
		if pixel == nil || pixel.CustomerID != customerID || !pixel.IsActive {
			s.logger.Warn("pixel not found or inactive", "pixel_id", eventInput.PixelID)
			continue
		}

		// Skip events with oversized payloads
		if len(eventInput.EventData) > maxEventDataSize || len(eventInput.UserData) > maxUserDataSize {
			s.logger.Warn("event payload too large, skipping",
				"pixel_id", eventInput.PixelID,
				"event_data_size", len(eventInput.EventData),
				"user_data_size", len(eventInput.UserData),
			)
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

		// Strip port from client IP
		ip := client.IP
		if host, _, err := net.SplitHostPort(client.IP); err == nil {
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
			ClientUserAgent: client.UserAgent,
		}

		inserted, err := s.eventRepo.Create(ctx, event)
		if err != nil {
			s.logger.Error("create event failed", "error", err)
			continue
		}
		if !inserted {
			s.logger.Info("duplicate event skipped", "event_id", event.EventID, "pixel_id", event.PixelID)
			continue
		}

		// Forward to Facebook CAPI asynchronously (semaphore-limited)
		go func(evt *domain.PixelEvent, px *domain.Pixel, cl ClientContext) {
			s.capiSem <- struct{}{}
			defer func() { <-s.capiSem }()
			s.forwardToCAPI(context.Background(), evt, px, cl)
		}(event, pixel, client)

		// Fan-out to backup pixel (best-effort, 1 level only, semaphore-limited)
		if pixel.BackupPixelID != nil {
			go func(evt *domain.PixelEvent, backupID string, cl ClientContext) {
				s.capiSem <- struct{}{}
				defer func() { <-s.capiSem }()
				s.forwardToBackupPixel(context.Background(), evt, backupID, cl)
			}(event, *pixel.BackupPixelID, client)
		}

		created++
	}

	// Refund quota for skipped events (duplicates, invalid pixels, etc.)
	skipped := len(input.Events) - created
	if skipped > 0 && s.quotaService != nil {
		if err := s.quotaService.DecrementEventQuota(ctx, customerID, int64(skipped)); err != nil {
			s.logger.Error("failed to refund skipped event quota", "error", err, "skipped", skipped)
		}
	}

	return created, nil
}

func (s *EventService) forwardToCAPI(ctx context.Context, event *domain.PixelEvent, pixel *domain.Pixel, client ClientContext) {
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

	// Facebook Click ID — priority: SDK > cookie > fbclid in URL
	if _, exists := userData["fbc"]; !exists {
		if isValidFBCookie(client.FBC) {
			userData["fbc"] = client.FBC
		} else if fbclid := extractFBClid(event.SourceURL); fbclid != "" {
			userData["fbc"] = fmt.Sprintf("fb.1.%d.%s", event.EventTime.UnixMilli(), fbclid)
		}
	}

	// Facebook Browser ID — priority: SDK > cookie
	if _, exists := userData["fbp"]; !exists {
		if isValidFBCookie(client.FBP) {
			userData["fbp"] = client.FBP
		}
	}

	// Default country for Thailand
	if _, exists := userData["country"]; !exists {
		userData["country"] = defaultCountry
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

	testEventCode := ""
	if pixel.TestEventCode != nil {
		testEventCode = *pixel.TestEventCode
	}

	resp, err := facebook.SendEventWithRetry(ctx, s.capiClient, pixel.FBPixelID, pixel.FBAccessToken, testEventCode, capiEvent, maxCAPIRetries, s.logger)
	if err != nil {
		s.logger.Error("forward to CAPI failed", "error", err, "event_id", event.ID)
		return
	}

	if resp != nil {
		if err := s.eventRepo.MarkForwarded(ctx, event.ID, 200, resp.EventsReceived); err != nil {
			s.logger.Error("mark forwarded failed", "error", err, "event_id", event.ID)
		}
		s.logger.Info("forwarded to CAPI", "event_id", event.ID, "events_received", resp.EventsReceived)
	}
}

func (s *EventService) forwardToBackupPixel(ctx context.Context, event *domain.PixelEvent, backupPixelID string, client ClientContext) {
	backupPixel, err := s.pixelRepo.GetByID(ctx, backupPixelID)
	if err != nil {
		s.logger.Error("get backup pixel failed", "backup_pixel_id", backupPixelID, "error", err)
		return
	}
	if backupPixel == nil || !backupPixel.IsActive {
		s.logger.Warn("backup pixel not available", "backup_pixel_id", backupPixelID)
		return
	}
	// Don't recurse: backup's backup is NOT forwarded (1 level only)
	s.forwardToCAPI(ctx, event, backupPixel, client)
}

func (s *EventService) ListByCustomerID(ctx context.Context, customerID string, pixelID string, page, perPage int) ([]*domain.PixelEvent, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}
	offset := (page - 1) * perPage

	events, total, err := s.eventRepo.ListByCustomerID(ctx, customerID, pixelID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list events: %w", err)
	}
	if events == nil {
		events = []*domain.PixelEvent{}
	}
	return events, total, nil
}

func (s *EventService) ListLatest(ctx context.Context, customerID, pixelID string, limit int) ([]*domain.RealtimeEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}
	events, err := s.eventRepo.ListLatestByCustomerID(ctx, customerID, pixelID, limit)
	if err != nil {
		return nil, fmt.Errorf("list latest events: %w", err)
	}
	if events == nil {
		return []*domain.RealtimeEvent{}, nil
	}
	return events, nil
}

const (
	maxCAPIRetries     = 3   // shared retry count for CAPI calls
	realtimeEventLimit = 50
	maxFBCookieLen     = 500 // max length for _fbc/_fbp cookie values
	maxFBClidLen       = 200 // max length for fbclid URL parameter
	defaultCountry     = "th" // Pixlinks v1 operates in Thailand only
	maxEventDataSize   = 64 * 1024 // 64 KB per event_data field
	maxUserDataSize    = 16 * 1024 // 16 KB per user_data field
)

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

func (s *EventService) GetByID(ctx context.Context, customerID, id string) (*domain.PixelEvent, error) {
	event, err := s.eventRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}
	if event == nil {
		return nil, nil
	}

	// Ownership check: verify the event's pixel belongs to the customer
	pixel, err := s.pixelRepo.GetByID(ctx, event.PixelID)
	if err != nil {
		return nil, fmt.Errorf("get pixel for ownership check: %w", err)
	}
	if pixel == nil || pixel.CustomerID != customerID {
		return nil, nil
	}

	return event, nil
}

// isValidFBCookie checks that a Facebook cookie value has the expected "fb." prefix
// and does not exceed the maximum allowed length.
func isValidFBCookie(value string) bool {
	return value != "" && len(value) <= maxFBCookieLen && strings.HasPrefix(value, "fb.")
}

// extractFBClid extracts the fbclid query parameter from an HTTP(S) URL.
func extractFBClid(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	fbclid := u.Query().Get("fbclid")
	if len(fbclid) > maxFBClidLen {
		return ""
	}
	return fbclid
}
