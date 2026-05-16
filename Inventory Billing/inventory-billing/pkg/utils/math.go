package utils

import "math"

// Round2 rounds a float64 to exactly 2 decimal places.
// Used for all monetary calculations to avoid floating-point drift.
func Round2(v float64) float64 {
	return math.Round(v*100) / 100
}
