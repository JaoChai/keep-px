package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/facebook"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

var (
	ErrReplayNotFound      = errors.New("replay session not found")
	ErrReplayNotCancellable = errors.New("replay session cannot be cancelled")
	ErrReplayNotRetryable   = errors.New("replay session cannot be retried")
	ErrReplaySamePixel      = errors.New("source and target pixel cannot be the same")
	ErrPixelNoCredentials   = errors.New("target pixel has no Facebook credentials configured")
)

var ErrInvalidDateFormat = errors.New("invalid date format, expected RFC3339")

const timeModeOriginal = "original"

const (
	maxRetries        = 3
	defaultBatchDelay = 200 * time.Millisecond
)

type ReplayService struct {
	replayRepo   repository.ReplaySessionRepository
	eventRepo    repository.EventRepository
	pixelRepo    repository.PixelRepository
	capiClient   *facebook.CAPIClient
	logger       *slog.Logger
	notifService *NotificationService
	quotaService *QuotaService
	shutdownCtx  context.Context
	wg           sync.WaitGroup
	sem          chan struct{} // concurrent replay limiter
}

func NewReplayService(
	shutdownCtx context.Context,
	replayRepo repository.ReplaySessionRepository,
	eventRepo repository.EventRepository,
	pixelRepo repository.PixelRepository,
	capiClient *facebook.CAPIClient,
	logger *slog.Logger,
	maxConcurrentReplays int,
	notifService *NotificationService,
	quotaService *QuotaService,
) *ReplayService {
	if maxConcurrentReplays <= 0 {
		maxConcurrentReplays = 5
	}
	return &ReplayService{
		replayRepo:   replayRepo,
		eventRepo:    eventRepo,
		pixelRepo:    pixelRepo,
		capiClient:   capiClient,
		logger:       logger,
		notifService: notifService,
		quotaService: quotaService,
		shutdownCtx:  shutdownCtx,
		sem:          make(chan struct{}, maxConcurrentReplays),
	}
}

// Shutdown waits for all background replay goroutines to finish.
func (s *ReplayService) Shutdown() {
	s.wg.Wait()
}

// createReplayNotification creates a notification for the replay session owner.
// It is nil-safe (no-op if notifService is nil) and uses a background context
// so that notifications are still created even when the request context is done.
func (s *ReplayService) createReplayNotification(session *domain.ReplaySession, notifType, title, body string) {
	if s.notifService == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		metadata, _ := json.Marshal(map[string]string{"session_id": session.ID})
		n := &domain.Notification{
			CustomerID: session.CustomerID,
			Type:       notifType,
			Title:      title,
			Body:       body,
			Metadata:   metadata,
		}
		if err := s.notifService.CreateNotification(ctx, n); err != nil {
			s.logger.Error("failed to create replay notification", "error", err, "session_id", session.ID, "type", notifType)
		}
	}()
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

type PreviewReplayResult struct {
	TotalEvents  int                  `json:"total_events"`
	SampleEvents []*domain.PixelEvent `json:"sample_events"`
	Warning      string               `json:"warning,omitempty"`
}

// validateReplayTarget checks that source and target are different and the target has credentials.
func validateReplayTarget(sourceID, targetID string, target *domain.Pixel) error {
	if sourceID == targetID {
		return ErrReplaySamePixel
	}
	if !target.HasCredentials() {
		return ErrPixelNoCredentials
	}
	return nil
}

// parseDateFilter parses an optional RFC3339 date string into a *time.Time.
func parseDateFilter(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidDateFormat, s)
	}
	return &t, nil
}

func (s *ReplayService) Create(ctx context.Context, customerID string, input CreateReplayInput) (*CreateReplayResult, error) {
	// Verify source pixel
	sourcePixel, err := s.pixelRepo.GetByID(ctx, input.SourcePixelID)
	if err != nil {
		return nil, fmt.Errorf("get source pixel: %w", err)
	}
	if sourcePixel == nil || sourcePixel.CustomerID != customerID {
		return nil, ErrPixelNotFound
	}

	// Verify target pixel
	targetPixel, err := s.pixelRepo.GetByID(ctx, input.TargetPixelID)
	if err != nil {
		return nil, fmt.Errorf("get target pixel: %w", err)
	}
	if targetPixel == nil || targetPixel.CustomerID != customerID {
		return nil, ErrPixelNotFound
	}

	if err := validateReplayTarget(input.SourcePixelID, input.TargetPixelID, targetPixel); err != nil {
		return nil, err
	}

	dateFrom, err := parseDateFilter(input.DateFrom)
	if err != nil {
		return nil, err
	}
	dateTo, err := parseDateFilter(input.DateTo)
	if err != nil {
		return nil, err
	}

	// Count events to replay
	events, err := s.eventRepo.GetEventsForReplay(ctx, input.SourcePixelID, input.EventTypes, dateFrom, dateTo, nil)
	if err != nil {
		return nil, fmt.Errorf("get events for replay: %w", err)
	}

	// Default TimeMode to "original"
	timeMode := input.TimeMode
	if timeMode == "" {
		timeMode = timeModeOriginal
	}

	// Warn about old events when using original time mode
	var warning string
	if timeMode == timeModeOriginal && len(events) > 0 {
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

	// Create session FIRST (before consuming credit) so we have a record
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

	// Consume a replay credit if quota service is available
	if s.quotaService != nil {
		credit, err := s.quotaService.ConsumeReplayCredit(ctx, customerID, len(events))
		if err != nil {
			// Mark session as failed since credit consumption failed
			if updateErr := s.replayRepo.UpdateStatusWithError(ctx, session.ID, "failed", err.Error()); updateErr != nil {
				s.logger.Error("failed to mark session failed after credit error", "error", updateErr, "session_id", session.ID)
			}
			return nil, err
		}
		// Cap events to the credit's limit if lower than the global max
		if credit.MaxEventsPerReplay > 0 && len(events) > credit.MaxEventsPerReplay {
			events = events[:credit.MaxEventsPerReplay]
			session.TotalEvents = len(events)
			if updateErr := s.replayRepo.UpdateTotalEvents(ctx, session.ID, len(events)); updateErr != nil {
				s.logger.Warn("failed to update total events after cap", "error", updateErr, "session_id", session.ID)
			}
		}
	}

	// Start replay in background (semaphore-limited)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		select {
		case s.sem <- struct{}{}:
			defer func() { <-s.sem }()
			s.executeReplay(s.shutdownCtx, session, targetPixel, events)
		case <-s.shutdownCtx.Done():
			s.logger.Warn("replay skipped due to shutdown", "session_id", session.ID)
			writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := s.replayRepo.UpdateStatusWithError(writeCtx, session.ID, "failed", "server shutdown before replay started"); err != nil {
				s.logger.Error("failed to mark session failed on shutdown", "error", err, "session_id", session.ID)
			}
			s.createReplayNotification(session, domain.NotificationTypeReplayFailed,
				"Replay failed",
				"Server shut down before your replay could start. Please retry.")
		}
	}()

	return &CreateReplayResult{Session: session, Warning: warning}, nil
}

func (s *ReplayService) sendBatchWithRetry(ctx context.Context, pixelID, accessToken string, events []facebook.CAPIEvent) (*facebook.CAPIResponse, error) {
	return facebook.SendEventsWithRetry(ctx, s.capiClient, pixelID, accessToken, "", events, maxRetries, s.logger)
}

func (s *ReplayService) executeReplay(ctx context.Context, session *domain.ReplaySession, targetPixel *domain.Pixel, events []*domain.PixelEvent) {
	if err := s.replayRepo.UpdateStatus(ctx, session.ID, "running"); err != nil {
		s.logger.Error("failed to update replay status to running", "error", err, "session_id", session.ID)
	}

	var replayed, failed int64
	var lastErr error
	var failedBatchRanges []domain.BatchRange
	const batchSize = 1000

	for i := 0; i < len(events); i += batchSize {
		// Check for cancellation before each batch
		status, err := s.replayRepo.GetStatus(ctx, session.ID)
		if err == nil && status == "cancelled" {
			s.logger.Info("replay cancelled by user", "session_id", session.ID, "replayed", replayed, "failed", failed)
			// Save failed batch ranges for potential retry
			if len(failedBatchRanges) > 0 {
				if rangesJSON, err := json.Marshal(failedBatchRanges); err == nil {
					if err := s.replayRepo.UpdateFailedBatches(ctx, session.ID, rangesJSON); err != nil {
						s.logger.Warn("failed to save failed batch ranges on cancel", "error", err, "session_id", session.ID)
					}
				}
			}
			if err := s.replayRepo.UpdateProgress(ctx, session.ID, int(replayed), int(failed)); err != nil {
				s.logger.Warn("failed to update progress on cancel", "error", err, "session_id", session.ID)
			}
			return
		}

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
			if _, exists := userData["country"]; !exists {
				userData["country"] = defaultCountry
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

		// Replay sends to the live endpoint; test_event_code is intentionally
		// omitted so replayed events do not appear in FB Test Events tool.
		_, err = s.sendBatchWithRetry(ctx, targetPixel.FBPixelID, targetPixel.FBAccessToken, capiEvents)
		if err != nil {
			lastErr = err
			// Fail-fast on auth error (any batch) — token is invalid, no point continuing
			if facebook.IsAuthError(err) {
				s.logger.Error("replay auth error, failing fast", "error", err, "session_id", session.ID)
				if err := s.replayRepo.UpdateStatusWithError(ctx, session.ID, "failed", sanitizeReplayError(err)); err != nil {
					s.logger.Error("failed to update replay status to failed", "error", err, "session_id", session.ID)
				}
				s.createReplayNotification(session, domain.NotificationTypeCAPIAuthError,
					"Replay failed — authentication error",
					fmt.Sprintf("Facebook rejected the access token for your replay session. %d events were not replayed.", session.TotalEvents))
				return
			}
			failed += int64(len(batch))
			failedBatchRanges = append(failedBatchRanges, domain.BatchRange{Start: i, End: end})
			s.logger.Error("replay batch failed", "error", err, "batch_start", i, "batch_end", end)
		} else {
			replayed += int64(len(batch))
		}

		if err := s.replayRepo.UpdateProgress(ctx, session.ID, int(replayed), int(failed)); err != nil {
			s.logger.Warn("failed to update replay progress", "error", err, "session_id", session.ID)
		}

		// Batch delay between batches (skip after the last batch)
		delay := time.Duration(session.BatchDelayMs) * time.Millisecond
		if delay == 0 {
			delay = defaultBatchDelay
		}
		if end < len(events) {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return
			}
		}
	}

	// Save failed batch ranges if any
	if len(failedBatchRanges) > 0 {
		if rangesJSON, err := json.Marshal(failedBatchRanges); err == nil {
			if err := s.replayRepo.UpdateFailedBatches(ctx, session.ID, rangesJSON); err != nil {
				s.logger.Warn("failed to save failed batch ranges", "error", err, "session_id", session.ID)
			}
		}
	}

	if failed > 0 && replayed == 0 {
		errMsg := "all batches failed"
		if lastErr != nil {
			errMsg = sanitizeReplayError(lastErr)
		}
		if err := s.replayRepo.UpdateStatusWithError(ctx, session.ID, "failed", errMsg); err != nil {
			s.logger.Error("failed to update replay status to failed", "error", err, "session_id", session.ID)
		}
		s.createReplayNotification(session, domain.NotificationTypeReplayFailed,
			"Replay failed",
			fmt.Sprintf("All %d events failed to replay. You can retry from the replay history.", session.TotalEvents))
	} else {
		if err := s.replayRepo.UpdateStatus(ctx, session.ID, "completed"); err != nil {
			s.logger.Error("failed to update replay status to completed", "error", err, "session_id", session.ID)
		}
		s.createReplayNotification(session, domain.NotificationTypeReplayCompleted,
			"Replay completed",
			fmt.Sprintf("Successfully replayed %d events (%d failed).", replayed, failed))
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

func (s *ReplayService) Cancel(ctx context.Context, customerID, sessionID string) (*domain.ReplaySession, error) {
	// Verify ownership first
	session, err := s.replayRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get replay session: %w", err)
	}
	if session == nil || session.CustomerID != customerID {
		return nil, ErrReplayNotFound
	}

	// Atomic cancel: only transitions pending/running -> cancelled
	cancelled, err := s.replayRepo.CancelSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("cancel replay session: %w", err)
	}
	if cancelled == nil {
		return nil, ErrReplayNotCancellable
	}
	return cancelled, nil
}

// MaxReplayEvents is the maximum number of events a single replay can process.
// Also used by event_repo to cap SQL queries.
const MaxReplayEvents = 100000

func (s *ReplayService) Preview(ctx context.Context, customerID string, input CreateReplayInput) (*PreviewReplayResult, error) {
	// Verify source pixel
	sourcePixel, err := s.pixelRepo.GetByID(ctx, input.SourcePixelID)
	if err != nil {
		return nil, fmt.Errorf("get source pixel: %w", err)
	}
	if sourcePixel == nil || sourcePixel.CustomerID != customerID {
		return nil, ErrPixelNotFound
	}

	// Verify target pixel
	targetPixel, err := s.pixelRepo.GetByID(ctx, input.TargetPixelID)
	if err != nil {
		return nil, fmt.Errorf("get target pixel: %w", err)
	}
	if targetPixel == nil || targetPixel.CustomerID != customerID {
		return nil, ErrPixelNotFound
	}

	if err := validateReplayTarget(input.SourcePixelID, input.TargetPixelID, targetPixel); err != nil {
		return nil, err
	}

	dateFrom, err := parseDateFilter(input.DateFrom)
	if err != nil {
		return nil, err
	}
	dateTo, err := parseDateFilter(input.DateTo)
	if err != nil {
		return nil, err
	}

	totalEvents, err := s.eventRepo.CountEventsForReplay(ctx, input.SourcePixelID, input.EventTypes, dateFrom, dateTo)
	if err != nil {
		return nil, fmt.Errorf("count events for replay preview: %w", err)
	}

	sampleEvents, err := s.eventRepo.GetEventsForReplayPreview(ctx, input.SourcePixelID, input.EventTypes, dateFrom, dateTo, 10)
	if err != nil {
		return nil, fmt.Errorf("get events for replay preview: %w", err)
	}
	if sampleEvents == nil {
		sampleEvents = []*domain.PixelEvent{}
	}

	// Default TimeMode
	timeMode := input.TimeMode
	if timeMode == "" {
		timeMode = timeModeOriginal
	}

	var warnings []string
	if totalEvents > MaxReplayEvents {
		warnings = append(warnings, fmt.Sprintf("%d events found, only first %d will be replayed", totalEvents, MaxReplayEvents))
		totalEvents = MaxReplayEvents
	}

	if timeMode == timeModeOriginal && len(sampleEvents) > 0 {
		var oldest time.Time
		for _, evt := range sampleEvents {
			if oldest.IsZero() || evt.EventTime.Before(oldest) {
				oldest = evt.EventTime
			}
		}
		age := time.Since(oldest)
		if age > 7*24*time.Hour {
			days := int(age.Hours() / 24)
			warnings = append(warnings, fmt.Sprintf("oldest event is %d days old. Facebook may reject events older than 7 days. Consider using time_mode: current.", days))
		}
	}

	var warning string
	if len(warnings) > 0 {
		warning = "Warning: " + strings.Join(warnings, ". ")
	}

	return &PreviewReplayResult{
		TotalEvents:  totalEvents,
		SampleEvents: sampleEvents,
		Warning:      warning,
	}, nil
}

func (s *ReplayService) Retry(ctx context.Context, customerID, sessionID string) (*domain.ReplaySession, error) {
	session, err := s.replayRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get replay session: %w", err)
	}
	if session == nil || session.CustomerID != customerID {
		return nil, ErrReplayNotFound
	}

	// Only failed or cancelled sessions can be retried
	if session.Status != "failed" && session.Status != "cancelled" {
		return nil, ErrReplayNotRetryable
	}

	// Verify target pixel still exists and is owned
	targetPixel, err := s.pixelRepo.GetByID(ctx, session.TargetPixelID)
	if err != nil {
		return nil, fmt.Errorf("get target pixel: %w", err)
	}
	if targetPixel == nil || targetPixel.CustomerID != customerID {
		return nil, ErrPixelNotFound
	}

	if !targetPixel.HasCredentials() {
		return nil, ErrPixelNoCredentials
	}

	// Get all events for replay (we'll use failed batch ranges if available)
	// Use createdBefore to avoid replaying events that arrived after the original session
	allEvents, err := s.eventRepo.GetEventsForReplay(ctx, session.SourcePixelID, session.EventTypes, session.DateFrom, session.DateTo, &session.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get events for retry: %w", err)
	}

	// Determine which events to retry
	var retryEvents []*domain.PixelEvent
	if session.FailedBatchRanges != nil {
		var batchRanges []domain.BatchRange
		if err := json.Unmarshal(session.FailedBatchRanges, &batchRanges); err == nil && len(batchRanges) > 0 {
			for _, br := range batchRanges {
				start := br.Start
				end := br.End
				if start < len(allEvents) {
					if end > len(allEvents) {
						end = len(allEvents)
					}
					retryEvents = append(retryEvents, allEvents[start:end]...)
				}
			}
		}
	}

	// Fall back to all events if no batch range info
	if len(retryEvents) == 0 {
		retryEvents = allEvents
	}

	// Create a NEW session FIRST (preserve history, before consuming credit)
	newSession := &domain.ReplaySession{
		CustomerID:    customerID,
		SourcePixelID: session.SourcePixelID,
		TargetPixelID: session.TargetPixelID,
		TotalEvents:   len(retryEvents),
		EventTypes:    session.EventTypes,
		DateFrom:      session.DateFrom,
		DateTo:        session.DateTo,
		TimeMode:      session.TimeMode,
		BatchDelayMs:  session.BatchDelayMs,
	}

	if err := s.replayRepo.Create(ctx, newSession); err != nil {
		return nil, fmt.Errorf("create retry replay session: %w", err)
	}

	// Consume a replay credit if quota service is available
	if s.quotaService != nil {
		credit, err := s.quotaService.ConsumeReplayCredit(ctx, customerID, len(retryEvents))
		if err != nil {
			// Mark session as failed since credit consumption failed
			if updateErr := s.replayRepo.UpdateStatusWithError(ctx, newSession.ID, "failed", err.Error()); updateErr != nil {
				s.logger.Error("failed to mark retry session failed after credit error", "error", updateErr, "session_id", newSession.ID)
			}
			return nil, err
		}
		// Cap events to the credit's limit if lower
		if credit.MaxEventsPerReplay > 0 && len(retryEvents) > credit.MaxEventsPerReplay {
			retryEvents = retryEvents[:credit.MaxEventsPerReplay]
			newSession.TotalEvents = len(retryEvents)
			if updateErr := s.replayRepo.UpdateTotalEvents(ctx, newSession.ID, len(retryEvents)); updateErr != nil {
				s.logger.Warn("failed to update total events after cap", "error", updateErr, "session_id", newSession.ID)
			}
		}
	}

	// Start replay in background (semaphore-limited)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		select {
		case s.sem <- struct{}{}:
			defer func() { <-s.sem }()
			s.executeReplay(s.shutdownCtx, newSession, targetPixel, retryEvents)
		case <-s.shutdownCtx.Done():
			s.logger.Warn("replay skipped due to shutdown", "session_id", newSession.ID)
			writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := s.replayRepo.UpdateStatusWithError(writeCtx, newSession.ID, "failed", "server shutdown before replay started"); err != nil {
				s.logger.Error("failed to mark session failed on shutdown", "error", err, "session_id", newSession.ID)
			}
			s.createReplayNotification(newSession, domain.NotificationTypeReplayFailed,
				"Replay failed",
				"Server shut down before your replay could start. Please retry.")
		}
	}()

	return newSession, nil
}

func (s *ReplayService) GetEventTypes(ctx context.Context, customerID, pixelID string) ([]string, error) {
	pixel, err := s.pixelRepo.GetByID(ctx, pixelID)
	if err != nil {
		return nil, fmt.Errorf("get pixel: %w", err)
	}
	if pixel == nil || pixel.CustomerID != customerID {
		return nil, ErrPixelNotFound
	}
	types, err := s.eventRepo.GetDistinctEventTypes(ctx, pixelID)
	if err != nil {
		return nil, fmt.Errorf("get event types: %w", err)
	}
	if types == nil {
		types = []string{}
	}
	return types, nil
}
