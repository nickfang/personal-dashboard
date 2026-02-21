package main

import (
	"testing"
)

func TestMapToSnapshot_OverallSummary(t *testing.T) {
	apiResp := &PollenAPIResponse{
		DailyInfo: []DailyInfo{{
			PollenTypeInfo: []PollenTypeInfo{
				{Code: "GRASS", InSeason: true, IndexInfo: IndexInfo{Value: 2, Category: "Low"}},
				{Code: "TREE", InSeason: true, IndexInfo: IndexInfo{Value: 4, Category: "High"}},
				{Code: "WEED", InSeason: false, IndexInfo: IndexInfo{Value: 0, Category: "None"}},
			},
			PlantInfo: []PlantInfo{
				{Code: "JUNIPER", DisplayName: "Juniper", InSeason: true, IndexInfo: IndexInfo{Value: 4, Category: "High"}},
				{Code: "OAK", DisplayName: "Oak", InSeason: false, IndexInfo: IndexInfo{Value: 0, Category: "None"}},
			},
		}},
	}

	snapshot := mapToSnapshot("house-nick", apiResp)

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
	apiResp := &PollenAPIResponse{
		DailyInfo: []DailyInfo{{
			PollenTypeInfo: []PollenTypeInfo{
				{Code: "GRASS", InSeason: true, IndexInfo: IndexInfo{Value: 2, Category: "Low"}},
				{Code: "TREE", InSeason: true, IndexInfo: IndexInfo{Value: 4, Category: "High"}},
				{Code: "WEED", InSeason: false, IndexInfo: IndexInfo{Value: 0, Category: "None"}},
			},
		}},
	}

	snapshot := mapToSnapshot("house-nick", apiResp)

	if len(snapshot.Types) != 3 {
		t.Fatalf("len(Types) = %d, want 3", len(snapshot.Types))
	}

	// Verify each type is mapped correctly
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
	apiResp := &PollenAPIResponse{
		DailyInfo: []DailyInfo{{
			PlantInfo: []PlantInfo{
				{Code: "JUNIPER", DisplayName: "Juniper", InSeason: true, IndexInfo: IndexInfo{Value: 4, Category: "High"}},
				{Code: "OAK", DisplayName: "Oak", InSeason: false, IndexInfo: IndexInfo{Value: 0, Category: "None"}},
				{Code: "RAGWEED", DisplayName: "Ragweed", InSeason: true, IndexInfo: IndexInfo{Value: 3, Category: "Moderate"}},
			},
		}},
	}

	snapshot := mapToSnapshot("house-nick", apiResp)

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
	apiResp := &PollenAPIResponse{
		DailyInfo: []DailyInfo{{
			PollenTypeInfo: []PollenTypeInfo{
				{Code: "GRASS", InSeason: false, IndexInfo: IndexInfo{Value: 0, Category: "None"}},
				{Code: "TREE", InSeason: false, IndexInfo: IndexInfo{Value: 0, Category: "None"}},
				{Code: "WEED", InSeason: false, IndexInfo: IndexInfo{Value: 0, Category: "None"}},
			},
		}},
	}

	snapshot := mapToSnapshot("house-nick", apiResp)

	if snapshot.OverallIndex != 0 {
		t.Errorf("OverallIndex = %d, want 0", snapshot.OverallIndex)
	}
	if snapshot.DominantType != "" {
		t.Errorf("DominantType = %s, want empty string (no dominant when all zero)", snapshot.DominantType)
	}
}

func TestMapToSnapshot_LocationID(t *testing.T) {
	apiResp := &PollenAPIResponse{
		DailyInfo: []DailyInfo{{
			PollenTypeInfo: []PollenTypeInfo{
				{Code: "TREE", InSeason: true, IndexInfo: IndexInfo{Value: 1, Category: "Very Low"}},
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
			snapshot := mapToSnapshot(tt.locationID, apiResp)
			if snapshot.LocationID != tt.locationID {
				t.Errorf("LocationID = %s, want %s", snapshot.LocationID, tt.locationID)
			}
		})
	}
}

func TestMapToSnapshot_CollectedAtIsSet(t *testing.T) {
	apiResp := &PollenAPIResponse{
		DailyInfo: []DailyInfo{{
			PollenTypeInfo: []PollenTypeInfo{
				{Code: "TREE", InSeason: true, IndexInfo: IndexInfo{Value: 1, Category: "Very Low"}},
			},
		}},
	}

	snapshot := mapToSnapshot("house-nick", apiResp)

	if snapshot.CollectedAt.IsZero() {
		t.Error("CollectedAt should be set to a non-zero time")
	}
}

func TestMapToSnapshot_TiedTypes(t *testing.T) {
	// When two types have the same highest UPI, the first one encountered wins
	apiResp := &PollenAPIResponse{
		DailyInfo: []DailyInfo{{
			PollenTypeInfo: []PollenTypeInfo{
				{Code: "GRASS", InSeason: true, IndexInfo: IndexInfo{Value: 3, Category: "Moderate"}},
				{Code: "TREE", InSeason: true, IndexInfo: IndexInfo{Value: 3, Category: "Moderate"}},
				{Code: "WEED", InSeason: false, IndexInfo: IndexInfo{Value: 1, Category: "Very Low"}},
			},
		}},
	}

	snapshot := mapToSnapshot("house-nick", apiResp)

	if snapshot.OverallIndex != 3 {
		t.Errorf("OverallIndex = %d, want 3", snapshot.OverallIndex)
	}
	// First type with the highest value should be dominant
	if snapshot.DominantType != "GRASS" {
		t.Errorf("DominantType = %s, want GRASS (first with highest UPI)", snapshot.DominantType)
	}
}
