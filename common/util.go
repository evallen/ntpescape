package common

import (
	"math/rand"
)

// Generate a random float64 between two floats `low` and `high`.
func RandF64InRange(low float64, high float64) float64 {
	return low + rand.Float64() * (high - low)
}