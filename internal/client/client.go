package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

// Client wraps the HTTP client for the Hazor API.
type Client struct {
	Endpoint   string
	APIKey     string
	HTTPClient *http.Client
	MaxRetries int
}

// NewClient creates a new Hazor API client.
func NewClient(endpoint, apiKey string) *Client {
	return &Client{
		Endpoint: endpoint,
		APIKey:   apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		MaxRetries: 5,
	}
}

// doRequest executes an HTTP request with retry logic for 429 responses.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBytes)
	}

	url := fmt.Sprintf("%s%s", c.Endpoint, path)

	var lastErr error
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			select {
			case <-ctx.Done():
				return nil, 0, ctx.Err()
			case <-time.After(backoff):
			}

			// Re-create the reader for retries
			if body != nil {
				jsonBytes, _ := json.Marshal(body)
				reqBody = bytes.NewReader(jsonBytes)
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", fmt.Sprintf("ApiKey %s", c.APIKey))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("rate limited (429)")
			continue
		}

		return respBody, resp.StatusCode, nil
	}

	return nil, 0, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// Create sends a POST request to create a resource.
func (c *Client) Create(ctx context.Context, path string, body interface{}) (map[string]interface{}, error) {
	respBody, statusCode, err := c.doRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}

	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("API error (status %d): %s", statusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// Read sends a GET request to read a resource.
func (c *Client) Read(ctx context.Context, path string) (map[string]interface{}, error) {
	respBody, statusCode, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	if statusCode == http.StatusNotFound {
		return nil, nil
	}

	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("API error (status %d): %s", statusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// Update sends a PUT request to update a resource.
func (c *Client) Update(ctx context.Context, path string, body interface{}) (map[string]interface{}, error) {
	respBody, statusCode, err := c.doRequest(ctx, http.MethodPut, path, body)
	if err != nil {
		return nil, err
	}

	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("API error (status %d): %s", statusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// Delete sends a DELETE request to remove a resource.
func (c *Client) Delete(ctx context.Context, path string) error {
	respBody, statusCode, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}

	if statusCode < 200 || statusCode >= 300 {
		return fmt.Errorf("API error (status %d): %s", statusCode, string(respBody))
	}

	return nil
}

// WaitForStatus polls a resource until it reaches the desired status or a terminal error status.
func (c *Client) WaitForStatus(ctx context.Context, path string, desiredStatus string, timeout time.Duration) (map[string]interface{}, error) {
	deadline := time.Now().Add(timeout)
	pollInterval := 5 * time.Second

	for time.Now().Before(deadline) {
		result, err := c.Read(ctx, path)
		if err != nil {
			return nil, fmt.Errorf("error polling status: %w", err)
		}
		if result == nil {
			return nil, fmt.Errorf("resource not found while polling")
		}

		status, ok := result["status"].(string)
		if !ok {
			return nil, fmt.Errorf("status field not found or not a string")
		}

		if status == desiredStatus {
			return result, nil
		}

		// Check for terminal error states
		switch status {
		case "error", "failed", "deleted", "terminated":
			return result, fmt.Errorf("resource reached terminal status: %s", status)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	return nil, fmt.Errorf("timed out waiting for status %q after %v", desiredStatus, timeout)
}
