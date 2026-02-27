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
}

func (s *ReplayService) Create(ctx context.Context, customerID string, input CreateReplayInput) (*domain.ReplaySession, error) {
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

	session := &domain.ReplaySession{
		CustomerID:    customerID,
		SourcePixelID: input.SourcePixelID,
		TargetPixelID: input.TargetPixelID,
		TotalEvents:   len(events),
		EventTypes:    input.EventTypes,
		DateFrom:      dateFrom,
		DateTo:        dateTo,
	}

	if err := s.replayRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create replay session: %w", err)
	}

	// Start replay in background
	go s.executeReplay(context.Background(), session, targetPixel, events)

	return session, nil
}

func (s *ReplayService) executeReplay(ctx context.Context, session *domain.ReplaySession, targetPixel *domain.Pixel, events []*domain.PixelEvent) {
	_ = s.replayRepo.UpdateStatus(ctx, session.ID, "running")

	var replayed, failed int64
	const batchSize = 1000

	for i := 0; i < len(events); i += batchSize {
		end := i + batchSize
		if end > len(events) {
			end = len(events)
		}
		batch := events[i:end]

		capiEvents := make([]facebook.CAPIEvent, 0, len(batch))
		for _, evt := range batch {
			capiEvt := facebook.CAPIEvent{
				EventName:             evt.EventName,
				EventTime:             evt.EventTime.Unix(),
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
			failed += int64(len(batch))
			s.logger.Error("replay batch failed", "error", err, "batch_start", i, "batch_end", end)
		} else {
			replayed += int64(len(batch))
		}

		_ = s.replayRepo.UpdateProgress(ctx, session.ID, int(replayed), int(failed))
	}

	if failed > 0 && replayed == 0 {
		_ = s.replayRepo.UpdateStatus(ctx, session.ID, "failed")
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
