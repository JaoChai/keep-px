package service

import (
	"context"
	"encoding/json"
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
	replayRepo := new(MockReplaySessionRepo)
	eventRepo := new(MockEventRepo)
	pixelRepo := new(MockPixelRepo)
	capiClient := facebook.NewCAPIClient("http://localhost:9999")
	logger := slog.Default()
	svc := NewReplayService(replayRepo, eventRepo, pixelRepo, capiClient, logger)
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
				er.On("GetEventsForReplay", mock.Anything, "pixel-1", []string(nil), (*time.Time)(nil), (*time.Time)(nil)).
					Return([]*domain.PixelEvent{
						{ID: "evt-1", EventName: "PageView", EventTime: time.Now().Add(-1 * time.Hour)},
					}, nil)
				rr.On("Create", mock.Anything, mock.AnythingOfType("*domain.ReplaySession")).Return(nil)
				// Background goroutine calls
				rr.On("UpdateStatus", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
				rr.On("UpdateProgress", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil).Maybe()
				rr.On("UpdateStatusWithError", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
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
				er.On("GetEventsForReplay", mock.Anything, "pixel-1", []string(nil), (*time.Time)(nil), (*time.Time)(nil)).
					Return([]*domain.PixelEvent{
						{ID: "evt-1", EventName: "PageView", EventTime: time.Now().Add(-1 * time.Hour)},
					}, nil)
				rr.On("Create", mock.Anything, mock.AnythingOfType("*domain.ReplaySession")).Return(nil)
				rr.On("UpdateStatus", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
				rr.On("UpdateProgress", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil).Maybe()
				rr.On("UpdateStatusWithError", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
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
				er.On("GetEventsForReplay", mock.Anything, "pixel-1", []string(nil), (*time.Time)(nil), (*time.Time)(nil)).
					Return([]*domain.PixelEvent{
						{ID: "evt-1", EventName: "PageView", EventTime: time.Now().Add(-1 * time.Hour)},
					}, nil)
				rr.On("Create", mock.Anything, mock.AnythingOfType("*domain.ReplaySession")).Return(nil)
				rr.On("UpdateStatus", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
				rr.On("UpdateProgress", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil).Maybe()
				rr.On("UpdateStatusWithError", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
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
				er.On("GetEventsForReplay", mock.Anything, "pixel-1", []string(nil), (*time.Time)(nil), (*time.Time)(nil)).
					Return([]*domain.PixelEvent{
						{ID: "evt-1", EventName: "PageView", EventTime: time.Now().Add(-10 * 24 * time.Hour)},
						{ID: "evt-2", EventName: "Purchase", EventTime: time.Now().Add(-1 * time.Hour)},
					}, nil)
				rr.On("Create", mock.Anything, mock.AnythingOfType("*domain.ReplaySession")).Return(nil)
				rr.On("UpdateStatus", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
				rr.On("UpdateProgress", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil).Maybe()
				rr.On("UpdateStatusWithError", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
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
	svc := NewReplayService(replayRepo, eventRepo, pixelRepo, capiClient, logger)

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
		json.NewDecoder(r.Body).Decode(&req)
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
	svc := NewReplayService(replayRepo, eventRepo, pixelRepo, capiClient, logger)

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
	svc := NewReplayService(replayRepo, eventRepo, pixelRepo, capiClient, logger)

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
	svc := NewReplayService(replayRepo, eventRepo, pixelRepo, capiClient, logger)

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
	replayRepo.On("UpdateProgress", mock.Anything, "session-1", mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil)
	replayRepo.On("UpdateStatusWithError", mock.Anything, "session-1", "failed", mock.AnythingOfType("string")).Return(nil)

	svc.executeReplay(context.Background(), session, targetPixel, events)

	replayRepo.AssertExpectations(t)
	// Should use UpdateStatusWithError for all-failed case, not UpdateStatus
	replayRepo.AssertCalled(t, "UpdateStatusWithError", mock.Anything, "session-1", "failed", mock.AnythingOfType("string"))
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
