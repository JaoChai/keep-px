package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/jaochai/pixlinks/backend/internal/config"
	"github.com/jaochai/pixlinks/backend/internal/repository/mocks"
	"github.com/jaochai/pixlinks/backend/internal/service"
)

const (
	testJWTSecret    = "test-secret"
	testCustomerID   = "cust-1"
	testPixelID      = "px-1"
	testPixelMissing = "px-nonexistent"
	testSalePageID   = "a0000000-0000-0000-0000-000000000001"
)

// testJWT generates a valid JWT token for testing with the given customerID and admin flag.
func testJWT(customerID string, isAdmin bool) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      customerID,
		"is_admin": isAdmin,
		"exp":      time.Now().Add(1 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	})
	tokenStr, err := token.SignedString([]byte(testJWTSecret))
	if err != nil {
		panic("failed to sign test JWT: " + err.Error())
	}
	return tokenStr
}

// doRequest creates an HTTP test request, optionally sets the Authorization header and JSON body,
// and returns the recorded response.
func doRequest(handler http.Handler, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			panic("failed to marshal request body: " + err.Error())
		}
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = &bytes.Buffer{}
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// testConfig returns a minimal config suitable for handler tests.
func testConfig() *config.Config {
	return &config.Config{
		JWTSecret:     testJWTSecret,
		JWTAccessTTL:  15 * time.Minute,
		JWTRefreshTTL: 7 * 24 * time.Hour,
		FrontendURL:   "http://localhost:5173",
	}
}

// testLogger returns a discard logger for tests.
func testLogger() *slog.Logger {
	return slog.Default()
}

// ---------------------------------------------------------------------------
// Service factory helpers — create real services backed by shared mock repos.
// ---------------------------------------------------------------------------

// newTestAuthService creates an AuthService with mock repos and test config.
func newTestAuthService(customerRepo *mocks.MockCustomerRepo, refreshTokenRepo *mocks.MockRefreshTokenRepo) *service.AuthService {
	return service.NewAuthService(customerRepo, refreshTokenRepo, testConfig())
}

// newTestQuotaService creates a QuotaService with mock repos.
func newTestQuotaService(
	creditRepo *mocks.MockReplayCreditRepo,
	subRepo *mocks.MockSubscriptionRepo,
	usageRepo *mocks.MockEventUsageRepo,
	pixelRepo *mocks.MockPixelRepo,
	salePageRepo *mocks.MockSalePageRepo,
	customerRepo *mocks.MockCustomerRepo,
) *service.QuotaService {
	return service.NewQuotaService(creditRepo, subRepo, usageRepo, pixelRepo, salePageRepo, customerRepo)
}

// newTestPixelService creates a PixelService with mock repos.
func newTestPixelService(pixelRepo *mocks.MockPixelRepo, quotaService *service.QuotaService) *service.PixelService {
	return service.NewPixelService(pixelRepo, nil, testLogger(), quotaService)
}

// newTestEventService creates an EventService with mock repos.
func newTestEventService(eventRepo *mocks.MockEventRepo, pixelRepo *mocks.MockPixelRepo, quotaService *service.QuotaService) *service.EventService {
	return service.NewEventService(eventRepo, pixelRepo, nil, testLogger(), quotaService)
}

// newTestSalePageService creates a SalePageService with mock repos.
func newTestSalePageService(
	salePageRepo *mocks.MockSalePageRepo,
	customerRepo *mocks.MockCustomerRepo,
	pixelRepo *mocks.MockPixelRepo,
	quotaService *service.QuotaService,
) *service.SalePageService {
	return service.NewSalePageService(context.Background(), salePageRepo, customerRepo, pixelRepo, quotaService, 60*time.Second)
}
