package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockWebhookEventRepo is a testify mock for repository.WebhookEventRepo.
type MockWebhookEventRepo struct{ mock.Mock }

func (m *MockWebhookEventRepo) CreateIfNotExists(ctx context.Context, stripeEventID, eventType string) (bool, error) {
	args := m.Called(ctx, stripeEventID, eventType)
	return args.Bool(0), args.Error(1)
}
func (m *MockWebhookEventRepo) Delete(ctx context.Context, stripeEventID string) error {
	args := m.Called(ctx, stripeEventID)
	return args.Error(0)
}
