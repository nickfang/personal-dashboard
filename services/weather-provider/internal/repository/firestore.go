package repository

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/nickfang/personal-dashboard/services/shared"
	"google.golang.org/api/iterator"
)

// Internal Firestore Models (Match weather-collector)
type WeatherPoint struct {
	LocationID           string    `firestore:"location"`
	Timestamp            time.Time `firestore:"timestamp"`
	HumidityPercent      int       `firestore:"humidity_pct"`
	PrecipitationPercent int       `firestore:"precipitation_pct"`
	UVIndex              int       `firestore:"uv_index"`
	PressureMb           float64   `firestore:"pressure_mb"`
	WindDirDeg           int       `firestore:"wind_dir_deg"`
	TempC                float64   `firestore:"temp_c"`
	TempFeelC            float64   `firestore:"temp_feel_c"`
	DewpointC            float64   `firestore:"dewpoint_c"`
	WindSpeedKph         float64   `firestore:"wind_speed_kph"`
	WindGustKph          float64   `firestore:"wind_gust_kph"`
	VisibilityKm         float64   `firestore:"visibility_km"`
	TempF                float64   `firestore:"temp_f"`
	TempFeelF            float64   `firestore:"temp_feel_f"`
	WindSpeedMph         float64   `firestore:"wind_speed_mph"`
	WindGustMph          float64   `firestore:"wind_gust_mph"`
	VisibilityM          float64   `firestore:"visibility_miles"`
	DewpointF            float64   `firestore:"dewpoint_f"`
}

type PressureStats struct {
	Timestamp time.Time `firestore:"timestamp"`
	Delta1h   *float64  `firestore:"delta_01h"`
	Delta3h   *float64  `firestore:"delta_03h"`
	Delta6h   *float64  `firestore:"delta_06h"`
	Delta12h  *float64  `firestore:"delta_12h"`
	Delta24h  *float64  `firestore:"delta_24h"`
	Trend     string    `firestore:"trend"`
}

type PressureCacheDoc struct {
	LocationID  string        `firestore:"-"` // Not in doc, but we use doc.ID
	LastUpdated time.Time     `firestore:"last_updated"`
	Analysis    PressureStats `firestore:"analysis"`
}

type WeatherCacheDoc struct {
	LocationID   string       `firestore:"-"` // not in doc, but we use doc.ID
	LastUpdated  time.Time    `firestore:"last_updated"`
	CurrentValue WeatherPoint `firestore:"current"`
}

type FirestoreRepository struct {
	client *firestore.Client
}

func NewFirestoreRepository(ctx context.Context, projectID string) (*FirestoreRepository, error) {
	client, err := firestore.NewClientWithDatabase(ctx, projectID, shared.WeatherDatabaseID)
	if err != nil {
		return nil, err
	}
	return &FirestoreRepository{client: client}, nil
}

func (r *FirestoreRepository) Close() error {
	return r.client.Close()
}

func (r *FirestoreRepository) GetByID(ctx context.Context, id string) (*PressureCacheDoc, error) {
	doc, err := r.client.Collection(shared.WeatherCacheCollection).Doc(id).Get(ctx)
	if err != nil {
		return nil, err
	}

	var cache PressureCacheDoc
	if err := doc.DataTo(&cache); err != nil {
		return nil, err
	}
	cache.LocationID = doc.Ref.ID
	return &cache, nil
}

func (r *FirestoreRepository) GetAll(ctx context.Context) ([]PressureCacheDoc, error) {
	var results []PressureCacheDoc
	// Safety: Limit query to 100 documents to prevent OOM.
	// In production, this should use pagination (cursors).
	iter := r.client.Collection(shared.WeatherCacheCollection).Limit(100).Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var cache PressureCacheDoc
		if err := doc.DataTo(&cache); err != nil {
			slog.Warn("Skipping invalid document in GetAll", "doc_id", doc.Ref.ID, "error", err)
			continue
		}
		cache.LocationID = doc.Ref.ID
		results = append(results, cache)
	}

	return results, nil
}

func (r *FirestoreRepository) GetLastWeather(ctx context.Context, id string) (*WeatherCacheDoc, error) {
	doc, err := r.client.Collection(shared.WeatherCacheCollection).Doc(id).Get(ctx)
	if err != nil {
		return nil, err
	}

	var cache WeatherCacheDoc
	if err := doc.DataTo(&cache); err != nil {
		return nil, err
	}
	cache.LocationID = doc.Ref.ID
	return &cache, nil
}

func (r *FirestoreRepository) GetAllLastWeather(ctx context.Context) ([]WeatherCacheDoc, error) {
	var results []WeatherCacheDoc
	iter := r.client.Collection(shared.WeatherCacheCollection).
		Select("current", "last_updated").Limit(100).
		Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var cache WeatherCacheDoc
		if err := doc.DataTo(&cache); err != nil {
			slog.Warn("Skipping invalid document in GetAllLastWeather", "doc_id", doc.Ref.ID, "error", err)
			continue
		}
		cache.LocationID = doc.Ref.ID
		results = append(results, cache)
	}
	return results, nil
}

func (r *FirestoreRepository) GetAllRaw(ctx context.Context) ([]WeatherPoint, error) {
	var results []WeatherPoint
	iter := r.client.Collection(shared.WeatherRawCollection).Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}

		var wp WeatherPoint
		if err := doc.DataTo(&wp); err != nil {
			return nil, err
		}
		results = append(results, wp)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.Before(results[j].Timestamp)
	})
	return results, nil
}
