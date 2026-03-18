package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/middleware"
	"github.com/jaochai/pixlinks/backend/internal/repository"
	"github.com/jaochai/pixlinks/backend/internal/service"
)

// ---------------------------------------------------------------------------
// Test setup
// ---------------------------------------------------------------------------

type adminTestEnv struct {
	adminRepo    *MockAdminRepo
	customerRepo *MockCustomerRepo
	creditRepo   *MockReplayCreditRepo
	replayRepo   *MockReplaySessionRepo
	handler      *AdminHandler
	router       http.Handler
}

func setupAdminTest(t *testing.T) *adminTestEnv {
	t.Helper()

	adminRepo := &MockAdminRepo{}
	customerRepo := &MockCustomerRepo{}
	creditRepo := &MockReplayCreditRepo{}
	replayRepo := &MockReplaySessionRepo{}

	adminService := service.NewAdminService(adminRepo, customerRepo, creditRepo, replayRepo, nil, testLogger())
	h := NewAdminHandler(adminService, testLogger())

	r := chi.NewRouter()
	r.Use(middleware.JWTAuth(testJWTSecret))
	r.Use(middleware.AdminOnly)

	// Customer management
	r.Get("/admin/customers", h.ListCustomers)
	r.Get("/admin/customers/{id}", h.GetCustomerDetail)
	r.Put("/admin/customers/{id}/plan", h.ChangePlan)
	r.Post("/admin/customers/{id}/suspend", h.SuspendCustomer)
	r.Post("/admin/customers/{id}/activate", h.ActivateCustomer)
	r.Post("/admin/customers/{id}/credits", h.GrantCredits)

	// Platform
	r.Get("/admin/overview", h.GetPlatformOverview)
	r.Get("/admin/revenue-chart", h.GetRevenueChart)
	r.Get("/admin/growth-chart", h.GetGrowthChart)

	// Billing
	r.Get("/admin/purchases", h.ListPurchases)
	r.Get("/admin/subscriptions", h.ListSubscriptions)
	r.Get("/admin/credit-grants", h.ListCreditGrants)

	// Sale pages
	r.Get("/admin/sale-pages", h.ListSalePages)
	r.Get("/admin/sale-pages/{id}", h.GetSalePageDetail)
	r.Post("/admin/sale-pages/{id}/disable", h.DisableSalePage)
	r.Post("/admin/sale-pages/{id}/enable", h.EnableSalePage)
	r.Delete("/admin/sale-pages/{id}", h.DeleteSalePage)

	// Pixels
	r.Get("/admin/pixels", h.ListPixels)
	r.Get("/admin/pixels/{id}", h.GetPixelDetail)
	r.Post("/admin/pixels/{id}/disable", h.DisablePixel)
	r.Post("/admin/pixels/{id}/enable", h.EnablePixel)

	// Replays
	r.Get("/admin/replays", h.ListReplays)
	r.Get("/admin/replays/{id}", h.GetReplayDetail)
	r.Post("/admin/replays/{id}/cancel", h.CancelReplay)

	// Events
	r.Get("/admin/events", h.ListEvents)
	r.Get("/admin/events/stats", h.GetEventStats)

	// Audit log
	r.Get("/admin/audit-log", h.ListAuditLog)

	return &adminTestEnv{
		adminRepo:    adminRepo,
		customerRepo: customerRepo,
		creditRepo:   creditRepo,
		replayRepo:   replayRepo,
		handler:      h,
		router:       r,
	}
}

var errDB = errors.New("database error")

// ---------------------------------------------------------------------------
// Tests: Auth / AdminOnly middleware
// ---------------------------------------------------------------------------

func TestAdminHandler_AuthRequired(t *testing.T) {
	t.Run("no auth returns 401", func(t *testing.T) {
		env := setupAdminTest(t)
		rec := doRequest(env.router, http.MethodGet, "/admin/customers", nil, "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("non-admin JWT returns 403", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, false)
		rec := doRequest(env.router, http.MethodGet, "/admin/customers", nil, token)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: ListCustomers
// ---------------------------------------------------------------------------

func TestAdminHandler_ListCustomers(t *testing.T) {
	t.Run("success returns paginated customers", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		customers := []*domain.Customer{
			{ID: "c1", Email: "a@example.com", Name: "Alice", Plan: domain.PlanSandbox, APIKey: "secret-key-1", PasswordHash: "hash1"},
			{ID: "c2", Email: "b@example.com", Name: "Bob", Plan: domain.PlanPaid, APIKey: "secret-key-2", PasswordHash: "hash2"},
		}
		env.adminRepo.On("ListCustomers", mock.Anything, "", "", "", 20, 0).
			Return(customers, 2, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/customers", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp PaginatedResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, 2, resp.Total)
		assert.Equal(t, 1, resp.Page)
		assert.Equal(t, 20, resp.PerPage)
		assert.Equal(t, 1, resp.TotalPages)
	})

	t.Run("success with search filter", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		customers := []*domain.Customer{
			{ID: "c1", Email: "alice@example.com", Name: "Alice"},
		}
		env.adminRepo.On("ListCustomers", mock.Anything, "alice", "", "", 20, 0).
			Return(customers, 1, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/customers?search=alice", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp PaginatedResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, 1, resp.Total)
	})

	t.Run("success with pagination params", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		customers := []*domain.Customer{
			{ID: "c3", Email: "c@example.com", Name: "Charlie"},
		}
		// page=2, per_page=10 => offset=10
		env.adminRepo.On("ListCustomers", mock.Anything, "", "", "", 10, 10).
			Return(customers, 15, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/customers?page=2&per_page=10", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp PaginatedResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, 15, resp.Total)
		assert.Equal(t, 2, resp.Page)
		assert.Equal(t, 10, resp.PerPage)
		assert.Equal(t, 2, resp.TotalPages)
	})

	t.Run("success with plan and status filters", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("ListCustomers", mock.Anything, "", "sandbox", "active", 20, 0).
			Return([]*domain.Customer{}, 0, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/customers?plan=sandbox&status=active", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("invalid plan returns 400", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		rec := doRequest(env.router, http.MethodGet, "/admin/customers?plan=invalid_plan", nil, token)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Contains(t, resp.Error, "invalid plan")
	})

	t.Run("internal error returns 500", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("ListCustomers", mock.Anything, "", "", "", 20, 0).
			Return(nil, 0, errDB)

		rec := doRequest(env.router, http.MethodGet, "/admin/customers", nil, token)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("total pages calculation with remainder", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("ListCustomers", mock.Anything, "", "", "", 10, 0).
			Return([]*domain.Customer{}, 25, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/customers?per_page=10", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp PaginatedResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, 3, resp.TotalPages) // 25/10 = 2 remainder 5 => 3
	})
}

// ---------------------------------------------------------------------------
// Tests: GetCustomerDetail
// ---------------------------------------------------------------------------

func TestAdminHandler_GetCustomerDetail(t *testing.T) {
	t.Run("success returns customer detail", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)
		targetID := "target-cust"

		detail := &domain.AdminCustomerDetail{
			Customer:   &domain.Customer{ID: targetID, Email: "target@example.com", Name: "Target", PasswordHash: "secret-hash"},
			PixelCount: 3,
			EventCount: 100,
		}
		env.adminRepo.On("GetCustomerDetail", mock.Anything, targetID).
			Return(detail, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/customers/"+targetID, nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.NotNil(t, resp.Data)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("GetCustomerDetail", mock.Anything, "nonexistent").
			Return((*domain.AdminCustomerDetail)(nil), nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/customers/nonexistent", nil, token)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Contains(t, resp.Error, "customer not found")
	})

	t.Run("internal error returns 500", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("GetCustomerDetail", mock.Anything, "err-cust").
			Return((*domain.AdminCustomerDetail)(nil), errDB)

		rec := doRequest(env.router, http.MethodGet, "/admin/customers/err-cust", nil, token)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: ChangePlan
// ---------------------------------------------------------------------------

func TestAdminHandler_ChangePlan(t *testing.T) {
	t.Run("success changes plan", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)
		targetID := "target-cust"

		env.customerRepo.On("GetByID", mock.Anything, targetID).
			Return(&domain.Customer{ID: targetID, Plan: domain.PlanSandbox}, nil)
		env.customerRepo.On("UpdatePlan", mock.Anything, targetID, domain.PlanPaid).
			Return(nil)
		env.adminRepo.On("CreateAuditLog", mock.Anything, mock.Anything).Return(nil)

		body := map[string]string{"plan": "paid"}
		rec := doRequest(env.router, http.MethodPut, "/admin/customers/"+targetID+"/plan", body, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, "plan updated", resp.Message)
	})

	t.Run("invalid plan returns 400", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		body := map[string]string{"plan": "invalid"}
		rec := doRequest(env.router, http.MethodPut, "/admin/customers/some-id/plan", body, token)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("missing plan returns 400", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		body := map[string]string{}
		rec := doRequest(env.router, http.MethodPut, "/admin/customers/some-id/plan", body, token)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid JSON body returns 400", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		rec := doRequest(env.router, http.MethodPut, "/admin/customers/some-id/plan", "not-json{{{", token)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("customer not found returns 404", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.customerRepo.On("GetByID", mock.Anything, "nonexistent").
			Return((*domain.Customer)(nil), nil)

		body := map[string]string{"plan": "paid"}
		rec := doRequest(env.router, http.MethodPut, "/admin/customers/nonexistent/plan", body, token)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: SuspendCustomer
// ---------------------------------------------------------------------------

func TestAdminHandler_SuspendCustomer(t *testing.T) {
	t.Run("success suspends customer", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)
		targetID := "target-cust"

		env.adminRepo.On("SuspendCustomer", mock.Anything, targetID).Return(nil)
		env.adminRepo.On("CreateAuditLog", mock.Anything, mock.Anything).Return(nil)

		rec := doRequest(env.router, http.MethodPost, "/admin/customers/"+targetID+"/suspend", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, "customer suspended", resp.Message)
	})

	t.Run("self-suspend returns 400", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		rec := doRequest(env.router, http.MethodPost, "/admin/customers/"+testCustomerID+"/suspend", nil, token)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Contains(t, resp.Error, "cannot suspend your own account")
	})

	t.Run("not found returns 404", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("SuspendCustomer", mock.Anything, "nonexistent").
			Return(repository.ErrNotFound)

		rec := doRequest(env.router, http.MethodPost, "/admin/customers/nonexistent/suspend", nil, token)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Contains(t, resp.Error, "customer not found")
	})

	t.Run("internal error returns 500", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("SuspendCustomer", mock.Anything, "err-cust").Return(errDB)

		rec := doRequest(env.router, http.MethodPost, "/admin/customers/err-cust/suspend", nil, token)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: ActivateCustomer
// ---------------------------------------------------------------------------

func TestAdminHandler_ActivateCustomer(t *testing.T) {
	t.Run("success activates customer", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)
		targetID := "target-cust"

		env.adminRepo.On("ActivateCustomer", mock.Anything, targetID).Return(nil)
		env.adminRepo.On("CreateAuditLog", mock.Anything, mock.Anything).Return(nil)

		rec := doRequest(env.router, http.MethodPost, "/admin/customers/"+targetID+"/activate", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, "customer activated", resp.Message)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("ActivateCustomer", mock.Anything, "nonexistent").
			Return(repository.ErrNotFound)

		rec := doRequest(env.router, http.MethodPost, "/admin/customers/nonexistent/activate", nil, token)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Contains(t, resp.Error, "customer not found")
	})

	t.Run("internal error returns 500", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("ActivateCustomer", mock.Anything, "err-cust").Return(errDB)

		rec := doRequest(env.router, http.MethodPost, "/admin/customers/err-cust/activate", nil, token)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: GrantCredits
// ---------------------------------------------------------------------------

func TestAdminHandler_GrantCredits(t *testing.T) {
	t.Run("customer not found from GetByID returns 404", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.customerRepo.On("GetByID", mock.Anything, "gone-cust").
			Return((*domain.Customer)(nil), nil)

		body := map[string]interface{}{
			"pack_type":             "starter",
			"total_replays":         10,
			"max_events_per_replay": 1000,
			"expiry_days":           30,
		}
		rec := doRequest(env.router, http.MethodPost, "/admin/customers/gone-cust/credits", body, token)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("missing required fields returns 400", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		body := map[string]interface{}{
			"pack_type": "starter",
			// missing total_replays, max_events_per_replay, expiry_days
		}
		rec := doRequest(env.router, http.MethodPost, "/admin/customers/some-id/credits", body, token)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		rec := doRequest(env.router, http.MethodPost, "/admin/customers/some-id/credits", "invalid{json", token)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Contains(t, resp.Error, "invalid request body")
	})

	t.Run("zero total_replays returns 400", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		body := map[string]interface{}{
			"pack_type":             "starter",
			"total_replays":         0,
			"max_events_per_replay": 1000,
			"expiry_days":           30,
		}
		rec := doRequest(env.router, http.MethodPost, "/admin/customers/some-id/credits", body, token)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

}

// ---------------------------------------------------------------------------
// Tests: GetPlatformOverview
// ---------------------------------------------------------------------------

func TestAdminHandler_GetPlatformOverview(t *testing.T) {
	t.Run("success returns platform stats", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		stats := &domain.PlatformStats{
			TotalCustomers:      100,
			ActiveCustomers:     95,
			SuspendedCustomers:  5,
			TotalPixels:         200,
			EventsToday:         5000,
			EventsThisMonth:     150000,
			TotalReplays:        50,
			SuccessfulReplays:   45,
			FailedReplays:       5,
			TotalRevenueTHB:     50000.0,
			RevenueThisMonthTHB: 10000.0,
			CustomersByPlan:     map[string]int{"sandbox": 80, "paid": 20},
		}
		env.adminRepo.On("GetPlatformStats", mock.Anything).Return(stats, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/overview", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.NotNil(t, resp.Data)
	})

	t.Run("internal error returns 500", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("GetPlatformStats", mock.Anything).
			Return((*domain.PlatformStats)(nil), errDB)

		rec := doRequest(env.router, http.MethodGet, "/admin/overview", nil, token)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("non-admin returns 403", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, false)

		rec := doRequest(env.router, http.MethodGet, "/admin/overview", nil, token)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: GetRevenueChart
// ---------------------------------------------------------------------------

func TestAdminHandler_GetRevenueChart(t *testing.T) {
	t.Run("success returns revenue chart with default days", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		chart := []*domain.RevenueChartPoint{
			{Date: "2026-03-01", AmountSatang: 100000, PurchaseCount: 2},
			{Date: "2026-03-02", AmountSatang: 200000, PurchaseCount: 3},
		}
		env.adminRepo.On("GetRevenueChart", mock.Anything, 30).Return(chart, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/revenue-chart", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.NotNil(t, resp.Data)
	})

	t.Run("success with custom days parameter", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("GetRevenueChart", mock.Anything, 7).
			Return([]*domain.RevenueChartPoint{}, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/revenue-chart?days=7", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("internal error returns 500", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("GetRevenueChart", mock.Anything, 30).
			Return(([]*domain.RevenueChartPoint)(nil), errDB)

		rec := doRequest(env.router, http.MethodGet, "/admin/revenue-chart", nil, token)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: GetGrowthChart
// ---------------------------------------------------------------------------

func TestAdminHandler_GetGrowthChart(t *testing.T) {
	t.Run("success returns growth chart", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		chart := []*domain.GrowthChartPoint{
			{Date: "2026-03-01", NewCustomers: 5, TotalCustomers: 95},
		}
		env.adminRepo.On("GetGrowthChart", mock.Anything, 30).Return(chart, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/growth-chart", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("internal error returns 500", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("GetGrowthChart", mock.Anything, 30).
			Return(([]*domain.GrowthChartPoint)(nil), errDB)

		rec := doRequest(env.router, http.MethodGet, "/admin/growth-chart", nil, token)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: ListPurchases
// ---------------------------------------------------------------------------

func TestAdminHandler_ListPurchases(t *testing.T) {
	t.Run("success returns paginated purchases", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		purchases := []*domain.AdminPurchase{
			{CustomerEmail: "a@test.com", CustomerName: "Alice"},
		}
		env.adminRepo.On("ListAllPurchases", mock.Anything, "", 20, 0).
			Return(purchases, 1, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/purchases", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp PaginatedResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, 1, resp.Total)
	})

	t.Run("success with status filter", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("ListAllPurchases", mock.Anything, "completed", 20, 0).
			Return([]*domain.AdminPurchase{}, 0, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/purchases?status=completed", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("internal error returns 500", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("ListAllPurchases", mock.Anything, "", 20, 0).
			Return(nil, 0, errDB)

		rec := doRequest(env.router, http.MethodGet, "/admin/purchases", nil, token)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: ListSubscriptions
// ---------------------------------------------------------------------------

func TestAdminHandler_ListSubscriptions(t *testing.T) {
	t.Run("success returns paginated subscriptions", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		subs := []*domain.AdminSubscription{
			{CustomerEmail: "a@test.com", CustomerName: "Alice"},
		}
		env.adminRepo.On("ListAllSubscriptions", mock.Anything, "", 20, 0).
			Return(subs, 1, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/subscriptions", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp PaginatedResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, 1, resp.Total)
	})

	t.Run("internal error returns 500", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("ListAllSubscriptions", mock.Anything, "", 20, 0).
			Return(nil, 0, errDB)

		rec := doRequest(env.router, http.MethodGet, "/admin/subscriptions", nil, token)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: ListCreditGrants
// ---------------------------------------------------------------------------

func TestAdminHandler_ListCreditGrants(t *testing.T) {
	t.Run("success returns paginated credit grants", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		grants := []*domain.AdminCreditGrantWithCustomer{
			{CustomerEmail: "a@test.com", CustomerName: "Alice"},
		}
		env.adminRepo.On("ListCreditGrants", mock.Anything, 20, 0).
			Return(grants, 1, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/credit-grants", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp PaginatedResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, 1, resp.Total)
	})

	t.Run("internal error returns 500", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("ListCreditGrants", mock.Anything, 20, 0).
			Return(nil, 0, errDB)

		rec := doRequest(env.router, http.MethodGet, "/admin/credit-grants", nil, token)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: ListSalePages
// ---------------------------------------------------------------------------

func TestAdminHandler_ListSalePages(t *testing.T) {
	t.Run("success returns paginated sale pages", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		pages := []*domain.AdminSalePage{
			{CustomerEmail: "a@test.com", CustomerName: "Alice"},
		}
		env.adminRepo.On("ListAllSalePages", mock.Anything, "", "", (*bool)(nil), 20, 0).
			Return(pages, 1, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/sale-pages", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp PaginatedResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, 1, resp.Total)
	})

	t.Run("success with search and customer_id filters", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("ListAllSalePages", mock.Anything, "test", "cust-123", (*bool)(nil), 20, 0).
			Return([]*domain.AdminSalePage{}, 0, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/sale-pages?search=test&customer_id=cust-123", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("success with published filter", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		published := true
		env.adminRepo.On("ListAllSalePages", mock.Anything, "", "", &published, 20, 0).
			Return([]*domain.AdminSalePage{}, 0, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/sale-pages?published=true", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("internal error returns 500", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("ListAllSalePages", mock.Anything, "", "", (*bool)(nil), 20, 0).
			Return(nil, 0, errDB)

		rec := doRequest(env.router, http.MethodGet, "/admin/sale-pages", nil, token)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: GetSalePageDetail
// ---------------------------------------------------------------------------

func TestAdminHandler_GetSalePageDetail(t *testing.T) {
	t.Run("success returns sale page detail", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		detail := &domain.AdminSalePageDetail{
			SalePage:      &domain.SalePage{ID: "sp-1", Name: "Test Page"},
			CustomerEmail: "owner@test.com",
			CustomerName:  "Owner",
		}
		env.adminRepo.On("GetSalePageAdminDetail", mock.Anything, "sp-1").
			Return(detail, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/sale-pages/sp-1", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.NotNil(t, resp.Data)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("GetSalePageAdminDetail", mock.Anything, "nonexistent").
			Return((*domain.AdminSalePageDetail)(nil), nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/sale-pages/nonexistent", nil, token)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Contains(t, resp.Error, "sale page not found")
	})

	t.Run("internal error returns 500", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("GetSalePageAdminDetail", mock.Anything, "err-sp").
			Return((*domain.AdminSalePageDetail)(nil), errDB)

		rec := doRequest(env.router, http.MethodGet, "/admin/sale-pages/err-sp", nil, token)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: DisableSalePage
// ---------------------------------------------------------------------------

func TestAdminHandler_DisableSalePage(t *testing.T) {
	t.Run("success disables sale page", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("SetSalePagePublished", mock.Anything, "sp-1", false).Return(nil)
		env.adminRepo.On("CreateAuditLog", mock.Anything, mock.Anything).Return(nil)

		rec := doRequest(env.router, http.MethodPost, "/admin/sale-pages/sp-1/disable", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, "sale page disabled", resp.Message)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("SetSalePagePublished", mock.Anything, "nonexistent", false).
			Return(repository.ErrNotFound)

		rec := doRequest(env.router, http.MethodPost, "/admin/sale-pages/nonexistent/disable", nil, token)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Contains(t, resp.Error, "sale page not found")
	})

	t.Run("internal error returns 500", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("SetSalePagePublished", mock.Anything, "sp-err", false).Return(errDB)

		rec := doRequest(env.router, http.MethodPost, "/admin/sale-pages/sp-err/disable", nil, token)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: EnableSalePage
// ---------------------------------------------------------------------------

func TestAdminHandler_EnableSalePage(t *testing.T) {
	t.Run("success enables sale page", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("SetSalePagePublished", mock.Anything, "sp-1", true).Return(nil)
		env.adminRepo.On("CreateAuditLog", mock.Anything, mock.Anything).Return(nil)

		rec := doRequest(env.router, http.MethodPost, "/admin/sale-pages/sp-1/enable", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, "sale page enabled", resp.Message)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("SetSalePagePublished", mock.Anything, "nonexistent", true).
			Return(repository.ErrNotFound)

		rec := doRequest(env.router, http.MethodPost, "/admin/sale-pages/nonexistent/enable", nil, token)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: DeleteSalePage
// ---------------------------------------------------------------------------

func TestAdminHandler_DeleteSalePage(t *testing.T) {
	t.Run("success deletes sale page", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("GetSalePageAdminDetail", mock.Anything, "sp-1").
			Return(&domain.AdminSalePageDetail{
				SalePage:      &domain.SalePage{ID: "sp-1", CustomerID: "owner-1"},
				CustomerEmail: "owner@test.com",
			}, nil)
		env.adminRepo.On("DeleteSalePageByAdmin", mock.Anything, "sp-1").Return(nil)
		env.adminRepo.On("CreateAuditLog", mock.Anything, mock.Anything).Return(nil)

		rec := doRequest(env.router, http.MethodDelete, "/admin/sale-pages/sp-1", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, "sale page deleted", resp.Message)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("GetSalePageAdminDetail", mock.Anything, "nonexistent").
			Return((*domain.AdminSalePageDetail)(nil), nil)

		rec := doRequest(env.router, http.MethodDelete, "/admin/sale-pages/nonexistent", nil, token)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: ListPixels
// ---------------------------------------------------------------------------

func TestAdminHandler_ListPixels(t *testing.T) {
	t.Run("success returns paginated pixels", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		pixels := []*domain.AdminPixel{
			{CustomerEmail: "a@test.com", CustomerName: "Alice"},
		}
		env.adminRepo.On("ListAllPixels", mock.Anything, "", "", (*bool)(nil), 20, 0).
			Return(pixels, 1, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/pixels", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp PaginatedResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, 1, resp.Total)
	})

	t.Run("success with search and active filter", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		active := true
		env.adminRepo.On("ListAllPixels", mock.Anything, "test", "", &active, 20, 0).
			Return([]*domain.AdminPixel{}, 0, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/pixels?search=test&active=true", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("internal error returns 500", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("ListAllPixels", mock.Anything, "", "", (*bool)(nil), 20, 0).
			Return(nil, 0, errDB)

		rec := doRequest(env.router, http.MethodGet, "/admin/pixels", nil, token)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: GetPixelDetail
// ---------------------------------------------------------------------------

func TestAdminHandler_GetPixelDetail(t *testing.T) {
	t.Run("success returns pixel detail", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		detail := &domain.AdminPixelDetail{
			Pixel:         &domain.Pixel{ID: "px-1", Name: "Test Pixel"},
			CustomerEmail: "owner@test.com",
			EventCount:    500,
		}
		env.adminRepo.On("GetPixelAdminDetail", mock.Anything, "px-1").
			Return(detail, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/pixels/px-1", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.NotNil(t, resp.Data)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("GetPixelAdminDetail", mock.Anything, "nonexistent").
			Return((*domain.AdminPixelDetail)(nil), nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/pixels/nonexistent", nil, token)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Contains(t, resp.Error, "pixel not found")
	})

	t.Run("internal error returns 500", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("GetPixelAdminDetail", mock.Anything, "err-px").
			Return((*domain.AdminPixelDetail)(nil), errDB)

		rec := doRequest(env.router, http.MethodGet, "/admin/pixels/err-px", nil, token)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: DisablePixel
// ---------------------------------------------------------------------------

func TestAdminHandler_DisablePixel(t *testing.T) {
	t.Run("success disables pixel", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("SetPixelActive", mock.Anything, "px-1", false).Return(nil)
		env.adminRepo.On("CreateAuditLog", mock.Anything, mock.Anything).Return(nil)

		rec := doRequest(env.router, http.MethodPost, "/admin/pixels/px-1/disable", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, "pixel disabled", resp.Message)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("SetPixelActive", mock.Anything, "nonexistent", false).
			Return(repository.ErrNotFound)

		rec := doRequest(env.router, http.MethodPost, "/admin/pixels/nonexistent/disable", nil, token)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Contains(t, resp.Error, "pixel not found")
	})

	t.Run("internal error returns 500", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("SetPixelActive", mock.Anything, "px-err", false).Return(errDB)

		rec := doRequest(env.router, http.MethodPost, "/admin/pixels/px-err/disable", nil, token)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: EnablePixel
// ---------------------------------------------------------------------------

func TestAdminHandler_EnablePixel(t *testing.T) {
	t.Run("success enables pixel", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("SetPixelActive", mock.Anything, "px-1", true).Return(nil)
		env.adminRepo.On("CreateAuditLog", mock.Anything, mock.Anything).Return(nil)

		rec := doRequest(env.router, http.MethodPost, "/admin/pixels/px-1/enable", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, "pixel enabled", resp.Message)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("SetPixelActive", mock.Anything, "nonexistent", true).
			Return(repository.ErrNotFound)

		rec := doRequest(env.router, http.MethodPost, "/admin/pixels/nonexistent/enable", nil, token)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: ListReplays
// ---------------------------------------------------------------------------

func TestAdminHandler_ListReplays(t *testing.T) {
	t.Run("success returns paginated replays", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		sessions := []*domain.AdminReplaySession{
			{CustomerEmail: "a@test.com", SourcePixelName: "Source", TargetPixelName: "Target"},
		}
		env.adminRepo.On("ListAllReplaySessions", mock.Anything, "", "", 20, 0).
			Return(sessions, 1, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/replays", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp PaginatedResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, 1, resp.Total)
	})

	t.Run("success with status and customer_id filters", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("ListAllReplaySessions", mock.Anything, "completed", "cust-123", 20, 0).
			Return([]*domain.AdminReplaySession{}, 0, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/replays?status=completed&customer_id=cust-123", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("internal error returns 500", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("ListAllReplaySessions", mock.Anything, "", "", 20, 0).
			Return(nil, 0, errDB)

		rec := doRequest(env.router, http.MethodGet, "/admin/replays", nil, token)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: GetReplayDetail
// ---------------------------------------------------------------------------

func TestAdminHandler_GetReplayDetail(t *testing.T) {
	t.Run("success returns replay detail", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		detail := &domain.AdminReplaySessionDetail{
			Session:       &domain.ReplaySession{ID: "replay-1", CustomerID: "cust-1"},
			CustomerEmail: "owner@test.com",
		}
		env.adminRepo.On("GetReplaySessionAdminDetail", mock.Anything, "replay-1").
			Return(detail, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/replays/replay-1", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.NotNil(t, resp.Data)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("GetReplaySessionAdminDetail", mock.Anything, "nonexistent").
			Return((*domain.AdminReplaySessionDetail)(nil), nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/replays/nonexistent", nil, token)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Contains(t, resp.Error, "replay session not found")
	})

	t.Run("internal error returns 500", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("GetReplaySessionAdminDetail", mock.Anything, "err-replay").
			Return((*domain.AdminReplaySessionDetail)(nil), errDB)

		rec := doRequest(env.router, http.MethodGet, "/admin/replays/err-replay", nil, token)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: CancelReplay
// ---------------------------------------------------------------------------

func TestAdminHandler_CancelReplay(t *testing.T) {
	t.Run("success cancels replay", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.replayRepo.On("CancelSession", mock.Anything, "replay-1").
			Return(&domain.ReplaySession{ID: "replay-1", CustomerID: "cust-owner"}, nil)
		env.adminRepo.On("CreateAuditLog", mock.Anything, mock.Anything).Return(nil)

		rec := doRequest(env.router, http.MethodPost, "/admin/replays/replay-1/cancel", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, "replay cancelled", resp.Message)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.replayRepo.On("CancelSession", mock.Anything, "nonexistent").
			Return((*domain.ReplaySession)(nil), repository.ErrNotFound)

		rec := doRequest(env.router, http.MethodPost, "/admin/replays/nonexistent/cancel", nil, token)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Contains(t, resp.Error, "replay session not found")
	})

	t.Run("internal error returns 500", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.replayRepo.On("CancelSession", mock.Anything, "err-replay").
			Return((*domain.ReplaySession)(nil), errDB)

		rec := doRequest(env.router, http.MethodPost, "/admin/replays/err-replay/cancel", nil, token)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: ListEvents
// ---------------------------------------------------------------------------

func TestAdminHandler_ListEvents(t *testing.T) {
	t.Run("success returns paginated events", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		events := []*domain.AdminEvent{
			{PixelName: "My Pixel", CustomerEmail: "a@test.com"},
		}
		env.adminRepo.On("ListAllEvents", mock.Anything, "", "", "", 20, 0).
			Return(events, 1, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/events", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp PaginatedResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, 1, resp.Total)
	})

	t.Run("success with all filters", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("ListAllEvents", mock.Anything, "cust-1", "px-1", "Purchase", 20, 0).
			Return([]*domain.AdminEvent{}, 0, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/events?customer_id=cust-1&pixel_id=px-1&event_name=Purchase", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("internal error returns 500", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("ListAllEvents", mock.Anything, "", "", "", 20, 0).
			Return(nil, 0, errDB)

		rec := doRequest(env.router, http.MethodGet, "/admin/events", nil, token)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: GetEventStats
// ---------------------------------------------------------------------------

func TestAdminHandler_GetEventStats(t *testing.T) {
	t.Run("success returns event stats with default hours", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		stats := &domain.AdminEventStats{
			TotalToday:      1000,
			TotalThisHour:   50,
			CAPISuccessRate: 98.5,
		}
		env.adminRepo.On("GetEventStats", mock.Anything, 24).Return(stats, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/events/stats", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp APIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.NotNil(t, resp.Data)
	})

	t.Run("success with custom hours", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("GetEventStats", mock.Anything, 12).
			Return(&domain.AdminEventStats{}, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/events/stats?hours=12", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("internal error returns 500", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("GetEventStats", mock.Anything, 24).
			Return((*domain.AdminEventStats)(nil), errDB)

		rec := doRequest(env.router, http.MethodGet, "/admin/events/stats", nil, token)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Tests: ListAuditLog
// ---------------------------------------------------------------------------

func TestAdminHandler_ListAuditLog(t *testing.T) {
	t.Run("success returns paginated audit log", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		entries := []*domain.AuditLogEntry{
			{ID: "audit-1", AdminID: testCustomerID, Action: "suspend_customer", TargetType: "customer", TargetID: "c-1", CreatedAt: time.Now()},
		}
		env.adminRepo.On("ListAuditLogs", mock.Anything, "", "", "", (*time.Time)(nil), (*time.Time)(nil), 20, 0).
			Return(entries, 1, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/audit-log", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp PaginatedResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, 1, resp.Total)
	})

	t.Run("success with filters", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("ListAuditLogs", mock.Anything, "admin-1", "suspend_customer", "target-1", mock.AnythingOfType("*time.Time"), mock.AnythingOfType("*time.Time"), 20, 0).
			Return([]*domain.AuditLogEntry{}, 0, nil)

		rec := doRequest(env.router, http.MethodGet, "/admin/audit-log?admin_id=admin-1&action=suspend_customer&target_customer_id=target-1&from=2026-03-01T00:00:00Z&to=2026-03-18T23:59:59Z", nil, token)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("internal error returns 500", func(t *testing.T) {
		env := setupAdminTest(t)
		token := testJWT(testCustomerID, true)

		env.adminRepo.On("ListAuditLogs", mock.Anything, "", "", "", (*time.Time)(nil), (*time.Time)(nil), 20, 0).
			Return(nil, 0, errDB)

		rec := doRequest(env.router, http.MethodGet, "/admin/audit-log", nil, token)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}
