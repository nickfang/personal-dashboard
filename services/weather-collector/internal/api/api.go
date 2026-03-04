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

// Fetcher defines the interface for fetching weather data from an external API.
type Fetcher interface {
	Fetch(apiKey string, location shared.Location) (*WeatherAPIResponse, error)
}

// nonRetryable wraps errors that should not be retried (e.g. 401, 403, bad JSON).
type nonRetryable struct{ error }

// Client fetches weather data from the Google Weather API.
type Client struct {
	httpClient *http.Client
}

// New creates a new weather API .
func New(httpApi *http.Client) *Client {
	return &Client{httpClient: httpApi}
}

func (c *Client) fetchWeather(apiKey string, location shared.Location) (*WeatherAPIResponse, error) {
	baseUrl := "https://weather.googleapis.com/v1/currentConditions:lookup"
	queryParams := url.Values{
		"location.latitude":  {fmt.Sprintf("%f", location.Lat)},
		"location.longitude": {fmt.Sprintf("%f", location.Long)},
		// "unitsSystem":        {"imperial"},
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
		err := fmt.Errorf("API request failed with status: %s", resp.Status)
		if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
			return nil, &nonRetryable{err}
		}
		return nil, err
	}
	var data WeatherAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, &nonRetryable{fmt.Errorf("failed to decode JSON: %w", err)}
	}
	return &data, nil
}

// Fetch retrieves current weather conditions for a given location.
func (c *Client) Fetch(apiKey string, location shared.Location) (*WeatherAPIResponse, error) {
	var lastErr error
	backoffs := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}

	for i := 0; i <= len(backoffs); i++ {
		wp, err := c.fetchWeather(apiKey, location)
		if err == nil {
			return wp, nil
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
