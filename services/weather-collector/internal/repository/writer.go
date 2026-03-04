package repository

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/nickfang/personal-dashboard/services/shared"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AnalyzeFunc is a callback that computes pressure statistics from history.
// This allows the service layer's business logic to run inside the repository's transaction.
type AnalyzeFunc func(history []PressurePoint) PressureStats

// Writer defines the interface for writing weather data to storage.
type Writer interface {
	SaveRaw(ctx context.Context, wp WeatherPoint) error
	UpdateCache(ctx context.Context, locationID string, wp WeatherPoint, analyze AnalyzeFunc) error
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

func (fw *FirestoreWriter) SaveRaw(ctx context.Context, wp WeatherPoint) error {
	_, _, err := fw.client.Collection(shared.WeatherRawCollection).Add(ctx, wp)
	return err
}

func (fw *FirestoreWriter) UpdateCache(ctx context.Context, locationID string, wp WeatherPoint, analyze AnalyzeFunc) error {
	cacheRef := fw.client.Collection(shared.WeatherCacheCollection).Doc(locationID)
	return fw.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		cache, err := getUpdatedCacheDoc(cacheRef, &wp, tx, analyze)
		if err != nil {
			return err
		}
		return tx.Set(cacheRef, cache)
	})
}

func getUpdatedCacheDoc(cacheRef *firestore.DocumentRef, wp *WeatherPoint, tx *firestore.Transaction, analyze AnalyzeFunc) (CacheDoc, error) {
	doc, err := tx.Get(cacheRef)
	var cache CacheDoc
	if status.Code(err) == codes.NotFound {
		cache = CacheDoc{History: []PressurePoint{}}
	} else if err != nil {
		return cache, fmt.Errorf("reading cache doc: %w", err)
	} else {
		if err := doc.DataTo(&cache); err != nil {
			return cache, err
		}
	}
	newPoint := PressurePoint{
		TimeStamp:       wp.Timestamp,
		TempC:           wp.TempC,
		TempF:           wp.TempF,
		HumidityPercent: wp.HumidityPercent,
		PressureMb:      wp.PressureMb,
		TempFeelC:       wp.TempFeelC,
		TempFeelF:       wp.TempFeelF,
		DewpointC:       wp.DewpointC,
		DewpointF:       wp.DewpointF,
	}
	cache.History = append(cache.History, newPoint)
	if len(cache.History) > MaxHistoryPoints {
		cache.History = cache.History[len(cache.History)-MaxHistoryPoints:]
	}

	cache.LastUpdated = wp.Timestamp
	cache.CurrentValue = *wp
	cache.Analysis = analyze(cache.History)

	return cache, nil
}
