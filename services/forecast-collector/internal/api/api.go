package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/nickfang/personal-dashboard/services/shared"
)

const pageSize = 24

// Fetcher defines the interface for fetching forecast data from an external API.
type Fetcher interface {
	Fetch(ctx context.Context, apiKey string, location shared.Location, horizonHours int) ([]ForecastHour, error)
}

// nonRetryable wraps errors that should not be retried (e.g. 401, 403, bad JSON).
type nonRetryable struct{ error }

// Client fetches hourly forecasts from the Google Weather API.
type Client struct {
	httpClient *http.Client
}

// New creates a new forecast API client.
func New(httpApi *http.Client) *Client {
	return &Client{httpClient: httpApi}
}

func (c *Client) fetchPage(ctx context.Context, apiKey string, location shared.Location, horizonHours int, pageToken string) (*forecastHoursResponse, error) {
	baseUrl := "https://weather.googleapis.com/v1/forecast/hours:lookup"
	queryParams := url.Values{
		"location.latitude":  {fmt.Sprintf("%f", location.Lat)},
		"location.longitude": {fmt.Sprintf("%f", location.Long)},
		"hours":              {strconv.Itoa(horizonHours)},
		"pageSize":           {strconv.Itoa(pageSize)},
	}
	if pageToken != "" {
		queryParams.Set("pageToken", pageToken)
	}
	reqURL := baseUrl + "?" + queryParams.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
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
		err := fmt.Errorf("API request failed with status: %s", resp.Status)
		if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
			return nil, &nonRetryable{err}
		}
		return nil, err
	}
	var data forecastHoursResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, &nonRetryable{fmt.Errorf("failed to decode JSON: %w", err)}
	}
	return &data, nil
}

func (c *Client) fetchPageWithRetry(ctx context.Context, apiKey string, location shared.Location, horizonHours int, pageToken string) (*forecastHoursResponse, error) {
	var lastErr error
	backoffs := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}

	for i := 0; i <= len(backoffs); i++ {
		page, err := c.fetchPage(ctx, apiKey, location, horizonHours, pageToken)
		if err == nil {
			return page, nil
		}
		var nr *nonRetryable
		if errors.As(err, &nr) {
			return nil, err
		}
		lastErr = err
		if i < len(backoffs) {
			// Context-aware backoff: a cancelled job shouldn't sit out the sleep.
			select {
			case <-time.After(backoffs[i]):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
	return nil, fmt.Errorf("exhausted retries: %w", lastErr)
}

// Fetch retrieves the hourly forecast for a location, following pagination
// until the horizon is covered or no further pages exist.
func (c *Client) Fetch(ctx context.Context, apiKey string, location shared.Location, horizonHours int) ([]ForecastHour, error) {
	var hours []ForecastHour
	pageToken := ""
	for {
		page, err := c.fetchPageWithRetry(ctx, apiKey, location, horizonHours, pageToken)
		if err != nil {
			return nil, err
		}
		hours = append(hours, page.ForecastHours...)
		pageToken = page.NextPageToken
		if pageToken == "" || len(hours) >= horizonHours {
			break
		}
	}
	if len(hours) == 0 {
		return nil, fmt.Errorf("API returned no forecast hours for %s", location.ID)
	}
	return hours, nil
}
