package service

// CtoF converts Celsius to Fahrenheit.
func CtoF(c float64) float64 {
	return (c * 1.8) + 32
}

// KtoM converts kilometers per hour to miles per hour.
func KtoM(k float64) float64 {
	return k * 0.621371
}
