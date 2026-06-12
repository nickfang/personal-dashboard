package repository

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/nickfang/personal-dashboard/services/shared"
)

// Writer defines the interface for writing forecast data to storage.
type Writer interface {
	SaveRaw(ctx context.Context, run ForecastRun) error
	UpdateCache(ctx context.Context, locationID string, run ForecastRun) error
}

type FirestoreWriter struct {
	client *firestore.Client
}

func NewFirestoreWriter(ctx context.Context, projectID string) (*FirestoreWriter, error) {
	client, err := firestore.NewClientWithDatabase(ctx, projectID, shared.WeatherDatabaseID)
	if err != nil {
		return nil, err
	}
	return &FirestoreWriter{client: client}, nil
}

func (fw *FirestoreWriter) Close() error {
	return fw.client.Close()
}

func (fw *FirestoreWriter) SaveRaw(ctx context.Context, run ForecastRun) error {
	_, _, err := fw.client.Collection(shared.ForecastRawCollection).Add(ctx, run)
	return err
}

// UpdateCache replaces the location's cache doc with the latest forecast run.
// Unlike the weather cache there is no history to merge, but the write stays
// transactional so alert merging (#64) can read the prior doc atomically.
func (fw *FirestoreWriter) UpdateCache(ctx context.Context, locationID string, run ForecastRun) error {
	cacheRef := fw.client.Collection(shared.ForecastCacheCollection).Doc(locationID)
	return fw.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		return tx.Set(cacheRef, buildCacheDoc(run))
	})
}

// buildCacheDoc derives the cache document from a forecast run.
func buildCacheDoc(run ForecastRun) ForecastCacheDoc {
	return ForecastCacheDoc{
		Location:    run.Location,
		LastUpdated: run.IssuedAt,
		IssuedAt:    run.IssuedAt,
		Points:      run.Points,
	}
}
