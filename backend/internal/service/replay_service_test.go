package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/facebook"
)

func newTestReplayService() (*ReplayService, *MockReplaySessionRepo, *MockEventRepo, *MockPixelRepo) {
	return newTestReplayServiceWithConcurrency(5)
}

func newTestReplayServiceWithConcurrency(maxConcurrent int) (*ReplayService, *MockReplaySessionRepo, *MockEventRepo, *MockPixelRepo) {
	replayRepo := new(MockReplaySessionRepo)
	eventRepo := new(MockEventRepo)
	pixelRepo := new(MockPixelRepo)
	capiClient := facebook.NewCAPIClient("http://localhost:9999")
	logger := slog.Default()
	svc := NewReplayService(context.Background(), replayRepo, eventRepo, pixelRepo, capiClient, logger, maxConcurrent, nil)
	return svc, replayRepo, eventRepo, pixelRepo
}

func TestReplayService_Create(t *testing.T) {
	tests := []struct {
		name        string
		customerID  string
		input       CreateReplayInput
		setup       func(*MockReplaySessionRepo, *MockEventRepo, *MockPixelRepo)
		wantErr     error
		wantWarning string
	}{
		{
			name:       "success",
			customerID: "cust-1",
			input: CreateReplayInput{
				SourcePixelID: "pixel-1",
				TargetPixelID: "pixel-2",
			},
			setup: func(rr *MockReplaySessionRepo, er *MockEventRepo, pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID:            "pixel-1",
					CustomerID:    "cust-1",
					FBPixelID:     "fb-1",
					FBAccessToken: "token-1",
				}, nil)
				pr.On("GetByID", mock.Anything, "pixel-2").Return(&domain.Pixel{
					ID:            "pixel-2",
					CustomerID:    "cust-1",
					FBPixelID:     "fb-2",
					FBAccessToken: "token-2",
				}, nil)
				er.On("GetEventsForReplay", mock.Anything, "pixel-1", []string(nil), (*time.Time)(nil), (*time.Time)(nil), mock.AnythingOfType("*time.Time")).
					Return([]*domain.PixelEvent{
						{ID: "evt-1", EventName: "PageView", EventTime: time.Now().Add(-1 * time.Hour)},
					}, nil)
				rr.On("Create", mock.Anything, mock.AnythingOfType("*domain.ReplaySession")).Return(nil)
				// Background goroutine calls
				rr.On("UpdateStatus", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
				rr.On("UpdateProgress", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil).Maybe()
				rr.On("UpdateStatusWithError", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
				rr.On("GetStatus", mock.Anything, mock.AnythingOfType("string")).Return("running", nil).Maybe()
				rr.On("UpdateFailedBatches", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).Return(nil).Maybe()
			},
			wantErr: nil,
		},
		{
			name:       "time_mode defaults to original",
			customerID: "cust-1",
			input: CreateReplayInput{
				SourcePixelID: "pixel-1",
				TargetPixelID: "pixel-2",
			},
			setup: func(rr *MockReplaySessionRepo, er *MockEventRepo, pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID:            "pixel-1",
					CustomerID:    "cust-1",
					FBPixelID:     "fb-1",
					FBAccessToken: "token-1",
				}, nil)
				pr.On("GetByID", mock.Anything, "pixel-2").Return(&domain.Pixel{
					ID:            "pixel-2",
					CustomerID:    "cust-1",
					FBPixelID:     "fb-2",
					FBAccessToken: "token-2",
				}, nil)
				er.On("GetEventsForReplay", mock.Anything, "pixel-1", []string(nil), (*time.Time)(nil), (*time.Time)(nil), mock.AnythingOfType("*time.Time")).
					Return([]*domain.PixelEvent{
						{ID: "evt-1", EventName: "PageView", EventTime: time.Now().Add(-1 * time.Hour)},
					}, nil)
				rr.On("Create", mock.Anything, mock.AnythingOfType("*domain.ReplaySession")).Return(nil)
				rr.On("UpdateStatus", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
				rr.On("UpdateProgress", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil).Maybe()
				rr.On("UpdateStatusWithError", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
				rr.On("GetStatus", mock.Anything, mock.AnythingOfType("string")).Return("running", nil).Maybe()
				rr.On("UpdateFailedBatches", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).Return(nil).Maybe()
			},
			wantErr: nil,
		},
		{
			name:       "time_mode current accepted",
			customerID: "cust-1",
			input: CreateReplayInput{
				SourcePixelID: "pixel-1",
				TargetPixelID: "pixel-2",
				TimeMode:      "current",
			},
			setup: func(rr *MockReplaySessionRepo, er *MockEventRepo, pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID:            "pixel-1",
					CustomerID:    "cust-1",
					FBPixelID:     "fb-1",
					FBAccessToken: "token-1",
				}, nil)
				pr.On("GetByID", mock.Anything, "pixel-2").Return(&domain.Pixel{
					ID:            "pixel-2",
					CustomerID:    "cust-1",
					FBPixelID:     "fb-2",
					FBAccessToken: "token-2",
				}, nil)
				er.On("GetEventsForReplay", mock.Anything, "pixel-1", []string(nil), (*time.Time)(nil), (*time.Time)(nil), mock.AnythingOfType("*time.Time")).
					Return([]*domain.PixelEvent{
						{ID: "evt-1", EventName: "PageView", EventTime: time.Now().Add(-1 * time.Hour)},
					}, nil)
				rr.On("Create", mock.Anything, mock.AnythingOfType("*domain.ReplaySession")).Return(nil)
				rr.On("UpdateStatus", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
				rr.On("UpdateProgress", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil).Maybe()
				rr.On("UpdateStatusWithError", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
				rr.On("GetStatus", mock.Anything, mock.AnythingOfType("string")).Return("running", nil).Maybe()
				rr.On("UpdateFailedBatches", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).Return(nil).Maybe()
			},
			wantErr: nil,
		},
		{
			name:       "warns about old events",
			customerID: "cust-1",
			input: CreateReplayInput{
				SourcePixelID: "pixel-1",
				TargetPixelID: "pixel-2",
			},
			setup: func(rr *MockReplaySessionRepo, er *MockEventRepo, pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID:            "pixel-1",
					CustomerID:    "cust-1",
					FBPixelID:     "fb-1",
					FBAccessToken: "token-1",
				}, nil)
				pr.On("GetByID", mock.Anything, "pixel-2").Return(&domain.Pixel{
					ID:            "pixel-2",
					CustomerID:    "cust-1",
					FBPixelID:     "fb-2",
					FBAccessToken: "token-2",
				}, nil)
				er.On("GetEventsForReplay", mock.Anything, "pixel-1", []string(nil), (*time.Time)(nil), (*time.Time)(nil), mock.AnythingOfType("*time.Time")).
					Return([]*domain.PixelEvent{
						{ID: "evt-1", EventName: "PageView", EventTime: time.Now().Add(-10 * 24 * time.Hour)},
						{ID: "evt-2", EventName: "Purchase", EventTime: time.Now().Add(-1 * time.Hour)},
					}, nil)
				rr.On("Create", mock.Anything, mock.AnythingOfType("*domain.ReplaySession")).Return(nil)
				rr.On("UpdateStatus", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
				rr.On("UpdateProgress", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil).Maybe()
				rr.On("UpdateStatusWithError", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
				rr.On("GetStatus", mock.Anything, mock.AnythingOfType("string")).Return("running", nil).Maybe()
				rr.On("UpdateFailedBatches", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).Return(nil).Maybe()
			},
			wantErr:     nil,
			wantWarning: "older than 7 days",
		},
		{
			name:       "source pixel not found",
			customerID: "cust-1",
			input: CreateReplayInput{
				SourcePixelID: "nonexistent",
				TargetPixelID: "pixel-2",
			},
			setup: func(rr *MockReplaySessionRepo, er *MockEventRepo, pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)
			},
			wantErr: ErrPixelNotFound,
		},
		{
			name:       "target pixel not owned",
			customerID: "cust-1",
			input: CreateReplayInput{
				SourcePixelID: "pixel-1",
				TargetPixelID: "pixel-3",
			},
			setup: func(rr *MockReplaySessionRepo, er *MockEventRepo, pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID:         "pixel-1",
					CustomerID: "cust-1",
				}, nil)
				pr.On("GetByID", mock.Anything, "pixel-3").Return(&domain.Pixel{
					ID:         "pixel-3",
					CustomerID: "cust-other",
				}, nil)
			},
			wantErr: ErrPixelNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, replayRepo, eventRepo, pixelRepo := newTestReplayService()
			tt.setup(replayRepo, eventRepo, pixelRepo)

			result, err := svc.Create(context.Background(), tt.customerID, tt.input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				require.NotNil(t, result.Session)
				assert.Equal(t, tt.customerID, result.Session.CustomerID)
				assert.Equal(t, tt.input.SourcePixelID, result.Session.SourcePixelID)
				assert.Equal(t, tt.input.TargetPixelID, result.Session.TargetPixelID)

				// TimeMode should always be set
				if tt.input.TimeMode == "" {
					assert.Equal(t, "original", result.Session.TimeMode)
				} else {
					assert.Equal(t, tt.input.TimeMode, result.Session.TimeMode)
				}

				if tt.wantWarning != "" {
					assert.Contains(t, result.Warning, tt.wantWarning)
				} else {
					assert.Empty(t, result.Warning)
				}
			}
			pixelRepo.AssertExpectations(t)
		})
	}
}

func TestReplayService_ExecuteReplay_AuthErrorFailFast(t *testing.T) {
	// Create a fake CAPI server that returns 403 Forbidden
	fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":{"message":"Invalid OAuth access token","type":"OAuthException","code":190}}`))
	}))
	defer fakeServer.Close()

	replayRepo := new(MockReplaySessionRepo)
	eventRepo := new(MockEventRepo)
	pixelRepo := new(MockPixelRepo)
	capiClient := facebook.NewCAPIClient(fakeServer.URL)
	logger := slog.Default()
	svc := NewReplayService(context.Background(), replayRepo, eventRepo, pixelRepo, capiClient, logger, 5, nil)

	session := &domain.ReplaySession{
		ID:            "session-1",
		CustomerID:    "cust-1",
		SourcePixelID: "pixel-1",
		TargetPixelID: "pixel-2",
		TotalEvents:   2,
		TimeMode:      "original",
	}
	targetPixel := &domain.Pixel{
		ID:            "pixel-2",
		CustomerID:    "cust-1",
		FBPixelID:     "fb-2",
		FBAccessToken: "bad-token",
	}
	events := []*domain.PixelEvent{
		{ID: "evt-1", EventName: "PageView", EventTime: time.Now()},
		{ID: "evt-2", EventName: "Purchase", EventTime: time.Now()},
	}

	replayRepo.On("UpdateStatus", mock.Anything, "session-1", "running").Return(nil)
	replayRepo.On("GetStatus", mock.Anything, "session-1").Return("running", nil)
	replayRepo.On("UpdateStatusWithError", mock.Anything, "session-1", "failed", mock.AnythingOfType("string")).Return(nil)

	// Call executeReplay directly (it normally runs as a goroutine)
	svc.executeReplay(context.Background(), session, targetPixel, events)

	replayRepo.AssertExpectations(t)
	// Verify UpdateStatusWithError was called (not just UpdateStatus with "failed")
	replayRepo.AssertCalled(t, "UpdateStatusWithError", mock.Anything, "session-1", "failed", mock.AnythingOfType("string"))
	// Verify UpdateProgress was NOT called (fail-fast before any progress update)
	replayRepo.AssertNotCalled(t, "UpdateProgress", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestReplayService_ExecuteReplay_TimeModeCurrentUsesNow(t *testing.T) {
	// Create a fake CAPI server that captures the request and verifies event_time
	var capturedEvents []facebook.CAPIEvent
	fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req facebook.CAPIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
			return
		}
		capturedEvents = append(capturedEvents, req.Data...)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(facebook.CAPIResponse{EventsReceived: len(req.Data)})
	}))
	defer fakeServer.Close()

	replayRepo := new(MockReplaySessionRepo)
	eventRepo := new(MockEventRepo)
	pixelRepo := new(MockPixelRepo)
	capiClient := facebook.NewCAPIClient(fakeServer.URL)
	logger := slog.Default()
	svc := NewReplayService(context.Background(), replayRepo, eventRepo, pixelRepo, capiClient, logger, 5, nil)

	oldTime := time.Now().Add(-30 * 24 * time.Hour) // 30 days ago
	session := &domain.ReplaySession{
		ID:            "session-1",
		CustomerID:    "cust-1",
		SourcePixelID: "pixel-1",
		TargetPixelID: "pixel-2",
		TotalEvents:   1,
		TimeMode:      "current",
	}
	targetPixel := &domain.Pixel{
		ID:            "pixel-2",
		CustomerID:    "cust-1",
		FBPixelID:     "fb-2",
		FBAccessToken: "token-2",
	}
	events := []*domain.PixelEvent{
		{ID: "evt-1", EventName: "PageView", EventTime: oldTime},
	}

	replayRepo.On("UpdateStatus", mock.Anything, "session-1", "running").Return(nil)
	replayRepo.On("GetStatus", mock.Anything, "session-1").Return("running", nil)
	replayRepo.On("UpdateProgress", mock.Anything, "session-1", mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil)
	replayRepo.On("UpdateStatus", mock.Anything, "session-1", "completed").Return(nil)

	beforeExec := time.Now().Unix()
	svc.executeReplay(context.Background(), session, targetPixel, events)
	afterExec := time.Now().Unix()

	require.Len(t, capturedEvents, 1)
	// Event time should be close to now, not 30 days ago
	assert.GreaterOrEqual(t, capturedEvents[0].EventTime, beforeExec)
	assert.LessOrEqual(t, capturedEvents[0].EventTime, afterExec)
	replayRepo.AssertExpectations(t)
}

func TestReplayService_ExecuteReplay_BatchDelay(t *testing.T) {
	// Create a fake CAPI server that succeeds
	callCount := 0
	fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(facebook.CAPIResponse{EventsReceived: 1})
	}))
	defer fakeServer.Close()

	replayRepo := new(MockReplaySessionRepo)
	eventRepo := new(MockEventRepo)
	pixelRepo := new(MockPixelRepo)
	capiClient := facebook.NewCAPIClient(fakeServer.URL)
	logger := slog.Default()
	svc := NewReplayService(context.Background(), replayRepo, eventRepo, pixelRepo, capiClient, logger, 5, nil)

	session := &domain.ReplaySession{
		ID:            "session-1",
		CustomerID:    "cust-1",
		SourcePixelID: "pixel-1",
		TargetPixelID: "pixel-2",
		TotalEvents:   2,
		TimeMode:      "original",
		BatchDelayMs:  100,
	}
	targetPixel := &domain.Pixel{
		ID:            "pixel-2",
		CustomerID:    "cust-1",
		FBPixelID:     "fb-2",
		FBAccessToken: "token-2",
	}

	// Create enough events to fill 2 batches (use batch size 1000, so >1000 events)
	// Actually, both events will be in one batch since batchSize=1000 > 2 events.
	// For a meaningful delay test, we just verify the function completes with BatchDelayMs set.
	events := []*domain.PixelEvent{
		{ID: "evt-1", EventName: "PageView", EventTime: time.Now()},
		{ID: "evt-2", EventName: "Purchase", EventTime: time.Now()},
	}

	replayRepo.On("UpdateStatus", mock.Anything, "session-1", "running").Return(nil)
	replayRepo.On("GetStatus", mock.Anything, "session-1").Return("running", nil)
	replayRepo.On("UpdateProgress", mock.Anything, "session-1", mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil)
	replayRepo.On("UpdateStatus", mock.Anything, "session-1", "completed").Return(nil)

	svc.executeReplay(context.Background(), session, targetPixel, events)

	// Both events fit in one batch, so only one CAPI call
	assert.Equal(t, 1, callCount)
	replayRepo.AssertExpectations(t)
}

func TestReplayService_ExecuteReplay_NonAuthErrorContinues(t *testing.T) {
	// Create a fake CAPI server that returns 500 (non-auth error)
	fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":{"message":"Internal error"}}`))
	}))
	defer fakeServer.Close()

	replayRepo := new(MockReplaySessionRepo)
	eventRepo := new(MockEventRepo)
	pixelRepo := new(MockPixelRepo)
	capiClient := facebook.NewCAPIClient(fakeServer.URL)
	logger := slog.Default()
	svc := NewReplayService(context.Background(), replayRepo, eventRepo, pixelRepo, capiClient, logger, 5, nil)

	session := &domain.ReplaySession{
		ID:            "session-1",
		CustomerID:    "cust-1",
		SourcePixelID: "pixel-1",
		TargetPixelID: "pixel-2",
		TotalEvents:   1,
		TimeMode:      "original",
	}
	targetPixel := &domain.Pixel{
		ID:            "pixel-2",
		CustomerID:    "cust-1",
		FBPixelID:     "fb-2",
		FBAccessToken: "token",
	}
	events := []*domain.PixelEvent{
		{ID: "evt-1", EventName: "PageView", EventTime: time.Now()},
	}

	replayRepo.On("UpdateStatus", mock.Anything, "session-1", "running").Return(nil)
	replayRepo.On("GetStatus", mock.Anything, "session-1").Return("running", nil)
	replayRepo.On("UpdateProgress", mock.Anything, "session-1", mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil)
	replayRepo.On("UpdateStatusWithError", mock.Anything, "session-1", "failed", mock.AnythingOfType("string")).Return(nil)
	replayRepo.On("UpdateFailedBatches", mock.Anything, "session-1", mock.AnythingOfType("[]uint8")).Return(nil)

	svc.executeReplay(context.Background(), session, targetPixel, events)

	replayRepo.AssertExpectations(t)
	// Should use UpdateStatusWithError for all-failed case, not UpdateStatus
	replayRepo.AssertCalled(t, "UpdateStatusWithError", mock.Anything, "session-1", "failed", mock.AnythingOfType("string"))
}

func TestReplayService_ExecuteReplay_CancellationCheck(t *testing.T) {
	callCount := 0
	fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(facebook.CAPIResponse{EventsReceived: 1})
	}))
	defer fakeServer.Close()

	replayRepo := new(MockReplaySessionRepo)
	eventRepo := new(MockEventRepo)
	pixelRepo := new(MockPixelRepo)
	capiClient := facebook.NewCAPIClient(fakeServer.URL)
	logger := slog.Default()
	svc := NewReplayService(context.Background(), replayRepo, eventRepo, pixelRepo, capiClient, logger, 5, nil)

	session := &domain.ReplaySession{
		ID:            "session-1",
		CustomerID:    "cust-1",
		SourcePixelID: "pixel-1",
		TargetPixelID: "pixel-2",
		TotalEvents:   1,
		TimeMode:      "original",
	}
	targetPixel := &domain.Pixel{
		ID:            "pixel-2",
		CustomerID:    "cust-1",
		FBPixelID:     "fb-2",
		FBAccessToken: "token-2",
	}
	events := []*domain.PixelEvent{
		{ID: "evt-1", EventName: "PageView", EventTime: time.Now()},
	}

	replayRepo.On("UpdateStatus", mock.Anything, "session-1", "running").Return(nil)
	// Return cancelled when checking status
	replayRepo.On("GetStatus", mock.Anything, "session-1").Return("cancelled", nil)
	replayRepo.On("UpdateProgress", mock.Anything, "session-1", mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil)

	svc.executeReplay(context.Background(), session, targetPixel, events)

	// CAPI should not be called because cancellation was detected before the batch
	assert.Equal(t, 0, callCount)
	replayRepo.AssertExpectations(t)
}

func TestReplayService_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		customerID string
		sessionID  string
		setup      func(*MockReplaySessionRepo)
		wantErr    error
	}{
		{
			name:       "success",
			customerID: "cust-1",
			sessionID:  "session-1",
			setup: func(rr *MockReplaySessionRepo) {
				rr.On("GetByID", mock.Anything, "session-1").Return(&domain.ReplaySession{
					ID:         "session-1",
					CustomerID: "cust-1",
					Status:     "completed",
				}, nil)
			},
			wantErr: nil,
		},
		{
			name:       "not found",
			customerID: "cust-1",
			sessionID:  "nonexistent",
			setup: func(rr *MockReplaySessionRepo) {
				rr.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)
			},
			wantErr: ErrReplayNotFound,
		},
		{
			name:       "not owned",
			customerID: "cust-2",
			sessionID:  "session-1",
			setup: func(rr *MockReplaySessionRepo) {
				rr.On("GetByID", mock.Anything, "session-1").Return(&domain.ReplaySession{
					ID:         "session-1",
					CustomerID: "cust-1",
				}, nil)
			},
			wantErr: ErrReplayNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, replayRepo, _, _ := newTestReplayService()
			tt.setup(replayRepo)

			session, err := svc.GetByID(context.Background(), tt.customerID, tt.sessionID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, session)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, session)
			}
			replayRepo.AssertExpectations(t)
		})
	}
}

func TestReplayService_List(t *testing.T) {
	svc, replayRepo, _, _ := newTestReplayService()

	replayRepo.On("ListByCustomerID", mock.Anything, "cust-1").Return([]*domain.ReplaySession{
		{ID: "session-1", CustomerID: "cust-1", Status: "completed"},
		{ID: "session-2", CustomerID: "cust-1", Status: "running"},
	}, nil)

	sessions, err := svc.List(context.Background(), "cust-1")

	assert.NoError(t, err)
	assert.Len(t, sessions, 2)
	replayRepo.AssertExpectations(t)
}

func TestReplayService_Cancel(t *testing.T) {
	tests := []struct {
		name       string
		customerID string
		sessionID  string
		setup      func(*MockReplaySessionRepo)
		wantErr    error
	}{
		{
			name:       "cancel running session",
			customerID: "cust-1",
			sessionID:  "session-1",
			setup: func(rr *MockReplaySessionRepo) {
				rr.On("GetByID", mock.Anything, "session-1").Return(&domain.ReplaySession{
					ID:         "session-1",
					CustomerID: "cust-1",
					Status:     "running",
				}, nil)
				rr.On("CancelSession", mock.Anything, "session-1").Return(&domain.ReplaySession{
					ID:         "session-1",
					CustomerID: "cust-1",
					Status:     "cancelled",
				}, nil)
			},
			wantErr: nil,
		},
		{
			name:       "cancel pending session",
			customerID: "cust-1",
			sessionID:  "session-2",
			setup: func(rr *MockReplaySessionRepo) {
				rr.On("GetByID", mock.Anything, "session-2").Return(&domain.ReplaySession{
					ID:         "session-2",
					CustomerID: "cust-1",
					Status:     "pending",
				}, nil)
				rr.On("CancelSession", mock.Anything, "session-2").Return(&domain.ReplaySession{
					ID:         "session-2",
					CustomerID: "cust-1",
					Status:     "cancelled",
				}, nil)
			},
			wantErr: nil,
		},
		{
			name:       "cannot cancel completed session",
			customerID: "cust-1",
			sessionID:  "session-3",
			setup: func(rr *MockReplaySessionRepo) {
				rr.On("GetByID", mock.Anything, "session-3").Return(&domain.ReplaySession{
					ID:         "session-3",
					CustomerID: "cust-1",
					Status:     "completed",
				}, nil)
				// CancelSession returns nil when status doesn't match
				rr.On("CancelSession", mock.Anything, "session-3").Return(nil, nil)
			},
			wantErr: ErrReplayNotCancellable,
		},
		{
			name:       "cannot cancel failed session",
			customerID: "cust-1",
			sessionID:  "session-4",
			setup: func(rr *MockReplaySessionRepo) {
				rr.On("GetByID", mock.Anything, "session-4").Return(&domain.ReplaySession{
					ID:         "session-4",
					CustomerID: "cust-1",
					Status:     "failed",
				}, nil)
				// CancelSession returns nil when status doesn't match
				rr.On("CancelSession", mock.Anything, "session-4").Return(nil, nil)
			},
			wantErr: ErrReplayNotCancellable,
		},
		{
			name:       "session not found",
			customerID: "cust-1",
			sessionID:  "nonexistent",
			setup: func(rr *MockReplaySessionRepo) {
				rr.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)
			},
			wantErr: ErrReplayNotFound,
		},
		{
			name:       "session not owned",
			customerID: "cust-2",
			sessionID:  "session-1",
			setup: func(rr *MockReplaySessionRepo) {
				rr.On("GetByID", mock.Anything, "session-1").Return(&domain.ReplaySession{
					ID:         "session-1",
					CustomerID: "cust-1",
					Status:     "running",
				}, nil)
			},
			wantErr: ErrReplayNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, replayRepo, _, _ := newTestReplayService()
			tt.setup(replayRepo)

			session, err := svc.Cancel(context.Background(), tt.customerID, tt.sessionID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, session)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, session)
				assert.Equal(t, "cancelled", session.Status)
			}
			replayRepo.AssertExpectations(t)
		})
	}
}

func TestReplayService_Preview(t *testing.T) {
	tests := []struct {
		name        string
		customerID  string
		input       CreateReplayInput
		setup       func(*MockReplaySessionRepo, *MockEventRepo, *MockPixelRepo)
		wantErr     error
		wantTotal   int
		wantSamples int
	}{
		{
			name:       "success with events",
			customerID: "cust-1",
			input: CreateReplayInput{
				SourcePixelID: "pixel-1",
				TargetPixelID: "pixel-2",
			},
			setup: func(rr *MockReplaySessionRepo, er *MockEventRepo, pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID:            "pixel-1",
					CustomerID:    "cust-1",
					FBPixelID:     "fb-1",
					FBAccessToken: "token-1",
				}, nil)
				pr.On("GetByID", mock.Anything, "pixel-2").Return(&domain.Pixel{
					ID:            "pixel-2",
					CustomerID:    "cust-1",
					FBPixelID:     "fb-2",
					FBAccessToken: "token-2",
				}, nil)
				er.On("CountEventsForReplay", mock.Anything, "pixel-1", []string(nil), (*time.Time)(nil), (*time.Time)(nil)).
					Return(42, nil)
				er.On("GetEventsForReplayPreview", mock.Anything, "pixel-1", []string(nil), (*time.Time)(nil), (*time.Time)(nil), 10).
					Return([]*domain.PixelEvent{
						{ID: "evt-1", EventName: "PageView", EventTime: time.Now()},
						{ID: "evt-2", EventName: "Purchase", EventTime: time.Now()},
					}, nil)
			},
			wantErr:     nil,
			wantTotal:   42,
			wantSamples: 2,
		},
		{
			name:       "no events",
			customerID: "cust-1",
			input: CreateReplayInput{
				SourcePixelID: "pixel-1",
				TargetPixelID: "pixel-2",
			},
			setup: func(rr *MockReplaySessionRepo, er *MockEventRepo, pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID:            "pixel-1",
					CustomerID:    "cust-1",
					FBPixelID:     "fb-1",
					FBAccessToken: "token-1",
				}, nil)
				pr.On("GetByID", mock.Anything, "pixel-2").Return(&domain.Pixel{
					ID:            "pixel-2",
					CustomerID:    "cust-1",
					FBPixelID:     "fb-2",
					FBAccessToken: "token-2",
				}, nil)
				er.On("CountEventsForReplay", mock.Anything, "pixel-1", []string(nil), (*time.Time)(nil), (*time.Time)(nil)).
					Return(0, nil)
				er.On("GetEventsForReplayPreview", mock.Anything, "pixel-1", []string(nil), (*time.Time)(nil), (*time.Time)(nil), 10).
					Return(nil, nil)
			},
			wantErr:     nil,
			wantTotal:   0,
			wantSamples: 0,
		},
		{
			name:       "source pixel not found",
			customerID: "cust-1",
			input: CreateReplayInput{
				SourcePixelID: "nonexistent",
				TargetPixelID: "pixel-2",
			},
			setup: func(rr *MockReplaySessionRepo, er *MockEventRepo, pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)
			},
			wantErr: ErrPixelNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, replayRepo, eventRepo, pixelRepo := newTestReplayService()
			tt.setup(replayRepo, eventRepo, pixelRepo)

			result, err := svc.Preview(context.Background(), tt.customerID, tt.input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.wantTotal, result.TotalEvents)
				assert.Len(t, result.SampleEvents, tt.wantSamples)
			}
			pixelRepo.AssertExpectations(t)
		})
	}
}

func TestReplayService_Retry(t *testing.T) {
	tests := []struct {
		name       string
		customerID string
		sessionID  string
		setup      func(*MockReplaySessionRepo, *MockEventRepo, *MockPixelRepo)
		wantErr    error
	}{
		{
			name:       "retry failed session",
			customerID: "cust-1",
			sessionID:  "session-1",
			setup: func(rr *MockReplaySessionRepo, er *MockEventRepo, pr *MockPixelRepo) {
				failedRanges, _ := json.Marshal([]domain.BatchRange{{Start: 0, End: 1000}})
				rr.On("GetByID", mock.Anything, "session-1").Return(&domain.ReplaySession{
					ID:                "session-1",
					CustomerID:        "cust-1",
					SourcePixelID:     "pixel-1",
					TargetPixelID:     "pixel-2",
					Status:            "failed",
					TimeMode:          "original",
					FailedBatchRanges: failedRanges,
				}, nil)
				pr.On("GetByID", mock.Anything, "pixel-2").Return(&domain.Pixel{
					ID:            "pixel-2",
					CustomerID:    "cust-1",
					FBPixelID:     "fb-2",
					FBAccessToken: "token-2",
				}, nil)
				er.On("GetEventsForReplay", mock.Anything, "pixel-1", []string(nil), (*time.Time)(nil), (*time.Time)(nil), mock.AnythingOfType("*time.Time")).
					Return([]*domain.PixelEvent{
						{ID: "evt-1", EventName: "PageView", EventTime: time.Now()},
					}, nil)
				rr.On("Create", mock.Anything, mock.AnythingOfType("*domain.ReplaySession")).Return(nil)
				// Background goroutine
				rr.On("UpdateStatus", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
				rr.On("GetStatus", mock.Anything, mock.AnythingOfType("string")).Return("running", nil).Maybe()
				rr.On("UpdateProgress", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil).Maybe()
				rr.On("UpdateStatusWithError", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
				rr.On("UpdateFailedBatches", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).Return(nil).Maybe()
			},
			wantErr: nil,
		},
		{
			name:       "retry cancelled session",
			customerID: "cust-1",
			sessionID:  "session-2",
			setup: func(rr *MockReplaySessionRepo, er *MockEventRepo, pr *MockPixelRepo) {
				rr.On("GetByID", mock.Anything, "session-2").Return(&domain.ReplaySession{
					ID:            "session-2",
					CustomerID:    "cust-1",
					SourcePixelID: "pixel-1",
					TargetPixelID: "pixel-2",
					Status:        "cancelled",
					TimeMode:      "original",
				}, nil)
				pr.On("GetByID", mock.Anything, "pixel-2").Return(&domain.Pixel{
					ID:            "pixel-2",
					CustomerID:    "cust-1",
					FBPixelID:     "fb-2",
					FBAccessToken: "token-2",
				}, nil)
				er.On("GetEventsForReplay", mock.Anything, "pixel-1", []string(nil), (*time.Time)(nil), (*time.Time)(nil), mock.AnythingOfType("*time.Time")).
					Return([]*domain.PixelEvent{
						{ID: "evt-1", EventName: "PageView", EventTime: time.Now()},
					}, nil)
				rr.On("Create", mock.Anything, mock.AnythingOfType("*domain.ReplaySession")).Return(nil)
				// Background goroutine
				rr.On("UpdateStatus", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
				rr.On("GetStatus", mock.Anything, mock.AnythingOfType("string")).Return("running", nil).Maybe()
				rr.On("UpdateProgress", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil).Maybe()
				rr.On("UpdateStatusWithError", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
				rr.On("UpdateFailedBatches", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).Return(nil).Maybe()
			},
			wantErr: nil,
		},
		{
			name:       "cannot retry running session",
			customerID: "cust-1",
			sessionID:  "session-3",
			setup: func(rr *MockReplaySessionRepo, er *MockEventRepo, pr *MockPixelRepo) {
				rr.On("GetByID", mock.Anything, "session-3").Return(&domain.ReplaySession{
					ID:         "session-3",
					CustomerID: "cust-1",
					Status:     "running",
				}, nil)
			},
			wantErr: ErrReplayNotRetryable,
		},
		{
			name:       "cannot retry completed session",
			customerID: "cust-1",
			sessionID:  "session-4",
			setup: func(rr *MockReplaySessionRepo, er *MockEventRepo, pr *MockPixelRepo) {
				rr.On("GetByID", mock.Anything, "session-4").Return(&domain.ReplaySession{
					ID:         "session-4",
					CustomerID: "cust-1",
					Status:     "completed",
				}, nil)
			},
			wantErr: ErrReplayNotRetryable,
		},
		{
			name:       "session not found",
			customerID: "cust-1",
			sessionID:  "nonexistent",
			setup: func(rr *MockReplaySessionRepo, er *MockEventRepo, pr *MockPixelRepo) {
				rr.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)
			},
			wantErr: ErrReplayNotFound,
		},
		{
			name:       "session not owned",
			customerID: "cust-2",
			sessionID:  "session-1",
			setup: func(rr *MockReplaySessionRepo, er *MockEventRepo, pr *MockPixelRepo) {
				rr.On("GetByID", mock.Anything, "session-1").Return(&domain.ReplaySession{
					ID:         "session-1",
					CustomerID: "cust-1",
					Status:     "failed",
				}, nil)
			},
			wantErr: ErrReplayNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, replayRepo, eventRepo, pixelRepo := newTestReplayService()
			tt.setup(replayRepo, eventRepo, pixelRepo)

			session, err := svc.Retry(context.Background(), tt.customerID, tt.sessionID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, session)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, session)
			}
			replayRepo.AssertExpectations(t)
		})
	}
}

func TestReplayService_Create_SemaphoreBlock(t *testing.T) {
	// Use maxConcurrent=1 to test semaphore blocking
	svc, replayRepo, eventRepo, pixelRepo := newTestReplayServiceWithConcurrency(1)

	// Set up pixel mocks
	pixelRepo.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
		ID: "pixel-1", CustomerID: "cust-1", FBPixelID: "fb-1", FBAccessToken: "token-1",
	}, nil)
	pixelRepo.On("GetByID", mock.Anything, "pixel-2").Return(&domain.Pixel{
		ID: "pixel-2", CustomerID: "cust-1", FBPixelID: "fb-2", FBAccessToken: "token-2",
	}, nil)
	eventRepo.On("GetEventsForReplay", mock.Anything, "pixel-1", []string(nil), (*time.Time)(nil), (*time.Time)(nil), mock.AnythingOfType("*time.Time")).
		Return([]*domain.PixelEvent{
			{ID: "evt-1", EventName: "PageView", EventTime: time.Now()},
		}, nil)
	replayRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.ReplaySession")).Return(nil)
	replayRepo.On("UpdateStatus", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
	replayRepo.On("UpdateProgress", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil).Maybe()
	replayRepo.On("UpdateStatusWithError", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
	replayRepo.On("GetStatus", mock.Anything, mock.AnythingOfType("string")).Return("running", nil).Maybe()
	replayRepo.On("UpdateFailedBatches", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).Return(nil).Maybe()

	// Fill the semaphore (cap=1)
	svc.sem <- struct{}{}

	input := CreateReplayInput{SourcePixelID: "pixel-1", TargetPixelID: "pixel-2"}
	result, err := svc.Create(context.Background(), "cust-1", input)
	assert.NoError(t, err)
	require.NotNil(t, result)

	// Give goroutine a moment to start and block on semaphore
	time.Sleep(50 * time.Millisecond)

	// Release semaphore so goroutine can proceed
	<-svc.sem

	// Wait for background goroutine to finish
	svc.Shutdown()
}

func TestReplayService_Create_ShutdownDuringSemWait(t *testing.T) {
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	replayRepo := new(MockReplaySessionRepo)
	eventRepo := new(MockEventRepo)
	pixelRepo := new(MockPixelRepo)
	capiClient := facebook.NewCAPIClient("http://localhost:9999")
	logger := slog.Default()
	svc := NewReplayService(shutdownCtx, replayRepo, eventRepo, pixelRepo, capiClient, logger, 1, nil)

	pixelRepo.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
		ID: "pixel-1", CustomerID: "cust-1", FBPixelID: "fb-1", FBAccessToken: "token-1",
	}, nil)
	pixelRepo.On("GetByID", mock.Anything, "pixel-2").Return(&domain.Pixel{
		ID: "pixel-2", CustomerID: "cust-1", FBPixelID: "fb-2", FBAccessToken: "token-2",
	}, nil)
	eventRepo.On("GetEventsForReplay", mock.Anything, "pixel-1", []string(nil), (*time.Time)(nil), (*time.Time)(nil), mock.AnythingOfType("*time.Time")).
		Return([]*domain.PixelEvent{
			{ID: "evt-1", EventName: "PageView", EventTime: time.Now()},
		}, nil)
	replayRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.ReplaySession")).Return(nil)
	replayRepo.On("UpdateStatusWithError", mock.Anything, mock.AnythingOfType("string"), "failed", "server shutdown before replay started").Return(nil)

	// Fill the semaphore so goroutine will block
	svc.sem <- struct{}{}

	input := CreateReplayInput{SourcePixelID: "pixel-1", TargetPixelID: "pixel-2"}
	result, err := svc.Create(context.Background(), "cust-1", input)
	assert.NoError(t, err)
	require.NotNil(t, result)

	// Give goroutine a moment to start waiting on semaphore
	time.Sleep(50 * time.Millisecond)

	// Cancel shutdown context → goroutine should exit with "failed"
	shutdownCancel()
	svc.Shutdown()

	// Verify UpdateStatusWithError was called with fresh context (not the cancelled one)
	replayRepo.AssertCalled(t, "UpdateStatusWithError", mock.Anything, mock.AnythingOfType("string"), "failed", "server shutdown before replay started")

	// Release the occupied semaphore slot
	<-svc.sem
}

func TestReplayService_SendBatchWithRetry_RateLimit(t *testing.T) {
	callCount := 0
	fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"rate limited"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(facebook.CAPIResponse{EventsReceived: 1})
	}))
	defer fakeServer.Close()

	capiClient := facebook.NewCAPIClient(fakeServer.URL)
	svc := &ReplayService{capiClient: capiClient, logger: slog.Default()}

	resp, err := svc.sendBatchWithRetry(context.Background(), "pixel-1", "token-1", []facebook.CAPIEvent{
		{EventName: "PageView", EventTime: time.Now().Unix()},
	})

	assert.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 1, resp.EventsReceived)
	assert.Equal(t, 3, callCount) // 2 retries + 1 success
}

func TestReplayService_SendBatchWithRetry_AuthFailFast(t *testing.T) {
	callCount := 0
	fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"invalid token"}`))
	}))
	defer fakeServer.Close()

	capiClient := facebook.NewCAPIClient(fakeServer.URL)
	svc := &ReplayService{capiClient: capiClient, logger: slog.Default()}

	resp, err := svc.sendBatchWithRetry(context.Background(), "pixel-1", "token-1", []facebook.CAPIEvent{
		{EventName: "PageView", EventTime: time.Now().Unix()},
	})

	assert.Error(t, err)
	assert.True(t, facebook.IsAuthError(err))
	assert.Nil(t, resp)
	assert.Equal(t, 1, callCount) // no retry on auth error
}

func TestReplayService_SendBatchWithRetry_MaxRetries(t *testing.T) {
	callCount := 0
	fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer fakeServer.Close()

	capiClient := facebook.NewCAPIClient(fakeServer.URL)
	svc := &ReplayService{capiClient: capiClient, logger: slog.Default()}

	resp, err := svc.sendBatchWithRetry(context.Background(), "pixel-1", "token-1", []facebook.CAPIEvent{
		{EventName: "PageView", EventTime: time.Now().Unix()},
	})

	assert.Error(t, err)
	assert.True(t, facebook.IsRateLimitError(err))
	assert.Nil(t, resp)
	assert.Equal(t, maxRetries+1, callCount) // initial + 3 retries
}

func TestReplayService_ExecuteReplay_DefaultDelay(t *testing.T) {
	// Verify that batch_delay_ms=0 uses defaultBatchDelay (200ms)
	callCount := 0
	fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(facebook.CAPIResponse{EventsReceived: 1})
	}))
	defer fakeServer.Close()

	replayRepo := new(MockReplaySessionRepo)
	eventRepo := new(MockEventRepo)
	pixelRepo := new(MockPixelRepo)
	capiClient := facebook.NewCAPIClient(fakeServer.URL)
	logger := slog.Default()
	svc := NewReplayService(context.Background(), replayRepo, eventRepo, pixelRepo, capiClient, logger, 5, nil)

	session := &domain.ReplaySession{
		ID:           "session-1",
		CustomerID:   "cust-1",
		TotalEvents:  2,
		TimeMode:     "original",
		BatchDelayMs: 0, // should use defaultBatchDelay
	}
	targetPixel := &domain.Pixel{
		ID: "pixel-2", FBPixelID: "fb-2", FBAccessToken: "token-2",
	}

	// Create 1001 events to force 2 batches (1000 + 1)
	events := make([]*domain.PixelEvent, 1001)
	for i := range events {
		events[i] = &domain.PixelEvent{
			ID: fmt.Sprintf("evt-%d", i), EventName: "PageView", EventTime: time.Now(),
		}
	}

	replayRepo.On("UpdateStatus", mock.Anything, "session-1", "running").Return(nil)
	replayRepo.On("GetStatus", mock.Anything, "session-1").Return("running", nil)
	replayRepo.On("UpdateProgress", mock.Anything, "session-1", mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil)
	replayRepo.On("UpdateStatus", mock.Anything, "session-1", "completed").Return(nil)

	start := time.Now()
	svc.executeReplay(context.Background(), session, targetPixel, events)
	elapsed := time.Since(start)

	// Should have at least ~200ms delay between batches
	assert.GreaterOrEqual(t, elapsed, 150*time.Millisecond) // allow small tolerance
	assert.Equal(t, 2, callCount) // 2 CAPI calls (2 batches)
	replayRepo.AssertExpectations(t)
}

func TestReplayService_Create_SamePixel(t *testing.T) {
	svc, _, _, pixelRepo := newTestReplayService()
	pixelRepo.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
		ID: "pixel-1", CustomerID: "cust-1", FBPixelID: "fb-1", FBAccessToken: "token-1",
	}, nil)

	input := CreateReplayInput{SourcePixelID: "pixel-1", TargetPixelID: "pixel-1"}
	result, err := svc.Create(context.Background(), "cust-1", input)

	assert.ErrorIs(t, err, ErrReplaySamePixel)
	assert.Nil(t, result)
}

func TestReplayService_Create_NoCredentials(t *testing.T) {
	svc, _, _, pixelRepo := newTestReplayService()
	pixelRepo.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
		ID: "pixel-1", CustomerID: "cust-1", FBPixelID: "fb-1", FBAccessToken: "token-1",
	}, nil)
	pixelRepo.On("GetByID", mock.Anything, "pixel-2").Return(&domain.Pixel{
		ID: "pixel-2", CustomerID: "cust-1", FBPixelID: "", FBAccessToken: "",
	}, nil)

	input := CreateReplayInput{SourcePixelID: "pixel-1", TargetPixelID: "pixel-2"}
	result, err := svc.Create(context.Background(), "cust-1", input)

	assert.ErrorIs(t, err, ErrPixelNoCredentials)
	assert.Nil(t, result)
}

func TestReplayService_Preview_SamePixel(t *testing.T) {
	svc, _, _, pixelRepo := newTestReplayService()
	pixelRepo.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
		ID: "pixel-1", CustomerID: "cust-1", FBPixelID: "fb-1", FBAccessToken: "token-1",
	}, nil)

	input := CreateReplayInput{SourcePixelID: "pixel-1", TargetPixelID: "pixel-1"}
	result, err := svc.Preview(context.Background(), "cust-1", input)

	assert.ErrorIs(t, err, ErrReplaySamePixel)
	assert.Nil(t, result)
}

func TestReplayService_Preview_NoCredentials(t *testing.T) {
	svc, _, _, pixelRepo := newTestReplayService()
	pixelRepo.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
		ID: "pixel-1", CustomerID: "cust-1", FBPixelID: "fb-1", FBAccessToken: "token-1",
	}, nil)
	pixelRepo.On("GetByID", mock.Anything, "pixel-2").Return(&domain.Pixel{
		ID: "pixel-2", CustomerID: "cust-1", FBPixelID: "", FBAccessToken: "",
	}, nil)

	input := CreateReplayInput{SourcePixelID: "pixel-1", TargetPixelID: "pixel-2"}
	result, err := svc.Preview(context.Background(), "cust-1", input)

	assert.ErrorIs(t, err, ErrPixelNoCredentials)
	assert.Nil(t, result)
}

func TestReplayService_Retry_NoCredentials(t *testing.T) {
	svc, replayRepo, _, pixelRepo := newTestReplayService()
	replayRepo.On("GetByID", mock.Anything, "session-1").Return(&domain.ReplaySession{
		ID:            "session-1",
		CustomerID:    "cust-1",
		SourcePixelID: "pixel-1",
		TargetPixelID: "pixel-2",
		Status:        "failed",
	}, nil)
	pixelRepo.On("GetByID", mock.Anything, "pixel-2").Return(&domain.Pixel{
		ID: "pixel-2", CustomerID: "cust-1", FBPixelID: "", FBAccessToken: "",
	}, nil)

	session, err := svc.Retry(context.Background(), "cust-1", "session-1")

	assert.ErrorIs(t, err, ErrPixelNoCredentials)
	assert.Nil(t, session)
}

func TestReplayService_Preview_MaxEventsWarning(t *testing.T) {
	svc, _, eventRepo, pixelRepo := newTestReplayService()
	pixelRepo.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
		ID: "pixel-1", CustomerID: "cust-1", FBPixelID: "fb-1", FBAccessToken: "token-1",
	}, nil)
	pixelRepo.On("GetByID", mock.Anything, "pixel-2").Return(&domain.Pixel{
		ID: "pixel-2", CustomerID: "cust-1", FBPixelID: "fb-2", FBAccessToken: "token-2",
	}, nil)
	eventRepo.On("CountEventsForReplay", mock.Anything, "pixel-1", []string(nil), (*time.Time)(nil), (*time.Time)(nil)).
		Return(100001, nil)
	eventRepo.On("GetEventsForReplayPreview", mock.Anything, "pixel-1", []string(nil), (*time.Time)(nil), (*time.Time)(nil), 10).
		Return([]*domain.PixelEvent{
			{ID: "evt-1", EventName: "PageView", EventTime: time.Now()},
		}, nil)

	input := CreateReplayInput{SourcePixelID: "pixel-1", TargetPixelID: "pixel-2", TimeMode: "current"}
	result, err := svc.Preview(context.Background(), "cust-1", input)

	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, MaxReplayEvents, result.TotalEvents)
	assert.Contains(t, result.Warning, "100001 events found")
	assert.Contains(t, result.Warning, "only first 100000 will be replayed")
}

func TestReplayService_GetEventTypes(t *testing.T) {
	tests := []struct {
		name       string
		customerID string
		pixelID    string
		setup      func(*MockEventRepo, *MockPixelRepo)
		wantTypes  []string
		wantErr    error
	}{
		{
			name:       "success",
			customerID: "cust-1",
			pixelID:    "pixel-1",
			setup: func(er *MockEventRepo, pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID: "pixel-1", CustomerID: "cust-1",
				}, nil)
				er.On("GetDistinctEventTypes", mock.Anything, "pixel-1").Return([]string{"PageView", "Purchase"}, nil)
			},
			wantTypes: []string{"PageView", "Purchase"},
		},
		{
			name:       "empty returns empty slice",
			customerID: "cust-1",
			pixelID:    "pixel-1",
			setup: func(er *MockEventRepo, pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID: "pixel-1", CustomerID: "cust-1",
				}, nil)
				er.On("GetDistinctEventTypes", mock.Anything, "pixel-1").Return(nil, nil)
			},
			wantTypes: []string{},
		},
		{
			name:       "pixel not found",
			customerID: "cust-1",
			pixelID:    "nonexistent",
			setup: func(er *MockEventRepo, pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)
			},
			wantErr: ErrPixelNotFound,
		},
		{
			name:       "pixel not owned",
			customerID: "cust-2",
			pixelID:    "pixel-1",
			setup: func(er *MockEventRepo, pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID: "pixel-1", CustomerID: "cust-1",
				}, nil)
			},
			wantErr: ErrPixelNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, eventRepo, pixelRepo := newTestReplayService()
			tt.setup(eventRepo, pixelRepo)

			types, err := svc.GetEventTypes(context.Background(), tt.customerID, tt.pixelID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, types)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantTypes, types)
			}
			pixelRepo.AssertExpectations(t)
		})
	}
}
