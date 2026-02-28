package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

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
			name:       "sdk user_data fields pass through to stored event",
			customerID: "cust-1",
			input: IngestBatchInput{
				Events: []IngestEventInput{
					{
						PixelID:   "pixel-1",
						EventName: "PageView",
						EventData: json.RawMessage(`{}`),
						UserData:  json.RawMessage(`{"fbp":"fb.1.1709150000000.456789","fbc":"fb.1.1709150000000.test123","external_id":"usr-ext-001"}`),
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
					var ud map[string]interface{}
					if err := json.Unmarshal(e.UserData, &ud); err != nil {
						return false
					}
					return ud["fbp"] == "fb.1.1709150000000.456789" &&
						ud["fbc"] == "fb.1.1709150000000.test123" &&
						ud["external_id"] == "usr-ext-001"
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

			created, err := svc.Ingest(context.Background(), tt.customerID, tt.input, ClientContext{
				IP:        "192.168.1.1",
				UserAgent: "TestAgent/1.0",
			})

			assert.NoError(t, err)
			assert.Equal(t, tt.wantCreated, created)
			eventRepo.AssertExpectations(t)
			pixelRepo.AssertExpectations(t)
		})
	}
}

func TestEventService_Ingest_WithFBCookies(t *testing.T) {
	svc, eventRepo, pixelRepo := newTestEventService()

	pixelRepo.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
		ID:            "pixel-1",
		CustomerID:    "cust-1",
		IsActive:      true,
		FBPixelID:     "fb-pixel-1",
		FBAccessToken: "token-1",
	}, nil)
	eventRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.PixelEvent")).Return(nil)

	input := IngestBatchInput{
		Events: []IngestEventInput{
			{
				PixelID:   "pixel-1",
				EventName: "PageView",
				EventData: json.RawMessage(`{}`),
			},
		},
	}

	created, err := svc.Ingest(context.Background(), "cust-1", input, ClientContext{
		IP:        "10.0.0.1",
		UserAgent: "TestAgent/1.0",
		FBC:       "fb.1.1709150000000.test123",
		FBP:       "fb.1.1709150000000.456789",
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, created)
	eventRepo.AssertExpectations(t)
	pixelRepo.AssertExpectations(t)
}

func TestExtractFBClid(t *testing.T) {
	tests := []struct {
		name     string
		inputURL string
		expected string
	}{
		{
			name:     "URL with fbclid",
			inputURL: "https://example.com?fbclid=abc123",
			expected: "abc123",
		},
		{
			name:     "URL without fbclid",
			inputURL: "https://example.com?utm=test",
			expected: "",
		},
		{
			name:     "empty URL",
			inputURL: "",
			expected: "",
		},
		{
			name:     "malformed URL",
			inputURL: "://bad",
			expected: "",
		},
		{
			name:     "fbclid with empty value",
			inputURL: "https://example.com?fbclid=",
			expected: "",
		},
		{
			name:     "fbclid among multiple params",
			inputURL: "https://example.com?utm_source=fb&fbclid=xyz789&ref=home",
			expected: "xyz789",
		},
		{
			name:     "non-http scheme rejected",
			inputURL: "javascript:void(0)?fbclid=evil",
			expected: "",
		},
		{
			name:     "http scheme accepted",
			inputURL: "http://example.com?fbclid=abc123",
			expected: "abc123",
		},
		{
			name:     "relative URL rejected (no scheme)",
			inputURL: "/path?fbclid=abc",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFBClid(tt.inputURL)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidFBCookie(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"valid fbc", "fb.1.1709150000000.test123", true},
		{"valid fbp", "fb.1.1709150000000.456789", true},
		{"empty string", "", false},
		{"no fb prefix", "notfb.1.123.abc", false},
		{"too long", "fb." + strings.Repeat("x", 500), false},
		{"just fb.", "fb.", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isValidFBCookie(tt.value))
		})
	}
}

func TestEventService_ListByCustomerID(t *testing.T) {
	tests := []struct {
		name      string
		page      int
		perPage   int
		pixelID   string
		setup     func(*MockEventRepo)
		wantLen   int
		wantTotal int
	}{
		{
			name:    "success with results",
			page:    1,
			perPage: 10,
			pixelID: "",
			setup: func(er *MockEventRepo) {
				er.On("ListByCustomerID", mock.Anything, "cust-1", "", 10, 0).Return([]*domain.PixelEvent{
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
			pixelID: "",
			setup: func(er *MockEventRepo) {
				er.On("ListByCustomerID", mock.Anything, "cust-1", "", 50, 0).Return([]*domain.PixelEvent{}, 0, nil)
			},
			wantLen:   0,
			wantTotal: 0,
		},
		{
			name:    "with pixel_id filter",
			page:    1,
			perPage: 50,
			pixelID: "px-1",
			setup: func(er *MockEventRepo) {
				er.On("ListByCustomerID", mock.Anything, "cust-1", "px-1", 50, 0).Return([]*domain.PixelEvent{
					{ID: "evt-1", EventName: "PageView"},
				}, 1, nil)
			},
			wantLen:   1,
			wantTotal: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, eventRepo, _ := newTestEventService()
			tt.setup(eventRepo)

			events, total, err := svc.ListByCustomerID(context.Background(), "cust-1", tt.pixelID, tt.page, tt.perPage)

			assert.NoError(t, err)
			assert.Len(t, events, tt.wantLen)
			assert.Equal(t, tt.wantTotal, total)
			eventRepo.AssertExpectations(t)
		})
	}
}

func TestEventService_ListRecent_Success(t *testing.T) {
	svc, eventRepo, _ := newTestEventService()
	since := time.Now().Add(-5 * time.Minute)

	eventRepo.On("ListRecentByCustomerID", mock.Anything, "cust-1", since, "", 50).Return([]*domain.RealtimeEvent{
		{ID: "evt-1", PixelID: "px-1", PixelName: "My Pixel", EventName: "PageView", CreatedAt: since.Add(time.Second)},
		{ID: "evt-2", PixelID: "px-1", PixelName: "My Pixel", EventName: "Purchase", CreatedAt: since.Add(2 * time.Second)},
	}, nil)

	events, err := svc.ListRecent(context.Background(), "cust-1", since, "")

	assert.NoError(t, err)
	assert.Len(t, events, 2)
	assert.Equal(t, "evt-1", events[0].ID)
	assert.Equal(t, "evt-2", events[1].ID)
	eventRepo.AssertExpectations(t)
}

func TestEventService_ListRecent_EmptyResult(t *testing.T) {
	svc, eventRepo, _ := newTestEventService()
	since := time.Now()

	eventRepo.On("ListRecentByCustomerID", mock.Anything, "cust-1", since, "", 50).Return(nil, nil)

	events, err := svc.ListRecent(context.Background(), "cust-1", since, "")

	assert.NoError(t, err)
	assert.NotNil(t, events)
	assert.Len(t, events, 0)
	eventRepo.AssertExpectations(t)
}

func TestEventService_ListRecent_WithPixelIDFilter(t *testing.T) {
	svc, eventRepo, _ := newTestEventService()
	since := time.Now().Add(-10 * time.Minute)

	eventRepo.On("ListRecentByCustomerID", mock.Anything, "cust-1", since, "px-2", 50).Return([]*domain.RealtimeEvent{
		{ID: "evt-3", PixelID: "px-2", PixelName: "Other Pixel", EventName: "AddToCart", CreatedAt: since.Add(time.Second)},
	}, nil)

	events, err := svc.ListRecent(context.Background(), "cust-1", since, "px-2")

	assert.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, "px-2", events[0].PixelID)
	eventRepo.AssertExpectations(t)
}

func TestEventService_ListLatest_Success(t *testing.T) {
	svc, eventRepo, _ := newTestEventService()

	eventRepo.On("ListLatestByCustomerID", mock.Anything, "cust-1", "", 100).Return([]*domain.RealtimeEvent{
		{ID: "evt-1", PixelID: "px-1", PixelName: "My Pixel", EventName: "PageView"},
		{ID: "evt-2", PixelID: "px-1", PixelName: "My Pixel", EventName: "Purchase"},
	}, nil)

	events, err := svc.ListLatest(context.Background(), "cust-1", "", 100)

	assert.NoError(t, err)
	assert.Len(t, events, 2)
	assert.Equal(t, "evt-1", events[0].ID)
	eventRepo.AssertExpectations(t)
}

func TestEventService_ListLatest_DefaultLimit(t *testing.T) {
	svc, eventRepo, _ := newTestEventService()

	eventRepo.On("ListLatestByCustomerID", mock.Anything, "cust-1", "", 100).Return([]*domain.RealtimeEvent{}, nil)

	events, err := svc.ListLatest(context.Background(), "cust-1", "", 0)

	assert.NoError(t, err)
	assert.NotNil(t, events)
	assert.Len(t, events, 0)
	eventRepo.AssertExpectations(t)
}

func TestEventService_ListLatest_ClampMaxLimit(t *testing.T) {
	svc, eventRepo, _ := newTestEventService()

	eventRepo.On("ListLatestByCustomerID", mock.Anything, "cust-1", "px-1", 200).Return([]*domain.RealtimeEvent{}, nil)

	events, err := svc.ListLatest(context.Background(), "cust-1", "px-1", 500)

	assert.NoError(t, err)
	assert.NotNil(t, events)
	eventRepo.AssertExpectations(t)
}

func TestEventService_ListLatest_EmptyResult(t *testing.T) {
	svc, eventRepo, _ := newTestEventService()

	eventRepo.On("ListLatestByCustomerID", mock.Anything, "cust-1", "", 100).Return(nil, nil)

	events, err := svc.ListLatest(context.Background(), "cust-1", "", 100)

	assert.NoError(t, err)
	assert.NotNil(t, events)
	assert.Len(t, events, 0)
	eventRepo.AssertExpectations(t)
}
