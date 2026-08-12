package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository/mocks"
	"github.com/jaochai/pixlinks/backend/internal/service"
	"github.com/jaochai/pixlinks/backend/internal/testutil"
)

const (
	testCustomerID   = "cust-1"
	testPixelID      = "px-1"
	testPixelMissing = "px-nonexistent"
	testSalePageID   = "a0000000-0000-0000-0000-000000000001"
	testAuthIssuer   = testutil.TestAuthIssuer
)

// testAuth เป็น registry รวมกุญแจ Ed25519 + lookup สำหรับเทสต์ทุกตัวในแพ็กเกจนี้
// (handler test ทั้งหมดเป็น package handler จึงเข้าถึงตัวแปรนี้ได้โดยตรง)
// handler test รันตามลำดับ (ไม่มี t.Parallel) จึงใช้ map ร่วมกันได้ปลอดภัย
var testAuth = testutil.NewTestAuth()

// testJWT generates a valid EdDSA JWT for testing with the given customerID and admin flag.
// สิทธิ์ isAdmin ถูกเก็บใน customer ที่ลงทะเบียนใน testAuth (ไม่ใช่ใน claim) สอดคล้องกับ
// middleware ใหม่ที่อ่านสิทธิ์จาก lookup ไม่ใช่จาก claim ที่ปลอมได้
func testJWT(customerID string, isAdmin bool) string {
	return testAuth.MintToken(customerID, &domain.Customer{ID: customerID, IsAdmin: isAdmin})
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

// testLogger returns a discard logger for tests.
func testLogger() *slog.Logger {
	return slog.Default()
}

// ---------------------------------------------------------------------------
// Service factory helpers — create real services backed by shared mock repos.
// ---------------------------------------------------------------------------

// newTestAuthService creates an AuthService with a mock customer repo.
func newTestAuthService(customerRepo *mocks.MockCustomerRepo) *service.AuthService {
	return service.NewAuthService(customerRepo)
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
