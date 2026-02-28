package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/facebook"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

var (
	ErrReplayNotFound = errors.New("replay session not found")
)

type ReplayService struct {
	replayRepo repository.ReplaySessionRepository
	eventRepo  repository.EventRepository
	pixelRepo  repository.PixelRepository
	capiClient *facebook.CAPIClient
	logger     *slog.Logger
}

func NewReplayService(
	replayRepo repository.ReplaySessionRepository,
	eventRepo repository.EventRepository,
	pixelRepo repository.PixelRepository,
	capiClient *facebook.CAPIClient,
	logger *slog.Logger,
) *ReplayService {
	return &ReplayService{
		replayRepo: replayRepo,
		eventRepo:  eventRepo,
		pixelRepo:  pixelRepo,
		capiClient: capiClient,
		logger:     logger,
	}
}

type CreateReplayInput struct {
	SourcePixelID string   `json:"source_pixel_id" validate:"required"`
	TargetPixelID string   `json:"target_pixel_id" validate:"required"`
	EventTypes    []string `json:"event_types,omitempty"`
	DateFrom      string   `json:"date_from,omitempty"`
	DateTo        string   `json:"date_to,omitempty"`
	TimeMode      string   `json:"time_mode" validate:"omitempty,oneof=original current"`
	BatchDelayMs  int      `json:"batch_delay_ms" validate:"min=0,max=60000"`
}

type CreateReplayResult struct {
	Session *domain.ReplaySession `json:"session"`
	Warning string                `json:"warning,omitempty"`
}

func (s *ReplayService) Create(ctx context.Context, customerID string, input CreateReplayInput) (*CreateReplayResult, error) {
	// Verify source pixel
	sourcePixel, err := s.pixelRepo.GetByID(ctx, input.SourcePixelID)
	if err != nil || sourcePixel == nil || sourcePixel.CustomerID != customerID {
		return nil, ErrPixelNotFound
	}

	// Verify target pixel
	targetPixel, err := s.pixelRepo.GetByID(ctx, input.TargetPixelID)
	if err != nil || targetPixel == nil || targetPixel.CustomerID != customerID {
		return nil, ErrPixelNotFound
	}

	var dateFrom, dateTo *time.Time
	if input.DateFrom != "" {
		t, err := time.Parse(time.RFC3339, input.DateFrom)
		if err == nil {
			dateFrom = &t
		}
	}
	if input.DateTo != "" {
		t, err := time.Parse(time.RFC3339, input.DateTo)
		if err == nil {
			dateTo = &t
		}
	}

	// Count events to replay
	events, err := s.eventRepo.GetEventsForReplay(ctx, input.SourcePixelID, input.EventTypes, dateFrom, dateTo)
	if err != nil {
		return nil, fmt.Errorf("get events for replay: %w", err)
	}

	// Default TimeMode to "original"
	timeMode := input.TimeMode
	if timeMode == "" {
		timeMode = "original"
	}

	// Warn about old events when using original time mode
	var warning string
	if timeMode == "original" && len(events) > 0 {
		var oldest time.Time
		for _, evt := range events {
			if oldest.IsZero() || evt.EventTime.Before(oldest) {
				oldest = evt.EventTime
			}
		}
		age := time.Since(oldest)
		if age > 7*24*time.Hour {
			days := int(age.Hours() / 24)
			warning = fmt.Sprintf("Warning: oldest event is %d days old. Facebook may reject events older than 7 days. Consider using time_mode: current.", days)
		}
	}

	session := &domain.ReplaySession{
		CustomerID:    customerID,
		SourcePixelID: input.SourcePixelID,
		TargetPixelID: input.TargetPixelID,
		TotalEvents:   len(events),
		EventTypes:    input.EventTypes,
		DateFrom:      dateFrom,
		DateTo:        dateTo,
		TimeMode:      timeMode,
		BatchDelayMs:  input.BatchDelayMs,
	}

	if err := s.replayRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create replay session: %w", err)
	}

	// Start replay in background
	go s.executeReplay(context.Background(), session, targetPixel, events)

	return &CreateReplayResult{Session: session, Warning: warning}, nil
}

func (s *ReplayService) executeReplay(ctx context.Context, session *domain.ReplaySession, targetPixel *domain.Pixel, events []*domain.PixelEvent) {
	_ = s.replayRepo.UpdateStatus(ctx, session.ID, "running")

	var replayed, failed int64
	var lastErr error
	const batchSize = 1000

	for i := 0; i < len(events); i += batchSize {
		end := i + batchSize
		if end > len(events) {
			end = len(events)
		}
		batch := events[i:end]

		capiEvents := make([]facebook.CAPIEvent, 0, len(batch))
		for _, evt := range batch {
			eventTime := evt.EventTime.Unix()
			if session.TimeMode == "current" {
				eventTime = time.Now().Unix()
			}

			capiEvt := facebook.CAPIEvent{
				EventName:             evt.EventName,
				EventTime:             eventTime,
				EventSourceURL:        evt.SourceURL,
				ActionSource:          "website",
				EventID:               uuid.New().String(),
				DataProcessingOptions: []string{},
			}

			if evt.EventData != nil {
				var cd map[string]interface{}
				_ = json.Unmarshal(evt.EventData, &cd)
				capiEvt.CustomData = cd
			}

			userData := make(map[string]interface{})
			if evt.UserData != nil {
				_ = json.Unmarshal(evt.UserData, &userData)
			}
			if evt.ClientIP != "" {
				userData["client_ip_address"] = evt.ClientIP
			}
			if evt.ClientUserAgent != "" {
				userData["client_user_agent"] = evt.ClientUserAgent
			}
			capiEvt.UserData = facebook.HashUserData(userData)

			capiEvents = append(capiEvents, capiEvt)
		}

		_, err := s.capiClient.SendEvents(ctx, targetPixel.FBPixelID, targetPixel.FBAccessToken, capiEvents)
		if err != nil {
			lastErr = err
			// Fail-fast on auth error (any batch) — token is invalid, no point continuing
			if facebook.IsAuthError(err) {
				s.logger.Error("replay auth error, failing fast", "error", err, "session_id", session.ID)
				_ = s.replayRepo.UpdateStatusWithError(ctx, session.ID, "failed", sanitizeReplayError(err))
				return
			}
			failed += int64(len(batch))
			s.logger.Error("replay batch failed", "error", err, "batch_start", i, "batch_end", end)
		} else {
			replayed += int64(len(batch))
		}

		_ = s.replayRepo.UpdateProgress(ctx, session.ID, int(replayed), int(failed))

		// Batch delay between batches (skip after the last batch)
		if session.BatchDelayMs > 0 && end < len(events) {
			time.Sleep(time.Duration(session.BatchDelayMs) * time.Millisecond)
		}
	}

	if failed > 0 && replayed == 0 {
		errMsg := "all batches failed"
		if lastErr != nil {
			errMsg = sanitizeReplayError(lastErr)
		}
		_ = s.replayRepo.UpdateStatusWithError(ctx, session.ID, "failed", errMsg)
	} else {
		_ = s.replayRepo.UpdateStatus(ctx, session.ID, "completed")
	}

	s.logger.Info("replay completed",
		"session_id", session.ID,
		"total", session.TotalEvents,
		"replayed", replayed,
		"failed", failed,
	)
}

// sanitizeReplayError converts raw errors into safe user-facing messages.
// Prevents leaking Facebook API internals (tokens, error bodies) through error_message.
func sanitizeReplayError(err error) string {
	var capiErr *facebook.CAPIError
	if errors.As(err, &capiErr) {
		switch capiErr.StatusCode {
		case 401, 403:
			return "Facebook authentication failed. Check your access token."
		case 400:
			return "Facebook rejected the request. Check your Pixel ID and Access Token."
		case 429:
			return "Facebook rate limit exceeded."
		default:
			return fmt.Sprintf("Facebook returned HTTP %d.", capiErr.StatusCode)
		}
	}
	return "replay failed"
}

func (s *ReplayService) GetByID(ctx context.Context, customerID, sessionID string) (*domain.ReplaySession, error) {
	session, err := s.replayRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get replay session: %w", err)
	}
	if session == nil || session.CustomerID != customerID {
		return nil, ErrReplayNotFound
	}
	return session, nil
}

func (s *ReplayService) List(ctx context.Context, customerID string) ([]*domain.ReplaySession, error) {
	sessions, err := s.replayRepo.ListByCustomerID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("list replay sessions: %w", err)
	}
	if sessions == nil {
		sessions = []*domain.ReplaySession{}
	}
	return sessions, nil
}
