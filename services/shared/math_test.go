package shared

import (
	"math"
	"testing"
)

func TestRoundFloat64(t *testing.T) {
	tests := []struct {
		name   string
		val    float64
		places int
		want   float64
	}{
		// The case this helper exists for: subtracting two nearby pressure
		// readings leaves binary-representation noise that would otherwise
		// reach the log line as 15 digits.
		{"strips subtraction noise", 1013.2 - 1019.4, 2, -6.2},
		{"already rounded is unchanged", -6.2, 2, -6.2},
		{"zero", 0, 2, 0},
		{"keeps precision below the cutoff", 1.23456789, 8, 1.23456789},

		// math.Round breaks halfway cases away from zero, in both directions.
		{"half rounds away from zero", 2.5, 0, 3},
		{"negative half rounds away from zero", -2.5, 0, -3},

		// Whether a decimal ".xx5" rounds up depends on which side of the
		// halfway point its binary representation lands on, so the two cases
		// below disagree despite looking identical. This is inherent to
		// scale-round-unscale and is documented, not a defect.
		{"decimal half below the binary midpoint rounds down", 1.005, 2, 1},
		{"decimal half above the binary midpoint rounds up", 2.675, 2, 2.68},

		// Negative places round to the left of the decimal point.
		{"negative places rounds to tens", 1234.5678, -1, 1230},
		{"negative places rounds to hundreds", 1234.5678, -2, 1200},

		{"positive infinity", math.Inf(1), 2, math.Inf(1)},
		{"negative infinity", math.Inf(-1), 2, math.Inf(-1)},
		// Scaling overflows before rounding can happen.
		{"overflows to infinity", math.MaxFloat64, 2, math.Inf(1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RoundFloat64(tt.val, tt.places); got != tt.want {
				t.Errorf("RoundFloat64(%v, %d) = %v, want %v", tt.val, tt.places, got, tt.want)
			}
		})
	}
}

func TestRoundFloat64_NaN(t *testing.T) {
	// NaN is never equal to itself, so it cannot be checked in the table.
	if got := RoundFloat64(math.NaN(), 2); !math.IsNaN(got) {
		t.Errorf("RoundFloat64(NaN, 2) = %v, want NaN", got)
	}
}

func TestRoundFloat64_SmallNegativeYieldsNegativeZero(t *testing.T) {
	// A value that rounds to zero from below keeps its sign bit, which a
	// table check would miss because -0 == 0. It matters because slog renders
	// the result as "-0" in the JSON log line.
	got := RoundFloat64(-0.001, 2)
	if got != 0 || !math.Signbit(got) {
		t.Errorf("RoundFloat64(-0.001, 2) = %v (signbit %v), want negative zero", got, math.Signbit(got))
	}
}
