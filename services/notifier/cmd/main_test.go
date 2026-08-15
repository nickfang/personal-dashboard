package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nickfang/personal-dashboard/services/notifier/internal/repository"
	"github.com/nickfang/personal-dashboard/services/notifier/internal/service"
	"github.com/nickfang/personal-dashboard/services/notifier/internal/testutil"
	"github.com/nickfang/personal-dashboard/services/shared"
)

var testLocations = []shared.Location{
	{ID: "house-nick", Lat: 30.0, Long: -97.0},
	{ID: "house-nita", Lat: 31.0, Long: -98.0},
}

func healthyStore() *testutil.MockStore {
	return &testutil.MockStore{
		ReadForecastFn: func(ctx context.Context, id string) (*repository.ForecastCacheDoc, error) {
			return &repository.ForecastCacheDoc{
				Location: id,
				IssuedAt: time.Now(),
				Points: []repository.ForecastPoint{
					{ValidTime: time.Now(), PressureMb: 1013},
				},
			}, nil
		},
	}
}

func TestObserveAll_AllLocationsSucceed(t *testing.T) {
	notifier := service.NewNotifierService(healthyStore())

	if err := observeAll(context.Background(), notifier, testLocations, time.Now()); err != nil {
		t.Fatalf("observeAll() returned error: %v", err)
	}
}

func TestObserveAll_PartialFailureContinues(t *testing.T) {
	calls := 0
	store := &testutil.MockStore{
		ReadForecastFn: func(ctx context.Context, id string) (*repository.ForecastCacheDoc, error) {
			calls++
			if calls == 1 {
				return nil, fmt.Errorf("firestore unavailable")
			}
			return &repository.ForecastCacheDoc{IssuedAt: time.Now()}, nil
		},
	}
	notifier := service.NewNotifierService(store)

	if err := observeAll(context.Background(), notifier, testLocations, time.Now()); err != nil {
		t.Fatalf("observeAll() should succeed with partial failures, got: %v", err)
	}
	if calls != 2 {
		t.Errorf("read %d locations, want 2 — a failure must not stop the loop", calls)
	}
}

func TestObserveAll_AllLocationsFail(t *testing.T) {
	store := &testutil.MockStore{
		ReadForecastFn: func(ctx context.Context, id string) (*repository.ForecastCacheDoc, error) {
			return nil, fmt.Errorf("firestore unavailable")
		},
	}
	notifier := service.NewNotifierService(store)

	if err := observeAll(context.Background(), notifier, testLocations, time.Now()); err == nil {
		t.Fatal("observeAll() should return an error when every location fails")
	}
}

func TestObserveAll_EmptyLocations(t *testing.T) {
	notifier := service.NewNotifierService(healthyStore())

	if err := observeAll(context.Background(), notifier, nil, time.Now()); err == nil {
		t.Fatal("observeAll() should return an error when no locations are provided")
	}
}

func TestObserveAll_SharesOneEvaluationTime(t *testing.T) {
	// Every location in a run must be evaluated against the same instant, so
	// a run straddling an hour boundary cannot split across two.
	store := &testutil.MockStore{
		ReadForecastFn: func(ctx context.Context, id string) (*repository.ForecastCacheDoc, error) {
			return &repository.ForecastCacheDoc{IssuedAt: time.Now().Add(-90 * time.Minute)}, nil
		},
	}
	notifier := service.NewNotifierService(store)

	now := time.Now()
	var seen []time.Time
	for _, loc := range testLocations {
		obs, err := notifier.Observe(context.Background(), loc, now)
		if err != nil {
			t.Fatalf("Observe(%s) returned error: %v", loc.ID, err)
		}
		seen = append(seen, obs.Now)
	}

	if len(seen) != 2 || !seen[0].Equal(seen[1]) || !seen[0].Equal(now) {
		t.Errorf("evaluation times = %v, want both equal to the pinned %v", seen, now)
	}
}
