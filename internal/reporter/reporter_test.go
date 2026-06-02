package reporter

import (
	"encoding/json"
	"math"
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

func TestCalculateNetworkScoreIgnoresDegradedNetwork(t *testing.T) {
	calculator := NewScoreCalculator()

	result := &models.TestResult{
		TestName: "网络性能测试",
		Status:   models.TestStatusDegraded,
		Metrics: map[string]interface{}{
			"average_latency_ms":     10.0,
			"download_speed_mbps":    -1.0,
			"upload_speed_mbps":      -1.0,
			"upload_speed_estimated": false,
		},
	}

	score := calculator.CalculateNetworkScore(result)
	if score != 0.0 {
		t.Fatalf("expected degraded network score 0, got %.2f", score)
	}
}

func TestBuildNetworkScoreBreakdownClampsNegativeSubScores(t *testing.T) {
	calculator := NewScoreCalculator()

	result := &models.TestResult{
		TestName: "网络性能测试",
		Status:   models.TestStatusDegraded,
		Metrics: map[string]interface{}{
			"average_latency_ms":  10.0,
			"download_speed_mbps": -1.0,
			"upload_speed_mbps":   -1.0,
		},
	}

	breakdown := calculator.buildNetworkScoreBreakdown(result)
	if breakdown["download_score"] != 0.0 {
		t.Fatalf("expected clamped download score 0, got %#v", breakdown["download_score"])
	}
	if breakdown["upload_score"] != 0.0 {
		t.Fatalf("expected clamped upload score 0, got %#v", breakdown["upload_score"])
	}
	if active, _ := breakdown["active"].(bool); active {
		t.Fatal("expected degraded network breakdown to be inactive")
	}
}

func TestBuildNetworkScoreBreakdownDoesNotScoreEstimatedUpload(t *testing.T) {
	calculator := NewScoreCalculator()

	result := &models.TestResult{
		TestName: "网络性能测试",
		Status:   models.TestStatusSuccess,
		Metrics: map[string]interface{}{
			"average_latency_ms":     10.0,
			"download_speed_mbps":    100.0,
			"upload_speed_mbps":      100.0,
			"upload_speed_estimated": true,
		},
	}

	breakdown := calculator.buildNetworkScoreBreakdown(result)
	if breakdown["upload_score"] != 0.0 {
		t.Fatalf("expected estimated upload score 0, got %#v", breakdown["upload_score"])
	}
	if breakdown["upload_estimated"] != true {
		t.Fatalf("expected upload_estimated true, got %#v", breakdown["upload_estimated"])
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

func TestFormatSingleTestResultShowsCPUBackendAndEvents(t *testing.T) {
	generator := NewReportGenerator()
	result := &models.TestResult{
		TestName:        "CPU性能测试",
		Status:          "success",
		DurationSeconds: 31.2,
		Metrics: map[string]interface{}{
			"backend":                    "sysbench",
			"single_core_score":          88.0,
			"single_core_events_per_sec": 1056.5,
			"multi_core_score":           92.0,
			"multi_core_events_per_sec":  8844.25,
			"total_score":                90.0,
			"cpu_cores":                  8,
		},
	}

	formatted := generator.formatSingleTestResult("CPU性能测试", result)

	expectedSnippets := []string{
		"测试后端:     sysbench",
		"单核吞吐:     1056.50 events/s",
		"多核吞吐:     8844.25 events/s",
		"总体评分:     90.00",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(formatted, snippet) {
			t.Fatalf("expected formatted report to contain %q, got:\n%s", snippet, formatted)
		}
	}
}

func TestFormatSingleTestResultShowsFioDetailedMetrics(t *testing.T) {
	generator := NewReportGenerator()
	result := &models.TestResult{
		TestName:        "磁盘性能测试",
		Status:          "success",
		DurationSeconds: 20.5,
		Metrics: map[string]interface{}{
			"backend":                     "fio",
			"read_speed_mbps":             1000.0,
			"write_speed_mbps":            800.0,
			"random_iops":                 12000,
			"random_read_iops":            7000.5,
			"random_write_iops":           5000.5,
			"random_read_latency_p95_ms":  1.25,
			"random_write_latency_p95_ms": 2.5,
			"score":                       95.0,
		},
	}

	formatted := generator.formatSingleTestResult("磁盘性能测试", result)

	expectedSnippets := []string{
		"测试后端:     fio",
		"随机读IOPS:   7000.50",
		"随机写IOPS:   5000.50",
		"随机读P95:    1.25 ms",
		"随机写P95:    2.50 ms",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(formatted, snippet) {
			t.Fatalf("expected formatted report to contain %q, got:\n%s", snippet, formatted)
		}
	}
}

func TestFormatSingleTestResultShowsCPURawScores(t *testing.T) {
	generator := NewReportGenerator()
	result := &models.TestResult{
		TestName:        "CPU性能测试",
		Status:          "success",
		DurationSeconds: 31.2,
		Metrics: map[string]interface{}{
			"backend":               "geekbench",
			"single_core_score":     98.0,
			"single_core_raw_score": 2450.0,
			"multi_core_score":      95.0,
			"multi_core_raw_score":  15200.0,
			"total_score":           96.0,
			"cpu_cores":             8,
		},
	}

	formatted := generator.formatSingleTestResult("CPU性能测试", result)

	expectedSnippets := []string{
		"测试后端:     geekbench",
		"单核原始分:   2450",
		"多核原始分:   15200",
		"总体评分:     96.00",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(formatted, snippet) {
			t.Fatalf("expected formatted report to contain %q, got:\n%s", snippet, formatted)
		}
	}
}

func TestFormatSingleTestResultShowsMemoryBackendAndSources(t *testing.T) {
	generator := NewReportGenerator()
	result := &models.TestResult{
		TestName:        "内存性能测试",
		Status:          "success",
		DurationSeconds: 4.5,
		Metrics: map[string]interface{}{
			"backend":            "sysbench",
			"read_speed_mbps":    3200.0,
			"read_speed_source":  "sysbench",
			"write_speed_mbps":   2800.0,
			"write_speed_source": "sysbench",
			"score":              35.0,
		},
	}

	formatted := generator.formatSingleTestResult("内存性能测试", result)

	expectedSnippets := []string{
		"测试后端:     sysbench",
		"读取速度:     3200.00 MB/s (sysbench)",
		"写入速度:     2800.00 MB/s (sysbench)",
		"测试评分:     78.67",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(formatted, snippet) {
			t.Fatalf("expected formatted report to contain %q, got:\n%s", snippet, formatted)
		}
	}
}

func TestFormatSingleTestResultShowsFioMixedMatrix(t *testing.T) {
	generator := NewReportGenerator()
	result := &models.TestResult{
		TestName:        "磁盘性能测试",
		Status:          "success",
		DurationSeconds: 20.2,
		Metrics: map[string]interface{}{
			"backend":                   "fio",
			"sequential_read_mbps":      1000.0,
			"sequential_write_mbps":     800.0,
			"random_iops":               3000,
			"fio_mixed_4k_read_mbps":    10.0,
			"fio_mixed_4k_write_mbps":   20.0,
			"fio_mixed_4k_total_mbps":   30.0,
			"fio_mixed_4k_total_iops":   3000.0,
			"fio_mixed_64k_read_mbps":   100.0,
			"fio_mixed_64k_write_mbps":  120.0,
			"fio_mixed_64k_total_mbps":  220.0,
			"fio_mixed_64k_total_iops":  3400.0,
			"fio_mixed_512k_read_mbps":  300.0,
			"fio_mixed_512k_write_mbps": 320.0,
			"fio_mixed_512k_total_mbps": 620.0,
			"fio_mixed_512k_total_iops": 1210.0,
			"fio_mixed_1m_read_mbps":    500.0,
			"fio_mixed_1m_write_mbps":   550.0,
			"fio_mixed_1m_total_mbps":   1050.0,
			"fio_mixed_1m_total_iops":   1050.0,
			"score":                     100.0,
		},
	}

	formatted := generator.formatSingleTestResult("磁盘性能测试", result)

	expectedSnippets := []string{
		"测试后端:     fio",
		"fio混合矩阵:  block | read MB/s | write MB/s | total MB/s | total IOPS",
		"4k |",
		"1m |",
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

func TestWebServerKeyMetricsShowsFioMixedMatrix(t *testing.T) {
	server := &WebServer{}
	result := &models.TestResult{
		TestName: "磁盘性能测试",
		Status:   "success",
		Metrics: map[string]interface{}{
			"backend":                 "fio",
			"fio_mixed_4k_total_iops": 3000.0,
			"fio_mixed_4k_total_mbps": 30.0,
			"fio_mixed_1m_total_iops": 1050.0,
			"fio_mixed_1m_total_mbps": 1050.0,
		},
	}

	metrics := server.getKeyMetrics(result)
	if !strings.Contains(metrics, "fio | fio mixed: 4k 3000 IOPS | 1m 1050.00 MB/s") {
		t.Fatalf("expected web key metrics to show fio mixed matrix, got %q", metrics)
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

func TestCalculateOverallScoreUsesCustomWeights(t *testing.T) {
	calculator := NewScoreCalculatorWithWeights(map[string]float64{
		"cpu":     0.70,
		"memory":  0.10,
		"disk":    0.10,
		"network": 0.10,
	})
	results := &models.TestResults{
		CPUResult: &models.TestResult{
			TestName: "CPU性能测试",
			Status:   "success",
			Metrics: map[string]interface{}{
				"total_score": 50.0,
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
	expectedTotal := (50.0*0.70 + 100.0*0.10) / 0.80
	if math.Abs(overall.TotalScore-expectedTotal) > 0.0001 {
		t.Fatalf("expected custom weighted total %.2f, got %.2f", expectedTotal, overall.TotalScore)
	}
	if overall.Weights["cpu"] != 0.70 {
		t.Fatalf("expected custom cpu weight, got %#v", overall.Weights)
	}
}

func TestScoreProfileChangesMetricBaselines(t *testing.T) {
	result := &models.TestResult{
		TestName: "内存性能测试",
		Status:   "success",
		Metrics: map[string]interface{}{
			"read_speed_mbps":  3000.0,
			"write_speed_mbps": 2000.0,
		},
	}

	vps := NewScoreCalculatorWithWeightsAndProfile(map[string]float64{
		"cpu":     0.30,
		"memory":  0.20,
		"disk":    0.25,
		"network": 0.25,
	}, "vps")
	server := NewScoreCalculatorWithWeightsAndProfile(map[string]float64{
		"cpu":     0.30,
		"memory":  0.20,
		"disk":    0.25,
		"network": 0.25,
	}, "server")

	if vps.CalculateMemoryScore(result) != 100.0 {
		t.Fatalf("expected vps memory score 100, got %.2f", vps.CalculateMemoryScore(result))
	}
	if server.CalculateMemoryScore(result) <= 0 || server.CalculateMemoryScore(result) >= 100 {
		t.Fatalf("expected server score to be lower but positive, got %.2f", server.CalculateMemoryScore(result))
	}
}

func TestBuildScoreBreakdownIncludesFormulaAndActiveWeight(t *testing.T) {
	calculator := NewScoreCalculator()
	results := &models.TestResults{
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

	breakdown := calculator.BuildScoreBreakdown(results, overall)
	memory, ok := breakdown["memory"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected memory breakdown, got %#v", breakdown["memory"])
	}
	if memory["score"] != 100.0 {
		t.Fatalf("expected memory score 100, got %#v", memory["score"])
	}
	if memory["formula"] == "" {
		t.Fatalf("expected memory formula, got %#v", memory)
	}
	normalized, ok := breakdown["normalized_total"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected normalized total breakdown, got %#v", breakdown["normalized_total"])
	}
	if normalized["active_weight"] != 0.20 {
		t.Fatalf("expected active weight 0.20, got %#v", normalized["active_weight"])
	}
	if breakdown["score_profile"] != "server" {
		t.Fatalf("expected server score profile, got %#v", breakdown["score_profile"])
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

func TestAddSummaryCountsDegradedResult(t *testing.T) {
	generator := NewReportGenerator()
	report := &models.Report{
		SessionID:  "session_degraded",
		Timestamp:  time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC),
		SystemInfo: &models.SystemInfo{},
		TestResults: &models.TestResults{
			CPUResult:    &models.TestResult{TestName: "CPU性能测试", Status: models.TestStatusSuccess},
			MemoryResult: &models.TestResult{TestName: "内存性能测试", Status: models.TestStatusSuccess},
			DiskResult:   &models.TestResult{TestName: "磁盘性能测试", Status: models.TestStatusSuccess},
			NetworkResult: &models.TestResult{
				TestName:     "网络性能测试",
				Status:       models.TestStatusDegraded,
				ErrorMessage: "下载速度测试失败",
				Metrics: map[string]interface{}{
					"backend":             models.NetworkBackendBuiltin,
					"average_latency_ms":  10.0,
					"download_speed_mbps": -1.0,
					"upload_speed_mbps":   -1.0,
				},
			},
		},
		Summary: make(map[string]interface{}),
	}

	generator.AddSummary(report, &models.OverallScore{TotalScore: 80, Grade: "良好"})

	if degraded, _ := report.Summary["tests_degraded"].(int); degraded != 1 {
		t.Fatalf("expected one degraded test, got %d", degraded)
	}
	if grade, _ := report.Summary["grade"].(string); grade != "未完成" {
		t.Fatalf("expected degraded report to be incomplete, got %q", grade)
	}
	notes := strings.Join(report.Summary["quality_notes"].([]string), "\n")
	if !strings.Contains(notes, "网络测试降级") {
		t.Fatalf("expected degraded quality note, got %s", notes)
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
			"score_breakdown": map[string]interface{}{
				"cpu": map[string]interface{}{
					"score":   80.0,
					"weight":  0.3,
					"formula": "CPU 分数优先使用 total_score。",
				},
				"normalized_total": map[string]interface{}{
					"score":         76.0,
					"active_weight": 0.3,
				},
			},
		},
	}

	formatted := generator.FormatReport(report)

	expectedSnippets := []string{
		"性能等级:       未完成",
		"说明:           由于未执行所有性能测试，无法给出完整的性能结论。",
		"评分说明:",
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
			CPUResult: &models.TestResult{
				TestName: "CPU性能测试",
				Status:   "success",
				Metrics: map[string]interface{}{
					"backend": "builtin",
				},
			},
			MemoryResult: &models.TestResult{
				TestName: "内存性能测试",
				Status:   "success",
				Metrics: map[string]interface{}{
					"backend": "builtin",
				},
			},
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

	if _, ok := report.Summary["score_breakdown"].(map[string]interface{}); !ok {
		t.Fatalf("expected score breakdown, got %#v", report.Summary["score_breakdown"])
	}

	notes, ok := report.Summary["quality_notes"].([]string)
	if !ok || len(notes) == 0 {
		t.Fatalf("expected quality notes, got %#v", report.Summary["quality_notes"])
	}
	joined := strings.Join(notes, "\n")
	expectedSnippets := []string{
		"CPU 测试使用内置后端",
		"内存测试使用内置后端",
		"磁盘测试使用内置后端",
		"网络上传速度为估算值",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(joined, snippet) {
			t.Fatalf("expected quality notes to contain %q, got %v", snippet, notes)
		}
	}
}

func TestAddSummaryAddsBenchmarkProfileAndConfidence(t *testing.T) {
	generator := NewReportGenerator()
	report := &models.Report{
		SessionID:  "session_profile",
		Timestamp:  time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC),
		SystemInfo: &models.SystemInfo{},
		TestResults: &models.TestResults{
			CPUResult: &models.TestResult{
				TestName: "CPU性能测试",
				Status:   "success",
				Metrics: map[string]interface{}{
					"backend": "sysbench",
				},
			},
			MemoryResult: &models.TestResult{
				TestName: "内存性能测试",
				Status:   "success",
				Metrics: map[string]interface{}{
					"backend": "sysbench",
				},
			},
			DiskResult: &models.TestResult{
				TestName: "磁盘性能测试",
				Status:   "success",
				Metrics: map[string]interface{}{
					"backend": "fio",
				},
			},
			NetworkResult: &models.TestResult{
				TestName: "网络性能测试",
				Status:   "success",
				Metrics: map[string]interface{}{
					"backend":                "iperf3",
					"upload_speed_estimated": false,
				},
			},
		},
		Summary: make(map[string]interface{}),
	}

	generator.AddSummary(report, &models.OverallScore{})

	profile, ok := report.Summary["benchmark_profile"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected benchmark profile, got %#v", report.Summary["benchmark_profile"])
	}
	if profile["name"] != "full_iperf3" {
		t.Fatalf("expected full_iperf3 profile, got %#v", profile)
	}
	if profile["mainstream_count"] != 4 {
		t.Fatalf("expected 4 mainstream backends, got %#v", profile["mainstream_count"])
	}

	confidence, ok := report.Summary["confidence_level"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected confidence level, got %#v", report.Summary["confidence_level"])
	}
	if confidence["level"] != "high" {
		t.Fatalf("expected high confidence, got %#v", confidence)
	}
}

func TestAddSummaryDetectsQuickBenchmarkProfile(t *testing.T) {
	generator := NewReportGenerator()
	report := &models.Report{
		SessionID:  "session_quick",
		Timestamp:  time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC),
		SystemInfo: &models.SystemInfo{},
		TestResults: &models.TestResults{
			CPUResult: &models.TestResult{
				TestName: "CPU性能测试",
				Status:   "success",
				Metrics:  map[string]interface{}{"backend": "builtin"},
			},
			MemoryResult: &models.TestResult{
				TestName: "内存性能测试",
				Status:   "success",
				Metrics:  map[string]interface{}{"backend": "builtin"},
			},
			DiskResult: &models.TestResult{
				TestName: "磁盘性能测试",
				Status:   "success",
				Metrics:  map[string]interface{}{"backend": "builtin"},
			},
		},
		Summary: make(map[string]interface{}),
	}

	generator.AddSummary(report, &models.OverallScore{})

	profile, ok := report.Summary["benchmark_profile"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected benchmark profile, got %#v", report.Summary["benchmark_profile"])
	}
	if profile["name"] != "quick" {
		t.Fatalf("expected quick profile, got %#v", profile)
	}
}
