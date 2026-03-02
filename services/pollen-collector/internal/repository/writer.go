package repository

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cloud.google.com/go/firestore"
	"github.com/nickfang/personal-dashboard/services/shared"
)

// Writer defines the interface for writing pollen data to storage.
type Writer interface {
	SaveRaw(ctx context.Context, snapshot PollenSnapshot) error
	UpdateCache(ctx context.Context, locationID string, snapshot PollenSnapshot) error
}

type FirestoreWriter struct {
	client *firestore.Client
}

func NewFirestoreWriter(ctx context.Context, projectID string) (*FirestoreWriter, error) {
	client, err := firestore.NewClientWithDatabase(ctx, projectID, shared.PollenDatabaseID)
	if err != nil {
		return nil, err
	}
	return &FirestoreWriter{client: client}, nil
}

func (fw *FirestoreWriter) Close() error {
	return fw.client.Close()
}

func (fw *FirestoreWriter) SaveRaw(ctx context.Context, snapshot PollenSnapshot) error {
	_, _, err := fw.client.Collection(shared.PollenRawCollection).Add(ctx, snapshot)
	return err
}

func (fw *FirestoreWriter) UpdateCache(ctx context.Context, locationID string, snapshot PollenSnapshot) error {
	cacheRef := fw.client.Collection(shared.PollenCacheCollection).Doc(locationID)

	return fw.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(cacheRef)
		var cache PollenCacheDoc
		if status.Code(err) == codes.NotFound {
			cache = PollenCacheDoc{History: []PollenSnapshot{}}
		} else if err != nil {
			return fmt.Errorf("reading cache doc: %w", err)
		} else {
			if err := doc.DataTo(&cache); err != nil {
				return err
			}
		}

		cache.History = append(cache.History, snapshot)
		if len(cache.History) > MaxHistoryPoints {
			cache.History = cache.History[len(cache.History)-MaxHistoryPoints:]
		}

		cache.LastUpdated = snapshot.CollectedAt
		cache.CurrentValue = snapshot

		return tx.Set(cacheRef, cache)
	})
}
