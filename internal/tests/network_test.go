package tests

import (
	"fmt"
	"math"
	"testing"

	"performance-assessment-system/pkg/logger"
)

func newTestNetworkTest(t *testing.T) *NetworkTest {
	t.Helper()

	log, err := logger.NewLogger(t.TempDir(), "error")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	return NewNetworkTest(log)
}

func TestUploadSpeedReturnsEstimateFromDownload(t *testing.T) {
	networkTest := newTestNetworkTest(t)

	uploadSpeed, estimated, err := networkTest.TestUploadSpeed(200.0)
	if err != nil {
		t.Fatalf("expected estimated upload speed, got error: %v", err)
	}
	if !estimated {
		t.Fatal("expected upload speed to be marked as estimated")
	}
	if uploadSpeed != 140.0 {
		t.Fatalf("expected estimated upload speed 140.0, got %.2f", uploadSpeed)
	}
}

func TestUploadSpeedFailsWithoutDownloadBaseline(t *testing.T) {
	networkTest := newTestNetworkTest(t)

	_, estimated, err := networkTest.TestUploadSpeed(0)
	if err == nil {
		t.Fatal("expected missing download baseline to return error")
	}
	if estimated {
		t.Fatal("expected estimated flag to be false on failure")
	}
}

func TestCalculateScoreNormalizesWhenUploadIsEstimated(t *testing.T) {
	networkTest := newTestNetworkTest(t)

	score := networkTest.calculateScore(25.0, 200.0, 140.0, true)
	if score != 100.0 {
		t.Fatalf("expected estimated upload path to normalize score to 100, got %.2f", score)
	}
}

func TestCalculateScoreUsesRealUploadWeight(t *testing.T) {
	networkTest := newTestNetworkTest(t)

	score := networkTest.calculateScore(50.0, 100.0, 25.0, false)
	expected := 90.0
	if math.Abs(score-expected) > 0.0001 {
		t.Fatalf("expected score %.2f, got %.2f", expected, score)
	}
}

func TestExecutePopulatesSourceMetricsForEstimatedUpload(t *testing.T) {
	networkTest := newTestNetworkTest(t)
	networkTest.latencyFn = func(_ []string) (float64, error) {
		return 12.5, nil
	}
	networkTest.downloadFn = func() (float64, error) {
		return 200.0, nil
	}
	networkTest.uploadFn = func(downloadSpeed float64) (float64, bool, error) {
		if downloadSpeed != 200.0 {
			t.Fatalf("expected download speed to be passed into upload function, got %.2f", downloadSpeed)
		}
		return 140.0, true, nil
	}

	result, err := networkTest.Execute()
	if err != nil {
		t.Fatalf("expected execute to succeed, got error: %v", err)
	}

	assertMetricFloat(t, result.Metrics, "average_latency_ms", 12.5)
	assertMetricFloat(t, result.Metrics, "download_speed_mbps", 200.0)
	assertMetricFloat(t, result.Metrics, "upload_speed_mbps", 140.0)
	assertMetricString(t, result.Metrics, "latency_source", "tcp_connect")
	assertMetricString(t, result.Metrics, "download_speed_source", "http_download")
	assertMetricString(t, result.Metrics, "upload_speed_source", "estimated_from_download")
	assertMetricBool(t, result.Metrics, "upload_speed_estimated", true)

	score, ok := result.Metrics["score"].(float64)
	if !ok {
		t.Fatal("expected score metric to be present")
	}
	if score != 100.0 {
		t.Fatalf("expected normalized score 100.0, got %.2f", score)
	}
}

func TestExecuteMarksFailedDownloadAndUnavailableUpload(t *testing.T) {
	networkTest := newTestNetworkTest(t)
	networkTest.latencyFn = func(_ []string) (float64, error) {
		return 0, fmt.Errorf("latency unavailable")
	}
	networkTest.downloadFn = func() (float64, error) {
		return 0, fmt.Errorf("download unavailable")
	}
	networkTest.uploadFn = func(downloadSpeed float64) (float64, bool, error) {
		if downloadSpeed != 0.0 {
			t.Fatalf("expected failed download to pass 0.0 into upload function, got %.2f", downloadSpeed)
		}
		return 0, false, fmt.Errorf("upload unavailable")
	}

	result, err := networkTest.Execute()
	if err != nil {
		t.Fatalf("expected execute to still succeed, got error: %v", err)
	}

	assertMetricFloat(t, result.Metrics, "average_latency_ms", -1.0)
	assertMetricFloat(t, result.Metrics, "download_speed_mbps", -1.0)
	assertMetricFloat(t, result.Metrics, "upload_speed_mbps", -1.0)
	assertMetricString(t, result.Metrics, "download_speed_source", "http_download_failed")
	assertMetricString(t, result.Metrics, "upload_speed_source", "unavailable")
	assertMetricBool(t, result.Metrics, "upload_speed_estimated", false)

	score, ok := result.Metrics["score"].(float64)
	if !ok {
		t.Fatal("expected score metric to be present")
	}
	if score != 0.0 {
		t.Fatalf("expected score 0.0 when latency and download fail, got %.2f", score)
	}
}

func assertMetricFloat(t *testing.T, metrics map[string]interface{}, key string, expected float64) {
	t.Helper()

	value, ok := metrics[key].(float64)
	if !ok {
		t.Fatalf("expected metric %s to be float64, got %T", key, metrics[key])
	}
	if math.Abs(value-expected) > 0.0001 {
		t.Fatalf("expected metric %s to be %.2f, got %.2f", key, expected, value)
	}
}

func assertMetricString(t *testing.T, metrics map[string]interface{}, key string, expected string) {
	t.Helper()

	value, ok := metrics[key].(string)
	if !ok {
		t.Fatalf("expected metric %s to be string, got %T", key, metrics[key])
	}
	if value != expected {
		t.Fatalf("expected metric %s to be %q, got %q", key, expected, value)
	}
}

func assertMetricBool(t *testing.T, metrics map[string]interface{}, key string, expected bool) {
	t.Helper()

	value, ok := metrics[key].(bool)
	if !ok {
		t.Fatalf("expected metric %s to be bool, got %T", key, metrics[key])
	}
	if value != expected {
		t.Fatalf("expected metric %s to be %t, got %t", key, expected, value)
	}
}
