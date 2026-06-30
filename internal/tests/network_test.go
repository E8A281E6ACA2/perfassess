package tests

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
	"github.com/E8A281E6ACA2/perfassess/pkg/logger"
)

func newTestNetworkTest(t *testing.T) *NetworkTest {
	t.Helper()

	log, err := logger.NewLogger(t.TempDir(), "error")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	networkTest := NewNetworkTest(log)
	networkTest.qualityTargets = nil
	return networkTest
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
	if score != 85.0 {
		t.Fatalf("expected estimated upload path to cap score at 85, got %.2f", score)
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
	networkTest.downloadFn = func() (NetworkDownloadResult, error) {
		return NetworkDownloadResult{SpeedMbps: 200.0}, nil
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
	if score != 85.0 {
		t.Fatalf("expected estimated upload score cap 85.0, got %.2f", score)
	}
}

func TestMeasureNetworkQualityTracksAvailabilityAndJitter(t *testing.T) {
	networkTest := newTestNetworkTest(t)
	networkTest.qualityTargets = []networkQualityTarget{
		{Name: "ipv4-test", Address: "1.1.1.1:443", Protocol: "ipv4"},
		{Name: "ipv6-test", Address: "[2001:db8::1]:443", Protocol: "ipv6"},
	}
	networkTest.qualitySamples = 3

	calls := map[string]int{}
	networkTest.qualityDialFn = func(address string, timeout time.Duration) (time.Duration, error) {
		if timeout != 2*time.Second {
			t.Fatalf("expected default quality timeout 2s, got %s", timeout)
		}
		calls[address]++
		switch address {
		case "1.1.1.1:443":
			switch calls[address] {
			case 1:
				return 10 * time.Millisecond, nil
			case 2:
				return 20 * time.Millisecond, nil
			default:
				return 0, fmt.Errorf("sample failed")
			}
		case "[2001:db8::1]:443":
			return 0, fmt.Errorf("ipv6 unavailable")
		default:
			t.Fatalf("unexpected quality target %q", address)
			return 0, nil
		}
	}

	results := networkTest.MeasureNetworkQuality()
	if len(results) != 2 {
		t.Fatalf("expected two quality results, got %d", len(results))
	}

	ipv4 := results[0]
	if !ipv4.Available {
		t.Fatal("expected ipv4 target to be available")
	}
	if ipv4.SuccessCount != 2 || ipv4.FailureCount != 1 {
		t.Fatalf("expected ipv4 success/failure 2/1, got %d/%d", ipv4.SuccessCount, ipv4.FailureCount)
	}
	if math.Abs(ipv4.FailureRate-(1.0/3.0)) > 0.0001 {
		t.Fatalf("expected ipv4 failure rate 1/3, got %.4f", ipv4.FailureRate)
	}
	if ipv4.AvgLatencyMs != 15 || ipv4.MinLatencyMs != 10 || ipv4.MaxLatencyMs != 20 || ipv4.JitterMs != 5 {
		t.Fatalf("unexpected ipv4 latency stats: %#v", ipv4)
	}

	ipv6 := results[1]
	if ipv6.Available {
		t.Fatal("expected ipv6 target to be unavailable")
	}
	if ipv6.SuccessCount != 0 || ipv6.FailureCount != 3 || ipv6.FailureRate != 1 {
		t.Fatalf("unexpected ipv6 stats: %#v", ipv6)
	}
}

func TestExecuteIncludesNetworkQualityMetrics(t *testing.T) {
	networkTest := newTestNetworkTest(t)
	networkTest.latencyFn = func(_ []string) (float64, error) {
		return 12.5, nil
	}
	networkTest.downloadFn = func() (NetworkDownloadResult, error) {
		return NetworkDownloadResult{SpeedMbps: 200.0}, nil
	}
	networkTest.uploadFn = func(downloadSpeed float64) (float64, bool, error) {
		return 140.0, true, nil
	}
	networkTest.qualityTargets = []networkQualityTarget{
		{Name: "ipv4-test", Address: "1.1.1.1:443", Protocol: "ipv4"},
		{Name: "ipv6-test", Address: "[2001:db8::1]:443", Protocol: "ipv6"},
	}
	networkTest.qualitySamples = 2
	networkTest.qualityDialFn = func(address string, timeout time.Duration) (time.Duration, error) {
		if address == "1.1.1.1:443" {
			return 10 * time.Millisecond, nil
		}
		return 0, fmt.Errorf("unavailable")
	}

	result, err := networkTest.Execute()
	if err != nil {
		t.Fatalf("expected execute to succeed, got error: %v", err)
	}

	assertMetricString(t, result.Metrics, "network_quality_profile", networkQualityProfile)
	assertMetricInt(t, result.Metrics, "network_quality_target_count", 2)
	assertMetricBool(t, result.Metrics, "network_quality_ipv4_available", true)
	assertMetricBool(t, result.Metrics, "network_quality_ipv6_available", false)
	assertMetricFloat(t, result.Metrics, "network_quality_ipv4_failure_rate", 0)
	assertMetricFloat(t, result.Metrics, "network_quality_ipv6_failure_rate", 1)
	assertMetricFloat(t, result.Metrics, "network_quality_avg_latency_ms", 10)
	assertMetricString(t, result.Metrics, "network_quality_1_target", "ipv4-test")
	assertMetricString(t, result.Metrics, "network_quality_2_protocol", "ipv6")
	assertMetricBool(t, result.Metrics, "network_quality_2_available", false)
	assertMetricInt(t, result.Metrics, "network_quality_2_failure_count", 2)
}

func TestDownloadSpeedFallsBackAcrossMultipleSources(t *testing.T) {
	failedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer failedServer.Close()

	successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("a", 1024*1024)))
	}))
	defer successServer.Close()

	networkTest := newTestNetworkTest(t)
	networkTest.downloadURLs = []string{
		failedServer.URL,
		successServer.URL,
	}

	result, err := networkTest.TestDownloadSpeed()
	if err != nil {
		t.Fatalf("expected fallback download source to succeed, got %v", err)
	}
	if result.SpeedMbps <= 0 {
		t.Fatalf("expected positive speed, got %.2f", result.SpeedMbps)
	}
	if result.SourceURL != successServer.URL {
		t.Fatalf("expected successful source URL %q, got %q", successServer.URL, result.SourceURL)
	}
}

func TestDownloadSpeedReportsAllSourceFailures(t *testing.T) {
	failedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer failedServer.Close()

	networkTest := newTestNetworkTest(t)
	networkTest.downloadURLs = []string{failedServer.URL}

	_, err := networkTest.TestDownloadSpeed()
	if err == nil {
		t.Fatal("expected all source failures to return error")
	}
	if !strings.Contains(err.Error(), "所有下载源失败") {
		t.Fatalf("expected all source failure error, got %v", err)
	}
}

func TestExecuteMarksFailedDownloadAndUnavailableUpload(t *testing.T) {
	networkTest := newTestNetworkTest(t)
	networkTest.latencyFn = func(_ []string) (float64, error) {
		return 0, fmt.Errorf("latency unavailable")
	}
	networkTest.downloadFn = func() (NetworkDownloadResult, error) {
		return NetworkDownloadResult{}, fmt.Errorf("download unavailable")
	}
	networkTest.uploadFn = func(downloadSpeed float64) (float64, bool, error) {
		if downloadSpeed != -1.0 {
			t.Fatalf("expected failed download to pass -1.0 into upload function, got %.2f", downloadSpeed)
		}
		return 0, false, fmt.Errorf("upload unavailable")
	}

	result, err := networkTest.Execute()
	if err != nil {
		t.Fatalf("expected execute to still succeed, got error: %v", err)
	}
	if result.Status != models.TestStatusDegraded {
		t.Fatalf("expected degraded status, got %q", result.Status)
	}
	if result.ErrorMessage == "" {
		t.Fatal("expected degraded result to carry error message")
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

func TestExecuteMarksPartialNetworkFailureAsDegraded(t *testing.T) {
	networkTest := newTestNetworkTest(t)
	networkTest.latencyFn = func(_ []string) (float64, error) {
		return 10.0, nil
	}
	networkTest.downloadFn = func() (NetworkDownloadResult, error) {
		return NetworkDownloadResult{}, fmt.Errorf("download unavailable")
	}
	networkTest.uploadFn = func(downloadSpeed float64) (float64, bool, error) {
		return 0, false, fmt.Errorf("upload unavailable")
	}

	result, err := networkTest.Execute()
	if err != nil {
		t.Fatalf("expected degraded execute to return result without error, got %v", err)
	}
	if result.Status != models.TestStatusDegraded {
		t.Fatalf("expected degraded status, got %q", result.Status)
	}
	assertMetricFloat(t, result.Metrics, "average_latency_ms", 10.0)
	assertMetricFloat(t, result.Metrics, "download_speed_mbps", -1.0)
	assertMetricFloat(t, result.Metrics, "score", 0.0)
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

func TestIperf3NetworkBackendUsesLatencyFallback(t *testing.T) {
	backend := NewIperf3NetworkBackend(Iperf3Config{
		Server: "127.0.0.1:5201",
		LatencyFn: func(hosts []string) (float64, error) {
			if len(hosts) != 1 || hosts[0] != "1.1.1.1:53" {
				t.Fatalf("unexpected latency hosts: %v", hosts)
			}
			return 8.5, nil
		},
	})

	latency, err := backend.MeasureLatency([]string{"1.1.1.1:53"})
	if err != nil {
		t.Fatalf("expected latency fallback to succeed, got error: %v", err)
	}
	if latency != 8.5 {
		t.Fatalf("expected latency 8.5, got %.2f", latency)
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

	download, err := backend.MeasureDownload()
	if err != nil {
		t.Fatalf("expected download measurement to parse, got error: %v", err)
	}
	if download.SpeedMbps != 125.0 {
		t.Fatalf("expected 125 Mbps, got %.2f", download.SpeedMbps)
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

func TestIperf3NetworkBackendRunsMultiServerMatrixAndCachesResult(t *testing.T) {
	runner := &recordingCommandRunner{
		outputs: [][]byte{
			[]byte(`{"end":{"sum_received":{"bits_per_second":100000000}}}`),
			[]byte(`{"end":{"sum_sent":{"bits_per_second":50000000}}}`),
			[]byte(`{"end":{"sum_received":{"bits_per_second":300000000}}}`),
			[]byte(`{"end":{"sum_sent":{"bits_per_second":150000000}}}`),
		},
	}
	latencyCalls := 0
	backend := NewIperf3NetworkBackend(Iperf3Config{
		Servers: []string{"node-a:5201", "[2001:db8::2]:5201"},
		Runner:  runner,
		LatencyFn: func(hosts []string) (float64, error) {
			latencyCalls++
			if len(hosts) != 1 {
				t.Fatalf("expected one latency host, got %v", hosts)
			}
			return float64(latencyCalls * 10), nil
		},
	})

	download, err := backend.MeasureDownload()
	if err != nil {
		t.Fatalf("expected matrix download to succeed, got %v", err)
	}
	if download.SpeedMbps != 200 {
		t.Fatalf("expected average download 200 Mbps, got %.2f", download.SpeedMbps)
	}
	upload, estimated, err := backend.MeasureUpload(download.SpeedMbps)
	if err != nil {
		t.Fatalf("expected matrix upload to succeed, got %v", err)
	}
	if estimated {
		t.Fatal("expected matrix upload to be real measurement")
	}
	if upload != 100 {
		t.Fatalf("expected average upload 100 Mbps, got %.2f", upload)
	}
	if runner.calls != 4 {
		t.Fatalf("expected matrix to run four iperf3 commands once, got %d", runner.calls)
	}
	if latencyCalls != 2 {
		t.Fatalf("expected latency fallback once per server, got %d", latencyCalls)
	}

	metrics := map[string]interface{}{}
	backend.AppendMetrics(metrics)
	assertMetricString(t, metrics, "iperf3_matrix_profile", "multi_server")
	assertMetricFloat(t, metrics, "iperf3_matrix_avg_download_mbps", 200)
	assertMetricFloat(t, metrics, "iperf3_matrix_avg_upload_mbps", 100)
	assertMetricFloat(t, metrics, "iperf3_matrix_best_download_mbps", 300)
	assertMetricFloat(t, metrics, "iperf3_matrix_best_upload_mbps", 150)
	assertMetricString(t, metrics, "iperf3_matrix_1_protocol", "ipv4")
	assertMetricString(t, metrics, "iperf3_matrix_2_protocol", "ipv6")
	assertMetricFloat(t, metrics, "iperf3_matrix_2_latency_ms", 20)
}

func TestLoadIperf3ServersFileParsesCommentsAndInlineComments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "iperf3-servers.txt")
	content := `
# private iperf3 nodes
node-a:5201 auth=owned

node-b:5201 auth=authorized # eu node
[2001:db8::2]:5201 auth=owned
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write iperf3 server file: %v", err)
	}

	servers, err := loadIperf3ServersFile(path)
	if err != nil {
		t.Fatalf("expected server file to parse, got %v", err)
	}
	assertStringSlice(t, servers, []string{"node-a:5201", "node-b:5201", "[2001:db8::2]:5201"})
}

func TestLoadIperf3ServerSpecsFileRejectsMissingAuthorization(t *testing.T) {
	path := filepath.Join(t.TempDir(), "iperf3-servers.txt")
	if err := os.WriteFile(path, []byte("node-a:5201 name=Tokyo\n"), 0o600); err != nil {
		t.Fatalf("failed to write iperf3 server file: %v", err)
	}

	_, err := loadIperf3ServerSpecsFile(path)
	if err == nil {
		t.Fatal("expected server file entry without auth metadata to fail")
	}
	if !strings.Contains(err.Error(), "auth=owned or auth=authorized") {
		t.Fatalf("expected authorization error, got %v", err)
	}
}

func TestLoadIperf3ServerSpecsFileParsesMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "iperf3-servers.txt")
	content := `
node-a:5201 name=Tokyo region=JP provider=SelfHosted auth=owned
[2001:db8::2]:5201 name=Frankfurt region=DE provider=Lab auth=authorized
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write iperf3 server file: %v", err)
	}

	specs, err := loadIperf3ServerSpecsFile(path)
	if err != nil {
		t.Fatalf("expected server file to parse, got %v", err)
	}
	if len(specs) != 2 {
		t.Fatalf("expected two specs, got %#v", specs)
	}
	if specs[0].Server != "node-a:5201" || specs[0].Name != "Tokyo" || specs[0].Region != "JP" || specs[0].Provider != "SelfHosted" || specs[0].Authorization != "owned" {
		t.Fatalf("unexpected first spec: %#v", specs[0])
	}
	if specs[1].Server != "[2001:db8::2]:5201" || specs[1].Name != "Frankfurt" || specs[1].Region != "DE" || specs[1].Provider != "Lab" || specs[1].Authorization != "authorized" {
		t.Fatalf("unexpected second spec: %#v", specs[1])
	}
}

func TestLoadIperf3ServerSpecsFileRejectsUnknownMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "iperf3-servers.txt")
	if err := os.WriteFile(path, []byte("node-a:5201 auth=owned weight=10\n"), 0o600); err != nil {
		t.Fatalf("failed to write iperf3 server file: %v", err)
	}

	_, err := loadIperf3ServerSpecsFile(path)
	if err == nil {
		t.Fatal("expected unknown metadata key to fail")
	}
	if !strings.Contains(err.Error(), "unsupported iperf3 server metadata key") {
		t.Fatalf("expected unsupported key error, got %v", err)
	}
}

func TestIperf3NetworkBackendRunsMatrixFromServerFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "iperf3-servers.txt")
	content := `
node-a:5201 name=Tokyo region=JP provider=SelfHosted auth=owned
node-b:5201 name=Frankfurt region=DE provider=Lab auth=authorized
node-a:5201 name=Tokyo region=JP provider=SelfHosted auth=owned
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write iperf3 server file: %v", err)
	}
	runner := &recordingCommandRunner{
		outputs: [][]byte{
			[]byte(`{"end":{"sum_received":{"bits_per_second":100000000}}}`),
			[]byte(`{"end":{"sum_sent":{"bits_per_second":50000000}}}`),
			[]byte(`{"end":{"sum_received":{"bits_per_second":300000000}}}`),
			[]byte(`{"end":{"sum_sent":{"bits_per_second":150000000}}}`),
		},
	}
	backend := NewIperf3NetworkBackend(Iperf3Config{
		ServerFile: path,
		Runner:     runner,
		LatencyFn: func(hosts []string) (float64, error) {
			return 10.0, nil
		},
	})

	download, err := backend.MeasureDownload()
	if err != nil {
		t.Fatalf("expected matrix download to succeed, got %v", err)
	}
	if download.SpeedMbps != 200 {
		t.Fatalf("expected average download 200 Mbps, got %.2f", download.SpeedMbps)
	}
	metrics := map[string]interface{}{}
	backend.AppendMetrics(metrics)
	assertMetricInt(t, metrics, "iperf3_matrix_server_count", 2)
	assertMetricString(t, metrics, "iperf3_matrix_1_server", "node-a:5201")
	assertMetricString(t, metrics, "iperf3_matrix_1_name", "Tokyo")
	assertMetricString(t, metrics, "iperf3_matrix_1_region", "JP")
	assertMetricString(t, metrics, "iperf3_matrix_1_provider", "SelfHosted")
	assertMetricString(t, metrics, "iperf3_matrix_1_authorization", "owned")
	assertMetricString(t, metrics, "iperf3_matrix_2_server", "node-b:5201")
	assertMetricString(t, metrics, "iperf3_matrix_2_name", "Frankfurt")
	assertMetricString(t, metrics, "iperf3_matrix_2_region", "DE")
	assertMetricString(t, metrics, "iperf3_matrix_2_provider", "Lab")
	assertMetricString(t, metrics, "iperf3_matrix_2_authorization", "authorized")
}

func TestSpeedtestNetworkBackendReportsMissingBinary(t *testing.T) {
	backend := NewSpeedtestNetworkBackend(SpeedtestConfig{
		Runner: &fakeCommandRunner{
			lookPathErr: fmt.Errorf("not found"),
		},
	})

	_, err := backend.MeasureDownload()
	if err == nil {
		t.Fatal("expected missing speedtest binary to fail")
	}
	if !strings.Contains(err.Error(), "安装 Ookla Speedtest CLI") {
		t.Fatalf("expected install hint, got %v", err)
	}
	category, stage, hint, ok := benchmarkErrorFields(err)
	if !ok {
		t.Fatalf("expected benchmark error fields, got %T", err)
	}
	if category != BenchmarkErrorMissingDependency || stage != "speedtest_lookup" || hint == "" {
		t.Fatalf("unexpected benchmark error fields: %q %q %q", category, stage, hint)
	}
}

func TestSpeedtestNetworkBackendClassifiesNetworkFailure(t *testing.T) {
	backend := NewSpeedtestNetworkBackend(SpeedtestConfig{
		Runner: &fakeCommandRunner{
			output:     []byte("Configuration - Could not resolve host"),
			err:        fmt.Errorf("exit status 1"),
			lookPathOK: true,
		},
	})

	_, err := backend.MeasureDownload()
	if err == nil {
		t.Fatal("expected speedtest command failure")
	}
	category, stage, hint, ok := benchmarkErrorFields(err)
	if !ok {
		t.Fatalf("expected benchmark error fields, got %T", err)
	}
	if category != BenchmarkErrorNetwork || stage != "speedtest_run" || !strings.Contains(hint, "网络不可达") {
		t.Fatalf("unexpected benchmark error fields: %q %q %q", category, stage, hint)
	}
}

func TestNetworkTestAddsBenchmarkErrorMetricsForBackendFailure(t *testing.T) {
	networkTest := NewNetworkTestWithBackend(newTestLogger(t), NewSpeedtestNetworkBackend(SpeedtestConfig{
		Runner: &fakeCommandRunner{lookPathErr: fmt.Errorf("not found")},
	}))
	networkTest.qualityTargets = nil

	result, err := networkTest.Execute()
	if err != nil {
		t.Fatalf("expected degraded network result without fatal error, got %v", err)
	}
	if result.Status != models.TestStatusDegraded {
		t.Fatalf("expected degraded status, got %q", result.Status)
	}
	assertMetricString(t, result.Metrics, "error_category", BenchmarkErrorMissingDependency)
	assertMetricString(t, result.Metrics, "error_stage", "speedtest_lookup")
	assertMetricString(t, result.Metrics, "network_error_latency_category", BenchmarkErrorMissingDependency)
	assertMetricString(t, result.Metrics, "network_error_download_category", BenchmarkErrorMissingDependency)
	assertMetricString(t, result.Metrics, "network_error_upload_category", BenchmarkErrorMissingDependency)
	if result.Metrics["error_hint"] == "" {
		t.Fatalf("expected error_hint metric, got %#v", result.Metrics)
	}
}

func TestSpeedtestNetworkBackendParsesMetricsAndCachesResult(t *testing.T) {
	runner := &fakeCommandRunner{
		output: []byte(`{
			"type": "result",
			"ping": {"latency": 12.5, "jitter": 1.1},
			"download": {"bandwidth": 125000000},
			"upload": {"bandwidth": 50000000},
			"server": {"id": 1234, "name": "Example Node", "location": "Frankfurt", "country": "Germany", "host": "speed.example:8080"},
			"result": {"id": "abcd", "url": "https://www.speedtest.net/result/c/abcd"},
			"interface": {"internalIp": "10.0.0.2", "name": "eth0", "macAddr": "00:11:22:33:44:55", "isVpn": false, "externalIp": "203.0.113.10"},
			"isp": "Example ISP"
		}`),
		lookPathOK: true,
	}
	backend := NewSpeedtestNetworkBackend(SpeedtestConfig{Runner: runner})

	latency, err := backend.MeasureLatency(nil)
	if err != nil {
		t.Fatalf("expected latency to parse, got %v", err)
	}
	if latency != 12.5 {
		t.Fatalf("expected latency 12.5, got %.2f", latency)
	}

	download, err := backend.MeasureDownload()
	if err != nil {
		t.Fatalf("expected download to parse, got %v", err)
	}
	if download.SpeedMbps != 1000 {
		t.Fatalf("expected 1000 Mbps download, got %.2f", download.SpeedMbps)
	}

	upload, estimated, err := backend.MeasureUpload(download.SpeedMbps)
	if err != nil {
		t.Fatalf("expected upload to parse, got %v", err)
	}
	if estimated {
		t.Fatal("expected speedtest upload to be real measurement")
	}
	if upload != 400 {
		t.Fatalf("expected 400 Mbps upload, got %.2f", upload)
	}
	if backend.Server() != "speed.example:8080" {
		t.Fatalf("expected server host, got %q", backend.Server())
	}
	if runner.calls != 1 {
		t.Fatalf("expected speedtest to run once due to cache, got %d", runner.calls)
	}
	assertStringSlice(t, runner.args, []string{"--format=json", "--accept-license", "--accept-gdpr"})

	metrics := map[string]interface{}{}
	backend.AppendMetrics(metrics)
	assertMetricString(t, metrics, "speedtest_profile", "ookla_cli")
	assertMetricInt(t, metrics, "speedtest_server_id", 1234)
	assertMetricString(t, metrics, "speedtest_server_name", "Example Node")
	assertMetricString(t, metrics, "speedtest_server_location", "Frankfurt")
	assertMetricString(t, metrics, "speedtest_server_country", "Germany")
	assertMetricString(t, metrics, "speedtest_server_host", "speed.example:8080")
	assertMetricString(t, metrics, "speedtest_result_url", "https://www.speedtest.net/result/c/abcd")
	assertMetricString(t, metrics, "speedtest_isp", "Example ISP")
	assertMetricString(t, metrics, "speedtest_interface_external_ip", "203.0.113.10")
	assertMetricFloat(t, metrics, "speedtest_ping_jitter_ms", 1.1)
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
	networkTest.downloadFn = func() (NetworkDownloadResult, error) {
		return NetworkDownloadResult{SpeedMbps: 100.0}, nil
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
	networkTest.downloadFn = func() (NetworkDownloadResult, error) {
		return NetworkDownloadResult{SpeedMbps: 100.0}, nil
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

func TestExecuteWithIperf3BackendUsesLatencyFallbackWithoutError(t *testing.T) {
	runner := &fakeCommandRunner{
		outputs: [][]byte{
			[]byte(`{"end":{"sum_received":{"bits_per_second":100000000}}}`),
			[]byte(`{"end":{"sum_sent":{"bits_per_second":50000000}}}`),
		},
		lookPathOK: true,
	}
	backend := NewIperf3NetworkBackend(Iperf3Config{
		Server: "127.0.0.1:5201",
		Runner: runner,
		LatencyFn: func(hosts []string) (float64, error) {
			return 9.5, nil
		},
	})
	networkTest := NewNetworkTestWithBackend(newTestLogger(t), backend)

	result, err := networkTest.Execute()
	if err != nil {
		t.Fatalf("expected execute to succeed, got error: %v", err)
	}

	assertMetricFloat(t, result.Metrics, "average_latency_ms", 9.5)
	assertMetricFloat(t, result.Metrics, "download_speed_mbps", 100.0)
	assertMetricFloat(t, result.Metrics, "upload_speed_mbps", 50.0)
	assertMetricString(t, result.Metrics, "download_speed_source", models.NetworkDownloadSourceIperf3)
	assertMetricString(t, result.Metrics, "upload_speed_source", models.NetworkUploadSourceIperf3)
	assertMetricString(t, result.Metrics, "network_error", "")
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
	outputs     [][]byte
	err         error
	lookPathOK  bool
	lookPathErr error
	name        string
	args        []string
	calls       int
}

type recordingCommandRunner struct {
	outputs [][]byte
	err     error
	names   []string
	args    [][]string
	calls   int
}

func (r *recordingCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	r.names = append(r.names, name)
	r.args = append(r.args, append([]string(nil), args...))
	r.calls++
	index := r.calls - 1
	if index >= len(r.outputs) {
		index = len(r.outputs) - 1
	}
	return r.outputs[index], r.err
}

func (r *recordingCommandRunner) LookPath(name string) (string, error) {
	return "/usr/bin/" + name, nil
}

func (r *fakeCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	r.name = name
	r.args = append([]string(nil), args...)
	r.calls++
	if len(r.outputs) > 0 {
		index := r.calls - 1
		if index >= len(r.outputs) {
			index = len(r.outputs) - 1
		}
		return r.outputs[index], r.err
	}
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
