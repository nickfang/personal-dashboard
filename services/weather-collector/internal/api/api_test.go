package api

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/nickfang/personal-dashboard/services/shared"
)

// roundTripFunc adapts a plain function into an http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// validWeatherJSON is a minimal valid response from the Google Weather API.
var validWeatherJSON = `{
	"temperature": {"degrees": 25.0},
	"feelsLikeTemperature": {"degrees": 27.0},
	"relativeHumidity": 60,
	"uvIndex": 5,
	"airPressure": {"meanSeaLevelMillibars": 1013.25},
	"wind": {
		"direction": {"degrees": 180},
		"speed": {"value": 15.0},
		"gust": {"value": 25.0}
	},
	"visibility": {"distance": 10.0},
	"dewPoint": {"degrees": 16.0},
	"precipitation": {"probability": {"probability": 20, "type": "RAIN"}}
}`

func TestFetch_APIKeyInHeaderNotURL(t *testing.T) {
	const testAPIKey = "test-secret-key-12345"
	var capturedReq *http.Request

	c := New(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			capturedReq = req.Clone(req.Context())
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(validWeatherJSON)),
				Header:     make(http.Header),
			}, nil
		}),
	})

	_, err := c.Fetch(testAPIKey, shared.Location{ID: "test-loc", Lat: 30.0, Long: -97.0})
	if err != nil {
		t.Fatalf("Fetch() returned error: %v", err)
	}

	if capturedReq == nil {
		t.Fatal("expected HTTP request to be made, but none was captured")
	}

	// The API key MUST be sent via the X-Goog-Api-Key header.
	if got := capturedReq.Header.Get("X-Goog-Api-Key"); got != testAPIKey {
		t.Errorf("X-Goog-Api-Key header = %q, want %q", got, testAPIKey)
	}

	// The API key MUST NOT appear anywhere in the URL.
	if capturedReq.URL.Query().Get("key") != "" {
		t.Error("API key found in URL 'key' query param; must use header instead")
	}
	if strings.Contains(capturedReq.URL.String(), testAPIKey) {
		t.Error("API key found in URL string; must use header instead")
	}
}

func TestFetch_ErrorDoesNotLeakAPIKey(t *testing.T) {
	const testAPIKey = "test-secret-key-12345"

	c := New(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Status:     "403 Forbidden",
				Body:       io.NopCloser(strings.NewReader(`{"error":"forbidden"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	})

	_, err := c.Fetch(testAPIKey, shared.Location{ID: "test-loc", Lat: 30.0, Long: -97.0})
	if err == nil {
		t.Fatal("Fetch() should return error for 403 status")
	}

	if strings.Contains(err.Error(), testAPIKey) {
		t.Errorf("error message leaks API key: %s", err.Error())
	}
}

func TestFetch_NonRetryableStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		status     string
	}{
		{"Unauthorized", http.StatusUnauthorized, "401 Unauthorized"},
		{"Forbidden", http.StatusForbidden, "403 Forbidden"},
		{"NotFound", http.StatusNotFound, "404 Not Found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(&http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tt.statusCode,
						Status:     tt.status,
						Body:       io.NopCloser(strings.NewReader(`{}`)),
						Header:     make(http.Header),
					}, nil
				}),
			})

			_, err := c.Fetch("fake-key", shared.Location{ID: "test-loc", Lat: 30.0, Long: -97.0})
			if err == nil {
				t.Fatal("Fetch() should return error for non-OK status")
			}

			// These status codes should be wrapped as nonRetryable
			var nr *nonRetryable
			if !errors.As(err, &nr) {
				t.Errorf("expected nonRetryable error for %d, got: %v", tt.statusCode, err)
			}
		})
	}
}
