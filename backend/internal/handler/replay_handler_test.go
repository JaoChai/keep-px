package handler

import (
	"context"
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
	"github.com/jaochai/pixlinks/backend/internal/service"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const replayTestCustomerID = "cust-replay-test"

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// replayToken returns a signed JWT for replay handler tests.
func replayToken(customerID string) string {
	return testJWT(customerID, false)
}

// replayParseResponse parses a JSON response body into a map.
func replayParseResponse(t *testing.T, rr *httpResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	return body
}

// replayRouter creates a chi router with JWT auth and all replay routes wired up.
func replayRouter(h *ReplayHandler) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.JWTAuth(testJWTSecret))
	r.Route("/replays", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Post("/preview", h.Preview)
		r.Get("/event-types", h.EventTypes)
		r.Get("/{id}", h.GetByID)
		r.Post("/{id}/cancel", h.Cancel)
		r.Post("/{id}/retry", h.Retry)
	})
	return r
}

// replaySourcePixel returns a test source pixel owned by replayTestCustomerID.
func replaySourcePixel() *domain.Pixel {
	return &domain.Pixel{
		ID:            "px-src",
		CustomerID:    replayTestCustomerID,
		FBPixelID:     "fb-src",
		FBAccessToken: "tok-src",
		IsActive:      true,
	}
}

// replayTargetPixel returns a test target pixel owned by replayTestCustomerID.
func replayTargetPixel() *domain.Pixel {
	return &domain.Pixel{
		ID:            "px-tgt",
		CustomerID:    replayTestCustomerID,
		FBPixelID:     "fb-tgt",
		FBAccessToken: "tok-tgt",
		IsActive:      true,
	}
}

// replayTargetPixelNoCredentials returns a pixel with no Facebook credentials.
func replayTargetPixelNoCredentials() *domain.Pixel {
	return &domain.Pixel{
		ID:            "px-tgt-nocred",
		CustomerID:    replayTestCustomerID,
		FBPixelID:     "",
		FBAccessToken: "",
		IsActive:      true,
	}
}

// newTestReplayServiceWithCAPI creates a ReplayService with a dummy CAPI client
// that prevents nil-pointer panics in background goroutines during handler tests.
func newTestReplayServiceWithCAPI(
	replayRepo *MockReplaySessionRepo,
	eventRepo *MockEventRepo,
	pixelRepo *MockPixelRepo,
) *service.ReplayService {
	capiClient := facebook.NewCAPIClient("http://localhost:19999")
	return service.NewReplayService(
		context.Background(),
		replayRepo,
		eventRepo,
		pixelRepo,
		capiClient,
		slog.Default(),
		5,   // maxConcurrentReplays
		nil, // notifService
		nil, // quotaService
	)
}

// setupBackgroundReplayMocks sets up Maybe() expectations for background
// goroutine calls that may or may not fire during the test.
func setupBackgroundReplayMocks(sr *MockReplaySessionRepo) {
	sr.On("UpdateStatus", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
	sr.On("UpdateProgress", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil).Maybe()
	sr.On("UpdateStatusWithError", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Maybe()
	sr.On("GetStatus", mock.Anything, mock.AnythingOfType("string")).Return("running", nil).Maybe()
	sr.On("UpdateFailedBatches", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).Return(nil).Maybe()
}

// ---------------------------------------------------------------------------
// Test: ReplayHandler.Create
// ---------------------------------------------------------------------------

func TestReplayHandler_Create(t *testing.T) {
	t.Run("success returns 201 with session data", func(t *testing.T) {
		sessionRepo := &MockReplaySessionRepo{}
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		svc := newTestReplayServiceWithCAPI(sessionRepo, eventRepo, pixelRepo)
		h := NewReplayHandler(svc)

		pixelRepo.On("GetByID", mock.Anything, "px-src").Return(replaySourcePixel(), nil)
		pixelRepo.On("GetByID", mock.Anything, "px-tgt").Return(replayTargetPixel(), nil)
		eventRepo.On("CountEventsForReplay", mock.Anything, "px-src", []string(nil), (*time.Time)(nil), (*time.Time)(nil)).Return(2, nil)
		eventRepo.On("GetEventsForReplay", mock.Anything, "px-src", []string(nil), (*time.Time)(nil), (*time.Time)(nil), mock.AnythingOfType("*time.Time")).
			Return([]*domain.PixelEvent{
				{ID: "e1", EventName: "PageView", EventTime: time.Now().Add(-1 * time.Hour)},
				{ID: "e2", EventName: "Purchase", EventTime: time.Now().Add(-30 * time.Minute)},
			}, nil)
		sessionRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.ReplaySession")).Return(nil)
		setupBackgroundReplayMocks(sessionRepo)

		token := replayToken(replayTestCustomerID)
		rr := doRequest(replayRouter(h), "POST", "/replays/", map[string]interface{}{
			"source_pixel_id": "px-src",
			"target_pixel_id": "px-tgt",
		}, token)

		assert.Equal(t, http.StatusCreated, rr.Code)
		resp := replayParseResponse(t, rr)
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "px-src", data["source_pixel_id"])
		assert.Equal(t, "px-tgt", data["target_pixel_id"])
		assert.Equal(t, float64(2), data["total_events"])
	})

	t.Run("same pixel source=target returns 400", func(t *testing.T) {
		sessionRepo := &MockReplaySessionRepo{}
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		svc := newTestReplayService(sessionRepo, eventRepo, pixelRepo, nil, nil)
		h := NewReplayHandler(svc)

		pixelRepo.On("GetByID", mock.Anything, "px-src").Return(replaySourcePixel(), nil)

		token := replayToken(replayTestCustomerID)
		rr := doRequest(replayRouter(h), "POST", "/replays/", map[string]interface{}{
			"source_pixel_id": "px-src",
			"target_pixel_id": "px-src",
		}, token)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		resp := replayParseResponse(t, rr)
		assert.Contains(t, resp["error"].(string), "same")
	})

	t.Run("no credentials on target returns 422", func(t *testing.T) {
		sessionRepo := &MockReplaySessionRepo{}
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		svc := newTestReplayService(sessionRepo, eventRepo, pixelRepo, nil, nil)
		h := NewReplayHandler(svc)

		pixelRepo.On("GetByID", mock.Anything, "px-src").Return(replaySourcePixel(), nil)
		pixelRepo.On("GetByID", mock.Anything, "px-tgt-nocred").Return(replayTargetPixelNoCredentials(), nil)

		token := replayToken(replayTestCustomerID)
		rr := doRequest(replayRouter(h), "POST", "/replays/", map[string]interface{}{
			"source_pixel_id": "px-src",
			"target_pixel_id": "px-tgt-nocred",
		}, token)

		assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
		resp := replayParseResponse(t, rr)
		assert.Contains(t, resp["error"].(string), "credentials")
	})

	t.Run("pixel not found returns 404", func(t *testing.T) {
		sessionRepo := &MockReplaySessionRepo{}
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		svc := newTestReplayService(sessionRepo, eventRepo, pixelRepo, nil, nil)
		h := NewReplayHandler(svc)

		pixelRepo.On("GetByID", mock.Anything, "px-missing").Return(nil, nil)

		token := replayToken(replayTestCustomerID)
		rr := doRequest(replayRouter(h), "POST", "/replays/", map[string]interface{}{
			"source_pixel_id": "px-missing",
			"target_pixel_id": "px-tgt",
		}, token)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("missing required fields returns 400", func(t *testing.T) {
		sessionRepo := &MockReplaySessionRepo{}
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		svc := newTestReplayService(sessionRepo, eventRepo, pixelRepo, nil, nil)
		h := NewReplayHandler(svc)

		token := replayToken(replayTestCustomerID)
		rr := doRequest(replayRouter(h), "POST", "/replays/", map[string]interface{}{}, token)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

// ---------------------------------------------------------------------------
// Test: ReplayHandler.List
// ---------------------------------------------------------------------------

func TestReplayHandler_List(t *testing.T) {
	t.Run("success returns 200 with sessions array", func(t *testing.T) {
		sessionRepo := &MockReplaySessionRepo{}
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		svc := newTestReplayService(sessionRepo, eventRepo, pixelRepo, nil, nil)
		h := NewReplayHandler(svc)

		sessions := []*domain.ReplaySession{
			{ID: "s1", CustomerID: replayTestCustomerID, SourcePixelID: testPixelID, TargetPixelID: "px-2", Status: "completed", TotalEvents: 10},
			{ID: "s2", CustomerID: replayTestCustomerID, SourcePixelID: testPixelID, TargetPixelID: "px-3", Status: "running", TotalEvents: 5},
		}
		sessionRepo.On("ListByCustomerID", mock.Anything, replayTestCustomerID).Return(sessions, nil)

		token := replayToken(replayTestCustomerID)
		rr := doRequest(replayRouter(h), "GET", "/replays/", nil, token)

		assert.Equal(t, http.StatusOK, rr.Code)
		resp := replayParseResponse(t, rr)
		data := resp["data"].([]interface{})
		assert.Len(t, data, 2)
	})

	t.Run("empty list returns 200 with empty array", func(t *testing.T) {
		sessionRepo := &MockReplaySessionRepo{}
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		svc := newTestReplayService(sessionRepo, eventRepo, pixelRepo, nil, nil)
		h := NewReplayHandler(svc)

		sessionRepo.On("ListByCustomerID", mock.Anything, replayTestCustomerID).Return(nil, nil)

		token := replayToken(replayTestCustomerID)
		rr := doRequest(replayRouter(h), "GET", "/replays/", nil, token)

		assert.Equal(t, http.StatusOK, rr.Code)
		resp := replayParseResponse(t, rr)
		data := resp["data"].([]interface{})
		assert.Len(t, data, 0)
	})
}

// ---------------------------------------------------------------------------
// Test: ReplayHandler.GetByID
// ---------------------------------------------------------------------------

func TestReplayHandler_GetByID(t *testing.T) {
	sessionID := uuid.New().String()

	t.Run("success returns 200", func(t *testing.T) {
		sessionRepo := &MockReplaySessionRepo{}
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		svc := newTestReplayService(sessionRepo, eventRepo, pixelRepo, nil, nil)
		h := NewReplayHandler(svc)

		session := &domain.ReplaySession{
			ID:            sessionID,
			CustomerID:    replayTestCustomerID,
			SourcePixelID: testPixelID,
			TargetPixelID: "px-2",
			Status:        "completed",
			TotalEvents:   10,
		}
		sessionRepo.On("GetByID", mock.Anything, sessionID).Return(session, nil)

		token := replayToken(replayTestCustomerID)
		rr := doRequest(replayRouter(h), "GET", "/replays/"+sessionID, nil, token)

		assert.Equal(t, http.StatusOK, rr.Code)
		resp := replayParseResponse(t, rr)
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, sessionID, data["id"])
	})

	t.Run("not found returns 404", func(t *testing.T) {
		sessionRepo := &MockReplaySessionRepo{}
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		svc := newTestReplayService(sessionRepo, eventRepo, pixelRepo, nil, nil)
		h := NewReplayHandler(svc)

		sessionRepo.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)

		token := replayToken(replayTestCustomerID)
		rr := doRequest(replayRouter(h), "GET", "/replays/nonexistent", nil, token)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("other customer's session returns 404", func(t *testing.T) {
		sessionRepo := &MockReplaySessionRepo{}
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		svc := newTestReplayService(sessionRepo, eventRepo, pixelRepo, nil, nil)
		h := NewReplayHandler(svc)

		session := &domain.ReplaySession{
			ID:         sessionID,
			CustomerID: "other-customer",
		}
		sessionRepo.On("GetByID", mock.Anything, sessionID).Return(session, nil)

		token := replayToken(replayTestCustomerID)
		rr := doRequest(replayRouter(h), "GET", "/replays/"+sessionID, nil, token)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

// ---------------------------------------------------------------------------
// Test: ReplayHandler.Cancel
// ---------------------------------------------------------------------------

func TestReplayHandler_Cancel(t *testing.T) {
	sessionID := uuid.New().String()

	t.Run("success returns 200", func(t *testing.T) {
		sessionRepo := &MockReplaySessionRepo{}
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		svc := newTestReplayService(sessionRepo, eventRepo, pixelRepo, nil, nil)
		h := NewReplayHandler(svc)

		session := &domain.ReplaySession{
			ID:         sessionID,
			CustomerID: replayTestCustomerID,
			Status:     "running",
		}
		sessionRepo.On("GetByID", mock.Anything, sessionID).Return(session, nil)
		cancelled := &domain.ReplaySession{
			ID:         sessionID,
			CustomerID: replayTestCustomerID,
			Status:     "cancelled",
		}
		sessionRepo.On("CancelSession", mock.Anything, sessionID).Return(cancelled, nil)

		token := replayToken(replayTestCustomerID)
		rr := doRequest(replayRouter(h), "POST", "/replays/"+sessionID+"/cancel", nil, token)

		assert.Equal(t, http.StatusOK, rr.Code)
		resp := replayParseResponse(t, rr)
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "cancelled", data["status"])
	})

	t.Run("already completed returns 409", func(t *testing.T) {
		sessionRepo := &MockReplaySessionRepo{}
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		svc := newTestReplayService(sessionRepo, eventRepo, pixelRepo, nil, nil)
		h := NewReplayHandler(svc)

		session := &domain.ReplaySession{
			ID:         sessionID,
			CustomerID: replayTestCustomerID,
			Status:     "completed",
		}
		sessionRepo.On("GetByID", mock.Anything, sessionID).Return(session, nil)
		// CancelSession returns nil when already completed (cannot cancel)
		sessionRepo.On("CancelSession", mock.Anything, sessionID).Return(nil, nil)

		token := replayToken(replayTestCustomerID)
		rr := doRequest(replayRouter(h), "POST", "/replays/"+sessionID+"/cancel", nil, token)

		assert.Equal(t, http.StatusConflict, rr.Code)
		resp := replayParseResponse(t, rr)
		assert.Contains(t, resp["error"].(string), "cannot be cancelled")
	})

	t.Run("not found returns 404", func(t *testing.T) {
		sessionRepo := &MockReplaySessionRepo{}
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		svc := newTestReplayService(sessionRepo, eventRepo, pixelRepo, nil, nil)
		h := NewReplayHandler(svc)

		sessionRepo.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)

		token := replayToken(replayTestCustomerID)
		rr := doRequest(replayRouter(h), "POST", "/replays/nonexistent/cancel", nil, token)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

// ---------------------------------------------------------------------------
// Test: ReplayHandler.Retry
// ---------------------------------------------------------------------------

func TestReplayHandler_Retry(t *testing.T) {
	sessionID := uuid.New().String()

	t.Run("success returns 201 with new session", func(t *testing.T) {
		sessionRepo := &MockReplaySessionRepo{}
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		svc := newTestReplayServiceWithCAPI(sessionRepo, eventRepo, pixelRepo)
		h := NewReplayHandler(svc)

		failedSession := &domain.ReplaySession{
			ID:            sessionID,
			CustomerID:    replayTestCustomerID,
			SourcePixelID: "px-src",
			TargetPixelID: "px-tgt",
			Status:        "failed",
			TotalEvents:   5,
			TimeMode:      "original",
			CreatedAt:     time.Now().Add(-1 * time.Hour),
		}
		sessionRepo.On("GetByID", mock.Anything, sessionID).Return(failedSession, nil)
		pixelRepo.On("GetByID", mock.Anything, "px-tgt").Return(replayTargetPixel(), nil)
		eventRepo.On("GetEventsForReplay", mock.Anything, "px-src", []string(nil), (*time.Time)(nil), (*time.Time)(nil), mock.AnythingOfType("*time.Time")).
			Return([]*domain.PixelEvent{
				{ID: "e1", EventName: "PageView", EventTime: time.Now().Add(-2 * time.Hour)},
			}, nil)
		sessionRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.ReplaySession")).Return(nil)
		setupBackgroundReplayMocks(sessionRepo)

		token := replayToken(replayTestCustomerID)
		rr := doRequest(replayRouter(h), "POST", "/replays/"+sessionID+"/retry", nil, token)

		assert.Equal(t, http.StatusCreated, rr.Code)
		resp := replayParseResponse(t, rr)
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "px-src", data["source_pixel_id"])
	})

	t.Run("not retryable (running) returns 409", func(t *testing.T) {
		sessionRepo := &MockReplaySessionRepo{}
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		svc := newTestReplayService(sessionRepo, eventRepo, pixelRepo, nil, nil)
		h := NewReplayHandler(svc)

		runningSession := &domain.ReplaySession{
			ID:         sessionID,
			CustomerID: replayTestCustomerID,
			Status:     "running",
		}
		sessionRepo.On("GetByID", mock.Anything, sessionID).Return(runningSession, nil)

		token := replayToken(replayTestCustomerID)
		rr := doRequest(replayRouter(h), "POST", "/replays/"+sessionID+"/retry", nil, token)

		assert.Equal(t, http.StatusConflict, rr.Code)
		resp := replayParseResponse(t, rr)
		assert.Contains(t, resp["error"].(string), "cannot be retried")
	})

	t.Run("not found returns 404", func(t *testing.T) {
		sessionRepo := &MockReplaySessionRepo{}
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		svc := newTestReplayService(sessionRepo, eventRepo, pixelRepo, nil, nil)
		h := NewReplayHandler(svc)

		sessionRepo.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)

		token := replayToken(replayTestCustomerID)
		rr := doRequest(replayRouter(h), "POST", "/replays/nonexistent/retry", nil, token)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

// ---------------------------------------------------------------------------
// Test: ReplayHandler.EventTypes
// ---------------------------------------------------------------------------

func TestReplayHandler_EventTypes(t *testing.T) {
	pixelID := uuid.New().String()

	t.Run("success with pixel_id returns 200", func(t *testing.T) {
		sessionRepo := &MockReplaySessionRepo{}
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		svc := newTestReplayService(sessionRepo, eventRepo, pixelRepo, nil, nil)
		h := NewReplayHandler(svc)

		pixelRepo.On("GetByID", mock.Anything, pixelID).Return(&domain.Pixel{
			ID:         pixelID,
			CustomerID: replayTestCustomerID,
		}, nil)
		eventRepo.On("GetDistinctEventTypes", mock.Anything, pixelID).Return([]string{"PageView", "Purchase", "Lead"}, nil)

		token := replayToken(replayTestCustomerID)
		rr := doRequest(replayRouter(h), "GET", "/replays/event-types?pixel_id="+pixelID, nil, token)

		assert.Equal(t, http.StatusOK, rr.Code)
		resp := replayParseResponse(t, rr)
		data := resp["data"].([]interface{})
		assert.Len(t, data, 3)
		assert.Equal(t, "PageView", data[0])
	})

	t.Run("missing pixel_id returns 400", func(t *testing.T) {
		sessionRepo := &MockReplaySessionRepo{}
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		svc := newTestReplayService(sessionRepo, eventRepo, pixelRepo, nil, nil)
		h := NewReplayHandler(svc)

		token := replayToken(replayTestCustomerID)
		rr := doRequest(replayRouter(h), "GET", "/replays/event-types", nil, token)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		resp := replayParseResponse(t, rr)
		assert.Contains(t, resp["error"].(string), "pixel_id")
	})

	t.Run("pixel not found returns 404", func(t *testing.T) {
		sessionRepo := &MockReplaySessionRepo{}
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		svc := newTestReplayService(sessionRepo, eventRepo, pixelRepo, nil, nil)
		h := NewReplayHandler(svc)

		pixelRepo.On("GetByID", mock.Anything, "px-unknown").Return(nil, nil)

		token := replayToken(replayTestCustomerID)
		rr := doRequest(replayRouter(h), "GET", "/replays/event-types?pixel_id=px-unknown", nil, token)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

// ---------------------------------------------------------------------------
// Test: ReplayHandler.Preview
// ---------------------------------------------------------------------------

func TestReplayHandler_Preview(t *testing.T) {
	t.Run("success returns 200 with event count and sample", func(t *testing.T) {
		sessionRepo := &MockReplaySessionRepo{}
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		svc := newTestReplayService(sessionRepo, eventRepo, pixelRepo, nil, nil)
		h := NewReplayHandler(svc)

		pixelRepo.On("GetByID", mock.Anything, "px-src").Return(replaySourcePixel(), nil)
		pixelRepo.On("GetByID", mock.Anything, "px-tgt").Return(replayTargetPixel(), nil)
		eventRepo.On("CountEventsForReplay", mock.Anything, "px-src", []string(nil), (*time.Time)(nil), (*time.Time)(nil)).Return(42, nil)
		eventRepo.On("GetEventsForReplayPreview", mock.Anything, "px-src", []string(nil), (*time.Time)(nil), (*time.Time)(nil), 10).
			Return([]*domain.PixelEvent{
				{ID: "e1", EventName: "PageView", EventTime: time.Now(), EventData: json.RawMessage(`{}`)},
				{ID: "e2", EventName: "Purchase", EventTime: time.Now(), EventData: json.RawMessage(`{"value":100}`)},
			}, nil)

		token := replayToken(replayTestCustomerID)
		rr := doRequest(replayRouter(h), "POST", "/replays/preview", map[string]interface{}{
			"source_pixel_id": "px-src",
			"target_pixel_id": "px-tgt",
		}, token)

		assert.Equal(t, http.StatusOK, rr.Code)
		resp := replayParseResponse(t, rr)
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(42), data["total_events"])
		samples := data["sample_events"].([]interface{})
		assert.Len(t, samples, 2)
	})

	t.Run("same pixel returns 400", func(t *testing.T) {
		sessionRepo := &MockReplaySessionRepo{}
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		svc := newTestReplayService(sessionRepo, eventRepo, pixelRepo, nil, nil)
		h := NewReplayHandler(svc)

		pixelRepo.On("GetByID", mock.Anything, "px-src").Return(replaySourcePixel(), nil)

		token := replayToken(replayTestCustomerID)
		rr := doRequest(replayRouter(h), "POST", "/replays/preview", map[string]interface{}{
			"source_pixel_id": "px-src",
			"target_pixel_id": "px-src",
		}, token)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("missing required fields returns 400", func(t *testing.T) {
		sessionRepo := &MockReplaySessionRepo{}
		eventRepo := &MockEventRepo{}
		pixelRepo := &MockPixelRepo{}
		svc := newTestReplayService(sessionRepo, eventRepo, pixelRepo, nil, nil)
		h := NewReplayHandler(svc)

		token := replayToken(replayTestCustomerID)
		rr := doRequest(replayRouter(h), "POST", "/replays/preview", map[string]interface{}{}, token)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
