package biz

import (
	"math"
)

// NormalizeClose normalizes the close price to a value between 0 and 1 given min and max.
func NormalizeClose(close, min, max float32) float32 {
	if max == min {
		return 0
	}
	return (close - min) / (max - min)
}

// StandardizeVector standardizes a vector to zero mean and unit variance.
func StandardizeVector(vec []float32) []float32 {
	n := float32(len(vec))
	if n == 0 {
		return nil
	}
	var sum float32
	for _, v := range vec {
		sum += v
	}
	mean := sum / n
	var sqSum float32
	for _, v := range vec {
		d := v - mean
		sqSum += d * d
	}
	std := float32(math.Sqrt(float64(sqSum / n)))
	if std == 0 {
		return make([]float32, len(vec))
	}
	res := make([]float32, len(vec))
	for i, v := range vec {
		res[i] = (v - mean) / std
	}
	return res
}
