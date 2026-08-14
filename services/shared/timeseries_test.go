package shared

import (
	"testing"
	"time"
)

type stamped struct {
	at    time.Time
	label string
}

func stampedAt(s stamped) time.Time { return s.at }

func series(base time.Time, offsets ...time.Duration) []stamped {
	out := make([]stamped, len(offsets))
	for i, o := range offsets {
		out[i] = stamped{at: base.Add(o), label: o.String()}
	}
	return out
}

func TestNearestIndex(t *testing.T) {
	base := time.Date(2026, 6, 12, 6, 0, 0, 0, time.UTC)
	hourly := series(base, 0, time.Hour, 2*time.Hour, 3*time.Hour)
	// A missing hour: nothing sits near +2h30m, which is the only way to be
	// outside a 30-minute tolerance when points are an hour apart.
	gapped := series(base, 0, time.Hour, 4*time.Hour)

	tests := []struct {
		name      string
		items     []stamped
		target    time.Time
		tolerance time.Duration
		want      int
	}{
		{"exact match", hourly, base.Add(2 * time.Hour), 30 * time.Minute, 2},
		{"within tolerance", hourly, base.Add(2*time.Hour + 20*time.Minute), 30 * time.Minute, 2},
		{"exactly at tolerance", hourly, base.Add(2*time.Hour + 30*time.Minute), 30 * time.Minute, 2},
		{"outside tolerance", gapped, base.Add(2*time.Hour + 30*time.Minute), 30 * time.Minute, -1},
		{"reaches across a missing hour when tolerance allows", gapped, base.Add(90 * time.Minute), 45 * time.Minute, 1},
		{"picks closest of two in range", hourly, base.Add(time.Hour + 40*time.Minute), 45 * time.Minute, 2},
		{"empty slice", nil, base, time.Hour, -1},
		{"target before all items", hourly, base.Add(-2 * time.Hour), 30 * time.Minute, -1},
		{"target after all items", hourly, base.Add(9 * time.Hour), 30 * time.Minute, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NearestIndex(tt.items, stampedAt, tt.target, tt.tolerance); got != tt.want {
				t.Errorf("NearestIndex() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNearestIndex_TieResolvesToEarlier(t *testing.T) {
	// Equidistant on both sides. Documented behavior is the earlier item —
	// weather-collector's getDelta picks the later one, which is why that
	// call site is not refactored onto this helper.
	base := time.Date(2026, 6, 12, 6, 0, 0, 0, time.UTC)
	items := series(base, 0, 2*time.Hour)
	target := base.Add(time.Hour)

	if got := NearestIndex(items, stampedAt, target, 90*time.Minute); got != 0 {
		t.Errorf("NearestIndex() = %d, want 0 (earlier item on a tie)", got)
	}
}

func TestNearestIndex_StopsScanningPastTheWindow(t *testing.T) {
	// The ordering precondition is what makes the early break safe; this
	// pins that the break actually happens rather than the loop running long.
	base := time.Date(2026, 6, 12, 6, 0, 0, 0, time.UTC)
	items := series(base, 0, time.Hour, 2*time.Hour, 3*time.Hour, 4*time.Hour, 5*time.Hour)

	calls := 0
	counting := func(s stamped) time.Time {
		calls++
		return s.at
	}

	if got := NearestIndex(items, counting, base.Add(time.Hour), 30*time.Minute); got != 1 {
		t.Fatalf("NearestIndex() = %d, want 1", got)
	}
	// Reaching index 2 (+2h) proves the target was passed; anything beyond
	// means the break never fired.
	if calls > 3 {
		t.Errorf("accessor called %d times, want at most 3 — the scan did not break past the window", calls)
	}
}
