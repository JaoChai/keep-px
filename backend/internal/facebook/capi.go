package facebook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type CAPIClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewCAPIClient(baseURL string) *CAPIClient {
	return &CAPIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type CAPIEvent struct {
	EventName             string                 `json:"event_name"`
	EventTime             int64                  `json:"event_time"`
	UserData              map[string]interface{} `json:"user_data,omitempty"`
	CustomData            map[string]interface{} `json:"custom_data,omitempty"`
	EventSourceURL        string                 `json:"event_source_url,omitempty"`
	ActionSource          string                 `json:"action_source"`
	EventID               string                 `json:"event_id,omitempty"`
	DataProcessingOptions []string               `json:"data_processing_options"`
}

type CAPIRequest struct {
	Data          []CAPIEvent `json:"data"`
	AccessToken   string      `json:"access_token,omitempty"`
	TestEventCode string      `json:"test_event_code,omitempty"`
}

type CAPIResponse struct {
	EventsReceived int      `json:"events_received"`
	Messages       []string `json:"messages,omitempty"`
	FBTraceID      string   `json:"fbtrace_id,omitempty"`
}

type CAPIError struct {
	StatusCode int
	Message    string
}

func (e *CAPIError) Error() string {
	return fmt.Sprintf("facebook CAPI error (status %d): %s", e.StatusCode, e.Message)
}

func (c *CAPIClient) SendEvents(ctx context.Context, pixelID, accessToken, testEventCode string, events []CAPIEvent) (*CAPIResponse, error) {
	url := fmt.Sprintf("%s/%s/events", c.baseURL, pixelID)

	reqBody := CAPIRequest{
		Data:          events,
		AccessToken:   accessToken,
		TestEventCode: testEventCode,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &CAPIError{
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
		}
	}

	var capiResp CAPIResponse
	if err := json.Unmarshal(respBody, &capiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &capiResp, nil
}

// IsAuthError checks if the error is a CAPIError with a 401 or 403 status code.
func IsAuthError(err error) bool {
	var capiErr *CAPIError
	if errors.As(err, &capiErr) {
		return capiErr.StatusCode == 401 || capiErr.StatusCode == 403
	}
	return false
}

// IsRateLimitError checks if the error is a CAPIError with a 429 status code.
func IsRateLimitError(err error) bool {
	var capiErr *CAPIError
	if errors.As(err, &capiErr) {
		return capiErr.StatusCode == 429
	}
	return false
}

func (c *CAPIClient) SendEvent(ctx context.Context, pixelID, accessToken, testEventCode string, event CAPIEvent) (*CAPIResponse, error) {
	return c.SendEvents(ctx, pixelID, accessToken, testEventCode, []CAPIEvent{event})
}

func (c *CAPIClient) SendEventsBatch(ctx context.Context, pixelID, accessToken, testEventCode string, events []CAPIEvent) ([]*CAPIResponse, error) {
	const maxBatchSize = 1000
	var responses []*CAPIResponse

	for i := 0; i < len(events); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(events) {
			end = len(events)
		}
		batch := events[i:end]

		resp, err := c.SendEvents(ctx, pixelID, accessToken, testEventCode, batch)
		if err != nil {
			return responses, fmt.Errorf("batch %d-%d: %w", i, end, err)
		}
		responses = append(responses, resp)
	}
	return responses, nil
}
