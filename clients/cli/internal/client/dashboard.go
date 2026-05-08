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

// DefaultTimeout is applied to a request when the caller's context carries
// no deadline of its own. Callers who need a different per-request limit
// can pass their own context (via context.WithTimeout / WithDeadline) and
// it will take precedence — fetch() only adds the default when the
// incoming context is unbounded.
//
// We deliberately do not also set http.Client.Timeout: that is a hard
// ceiling rather than a default, and it would silently override any
// per-request value larger than itself, defeating the override semantics.
const DefaultTimeout = 10 * time.Second

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
		http:    &http.Client{},
	}, nil
}

// Fetch calls GET {baseURL}/v1/dashboard and parses the JSON response.
func (c *Client) Fetch(ctx context.Context) (*Response, error) {
	return c.fetch(ctx, "/v1/dashboard", "fetch dashboard")
}

// FetchLocation calls GET {baseURL}/v1/dashboard/{locationID} and parses the
// JSON response. The response shape matches Fetch(): three maps each
// containing a single entry keyed by locationID.
func (c *Client) FetchLocation(ctx context.Context, locationID string) (*Response, error) {
	if locationID == "" {
		return nil, fmt.Errorf("fetch location: empty locationID")
	}
	path := "/v1/dashboard/" + url.PathEscape(locationID)
	return c.fetch(ctx, path, "fetch location")
}

func (c *Client) fetch(ctx context.Context, path, errLabel string) (*Response, error) {
	// Apply the default timeout only when the caller hasn't supplied one;
	// any caller-provided deadline takes precedence and is left untouched.
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: build request: %w", errLabel, err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errLabel, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("%s: unexpected status %d: %s", errLabel, resp.StatusCode, string(body))
	}

	var out Response
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("%s: decode: %w", errLabel, err)
	}
	return &out, nil
}
