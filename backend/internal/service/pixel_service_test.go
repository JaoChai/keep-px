package service

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/facebook"
)

func newTestPixelService() (*PixelService, *MockPixelRepo) {
	pixelRepo := new(MockPixelRepo)
	capiClient := facebook.NewCAPIClient("http://localhost:9999")
	svc := NewPixelService(pixelRepo, capiClient, slog.New(slog.NewTextHandler(io.Discard, nil)))
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

func TestPixelService_TestConnection(t *testing.T) {
	tests := []struct {
		name       string
		customerID string
		pixelID    string
		setup      func(*MockPixelRepo)
		fbHandler  http.HandlerFunc
		wantErr    error
		wantErrMsg string
	}{
		{
			name:       "success",
			customerID: "cust-1",
			pixelID:    "pixel-1",
			setup: func(pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID:            "pixel-1",
					CustomerID:    "cust-1",
					FBPixelID:     "123456",
					FBAccessToken: "valid-token",
				}, nil)
			},
			fbHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"events_received": 1,
					"fbtrace_id":      "test-trace-123",
				})
			},
		},
		{
			name:       "pixel not found",
			customerID: "cust-1",
			pixelID:    "nonexistent",
			setup: func(pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)
			},
			wantErr: ErrPixelNotFound,
		},
		{
			name:       "pixel not owned",
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
		{
			name:       "missing access token",
			customerID: "cust-1",
			pixelID:    "pixel-1",
			setup: func(pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID:            "pixel-1",
					CustomerID:    "cust-1",
					FBPixelID:     "123456",
					FBAccessToken: "",
				}, nil)
			},
			wantErr: ErrPixelNoAccessToken,
		},
		{
			name:       "capi error",
			customerID: "cust-1",
			pixelID:    "pixel-1",
			setup: func(pr *MockPixelRepo) {
				pr.On("GetByID", mock.Anything, "pixel-1").Return(&domain.Pixel{
					ID:            "pixel-1",
					CustomerID:    "cust-1",
					FBPixelID:     "123456",
					FBAccessToken: "invalid-token",
				}, nil)
			},
			fbHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Invalid OAuth access token"))
			},
			wantErrMsg: "test connection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pixelRepo := new(MockPixelRepo)
			tt.setup(pixelRepo)

			var capiClient *facebook.CAPIClient
			if tt.fbHandler != nil {
				server := httptest.NewServer(tt.fbHandler)
				defer server.Close()
				capiClient = facebook.NewCAPIClient(server.URL)
			} else {
				capiClient = facebook.NewCAPIClient("http://localhost:9999")
			}

			svc := NewPixelService(pixelRepo, capiClient, slog.New(slog.NewTextHandler(io.Discard, nil)))

			resp, err := svc.TestConnection(context.Background(), tt.customerID, tt.pixelID, "TestAgent/1.0")

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, resp)
			} else if tt.wantErrMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, 1, resp.EventsReceived)
			}
			pixelRepo.AssertExpectations(t)
		})
	}
}

func strPtr(s string) *string { return &s }
