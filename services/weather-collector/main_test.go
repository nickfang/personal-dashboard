package main

import (
	"testing"
	"time"
)

func TestCtoF(t *testing.T) {
	tests := []struct {
		name     string
		celsius  float64
		expected float64
	}{
		{"Freezing", 0.0, 32.0},
		{"Boiling", 100.0, 212.0},
		{"Negative", -40.0, -40.0},
		{"Room Temp", 20.0, 68.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CtoF(tt.celsius); got != tt.expected {
				t.Errorf("CtoF(%f) = %f, want %f", tt.celsius, got, tt.expected)
			}
		})
	}
}

func TestKtoM(t *testing.T) {
	tests := []struct {
		name     string
		kph      float64
		expected float64
	}{
		{"Zero", 0.0, 0.0},
		{"100 kph", 100.0, 62.1371},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := KtoM(tt.kph)
			// Allow for small float point errors
			diff := got - tt.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.0001 {
				t.Errorf("KtoM(%f) = %f, want %f", tt.kph, got, tt.expected)
			}
		})
	}
}

func TestCalculatePressureStats(t *testing.T) {
	now := time.Now()
	
	// Helper to create a point at T minus hours
	mkPoint := func(hoursAgo int, pressure float64) PressurePoint {
		return PressurePoint{
			TimeStamp:  now.Add(time.Duration(-hoursAgo) * time.Hour),
			PressureMb: pressure,
		}
	}

	tests := []struct {
		name          string
		history       []PressurePoint
		wantTrend     string
		wantDelta3h   *float64
		wantDelta24h  *float64
	}{
		{
			name:      "Empty History",
			history:   []PressurePoint{},
			wantTrend: "stable",
		},
		{
			name:      "Single Point",
			history:   []PressurePoint{mkPoint(0, 1013.0)},
			wantTrend: "stable",
		},
		{
			name: "Stable Pressure",
			history: []PressurePoint{
				mkPoint(3, 1013.0),
				mkPoint(2, 1013.1),
				mkPoint(1, 1013.0),
				mkPoint(0, 1013.2), // Current
			},
			wantTrend:   "stable",
			wantDelta3h: floatPtr(0.2), // 1013.2 - 1013.0
		},
		{
			name: "Rising Pressure",
			history: []PressurePoint{
				mkPoint(4, 1010.0),
				mkPoint(3, 1011.0), // 3h ago
				mkPoint(2, 1012.0),
				mkPoint(1, 1013.0),
				mkPoint(0, 1014.0), // Current (1014 - 1011 = 3.0 increase)
			},
			wantTrend:   "rising",
			wantDelta3h: floatPtr(3.0),
		},
		{
			name: "Falling Pressure",
			history: []PressurePoint{
				mkPoint(3, 1020.0), // T-3h
				mkPoint(2, 1019.0), // T-2h
				mkPoint(0, 1018.0), // T-0h (Current)
			},
			wantTrend:   "falling",
			wantDelta3h: floatPtr(-2.0), // 1018 - 1020
		},
		{
			name: "Long History with Gap",
			history: []PressurePoint{
				mkPoint(24, 1000.0),
				mkPoint(12, 1005.0),
				mkPoint(0, 1010.0),
			},
			wantTrend:    "stable", 
			wantDelta24h: floatPtr(10.0), // Now matches because of timestamp logic!
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := calculatePressureStats(tt.history)
			
			if stats.Trend != tt.wantTrend {
				t.Errorf("Trend = %s, want %s", stats.Trend, tt.wantTrend)
			}

			if !compareFloatPtr(stats.Delta3h, tt.wantDelta3h) {
				t.Errorf("Delta3h = %v, want %v", stats.Delta3h, tt.wantDelta3h)
			}
			
			if !compareFloatPtr(stats.Delta24h, tt.wantDelta24h) {
				t.Errorf("Delta24h = %v, want %v", stats.Delta24h, tt.wantDelta24h)
			}
		})
	}
}

func floatPtr(f float64) *float64 {
	return &f
}

func compareFloatPtr(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return abs(*a - *b) < 0.001
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}