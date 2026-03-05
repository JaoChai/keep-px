package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

var errDBDown = errors.New("db down")

func newTestAdminService() (*AdminService, *MockAdminRepo, *MockCustomerRepo, *MockReplayCreditRepo, *MockReplaySessionRepo) {
	adminRepo := new(MockAdminRepo)
	customerRepo := new(MockCustomerRepo)
	creditRepo := new(MockReplayCreditRepo)
	replaySessionRepo := new(MockReplaySessionRepo)
	svc := NewAdminService(adminRepo, customerRepo, creditRepo, replaySessionRepo, nil)
	return svc, adminRepo, customerRepo, creditRepo, replaySessionRepo
}

func TestAdminService_ListCustomers(t *testing.T) {
	tests := []struct {
		name      string
		search    string
		plan      string
		status    string
		page      int
		perPage   int
		setup     func(*MockAdminRepo)
		wantLen   int
		wantTotal int
		wantErr   error
	}{
		{
			name:    "success with defaults",
			search:  "",
			plan:    "",
			status:  "",
			page:    0,
			perPage: 0,
			setup: func(ar *MockAdminRepo) {
				ar.On("ListCustomers", mock.Anything, "", "", "", 20, 0).Return([]*domain.Customer{
					{ID: "c1", Email: "a@b.com"},
				}, 1, nil)
			},
			wantLen:   1,
			wantTotal: 1,
		},
		{
			name:    "invalid plan returns ErrInvalidPlan",
			plan:    "nonexistent",
			page:    1,
			perPage: 20,
			setup:   func(ar *MockAdminRepo) {},
			wantErr: ErrInvalidPlan,
		},
		{
			name:    "invalid status gets cleared",
			status:  "bogus",
			page:    1,
			perPage: 10,
			setup: func(ar *MockAdminRepo) {
				ar.On("ListCustomers", mock.Anything, "", "", "", 10, 0).Return([]*domain.Customer{}, 0, nil)
			},
			wantLen:   0,
			wantTotal: 0,
		},
		{
			name:    "repo error propagates",
			page:    1,
			perPage: 20,
			setup: func(ar *MockAdminRepo) {
				ar.On("ListCustomers", mock.Anything, "", "", "", 20, 0).Return(nil, 0, errDBDown)
			},
			wantErr: errDBDown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, adminRepo, _, _, _ := newTestAdminService()
			tt.setup(adminRepo)

			customers, total, err := svc.ListCustomers(context.Background(), tt.search, tt.plan, tt.status, tt.page, tt.perPage)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.Len(t, customers, tt.wantLen)
			assert.Equal(t, tt.wantTotal, total)
			adminRepo.AssertExpectations(t)
		})
	}
}

func TestAdminService_GetCustomerDetail(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		setup   func(*MockAdminRepo)
		wantErr error
	}{
		{
			name: "success",
			id:   "c1",
			setup: func(ar *MockAdminRepo) {
				ar.On("GetCustomerDetail", mock.Anything, "c1").Return(&domain.AdminCustomerDetail{
					Customer: &domain.Customer{ID: "c1"},
				}, nil)
			},
		},
		{
			name: "not found",
			id:   "missing",
			setup: func(ar *MockAdminRepo) {
				ar.On("GetCustomerDetail", mock.Anything, "missing").Return(nil, nil)
			},
			wantErr: ErrCustomerNotFound,
		},
		{
			name: "repo error",
			id:   "c1",
			setup: func(ar *MockAdminRepo) {
				ar.On("GetCustomerDetail", mock.Anything, "c1").Return(nil, errDBDown)
			},
			wantErr: errDBDown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, adminRepo, _, _, _ := newTestAdminService()
			tt.setup(adminRepo)

			detail, err := svc.GetCustomerDetail(context.Background(), tt.id)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, detail)
			adminRepo.AssertExpectations(t)
		})
	}
}

func TestAdminService_ChangePlan(t *testing.T) {
	tests := []struct {
		name      string
		custID    string
		newPlan   string
		setupCust func(*MockCustomerRepo)
		setupAdm  func(*MockAdminRepo)
		wantErr   error
	}{
		{
			name:    "success",
			custID:  "c1",
			newPlan: domain.PlanLaunch,
			setupCust: func(cr *MockCustomerRepo) {
				cr.On("GetByID", mock.Anything, "c1").Return(&domain.Customer{ID: "c1"}, nil)
				cr.On("UpdatePlan", mock.Anything, "c1", domain.PlanLaunch).Return(nil)
			},
			setupAdm: func(ar *MockAdminRepo) {
				ar.On("CreateAuditLog", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name:      "invalid plan",
			custID:    "c1",
			newPlan:   "gold",
			setupCust: func(cr *MockCustomerRepo) {},
			setupAdm:  func(ar *MockAdminRepo) {},
			wantErr:   ErrInvalidPlan,
		},
		{
			name:    "customer not found",
			custID:  "missing",
			newPlan: domain.PlanShield,
			setupCust: func(cr *MockCustomerRepo) {
				cr.On("GetByID", mock.Anything, "missing").Return(nil, nil)
			},
			setupAdm: func(ar *MockAdminRepo) {},
			wantErr:  ErrCustomerNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, adminRepo, customerRepo, _, _ := newTestAdminService()
			tt.setupCust(customerRepo)
			tt.setupAdm(adminRepo)

			err := svc.ChangePlan(context.Background(), "admin-1", tt.custID, tt.newPlan)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			customerRepo.AssertExpectations(t)
		})
	}
}

func TestAdminService_SuspendCustomer(t *testing.T) {
	tests := []struct {
		name    string
		adminID string
		custID  string
		setup   func(*MockAdminRepo)
		wantErr error
	}{
		{
			name:    "success",
			adminID: "admin-1",
			custID:  "c1",
			setup: func(ar *MockAdminRepo) {
				ar.On("SuspendCustomer", mock.Anything, "c1").Return(nil)
				ar.On("CreateAuditLog", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name:    "self-suspend blocked",
			adminID: "c1",
			custID:  "c1",
			setup:   func(ar *MockAdminRepo) {},
			wantErr: ErrAdminSelfSuspend,
		},
		{
			name:    "not found",
			adminID: "admin-1",
			custID:  "missing",
			setup: func(ar *MockAdminRepo) {
				ar.On("SuspendCustomer", mock.Anything, "missing").Return(repository.ErrNotFound)
			},
			wantErr: ErrCustomerNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, adminRepo, _, _, _ := newTestAdminService()
			tt.setup(adminRepo)

			err := svc.SuspendCustomer(context.Background(), tt.adminID, tt.custID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			adminRepo.AssertExpectations(t)
		})
	}
}

func TestAdminService_ActivateCustomer(t *testing.T) {
	tests := []struct {
		name    string
		custID  string
		setup   func(*MockAdminRepo)
		wantErr error
	}{
		{
			name:   "success",
			custID: "c1",
			setup: func(ar *MockAdminRepo) {
				ar.On("ActivateCustomer", mock.Anything, "c1").Return(nil)
				ar.On("CreateAuditLog", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name:   "not found",
			custID: "missing",
			setup: func(ar *MockAdminRepo) {
				ar.On("ActivateCustomer", mock.Anything, "missing").Return(repository.ErrNotFound)
			},
			wantErr: ErrCustomerNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, adminRepo, _, _, _ := newTestAdminService()
			tt.setup(adminRepo)

			err := svc.ActivateCustomer(context.Background(), "admin-1", tt.custID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			adminRepo.AssertExpectations(t)
		})
	}
}

func TestAdminService_GetRevenueChart(t *testing.T) {
	tests := []struct {
		name     string
		days     int
		wantDays int
	}{
		{"valid days", 30, 30},
		{"zero defaults to 30", 0, 30},
		{"over 365 defaults to 30", 400, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, adminRepo, _, _, _ := newTestAdminService()
			adminRepo.On("GetRevenueChart", mock.Anything, tt.wantDays).Return([]*domain.RevenueChartPoint{
				{Date: "2026-03-01", AmountSatang: 100},
			}, nil)

			points, err := svc.GetRevenueChart(context.Background(), tt.days)

			assert.NoError(t, err)
			assert.Len(t, points, 1)
			adminRepo.AssertExpectations(t)
		})
	}
}

func TestAdminService_ListAllSalePages(t *testing.T) {
	svc, adminRepo, _, _, _ := newTestAdminService()
	adminRepo.On("ListAllSalePages", mock.Anything, "", "", (*bool)(nil), 20, 0).Return([]*domain.AdminSalePage{
		{CustomerEmail: "a@b.com"},
	}, 1, nil)

	pages, total, err := svc.ListAllSalePages(context.Background(), "", "", nil, 1, 20)
	assert.NoError(t, err)
	assert.Len(t, pages, 1)
	assert.Equal(t, 1, total)
	adminRepo.AssertExpectations(t)
}

func TestAdminService_DisableSalePage(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*MockAdminRepo)
		wantErr error
	}{
		{
			name: "success",
			setup: func(ar *MockAdminRepo) {
				ar.On("SetSalePagePublished", mock.Anything, "sp1", false).Return(nil)
				ar.On("CreateAuditLog", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name: "not found",
			setup: func(ar *MockAdminRepo) {
				ar.On("SetSalePagePublished", mock.Anything, "sp1", false).Return(repository.ErrNotFound)
			},
			wantErr: ErrSalePageNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, adminRepo, _, _, _ := newTestAdminService()
			tt.setup(adminRepo)

			err := svc.DisableSalePage(context.Background(), "admin-1", "sp1")

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			adminRepo.AssertExpectations(t)
		})
	}
}

func TestAdminService_ListAllPixels(t *testing.T) {
	svc, adminRepo, _, _, _ := newTestAdminService()
	adminRepo.On("ListAllPixels", mock.Anything, "", "", (*bool)(nil), 20, 0).Return([]*domain.AdminPixel{
		{CustomerEmail: "a@b.com"},
	}, 1, nil)

	pixels, total, err := svc.ListAllPixels(context.Background(), "", "", nil, 1, 20)
	assert.NoError(t, err)
	assert.Len(t, pixels, 1)
	assert.Equal(t, 1, total)
	adminRepo.AssertExpectations(t)
}

func TestAdminService_CancelReplay(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*MockReplaySessionRepo, *MockAdminRepo)
		wantErr error
	}{
		{
			name: "success",
			setup: func(rsr *MockReplaySessionRepo, ar *MockAdminRepo) {
				rsr.On("CancelSession", mock.Anything, "rs1").Return(&domain.ReplaySession{ID: "rs1", CustomerID: "c1"}, nil)
				ar.On("CreateAuditLog", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name: "not found",
			setup: func(rsr *MockReplaySessionRepo, ar *MockAdminRepo) {
				rsr.On("CancelSession", mock.Anything, "rs1").Return(nil, repository.ErrNotFound)
			},
			wantErr: ErrReplayNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, adminRepo, _, _, replayRepo := newTestAdminService()
			tt.setup(replayRepo, adminRepo)

			err := svc.CancelReplay(context.Background(), "admin-1", "rs1")

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestAdminService_GetEventStats(t *testing.T) {
	tests := []struct {
		name      string
		hours     int
		wantHours int
	}{
		{"valid hours", 12, 12},
		{"zero defaults to 24", 0, 24},
		{"over 72 defaults to 24", 100, 24},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, adminRepo, _, _, _ := newTestAdminService()
			adminRepo.On("GetEventStats", mock.Anything, tt.wantHours).Return(&domain.AdminEventStats{
				TotalToday: 100,
			}, nil)

			stats, err := svc.GetEventStats(context.Background(), tt.hours)

			assert.NoError(t, err)
			assert.Equal(t, int64(100), stats.TotalToday)
			adminRepo.AssertExpectations(t)
		})
	}
}

func TestAdminService_GetGrowthChart(t *testing.T) {
	tests := []struct {
		name     string
		days     int
		wantDays int
	}{
		{"valid days", 30, 30},
		{"zero defaults to 30", 0, 30},
		{"over 365 defaults to 30", 400, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, adminRepo, _, _, _ := newTestAdminService()
			adminRepo.On("GetGrowthChart", mock.Anything, tt.wantDays).Return([]*domain.GrowthChartPoint{
				{Date: "2026-03-01", NewCustomers: 5, TotalCustomers: 100},
			}, nil)

			points, err := svc.GetGrowthChart(context.Background(), tt.days)

			assert.NoError(t, err)
			assert.Len(t, points, 1)
			adminRepo.AssertExpectations(t)
		})
	}
}
