package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

func newTestRuleService() (*RuleService, *MockEventRuleRepo, *MockPixelRepo) {
	ruleRepo := new(MockEventRuleRepo)
	pixelRepo := new(MockPixelRepo)
	svc := NewRuleService(ruleRepo, pixelRepo)
	return svc, ruleRepo, pixelRepo
}

func TestRuleService_Create(t *testing.T) {
	tests := []struct {
		name       string
		customerID string
		pixelID    string
		input      CreateRuleInput
		setup      func(*MockEventRuleRepo, *MockPixelRepo)
		wantErr    error
	}{
		{
			name:       "success",
			customerID: "cust-1",
			pixelID:    "pixel-1",
			input: CreateRuleInput{
				PageURL:     "https://example.com",
				EventName:   "Purchase",
				TriggerType: "click",
				CSSSelector: "#buy-btn",
			},
			setup: func(rr *MockEventRuleRepo, pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID:         "pixel-1",
					CustomerID: "cust-1",
				}, nil)
				rr.On("Create", mock.Anything, mock.AnythingOfType("*domain.EventRule")).Return(nil)
			},
			wantErr: nil,
		},
		{
			name:       "pixel not found",
			customerID: "cust-1",
			pixelID:    "nonexistent",
			input: CreateRuleInput{
				PageURL:     "https://example.com",
				EventName:   "Purchase",
				TriggerType: "click",
			},
			setup: func(rr *MockEventRuleRepo, pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)
			},
			wantErr: ErrPixelNotFound,
		},
		{
			name:       "pixel not owned",
			customerID: "cust-2",
			pixelID:    "pixel-1",
			input: CreateRuleInput{
				PageURL:     "https://example.com",
				EventName:   "Purchase",
				TriggerType: "click",
			},
			setup: func(rr *MockEventRuleRepo, pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID:         "pixel-1",
					CustomerID: "cust-1",
				}, nil)
			},
			wantErr: ErrPixelNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, ruleRepo, pixelRepo := newTestRuleService()
			tt.setup(ruleRepo, pixelRepo)

			rule, err := svc.Create(context.Background(), tt.customerID, tt.pixelID, tt.input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, rule)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, rule)
				assert.Equal(t, tt.pixelID, rule.PixelID)
				assert.Equal(t, tt.input.EventName, rule.EventName)
			}
			ruleRepo.AssertExpectations(t)
			pixelRepo.AssertExpectations(t)
		})
	}
}

func TestRuleService_Update(t *testing.T) {
	tests := []struct {
		name       string
		customerID string
		ruleID     string
		input      UpdateRuleInput
		setup      func(*MockEventRuleRepo, *MockPixelRepo)
		wantErr    error
	}{
		{
			name:       "success",
			customerID: "cust-1",
			ruleID:     "rule-1",
			input:      UpdateRuleInput{EventName: strPtr("AddToCart")},
			setup: func(rr *MockEventRuleRepo, pr *MockPixelRepo) {
				rr.On("GetByID", mock.Anything, "rule-1").Return(&domain.EventRule{
					ID:        "rule-1",
					PixelID:   "pixel-1",
					EventName: "Purchase",
				}, nil)
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID:         "pixel-1",
					CustomerID: "cust-1",
				}, nil)
				rr.On("Update", mock.Anything, mock.AnythingOfType("*domain.EventRule")).Return(nil)
			},
			wantErr: nil,
		},
		{
			name:       "rule not found",
			customerID: "cust-1",
			ruleID:     "nonexistent",
			input:      UpdateRuleInput{EventName: strPtr("AddToCart")},
			setup: func(rr *MockEventRuleRepo, pr *MockPixelRepo) {
				rr.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)
			},
			wantErr: ErrRuleNotFound,
		},
		{
			name:       "pixel not owned",
			customerID: "cust-2",
			ruleID:     "rule-1",
			input:      UpdateRuleInput{EventName: strPtr("AddToCart")},
			setup: func(rr *MockEventRuleRepo, pr *MockPixelRepo) {
				rr.On("GetByID", mock.Anything, "rule-1").Return(&domain.EventRule{
					ID:      "rule-1",
					PixelID: "pixel-1",
				}, nil)
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID:         "pixel-1",
					CustomerID: "cust-1",
				}, nil)
			},
			wantErr: ErrPixelNotOwned,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, ruleRepo, pixelRepo := newTestRuleService()
			tt.setup(ruleRepo, pixelRepo)

			rule, err := svc.Update(context.Background(), tt.customerID, tt.ruleID, tt.input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, rule)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, rule)
				assert.Equal(t, "AddToCart", rule.EventName)
			}
			ruleRepo.AssertExpectations(t)
			pixelRepo.AssertExpectations(t)
		})
	}
}

func TestRuleService_Delete(t *testing.T) {
	tests := []struct {
		name       string
		customerID string
		ruleID     string
		setup      func(*MockEventRuleRepo, *MockPixelRepo)
		wantErr    error
	}{
		{
			name:       "success",
			customerID: "cust-1",
			ruleID:     "rule-1",
			setup: func(rr *MockEventRuleRepo, pr *MockPixelRepo) {
				rr.On("GetByID", mock.Anything, "rule-1").Return(&domain.EventRule{
					ID:      "rule-1",
					PixelID: "pixel-1",
				}, nil)
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID:         "pixel-1",
					CustomerID: "cust-1",
				}, nil)
				rr.On("Delete", mock.Anything, "rule-1").Return(nil)
			},
			wantErr: nil,
		},
		{
			name:       "rule not found",
			customerID: "cust-1",
			ruleID:     "nonexistent",
			setup: func(rr *MockEventRuleRepo, pr *MockPixelRepo) {
				rr.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)
			},
			wantErr: ErrRuleNotFound,
		},
		{
			name:       "pixel not owned",
			customerID: "cust-2",
			ruleID:     "rule-1",
			setup: func(rr *MockEventRuleRepo, pr *MockPixelRepo) {
				rr.On("GetByID", mock.Anything, "rule-1").Return(&domain.EventRule{
					ID:      "rule-1",
					PixelID: "pixel-1",
				}, nil)
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID:         "pixel-1",
					CustomerID: "cust-1",
				}, nil)
			},
			wantErr: ErrPixelNotOwned,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, ruleRepo, pixelRepo := newTestRuleService()
			tt.setup(ruleRepo, pixelRepo)

			err := svc.Delete(context.Background(), tt.customerID, tt.ruleID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			ruleRepo.AssertExpectations(t)
			pixelRepo.AssertExpectations(t)
		})
	}
}
