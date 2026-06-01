package tests

import (
	"context"
	"fmt"
	"math"
	"strings"
	"testing"

	"performance-assessment-system/internal/models"
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
	assertMetricString(t, result.Metrics, "backend", models.NetworkBackendBuiltin)
	assertMetricString(t, result.Metrics, "latency_source", models.NetworkLatencySourceTCPConnect)
	assertMetricString(t, result.Metrics, "download_speed_source", models.NetworkDownloadSourceHTTP)
	assertMetricString(t, result.Metrics, "upload_speed_source", models.NetworkUploadSourceEstimated)
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
	assertMetricString(t, result.Metrics, "backend", models.NetworkBackendBuiltin)
	assertMetricString(t, result.Metrics, "download_speed_source", models.NetworkDownloadSourceHTTPFailed)
	assertMetricString(t, result.Metrics, "upload_speed_source", models.NetworkUploadSourceUnavailable)
	assertMetricString(t, result.Metrics, "network_error", "延迟测试失败: latency unavailable; 下载速度测试失败: download unavailable; 上传速度测试失败: upload unavailable")
	assertMetricBool(t, result.Metrics, "upload_speed_estimated", false)

	score, ok := result.Metrics["score"].(float64)
	if !ok {
		t.Fatal("expected score metric to be present")
	}
	if score != 0.0 {
		t.Fatalf("expected score 0.0 when latency and download fail, got %.2f", score)
	}
}

func TestNetworkMetricsToMetricsMap(t *testing.T) {
	metrics := (&models.NetworkMetrics{
		Backend:           models.NetworkBackendBuiltin,
		BackendServer:     "127.0.0.1:5201",
		LatencyMs:         11.2,
		AverageLatencyMs:  11.2,
		LatencySource:     models.NetworkLatencySourceTCPConnect,
		DownloadSpeedMbps: 300.5,
		DownloadSource:    models.NetworkDownloadSourceHTTP,
		UploadSpeedMbps:   210.35,
		UploadSource:      models.NetworkUploadSourceEstimated,
		UploadEstimated:   true,
		ErrorMessage:      "iperf3 is not installed",
		Score:             95.0,
	}).ToMetricsMap()

	assertMetricString(t, metrics, "backend", models.NetworkBackendBuiltin)
	assertMetricString(t, metrics, "backend_server", "127.0.0.1:5201")
	assertMetricFloat(t, metrics, "latency_ms", 11.2)
	assertMetricFloat(t, metrics, "average_latency_ms", 11.2)
	assertMetricFloat(t, metrics, "download_speed_mbps", 300.5)
	assertMetricFloat(t, metrics, "upload_speed_mbps", 210.35)
	assertMetricFloat(t, metrics, "score", 95.0)
	assertMetricString(t, metrics, "latency_source", models.NetworkLatencySourceTCPConnect)
	assertMetricString(t, metrics, "download_speed_source", models.NetworkDownloadSourceHTTP)
	assertMetricString(t, metrics, "upload_speed_source", models.NetworkUploadSourceEstimated)
	assertMetricString(t, metrics, "network_error", "iperf3 is not installed")
	assertMetricBool(t, metrics, "upload_speed_estimated", true)
}

func TestIperf3NetworkBackendRequiresServer(t *testing.T) {
	backend := NewIperf3NetworkBackend(Iperf3Config{})

	if backend.Name() != models.NetworkBackendIperf3 {
		t.Fatalf("expected backend name %q, got %q", models.NetworkBackendIperf3, backend.Name())
	}
	if _, err := backend.MeasureDownload(); err == nil {
		t.Fatal("expected iperf3 download measurement to require server")
	}
	if _, _, err := backend.MeasureUpload(100); err == nil {
		t.Fatal("expected iperf3 upload measurement to require server")
	}
}

func TestIperf3NetworkBackendReportsMissingBinary(t *testing.T) {
	runner := &fakeCommandRunner{
		lookPathErr: fmt.Errorf("not found"),
	}
	backend := NewIperf3NetworkBackend(Iperf3Config{
		Server: "127.0.0.1:5201",
		Runner: runner,
	})

	_, err := backend.MeasureDownload()
	if err == nil {
		t.Fatal("expected missing iperf3 binary to fail")
	}
	if !strings.Contains(err.Error(), "sudo apt install iperf3") {
		t.Fatalf("expected install hint in error, got %v", err)
	}
}

func TestIperf3NetworkBackendParsesDownloadMbps(t *testing.T) {
	runner := &fakeCommandRunner{
		output:     []byte(`{"end":{"sum_received":{"bits_per_second":125000000}}}`),
		lookPathOK: true,
	}
	backend := NewIperf3NetworkBackend(Iperf3Config{
		Server: "127.0.0.1:5201",
		Runner: runner,
	})

	speed, err := backend.MeasureDownload()
	if err != nil {
		t.Fatalf("expected download measurement to parse, got error: %v", err)
	}
	if speed != 125.0 {
		t.Fatalf("expected 125 Mbps, got %.2f", speed)
	}
	if runner.name != "iperf3" {
		t.Fatalf("expected iperf3 command, got %q", runner.name)
	}
	assertStringSlice(t, runner.args, []string{"-c", "127.0.0.1", "-p", "5201", "--json"})
}

func TestIperf3NetworkBackendAllowsServerWithoutPort(t *testing.T) {
	runner := &fakeCommandRunner{
		output:     []byte(`{"end":{"sum_received":{"bits_per_second":125000000}}}`),
		lookPathOK: true,
	}
	backend := NewIperf3NetworkBackend(Iperf3Config{
		Server: "iperf.example",
		Runner: runner,
	})

	_, err := backend.MeasureDownload()
	if err != nil {
		t.Fatalf("expected server without port to use iperf3 default port, got error: %v", err)
	}
	assertStringSlice(t, runner.args, []string{"-c", "iperf.example", "--json"})
}

func TestIperf3NetworkBackendParsesBracketIPv6Server(t *testing.T) {
	runner := &fakeCommandRunner{
		output:     []byte(`{"end":{"sum_received":{"bits_per_second":125000000}}}`),
		lookPathOK: true,
	}
	backend := NewIperf3NetworkBackend(Iperf3Config{
		Server: "[2001:db8::1]:5201",
		Runner: runner,
	})

	_, err := backend.MeasureDownload()
	if err != nil {
		t.Fatalf("expected bracket IPv6 server to parse, got error: %v", err)
	}
	assertStringSlice(t, runner.args, []string{"-c", "2001:db8::1", "-p", "5201", "--json"})
}

func TestIperf3NetworkBackendRejectsInvalidPort(t *testing.T) {
	backend := NewIperf3NetworkBackend(Iperf3Config{
		Server: "127.0.0.1:not-a-port",
		Runner: &fakeCommandRunner{
			lookPathOK: true,
		},
	})

	_, err := backend.MeasureDownload()
	if err == nil {
		t.Fatal("expected invalid port to fail")
	}
	if !strings.Contains(err.Error(), "invalid iperf3 server port") {
		t.Fatalf("expected invalid port error, got %v", err)
	}
}

func TestIperf3NetworkBackendParsesUploadMbps(t *testing.T) {
	runner := &fakeCommandRunner{
		output:     []byte(`{"end":{"sum_sent":{"bits_per_second":42000000}}}`),
		lookPathOK: true,
	}
	backend := NewIperf3NetworkBackend(Iperf3Config{
		Server: "iperf.example:5201",
		Runner: runner,
	})

	speed, estimated, err := backend.MeasureUpload(0)
	if err != nil {
		t.Fatalf("expected upload measurement to parse, got error: %v", err)
	}
	if estimated {
		t.Fatal("expected iperf3 upload to be marked as real measurement")
	}
	if speed != 42.0 {
		t.Fatalf("expected 42 Mbps, got %.2f", speed)
	}
	assertStringSlice(t, runner.args, []string{"-c", "iperf.example", "-p", "5201", "--reverse", "--json"})
}

func TestParseIperf3MbpsRejectsMissingThroughput(t *testing.T) {
	if _, err := parseIperf3Mbps([]byte(`{"end":{}}`)); err == nil {
		t.Fatal("expected missing throughput to fail")
	}
}

func TestExecuteUsesConfiguredBackendName(t *testing.T) {
	networkTest := NewNetworkTestWithBackend(newTestLogger(t), NewIperf3NetworkBackend(Iperf3Config{Server: "127.0.0.1:5201"}))
	networkTest.latencyFn = func(_ []string) (float64, error) {
		return 10.0, nil
	}
	networkTest.downloadFn = func() (float64, error) {
		return 100.0, nil
	}
	networkTest.uploadFn = func(downloadSpeed float64) (float64, bool, error) {
		return 70.0, true, nil
	}

	result, err := networkTest.Execute()
	if err != nil {
		t.Fatalf("expected execute to succeed, got error: %v", err)
	}

	assertMetricString(t, result.Metrics, "backend", models.NetworkBackendIperf3)
	assertMetricString(t, result.Metrics, "backend_server", "127.0.0.1:5201")
}

func TestExecuteUsesBackendSpecificSources(t *testing.T) {
	networkTest := NewNetworkTestWithBackend(newTestLogger(t), NewIperf3NetworkBackend(Iperf3Config{Server: "127.0.0.1:5201"}))
	networkTest.latencyFn = func(_ []string) (float64, error) {
		return 10.0, nil
	}
	networkTest.downloadFn = func() (float64, error) {
		return 100.0, nil
	}
	networkTest.uploadFn = func(downloadSpeed float64) (float64, bool, error) {
		return 80.0, false, nil
	}

	result, err := networkTest.Execute()
	if err != nil {
		t.Fatalf("expected execute to succeed, got error: %v", err)
	}

	assertMetricString(t, result.Metrics, "download_speed_source", models.NetworkDownloadSourceIperf3)
	assertMetricString(t, result.Metrics, "upload_speed_source", models.NetworkUploadSourceIperf3)
	assertMetricBool(t, result.Metrics, "upload_speed_estimated", false)
}

func newTestLogger(t *testing.T) *logger.Logger {
	t.Helper()

	log, err := logger.NewLogger(t.TempDir(), "error")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	return log
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

func assertMetricInt(t *testing.T, metrics map[string]interface{}, key string, expected int) {
	t.Helper()

	value, ok := metrics[key].(int)
	if !ok {
		t.Fatalf("expected metric %s to be int, got %T", key, metrics[key])
	}
	if value != expected {
		t.Fatalf("expected metric %s to be %d, got %d", key, expected, value)
	}
}

type fakeCommandRunner struct {
	output      []byte
	err         error
	lookPathOK  bool
	lookPathErr error
	name        string
	args        []string
}

func (r *fakeCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	r.name = name
	r.args = append([]string(nil), args...)
	return r.output, r.err
}

func (r *fakeCommandRunner) LookPath(name string) (string, error) {
	if r.lookPathErr != nil {
		return "", r.lookPathErr
	}
	if r.lookPathOK {
		return "/usr/bin/" + name, nil
	}
	return "", fmt.Errorf("not found")
}

func assertStringSlice(t *testing.T, actual, expected []string) {
	t.Helper()

	if len(actual) != len(expected) {
		t.Fatalf("expected args %v, got %v", expected, actual)
	}
	for i := range expected {
		if actual[i] != expected[i] {
			t.Fatalf("expected args %v, got %v", expected, actual)
		}
	}
}
