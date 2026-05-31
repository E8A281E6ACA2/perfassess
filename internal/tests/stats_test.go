package tests

import (
	"math"
	"testing"
)

func TestCalculateSampleStats(t *testing.T) {
	stats := calculateSampleStats([]float64{30, 10, 20})

	if stats.Count != 3 {
		t.Fatalf("expected count 3, got %d", stats.Count)
	}
	if stats.Min != 10 {
		t.Fatalf("expected min 10, got %.2f", stats.Min)
	}
	if stats.Median != 20 {
		t.Fatalf("expected median 20, got %.2f", stats.Median)
	}
	if stats.Max != 30 {
		t.Fatalf("expected max 30, got %.2f", stats.Max)
	}
	expectedStdDev := math.Sqrt(200.0 / 3.0)
	if math.Abs(stats.StdDev-expectedStdDev) > 0.0001 {
		t.Fatalf("expected stddev %.4f, got %.4f", expectedStdDev, stats.StdDev)
	}
}

func TestCalculateSampleStatsEvenCount(t *testing.T) {
	stats := calculateSampleStats([]float64{40, 10, 30, 20})

	if stats.Median != 25 {
		t.Fatalf("expected median 25, got %.2f", stats.Median)
	}
}

func TestAddSampleStatsMetrics(t *testing.T) {
	metrics := map[string]interface{}{}
	addSampleStatsMetrics(metrics, "read_speed_mbps", sampleStats{
		Count:  3,
		Min:    100,
		Median: 120,
		Max:    140,
		StdDev: 16.3,
	})

	assertMetricInt(t, metrics, "read_speed_mbps_samples", 3)
	assertMetricFloat(t, metrics, "read_speed_mbps_min", 100)
	assertMetricFloat(t, metrics, "read_speed_mbps_median", 120)
	assertMetricFloat(t, metrics, "read_speed_mbps_max", 140)
	assertMetricFloat(t, metrics, "read_speed_mbps_stddev", 16.3)
}
