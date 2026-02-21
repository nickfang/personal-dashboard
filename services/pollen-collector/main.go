package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/joho/godotenv"
	"github.com/nickfang/personal-dashboard/services/shared"
)

const (
	MAX_HISTORY_POINTS    = 28 // 14 days × 2 readings/day
	POLLEN_RAW_COLLECTION = shared.PollenRawCollection
)

// Google Pollen API response types
type PollenAPIResponse struct {
	DailyInfo []DailyInfo `json:"dailyInfo"`
}

type DailyInfo struct {
	Date           APIDate          `json:"date"`
	PollenTypeInfo []PollenTypeInfo `json:"pollenTypeInfo"`
	PlantInfo      []PlantInfo      `json:"plantInfo"`
}

type APIDate struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

type PollenTypeInfo struct {
	Code        string    `json:"code"`
	DisplayName string    `json:"displayName"`
	InSeason    bool      `json:"inSeason"`
	IndexInfo   IndexInfo `json:"indexInfo"`
}

type PlantInfo struct {
	Code        string    `json:"code"`
	DisplayName string    `json:"displayName"`
	InSeason    bool      `json:"inSeason"`
	IndexInfo   IndexInfo `json:"indexInfo"`
}

type IndexInfo struct {
	Value    int    `json:"value"`
	Category string `json:"category"`
}

// Firestore storage models
type StoredPollenType struct {
	Code     string `firestore:"code"`
	Index    int    `firestore:"index"`
	Category string `firestore:"category"`
	InSeason bool   `firestore:"in_season"`
}

type StoredPollenPlant struct {
	Code        string `firestore:"code"`
	DisplayName string `firestore:"display_name"`
	Index       int    `firestore:"index"`
	Category    string `firestore:"category"`
	InSeason    bool   `firestore:"in_season"`
}

type PollenSnapshot struct {
	LocationID      string              `firestore:"location_id"`
	CollectedAt     time.Time           `firestore:"collected_at"`
	OverallIndex    int                 `firestore:"overall_index"`
	OverallCategory string              `firestore:"overall_category"`
	DominantType    string              `firestore:"dominant_type"`
	Types           []StoredPollenType  `firestore:"types"`
	Plants          []StoredPollenPlant `firestore:"plants"`
}

type PollenCacheDoc struct {
	LastUpdated time.Time        `firestore:"last_updated"`
	Current     PollenSnapshot   `firestore:"current"`
	History     []PollenSnapshot `firestore:"history"`
}

func fetchPollenWithRetry(apiKey string, loc shared.Location) (*PollenAPIResponse, error) {
	var lastErr error
	backoffs := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}

	for i := 0; i <= len(backoffs); i++ {
		data, err := fetchPollen(apiKey, loc)
		if err == nil {
			return data, nil
		}
		lastErr = err
		if i < len(backoffs) {
			time.Sleep(backoffs[i])
		}
	}
	return nil, fmt.Errorf("exhausted retries: %w", lastErr)
}

func fetchPollen(apiKey string, loc shared.Location) (*PollenAPIResponse, error) {
	url := fmt.Sprintf(
		"https://pollen.googleapis.com/v1/forecast:lookup?key=%s&location.latitude=%f&location.longitude=%f&days=1",
		apiKey, loc.Lat, loc.Long,
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Pollen API returned status: %s", resp.Status)
	}

	var data PollenAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode pollen JSON: %w", err)
	}

	if len(data.DailyInfo) == 0 {
		return nil, fmt.Errorf("no daily info returned for %s", loc.ID)
	}

	return &data, nil
}

// mapToSnapshot converts an API response into a PollenSnapshot for storage.
// It also computes the overall summary (highest UPI across the 3 pollen types).
func mapToSnapshot(locationID string, apiResp *PollenAPIResponse) PollenSnapshot {
	today := apiResp.DailyInfo[0]

	snapshot := PollenSnapshot{
		LocationID:  locationID,
		CollectedAt: time.Now(),
	}

	// Map pollen types
	for _, t := range today.PollenTypeInfo {
		snapshot.Types = append(snapshot.Types, StoredPollenType{
			Code:     t.Code,
			Index:    t.IndexInfo.Value,
			Category: t.IndexInfo.Category,
			InSeason: t.InSeason,
		})
	}

	// Map plants
	for _, p := range today.PlantInfo {
		snapshot.Plants = append(snapshot.Plants, StoredPollenPlant{
			Code:        p.Code,
			DisplayName: p.DisplayName,
			Index:       p.IndexInfo.Value,
			Category:    p.IndexInfo.Category,
			InSeason:    p.InSeason,
		})
	}

	// Compute overall summary: find the highest UPI across the 3 types
	for _, t := range snapshot.Types {
		if t.Index > snapshot.OverallIndex {
			snapshot.OverallIndex = t.Index
			snapshot.OverallCategory = t.Category
			snapshot.DominantType = t.Code
		}
	}

	return snapshot
}

func saveRawPollenData(ctx context.Context, client *firestore.Client, snapshot PollenSnapshot) error {
	_, _, err := client.Collection(POLLEN_RAW_COLLECTION).Add(ctx, snapshot)
	return err
}

func updatePollenCache(ctx context.Context, client *firestore.Client, locationID string, snapshot PollenSnapshot) error {
	cacheRef := client.Collection(shared.PollenCacheCollection).Doc(locationID)

	return client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(cacheRef)
		var cache PollenCacheDoc
		if err == nil {
			if err := doc.DataTo(&cache); err != nil {
				return err
			}
		} else {
			cache = PollenCacheDoc{History: []PollenSnapshot{}}
		}

		cache.History = append(cache.History, snapshot)
		if len(cache.History) > MAX_HISTORY_POINTS {
			cache.History = cache.History[len(cache.History)-MAX_HISTORY_POINTS:]
		}

		cache.LastUpdated = snapshot.CollectedAt
		cache.Current = snapshot

		return tx.Set(cacheRef, cache)
	})
}

func main() {
	shared.InitLogging()

	if err := godotenv.Load(); err != nil {
		slog.Debug("No .env file found, using system environment variables", "error", err)
	}

	ctx := context.Background()
	apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	projectID := os.Getenv("GCP_PROJECT_ID")

	if apiKey == "" || projectID == "" {
		slog.Error("Missing required env vars", "vars", "GOOGLE_MAPS_API_KEY, GCP_PROJECT_ID")
		os.Exit(1)
	}

	client, err := firestore.NewClientWithDatabase(ctx, projectID, shared.DatabaseID)
	if err != nil {
		slog.Error("Failed to create firestore client", "error", err)
		os.Exit(1)
	}
	defer client.Close()

	for _, loc := range shared.Locations {
		apiResp, err := fetchPollenWithRetry(apiKey, loc)
		if err != nil {
			slog.Error("Failed to fetch pollen after retries", "location", loc.ID, "error", err)
			continue
		}

		snapshot := mapToSnapshot(loc.ID, apiResp)
		if err := saveRawPollenData(ctx, client, snapshot); err != nil {
			slog.Error("Error saving raw pollen data", "location", loc.ID, "error", err)
			continue
		}

		if err := updatePollenCache(ctx, client, loc.ID, snapshot); err != nil {
			slog.Error("Error updating pollen cache", "location", loc.ID, "error", err)
			continue
		}

		slog.Info("Processed pollen", "location", loc.ID, "overall_index", snapshot.OverallIndex, "dominant", snapshot.DominantType)
	}
}
