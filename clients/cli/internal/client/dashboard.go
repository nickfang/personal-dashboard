package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client fetches the aggregated dashboard payload from dashboard-api.
type Client struct {
	baseURL string
	http    *http.Client
}

// New returns a Client for the given dashboard-api base URL. The URL must
// parse cleanly and carry an http or https scheme plus a non-empty host,
// so misconfigurations fail fast at startup rather than on first fetch.
func New(baseURL string) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base URL %q: %w", baseURL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("base URL %q: scheme must be http or https", baseURL)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("base URL %q: missing host", baseURL)
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// Fetch calls GET {baseURL}/v1/dashboard and parses the JSON response.
func (c *Client) Fetch(ctx context.Context) (*Response, error) {
	url := c.baseURL + "/v1/dashboard"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch dashboard: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch dashboard: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("fetch dashboard: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var out Response
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("fetch dashboard: decode: %w", err)
	}
	return &out, nil
}
