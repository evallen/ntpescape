package common

import (
	"math/rand"
)

func RandF64InRange(low float64, high float64) float64 {
	return low + rand.Float64() * (high - low)
}