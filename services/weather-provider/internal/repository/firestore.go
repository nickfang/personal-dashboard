package repository

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

const (
	WEATHER_CACHE_COLLECTION = "weather_cache"
	DATABASE_ID              = "weather-log"
)

// Internal Firestore Models (Match weather-collector)
type WeatherPoint struct {
	Location             string    `firestore:"location"`
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
	Delta3h   *float64  `firestore:"delta_3h"`
	Delta6h   *float64  `firestore:"delta_6h"`
	Delta12h  *float64  `firestore:"delta_12h"`
	Delta24h  *float64  `firestore:"delta_24h"`
	Trend     string    `firestore:"trend"`
}

type CacheDoc struct {
	LocationID   string        `firestore:"-"` // Not in doc, but we use doc.ID
	LastUpdated  time.Time     `firestore:"last_updated"`
	CurrentValue WeatherPoint  `firestore:"current"`
	Analysis     PressureStats `firestore:"analysis"`
}

type FirestoreRepository struct {
	client *firestore.Client
}

func NewFirestoreRepository(ctx context.Context, projectID string) (*FirestoreRepository, error) {
	client, err := firestore.NewClientWithDatabase(ctx, projectID, DATABASE_ID)
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
	iter := r.client.Collection(WEATHER_CACHE_COLLECTION).Documents(ctx)
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
			return nil, err
		}
		cache.LocationID = doc.Ref.ID
		results = append(results, cache)
	}

	return results, nil
}

func (r *FirestoreRepository) GetByID(ctx context.Context, id string) (*CacheDoc, error) {
	doc, err := r.client.Collection(WEATHER_CACHE_COLLECTION).Doc(id).Get(ctx)
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
