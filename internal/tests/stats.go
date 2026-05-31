package tests

import (
	"math"
	"sort"
)

type sampleStats struct {
	Count  int
	Min    float64
	Median float64
	Max    float64
	StdDev float64
}

func calculateSampleStats(samples []float64) sampleStats {
	if len(samples) == 0 {
		return sampleStats{}
	}

	sorted := append([]float64(nil), samples...)
	sort.Float64s(sorted)

	stats := sampleStats{
		Count: len(sorted),
		Min:   sorted[0],
		Max:   sorted[len(sorted)-1],
	}
	if len(sorted)%2 == 1 {
		stats.Median = sorted[len(sorted)/2]
	} else {
		mid := len(sorted) / 2
		stats.Median = (sorted[mid-1] + sorted[mid]) / 2
	}

	mean := 0.0
	for _, sample := range sorted {
		mean += sample
	}
	mean /= float64(len(sorted))

	variance := 0.0
	for _, sample := range sorted {
		diff := sample - mean
		variance += diff * diff
	}
	variance /= float64(len(sorted))
	stats.StdDev = math.Sqrt(variance)

	return stats
}

func addSampleStatsMetrics(metrics map[string]interface{}, prefix string, stats sampleStats) {
	metrics[prefix+"_samples"] = stats.Count
	metrics[prefix+"_min"] = stats.Min
	metrics[prefix+"_median"] = stats.Median
	metrics[prefix+"_max"] = stats.Max
	metrics[prefix+"_stddev"] = stats.StdDev
}
