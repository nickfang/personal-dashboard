package repository

import (
	"context"
	"log/slog"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/nickfang/personal-dashboard/services/shared"
	"google.golang.org/api/iterator"
)

// Internal Firestore Models (Match pollen-collector)
type StorePollenType struct {
	Code     string `firestore:"code"`
	Index    int    `firestore:"index"`
	Category string `firestore:"category"`
	InSeason bool   `firestore:"in_season"`
}

type StorePollenPlant struct {
	Code        string `firestore:"code"`
	DisplayName string `firestore:"display_name"`
	Index       int    `firestore:"index"`
	Category    string `firestore:"category"`
	InSeason    bool   `firestore:"in_season"`
}

type PollenSnapshot struct {
	LocationID      string             `firestore:"location_id"`
	CollectedAt     time.Time          `firestore:"collected_at"`
	OverallIndex    int                `firestore:"overall_index"`
	OverallCategory string             `firestore:"overall_category"`
	DominantType    string             `firestore:"dominant_type"`
	Types           []StorePollenType  `firestore:"types"`
	Plants          []StorePollenPlant `firestore:"plants"`
}

type CacheDoc struct {
	LocationID   string         `firestore:"-"` // Not in doc, but we use doc.ID
	LastUpdated  time.Time      `firestore:"last_updated"`
	CurrentValue PollenSnapshot `firestore:"current"`
}

type FirestoreRepository struct {
	client *firestore.Client
}

func NewFirestoreRepository(ctx context.Context, projectID string) (*FirestoreRepository, error) {
	client, err := firestore.NewClientWithDatabase(ctx, projectID, shared.DatabaseID)
	if err != nil {
		return nil, err
	}
	return &FirestoreRepository{client: client}, nil
}

func (r *FirestoreRepository) Close() error {
	return r.client.Close()
}

func (r *FirestoreRepository) GetAll(ctx context.Context) ([]CacheDoc, error) {
	var results []CacheDoc
	iter := r.client.Collection(shared.PollenCacheCollection).Limit(100).Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var cache CacheDoc
		if err := doc.DataTo(&cache); err != nil {
			slog.Warn("Skipping invalid document in GetAll", "doc_id", doc.Ref.ID, "error", err)
			continue
		}
		cache.LocationID = doc.Ref.ID
		results = append(results, cache)
	}

	return results, nil
}

func (r *FirestoreRepository) GetByID(ctx context.Context, id string) (*CacheDoc, error) {
	doc, err := r.client.Collection(shared.PollenCacheCollection).Doc(id).Get(ctx)
	if err != nil {
		return nil, err
	}

	var cache CacheDoc
	if err := doc.DataTo(&cache); err != nil {
		return nil, err
	}
	cache.LocationID = doc.Ref.ID
	return &cache, nil
}
