package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/nickfang/personal-dashboard/services/shared"
)

// Fetcher defines the interface for fetching pollen data from an external API.
type Fetcher interface {
	Fetch(apiKey string, location shared.Location) (*PollenAPIResponse, error)
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

func (c *Client) fetchPollen(apiKey string, location shared.Location) (*PollenAPIResponse, error) {
	baseUrl := "https://pollen.googleapis.com/v1/forecast:lookup"
	queryParams := url.Values{
		"location.latitude":  {fmt.Sprintf("%f", location.Lat)},
		"location.longitude": {fmt.Sprintf("%f", location.Long)},
		"days":               {"1"},
	}
	url := baseUrl + "?" + queryParams.Encode()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Goog-Api-Key", apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("pollen API returned status: %s", resp.Status)
		// Only retry on 429 (rate limit) and 5xx (server errors)
		if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
			return nil, &nonRetryable{err}
		}
		return nil, err
	}

	var data PollenAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, &nonRetryable{fmt.Errorf("failed to decode pollen JSON: %w", err)}
	}

	if len(data.DailyInfo) == 0 {
		return nil, &nonRetryable{fmt.Errorf("no daily info returned for %s", location.ID)}
	}

	return &data, nil
}

// Fetch retrieves pollen data for a given location.
func (c *Client) Fetch(apiKey string, location shared.Location) (*PollenAPIResponse, error) {
	var lastErr error
	backoffs := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}

	for i := 0; i <= len(backoffs); i++ {
		data, err := c.fetchPollen(apiKey, location)
		if err == nil {
			return data, nil
		}
		var nr *nonRetryable
		if errors.As(err, &nr) {
			return nil, err
		}
		lastErr = err
		if i < len(backoffs) {
			time.Sleep(backoffs[i])
		}
	}
	return nil, fmt.Errorf("exhausted retries: %w", lastErr)
}
