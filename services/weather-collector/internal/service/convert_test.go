package service

import "testing"

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
