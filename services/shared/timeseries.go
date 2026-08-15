package shared

import "time"

// NearestIndex returns the index of the item whose time is closest to target
// within tolerance, or -1 when nothing qualifies.
//
// Items must be ordered ascending by time. The scan stops once it is past the
// tolerance window, so the cost is proportional to the distance to target
// rather than to the length of the slice.
//
// The boundary is inclusive: an item exactly tolerance away from target
// qualifies. Ties resolve to the earlier item.
//
// Tolerance is a parameter rather than a constant because callers genuinely
// disagree: detection matches within 30 minutes so a window delta is measured
// over roughly the interval it claims, while display paths accept 45 minutes
// to survive a missing forecast hour.
//
// This deliberately does not yet replace the equivalent searches in
// weather-collector, forecast-collector, dashboard-api, or the CLI.
// weather-collector's getDelta scans descending and so resolves a tie to the
// *later* item, which is the opposite of this. Moving those onto this helper
// is a behavior decision that needs its own test, not a mechanical
// substitution — see issue #80.
func NearestIndex[T any](items []T, at func(T) time.Time, target time.Time, tolerance time.Duration) int {
	best := -1
	minDiff := tolerance + time.Nanosecond
	for i, item := range items {
		t := at(item)
		offset := t.Sub(target)
		diff := offset
		if diff < 0 {
			diff = -diff
		}
		if diff <= tolerance && diff < minDiff {
			minDiff = diff
			best = i
		}
		if offset > tolerance {
			break
		}
	}
	return best
}
