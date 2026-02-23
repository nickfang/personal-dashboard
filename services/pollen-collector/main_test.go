package main

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/nickfang/personal-dashboard/services/shared"
)

// roundTripFunc adapts a plain function into an http.RoundTripper.
// This lets tests intercept outgoing HTTP requests at the transport
// layer without needing a live server or modifying production code.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// validPollenJSON is a minimal valid response from the Google Pollen API.
var validPollenJSON = `{
	"dailyInfo": [{
		"date": {"year": 2026, "month": 2, "day": 23},
		"pollenTypeInfo": [
			{"code": "TREE", "displayName": "Tree", "inSeason": true, "indexInfo": {"value": 2, "category": "Low"}}
		],
		"plantInfo": [
			{"code": "JUNIPER", "displayName": "Juniper", "inSeason": true, "indexInfo": {"value": 2, "category": "Low"}}
		]
	}]
}`

func TestFetchPollen_APIKeyInHeaderNotURL(t *testing.T) {
	const testAPIKey = "test-secret-key-12345"
	var capturedReq *http.Request

	origClient := httpClient
	httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			capturedReq = req.Clone(req.Context())
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(validPollenJSON)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	defer func() { httpClient = origClient }()

	loc := shared.Location{ID: "test-loc", Lat: 30.0, Long: -97.0}
	_, err := fetchPollen(testAPIKey, loc)
	if err != nil {
		t.Fatalf("fetchPollen() returned error: %v", err)
	}

	// The API key MUST be sent via the X-Goog-Api-Key header.
	if got := capturedReq.Header.Get("X-Goog-Api-Key"); got != testAPIKey {
		t.Errorf("X-Goog-Api-Key header = %q, want %q", got, testAPIKey)
	}

	// The API key MUST NOT appear anywhere in the URL.
	if capturedReq.URL.Query().Get("key") != "" {
		t.Error("API key found in URL 'key' query param; must use header instead")
	}
	if strings.Contains(capturedReq.URL.String(), testAPIKey) {
		t.Error("API key found in URL string; must use header instead")
	}
}

func TestFetchPollen_ErrorDoesNotLeakAPIKey(t *testing.T) {
	const testAPIKey = "test-secret-key-12345"

	origClient := httpClient
	httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Status:     "403 Forbidden",
				Body:       io.NopCloser(strings.NewReader(`{"error":"forbidden"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	defer func() { httpClient = origClient }()

	loc := shared.Location{ID: "test-loc", Lat: 30.0, Long: -97.0}
	_, err := fetchPollen(testAPIKey, loc)
	if err == nil {
		t.Fatal("fetchPollen() should return error for 403 status")
	}

	if strings.Contains(err.Error(), testAPIKey) {
		t.Errorf("error message leaks API key: %s", err.Error())
	}
}

func TestFetchPollen_NonRetryableStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		status     string
	}{
		{"Unauthorized", http.StatusUnauthorized, "401 Unauthorized"},
		{"Forbidden", http.StatusForbidden, "403 Forbidden"},
		{"NotFound", http.StatusNotFound, "404 Not Found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origClient := httpClient
			httpClient = &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tt.statusCode,
						Status:     tt.status,
						Body:       io.NopCloser(strings.NewReader(`{}`)),
						Header:     make(http.Header),
					}, nil
				}),
			}
			defer func() { httpClient = origClient }()

			loc := shared.Location{ID: "test-loc", Lat: 30.0, Long: -97.0}
			_, err := fetchPollen("fake-key", loc)
			if err == nil {
				t.Fatal("fetchPollen() should return error for non-OK status")
			}

			// These status codes should be wrapped as nonRetryable
			var nr *nonRetryable
			if !errors.As(err, &nr) {
				t.Errorf("expected nonRetryable error for %d, got: %v", tt.statusCode, err)
			}
		})
	}
}

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
