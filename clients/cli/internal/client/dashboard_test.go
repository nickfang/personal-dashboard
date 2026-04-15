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

	c := New(srv.URL)
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

	c := New(srv.URL)
	if _, err := c.Fetch(context.Background()); err == nil {
		t.Fatal("expected error on 500, got nil")
	}
}
