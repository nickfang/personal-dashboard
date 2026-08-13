package repository

import (
	"testing"
	"time"

	"github.com/nickfang/personal-dashboard/services/shared"
)

func TestBuildCacheDoc(t *testing.T) {
	issuedAt := time.Date(2026, 6, 12, 6, 0, 0, 0, time.UTC)
	run := ForecastRun{
		Location: "house-nick",
		IssuedAt: issuedAt,
		Points: []ForecastPoint{
			{ValidTime: issuedAt, PressureMb: 1012.65},
			{ValidTime: issuedAt.Add(time.Hour), PressureMb: 1013.13},
		},
	}
	alerts := []shared.Alert{
		{ID: "alert-1", Location: "house-nick", Status: shared.AlertStatusActive},
	}

	doc := buildCacheDoc(run, alerts)

	if doc.Location != "house-nick" {
		t.Errorf("Location = %q, want %q", doc.Location, "house-nick")
	}
	if !doc.IssuedAt.Equal(issuedAt) {
		t.Errorf("IssuedAt = %v, want %v", doc.IssuedAt, issuedAt)
	}
	if len(doc.Points) != 2 {
		t.Fatalf("len(Points) = %d, want 2", len(doc.Points))
	}
	if doc.Points[1].PressureMb != 1013.13 {
		t.Errorf("Points[1].PressureMb = %v, want 1013.13", doc.Points[1].PressureMb)
	}
	if len(doc.Alerts) != 1 || doc.Alerts[0].ID != "alert-1" {
		t.Errorf("Alerts = %v, want the merged alert set", doc.Alerts)
	}
}

func TestApplyNotifiedAt(t *testing.T) {
	at := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	earlier := time.Date(2026, 6, 12, 6, 0, 0, 0, time.UTC)
	stored := []shared.Alert{
		{ID: "delivered-now", Status: shared.AlertStatusActive},
		{ID: "untouched", Status: shared.AlertStatusActive},
		{ID: "delivered-before", Status: shared.AlertStatusActive, NotifiedAt: earlier},
	}

	got := applyNotifiedAt(stored, []string{"delivered-now"}, at)

	if len(got) != 3 {
		t.Fatalf("len = %d, want 3 — the whole set is rewritten, not just the marked ones", len(got))
	}
	if !got[0].NotifiedAt.Equal(at) {
		t.Errorf("delivered-now NotifiedAt = %v, want %v", got[0].NotifiedAt, at)
	}
	if !got[1].NotifiedAt.IsZero() {
		t.Errorf("untouched NotifiedAt = %v, want zero", got[1].NotifiedAt)
	}
	if !got[2].NotifiedAt.Equal(earlier) {
		t.Errorf("delivered-before NotifiedAt = %v, want its stored %v", got[2].NotifiedAt, earlier)
	}
	if got[0].Status != shared.AlertStatusActive {
		t.Errorf("Status = %q, want active — delivery does not change whether the condition is present", got[0].Status)
	}
	if !stored[0].NotifiedAt.IsZero() {
		t.Error("applyNotifiedAt mutated its input")
	}
}

func TestApplyNotifiedAt_UnknownIDIsIgnored(t *testing.T) {
	at := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	stored := []shared.Alert{{ID: "a", Status: shared.AlertStatusActive}}

	// A concurrent run can replace the alert set between delivery and
	// marking; matching by ID means the stale ID is simply dropped.
	got := applyNotifiedAt(stored, []string{"gone"}, at)

	if len(got) != 1 || !got[0].NotifiedAt.IsZero() {
		t.Errorf("got %v, want the stored set unchanged", got)
	}
}
