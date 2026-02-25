package client

import (
	"context"
	"net/http"
)

// Fetcher defines the interface for fetching weather data from an external API.
type Fetcher interface {
	Fetch(ctx context.Context, apiKey string, locationID string, lat, long float64) (*WeatherAPIResponse, error)
}

// nonRetryable wraps errors that should not be retried (e.g. 401, 403, bad JSON).
type nonRetryable struct{ error }

// Client fetches weather data from the Google Weather API.
type Client struct {
	httpClient *http.Client
}

// New creates a new weather API client.
func New(httpClient *http.Client) *Client {
	return &Client{httpClient: httpClient}
}

// Fetch retrieves current weather conditions for a given location.
// TODO: implement — HTTP call, retry logic, JSON decoding.
func (c *Client) Fetch(ctx context.Context, apiKey string, locationID string, lat, long float64) (*WeatherAPIResponse, error) {
	return nil, nil
}
