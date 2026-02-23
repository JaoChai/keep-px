package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

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
		name       string
		customerID string
		input      CreateReplayInput
		setup      func(*MockReplaySessionRepo, *MockEventRepo, *MockPixelRepo)
		wantErr    error
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
						{ID: "evt-1", EventName: "PageView"},
					}, nil)
				rr.On("Create", mock.Anything, mock.AnythingOfType("*domain.ReplaySession")).Return(nil)
				// Background goroutine calls
				rr.On("UpdateStatus", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
				rr.On("UpdateProgress", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil).Maybe()
			},
			wantErr: nil,
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

			session, err := svc.Create(context.Background(), tt.customerID, tt.input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, session)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, session)
				assert.Equal(t, tt.customerID, session.CustomerID)
				assert.Equal(t, tt.input.SourcePixelID, session.SourcePixelID)
				assert.Equal(t, tt.input.TargetPixelID, session.TargetPixelID)
			}
			pixelRepo.AssertExpectations(t)
		})
	}
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
