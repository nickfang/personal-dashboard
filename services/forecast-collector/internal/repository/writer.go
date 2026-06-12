package repository

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/nickfang/personal-dashboard/services/shared"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MergeFunc reconciles freshly detected alerts against the previously stored
// set. Like weather-collector's AnalyzeFunc, it lets the service layer's
// business logic run inside the repository's transaction.
type MergeFunc func(prev []shared.Alert) []shared.Alert

// Writer defines the interface for writing forecast data to storage.
type Writer interface {
	SaveRaw(ctx context.Context, run ForecastRun) error
	UpdateCache(ctx context.Context, locationID string, run ForecastRun, merge MergeFunc) error
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
// The prior doc is read inside the transaction so alert merging sees the
// stored alert state atomically with the write that replaces it.
func (fw *FirestoreWriter) UpdateCache(ctx context.Context, locationID string, run ForecastRun, merge MergeFunc) error {
	cacheRef := fw.client.Collection(shared.ForecastCacheCollection).Doc(locationID)
	return fw.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		var prev ForecastCacheDoc
		doc, err := tx.Get(cacheRef)
		if status.Code(err) == codes.NotFound {
			// First run for this location: no prior alerts.
		} else if err != nil {
			return fmt.Errorf("reading forecast cache doc: %w", err)
		} else if err := doc.DataTo(&prev); err != nil {
			return err
		}
		return tx.Set(cacheRef, buildCacheDoc(run, merge(prev.Alerts)))
	})
}

// buildCacheDoc derives the cache document from a forecast run and the
// merged alert set.
func buildCacheDoc(run ForecastRun, alerts []shared.Alert) ForecastCacheDoc {
	return ForecastCacheDoc{
		Location:    run.Location,
		LastUpdated: run.IssuedAt,
		IssuedAt:    run.IssuedAt,
		Points:      run.Points,
		Alerts:      alerts,
	}
}
