package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

func newTestPixelService() (*PixelService, *MockPixelRepo) {
	pixelRepo := new(MockPixelRepo)
	svc := NewPixelService(pixelRepo)
	return svc, pixelRepo
}

func TestPixelService_Create(t *testing.T) {
	svc, pixelRepo := newTestPixelService()

	pixelRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Pixel")).Return(nil)

	pixel, err := svc.Create(context.Background(), "cust-1", CreatePixelInput{
		FBPixelID:     "123456",
		FBAccessToken: "token-abc",
		Name:          "My Pixel",
	})

	assert.NoError(t, err)
	assert.NotNil(t, pixel)
	assert.Equal(t, "cust-1", pixel.CustomerID)
	assert.Equal(t, "123456", pixel.FBPixelID)
	assert.Equal(t, "My Pixel", pixel.Name)
	pixelRepo.AssertExpectations(t)
}

func TestPixelService_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		customerID string
		pixelID    string
		setup      func(*MockPixelRepo)
		wantErr    error
	}{
		{
			name:       "success",
			customerID: "cust-1",
			pixelID:    "pixel-1",
			setup: func(pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID:         "pixel-1",
					CustomerID: "cust-1",
					Name:       "My Pixel",
				}, nil)
			},
			wantErr: nil,
		},
		{
			name:       "not found",
			customerID: "cust-1",
			pixelID:    "nonexistent",
			setup: func(pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)
			},
			wantErr: ErrPixelNotFound,
		},
		{
			name:       "not owned",
			customerID: "cust-2",
			pixelID:    "pixel-1",
			setup: func(pr *MockPixelRepo) {
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
			svc, pixelRepo := newTestPixelService()
			tt.setup(pixelRepo)

			pixel, err := svc.GetByID(context.Background(), tt.customerID, tt.pixelID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, pixel)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, pixel)
			}
			pixelRepo.AssertExpectations(t)
		})
	}
}

func TestPixelService_List(t *testing.T) {
	svc, pixelRepo := newTestPixelService()

	pixelRepo.On("ListByCustomerID", mock.Anything, "cust-1").Return([]*domain.Pixel{
		{ID: "pixel-1", CustomerID: "cust-1", Name: "Pixel 1"},
		{ID: "pixel-2", CustomerID: "cust-1", Name: "Pixel 2"},
	}, nil)

	pixels, err := svc.List(context.Background(), "cust-1")

	assert.NoError(t, err)
	assert.Len(t, pixels, 2)
	pixelRepo.AssertExpectations(t)
}

func TestPixelService_Update(t *testing.T) {
	tests := []struct {
		name       string
		customerID string
		pixelID    string
		input      UpdatePixelInput
		setup      func(*MockPixelRepo)
		wantErr    error
	}{
		{
			name:       "success",
			customerID: "cust-1",
			pixelID:    "pixel-1",
			input: UpdatePixelInput{
				Name: strPtr("Updated Pixel"),
			},
			setup: func(pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID:         "pixel-1",
					CustomerID: "cust-1",
					Name:       "Old Pixel",
				}, nil)
				pr.On("Update", mock.Anything, mock.AnythingOfType("*domain.Pixel")).Return(nil)
			},
			wantErr: nil,
		},
		{
			name:       "not found",
			customerID: "cust-1",
			pixelID:    "nonexistent",
			input:      UpdatePixelInput{Name: strPtr("Updated")},
			setup: func(pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)
			},
			wantErr: ErrPixelNotFound,
		},
		{
			name:       "not owned",
			customerID: "cust-2",
			pixelID:    "pixel-1",
			input:      UpdatePixelInput{Name: strPtr("Updated")},
			setup: func(pr *MockPixelRepo) {
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
			svc, pixelRepo := newTestPixelService()
			tt.setup(pixelRepo)

			pixel, err := svc.Update(context.Background(), tt.customerID, tt.pixelID, tt.input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, pixel)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, pixel)
				assert.Equal(t, "Updated Pixel", pixel.Name)
			}
			pixelRepo.AssertExpectations(t)
		})
	}
}

func TestPixelService_Delete(t *testing.T) {
	tests := []struct {
		name       string
		customerID string
		pixelID    string
		setup      func(*MockPixelRepo)
		wantErr    error
	}{
		{
			name:       "success",
			customerID: "cust-1",
			pixelID:    "pixel-1",
			setup: func(pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID:         "pixel-1",
					CustomerID: "cust-1",
				}, nil)
				pr.On("Delete", mock.Anything, "pixel-1").Return(nil)
			},
			wantErr: nil,
		},
		{
			name:       "not found",
			customerID: "cust-1",
			pixelID:    "nonexistent",
			setup: func(pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)
			},
			wantErr: ErrPixelNotFound,
		},
		{
			name:       "not owned",
			customerID: "cust-2",
			pixelID:    "pixel-1",
			setup: func(pr *MockPixelRepo) {
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
			svc, pixelRepo := newTestPixelService()
			tt.setup(pixelRepo)

			err := svc.Delete(context.Background(), tt.customerID, tt.pixelID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			pixelRepo.AssertExpectations(t)
		})
	}
}

func strPtr(s string) *string { return &s }
