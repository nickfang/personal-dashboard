package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const samplePayload = `{
  "weather": {
    "house-nick": {
      "locationId": "house-nick",
      "lastUpdated": "2026-04-13T14:30:05Z",
      "tempC": 29.6,
      "tempF": 85.2,
      "tempFeelC": 31.7,
      "tempFeelF": 89.1,
      "humidityPercent": 62,
      "precipitationPercent": 10,
      "pressureMb": 1013.25
    }
  },
  "pressure": {
    "house-nick": {
      "locationId": "house-nick",
      "lastUpdated": "2026-04-13T14:30:05Z",
      "delta1h": 0.3,
      "delta3h": 0.8,
      "delta6h": 1.2,
      "delta12h": 2.0,
      "delta24h": 3.1,
      "trend": "rising"
    }
  },
  "pollen": {
    "house-nick": {
      "locationId": "house-nick",
      "collectedAt": "2026-04-13T06:00:00Z",
      "overallIndex": 4,
      "overallCategory": "High",
      "dominantType": "TREE",
      "types": [
        {"code": "TREE", "index": 4, "category": "High", "inSeason": true},
        {"code": "GRASS", "index": 1, "category": "Low", "inSeason": false}
      ],
      "plants": [
        {"code": "JUNIPER", "displayName": "Juniper", "index": 4, "category": "High", "inSeason": true},
        {"code": "OAK", "displayName": "Oak", "index": 2, "category": "Moderate", "inSeason": true}
      ]
    }
  }
}`

func TestClientFetch_HappyPath(t *testing.T) {
	var gotPath, gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(samplePayload))
	}))
	defer srv.Close()

	c, err := New(srv.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	resp, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if gotPath != "/v1/dashboard" {
		t.Errorf("path = %q, want /v1/dashboard", gotPath)
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept header = %q, want application/json", gotAccept)
	}

	w, ok := resp.Weather["house-nick"]
	if !ok {
		t.Fatal("weather[house-nick] missing")
	}
	if w.TempF != 85.2 {
		t.Errorf("TempF = %v, want 85.2", w.TempF)
	}
	if w.HumidityPercent != 62 {
		t.Errorf("HumidityPercent = %v, want 62", w.HumidityPercent)
	}
	if w.PressureMb != 1013.25 {
		t.Errorf("PressureMb = %v, want 1013.25", w.PressureMb)
	}

	p, ok := resp.Pressure["house-nick"]
	if !ok {
		t.Fatal("pressure[house-nick] missing")
	}
	if p.Trend != "rising" {
		t.Errorf("Trend = %q, want rising", p.Trend)
	}
	if p.Delta24h != 3.1 {
		t.Errorf("Delta24h = %v, want 3.1", p.Delta24h)
	}

	pl, ok := resp.Pollen["house-nick"]
	if !ok {
		t.Fatal("pollen[house-nick] missing")
	}
	if pl.OverallIndex != 4 {
		t.Errorf("OverallIndex = %v, want 4", pl.OverallIndex)
	}
	if pl.DominantType != "TREE" {
		t.Errorf("DominantType = %q, want TREE", pl.DominantType)
	}
	if len(pl.Types) != 2 || pl.Types[0].Code != "TREE" || !pl.Types[0].InSeason {
		t.Errorf("Types parsed incorrectly: %+v", pl.Types)
	}
	if len(pl.Plants) != 2 || pl.Plants[0].DisplayName != "Juniper" {
		t.Errorf("Plants parsed incorrectly: %+v", pl.Plants)
	}
}

func TestClientFetch_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c, err := New(srv.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := c.Fetch(context.Background()); err == nil {
		t.Fatal("expected error on 500, got nil")
	}
}

const sampleSingleLocationPayload = `{
  "weather": {
    "house-nick": {
      "locationId": "house-nick",
      "lastUpdated": "2026-04-13T14:30:05Z",
      "tempC": 29.6,
      "tempF": 85.2,
      "humidityPercent": 62,
      "pressureMb": 1013.25
    }
  },
  "pressure": {
    "house-nick": {
      "locationId": "house-nick",
      "lastUpdated": "2026-04-13T14:30:05Z",
      "delta1h": 0.3,
      "trend": "rising"
    }
  },
  "pollen": {
    "house-nick": {
      "locationId": "house-nick",
      "collectedAt": "2026-04-13T06:00:00Z",
      "overallIndex": 4,
      "overallCategory": "High",
      "dominantType": "TREE"
    }
  }
}`

func TestClientFetchLocation_HappyPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleSingleLocationPayload))
	}))
	defer srv.Close()

	c, err := New(srv.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	resp, err := c.FetchLocation(context.Background(), "house-nick")
	if err != nil {
		t.Fatalf("FetchLocation: %v", err)
	}

	if gotPath != "/v1/dashboard/house-nick" {
		t.Errorf("path = %q, want /v1/dashboard/house-nick", gotPath)
	}

	if w, ok := resp.Weather["house-nick"]; !ok {
		t.Fatal("weather[house-nick] missing")
	} else if w.TempF != 85.2 {
		t.Errorf("TempF = %v, want 85.2", w.TempF)
	}
	if _, ok := resp.Pressure["house-nick"]; !ok {
		t.Error("pressure[house-nick] missing")
	}
	if _, ok := resp.Pollen["house-nick"]; !ok {
		t.Error("pollen[house-nick] missing")
	}
}

func TestClientFetchLocation_PathEscapes(t *testing.T) {
	var gotURI string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// RequestURI is the raw URI as sent on the wire — URL.Path would
		// already be decoded, masking whether escaping happened.
		gotURI = r.RequestURI
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c, _ := New(srv.URL)
	if _, err := c.FetchLocation(context.Background(), "weird id/with spaces"); err != nil {
		t.Fatalf("FetchLocation: %v", err)
	}
	if gotURI != "/v1/dashboard/weird%20id%2Fwith%20spaces" {
		t.Errorf("RequestURI = %q, want escaped form", gotURI)
	}
}

func TestClientFetchLocation_EmptyID(t *testing.T) {
	c, _ := New("http://localhost:9999")
	if _, err := c.FetchLocation(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty locationID, got nil")
	}
}

func TestResponseMerge_LWW_NewerSrcWins(t *testing.T) {
	const olderTS = "2026-04-13T13:00:00Z"
	const newerTS = "2026-04-13T14:30:05Z"
	const olderPollenTS = "2026-04-12T06:00:00Z"

	dst := &Response{
		Weather: map[string]Weather{
			"house-nick": {LocationID: "house-nick", LastUpdated: olderTS, TempF: 70.0},
			"house-nita": {LocationID: "house-nita", LastUpdated: olderTS, TempF: 71.0},
		},
		Pressure: map[string]Pressure{
			"house-nick": {LocationID: "house-nick", LastUpdated: olderTS, Trend: "steady"},
		},
		Pollen: map[string]Pollen{
			"house-nita": {LocationID: "house-nita", CollectedAt: olderPollenTS, OverallIndex: 1},
		},
	}
	src := &Response{
		Weather: map[string]Weather{
			"house-nick": {LocationID: "house-nick", LastUpdated: newerTS, TempF: 85.2}, // newer → overwrite
		},
		Pressure: map[string]Pressure{
			"house-nick": {LocationID: "house-nick", LastUpdated: newerTS, Trend: "rising"}, // newer → overwrite
		},
		// pollen empty in src — house-nita's pollen should remain
	}

	dst.Merge(src)

	if got := dst.Weather["house-nick"].TempF; got != 85.2 {
		t.Errorf("Weather[house-nick].TempF = %v, want overwritten 85.2", got)
	}
	if got := dst.Weather["house-nita"].TempF; got != 71.0 {
		t.Errorf("Weather[house-nita].TempF = %v, want preserved 71.0", got)
	}
	if got := dst.Pressure["house-nick"].Trend; got != "rising" {
		t.Errorf("Pressure[house-nick].Trend = %q, want overwritten 'rising'", got)
	}
	if _, ok := dst.Pollen["house-nita"]; !ok {
		t.Error("Pollen[house-nita] should be preserved when src has no pollen")
	}
}

// TestResponseMerge_LWW_OlderSrcLoses locks in the load-bearing property:
// a stale single-location response that arrives after a fresher full
// response must NOT clobber the fresher data. This is the M1 race-window
// fix from the issue-60 review.
func TestResponseMerge_LWW_OlderSrcLoses(t *testing.T) {
	const freshTS = "2026-04-13T14:30:05Z"
	const staleTS = "2026-04-13T13:00:00Z"
	const freshPollenTS = "2026-04-13T06:00:00Z"
	const stalePollenTS = "2026-04-12T06:00:00Z"

	dst := &Response{
		Weather: map[string]Weather{
			"house-nick": {LocationID: "house-nick", LastUpdated: freshTS, TempF: 85.2},
		},
		Pressure: map[string]Pressure{
			"house-nick": {LocationID: "house-nick", LastUpdated: freshTS, Trend: "rising"},
		},
		Pollen: map[string]Pollen{
			"house-nick": {LocationID: "house-nick", CollectedAt: freshPollenTS, OverallIndex: 4},
		},
	}
	stale := &Response{
		Weather: map[string]Weather{
			"house-nick": {LocationID: "house-nick", LastUpdated: staleTS, TempF: 70.0},
		},
		Pressure: map[string]Pressure{
			"house-nick": {LocationID: "house-nick", LastUpdated: staleTS, Trend: "steady"},
		},
		Pollen: map[string]Pollen{
			"house-nick": {LocationID: "house-nick", CollectedAt: stalePollenTS, OverallIndex: 1},
		},
	}

	dst.Merge(stale)

	if got := dst.Weather["house-nick"].TempF; got != 85.2 {
		t.Errorf("Weather[house-nick].TempF = %v, want preserved 85.2 (stale src must not overwrite)", got)
	}
	if got := dst.Pressure["house-nick"].Trend; got != "rising" {
		t.Errorf("Pressure[house-nick].Trend = %q, want preserved 'rising'", got)
	}
	if got := dst.Pollen["house-nick"].OverallIndex; got != 4 {
		t.Errorf("Pollen[house-nick].OverallIndex = %v, want preserved 4", got)
	}
}

// TestResponseMerge_LWW_EqualTimestampSkips covers the steady-state case
// today: weather upstream updates hourly, so two fetches in the same hour
// return identical timestamps. Equal-timestamp merges should be no-ops
// (we use > rather than >=) — keeps the property "merge is idempotent
// for byte-equal inputs."
func TestResponseMerge_LWW_EqualTimestampSkips(t *testing.T) {
	const ts = "2026-04-13T14:30:05Z"
	dst := &Response{
		Weather: map[string]Weather{
			"house-nick": {LocationID: "house-nick", LastUpdated: ts, TempF: 85.2},
		},
	}
	src := &Response{
		Weather: map[string]Weather{
			// Same timestamp, different value (shouldn't happen in practice,
			// but proves the equal-timestamp branch).
			"house-nick": {LocationID: "house-nick", LastUpdated: ts, TempF: 99.0},
		},
	}
	dst.Merge(src)
	if got := dst.Weather["house-nick"].TempF; got != 85.2 {
		t.Errorf("Weather[house-nick].TempF = %v, want preserved 85.2 on equal timestamps", got)
	}
}

func TestResponseMerge_NilGuards(t *testing.T) {
	// nil receiver is a no-op (would crash otherwise).
	var nilDst *Response
	nilDst.Merge(&Response{})

	// nil src is a no-op.
	dst := &Response{Weather: map[string]Weather{"x": {TempF: 1}}}
	dst.Merge(nil)
	if dst.Weather["x"].TempF != 1 {
		t.Error("Merge(nil) mutated dst")
	}

	// nil maps in dst should be initialised by Merge before writing.
	empty := &Response{}
	empty.Merge(&Response{
		Weather:  map[string]Weather{"x": {TempF: 9}},
		Pressure: map[string]Pressure{"x": {Trend: "rising"}},
		Pollen:   map[string]Pollen{"x": {OverallIndex: 3}},
	})
	if empty.Weather["x"].TempF != 9 {
		t.Error("Merge into nil Weather map failed")
	}
	if empty.Pressure["x"].Trend != "rising" {
		t.Error("Merge into nil Pressure map failed")
	}
	if empty.Pollen["x"].OverallIndex != 3 {
		t.Error("Merge into nil Pollen map failed")
	}
}
