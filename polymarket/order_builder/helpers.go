package order_builder

import (
	"math"
	"strconv"
	"strings"
)

// RoundDown rounds x down to sigDigits decimal places.
func RoundDown(x float64, sigDigits int) float64 {
	return math.Floor(x*math.Pow(10, float64(sigDigits))) / math.Pow(10, float64(sigDigits))
}

// RoundNormal rounds x to sigDigits decimal places.
func RoundNormal(x float64, sigDigits int) float64 {
	return math.Round(x*math.Pow(10, float64(sigDigits))) / math.Pow(10, float64(sigDigits))
}

// RoundUp rounds x up to sigDigits decimal places.
func RoundUp(x float64, sigDigits int) float64 {
	return math.Ceil(x*math.Pow(10, float64(sigDigits))) / math.Pow(10, float64(sigDigits))
}

// ToTokenDecimals converts a float to token decimals (6 decimal places).
func ToTokenDecimals(x float64) int64 {
	f := (math.Pow(10, 6)) * x
	if DecimalPlaces(f) > 0 {
		f = RoundNormal(f, 0)
	}
	return int64(f)
}

// DecimalPlaces returns the number of decimal places in x.
func DecimalPlaces(x float64) int {
	s := strconv.FormatFloat(x, 'f', -1, 64)
	if strings.Contains(s, ".") {
		parts := strings.Split(s, ".")
		return len(parts[1])
	}
	return 0
}
