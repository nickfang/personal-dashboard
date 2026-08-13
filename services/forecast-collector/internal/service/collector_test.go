package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/api"
	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/repository"
	"github.com/nickfang/personal-dashboard/services/forecast-collector/internal/testutil"
	"github.com/nickfang/personal-dashboard/services/shared"
	"github.com/nickfang/personal-dashboard/services/shared/notify"
)

var testLocation = shared.Location{ID: "test-loc", Lat: 30.0, Long: -97.0}

func testConfig() Config {
	return Config{HorizonHours: 72, Alert: DefaultAlertConfig()}
}

func happyHours() []api.ForecastHour {
	return []api.ForecastHour{validHour()}
}

func TestCollect_SavesRawThenCache(t *testing.T) {
	var savedRun *repository.ForecastRun
	var cachedLocation string
	var cachedRun *repository.ForecastRun

	fetcher := &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location, horizonHours int) ([]api.ForecastHour, error) {
			if horizonHours != 72 {
				t.Errorf("Fetch horizonHours = %d, want 72", horizonHours)
			}
			return happyHours(), nil
		},
	}
	writer := &testutil.MockWriter{
		SaveRawFn: func(ctx context.Context, run repository.ForecastRun) error {
			savedRun = &run
			return nil
		},
		UpdateCacheFn: func(ctx context.Context, locationID string, run repository.ForecastRun, merge repository.MergeFunc) ([]shared.Alert, error) {
			cachedLocation = locationID
			cachedRun = &run
			return nil, nil
		},
	}

	collector := NewCollectorService(fetcher, writer, &testutil.MockSender{}, testConfig())
	if err := collector.Collect(context.Background(), "test-key", testLocation); err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}

	if savedRun == nil {
		t.Fatal("SaveRaw was not called")
	}
	if savedRun.Location != "test-loc" {
		t.Errorf("saved run location = %q, want test-loc", savedRun.Location)
	}
	if len(savedRun.Points) != 1 {
		t.Fatalf("saved run has %d points, want 1", len(savedRun.Points))
	}
	if cachedLocation != "test-loc" {
		t.Errorf("UpdateCache locationID = %q, want test-loc", cachedLocation)
	}
	if cachedRun == nil || len(cachedRun.Points) != 1 {
		t.Fatal("UpdateCache did not receive the mapped run")
	}
}

func TestCollect_WiresDetectedAlertsIntoMerge(t *testing.T) {
	// Four hours with a 6 mb drop: detection must produce one alert, and the
	// merge func passed to UpdateCache must surface it against empty state.
	droppingHours := make([]api.ForecastHour, 4)
	for i, p := range []float64{1013, 1011, 1009, 1007} {
		h := validHour()
		h.Interval.StartTime = h.Interval.StartTime.Add(time.Duration(i) * time.Hour)
		h.AirPressure.MeanSeaLevelMillibars = p
		droppingHours[i] = h
	}

	var capturedMerge repository.MergeFunc
	fetcher := &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location, horizonHours int) ([]api.ForecastHour, error) {
			return droppingHours, nil
		},
	}
	writer := &testutil.MockWriter{
		SaveRawFn: func(ctx context.Context, run repository.ForecastRun) error {
			return nil
		},
		UpdateCacheFn: func(ctx context.Context, locationID string, run repository.ForecastRun, merge repository.MergeFunc) ([]shared.Alert, error) {
			capturedMerge = merge
			return nil, nil
		},
	}

	collector := NewCollectorService(fetcher, writer, &testutil.MockSender{}, testConfig())
	if err := collector.Collect(context.Background(), "test-key", testLocation); err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}

	if capturedMerge == nil {
		t.Fatal("UpdateCache did not receive a merge func")
	}
	alerts := capturedMerge(nil)
	if len(alerts) != 1 {
		t.Fatalf("merge(nil) produced %d alerts, want 1 detected drop", len(alerts))
	}
	if alerts[0].Location != "test-loc" || alerts[0].Status != shared.AlertStatusActive {
		t.Errorf("alert = %q/%q, want test-loc/active", alerts[0].Location, alerts[0].Status)
	}
	if alerts[0].Value != -6 {
		t.Errorf("Value = %v, want -6", alerts[0].Value)
	}
}

func TestCollect_FetchErrorPropagates(t *testing.T) {
	fetcher := &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location, horizonHours int) ([]api.ForecastHour, error) {
			return nil, fmt.Errorf("API unavailable")
		},
	}
	writer := &testutil.MockWriter{
		SaveRawFn: func(ctx context.Context, run repository.ForecastRun) error {
			t.Fatal("SaveRaw should not be called when fetch fails")
			return nil
		},
		UpdateCacheFn: func(ctx context.Context, locationID string, run repository.ForecastRun, merge repository.MergeFunc) ([]shared.Alert, error) {
			return nil, nil
		},
	}

	collector := NewCollectorService(fetcher, writer, &testutil.MockSender{}, testConfig())
	if err := collector.Collect(context.Background(), "test-key", testLocation); err == nil {
		t.Fatal("Collect() should propagate fetch errors")
	}
}

func TestCollect_AllPointsInvalidFails(t *testing.T) {
	bad := validHour()
	bad.AirPressure.MeanSeaLevelMillibars = 0
	fetcher := &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location, horizonHours int) ([]api.ForecastHour, error) {
			return []api.ForecastHour{bad}, nil
		},
	}
	writer := &testutil.MockWriter{
		SaveRawFn: func(ctx context.Context, run repository.ForecastRun) error {
			t.Fatal("SaveRaw should not be called when no points are valid")
			return nil
		},
		UpdateCacheFn: func(ctx context.Context, locationID string, run repository.ForecastRun, merge repository.MergeFunc) ([]shared.Alert, error) {
			return nil, nil
		},
	}

	collector := NewCollectorService(fetcher, writer, &testutil.MockSender{}, testConfig())
	if err := collector.Collect(context.Background(), "test-key", testLocation); err == nil {
		t.Fatal("Collect() should fail when every forecast hour is invalid")
	}
}

func TestCollect_SaveRawErrorPropagates(t *testing.T) {
	fetcher := &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location, horizonHours int) ([]api.ForecastHour, error) {
			return happyHours(), nil
		},
	}
	writer := &testutil.MockWriter{
		SaveRawFn: func(ctx context.Context, run repository.ForecastRun) error {
			return fmt.Errorf("firestore unavailable")
		},
		UpdateCacheFn: func(ctx context.Context, locationID string, run repository.ForecastRun, merge repository.MergeFunc) ([]shared.Alert, error) {
			t.Fatal("UpdateCache should not be called when SaveRaw fails")
			return nil, nil
		},
	}

	collector := NewCollectorService(fetcher, writer, &testutil.MockSender{}, testConfig())
	if err := collector.Collect(context.Background(), "test-key", testLocation); err == nil {
		t.Fatal("Collect() should propagate SaveRaw errors")
	}
}

// deliveryWriter returns a writer whose UpdateCache commits the given alert
// set and whose MarkNotified records the IDs it was handed.
func deliveryWriter(committed []shared.Alert, marked *[]string, markErr error) *testutil.MockWriter {
	return &testutil.MockWriter{
		UpdateCacheFn: func(ctx context.Context, locationID string, run repository.ForecastRun, merge repository.MergeFunc) ([]shared.Alert, error) {
			return committed, nil
		},
		MarkNotifiedFn: func(ctx context.Context, locationID string, alertIDs []string, at time.Time) error {
			*marked = append(*marked, alertIDs...)
			return markErr
		},
	}
}

func deliveredIDs(sent []notify.Notification) []string {
	ids := make([]string, len(sent))
	for i, n := range sent {
		ids[i] = n.Alert.ID
	}
	return ids
}

// happyFetcherFor returns a fetcher producing one valid, alert-free hour, for
// tests that care about the committed alert set rather than detection.
func happyFetcherFor(t *testing.T) *testutil.MockFetcher {
	t.Helper()
	return &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location, horizonHours int) ([]api.ForecastHour, error) {
			return happyHours(), nil
		},
	}
}

func TestCollect_DeliversOnlyUndeliveredActiveAlerts(t *testing.T) {
	past := time.Date(2026, 6, 12, 6, 0, 0, 0, time.UTC)
	committed := []shared.Alert{
		{ID: "undelivered", Status: shared.AlertStatusActive},
		{ID: "already-delivered", Status: shared.AlertStatusActive, NotifiedAt: past},
		{ID: "resolved", Status: shared.AlertStatusResolved, NotifiedAt: past},
		{ID: "resolved-undelivered", Status: shared.AlertStatusResolved},
	}
	var marked []string
	sender := &testutil.MockSender{}

	collector := NewCollectorService(happyFetcherFor(t), deliveryWriter(committed, &marked, nil), sender, testConfig())
	if err := collector.Collect(context.Background(), "test-key", testLocation); err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}

	if got := deliveredIDs(sender.Sent); len(got) != 1 || got[0] != "undelivered" {
		t.Errorf("delivered %v, want only [undelivered]", got)
	}
	if len(marked) != 1 || marked[0] != "undelivered" {
		t.Errorf("MarkNotified got %v, want [undelivered]", marked)
	}
}

func TestCollect_EscalationClearingNotifiedAtRedelivers(t *testing.T) {
	// Drive the real merge the way the repository does: a delivered warning
	// that worsens past the escalation step comes back with NotifiedAt
	// cleared, so the same single gate re-arms delivery.
	now := time.Now().Truncate(time.Hour)
	stored := shared.Alert{
		ID:          "episode-1",
		Location:    testLocation.ID,
		RuleID:      "pressure-drop-3h",
		Severity:    shared.AlertSeverityWarning,
		Value:       -5.0,
		WindowStart: now.Add(time.Hour),
		WindowEnd:   now.Add(4 * time.Hour),
		Status:      shared.AlertStatusActive,
		NotifiedAt:  now.Add(-6 * time.Hour),
	}

	var marked []string
	sender := &testutil.MockSender{}
	writer := &testutil.MockWriter{
		UpdateCacheFn: func(ctx context.Context, locationID string, run repository.ForecastRun, merge repository.MergeFunc) ([]shared.Alert, error) {
			return merge([]shared.Alert{stored}), nil
		},
		MarkNotifiedFn: func(ctx context.Context, locationID string, alertIDs []string, at time.Time) error {
			marked = append(marked, alertIDs...)
			return nil
		},
	}

	// Four hours dropping 6 mb — worse than the stored -5.0 by
	// AlertEscalationStepMb, over a window overlapping the stored one.
	droppingHours := make([]api.ForecastHour, 4)
	for i, p := range []float64{1013, 1011, 1009, 1007} {
		h := validHour()
		h.Interval.StartTime = now.Add(time.Duration(i+1) * time.Hour)
		h.AirPressure.MeanSeaLevelMillibars = p
		droppingHours[i] = h
	}
	fetcher := &testutil.MockFetcher{
		FetchFn: func(apiKey string, location shared.Location, horizonHours int) ([]api.ForecastHour, error) {
			return droppingHours, nil
		},
	}

	collector := NewCollectorService(fetcher, writer, sender, testConfig())
	if err := collector.Collect(context.Background(), "test-key", testLocation); err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}

	if got := deliveredIDs(sender.Sent); len(got) != 1 || got[0] != "episode-1" {
		t.Fatalf("delivered %v, want the escalated alert re-sent as [episode-1]", got)
	}
	if len(marked) != 1 || marked[0] != "episode-1" {
		t.Errorf("MarkNotified got %v, want [episode-1]", marked)
	}
}

func TestCollect_SendFailureDoesNotFailRun(t *testing.T) {
	committed := []shared.Alert{{ID: "undelivered", Status: shared.AlertStatusActive}}
	var marked []string
	sender := &testutil.MockSender{Err: fmt.Errorf("smtp unavailable")}

	collector := NewCollectorService(happyFetcherFor(t), deliveryWriter(committed, &marked, nil), sender, testConfig())
	if err := collector.Collect(context.Background(), "test-key", testLocation); err != nil {
		t.Fatalf("Collect() should not fail when delivery fails, got: %v", err)
	}
	if len(marked) != 0 {
		t.Errorf("MarkNotified got %v, want nothing marked when the send failed", marked)
	}
}

func TestCollect_MarkNotifiedFailureDoesNotFailRun(t *testing.T) {
	committed := []shared.Alert{{ID: "undelivered", Status: shared.AlertStatusActive}}
	var marked []string
	sender := &testutil.MockSender{}

	writer := deliveryWriter(committed, &marked, fmt.Errorf("firestore unavailable"))
	collector := NewCollectorService(happyFetcherFor(t), writer, sender, testConfig())
	if err := collector.Collect(context.Background(), "test-key", testLocation); err != nil {
		t.Fatalf("Collect() should not fail when marking fails, got: %v", err)
	}
	if len(sender.Sent) != 1 {
		t.Errorf("delivered %d alerts, want 1 — the alert re-delivers next run", len(sender.Sent))
	}
}
