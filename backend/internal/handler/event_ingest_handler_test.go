package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/facebook"
	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/repository"
	"github.com/jaochai/pixlinks/backend/internal/service"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const eventTestCustomerID = "cust-evt-test"

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// eventRouter sets up a chi router with JWT auth and all event-related routes.
func eventRouter(h *EventHandler) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.JWTAuth(testJWTSecret))
	r.Route("/events", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/recent", h.ListRecent)
		r.Get("/{id}", h.GetByID)
	})
	return r
}

// eventIngestRouter sets up a chi router with JWT auth for the ingest endpoint.
// In production, ingest uses API key auth, but for handler tests we just need
// a customer ID in context; JWT is simpler for testing.
func eventIngestRouter(h *EventHandler) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.JWTAuth(testJWTSecret))
	r.Post("/events/ingest", h.Ingest)
	return r
}

// eventToken returns a signed JWT for event handler tests.
func eventToken(customerID string) string {
	return testJWT(customerID, false)
}

// eventParseResponse parses a JSON response body into a map.
func eventParseResponse(t *testing.T, rr *httpResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	return body
}

// newTestEventServiceWithCAPI creates an EventService with a dummy CAPI client
// that prevents nil-pointer panics in background goroutines during ingest tests.
func newTestEventServiceWithCAPI(eventRepo *MockEventRepo, pixelRepo *MockPixelRepo, quotaService *service.QuotaService) *service.EventService {
	capiClient := facebook.NewCAPIClient("http://localhost:19999")
	return service.NewEventService(eventRepo, pixelRepo, capiClient, slog.Default(), quotaService)
}

// ---------------------------------------------------------------------------
// Test: EventHandler.Ingest
// ---------------------------------------------------------------------------

func TestEventHandler_Ingest(t *testing.T) {
	pixelID := uuid.New().String()

	t.Run("success: batch of 2 events returns 202", func(t *testing.T) {
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		subRepo := &MockSubscriptionRepo{}
		usageRepo := &MockEventUsageRepo{}
		creditRepo := &MockReplayCreditRepo{}
		salePageRepo := &MockSalePageRepo{}
		customerRepo := &MockCustomerRepo{}

		quotaSvc := newTestQuotaService(creditRepo, subRepo, usageRepo, pixelRepo, salePageRepo, customerRepo)
		eventSvc := newTestEventServiceWithCAPI(eventRepo, pixelRepo, quotaSvc)
		h := NewEventHandler(eventSvc, testLogger())

		// Quota: allow events
		subRepo.On("GetMaxEventsPerMonth", mock.Anything, eventTestCustomerID).Return(int64(100000), nil)
		usageRepo.On("CheckAndIncrement", mock.Anything, eventTestCustomerID, int64(2), int64(100000)).Return(nil)

		// Pixel repo: batch fetch
		pixelRepo.On("GetByIDs", mock.Anything, mock.AnythingOfType("[]string")).Return([]*domain.Pixel{
			{ID: pixelID, CustomerID: eventTestCustomerID, FBPixelID: "fb-1", FBAccessToken: "tok", IsActive: true},
		}, nil)

		// Event repo: create events
		eventRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.PixelEvent")).Return(true, nil).Times(2)

		r := eventIngestRouter(h)
		token := eventToken(eventTestCustomerID)

		body := map[string]interface{}{
			"events": []map[string]interface{}{
				{"pixel_id": pixelID, "event_name": "PageView", "event_data": map[string]interface{}{}, "source_url": "https://example.com"},
				{"pixel_id": pixelID, "event_name": "Purchase", "event_data": map[string]interface{}{"value": 100}, "source_url": "https://example.com/buy"},
			},
		}

		rr := doRequest(r, "POST", "/events/ingest", body, token)

		assert.Equal(t, http.StatusAccepted, rr.Code)
		resp := eventParseResponse(t, rr)
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(2), data["events_accepted"])
	})

	t.Run("no auth header returns 401", func(t *testing.T) {
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		eventSvc := newTestEventService(eventRepo, pixelRepo, nil)
		h := NewEventHandler(eventSvc, testLogger())

		r := eventIngestRouter(h)

		body := map[string]interface{}{
			"events": []map[string]interface{}{
				{"pixel_id": pixelID, "event_name": "PageView"},
			},
		}

		rr := doRequest(r, "POST", "/events/ingest", body, "") // no token

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("empty events array returns 400", func(t *testing.T) {
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		eventSvc := newTestEventService(eventRepo, pixelRepo, nil)
		h := NewEventHandler(eventSvc, testLogger())

		r := eventIngestRouter(h)
		token := eventToken(eventTestCustomerID)

		body := map[string]interface{}{
			"events": []map[string]interface{}{},
		}

		rr := doRequest(r, "POST", "/events/ingest", body, token)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		resp := eventParseResponse(t, rr)
		assert.NotEmpty(t, resp["error"])
	})

	t.Run("quota exceeded returns 402", func(t *testing.T) {
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		subRepo := &MockSubscriptionRepo{}
		usageRepo := &MockEventUsageRepo{}
		creditRepo := &MockReplayCreditRepo{}
		salePageRepo := &MockSalePageRepo{}
		customerRepo := &MockCustomerRepo{}

		quotaSvc := newTestQuotaService(creditRepo, subRepo, usageRepo, pixelRepo, salePageRepo, customerRepo)
		eventSvc := newTestEventServiceWithCAPI(eventRepo, pixelRepo, quotaSvc)
		h := NewEventHandler(eventSvc, testLogger())

		// Quota: exceeded
		subRepo.On("GetMaxEventsPerMonth", mock.Anything, eventTestCustomerID).Return(int64(5000), nil)
		usageRepo.On("CheckAndIncrement", mock.Anything, eventTestCustomerID, int64(1), int64(5000)).Return(repository.ErrQuotaExceeded)

		r := eventIngestRouter(h)
		token := eventToken(eventTestCustomerID)

		body := map[string]interface{}{
			"events": []map[string]interface{}{
				{"pixel_id": pixelID, "event_name": "PageView"},
			},
		}

		rr := doRequest(r, "POST", "/events/ingest", body, token)

		assert.Equal(t, http.StatusPaymentRequired, rr.Code)
		resp := eventParseResponse(t, rr)
		assert.Contains(t, resp["error"].(string), "quota")
	})
}

// ---------------------------------------------------------------------------
// Test: EventHandler.List
// ---------------------------------------------------------------------------

func TestEventHandler_List(t *testing.T) {
	pixelID := uuid.New().String()

	t.Run("success: paginated response with defaults", func(t *testing.T) {
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		eventSvc := newTestEventService(eventRepo, pixelRepo, nil)
		h := NewEventHandler(eventSvc, testLogger())

		events := []*domain.PixelEvent{
			{ID: "e1", PixelID: pixelID, EventName: "PageView", EventData: json.RawMessage(`{}`)},
			{ID: "e2", PixelID: pixelID, EventName: "Purchase", EventData: json.RawMessage(`{}`)},
		}
		// Default: page=1, per_page=50 -> offset=0, limit=50
		eventRepo.On("ListByCustomerID", mock.Anything, eventTestCustomerID, "", "", 50, 0).Return(events, 2, nil)

		r := eventRouter(h)
		token := eventToken(eventTestCustomerID)

		rr := doRequest(r, "GET", "/events/", nil, token)

		assert.Equal(t, http.StatusOK, rr.Code)
		resp := eventParseResponse(t, rr)
		assert.Equal(t, float64(2), resp["total"])
		assert.Equal(t, float64(1), resp["page"])
		assert.Equal(t, float64(50), resp["per_page"])
		data := resp["data"].([]interface{})
		assert.Len(t, data, 2)
	})

	t.Run("with pixel_id filter", func(t *testing.T) {
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		eventSvc := newTestEventService(eventRepo, pixelRepo, nil)
		h := NewEventHandler(eventSvc, testLogger())

		events := []*domain.PixelEvent{
			{ID: "e1", PixelID: pixelID, EventName: "PageView", EventData: json.RawMessage(`{}`)},
		}
		eventRepo.On("ListByCustomerID", mock.Anything, eventTestCustomerID, pixelID, "", 50, 0).Return(events, 1, nil)

		r := eventRouter(h)
		token := eventToken(eventTestCustomerID)

		rr := doRequest(r, "GET", "/events/?pixel_id="+pixelID, nil, token)

		assert.Equal(t, http.StatusOK, rr.Code)
		resp := eventParseResponse(t, rr)
		assert.Equal(t, float64(1), resp["total"])
	})

	t.Run("invalid pixel_id returns 400", func(t *testing.T) {
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		eventSvc := newTestEventService(eventRepo, pixelRepo, nil)
		h := NewEventHandler(eventSvc, testLogger())

		r := eventRouter(h)
		token := eventToken(eventTestCustomerID)

		rr := doRequest(r, "GET", "/events/?pixel_id=not-a-uuid", nil, token)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

// ---------------------------------------------------------------------------
// Test: EventHandler.ListRecent
// ---------------------------------------------------------------------------

func TestEventHandler_ListRecent(t *testing.T) {
	t.Run("without since returns latest events", func(t *testing.T) {
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		eventSvc := newTestEventService(eventRepo, pixelRepo, nil)
		h := NewEventHandler(eventSvc, testLogger())

		realtimeEvents := []*domain.RealtimeEvent{
			{ID: "e1", PixelID: testPixelID, EventName: "PageView"},
		}
		// Without "since" and no "limit", handler passes limit=0, service clamps to 100
		eventRepo.On("ListLatestByCustomerID", mock.Anything, eventTestCustomerID, "", 100).Return(realtimeEvents, nil)

		r := eventRouter(h)
		token := eventToken(eventTestCustomerID)

		rr := doRequest(r, "GET", "/events/recent", nil, token)

		assert.Equal(t, http.StatusOK, rr.Code)
		resp := eventParseResponse(t, rr)
		data := resp["data"].([]interface{})
		assert.Len(t, data, 1)
	})

	t.Run("with since returns events after timestamp", func(t *testing.T) {
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		eventSvc := newTestEventService(eventRepo, pixelRepo, nil)
		h := NewEventHandler(eventSvc, testLogger())

		realtimeEvents := []*domain.RealtimeEvent{
			{ID: "e2", PixelID: testPixelID, EventName: "Purchase"},
		}
		// "since" within 5 minutes should pass through without clamping
		eventRepo.On("ListRecentByCustomerID", mock.Anything, eventTestCustomerID, mock.AnythingOfType("time.Time"), "", 50).Return(realtimeEvents, nil)

		r := eventRouter(h)
		token := eventToken(eventTestCustomerID)

		// Use UTC to avoid timezone offset issues with '+' encoding in URLs
		since := time.Now().UTC().Add(-1 * time.Minute).Format(time.RFC3339)
		rr := doRequest(r, "GET", "/events/recent?since="+since, nil, token)

		assert.Equal(t, http.StatusOK, rr.Code)
		resp := eventParseResponse(t, rr)
		data := resp["data"].([]interface{})
		assert.Len(t, data, 1)
	})

	t.Run("invalid since format returns 400", func(t *testing.T) {
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		eventSvc := newTestEventService(eventRepo, pixelRepo, nil)
		h := NewEventHandler(eventSvc, testLogger())

		r := eventRouter(h)
		token := eventToken(eventTestCustomerID)

		rr := doRequest(r, "GET", "/events/recent?since=not-a-date", nil, token)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

// ---------------------------------------------------------------------------
// Test: EventHandler.GetByID
// ---------------------------------------------------------------------------

func TestEventHandler_GetByID(t *testing.T) {
	eventID := uuid.New().String()

	t.Run("success returns 200 with event", func(t *testing.T) {
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		eventSvc := newTestEventService(eventRepo, pixelRepo, nil)
		h := NewEventHandler(eventSvc, testLogger())

		event := &domain.PixelEvent{
			ID:        eventID,
			PixelID:   testPixelID,
			EventName: "PageView",
			EventData: json.RawMessage(`{}`),
		}
		eventRepo.On("GetByID", mock.Anything, eventID).Return(event, nil)
		pixelRepo.On("GetByID", mock.Anything, testPixelID).Return(&domain.Pixel{
			ID:         testPixelID,
			CustomerID: eventTestCustomerID,
		}, nil)

		r := eventRouter(h)
		token := eventToken(eventTestCustomerID)

		rr := doRequest(r, "GET", "/events/"+eventID, nil, token)

		assert.Equal(t, http.StatusOK, rr.Code)
		resp := eventParseResponse(t, rr)
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, eventID, data["id"])
	})

	t.Run("not found returns 404", func(t *testing.T) {
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		eventSvc := newTestEventService(eventRepo, pixelRepo, nil)
		h := NewEventHandler(eventSvc, testLogger())

		eventRepo.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)

		r := eventRouter(h)
		token := eventToken(eventTestCustomerID)

		rr := doRequest(r, "GET", "/events/nonexistent", nil, token)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("event from different customer returns 404", func(t *testing.T) {
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		eventSvc := newTestEventService(eventRepo, pixelRepo, nil)
		h := NewEventHandler(eventSvc, testLogger())

		event := &domain.PixelEvent{
			ID:        eventID,
			PixelID:   "px-other",
			EventName: "PageView",
			EventData: json.RawMessage(`{}`),
		}
		eventRepo.On("GetByID", mock.Anything, eventID).Return(event, nil)
		pixelRepo.On("GetByID", mock.Anything, "px-other").Return(&domain.Pixel{
			ID:         "px-other",
			CustomerID: "other-customer",
		}, nil)

		r := eventRouter(h)
		token := eventToken(eventTestCustomerID)

		rr := doRequest(r, "GET", "/events/"+eventID, nil, token)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}
