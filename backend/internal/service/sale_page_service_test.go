package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

func newTestSalePageService(t *testing.T) (*SalePageService, *MockSalePageRepo, *MockCustomerRepo, *MockPixelRepo) {
	t.Helper()
	salePageRepo := new(MockSalePageRepo)
	customerRepo := new(MockCustomerRepo)
	pixelRepo := new(MockPixelRepo)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	svc := NewSalePageService(ctx, salePageRepo, customerRepo, pixelRepo, nil, 60*time.Second)
	return svc, salePageRepo, customerRepo, pixelRepo
}

type quotaMocks struct {
	salePageRepo *MockSalePageRepo
	customerRepo *MockCustomerRepo
	pixelRepo    *MockPixelRepo
	subRepo      *MockSubscriptionRepo
	creditRepo   *MockReplayCreditRepo
	usageRepo    *MockEventUsageRepo
}

func newTestSalePageServiceWithQuota(t *testing.T) (*SalePageService, *quotaMocks) {
	t.Helper()
	m := &quotaMocks{
		salePageRepo: new(MockSalePageRepo),
		customerRepo: new(MockCustomerRepo),
		pixelRepo:    new(MockPixelRepo),
		subRepo:      new(MockSubscriptionRepo),
		creditRepo:   new(MockReplayCreditRepo),
		usageRepo:    new(MockEventUsageRepo),
	}

	quotaService := NewQuotaService(m.creditRepo, m.subRepo, m.usageRepo, m.pixelRepo, m.salePageRepo, m.customerRepo)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	svc := NewSalePageService(ctx, m.salePageRepo, m.customerRepo, m.pixelRepo, quotaService, 60*time.Second)
	return svc, m
}

var validV1Content = json.RawMessage(`{"hero":{"title":"Test","subtitle":"Sub","image_url":""},"body":{"description":"Test page","features":[],"images":[]},"cta":{"button_text":"Buy","button_link":""},"contact":{},"tracking":{},"style":{}}`)
var validV2Content = json.RawMessage(`{"version":2,"blocks":[{"id":"b1","type":"text","text":"hello"}],"style":{},"tracking":{}}`)

// ---------- Create ----------

func TestSalePageService_Create(t *testing.T) {
	tests := []struct {
		name    string
		input   CreateSalePageInput
		setup   func(*MockSalePageRepo, *MockPixelRepo)
		wantErr error
		errMsg  string
		check   func(*testing.T, *domain.SalePage)
	}{
		{
			name: "success_with_auto_slug",
			input: CreateSalePageInput{
				Name:         "My Page",
				TemplateName: "default",
				Content:      validV1Content,
			},
			setup: func(sp *MockSalePageRepo, pr *MockPixelRepo) {
				sp.On("SlugExists", mock.Anything, mock.AnythingOfType("string")).Return(false, nil).Once()
				sp.On("Create", mock.Anything, mock.AnythingOfType("*domain.SalePage")).Return(nil)
			},
			check: func(t *testing.T, page *domain.SalePage) {
				assert.NotEmpty(t, page.Slug)
				assert.True(t, len(page.Slug) > 2, "auto-generated slug should have prefix + chars")
				assert.Equal(t, "My Page", page.Name)
				assert.Equal(t, "cust-1", page.CustomerID)
			},
		},
		{
			name: "success_with_custom_slug",
			input: CreateSalePageInput{
				Name:         "My Page",
				Slug:         "my-page",
				TemplateName: "default",
				Content:      validV1Content,
			},
			setup: func(sp *MockSalePageRepo, pr *MockPixelRepo) {
				sp.On("SlugExists", mock.Anything, "my-page").Return(false, nil)
				sp.On("Create", mock.Anything, mock.AnythingOfType("*domain.SalePage")).Return(nil)
			},
			check: func(t *testing.T, page *domain.SalePage) {
				assert.Equal(t, "my-page", page.Slug)
			},
		},
		{
			name: "success_with_pixels",
			input: CreateSalePageInput{
				Name:         "Page With Pixels",
				Slug:         "pixel-page",
				TemplateName: "default",
				Content:      validV1Content,
				PixelIDs:     []string{"pixel-1", "pixel-2"},
			},
			setup: func(sp *MockSalePageRepo, pr *MockPixelRepo) {
				sp.On("SlugExists", mock.Anything, "pixel-page").Return(false, nil)
				pr.On("GetByIDs", mock.Anything, []string{"pixel-1", "pixel-2"}).Return([]*domain.Pixel{
					{ID: "pixel-1", CustomerID: "cust-1"},
					{ID: "pixel-2", CustomerID: "cust-1"},
				}, nil)
				sp.On("Create", mock.Anything, mock.AnythingOfType("*domain.SalePage")).Return(nil)
			},
			check: func(t *testing.T, page *domain.SalePage) {
				assert.Equal(t, []string{"pixel-1", "pixel-2"}, page.PixelIDs)
			},
		},
		{
			name: "success_with_v2_content",
			input: CreateSalePageInput{
				Name:         "V2 Page",
				Slug:         "v2-page",
				TemplateName: "blocks",
				Content:      validV2Content,
			},
			setup: func(sp *MockSalePageRepo, pr *MockPixelRepo) {
				sp.On("SlugExists", mock.Anything, "v2-page").Return(false, nil)
				sp.On("Create", mock.Anything, mock.AnythingOfType("*domain.SalePage")).Return(nil)
			},
			check: func(t *testing.T, page *domain.SalePage) {
				assert.Equal(t, "v2-page", page.Slug)
			},
		},
		{
			name: "slug_taken",
			input: CreateSalePageInput{
				Name:         "My Page",
				Slug:         "taken-slug",
				TemplateName: "default",
				Content:      validV1Content,
			},
			setup: func(sp *MockSalePageRepo, pr *MockPixelRepo) {
				sp.On("SlugExists", mock.Anything, "taken-slug").Return(true, nil)
			},
			wantErr: ErrSlugTaken,
		},
		{
			name: "reserved_slug_admin",
			input: CreateSalePageInput{
				Name:         "My Page",
				Slug:         "admin",
				TemplateName: "default",
				Content:      validV1Content,
			},
			setup:   func(sp *MockSalePageRepo, pr *MockPixelRepo) {},
			wantErr: ErrSlugTaken,
		},
		{
			name: "reserved_slug_api",
			input: CreateSalePageInput{
				Name:         "My Page",
				Slug:         "api",
				TemplateName: "default",
				Content:      validV1Content,
			},
			setup:   func(sp *MockSalePageRepo, pr *MockPixelRepo) {},
			wantErr: ErrSlugTaken,
		},
		{
			name: "reserved_slug_login",
			input: CreateSalePageInput{
				Name:         "My Page",
				Slug:         "login",
				TemplateName: "default",
				Content:      validV1Content,
			},
			setup:   func(sp *MockSalePageRepo, pr *MockPixelRepo) {},
			wantErr: ErrSlugTaken,
		},
		{
			name: "invalid_slug_uppercase",
			input: CreateSalePageInput{
				Name:         "My Page",
				Slug:         "MyPage",
				TemplateName: "default",
				Content:      validV1Content,
			},
			setup:   func(sp *MockSalePageRepo, pr *MockPixelRepo) {},
			wantErr: ErrInvalidSlug,
		},
		{
			name: "invalid_slug_spaces",
			input: CreateSalePageInput{
				Name:         "My Page",
				Slug:         "my page",
				TemplateName: "default",
				Content:      validV1Content,
			},
			setup:   func(sp *MockSalePageRepo, pr *MockPixelRepo) {},
			wantErr: ErrInvalidSlug,
		},
		{
			name: "invalid_slug_leading_hyphen",
			input: CreateSalePageInput{
				Name:         "My Page",
				Slug:         "-my-page",
				TemplateName: "default",
				Content:      validV1Content,
			},
			setup:   func(sp *MockSalePageRepo, pr *MockPixelRepo) {},
			wantErr: ErrInvalidSlug,
		},
		{
			name: "invalid_content_v1_malformed_json",
			input: CreateSalePageInput{
				Name:         "My Page",
				Slug:         "good-slug",
				TemplateName: "default",
				Content:      json.RawMessage(`{not valid json`),
			},
			setup:   func(sp *MockSalePageRepo, pr *MockPixelRepo) {},
			wantErr: ErrInvalidContent,
		},
		{
			name: "invalid_content_v2_no_blocks",
			input: CreateSalePageInput{
				Name:         "My Page",
				Slug:         "good-slug",
				TemplateName: "blocks",
				Content:      json.RawMessage(`{"version":2,"blocks":[],"style":{},"tracking":{}}`),
			},
			setup:   func(sp *MockSalePageRepo, pr *MockPixelRepo) {},
			wantErr: ErrInvalidContent,
		},
		{
			name: "invalid_content_v2_too_many_blocks",
			input: CreateSalePageInput{
				Name:         "My Page",
				Slug:         "good-slug",
				TemplateName: "blocks",
				Content:      generateTooManyBlocksContent(101),
			},
			setup:   func(sp *MockSalePageRepo, pr *MockPixelRepo) {},
			wantErr: ErrInvalidContent,
		},
		{
			name: "pixel_not_found",
			input: CreateSalePageInput{
				Name:         "My Page",
				Slug:         "good-slug",
				TemplateName: "default",
				Content:      validV1Content,
				PixelIDs:     []string{"pixel-missing"},
			},
			setup: func(sp *MockSalePageRepo, pr *MockPixelRepo) {
				sp.On("SlugExists", mock.Anything, "good-slug").Return(false, nil)
				pr.On("GetByIDs", mock.Anything, []string{"pixel-missing"}).Return([]*domain.Pixel{}, nil)
			},
			wantErr: ErrPixelNotFound,
		},
		{
			name: "pixel_not_owned",
			input: CreateSalePageInput{
				Name:         "My Page",
				Slug:         "good-slug",
				TemplateName: "default",
				Content:      validV1Content,
				PixelIDs:     []string{"pixel-other"},
			},
			setup: func(sp *MockSalePageRepo, pr *MockPixelRepo) {
				sp.On("SlugExists", mock.Anything, "good-slug").Return(false, nil)
				pr.On("GetByIDs", mock.Anything, []string{"pixel-other"}).Return([]*domain.Pixel{
					{ID: "pixel-other", CustomerID: "cust-other"},
				}, nil)
			},
			wantErr: ErrPixelNotOwned,
		},
		{
			name: "db_unique_constraint",
			input: CreateSalePageInput{
				Name:         "My Page",
				Slug:         "racy-slug",
				TemplateName: "default",
				Content:      validV1Content,
			},
			setup: func(sp *MockSalePageRepo, pr *MockPixelRepo) {
				sp.On("SlugExists", mock.Anything, "racy-slug").Return(false, nil)
				sp.On("Create", mock.Anything, mock.AnythingOfType("*domain.SalePage")).Return(
					&pgconn.PgError{Code: "23505"},
				)
			},
			wantErr: ErrSlugTaken,
		},
		{
			name: "nil_pixel_ids_becomes_empty_slice",
			input: CreateSalePageInput{
				Name:         "My Page",
				Slug:         "no-pixels",
				TemplateName: "default",
				Content:      validV1Content,
				PixelIDs:     nil,
			},
			setup: func(sp *MockSalePageRepo, pr *MockPixelRepo) {
				sp.On("SlugExists", mock.Anything, "no-pixels").Return(false, nil)
				sp.On("Create", mock.Anything, mock.MatchedBy(func(p *domain.SalePage) bool {
					return p.PixelIDs != nil && len(p.PixelIDs) == 0
				})).Return(nil)
			},
			check: func(t *testing.T, page *domain.SalePage) {
				assert.NotNil(t, page.PixelIDs)
				assert.Empty(t, page.PixelIDs)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, salePageRepo, _, pixelRepo := newTestSalePageService(t)
			tt.setup(salePageRepo, pixelRepo)

			page, err := svc.Create(context.Background(), "cust-1", tt.input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, page)
			} else if tt.errMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, page)
			} else {
				require.NoError(t, err)
				require.NotNil(t, page)
				if tt.check != nil {
					tt.check(t, page)
				}
			}
			salePageRepo.AssertExpectations(t)
			pixelRepo.AssertExpectations(t)
		})
	}
}

func TestSalePageService_Create_QuotaExceeded(t *testing.T) {
	svc, m := newTestSalePageServiceWithQuota(t)

	// Set up quota check: sandbox plan allows 1 sale page, customer already has 1
	m.customerRepo.On("GetByID", mock.Anything, "cust-1").Return(&domain.Customer{
		ID:   "cust-1",
		Plan: domain.PlanSandbox,
	}, nil)
	m.subRepo.On("GetPixelSlotQuantity", mock.Anything, "cust-1").Return(0, nil)
	m.creditRepo.On("GetActiveByCustomerID", mock.Anything, "cust-1").Return([]*domain.ReplayCredit{}, nil)
	m.salePageRepo.On("CountByCustomerID", mock.Anything, "cust-1").Return(1, nil)
	m.usageRepo.On("GetCurrentMonth", mock.Anything, "cust-1").Return(nil, nil)

	input := CreateSalePageInput{
		Name:         "My Page",
		Slug:         "my-page",
		TemplateName: "default",
		Content:      validV1Content,
	}

	page, err := svc.Create(context.Background(), "cust-1", input)

	assert.ErrorIs(t, err, ErrQuotaSalePagesExceeded)
	assert.Nil(t, page)
	m.customerRepo.AssertExpectations(t)
}

// ---------- GetByID ----------

func TestSalePageService_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		customerID string
		pageID     string
		setup      func(*MockSalePageRepo)
		wantErr    error
	}{
		{
			name:       "success",
			customerID: "cust-1",
			pageID:     "page-1",
			setup: func(sp *MockSalePageRepo) {
				sp.On("GetByID", mock.Anything, "page-1").Return(&domain.SalePage{
					ID:         "page-1",
					CustomerID: "cust-1",
					Name:       "Test Page",
				}, nil)
			},
		},
		{
			name:       "not_found",
			customerID: "cust-1",
			pageID:     "nonexistent",
			setup: func(sp *MockSalePageRepo) {
				sp.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)
			},
			wantErr: ErrSalePageNotFound,
		},
		{
			name:       "not_owned",
			customerID: "cust-2",
			pageID:     "page-1",
			setup: func(sp *MockSalePageRepo) {
				sp.On("GetByID", mock.Anything, "page-1").Return(&domain.SalePage{
					ID:         "page-1",
					CustomerID: "cust-1",
				}, nil)
			},
			wantErr: ErrSalePageNotOwned,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, salePageRepo, _, _ := newTestSalePageService(t)
			tt.setup(salePageRepo)

			page, err := svc.GetByID(context.Background(), tt.customerID, tt.pageID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, page)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, page)
			}
			salePageRepo.AssertExpectations(t)
		})
	}
}

// ---------- List ----------

func TestSalePageService_List(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, salePageRepo, _, _ := newTestSalePageService(t)

		salePageRepo.On("ListByCustomerID", mock.Anything, "cust-1", 20, 0).Return([]*domain.SalePage{
			{ID: "page-1", CustomerID: "cust-1", Name: "Page 1"},
			{ID: "page-2", CustomerID: "cust-1", Name: "Page 2"},
		}, 2, nil)

		pages, total, err := svc.List(context.Background(), "cust-1", 1, 20)

		assert.NoError(t, err)
		assert.Len(t, pages, 2)
		assert.Equal(t, 2, total)
		salePageRepo.AssertExpectations(t)
	})

	t.Run("empty_returns_empty_slice", func(t *testing.T) {
		svc, salePageRepo, _, _ := newTestSalePageService(t)

		salePageRepo.On("ListByCustomerID", mock.Anything, "cust-1", 20, 0).Return(nil, 0, nil)

		pages, total, err := svc.List(context.Background(), "cust-1", 1, 20)

		assert.NoError(t, err)
		assert.NotNil(t, pages)
		assert.Len(t, pages, 0)
		assert.Equal(t, 0, total)
		salePageRepo.AssertExpectations(t)
	})

	t.Run("repo_error", func(t *testing.T) {
		svc, salePageRepo, _, _ := newTestSalePageService(t)

		salePageRepo.On("ListByCustomerID", mock.Anything, "cust-1", 20, 0).Return(nil, 0, errors.New("db error"))

		pages, total, err := svc.List(context.Background(), "cust-1", 1, 20)

		assert.Error(t, err)
		assert.Nil(t, pages)
		assert.Equal(t, 0, total)
		salePageRepo.AssertExpectations(t)
	})
}

// ---------- Update ----------

func TestSalePageService_Update(t *testing.T) {
	tests := []struct {
		name       string
		customerID string
		pageID     string
		input      UpdateSalePageInput
		setup      func(*MockSalePageRepo, *MockPixelRepo)
		wantErr    error
		check      func(*testing.T, *domain.SalePage)
	}{
		{
			name:       "success_name",
			customerID: "cust-1",
			pageID:     "page-1",
			input:      UpdateSalePageInput{Name: strPtr("Updated Name")},
			setup: func(sp *MockSalePageRepo, pr *MockPixelRepo) {
				sp.On("GetByID", mock.Anything, "page-1").Return(&domain.SalePage{
					ID:         "page-1",
					CustomerID: "cust-1",
					Name:       "Old Name",
					Slug:       "old-slug",
				}, nil)
				sp.On("Update", mock.Anything, mock.AnythingOfType("*domain.SalePage")).Return(nil)
			},
			check: func(t *testing.T, page *domain.SalePage) {
				assert.Equal(t, "Updated Name", page.Name)
			},
		},
		{
			name:       "success_slug",
			customerID: "cust-1",
			pageID:     "page-1",
			input:      UpdateSalePageInput{Slug: strPtr("new-slug")},
			setup: func(sp *MockSalePageRepo, pr *MockPixelRepo) {
				sp.On("GetByID", mock.Anything, "page-1").Return(&domain.SalePage{
					ID:         "page-1",
					CustomerID: "cust-1",
					Name:       "My Page",
					Slug:       "old-slug",
				}, nil)
				sp.On("SlugExists", mock.Anything, "new-slug").Return(false, nil)
				sp.On("Update", mock.Anything, mock.AnythingOfType("*domain.SalePage")).Return(nil)
			},
			check: func(t *testing.T, page *domain.SalePage) {
				assert.Equal(t, "new-slug", page.Slug)
			},
		},
		{
			name:       "success_same_slug_no_check",
			customerID: "cust-1",
			pageID:     "page-1",
			input:      UpdateSalePageInput{Slug: strPtr("same-slug")},
			setup: func(sp *MockSalePageRepo, pr *MockPixelRepo) {
				sp.On("GetByID", mock.Anything, "page-1").Return(&domain.SalePage{
					ID:         "page-1",
					CustomerID: "cust-1",
					Name:       "My Page",
					Slug:       "same-slug",
				}, nil)
				// SlugExists should NOT be called when slug is unchanged
				sp.On("Update", mock.Anything, mock.AnythingOfType("*domain.SalePage")).Return(nil)
			},
			check: func(t *testing.T, page *domain.SalePage) {
				assert.Equal(t, "same-slug", page.Slug)
			},
		},
		{
			name:       "slug_taken",
			customerID: "cust-1",
			pageID:     "page-1",
			input:      UpdateSalePageInput{Slug: strPtr("taken-slug")},
			setup: func(sp *MockSalePageRepo, pr *MockPixelRepo) {
				sp.On("GetByID", mock.Anything, "page-1").Return(&domain.SalePage{
					ID:         "page-1",
					CustomerID: "cust-1",
					Slug:       "old-slug",
				}, nil)
				sp.On("SlugExists", mock.Anything, "taken-slug").Return(true, nil)
			},
			wantErr: ErrSlugTaken,
		},
		{
			name:       "slug_reserved",
			customerID: "cust-1",
			pageID:     "page-1",
			input:      UpdateSalePageInput{Slug: strPtr("admin")},
			setup: func(sp *MockSalePageRepo, pr *MockPixelRepo) {
				sp.On("GetByID", mock.Anything, "page-1").Return(&domain.SalePage{
					ID:         "page-1",
					CustomerID: "cust-1",
					Slug:       "old-slug",
				}, nil)
			},
			wantErr: ErrSlugTaken,
		},
		{
			name:       "slug_invalid",
			customerID: "cust-1",
			pageID:     "page-1",
			input:      UpdateSalePageInput{Slug: strPtr("Invalid Slug")},
			setup: func(sp *MockSalePageRepo, pr *MockPixelRepo) {
				sp.On("GetByID", mock.Anything, "page-1").Return(&domain.SalePage{
					ID:         "page-1",
					CustomerID: "cust-1",
					Slug:       "old-slug",
				}, nil)
			},
			wantErr: ErrInvalidSlug,
		},
		{
			name:       "not_found",
			customerID: "cust-1",
			pageID:     "nonexistent",
			input:      UpdateSalePageInput{Name: strPtr("Updated")},
			setup: func(sp *MockSalePageRepo, pr *MockPixelRepo) {
				sp.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)
			},
			wantErr: ErrSalePageNotFound,
		},
		{
			name:       "not_owned",
			customerID: "cust-2",
			pageID:     "page-1",
			input:      UpdateSalePageInput{Name: strPtr("Updated")},
			setup: func(sp *MockSalePageRepo, pr *MockPixelRepo) {
				sp.On("GetByID", mock.Anything, "page-1").Return(&domain.SalePage{
					ID:         "page-1",
					CustomerID: "cust-1",
				}, nil)
			},
			wantErr: ErrSalePageNotOwned,
		},
		{
			name:       "pixel_ownership_check",
			customerID: "cust-1",
			pageID:     "page-1",
			input: UpdateSalePageInput{
				PixelIDs: &[]string{"pixel-1", "pixel-2"},
			},
			setup: func(sp *MockSalePageRepo, pr *MockPixelRepo) {
				sp.On("GetByID", mock.Anything, "page-1").Return(&domain.SalePage{
					ID:         "page-1",
					CustomerID: "cust-1",
					Slug:       "my-slug",
				}, nil)
				pr.On("GetByIDs", mock.Anything, []string{"pixel-1", "pixel-2"}).Return([]*domain.Pixel{
					{ID: "pixel-1", CustomerID: "cust-1"},
					{ID: "pixel-2", CustomerID: "cust-1"},
				}, nil)
				sp.On("Update", mock.Anything, mock.AnythingOfType("*domain.SalePage")).Return(nil)
			},
			check: func(t *testing.T, page *domain.SalePage) {
				assert.Equal(t, []string{"pixel-1", "pixel-2"}, page.PixelIDs)
			},
		},
		{
			name:       "pixel_not_owned_on_update",
			customerID: "cust-1",
			pageID:     "page-1",
			input: UpdateSalePageInput{
				PixelIDs: &[]string{"pixel-other"},
			},
			setup: func(sp *MockSalePageRepo, pr *MockPixelRepo) {
				sp.On("GetByID", mock.Anything, "page-1").Return(&domain.SalePage{
					ID:         "page-1",
					CustomerID: "cust-1",
					Slug:       "my-slug",
				}, nil)
				pr.On("GetByIDs", mock.Anything, []string{"pixel-other"}).Return([]*domain.Pixel{
					{ID: "pixel-other", CustomerID: "cust-other"},
				}, nil)
			},
			wantErr: ErrPixelNotOwned,
		},
		{
			name:       "update_content_valid",
			customerID: "cust-1",
			pageID:     "page-1",
			input: UpdateSalePageInput{
				Content: rawPtr(validV2Content),
			},
			setup: func(sp *MockSalePageRepo, pr *MockPixelRepo) {
				sp.On("GetByID", mock.Anything, "page-1").Return(&domain.SalePage{
					ID:         "page-1",
					CustomerID: "cust-1",
					Slug:       "my-slug",
					Content:    validV1Content,
				}, nil)
				sp.On("Update", mock.Anything, mock.AnythingOfType("*domain.SalePage")).Return(nil)
			},
			check: func(t *testing.T, page *domain.SalePage) {
				assert.NotNil(t, page.Content)
			},
		},
		{
			name:       "update_content_invalid",
			customerID: "cust-1",
			pageID:     "page-1",
			input: UpdateSalePageInput{
				Content: rawPtr(json.RawMessage(`{not valid`)),
			},
			setup: func(sp *MockSalePageRepo, pr *MockPixelRepo) {
				sp.On("GetByID", mock.Anything, "page-1").Return(&domain.SalePage{
					ID:         "page-1",
					CustomerID: "cust-1",
					Slug:       "my-slug",
				}, nil)
			},
			wantErr: ErrInvalidContent,
		},
		{
			name:       "update_published",
			customerID: "cust-1",
			pageID:     "page-1",
			input: UpdateSalePageInput{
				IsPublished: boolPtr(true),
			},
			setup: func(sp *MockSalePageRepo, pr *MockPixelRepo) {
				sp.On("GetByID", mock.Anything, "page-1").Return(&domain.SalePage{
					ID:          "page-1",
					CustomerID:  "cust-1",
					Slug:        "my-slug",
					IsPublished: false,
				}, nil)
				sp.On("Update", mock.Anything, mock.AnythingOfType("*domain.SalePage")).Return(nil)
			},
			check: func(t *testing.T, page *domain.SalePage) {
				assert.True(t, page.IsPublished)
			},
		},
		{
			name:       "db_unique_constraint_on_update",
			customerID: "cust-1",
			pageID:     "page-1",
			input:      UpdateSalePageInput{Slug: strPtr("racy-slug")},
			setup: func(sp *MockSalePageRepo, pr *MockPixelRepo) {
				sp.On("GetByID", mock.Anything, "page-1").Return(&domain.SalePage{
					ID:         "page-1",
					CustomerID: "cust-1",
					Slug:       "old-slug",
				}, nil)
				sp.On("SlugExists", mock.Anything, "racy-slug").Return(false, nil)
				sp.On("Update", mock.Anything, mock.AnythingOfType("*domain.SalePage")).Return(
					&pgconn.PgError{Code: "23505"},
				)
			},
			wantErr: ErrSlugTaken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, salePageRepo, _, pixelRepo := newTestSalePageService(t)
			tt.setup(salePageRepo, pixelRepo)

			page, err := svc.Update(context.Background(), tt.customerID, tt.pageID, tt.input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, page)
			} else {
				require.NoError(t, err)
				require.NotNil(t, page)
				if tt.check != nil {
					tt.check(t, page)
				}
			}
			salePageRepo.AssertExpectations(t)
			pixelRepo.AssertExpectations(t)
		})
	}
}

// ---------- Delete ----------

func TestSalePageService_Delete(t *testing.T) {
	tests := []struct {
		name       string
		customerID string
		pageID     string
		setup      func(*MockSalePageRepo)
		wantErr    error
	}{
		{
			name:       "success",
			customerID: "cust-1",
			pageID:     "page-1",
			setup: func(sp *MockSalePageRepo) {
				sp.On("GetByID", mock.Anything, "page-1").Return(&domain.SalePage{
					ID:         "page-1",
					CustomerID: "cust-1",
					Slug:       "my-slug",
				}, nil)
				sp.On("Delete", mock.Anything, "page-1").Return(nil)
			},
		},
		{
			name:       "not_found",
			customerID: "cust-1",
			pageID:     "nonexistent",
			setup: func(sp *MockSalePageRepo) {
				sp.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)
			},
			wantErr: ErrSalePageNotFound,
		},
		{
			name:       "not_owned",
			customerID: "cust-2",
			pageID:     "page-1",
			setup: func(sp *MockSalePageRepo) {
				sp.On("GetByID", mock.Anything, "page-1").Return(&domain.SalePage{
					ID:         "page-1",
					CustomerID: "cust-1",
				}, nil)
			},
			wantErr: ErrSalePageNotOwned,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, salePageRepo, _, _ := newTestSalePageService(t)
			tt.setup(salePageRepo)

			err := svc.Delete(context.Background(), tt.customerID, tt.pageID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			salePageRepo.AssertExpectations(t)
		})
	}
}

// ---------- GetBySlug ----------

func TestSalePageService_GetBySlug(t *testing.T) {
	tests := []struct {
		name    string
		slug    string
		setup   func(*MockSalePageRepo)
		wantErr error
	}{
		{
			name: "published",
			slug: "my-page",
			setup: func(sp *MockSalePageRepo) {
				sp.On("GetBySlug", mock.Anything, "my-page").Return(&domain.SalePage{
					ID:          "page-1",
					CustomerID:  "cust-1",
					Slug:        "my-page",
					IsPublished: true,
				}, nil)
			},
		},
		{
			name: "not_published",
			slug: "draft-page",
			setup: func(sp *MockSalePageRepo) {
				sp.On("GetBySlug", mock.Anything, "draft-page").Return(&domain.SalePage{
					ID:          "page-1",
					CustomerID:  "cust-1",
					Slug:        "draft-page",
					IsPublished: false,
				}, nil)
			},
			wantErr: ErrSalePageNotFound,
		},
		{
			name: "not_found",
			slug: "nonexistent",
			setup: func(sp *MockSalePageRepo) {
				sp.On("GetBySlug", mock.Anything, "nonexistent").Return(nil, nil)
			},
			wantErr: ErrSalePageNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, salePageRepo, _, _ := newTestSalePageService(t)
			tt.setup(salePageRepo)

			page, err := svc.GetBySlug(context.Background(), tt.slug)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, page)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, page)
				assert.True(t, page.IsPublished)
			}
			salePageRepo.AssertExpectations(t)
		})
	}
}

// ---------- GetPublishData ----------

func TestSalePageService_GetPublishData(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, salePageRepo, customerRepo, pixelRepo := newTestSalePageService(t)

		salePageRepo.On("GetBySlug", mock.Anything, "my-page").Return(&domain.SalePage{
			ID:          "page-1",
			CustomerID:  "cust-1",
			Slug:        "my-page",
			IsPublished: true,
			PixelIDs:    []string{"pixel-1"},
		}, nil)
		customerRepo.On("GetByID", mock.Anything, "cust-1").Return(&domain.Customer{
			ID:     "cust-1",
			APIKey: "api-key-123",
		}, nil)
		pixelRepo.On("GetByIDs", mock.Anything, []string{"pixel-1"}).Return([]*domain.Pixel{
			{ID: "pixel-1", FBPixelID: "fb-123"},
		}, nil)

		data, err := svc.GetPublishData(context.Background(), "my-page")

		require.NoError(t, err)
		require.NotNil(t, data)
		assert.Equal(t, "page-1", data.Page.ID)
		assert.Equal(t, "api-key-123", data.APIKey)
		assert.Len(t, data.Pixels, 1)
		assert.Equal(t, "pixel-1", data.Pixels[0].PixelID)
		assert.Equal(t, "fb-123", data.Pixels[0].FBPixelID)
		salePageRepo.AssertExpectations(t)
		customerRepo.AssertExpectations(t)
		pixelRepo.AssertExpectations(t)
	})

	t.Run("success_no_pixels", func(t *testing.T) {
		svc, salePageRepo, customerRepo, _ := newTestSalePageService(t)

		salePageRepo.On("GetBySlug", mock.Anything, "no-pixel-page").Return(&domain.SalePage{
			ID:          "page-2",
			CustomerID:  "cust-1",
			Slug:        "no-pixel-page",
			IsPublished: true,
			PixelIDs:    []string{},
		}, nil)
		customerRepo.On("GetByID", mock.Anything, "cust-1").Return(&domain.Customer{
			ID:     "cust-1",
			APIKey: "api-key-123",
		}, nil)

		data, err := svc.GetPublishData(context.Background(), "no-pixel-page")

		require.NoError(t, err)
		require.NotNil(t, data)
		assert.Nil(t, data.Pixels)
		salePageRepo.AssertExpectations(t)
		customerRepo.AssertExpectations(t)
	})

	t.Run("page_not_found", func(t *testing.T) {
		svc, salePageRepo, _, _ := newTestSalePageService(t)

		salePageRepo.On("GetBySlug", mock.Anything, "nonexistent").Return(nil, nil)

		data, err := svc.GetPublishData(context.Background(), "nonexistent")

		assert.ErrorIs(t, err, ErrSalePageNotFound)
		assert.Nil(t, data)
		salePageRepo.AssertExpectations(t)
	})

	t.Run("page_not_published", func(t *testing.T) {
		svc, salePageRepo, _, _ := newTestSalePageService(t)

		salePageRepo.On("GetBySlug", mock.Anything, "draft").Return(&domain.SalePage{
			ID:          "page-1",
			CustomerID:  "cust-1",
			Slug:        "draft",
			IsPublished: false,
		}, nil)

		data, err := svc.GetPublishData(context.Background(), "draft")

		assert.ErrorIs(t, err, ErrSalePageNotFound)
		assert.Nil(t, data)
		salePageRepo.AssertExpectations(t)
	})

	t.Run("customer_not_found", func(t *testing.T) {
		svc, salePageRepo, customerRepo, _ := newTestSalePageService(t)

		salePageRepo.On("GetBySlug", mock.Anything, "orphan-page").Return(&domain.SalePage{
			ID:          "page-1",
			CustomerID:  "cust-deleted",
			Slug:        "orphan-page",
			IsPublished: true,
		}, nil)
		customerRepo.On("GetByID", mock.Anything, "cust-deleted").Return(nil, nil)

		data, err := svc.GetPublishData(context.Background(), "orphan-page")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "customer not found")
		assert.Nil(t, data)
		salePageRepo.AssertExpectations(t)
		customerRepo.AssertExpectations(t)
	})

	t.Run("cache_hit", func(t *testing.T) {
		svc, salePageRepo, customerRepo, pixelRepo := newTestSalePageService(t)

		salePageRepo.On("GetBySlug", mock.Anything, "cached-page").Return(&domain.SalePage{
			ID:          "page-1",
			CustomerID:  "cust-1",
			Slug:        "cached-page",
			IsPublished: true,
			PixelIDs:    []string{},
		}, nil)
		customerRepo.On("GetByID", mock.Anything, "cust-1").Return(&domain.Customer{
			ID:     "cust-1",
			APIKey: "api-key-123",
		}, nil)

		// First call: populates cache
		data1, err := svc.GetPublishData(context.Background(), "cached-page")
		require.NoError(t, err)
		require.NotNil(t, data1)

		// Second call: should use cache (repo not called again)
		data2, err := svc.GetPublishData(context.Background(), "cached-page")
		require.NoError(t, err)
		require.NotNil(t, data2)

		assert.Equal(t, data1, data2)

		// GetBySlug should have been called exactly once
		salePageRepo.AssertNumberOfCalls(t, "GetBySlug", 1)
		customerRepo.AssertNumberOfCalls(t, "GetByID", 1)
		_ = pixelRepo
	})
}

// ---------- validateContent ----------

func TestValidateContent(t *testing.T) {
	tests := []struct {
		name    string
		content json.RawMessage
		wantErr error
		errMsg  string
	}{
		{
			name:    "valid_v1",
			content: validV1Content,
		},
		{
			name:    "valid_v2_text_block",
			content: json.RawMessage(`{"version":2,"blocks":[{"id":"b1","type":"text","text":"hello world"}],"style":{},"tracking":{}}`),
		},
		{
			name:    "valid_v2_image_block",
			content: json.RawMessage(`{"version":2,"blocks":[{"id":"b1","type":"image","image_url":"https://example.com/img.jpg"}],"style":{},"tracking":{}}`),
		},
		{
			name:    "valid_v2_button_line",
			content: json.RawMessage(`{"version":2,"blocks":[{"id":"b1","type":"button","button_style":"line","button_text":"Add Line"}],"style":{},"tracking":{}}`),
		},
		{
			name:    "valid_v2_button_website",
			content: json.RawMessage(`{"version":2,"blocks":[{"id":"b1","type":"button","button_style":"website","button_url":"https://example.com","button_text":"Visit"}],"style":{},"tracking":{}}`),
		},
		{
			name:    "valid_v2_button_custom",
			content: json.RawMessage(`{"version":2,"blocks":[{"id":"b1","type":"button","button_style":"custom","button_url":"https://example.com","button_text":"Go"}],"style":{},"tracking":{}}`),
		},
		{
			name:    "valid_v2_multiple_block_types",
			content: json.RawMessage(`{"version":2,"blocks":[{"id":"b1","type":"text","text":"Title"},{"id":"b2","type":"image","image_url":"https://example.com/img.jpg"},{"id":"b3","type":"button","button_style":"line","button_text":"Add Line"}],"style":{},"tracking":{}}`),
		},
		{
			name:    "valid_v1_with_style",
			content: json.RawMessage(`{"hero":{"title":"T"},"body":{"description":"D"},"cta":{},"contact":{},"tracking":{},"style":{"bg_color":"#ffffff","accent_color":"#667eea","text_color":"#1a1a2e"}}`),
		},
		{
			name:    "malformed_json",
			content: json.RawMessage(`{not valid`),
			wantErr: ErrInvalidContent,
		},
		{
			name:    "v2_empty_blocks",
			content: json.RawMessage(`{"version":2,"blocks":[],"style":{},"tracking":{}}`),
			wantErr: ErrInvalidContent,
			errMsg:  "at least one block",
		},
		{
			name:    "v2_too_many_blocks",
			content: generateTooManyBlocksContent(101),
			wantErr: ErrInvalidContent,
			errMsg:  "too many blocks",
		},
		{
			name:    "v2_block_missing_id",
			content: json.RawMessage(`{"version":2,"blocks":[{"type":"text","text":"hello"}],"style":{},"tracking":{}}`),
			wantErr: ErrInvalidContent,
			errMsg:  "missing ID",
		},
		{
			name:    "v2_unknown_block_type",
			content: json.RawMessage(`{"version":2,"blocks":[{"id":"b1","type":"video"}],"style":{},"tracking":{}}`),
			wantErr: ErrInvalidContent,
			errMsg:  "unknown type",
		},
		{
			name:    "v2_invalid_button_style",
			content: json.RawMessage(`{"version":2,"blocks":[{"id":"b1","type":"button","button_style":"fancy"}],"style":{},"tracking":{}}`),
			wantErr: ErrInvalidContent,
			errMsg:  "invalid button_style",
		},
		{
			name:    "invalid_hex_color_bg",
			content: json.RawMessage(`{"hero":{},"body":{},"cta":{},"contact":{},"tracking":{},"style":{"bg_color":"red"}}`),
			wantErr: ErrInvalidContent,
			errMsg:  "bg_color must be a valid hex color",
		},
		{
			name:    "invalid_hex_color_accent",
			content: json.RawMessage(`{"hero":{},"body":{},"cta":{},"contact":{},"tracking":{},"style":{"accent_color":"#xyz"}}`),
			wantErr: ErrInvalidContent,
			errMsg:  "accent_color must be a valid hex color",
		},
		{
			name:    "invalid_hex_color_text",
			content: json.RawMessage(`{"hero":{},"body":{},"cta":{},"contact":{},"tracking":{},"style":{"text_color":"123456"}}`),
			wantErr: ErrInvalidContent,
			errMsg:  "text_color must be a valid hex color",
		},
		{
			name:    "javascript_url_rejected_in_image",
			content: json.RawMessage(`{"version":2,"blocks":[{"id":"b1","type":"image","image_url":"javascript:alert(1)"}],"style":{},"tracking":{}}`),
			wantErr: ErrInvalidContent,
			errMsg:  "URL must use http or https scheme",
		},
		{
			name:    "javascript_url_rejected_in_button",
			content: json.RawMessage(`{"version":2,"blocks":[{"id":"b1","type":"button","button_style":"website","button_url":"javascript:alert(1)"}],"style":{},"tracking":{}}`),
			wantErr: ErrInvalidContent,
			errMsg:  "URL must use http or https scheme",
		},
		{
			name:    "data_url_rejected",
			content: json.RawMessage(`{"version":2,"blocks":[{"id":"b1","type":"image","image_url":"data:text/html,<h1>evil</h1>"}],"style":{},"tracking":{}}`),
			wantErr: ErrInvalidContent,
			errMsg:  "URL must use http or https scheme",
		},
		{
			name:    "v2_image_block_with_valid_link",
			content: json.RawMessage(`{"version":2,"blocks":[{"id":"b1","type":"image","image_url":"https://example.com/img.jpg","link_url":"https://example.com"}],"style":{},"tracking":{}}`),
		},
		{
			name:    "v2_image_block_javascript_link_rejected",
			content: json.RawMessage(`{"version":2,"blocks":[{"id":"b1","type":"image","image_url":"https://example.com/img.jpg","link_url":"javascript:void(0)"}],"style":{},"tracking":{}}`),
			wantErr: ErrInvalidContent,
			errMsg:  "URL must use http or https scheme",
		},
		{
			name:    "bg_image_url_must_be_https",
			content: json.RawMessage(`{"hero":{},"body":{},"cta":{},"contact":{},"tracking":{},"style":{"bg_image_url":"http://example.com/bg.jpg"}}`),
			wantErr: ErrInvalidContent,
			errMsg:  "bg_image_url must start with https://",
		},
		{
			name:    "v2_style_invalid_color",
			content: json.RawMessage(`{"version":2,"blocks":[{"id":"b1","type":"text","text":"hi"}],"style":{"bg_color":"bad"},"tracking":{}}`),
			wantErr: ErrInvalidContent,
			errMsg:  "bg_color must be a valid hex color",
		},
		{
			name:    "v2_exactly_100_blocks_valid",
			content: generateTooManyBlocksContent(100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateContent(tt.content)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------- validateSlug ----------

func TestValidateSlug(t *testing.T) {
	tests := []struct {
		name    string
		slug    string
		wantErr error
	}{
		{name: "valid_simple", slug: "hello"},
		{name: "valid_with_hyphen", slug: "my-page"},
		{name: "valid_with_numbers", slug: "test123"},
		{name: "valid_numbers_and_hyphens", slug: "page-1-v2"},
		{name: "valid_single_char", slug: "a"},
		{name: "invalid_uppercase", slug: "Hello", wantErr: ErrInvalidSlug},
		{name: "invalid_space", slug: "my page", wantErr: ErrInvalidSlug},
		{name: "invalid_underscore", slug: "my_page", wantErr: ErrInvalidSlug},
		{name: "invalid_leading_hyphen", slug: "-my-page", wantErr: ErrInvalidSlug},
		{name: "invalid_trailing_hyphen", slug: "my-page-", wantErr: ErrInvalidSlug},
		{name: "invalid_double_hyphen", slug: "my--page", wantErr: ErrInvalidSlug},
		{name: "invalid_special_chars", slug: "my@page", wantErr: ErrInvalidSlug},
		{name: "reserved_admin", slug: "admin", wantErr: ErrSlugTaken},
		{name: "reserved_api", slug: "api", wantErr: ErrSlugTaken},
		{name: "reserved_login", slug: "login", wantErr: ErrSlugTaken},
		{name: "reserved_register", slug: "register", wantErr: ErrSlugTaken},
		{name: "reserved_dashboard", slug: "dashboard", wantErr: ErrSlugTaken},
		{name: "reserved_health", slug: "health", wantErr: ErrSlugTaken},
		{name: "reserved_p", slug: "p", wantErr: ErrSlugTaken},
		{name: "empty_string", slug: "", wantErr: ErrInvalidSlug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSlug(tt.slug)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------- validateSafeURL ----------

func TestValidateSafeURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{name: "empty", url: ""},
		{name: "https", url: "https://example.com"},
		{name: "http", url: "http://example.com"},
		{name: "relative", url: "/path/to/page"},
		{name: "javascript", url: "javascript:alert(1)", wantErr: true, errMsg: "http or https"},
		{name: "data", url: "data:text/html,hi", wantErr: true, errMsg: "http or https"},
		{name: "ftp", url: "ftp://example.com/file", wantErr: true, errMsg: "http or https"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSafeURL(tt.url)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------- validatePageStyle ----------

func TestValidatePageStyle(t *testing.T) {
	tests := []struct {
		name    string
		style   domain.PageStyle
		wantErr bool
	}{
		{name: "empty_style", style: domain.PageStyle{}},
		{name: "valid_colors", style: domain.PageStyle{BgColor: "#ffffff", AccentColor: "#667eea", TextColor: "#1a1a2e"}},
		{name: "valid_bg_image", style: domain.PageStyle{BgImageURL: "https://example.com/bg.jpg"}},
		{name: "invalid_bg_color", style: domain.PageStyle{BgColor: "red"}, wantErr: true},
		{name: "invalid_accent_color", style: domain.PageStyle{AccentColor: "#gg0000"}, wantErr: true},
		{name: "invalid_text_color", style: domain.PageStyle{TextColor: "000000"}, wantErr: true},
		{name: "http_bg_image_rejected", style: domain.PageStyle{BgImageURL: "http://example.com/bg.jpg"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePageStyle(tt.style)

			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidContent)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------- generateRandomSlug ----------

func TestGenerateRandomSlug(t *testing.T) {
	slug, err := generateRandomSlug()
	require.NoError(t, err)
	assert.True(t, len(slug) == 10, "slug should be 'p-' + 8 chars = 10 chars, got %d: %s", len(slug), slug)
	assert.Equal(t, "p-", slug[:2])
	// Validate the generated slug passes validation
	assert.NoError(t, validateSlug(slug))
}

func TestGenerateRandomSlug_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		slug, err := generateRandomSlug()
		require.NoError(t, err)
		assert.False(t, seen[slug], "duplicate slug generated: %s", slug)
		seen[slug] = true
	}
}

// ---------- helpers ----------

func rawPtr(r json.RawMessage) *json.RawMessage {
	return &r
}

func boolPtr(b bool) *bool {
	return &b
}

// generateTooManyBlocksContent generates v2 content with the specified number of blocks.
func generateTooManyBlocksContent(n int) json.RawMessage {
	blocks := make([]domain.Block, n)
	for i := 0; i < n; i++ {
		blocks[i] = domain.Block{
			ID:   "b" + json.Number(rune('0'+i%10)).String(),
			Type: domain.BlockTypeText,
			Text: "block",
		}
	}
	// Use a simpler approach: build manually with proper unique IDs
	var blocksJSON []byte
	blocksJSON = append(blocksJSON, '[')
	for i := 0; i < n; i++ {
		if i > 0 {
			blocksJSON = append(blocksJSON, ',')
		}
		id := "b" + intToStr(i)
		blocksJSON = append(blocksJSON, []byte(`{"id":"`+id+`","type":"text","text":"block"}`)...)
	}
	blocksJSON = append(blocksJSON, ']')

	return json.RawMessage(`{"version":2,"blocks":` + string(blocksJSON) + `,"style":{},"tracking":{}}`)
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}
