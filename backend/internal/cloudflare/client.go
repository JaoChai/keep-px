package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const baseURL = "https://api.cloudflare.com/client/v4"

// Client communicates with the Cloudflare API for custom hostname and KV management.
type Client struct {
	httpClient    *http.Client
	apiToken      string
	accountID     string
	zoneID        string
	kvNamespaceID string
}

// NewClient creates a new Cloudflare API client.
func NewClient(apiToken, accountID, zoneID, kvNamespaceID string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiToken:      apiToken,
		accountID:     accountID,
		zoneID:        zoneID,
		kvNamespaceID: kvNamespaceID,
	}
}

// CFError represents an error returned by the Cloudflare API.
type CFError struct {
	StatusCode int
	Message    string
}

func (e *CFError) Error() string {
	return fmt.Sprintf("cloudflare API error (status %d): %s", e.StatusCode, e.Message)
}

// CustomHostnameResponse represents a Cloudflare custom hostname.
type CustomHostnameResponse struct {
	ID                 string    `json:"id"`
	Hostname           string    `json:"hostname"`
	Status             string    `json:"status"`
	SSL                SSLStatus `json:"ssl"`
	VerificationErrors []string  `json:"verification_errors,omitempty"`
}

// SSLStatus represents the SSL status of a custom hostname.
type SSLStatus struct {
	Status string `json:"status"`
}

// cfAPIResponse is the standard Cloudflare API envelope.
type cfAPIResponse struct {
	Success bool            `json:"success"`
	Errors  []cfAPIError    `json:"errors"`
	Result  json.RawMessage `json:"result"`
}

// cfAPIError represents a single error from the Cloudflare API.
type cfAPIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// doRequest executes an HTTP request against the Cloudflare API and returns the raw
// response body. It handles authentication, content-type, status checking, and
// Cloudflare-level error parsing.
func (c *Client) doRequest(ctx context.Context, method, url string, body []byte, contentType string) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	if body != nil && contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &CFError{
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
		}
	}

	// For JSON responses, also check the Cloudflare success envelope.
	var apiResp cfAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err == nil && !apiResp.Success {
		msg := "unknown error"
		if len(apiResp.Errors) > 0 {
			msg = apiResp.Errors[0].Message
		}
		return nil, &CFError{
			StatusCode: resp.StatusCode,
			Message:    msg,
		}
	}

	return respBody, nil
}

// CreateCustomHostname registers a new custom hostname with Cloudflare.
func (c *Client) CreateCustomHostname(ctx context.Context, hostname string) (*CustomHostnameResponse, error) {
	url := fmt.Sprintf("%s/zones/%s/custom_hostnames", baseURL, c.zoneID)

	payload := map[string]interface{}{
		"hostname": hostname,
		"ssl": map[string]string{
			"method": "http",
			"type":   "dv",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	respBody, err := c.doRequest(ctx, http.MethodPost, url, body, "application/json")
	if err != nil {
		return nil, err
	}

	var apiResp cfAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	var result CustomHostnameResponse
	if err := json.Unmarshal(apiResp.Result, &result); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}

	return &result, nil
}

// GetCustomHostname retrieves the status of a custom hostname.
func (c *Client) GetCustomHostname(ctx context.Context, hostnameID string) (*CustomHostnameResponse, error) {
	url := fmt.Sprintf("%s/zones/%s/custom_hostnames/%s", baseURL, c.zoneID, hostnameID)

	respBody, err := c.doRequest(ctx, http.MethodGet, url, nil, "")
	if err != nil {
		return nil, err
	}

	var apiResp cfAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	var result CustomHostnameResponse
	if err := json.Unmarshal(apiResp.Result, &result); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}

	return &result, nil
}

// DeleteCustomHostname removes a custom hostname from Cloudflare.
func (c *Client) DeleteCustomHostname(ctx context.Context, hostnameID string) error {
	url := fmt.Sprintf("%s/zones/%s/custom_hostnames/%s", baseURL, c.zoneID, hostnameID)

	_, err := c.doRequest(ctx, http.MethodDelete, url, nil, "")
	return err
}

// PutKVValue stores a key-value pair in Cloudflare Workers KV.
func (c *Client) PutKVValue(ctx context.Context, key string, value []byte) error {
	url := fmt.Sprintf("%s/accounts/%s/storage/kv/namespaces/%s/values/%s",
		baseURL, c.accountID, c.kvNamespaceID, key)

	_, err := c.doRequest(ctx, http.MethodPut, url, value, "application/octet-stream")
	return err
}

// DeleteKVValue removes a key-value pair from Cloudflare Workers KV.
func (c *Client) DeleteKVValue(ctx context.Context, key string) error {
	url := fmt.Sprintf("%s/accounts/%s/storage/kv/namespaces/%s/values/%s",
		baseURL, c.accountID, c.kvNamespaceID, key)

	_, err := c.doRequest(ctx, http.MethodDelete, url, nil, "")
	return err
}
