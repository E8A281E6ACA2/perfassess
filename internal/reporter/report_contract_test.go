package reporter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"performance-assessment-system/internal/models"
)

func TestReportJSONSampleContract(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "docs", "examples", "report-json-sample.json"))
	if err != nil {
		t.Fatalf("failed to read sample report: %v", err)
	}

	var sample map[string]interface{}
	if err := json.Unmarshal(content, &sample); err != nil {
		t.Fatalf("sample report is not valid JSON: %v", err)
	}

	requiredTopLevel := []string{"session_id", "timestamp", "system_info", "test_results", "summary"}
	for _, key := range requiredTopLevel {
		if _, ok := sample[key]; !ok {
			t.Fatalf("sample report missing top-level key %q", key)
		}
	}

	summary := objectAt(t, sample, "summary")
	for _, key := range []string{"benchmark_profile", "confidence_level", "score_breakdown"} {
		if _, ok := summary[key]; !ok {
			t.Fatalf("sample summary missing key %q", key)
		}
	}

	profile := objectAt(t, summary, "benchmark_profile")
	if profile["name"] != "full_iperf3" {
		t.Fatalf("expected sample profile full_iperf3, got %#v", profile["name"])
	}

	confidence := objectAt(t, summary, "confidence_level")
	if confidence["level"] != "high" {
		t.Fatalf("expected sample confidence high, got %#v", confidence["level"])
	}

	breakdown := objectAt(t, summary, "score_breakdown")
	normalized := objectAt(t, breakdown, "normalized_total")
	if _, ok := normalized["active_weight"].(float64); !ok {
		t.Fatalf("expected normalized active_weight number, got %#v", normalized["active_weight"])
	}
}

func TestFormatReportSnapshot(t *testing.T) {
	generator := NewReportGeneratorWithWeights(map[string]float64{
		"cpu":     0.30,
		"memory":  0.20,
		"disk":    0.25,
		"network": 0.25,
	})
	report, err := generator.GenerateReport("snapshot_session", snapshotSystemInfo(), snapshotTestResults())
	if err != nil {
		t.Fatalf("expected report generation to succeed, got %v", err)
	}
	report.Timestamp = time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	report.FormattedContent = generator.FormatReport(report)

	expected, err := os.ReadFile(filepath.Join("testdata", "text_report_snapshot.txt"))
	if err != nil {
		t.Fatalf("failed to read text report snapshot: %v", err)
	}

	actual := strings.TrimSpace(report.FormattedContent)
	want := strings.TrimSpace(string(expected))
	if actual != want {
		t.Fatalf("text report snapshot mismatch\n--- got ---\n%s\n--- want ---\n%s", actual, want)
	}
}

func objectAt(t *testing.T, source map[string]interface{}, key string) map[string]interface{} {
	t.Helper()
	value, ok := source[key]
	if !ok {
		t.Fatalf("missing object key %q", key)
	}
	object, ok := value.(map[string]interface{})
	if !ok {
		t.Fatalf("expected %q to be object, got %T", key, value)
	}
	return object
}

func snapshotSystemInfo() *models.SystemInfo {
	return &models.SystemInfo{
		CPU: &models.CPUInfo{
			Model:        "Snapshot CPU",
			Cores:        4,
			Threads:      8,
			FrequencyMHz: 2400,
		},
		Memory: &models.MemoryInfo{
			TotalMB:     8192,
			AvailableMB: 4096,
			MemoryType:  "Unknown",
		},
		Disk: &models.DiskInfo{
			TotalGB:     100,
			AvailableGB: 60,
			DiskType:    "SSD",
		},
		OS: &models.OSInfo{
			Name:         "linux",
			Version:      "snapshot",
			Architecture: "x86_64",
		},
	}
}

func snapshotTestResults() *models.TestResults {
	return &models.TestResults{
		CPUResult: &models.TestResult{
			TestName:        "CPU性能测试",
			Status:          "success",
			DurationSeconds: 30,
			Metrics: map[string]interface{}{
				"backend":                    "sysbench",
				"single_core_score":          80.0,
				"single_core_events_per_sec": 1000.0,
				"multi_core_score":           90.0,
				"multi_core_events_per_sec":  8000.0,
				"total_score":                86.0,
				"cpu_cores":                  8,
			},
		},
		MemoryResult: &models.TestResult{
			TestName:        "内存性能测试",
			Status:          "success",
			DurationSeconds: 12,
			Metrics: map[string]interface{}{
				"backend":            "sysbench",
				"read_speed_mbps":    3200.0,
				"read_speed_source":  "sysbench",
				"write_speed_mbps":   2800.0,
				"write_speed_source": "sysbench",
				"score":              33.5,
			},
		},
		DiskResult: &models.TestResult{
			TestName:        "磁盘性能测试",
			Status:          "success",
			DurationSeconds: 20,
			Metrics: map[string]interface{}{
				"backend":                     "fio",
				"read_speed_mbps":             1000.0,
				"write_speed_mbps":            800.0,
				"random_iops":                 12000,
				"random_read_iops":            7000.0,
				"random_write_iops":           5000.0,
				"random_read_latency_p95_ms":  1.25,
				"random_write_latency_p95_ms": 2.5,
				"score":                       95.0,
			},
		},
		NetworkResult: &models.TestResult{
			TestName:        "网络性能测试",
			Status:          "success",
			DurationSeconds: 8,
			Metrics: map[string]interface{}{
				"backend":                "iperf3",
				"backend_server":         "127.0.0.1:5201",
				"average_latency_ms":     10.0,
				"latency_source":         "tcp_connect",
				"download_speed_mbps":    900.0,
				"download_speed_source":  "iperf3_download",
				"upload_speed_mbps":      850.0,
				"upload_speed_source":    "iperf3_upload",
				"upload_speed_estimated": false,
				"score":                  100.0,
			},
		},
	}
}
