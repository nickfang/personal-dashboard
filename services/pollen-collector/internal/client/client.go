package client

import (
	"context"
	"net/http"
)

// Fetcher defines the interface for fetching pollen data from an external API.
type Fetcher interface {
	Fetch(ctx context.Context, apiKey string, locationID string, lat, long float64) (*PollenAPIResponse, error)
}

// nonRetryable wraps errors that should not be retried (e.g. 401, 403, bad JSON).
type nonRetryable struct{ error }

// Client fetches pollen data from the Google Pollen API.
type Client struct {
	httpClient *http.Client
}

// New creates a new pollen API client.
func New(httpClient *http.Client) *Client {
	return &Client{httpClient: httpClient}
}

// Fetch retrieves pollen data for a given location.
// TODO: implement — HTTP call, retry logic, JSON decoding.
func (c *Client) Fetch(ctx context.Context, apiKey string, locationID string, lat, long float64) (*PollenAPIResponse, error) {
	return nil, nil
}
