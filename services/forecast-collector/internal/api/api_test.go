package api

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/nickfang/personal-dashboard/services/shared"
)

// roundTripFunc adapts a plain function into an http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("reading fixture %s: %v", name, err)
	}
	return data
}

func jsonResponse(body []byte) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     make(http.Header),
	}
}

var testLocation = shared.Location{ID: "test-loc", Lat: 30.0, Long: -97.0}

func TestFetch_PaginatesAndParsesFixture(t *testing.T) {
	var requests []*http.Request
	pages := [][]byte{fixture(t, "forecast_hours.json"), fixture(t, "forecast_hours_page2.json")}

	c := New(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requests = append(requests, req.Clone(req.Context()))
			page := pages[0]
			pages = pages[1:]
			return jsonResponse(page), nil
		}),
	})

	hours, err := c.Fetch("test-key", testLocation, 72)
	if err != nil {
		t.Fatalf("Fetch() returned error: %v", err)
	}

	if len(requests) != 2 {
		t.Fatalf("expected 2 requests (two pages), got %d", len(requests))
	}
	if got := requests[0].URL.Query().Get("pageToken"); got != "" {
		t.Errorf("first request pageToken = %q, want empty", got)
	}
	if got := requests[1].URL.Query().Get("pageToken"); got != "test-page-2-token" {
		t.Errorf("second request pageToken = %q, want %q", got, "test-page-2-token")
	}
	if got := requests[0].URL.Query().Get("hours"); got != "72" {
		t.Errorf("hours param = %q, want %q", got, "72")
	}

	if len(hours) != 6 {
		t.Fatalf("expected 6 hours across both pages, got %d", len(hours))
	}
	first := hours[0]
	if got := first.Interval.StartTime.Format("2006-01-02T15:04:05Z"); got != "2026-06-12T05:00:00Z" {
		t.Errorf("first hour startTime = %q, want 2026-06-12T05:00:00Z", got)
	}
	if first.AirPressure.MeanSeaLevelMillibars != 1012.65 {
		t.Errorf("first hour pressure = %v, want 1012.65", first.AirPressure.MeanSeaLevelMillibars)
	}
	if first.Temperature.Degrees != 27.5 {
		t.Errorf("first hour temp = %v, want 27.5", first.Temperature.Degrees)
	}
	if first.Precipitation.Probability.Percent != 5 {
		t.Errorf("first hour precipitation pct = %v, want 5", first.Precipitation.Probability.Percent)
	}
	if first.RelativeHumidityPercent != 82 {
		t.Errorf("first hour humidity = %v, want 82", first.RelativeHumidityPercent)
	}
	// Last hour comes from page 2, proving concatenation.
	last := hours[len(hours)-1]
	if last.AirPressure.MeanSeaLevelMillibars != 1013.35 {
		t.Errorf("last hour pressure = %v, want 1013.35", last.AirPressure.MeanSeaLevelMillibars)
	}
}

func TestFetch_StopsWhenHorizonCovered(t *testing.T) {
	requestCount := 0
	c := New(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requestCount++
			return jsonResponse(fixture(t, "forecast_hours.json")), nil
		}),
	})

	// Fixture has 3 hours and always returns a nextPageToken; a horizon of 3
	// must stop after the first page rather than looping forever.
	hours, err := c.Fetch("test-key", testLocation, 3)
	if err != nil {
		t.Fatalf("Fetch() returned error: %v", err)
	}
	if requestCount != 1 {
		t.Errorf("expected 1 request when horizon covered by first page, got %d", requestCount)
	}
	if len(hours) != 3 {
		t.Errorf("expected 3 hours, got %d", len(hours))
	}
}

func TestFetch_APIKeyInHeaderNotURL(t *testing.T) {
	const testAPIKey = "test-secret-key-12345"
	var capturedReq *http.Request

	c := New(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			capturedReq = req.Clone(req.Context())
			return jsonResponse(fixture(t, "forecast_hours_page2.json")), nil
		}),
	})

	_, err := c.Fetch(testAPIKey, testLocation, 72)
	if err != nil {
		t.Fatalf("Fetch() returned error: %v", err)
	}

	if capturedReq == nil {
		t.Fatal("expected HTTP request to be made, but none was captured")
	}
	if got := capturedReq.Header.Get("X-Goog-Api-Key"); got != testAPIKey {
		t.Errorf("X-Goog-Api-Key header = %q, want %q", got, testAPIKey)
	}
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

	_, err := c.Fetch(testAPIKey, testLocation, 72)
	if err == nil {
		t.Fatal("Fetch() should return error for 403 status")
	}
	if strings.Contains(err.Error(), testAPIKey) {
		t.Errorf("error message leaks API key: %s", err.Error())
	}
}

func TestFetchPage_NonRetryableStatusCodes(t *testing.T) {
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

			_, err := c.fetchPage("fake-key", testLocation, 72, "")
			if err == nil {
				t.Fatal("fetchPage() should return error for non-OK status")
			}
			var nr *nonRetryable
			if !errors.As(err, &nr) {
				t.Errorf("expected nonRetryable error for %d, got: %v", tt.statusCode, err)
			}
		})
	}
}

func TestFetchPage_ServerErrorIsRetryable(t *testing.T) {
	c := New(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Status:     "500 Internal Server Error",
				Body:       io.NopCloser(strings.NewReader(`{}`)),
				Header:     make(http.Header),
			}, nil
		}),
	})

	_, err := c.fetchPage("fake-key", testLocation, 72, "")
	if err == nil {
		t.Fatal("fetchPage() should return error for 500 status")
	}
	var nr *nonRetryable
	if errors.As(err, &nr) {
		t.Errorf("500 must be retryable, got nonRetryable: %v", err)
	}
}

func TestFetchPage_MalformedJSONIsNonRetryable(t *testing.T) {
	c := New(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return jsonResponse([]byte(`{not json`)), nil
		}),
	})

	_, err := c.fetchPage("fake-key", testLocation, 72, "")
	if err == nil {
		t.Fatal("fetchPage() should return error for malformed JSON")
	}
	var nr *nonRetryable
	if !errors.As(err, &nr) {
		t.Errorf("expected nonRetryable error for malformed JSON, got: %v", err)
	}
}
