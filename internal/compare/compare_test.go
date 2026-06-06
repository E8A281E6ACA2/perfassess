package compare

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
)

func TestCompareReportsFindsWinnerAndDifferences(t *testing.T) {
	a := testReport(80, 70, 90, 60, 75, "server", "良好")
	b := testReport(90, 80, 95, 70, 85, "server", "良好")

	result := CompareReports(a, b, "a.json", "b.json")

	if !result.Comparable {
		t.Fatalf("expected reports to be comparable, got warnings %v", result.Warnings)
	}
	if result.Winner != "B" {
		t.Fatalf("expected B to win, got %q", result.Winner)
	}
	if result.TotalDifference.Delta != 10 {
		t.Fatalf("expected total delta 10, got %.2f", result.TotalDifference.Delta)
	}
	if len(result.MetricDifferences) != 4 {
		t.Fatalf("expected four metric differences, got %d", len(result.MetricDifferences))
	}
}

func TestCompareReportsWarnsWhenScoreProfileDiffers(t *testing.T) {
	a := testReport(80, 70, 90, 60, 75, "server", "良好")
	b := testReport(90, 80, 95, 70, 85, "vps", "良好")

	result := CompareReports(a, b, "a.json", "b.json")

	if result.Comparable {
		t.Fatal("expected different score profiles to be non-comparable")
	}
	if result.Winner != "不可直接比较" {
		t.Fatalf("expected non-comparable winner text, got %q", result.Winner)
	}
	if len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], "评分基准不同") {
		t.Fatalf("expected score profile warning, got %v", result.Warnings)
	}
}

func TestCompareFilesReadsJSONReports(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, "a.json")
	pathB := filepath.Join(dir, "b.json")
	writeReport(t, pathA, testReport(80, 70, 90, 60, 75, "server", "良好"))
	writeReport(t, pathB, testReport(85, 75, 95, 65, 80, "server", "良好"))

	result, err := CompareFiles(pathA, pathB)
	if err != nil {
		t.Fatalf("expected compare files to succeed, got %v", err)
	}
	if result.Winner != "B" {
		t.Fatalf("expected B to win, got %q", result.Winner)
	}
}

func TestCompareReportsFallsBackToScoreBreakdown(t *testing.T) {
	a := testReport(0, 0, 0, 0, 0, "server", "良好")
	b := testReport(0, 0, 0, 0, 0, "server", "良好")
	a.Summary = map[string]interface{}{
		"score_profile": "server",
		"score_breakdown": map[string]interface{}{
			"cpu":              map[string]interface{}{"score": 80.0},
			"memory":           map[string]interface{}{"score": 70.0},
			"disk":             map[string]interface{}{"score": 90.0},
			"network":          map[string]interface{}{"score": 60.0},
			"normalized_total": map[string]interface{}{"score": 75.0},
		},
	}
	b.Summary = map[string]interface{}{
		"score_profile": "server",
		"score_breakdown": map[string]interface{}{
			"cpu":              map[string]interface{}{"score": 90.0},
			"memory":           map[string]interface{}{"score": 80.0},
			"disk":             map[string]interface{}{"score": 95.0},
			"network":          map[string]interface{}{"score": 70.0},
			"normalized_total": map[string]interface{}{"score": 85.0},
		},
	}

	result := CompareReports(a, b, "a.json", "b.json")
	if result.TotalDifference.ReportA != 75 || result.TotalDifference.ReportB != 85 {
		t.Fatalf("expected total score fallback from breakdown, got %#v", result.TotalDifference)
	}
	if result.MetricDifferences[0].ReportA != 80 || result.MetricDifferences[0].ReportB != 90 {
		t.Fatalf("expected cpu score fallback from breakdown, got %#v", result.MetricDifferences[0])
	}
}

func TestFormatTextIncludesWarningsAndMetrics(t *testing.T) {
	result := CompareReports(
		testReport(80, 70, 90, 60, 75, "server", "良好"),
		testReport(90, 80, 95, 70, 85, "vps", "良好"),
		"a.json",
		"b.json",
	)

	text := FormatText(result)
	for _, snippet := range []string{"报告 A: a.json", "可直接比较: 否", "总体评分", "CPU评分"} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("expected text to contain %q, got:\n%s", snippet, text)
		}
	}
}

func TestFormatJSONOutputsMachineReadableResult(t *testing.T) {
	result := CompareReports(
		testReport(80, 70, 90, 60, 75, "server", "良好"),
		testReport(90, 80, 95, 70, 85, "server", "良好"),
		"a.json",
		"b.json",
	)

	content, err := FormatJSON(result)
	if err != nil {
		t.Fatalf("expected json formatting to succeed, got %v", err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(content), &decoded); err != nil {
		t.Fatalf("expected valid json, got %v\n%s", err, content)
	}
	if decoded["winner"] != "B" {
		t.Fatalf("expected winner B, got %#v", decoded["winner"])
	}
}

func testReport(cpu, memory, disk, network, total float64, profile string, grade string) *models.Report {
	return &models.Report{
		SessionID: "test",
		Timestamp: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		Summary: map[string]interface{}{
			"cpu_score":     cpu,
			"memory_score":  memory,
			"disk_score":    disk,
			"network_score": network,
			"total_score":   total,
			"grade":         grade,
			"score_profile": profile,
			"benchmark_profile": map[string]interface{}{
				"name": "custom",
			},
		},
	}
}

func writeReport(t *testing.T, path string, report *models.Report) {
	t.Helper()
	content, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("failed to marshal report: %v", err)
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("failed to write report: %v", err)
	}
}
