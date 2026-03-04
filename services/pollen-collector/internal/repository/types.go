package repository

import "time"

const MaxHistoryPoints = 28 // 14 days × 2 readings/day

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
	LastUpdated  time.Time        `firestore:"last_updated"`
	CurrentValue PollenSnapshot   `firestore:"current"`
	History      []PollenSnapshot `firestore:"history"`
}
