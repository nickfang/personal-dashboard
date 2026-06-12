package repository

import (
	"testing"
	"time"
)

func TestBuildCacheDoc(t *testing.T) {
	issuedAt := time.Date(2026, 6, 12, 6, 0, 0, 0, time.UTC)
	run := ForecastRun{
		Location:     "house-nick",
		IssuedAt:     issuedAt,
		HorizonHours: 72,
		Points: []ForecastPoint{
			{ValidTime: issuedAt, PressureMb: 1012.65},
			{ValidTime: issuedAt.Add(time.Hour), PressureMb: 1013.13},
		},
	}

	doc := buildCacheDoc(run)

	if doc.Location != "house-nick" {
		t.Errorf("Location = %q, want %q", doc.Location, "house-nick")
	}
	if !doc.IssuedAt.Equal(issuedAt) {
		t.Errorf("IssuedAt = %v, want %v", doc.IssuedAt, issuedAt)
	}
	if !doc.LastUpdated.Equal(issuedAt) {
		t.Errorf("LastUpdated = %v, want %v", doc.LastUpdated, issuedAt)
	}
	if len(doc.Points) != 2 {
		t.Fatalf("len(Points) = %d, want 2", len(doc.Points))
	}
	if doc.Points[1].PressureMb != 1013.13 {
		t.Errorf("Points[1].PressureMb = %v, want 1013.13", doc.Points[1].PressureMb)
	}
}
