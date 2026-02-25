package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/nickfang/personal-dashboard/services/pollen-collector/internal/client"
	"github.com/nickfang/personal-dashboard/services/pollen-collector/internal/repository"
	"github.com/nickfang/personal-dashboard/services/pollen-collector/internal/testutil"
)

// --- Mapping tests (migrated from main_test.go) ---

func TestMapToSnapshot_OverallSummary(t *testing.T) {
	apiResp := &client.PollenAPIResponse{
		DailyInfo: []client.DailyInfo{{
			PollenTypeInfo: []client.PollenTypeInfo{
				{Code: "GRASS", InSeason: true, IndexInfo: client.IndexInfo{Value: 2, Category: "Low"}},
				{Code: "TREE", InSeason: true, IndexInfo: client.IndexInfo{Value: 4, Category: "High"}},
				{Code: "WEED", InSeason: false, IndexInfo: client.IndexInfo{Value: 0, Category: "None"}},
			},
			PlantInfo: []client.PlantInfo{
				{Code: "JUNIPER", DisplayName: "Juniper", InSeason: true, IndexInfo: client.IndexInfo{Value: 4, Category: "High"}},
				{Code: "OAK", DisplayName: "Oak", InSeason: false, IndexInfo: client.IndexInfo{Value: 0, Category: "None"}},
			},
		}},
	}

	snapshot := MapToSnapshot("house-nick", apiResp)

	if snapshot.OverallIndex != 4 {
		t.Errorf("OverallIndex = %d, want 4", snapshot.OverallIndex)
	}
	if snapshot.OverallCategory != "High" {
		t.Errorf("OverallCategory = %s, want High", snapshot.OverallCategory)
	}
	if snapshot.DominantType != "TREE" {
		t.Errorf("DominantType = %s, want TREE", snapshot.DominantType)
	}
}

func TestMapToSnapshot_TypeMapping(t *testing.T) {
	apiResp := &client.PollenAPIResponse{
		DailyInfo: []client.DailyInfo{{
			PollenTypeInfo: []client.PollenTypeInfo{
				{Code: "GRASS", InSeason: true, IndexInfo: client.IndexInfo{Value: 2, Category: "Low"}},
				{Code: "TREE", InSeason: true, IndexInfo: client.IndexInfo{Value: 4, Category: "High"}},
				{Code: "WEED", InSeason: false, IndexInfo: client.IndexInfo{Value: 0, Category: "None"}},
			},
		}},
	}

	snapshot := MapToSnapshot("house-nick", apiResp)

	if len(snapshot.Types) != 3 {
		t.Fatalf("len(Types) = %d, want 3", len(snapshot.Types))
	}

	tests := []struct {
		index    int
		code     string
		upi      int
		category string
		inSeason bool
	}{
		{0, "GRASS", 2, "Low", true},
		{1, "TREE", 4, "High", true},
		{2, "WEED", 0, "None", false},
	}

	for _, tt := range tests {
		st := snapshot.Types[tt.index]
		if st.Code != tt.code {
			t.Errorf("Types[%d].Code = %s, want %s", tt.index, st.Code, tt.code)
		}
		if st.Index != tt.upi {
			t.Errorf("Types[%d].Index = %d, want %d", tt.index, st.Index, tt.upi)
		}
		if st.Category != tt.category {
			t.Errorf("Types[%d].Category = %s, want %s", tt.index, st.Category, tt.category)
		}
		if st.InSeason != tt.inSeason {
			t.Errorf("Types[%d].InSeason = %v, want %v", tt.index, st.InSeason, tt.inSeason)
		}
	}
}

func TestMapToSnapshot_PlantMapping(t *testing.T) {
	apiResp := &client.PollenAPIResponse{
		DailyInfo: []client.DailyInfo{{
			PlantInfo: []client.PlantInfo{
				{Code: "JUNIPER", DisplayName: "Juniper", InSeason: true, IndexInfo: client.IndexInfo{Value: 4, Category: "High"}},
				{Code: "OAK", DisplayName: "Oak", InSeason: false, IndexInfo: client.IndexInfo{Value: 0, Category: "None"}},
				{Code: "RAGWEED", DisplayName: "Ragweed", InSeason: true, IndexInfo: client.IndexInfo{Value: 3, Category: "Moderate"}},
			},
		}},
	}

	snapshot := MapToSnapshot("house-nick", apiResp)

	if len(snapshot.Plants) != 3 {
		t.Fatalf("len(Plants) = %d, want 3", len(snapshot.Plants))
	}

	tests := []struct {
		index       int
		code        string
		displayName string
		upi         int
		category    string
		inSeason    bool
	}{
		{0, "JUNIPER", "Juniper", 4, "High", true},
		{1, "OAK", "Oak", 0, "None", false},
		{2, "RAGWEED", "Ragweed", 3, "Moderate", true},
	}

	for _, tt := range tests {
		sp := snapshot.Plants[tt.index]
		if sp.Code != tt.code {
			t.Errorf("Plants[%d].Code = %s, want %s", tt.index, sp.Code, tt.code)
		}
		if sp.DisplayName != tt.displayName {
			t.Errorf("Plants[%d].DisplayName = %s, want %s", tt.index, sp.DisplayName, tt.displayName)
		}
		if sp.Index != tt.upi {
			t.Errorf("Plants[%d].Index = %d, want %d", tt.index, sp.Index, tt.upi)
		}
		if sp.Category != tt.category {
			t.Errorf("Plants[%d].Category = %s, want %s", tt.index, sp.Category, tt.category)
		}
		if sp.InSeason != tt.inSeason {
			t.Errorf("Plants[%d].InSeason = %v, want %v", tt.index, sp.InSeason, tt.inSeason)
		}
	}
}

func TestMapToSnapshot_AllZero(t *testing.T) {
	apiResp := &client.PollenAPIResponse{
		DailyInfo: []client.DailyInfo{{
			PollenTypeInfo: []client.PollenTypeInfo{
				{Code: "GRASS", InSeason: false, IndexInfo: client.IndexInfo{Value: 0, Category: "None"}},
				{Code: "TREE", InSeason: false, IndexInfo: client.IndexInfo{Value: 0, Category: "None"}},
				{Code: "WEED", InSeason: false, IndexInfo: client.IndexInfo{Value: 0, Category: "None"}},
			},
		}},
	}

	snapshot := MapToSnapshot("house-nick", apiResp)

	if snapshot.OverallIndex != 0 {
		t.Errorf("OverallIndex = %d, want 0", snapshot.OverallIndex)
	}
	if snapshot.DominantType != "" {
		t.Errorf("DominantType = %s, want empty string (no dominant when all zero)", snapshot.DominantType)
	}
}

func TestMapToSnapshot_LocationID(t *testing.T) {
	apiResp := &client.PollenAPIResponse{
		DailyInfo: []client.DailyInfo{{
			PollenTypeInfo: []client.PollenTypeInfo{
				{Code: "TREE", InSeason: true, IndexInfo: client.IndexInfo{Value: 1, Category: "Very Low"}},
			},
		}},
	}

	tests := []struct {
		locationID string
	}{
		{"house-nick"},
		{"house-nita"},
		{"distribution-hall"},
	}

	for _, tt := range tests {
		t.Run(tt.locationID, func(t *testing.T) {
			snapshot := MapToSnapshot(tt.locationID, apiResp)
			if snapshot.LocationID != tt.locationID {
				t.Errorf("LocationID = %s, want %s", snapshot.LocationID, tt.locationID)
			}
		})
	}
}

func TestMapToSnapshot_CollectedAtIsSet(t *testing.T) {
	apiResp := &client.PollenAPIResponse{
		DailyInfo: []client.DailyInfo{{
			PollenTypeInfo: []client.PollenTypeInfo{
				{Code: "TREE", InSeason: true, IndexInfo: client.IndexInfo{Value: 1, Category: "Very Low"}},
			},
		}},
	}

	snapshot := MapToSnapshot("house-nick", apiResp)

	if snapshot.CollectedAt.IsZero() {
		t.Error("CollectedAt should be set to a non-zero time")
	}
}

func TestMapToSnapshot_TiedTypes(t *testing.T) {
	// When two types have the same highest UPI, the first one encountered wins
	apiResp := &client.PollenAPIResponse{
		DailyInfo: []client.DailyInfo{{
			PollenTypeInfo: []client.PollenTypeInfo{
				{Code: "GRASS", InSeason: true, IndexInfo: client.IndexInfo{Value: 3, Category: "Moderate"}},
				{Code: "TREE", InSeason: true, IndexInfo: client.IndexInfo{Value: 3, Category: "Moderate"}},
				{Code: "WEED", InSeason: false, IndexInfo: client.IndexInfo{Value: 1, Category: "Very Low"}},
			},
		}},
	}

	snapshot := MapToSnapshot("house-nick", apiResp)

	if snapshot.OverallIndex != 3 {
		t.Errorf("OverallIndex = %d, want 3", snapshot.OverallIndex)
	}
	// First type with the highest value should be dominant
	if snapshot.DominantType != "GRASS" {
		t.Errorf("DominantType = %s, want GRASS (first with highest UPI)", snapshot.DominantType)
	}
}

// --- Orchestration tests (new) ---

func TestCollect_Success(t *testing.T) {
	var savedSnapshot repository.PollenSnapshot
	var cachedLocationID string

	fetcher := &testutil.MockFetcher{
		FetchFn: func(ctx context.Context, apiKey, locationID string, lat, long float64) (*client.PollenAPIResponse, error) {
			return &client.PollenAPIResponse{
				DailyInfo: []client.DailyInfo{{
					PollenTypeInfo: []client.PollenTypeInfo{
						{Code: "TREE", InSeason: true, IndexInfo: client.IndexInfo{Value: 3, Category: "Moderate"}},
					},
				}},
			}, nil
		},
	}

	writer := &testutil.MockWriter{
		SaveRawFn: func(ctx context.Context, snapshot repository.PollenSnapshot) error {
			savedSnapshot = snapshot
			return nil
		},
		UpdateCacheFn: func(ctx context.Context, locationID string, snapshot repository.PollenSnapshot) error {
			cachedLocationID = locationID
			return nil
		},
	}

	svc := NewCollectorService(fetcher, writer)
	err := svc.Collect(context.Background(), "test-key", "house-nick", 30.0, -97.0)

	if err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}

	if savedSnapshot.LocationID != "house-nick" {
		t.Errorf("SaveRaw snapshot LocationID = %q, want %q", savedSnapshot.LocationID, "house-nick")
	}
	if cachedLocationID != "house-nick" {
		t.Errorf("UpdateCache locationID = %q, want %q", cachedLocationID, "house-nick")
	}
}

func TestCollect_FetchError(t *testing.T) {
	writerCalled := false

	fetcher := &testutil.MockFetcher{
		FetchFn: func(ctx context.Context, apiKey, locationID string, lat, long float64) (*client.PollenAPIResponse, error) {
			return nil, fmt.Errorf("API unavailable")
		},
	}

	writer := &testutil.MockWriter{
		SaveRawFn: func(ctx context.Context, snapshot repository.PollenSnapshot) error {
			writerCalled = true
			return nil
		},
		UpdateCacheFn: func(ctx context.Context, locationID string, snapshot repository.PollenSnapshot) error {
			writerCalled = true
			return nil
		},
	}

	svc := NewCollectorService(fetcher, writer)
	err := svc.Collect(context.Background(), "test-key", "house-nick", 30.0, -97.0)

	if err == nil {
		t.Fatal("Collect() should return error when fetch fails")
	}

	if writerCalled {
		t.Error("Writer should not be called when fetch fails")
	}
}

func TestCollect_SaveRawError(t *testing.T) {
	cacheCalled := false

	fetcher := &testutil.MockFetcher{
		FetchFn: func(ctx context.Context, apiKey, locationID string, lat, long float64) (*client.PollenAPIResponse, error) {
			return &client.PollenAPIResponse{
				DailyInfo: []client.DailyInfo{{
					PollenTypeInfo: []client.PollenTypeInfo{
						{Code: "TREE", InSeason: true, IndexInfo: client.IndexInfo{Value: 1, Category: "Very Low"}},
					},
				}},
			}, nil
		},
	}

	writer := &testutil.MockWriter{
		SaveRawFn: func(ctx context.Context, snapshot repository.PollenSnapshot) error {
			return fmt.Errorf("firestore write failed")
		},
		UpdateCacheFn: func(ctx context.Context, locationID string, snapshot repository.PollenSnapshot) error {
			cacheCalled = true
			return nil
		},
	}

	svc := NewCollectorService(fetcher, writer)
	err := svc.Collect(context.Background(), "test-key", "house-nick", 30.0, -97.0)

	if err == nil {
		t.Fatal("Collect() should return error when SaveRaw fails")
	}
	if cacheCalled {
		t.Error("UpdateCache should not be called when SaveRaw fails")
	}
}

func TestCollect_UpdateCacheError(t *testing.T) {
	fetcher := &testutil.MockFetcher{
		FetchFn: func(ctx context.Context, apiKey, locationID string, lat, long float64) (*client.PollenAPIResponse, error) {
			return &client.PollenAPIResponse{
				DailyInfo: []client.DailyInfo{{
					PollenTypeInfo: []client.PollenTypeInfo{
						{Code: "TREE", InSeason: true, IndexInfo: client.IndexInfo{Value: 1, Category: "Very Low"}},
					},
				}},
			}, nil
		},
	}

	writer := &testutil.MockWriter{
		SaveRawFn: func(ctx context.Context, snapshot repository.PollenSnapshot) error {
			return nil
		},
		UpdateCacheFn: func(ctx context.Context, locationID string, snapshot repository.PollenSnapshot) error {
			return fmt.Errorf("cache update failed")
		},
	}

	svc := NewCollectorService(fetcher, writer)
	err := svc.Collect(context.Background(), "test-key", "house-nick", 30.0, -97.0)

	if err == nil {
		t.Fatal("Collect() should return error when UpdateCache fails")
	}
}
