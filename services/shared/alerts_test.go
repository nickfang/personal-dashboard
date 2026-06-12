package shared

import (
	"testing"
	"time"
)

var baseTime = time.Date(2026, 6, 12, 6, 0, 0, 0, time.UTC)

func pressureAlert(windowStartHours, windowEndHours int, value float64, status string) Alert {
	a := Alert{
		Location:    "house-nick",
		RuleID:      "pressure-drop-3h",
		Severity:    AlertSeverityWarning,
		Value:       value,
		Threshold:   5,
		WindowStart: baseTime.Add(time.Duration(windowStartHours) * time.Hour),
		WindowEnd:   baseTime.Add(time.Duration(windowEndHours) * time.Hour),
		Status:      status,
		IssuedAt:    baseTime,
	}
	a.ID = a.ComputeID()
	return a
}

func TestComputeID_Stable(t *testing.T) {
	a := pressureAlert(2, 5, -6.0, AlertStatusActive)
	b := pressureAlert(2, 5, -8.5, AlertStatusResolved)
	// Value, status, and window end don't affect identity.
	b.WindowEnd = baseTime.Add(9 * time.Hour)
	if a.ComputeID() != b.ComputeID() {
		t.Error("ComputeID should be stable for same location/rule/windowStart")
	}

	// Sub-hour differences in windowStart map to the same identity.
	c := pressureAlert(2, 5, -6.0, AlertStatusActive)
	c.WindowStart = c.WindowStart.Add(30 * time.Minute)
	if a.ComputeID() != c.ComputeID() {
		t.Error("ComputeID should truncate windowStart to the hour")
	}
}

func TestComputeID_DiffersByLocationRuleAndHour(t *testing.T) {
	a := pressureAlert(2, 5, -6.0, AlertStatusActive)

	byLocation := pressureAlert(2, 5, -6.0, AlertStatusActive)
	byLocation.Location = "house-nita"
	if a.ComputeID() == byLocation.ComputeID() {
		t.Error("ComputeID should differ by location")
	}

	byRule := pressureAlert(2, 5, -6.0, AlertStatusActive)
	byRule.RuleID = "juniper-high"
	if a.ComputeID() == byRule.ComputeID() {
		t.Error("ComputeID should differ by rule")
	}

	byHour := pressureAlert(3, 6, -6.0, AlertStatusActive)
	if a.ComputeID() == byHour.ComputeID() {
		t.Error("ComputeID should differ by window start hour")
	}
}

func TestMergeAlerts_NewAlertStaysActive(t *testing.T) {
	d := pressureAlert(2, 5, -6.0, AlertStatusActive)

	got := MergeAlerts(nil, []Alert{d}, baseTime)

	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Status != AlertStatusActive {
		t.Errorf("Status = %q, want active", got[0].Status)
	}
	if got[0].ID != d.ID {
		t.Errorf("ID = %q, want detector-computed %q", got[0].ID, d.ID)
	}
}

func TestMergeAlerts_OverlapKeepsOriginalIDAndUpdatesNumbers(t *testing.T) {
	prev := pressureAlert(2, 5, -6.0, AlertStatusActive)
	// Next run: same episode shifted and deepened.
	detected := pressureAlert(3, 7, -7.5, AlertStatusActive)

	got := MergeAlerts([]Alert{prev}, []Alert{detected}, baseTime)

	if len(got) != 1 {
		t.Fatalf("len = %d, want 1 (episodes should merge)", len(got))
	}
	if got[0].ID != prev.ID {
		t.Errorf("ID = %q, want original %q", got[0].ID, prev.ID)
	}
	if got[0].Value != -7.5 {
		t.Errorf("Value = %v, want updated -7.5", got[0].Value)
	}
	if !got[0].WindowStart.Equal(detected.WindowStart) || !got[0].WindowEnd.Equal(detected.WindowEnd) {
		t.Error("window should come from the new detection")
	}
	if got[0].Status != AlertStatusActive {
		t.Errorf("Status = %q, want active", got[0].Status)
	}
}

func TestMergeAlerts_NotifiedStaysNotifiedWhenUnchanged(t *testing.T) {
	prev := pressureAlert(2, 5, -6.0, AlertStatusNotified)
	detected := pressureAlert(2, 5, -6.4, AlertStatusActive) // within the 1 mb step

	got := MergeAlerts([]Alert{prev}, []Alert{detected}, baseTime)

	if len(got) != 1 || got[0].Status != AlertStatusNotified {
		t.Fatalf("Status = %v, want notified (no meaningful worsening)", got)
	}
}

func TestMergeAlerts_NotifiedReactivatesWhenWorsened(t *testing.T) {
	prev := pressureAlert(2, 5, -6.0, AlertStatusNotified)
	detected := pressureAlert(2, 5, -7.2, AlertStatusActive) // worsened ≥1 mb

	got := MergeAlerts([]Alert{prev}, []Alert{detected}, baseTime)

	if len(got) != 1 || got[0].Status != AlertStatusActive {
		t.Fatalf("Status = %v, want active (worsened past escalation step)", got)
	}
}

func TestMergeAlerts_NotifiedReactivatesOnSeverityUpgrade(t *testing.T) {
	prev := pressureAlert(2, 5, -9.5, AlertStatusNotified)
	detected := pressureAlert(2, 5, -10.2, AlertStatusActive)
	detected.Severity = AlertSeveritySevere

	got := MergeAlerts([]Alert{prev}, []Alert{detected}, baseTime)

	if len(got) != 1 || got[0].Status != AlertStatusActive {
		t.Fatalf("Status = %v, want active (warning→severe upgrade)", got)
	}
}

func TestMergeAlerts_ResolvedReactivatesOnRedetection(t *testing.T) {
	prev := pressureAlert(2, 5, -6.0, AlertStatusResolved)
	detected := pressureAlert(2, 5, -5.5, AlertStatusActive)

	got := MergeAlerts([]Alert{prev}, []Alert{detected}, baseTime)

	if len(got) != 1 || got[0].Status != AlertStatusActive {
		t.Fatalf("Status = %v, want active (drop is back)", got)
	}
	if got[0].ID != prev.ID {
		t.Errorf("ID = %q, want original %q", got[0].ID, prev.ID)
	}
}

func TestMergeAlerts_UndetectedFutureAlertResolves(t *testing.T) {
	prev := pressureAlert(2, 5, -6.0, AlertStatusActive)

	got := MergeAlerts([]Alert{prev}, nil, baseTime)

	if len(got) != 1 || got[0].Status != AlertStatusResolved {
		t.Fatalf("got %v, want the alert kept as resolved", got)
	}
}

func TestMergeAlerts_PastWindowPruned(t *testing.T) {
	past := pressureAlert(-6, -3, -6.0, AlertStatusResolved)
	current := pressureAlert(2, 5, -6.0, AlertStatusActive)

	got := MergeAlerts([]Alert{past, current}, nil, baseTime)

	if len(got) != 1 {
		t.Fatalf("len = %d, want 1 (past alert pruned)", len(got))
	}
	if got[0].ID != current.ID {
		t.Errorf("surviving alert = %q, want %q", got[0].ID, current.ID)
	}
}

func TestMergeAlerts_GreatestOverlapWins(t *testing.T) {
	early := pressureAlert(0, 4, -5.5, AlertStatusNotified)
	late := pressureAlert(4, 9, -6.0, AlertStatusActive)
	// One detected episode overlapping both: 1h of `early`, 5h of `late`.
	detected := pressureAlert(3, 9, -7.0, AlertStatusActive)

	got := MergeAlerts([]Alert{early, late}, []Alert{detected}, baseTime)

	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (merged episode + resolved leftover)", len(got))
	}
	// Sorted by WindowStart: early (resolved) first, merged second.
	if got[0].ID != early.ID || got[0].Status != AlertStatusResolved {
		t.Errorf("leftover = %q/%q, want early alert resolved", got[0].ID, got[0].Status)
	}
	if got[1].ID != late.ID {
		t.Errorf("merged ID = %q, want %q (greatest overlap)", got[1].ID, late.ID)
	}
}

func TestMergeAlerts_DifferentRuleOrLocationNeverMatches(t *testing.T) {
	prev := pressureAlert(2, 5, -6.0, AlertStatusNotified)
	otherLocation := pressureAlert(2, 5, -6.0, AlertStatusActive)
	otherLocation.Location = "house-nita"
	otherLocation.ID = otherLocation.ComputeID()

	got := MergeAlerts([]Alert{prev}, []Alert{otherLocation}, baseTime)

	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (no cross-location merge)", len(got))
	}
}
