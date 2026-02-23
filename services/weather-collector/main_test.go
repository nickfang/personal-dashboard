package main

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/nickfang/personal-dashboard/services/shared"
)

// roundTripFunc adapts a plain function into an http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// validWeatherJSON is a minimal valid response from the Google Weather API.
// AirPressure must be non-zero or mapToWeatherPoint rejects it.
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

func TestFetchWeather_APIKeyInHeaderNotURL(t *testing.T) {
	const testAPIKey = "test-secret-key-12345"
	var capturedReq *http.Request

	origClient := httpClient
	httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			capturedReq = req.Clone(req.Context())
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(validWeatherJSON)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	defer func() { httpClient = origClient }()

	loc := shared.Location{ID: "test-loc", Lat: 30.0, Long: -97.0}
	_, err := fetchWeather(testAPIKey, loc)
	if err != nil {
		t.Fatalf("fetchWeather() returned error: %v", err)
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

func TestFetchWeather_ErrorDoesNotLeakAPIKey(t *testing.T) {
	const testAPIKey = "test-secret-key-12345"

	origClient := httpClient
	httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Status:     "403 Forbidden",
				Body:       io.NopCloser(strings.NewReader(`{"error":"forbidden"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	defer func() { httpClient = origClient }()

	loc := shared.Location{ID: "test-loc", Lat: 30.0, Long: -97.0}
	_, err := fetchWeather(testAPIKey, loc)
	if err == nil {
		t.Fatal("fetchWeather() should return error for 403 status")
	}

	if strings.Contains(err.Error(), testAPIKey) {
		t.Errorf("error message leaks API key: %s", err.Error())
	}
}

func TestFetchWeather_NonRetryableStatusCodes(t *testing.T) {
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
			origClient := httpClient
			httpClient = &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tt.statusCode,
						Status:     tt.status,
						Body:       io.NopCloser(strings.NewReader(`{}`)),
						Header:     make(http.Header),
					}, nil
				}),
			}
			defer func() { httpClient = origClient }()

			loc := shared.Location{ID: "test-loc", Lat: 30.0, Long: -97.0}
			_, err := fetchWeather("fake-key", loc)
			if err == nil {
				t.Fatal("fetchWeather() should return error for non-OK status")
			}

			var nr *nonRetryable
			if !errors.As(err, &nr) {
				t.Errorf("expected nonRetryable error for %d, got: %v", tt.statusCode, err)
			}
		})
	}
}

func TestCtoF(t *testing.T) {
	tests := []struct {
		name     string
		celsius  float64
		expected float64
	}{
		{"Freezing", 0.0, 32.0},
		{"Boiling", 100.0, 212.0},
		{"Negative", -40.0, -40.0},
		{"Room Temp", 20.0, 68.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CtoF(tt.celsius); got != tt.expected {
				t.Errorf("CtoF(%f) = %f, want %f", tt.celsius, got, tt.expected)
			}
		})
	}
}

func TestKtoM(t *testing.T) {
	tests := []struct {
		name     string
		kph      float64
		expected float64
	}{
		{"Zero", 0.0, 0.0},
		{"100 kph", 100.0, 62.1371},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := KtoM(tt.kph)
			// Allow for small float point errors
			diff := got - tt.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.0001 {
				t.Errorf("KtoM(%f) = %f, want %f", tt.kph, got, tt.expected)
			}
		})
	}
}

func TestCalculatePressureStats(t *testing.T) {
	now := time.Now()

	// Helper to create a point at T minus hours
	mkPoint := func(hoursAgo int, pressure float64) PressurePoint {
		return PressurePoint{
			TimeStamp:  now.Add(time.Duration(-hoursAgo) * time.Hour),
			PressureMb: pressure,
		}
	}

	tests := []struct {
		name         string
		history      []PressurePoint
		wantTrend    string
		wantDelta3h  *float64
		wantDelta24h *float64
	}{
		{
			name:      "Empty History",
			history:   []PressurePoint{},
			wantTrend: "unknown",
		},
		{
			name:      "Single Point",
			history:   []PressurePoint{mkPoint(0, 1013.0)},
			wantTrend: "unknown",
		},
		{
			name: "Stable Pressure",
			history: []PressurePoint{
				mkPoint(3, 1013.0),
				mkPoint(2, 1013.1),
				mkPoint(1, 1013.0),
				mkPoint(0, 1013.2), // Current
			},
			wantTrend:   "stable",
			wantDelta3h: floatPtr(0.2), // 1013.2 - 1013.0
		},
		{
			name: "Rising Pressure",
			history: []PressurePoint{
				mkPoint(4, 1010.0),
				mkPoint(3, 1011.0), // 3h ago
				mkPoint(2, 1012.0),
				mkPoint(1, 1013.0),
				mkPoint(0, 1014.0), // Current (1014 - 1011 = 3.0 increase)
			},
			wantTrend:   "rising",
			wantDelta3h: floatPtr(3.0),
		},
		{
			name: "Falling Pressure",
			history: []PressurePoint{
				mkPoint(3, 1020.0),
				mkPoint(2, 1019.0),
				mkPoint(0, 1018.0),
			},
			wantTrend:   "falling",
			wantDelta3h: floatPtr(-2.0), // 1018 - 1020
		},
		{
			name: "Long History with Gap",
			history: []PressurePoint{
				mkPoint(24, 1000.0),
				mkPoint(12, 1005.0),
				mkPoint(0, 1010.0),
			},
			wantTrend:    "unknown",      // 3h is missing in this test data
			wantDelta24h: floatPtr(10.0), // Now matches because of timestamp logic!
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := calculatePressureStats(tt.history)

			if stats.Trend != tt.wantTrend {
				t.Errorf("Trend = %s, want %s", stats.Trend, tt.wantTrend)
			}

			if !compareFloatPtr(stats.Delta3h, tt.wantDelta3h) {
				t.Errorf("Delta3h = %v, want %v", stats.Delta3h, tt.wantDelta3h)
			}

			if !compareFloatPtr(stats.Delta24h, tt.wantDelta24h) {
				t.Errorf("Delta24h = %v, want %v", stats.Delta24h, tt.wantDelta24h)
			}
		})
	}
}

func floatPtr(f float64) *float64 {
	return &f
}

func compareFloatPtr(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return abs(*a-*b) < 0.001
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
