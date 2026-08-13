package repository

import (
	"context"
	"fmt"
	"time"

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
	UpdateCache(ctx context.Context, locationID string, run ForecastRun, merge MergeFunc) ([]shared.Alert, error)
	MarkNotified(ctx context.Context, locationID string, alertIDs []string, at time.Time) error
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

// UpdateCache replaces the location's cache doc with the latest forecast run
// and returns the merged alert set that was committed. The prior doc is read
// inside the transaction so alert merging sees the stored alert state
// atomically with the write that replaces it.
//
// The committed set is returned explicitly rather than captured through the
// MergeFunc closure: Firestore retries transactions, so the closure can run
// more than once and relying on last-write-wins there would be subtle in a
// way this is not.
func (fw *FirestoreWriter) UpdateCache(ctx context.Context, locationID string, run ForecastRun, merge MergeFunc) ([]shared.Alert, error) {
	cacheRef := fw.client.Collection(shared.ForecastCacheCollection).Doc(locationID)
	var committed []shared.Alert
	err := fw.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		var prev ForecastCacheDoc
		doc, err := tx.Get(cacheRef)
		if status.Code(err) == codes.NotFound {
			// First run for this location: no prior alerts.
		} else if err != nil {
			return fmt.Errorf("reading forecast cache doc: %w", err)
		} else if err := doc.DataTo(&prev); err != nil {
			return err
		}
		merged := merge(prev.Alerts)
		if err := tx.Set(cacheRef, buildCacheDoc(run, merged)); err != nil {
			return err
		}
		committed = merged
		return nil
	})
	if err != nil {
		return nil, err
	}
	return committed, nil
}

// MarkNotified records delivery against the listed alert IDs.
//
// It re-reads the cache doc inside a transaction and matches by ID rather
// than by position, so a concurrent collection run that reordered or replaced
// the alert set cannot be clobbered. Status is untouched: it tracks whether
// the condition is present, which delivery does not change.
//
// Only the alerts field is written — points carries 72 forecast hours and
// there is no reason to rewrite it.
func (fw *FirestoreWriter) MarkNotified(ctx context.Context, locationID string, alertIDs []string, at time.Time) error {
	if len(alertIDs) == 0 {
		return nil
	}
	cacheRef := fw.client.Collection(shared.ForecastCacheCollection).Doc(locationID)
	return fw.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(cacheRef)
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("forecast cache doc for %s not found", locationID)
		} else if err != nil {
			return fmt.Errorf("reading forecast cache doc: %w", err)
		}
		var cached ForecastCacheDoc
		if err := doc.DataTo(&cached); err != nil {
			return err
		}
		return tx.Update(cacheRef, []firestore.Update{
			{Path: "alerts", Value: applyNotifiedAt(cached.Alerts, alertIDs, at)},
		})
	})
}

// applyNotifiedAt stamps at onto the alerts whose IDs are listed, leaving
// every other alert as stored.
func applyNotifiedAt(alerts []shared.Alert, alertIDs []string, at time.Time) []shared.Alert {
	wanted := make(map[string]bool, len(alertIDs))
	for _, id := range alertIDs {
		wanted[id] = true
	}
	updated := make([]shared.Alert, len(alerts))
	copy(updated, alerts)
	for i := range updated {
		if wanted[updated[i].ID] {
			updated[i].NotifiedAt = at
		}
	}
	return updated
}

// buildCacheDoc derives the cache document from a forecast run and the
// merged alert set.
func buildCacheDoc(run ForecastRun, alerts []shared.Alert) ForecastCacheDoc {
	return ForecastCacheDoc{
		Location: run.Location,
		IssuedAt: run.IssuedAt,
		Points:   run.Points,
		Alerts:   alerts,
	}
}
