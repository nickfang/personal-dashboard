package repository

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/nickfang/personal-dashboard/services/shared"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Store reads the cache documents this job observes. The interface exposes no
// writes: this job records what it sees and delivers nothing (see #79, #80).
//
// That is a code-level guarantee, not an IAM one. The shared cloud-run-job
// module grants every job service account project-wide roles/datastore.user,
// which includes write, so nothing at the infrastructure layer stops this job
// from modifying documents owned by the collectors. Narrowing it to
// roles/datastore.viewer needs a variable on that module.
type Store interface {
	// ReadObservation returns the latest observed conditions, or nil when
	// the location has no cache document yet. A missing observation is a
	// recordable state, not an error — weather-collector may simply not have
	// run for this location.
	ReadObservation(ctx context.Context, locationID string) (*WeatherCacheDoc, error)

	// ReadForecast returns the latest forecast. A missing forecast means
	// there is nothing to observe against, so it is an error.
	ReadForecast(ctx context.Context, locationID string) (*ForecastCacheDoc, error)
}

type FirestoreStore struct {
	client *firestore.Client
}

// NewFirestoreStore connects to the weather database, which holds both
// weather_cache and forecast_cache — one client serves both reads.
func NewFirestoreStore(ctx context.Context, projectID string) (*FirestoreStore, error) {
	client, err := firestore.NewClientWithDatabase(ctx, projectID, shared.WeatherDatabaseID)
	if err != nil {
		return nil, err
	}
	return &FirestoreStore{client: client}, nil
}

func (s *FirestoreStore) Close() error {
	return s.client.Close()
}

func (s *FirestoreStore) ReadObservation(ctx context.Context, locationID string) (*WeatherCacheDoc, error) {
	doc, err := s.client.Collection(shared.WeatherCacheCollection).Doc(locationID).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading weather cache for %s: %w", locationID, err)
	}
	var out WeatherCacheDoc
	if err := doc.DataTo(&out); err != nil {
		return nil, fmt.Errorf("decoding weather cache for %s: %w", locationID, err)
	}
	return &out, nil
}

func (s *FirestoreStore) ReadForecast(ctx context.Context, locationID string) (*ForecastCacheDoc, error) {
	doc, err := s.client.Collection(shared.ForecastCacheCollection).Doc(locationID).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, fmt.Errorf("no forecast cache document for %s", locationID)
	}
	if err != nil {
		return nil, fmt.Errorf("reading forecast cache for %s: %w", locationID, err)
	}
	var out ForecastCacheDoc
	if err := doc.DataTo(&out); err != nil {
		return nil, fmt.Errorf("decoding forecast cache for %s: %w", locationID, err)
	}
	return &out, nil
}
