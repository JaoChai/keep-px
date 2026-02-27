package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/facebook"
)

func newTestEventService() (*EventService, *MockEventRepo, *MockPixelRepo) {
	eventRepo := new(MockEventRepo)
	pixelRepo := new(MockPixelRepo)
	capiClient := facebook.NewCAPIClient("http://localhost:9999")
	logger := slog.Default()
	svc := NewEventService(eventRepo, pixelRepo, capiClient, logger)
	return svc, eventRepo, pixelRepo
}

func TestEventService_Ingest(t *testing.T) {
	tests := []struct {
		name        string
		customerID  string
		input       IngestBatchInput
		setup       func(*MockEventRepo, *MockPixelRepo)
		wantCreated int
	}{
		{
			name:       "valid pixel",
			customerID: "cust-1",
			input: IngestBatchInput{
				Events: []IngestEventInput{
					{
						PixelID:   "pixel-1",
						EventName: "PageView",
						EventData: json.RawMessage(`{"page":"home"}`),
					},
				},
			},
			setup: func(er *MockEventRepo, pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID:         "pixel-1",
					CustomerID: "cust-1",
					IsActive:   true,
				}, nil)
				er.On("Create", mock.Anything, mock.AnythingOfType("*domain.PixelEvent")).Return(nil)
			},
			wantCreated: 1,
		},
		{
			name:       "pixel not found",
			customerID: "cust-1",
			input: IngestBatchInput{
				Events: []IngestEventInput{
					{
						PixelID:   "nonexistent",
						EventName: "PageView",
					},
				},
			},
			setup: func(er *MockEventRepo, pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)
			},
			wantCreated: 0,
		},
		{
			name:       "pixel inactive",
			customerID: "cust-1",
			input: IngestBatchInput{
				Events: []IngestEventInput{
					{
						PixelID:   "pixel-2",
						EventName: "PageView",
					},
				},
			},
			setup: func(er *MockEventRepo, pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "pixel-2").Return(&domain.Pixel{
					ID:         "pixel-2",
					CustomerID: "cust-1",
					IsActive:   false,
				}, nil)
			},
			wantCreated: 0,
		},
		{
			name:       "pixel not owned by customer",
			customerID: "cust-2",
			input: IngestBatchInput{
				Events: []IngestEventInput{
					{
						PixelID:   "pixel-1",
						EventName: "PageView",
					},
				},
			},
			setup: func(er *MockEventRepo, pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID:         "pixel-1",
					CustomerID: "cust-1",
					IsActive:   true,
				}, nil)
			},
			wantCreated: 0,
		},
		{
			name:       "event_id from input preserved",
			customerID: "cust-1",
			input: IngestBatchInput{
				Events: []IngestEventInput{
					{
						PixelID:   "pixel-1",
						EventName: "Purchase",
						EventData: json.RawMessage(`{}`),
						EventID:   "custom-event-id-123",
					},
				},
			},
			setup: func(er *MockEventRepo, pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID:         "pixel-1",
					CustomerID: "cust-1",
					IsActive:   true,
				}, nil)
				er.On("Create", mock.Anything, mock.MatchedBy(func(e *domain.PixelEvent) bool {
					return e.EventID == "custom-event-id-123"
				})).Return(nil)
			},
			wantCreated: 1,
		},
		{
			name:       "event_id generated when empty",
			customerID: "cust-1",
			input: IngestBatchInput{
				Events: []IngestEventInput{
					{
						PixelID:   "pixel-1",
						EventName: "PageView",
						EventData: json.RawMessage(`{}`),
					},
				},
			},
			setup: func(er *MockEventRepo, pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID:         "pixel-1",
					CustomerID: "cust-1",
					IsActive:   true,
				}, nil)
				er.On("Create", mock.Anything, mock.MatchedBy(func(e *domain.PixelEvent) bool {
					return e.EventID != "" && len(e.EventID) == 36
				})).Return(nil)
			},
			wantCreated: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, eventRepo, pixelRepo := newTestEventService()
			tt.setup(eventRepo, pixelRepo)

			created, err := svc.Ingest(context.Background(), tt.customerID, tt.input, "192.168.1.1:12345", "TestAgent/1.0")

			assert.NoError(t, err)
			assert.Equal(t, tt.wantCreated, created)
			eventRepo.AssertExpectations(t)
			pixelRepo.AssertExpectations(t)
		})
	}
}

func TestEventService_ListByCustomerID(t *testing.T) {
	tests := []struct {
		name      string
		page      int
		perPage   int
		setup     func(*MockEventRepo)
		wantLen   int
		wantTotal int
	}{
		{
			name:    "success with results",
			page:    1,
			perPage: 10,
			setup: func(er *MockEventRepo) {
				er.On("ListByCustomerID", mock.Anything, "cust-1", 10, 0).Return([]*domain.PixelEvent{
					{ID: "evt-1", EventName: "PageView"},
					{ID: "evt-2", EventName: "Purchase"},
				}, 2, nil)
			},
			wantLen:   2,
			wantTotal: 2,
		},
		{
			name:    "defaults for invalid page values",
			page:    0,
			perPage: 0,
			setup: func(er *MockEventRepo) {
				er.On("ListByCustomerID", mock.Anything, "cust-1", 50, 0).Return([]*domain.PixelEvent{}, 0, nil)
			},
			wantLen:   0,
			wantTotal: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, eventRepo, _ := newTestEventService()
			tt.setup(eventRepo)

			events, total, err := svc.ListByCustomerID(context.Background(), "cust-1", tt.page, tt.perPage)

			assert.NoError(t, err)
			assert.Len(t, events, tt.wantLen)
			assert.Equal(t, tt.wantTotal, total)
			eventRepo.AssertExpectations(t)
		})
	}
}
