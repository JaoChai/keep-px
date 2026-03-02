package facebook

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// isRetryableError returns true for errors that should be retried (429 rate limit, 5xx server errors).
func isRetryableError(err error) bool {
	if IsRateLimitError(err) {
		return true
	}
	var capiErr *CAPIError
	if errors.As(err, &capiErr) {
		return capiErr.StatusCode >= 500
	}
	return false
}

// SendEventWithRetry sends a single event with exponential backoff retry.
// It fails fast on auth errors (401/403) and retries on 429/5xx.
func SendEventWithRetry(ctx context.Context, client *CAPIClient, pixelID, accessToken, testCode string, event CAPIEvent, maxRetries int, logger *slog.Logger) (*CAPIResponse, error) {
	return SendEventsWithRetry(ctx, client, pixelID, accessToken, testCode, []CAPIEvent{event}, maxRetries, logger)
}

// SendEventsWithRetry sends a batch of events with exponential backoff retry.
// It fails fast on auth errors (401/403) and retries on 429/5xx.
func SendEventsWithRetry(ctx context.Context, client *CAPIClient, pixelID, accessToken, testCode string, events []CAPIEvent, maxRetries int, logger *slog.Logger) (*CAPIResponse, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err := client.SendEvents(ctx, pixelID, accessToken, testCode, events)
		if err == nil {
			return resp, nil
		}
		lastErr = err

		// Fail fast on auth errors — token is invalid, retrying won't help
		if IsAuthError(err) {
			return nil, err
		}

		// Only retry on retryable errors
		if !isRetryableError(err) || attempt >= maxRetries {
			break
		}

		backoff := time.Duration(1<<uint(attempt)) * time.Second // 1s, 2s, 4s
		logger.Warn("CAPI request failed, retrying",
			"attempt", attempt+1,
			"max_retries", maxRetries,
			"backoff", backoff,
			"error", err,
		)
		timer := time.NewTimer(backoff)
		select {
		case <-timer.C:
			continue
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		}
	}
	return nil, lastErr
}
