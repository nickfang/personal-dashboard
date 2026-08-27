package shared

import (
	"math"
)

func RoundFloat64(val float64, places int) float64 {
	rounder := math.Pow(10, float64(places))
	return math.Round(val*rounder) / rounder
}
