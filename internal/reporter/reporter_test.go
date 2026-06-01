package reporter

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"performance-assessment-system/internal/models"
)

func TestCalculateNetworkScoreIgnoresEstimatedUpload(t *testing.T) {
	calculator := NewScoreCalculator()

	result := &models.TestResult{
		TestName: "网络性能测试",
		Status:   "success",
		Metrics: map[string]interface{}{
			"average_latency_ms":     25.0,
			"download_speed_mbps":    200.0,
			"upload_speed_mbps":      140.0,
			"upload_speed_estimated": true,
		},
	}

	score := calculator.CalculateNetworkScore(result)
	if score != 85.0 {
		t.Fatalf("expected estimated upload to cap network score at 85, got %.2f", score)
	}
}

func TestCalculateNetworkScoreUsesRealUpload(t *testing.T) {
	calculator := NewScoreCalculator()

	result := &models.TestResult{
		TestName: "网络性能测试",
		Status:   "success",
		Metrics: map[string]interface{}{
			"average_latency_ms":     50.0,
			"download_speed_mbps":    100.0,
			"upload_speed_mbps":      25.0,
			"upload_speed_estimated": false,
		},
	}

	score := calculator.CalculateNetworkScore(result)
	if score != 85.0 {
		t.Fatalf("expected real upload to participate in score, got %.2f", score)
	}
}

func TestFormatSingleTestResultMarksEstimatedUpload(t *testing.T) {
	generator := NewReportGenerator()
	result := &models.TestResult{
		TestName:        "网络性能测试",
		Status:          "success",
		DurationSeconds: 3.2,
		Metrics: map[string]interface{}{
			"backend":                "iperf3",
			"backend_server":         "127.0.0.1:5201",
			"average_latency_ms":     12.34,
			"latency_source":         "tcp_connect",
			"download_speed_mbps":    321.0,
			"download_speed_source":  "http_download",
			"upload_speed_mbps":      200.0,
			"upload_speed_source":    "estimated_from_download",
			"upload_speed_estimated": true,
			"score":                  91.0,
		},
	}

	formatted := generator.formatSingleTestResult("网络性能测试", result)

	expectedSnippets := []string{
		"测试后端:     iperf3",
		"后端服务端:   127.0.0.1:5201",
		"平均延迟:     12.34 ms (tcp_connect)",
		"下载速度:     321.00 Mbps (http_download)",
		"上传速度:     200.00 Mbps (estimated_from_download)",
		"上传说明:     当前结果为估算值，不参与真实上传评分，网络评分上限为 85",
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(formatted, snippet) {
			t.Fatalf("expected formatted report to contain %q, got:\n%s", snippet, formatted)
		}
	}
}

func TestFormatSingleTestResultShowsNetworkError(t *testing.T) {
	generator := NewReportGenerator()
	result := &models.TestResult{
		TestName:        "网络性能测试",
		Status:          "success",
		DurationSeconds: 1.4,
		Metrics: map[string]interface{}{
			"average_latency_ms":  -1.0,
			"download_speed_mbps": -1.0,
			"upload_speed_mbps":   -1.0,
			"network_error":       "iperf3 is not installed",
			"score":               0.0,
		},
	}

	formatted := generator.formatSingleTestResult("网络性能测试", result)
	if !strings.Contains(formatted, "网络说明:     iperf3 is not installed") {
		t.Fatalf("expected formatted report to include network error, got:\n%s", formatted)
	}
}

func TestWebServerKeyMetricsMarksEstimatedUpload(t *testing.T) {
	server := &WebServer{}
	result := &models.TestResult{
		TestName: "网络性能测试",
		Status:   "success",
		Metrics: map[string]interface{}{
			"backend":                "builtin",
			"average_latency_ms":     8.5,
			"upload_speed_estimated": true,
		},
	}

	metrics := server.getKeyMetrics(result)
	if !strings.Contains(metrics, "builtin | 延迟: 8.50 ms | 上传: 估算值") {
		t.Fatalf("expected web key metrics to mark estimated upload, got %q", metrics)
	}
}

func TestWebServerKeyMetricsShowsNetworkError(t *testing.T) {
	server := &WebServer{}
	result := &models.TestResult{
		TestName: "网络性能测试",
		Status:   "success",
		Metrics: map[string]interface{}{
			"network_error": "iperf3 server is required",
		},
	}

	metrics := server.getKeyMetrics(result)
	if metrics != "iperf3 server is required" {
		t.Fatalf("expected web key metrics to show network error, got %q", metrics)
	}
}

func TestCalculateOverallScoreNormalizesWeightsForExecutedTests(t *testing.T) {
	calculator := NewScoreCalculator()
	results := &models.TestResults{
		CPUResult: &models.TestResult{
			TestName: "CPU性能测试",
			Status:   "success",
			Metrics: map[string]interface{}{
				"total_score": 80.0,
			},
		},
		MemoryResult: &models.TestResult{
			TestName: "内存性能测试",
			Status:   "success",
			Metrics: map[string]interface{}{
				"read_speed_mbps":  5000.0,
				"write_speed_mbps": 3000.0,
			},
		},
	}

	overall := calculator.CalculateOverallScore(results)

	if overall.CPUScore != 80.0 {
		t.Fatalf("expected cpu score 80.0, got %.2f", overall.CPUScore)
	}
	if overall.MemoryScore != 100.0 {
		t.Fatalf("expected memory score 100.0, got %.2f", overall.MemoryScore)
	}

	expectedTotal := (80.0*0.30 + 100.0*0.20) / 0.50
	if overall.TotalScore != expectedTotal {
		t.Fatalf("expected normalized total %.2f, got %.2f", expectedTotal, overall.TotalScore)
	}
	if overall.Grade != "良好" {
		t.Fatalf("expected grade 良好, got %s", overall.Grade)
	}
}

func TestAddSummaryMarksIncompleteReport(t *testing.T) {
	generator := NewReportGenerator()
	report := &models.Report{
		SessionID:  "session_test",
		Timestamp:  time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC),
		SystemInfo: &models.SystemInfo{},
		TestResults: &models.TestResults{
			CPUResult:    &models.TestResult{TestName: "CPU性能测试", Status: "success"},
			MemoryResult: &models.TestResult{TestName: "内存性能测试", Status: "success"},
			DiskResult:   &models.TestResult{TestName: "磁盘性能测试", Status: "failed"},
		},
		Summary: make(map[string]interface{}),
	}
	overall := &models.OverallScore{
		CPUScore:     90,
		MemoryScore:  88,
		DiskScore:    0,
		NetworkScore: 0,
		TotalScore:   71.2,
		Grade:        "一般",
	}

	generator.AddSummary(report, overall)

	if grade, _ := report.Summary["grade"].(string); grade != "未完成" {
		t.Fatalf("expected report summary grade 未完成, got %q", grade)
	}
	if overall.Grade != "未完成" {
		t.Fatalf("expected overall grade to be mutated to 未完成, got %q", overall.Grade)
	}
	if note, _ := report.Summary["performance_note"].(string); note == "" {
		t.Fatal("expected performance_note to be populated for incomplete report")
	}
	if success, _ := report.Summary["tests_success"].(int); success != 2 {
		t.Fatalf("expected 2 successful tests, got %d", success)
	}
	if failed, _ := report.Summary["tests_failed"].(int); failed != 1 {
		t.Fatalf("expected 1 failed test, got %d", failed)
	}
}

func TestFormatReportIncludesIncompleteNote(t *testing.T) {
	generator := NewReportGenerator()
	overall := &models.OverallScore{
		CPUScore:     80,
		MemoryScore:  70,
		DiskScore:    0,
		NetworkScore: 0,
		TotalScore:   76,
		Grade:        "未完成",
	}
	report := &models.Report{
		SessionID:  "session_test",
		Timestamp:  time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC),
		SystemInfo: &models.SystemInfo{},
		TestResults: &models.TestResults{
			CPUResult: &models.TestResult{TestName: "CPU性能测试", Status: "success"},
		},
		Summary: map[string]interface{}{
			"overall_score":    overall,
			"tests_success":    1,
			"tests_failed":     0,
			"tests_skipped":    0,
			"performance_note": "由于未执行所有性能测试，无法给出完整的性能结论。",
		},
	}

	formatted := generator.FormatReport(report)

	expectedSnippets := []string{
		"性能等级:       未完成",
		"说明:           由于未执行所有性能测试，无法给出完整的性能结论。",
		"成功测试:       1",
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(formatted, snippet) {
			t.Fatalf("expected formatted report to contain %q, got:\n%s", snippet, formatted)
		}
	}
}

func TestOutputFormatterOutputsJSONReport(t *testing.T) {
	formatter := NewOutputFormatter()
	report := &models.Report{
		SessionID:   "session_json",
		Timestamp:   time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC),
		SystemInfo:  &models.SystemInfo{},
		TestResults: &models.TestResults{},
		Summary: map[string]interface{}{
			"grade": "未完成",
		},
		FormattedContent: "text report",
	}

	content, err := formatter.OutputJSON(report)
	if err != nil {
		t.Fatalf("expected JSON output to succeed, got %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(content), &decoded); err != nil {
		t.Fatalf("expected valid JSON, got %v\n%s", err, content)
	}
	if decoded["session_id"] != "session_json" {
		t.Fatalf("expected session_id in JSON, got %v", decoded["session_id"])
	}
	if _, exists := decoded["FormattedContent"]; exists {
		t.Fatal("expected formatted content to be omitted from JSON")
	}
}

func TestAddSummaryAddsQualityNotes(t *testing.T) {
	generator := NewReportGenerator()
	report := &models.Report{
		SessionID:  "session_quality",
		Timestamp:  time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC),
		SystemInfo: &models.SystemInfo{},
		TestResults: &models.TestResults{
			DiskResult: &models.TestResult{
				TestName: "磁盘性能测试",
				Status:   "success",
				Metrics: map[string]interface{}{
					"backend": "builtin",
				},
			},
			NetworkResult: &models.TestResult{
				TestName: "网络性能测试",
				Status:   "success",
				Metrics: map[string]interface{}{
					"average_latency_ms":     10.0,
					"upload_speed_estimated": true,
				},
			},
		},
		Summary: make(map[string]interface{}),
	}

	generator.AddSummary(report, &models.OverallScore{})

	notes, ok := report.Summary["quality_notes"].([]string)
	if !ok || len(notes) == 0 {
		t.Fatalf("expected quality notes, got %#v", report.Summary["quality_notes"])
	}
	joined := strings.Join(notes, "\n")
	expectedSnippets := []string{
		"CPU测试未执行",
		"磁盘测试使用内置后端",
		"网络上传速度为估算值",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(joined, snippet) {
			t.Fatalf("expected quality notes to contain %q, got %v", snippet, notes)
		}
	}
}
