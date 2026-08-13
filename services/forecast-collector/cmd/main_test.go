package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/api"
	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/service"
	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/testutil"
	"github.com/nickfang/personal-dashboard/services/shared"
	"github.com/nickfang/personal-dashboard/services/shared/notify"
)

func happyHour() api.ForecastHour {
	var h api.ForecastHour
	h.Interval.StartTime = time.Now().Truncate(time.Hour)
	h.AirPressure.MeanSeaLevelMillibars = 1013.25
	h.Temperature.Degrees = 25.0
	return h
}

func happyFetcher() *testutil.MockFetcher {
	return &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location, horizonHours int) ([]api.ForecastHour, error) {
			return []api.ForecastHour{happyHour()}, nil
		},
	}
}

func happyWriter() *testutil.MockWriter {
	return &testutil.MockWriter{}
}

func failingFetcher() *testutil.MockFetcher {
	return &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location, horizonHours int) ([]api.ForecastHour, error) {
			return nil, fmt.Errorf("API unavailable")
		},
	}
}

func testConfig() service.Config {
	return service.Config{HorizonHours: 72, Alert: service.DefaultAlertConfig()}
}

var testLocations = []shared.Location{
	{ID: "house-nick", Lat: 30.0, Long: -97.0},
	{ID: "house-nita", Lat: 31.0, Long: -98.0},
}

func TestCollectAll_AllLocationsSucceed(t *testing.T) {
	collector := service.NewCollectorService(happyFetcher(), happyWriter(), notify.NopSender{}, testConfig())

	err := collectAll(context.Background(), "test-key", collector, testLocations)
	if err != nil {
		t.Fatalf("collectAll() returned error: %v", err)
	}
}

func TestCollectAll_PartialFailure(t *testing.T) {
	callCount := 0
	fetcher := &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location, horizonHours int) ([]api.ForecastHour, error) {
			callCount++
			if callCount == 1 {
				return nil, fmt.Errorf("API unavailable")
			}
			return []api.ForecastHour{happyHour()}, nil
		},
	}

	collector := service.NewCollectorService(fetcher, happyWriter(), notify.NopSender{}, testConfig())

	err := collectAll(context.Background(), "test-key", collector, testLocations)
	if err != nil {
		t.Fatalf("collectAll() should succeed with partial failures, got: %v", err)
	}
}

func TestCollectAll_AllLocationsFail(t *testing.T) {
	collector := service.NewCollectorService(failingFetcher(), happyWriter(), notify.NopSender{}, testConfig())

	err := collectAll(context.Background(), "test-key", collector, testLocations)
	if err == nil {
		t.Fatal("collectAll() should return error when all locations fail")
	}
}

func TestCollectAll_EmptyLocations(t *testing.T) {
	collector := service.NewCollectorService(happyFetcher(), happyWriter(), notify.NopSender{}, testConfig())

	err := collectAll(context.Background(), "test-key", collector, []shared.Location{})
	if err == nil {
		t.Fatal("collectAll() should return error when no locations are provided")
	}
}

func TestEnvFloat(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  float64
	}{
		{"unset", "", 5.0},
		{"valid", "7.5", 7.5},
		{"invalid", "abc", 5.0},
		{"negative", "-2", 5.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				t.Setenv("TEST_THRESHOLD", tt.value)
			}
			if got := envFloat("TEST_THRESHOLD", 5.0); got != tt.want {
				t.Errorf("envFloat(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestEnvInt(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int
	}{
		{"unset", "", 72},
		{"valid", "120", 120},
		{"invalid", "abc", 72},
		{"negative", "-5", 72},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				t.Setenv("TEST_HORIZON", tt.value)
			}
			if got := envInt("TEST_HORIZON", 72); got != tt.want {
				t.Errorf("envInt(%q) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}
}

func TestNewSender(t *testing.T) {
	const (
		user     = "me@gmail.com"
		password = "app-password"
		to       = "you@gmail.com"
	)
	tests := []struct {
		name                    string
		enabled, user, pass, to string
		wantSMTP                bool
	}{
		{name: "unset defaults to no delivery"},
		{name: "disabled", enabled: "false", user: user, pass: password, to: to},
		{name: "enabled and configured", enabled: "true", user: user, pass: password, to: to, wantSMTP: true},
		{name: "missing password", enabled: "true", user: user, to: to},
		{name: "missing user", enabled: "true", pass: password, to: to},
		{name: "missing recipient", enabled: "true", user: user, pass: password},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("NOTIFY_ENABLED", tt.enabled)
			t.Setenv("NOTIFY_SMTP_USER", tt.user)
			t.Setenv("NOTIFY_SMTP_PASSWORD", tt.pass)
			t.Setenv("NOTIFY_EMAIL_TO", tt.to)

			// Anything short of fully configured must degrade to a no-op
			// sender, not an error: the job still has to collect.
			_, isSMTP := newSender().(*notify.SMTPSender)
			if isSMTP != tt.wantSMTP {
				t.Errorf("newSender() SMTP = %v, want %v", isSMTP, tt.wantSMTP)
			}
		})
	}
}
