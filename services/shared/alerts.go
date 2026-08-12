package shared

import (
	"crypto/sha1"
	"encoding/hex"
	"sort"
	"time"
)

const (
	AlertStatusActive   = "active"
	AlertStatusResolved = "resolved"

	AlertSeverityWarning = "warning"
	AlertSeveritySevere  = "severe"

	// AlertEscalationStepMb guards re-notification against forecast noise: an
	// already-delivered alert is only re-armed for delivery when its predicted
	// magnitude worsens by at least this much (or its severity upgrades).
	AlertEscalationStepMb = 1.0
)

// Alert is a source-agnostic record of a detected condition (pressure drop,
// pollen spike, ...). Detectors create them at write time; providers and
// clients only read them. Future sources reuse this struct unchanged.
//
// Status and NotifiedAt track two independent facts. Status says whether the
// condition is currently in the forecast; NotifiedAt says whether the user has
// been told about it. Collapsing them into one field would mean a delivered
// alert disappears from the dashboard while the drop is still hours away, and
// would lose the delivery record whenever forecast noise briefly resolves an
// alert that is still building.
type Alert struct {
	ID          string    `firestore:"id"`
	Location    string    `firestore:"location"`
	RuleID      string    `firestore:"rule_id"`
	Severity    string    `firestore:"severity"`
	Value       float64   `firestore:"value"`
	Threshold   float64   `firestore:"threshold"`
	WindowStart time.Time `firestore:"window_start"`
	WindowEnd   time.Time `firestore:"window_end"`
	Message     string    `firestore:"message"`
	Status      string    `firestore:"status"`
	IssuedAt    time.Time `firestore:"issued_at"`

	// NotifiedAt is when this alert was delivered to the user; the zero value
	// means it never was. Delivery is gated on Status == AlertStatusActive and
	// a zero NotifiedAt, so clearing it re-arms delivery.
	NotifiedAt time.Time `firestore:"notified_at"`
}

// ComputeID derives a stable identity for a new alert from its location,
// rule, and window start truncated to the hour. Merging preserves the
// original ID across runs even as the episode's window shifts.
func (a Alert) ComputeID() string {
	key := a.Location + "|" + a.RuleID + "|" + a.WindowStart.UTC().Truncate(time.Hour).Format(time.RFC3339)
	sum := sha1.Sum([]byte(key))
	return hex.EncodeToString(sum[:])
}

// overlaps reports whether two alerts describe the same kind of condition in
// intersecting time windows — the definition of "same episode" for merging.
func (a Alert) overlaps(b Alert) bool {
	return a.Location == b.Location && a.RuleID == b.RuleID &&
		a.WindowStart.Before(b.WindowEnd) && b.WindowStart.Before(a.WindowEnd)
}

func overlapDuration(a, b Alert) time.Duration {
	start := a.WindowStart
	if b.WindowStart.After(start) {
		start = b.WindowStart
	}
	end := a.WindowEnd
	if b.WindowEnd.Before(end) {
		end = b.WindowEnd
	}
	return end.Sub(start)
}

// MergeAlerts reconciles freshly detected alerts against the previously
// stored set:
//
//   - alerts whose window has fully passed are pruned,
//   - a detected alert overlapping a stored one keeps the stored ID and
//     delivery record, and updates its numbers,
//   - a delivered alert has its NotifiedAt cleared — re-arming delivery —
//     only when the prediction worsens or its severity upgrades,
//   - every detected alert is "active", whether or not it was previously
//     resolved,
//   - stored alerts no longer detected are marked "resolved" but keep their
//     delivery record, so forecast noise cannot cause a re-notification.
func MergeAlerts(prev, detected []Alert, now time.Time) []Alert {
	var live []Alert
	for _, p := range prev {
		if p.WindowEnd.After(now) {
			live = append(live, p)
		}
	}

	sorted := append([]Alert(nil), detected...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].WindowStart.Before(sorted[j].WindowStart) })

	consumed := make([]bool, len(live))
	result := make([]Alert, 0, len(sorted)+len(live))

	for _, d := range sorted {
		bestIdx := -1
		var bestOverlap time.Duration
		for i, p := range live {
			if consumed[i] || !d.overlaps(p) {
				continue
			}
			ov := overlapDuration(d, p)
			if bestIdx == -1 || ov > bestOverlap ||
				(ov == bestOverlap && p.WindowStart.Before(live[bestIdx].WindowStart)) {
				bestIdx, bestOverlap = i, ov
			}
		}
		if bestIdx == -1 {
			result = append(result, d)
			continue
		}
		consumed[bestIdx] = true
		merged := d
		merged.ID = live[bestIdx].ID
		merged.Status = AlertStatusActive
		merged.NotifiedAt = mergedNotifiedAt(live[bestIdx], d)
		result = append(result, merged)
	}

	for i, p := range live {
		if !consumed[i] {
			p.Status = AlertStatusResolved
			result = append(result, p)
		}
	}

	sort.Slice(result, func(i, j int) bool { return result[i].WindowStart.Before(result[j].WindowStart) })
	return result
}

// mergedNotifiedAt carries the stored delivery record onto a re-detected
// alert, clearing it — and so re-arming delivery — when the prediction has
// meaningfully worsened.
//
// This keys off the stored NotifiedAt rather than a status value so that an
// escalation is still caught when forecast noise has briefly resolved the
// alert in between. Keying off status would miss exactly that case, silently
// dropping a warning that has since upgraded to severe.
func mergedNotifiedAt(prev, detected Alert) time.Time {
	if prev.NotifiedAt.IsZero() {
		return time.Time{}
	}
	// Values are negative deltas; "worse" means more negative.
	worsened := detected.Value <= prev.Value-AlertEscalationStepMb
	upgraded := prev.Severity != AlertSeveritySevere && detected.Severity == AlertSeveritySevere
	if worsened || upgraded {
		return time.Time{}
	}
	return prev.NotifiedAt
}
