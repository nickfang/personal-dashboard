package testutil

import (
	"context"
	"time"

	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/api"
	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/repository"
	"github.com/nickfang/personal-dashboard/services/shared"
	"github.com/nickfang/personal-dashboard/services/shared/notify"
)

// MockFetcher implements api.Fetcher for testing.
type MockFetcher struct {
	FetchFn func(apiKey string, location shared.Location, horizonHours int) ([]api.ForecastHour, error)
}

func (m *MockFetcher) Fetch(apiKey string, location shared.Location, horizonHours int) ([]api.ForecastHour, error) {
	return m.FetchFn(apiKey, location, horizonHours)
}

// MockWriter implements repository.Writer for testing. Unset funcs behave as
// successful no-ops, so a test only has to supply the calls it cares about.
type MockWriter struct {
	SaveRawFn      func(ctx context.Context, run repository.ForecastRun) error
	UpdateCacheFn  func(ctx context.Context, locationID string, run repository.ForecastRun, merge repository.MergeFunc) ([]shared.Alert, error)
	MarkNotifiedFn func(ctx context.Context, locationID string, alertIDs []string, at time.Time) error
}

func (m *MockWriter) SaveRaw(ctx context.Context, run repository.ForecastRun) error {
	if m.SaveRawFn == nil {
		return nil
	}
	return m.SaveRawFn(ctx, run)
}

func (m *MockWriter) UpdateCache(ctx context.Context, locationID string, run repository.ForecastRun, merge repository.MergeFunc) ([]shared.Alert, error) {
	if m.UpdateCacheFn == nil {
		return nil, nil
	}
	return m.UpdateCacheFn(ctx, locationID, run, merge)
}

func (m *MockWriter) MarkNotified(ctx context.Context, locationID string, alertIDs []string, at time.Time) error {
	if m.MarkNotifiedFn == nil {
		return nil
	}
	return m.MarkNotifiedFn(ctx, locationID, alertIDs, at)
}

// MockSender implements notify.Sender for testing: it records what it was
// asked to deliver and can be made to fail.
type MockSender struct {
	Sent []notify.Notification
	Err  error
}

func (m *MockSender) Send(ctx context.Context, n notify.Notification) error {
	if m.Err != nil {
		return m.Err
	}
	m.Sent = append(m.Sent, n)
	return nil
}
