package compare

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
)

type ReportLabel string

const (
	ReportA ReportLabel = "A"
	ReportB ReportLabel = "B"
)

type Result struct {
	ReportA           string                 `json:"report_a"`
	ReportB           string                 `json:"report_b"`
	Winner            string                 `json:"winner"`
	ScoreProfileA     string                 `json:"score_profile_a"`
	ScoreProfileB     string                 `json:"score_profile_b"`
	Comparable        bool                   `json:"comparable"`
	Warnings          []string               `json:"warnings,omitempty"`
	TotalDifference   MetricDifference       `json:"total_difference"`
	MetricDifferences []MetricDifference     `json:"metric_differences"`
	BenchmarkProfiles map[string]interface{} `json:"benchmark_profiles,omitempty"`
}

type MetricDifference struct {
	Name           string  `json:"name"`
	ReportA        float64 `json:"report_a"`
	ReportB        float64 `json:"report_b"`
	Delta          float64 `json:"delta"`
	DeltaPercent   float64 `json:"delta_percent"`
	Winner         string  `json:"winner"`
	HigherIsBetter bool    `json:"higher_is_better"`
}

func CompareFiles(pathA string, pathB string) (*Result, error) {
	reportA, err := readReport(pathA)
	if err != nil {
		return nil, fmt.Errorf("读取报告 A 失败: %w", err)
	}
	reportB, err := readReport(pathB)
	if err != nil {
		return nil, fmt.Errorf("读取报告 B 失败: %w", err)
	}
	return CompareReports(reportA, reportB, pathA, pathB), nil
}

func CompareReports(reportA *models.Report, reportB *models.Report, labelA string, labelB string) *Result {
	result := &Result{
		ReportA:       labelA,
		ReportB:       labelB,
		ScoreProfileA: scoreProfile(reportA),
		ScoreProfileB: scoreProfile(reportB),
		Comparable:    true,
		BenchmarkProfiles: map[string]interface{}{
			"report_a": summaryObject(reportA, "benchmark_profile"),
			"report_b": summaryObject(reportB, "benchmark_profile"),
		},
	}

	if result.ScoreProfileA != "" && result.ScoreProfileB != "" && result.ScoreProfileA != result.ScoreProfileB {
		result.Comparable = false
		result.Warnings = append(result.Warnings, fmt.Sprintf("两份报告评分基准不同：A=%s, B=%s，不能直接比较总分。", result.ScoreProfileA, result.ScoreProfileB))
	}
	if grade(reportA) == "未完成" || grade(reportB) == "未完成" {
		result.Warnings = append(result.Warnings, "至少一份报告未完成全部核心测试，比较结果仅供参考。")
	}

	result.MetricDifferences = []MetricDifference{
		diff("CPU评分", score(reportA, "cpu_score"), score(reportB, "cpu_score"), true),
		diff("内存评分", score(reportA, "memory_score"), score(reportB, "memory_score"), true),
		diff("磁盘评分", score(reportA, "disk_score"), score(reportB, "disk_score"), true),
		diff("网络评分", score(reportA, "network_score"), score(reportB, "network_score"), true),
	}
	result.TotalDifference = diff("总体评分", score(reportA, "total_score"), score(reportB, "total_score"), true)
	result.Winner = result.TotalDifference.Winner
	if !result.Comparable {
		result.Winner = "不可直接比较"
	}
	return result
}

func FormatText(result *Result) string {
	if result == nil {
		return "比较结果不可用\n"
	}

	var sb strings.Builder
	sb.WriteString("=== 报告对比结果 ===\n\n")
	sb.WriteString(fmt.Sprintf("报告 A: %s\n", result.ReportA))
	sb.WriteString(fmt.Sprintf("报告 B: %s\n", result.ReportB))
	sb.WriteString(fmt.Sprintf("评分基准: A=%s, B=%s\n", emptyAsUnknown(result.ScoreProfileA), emptyAsUnknown(result.ScoreProfileB)))
	sb.WriteString(fmt.Sprintf("可直接比较: %s\n", boolText(result.Comparable)))
	sb.WriteString(fmt.Sprintf("胜出方: %s\n\n", result.Winner))

	if len(result.Warnings) > 0 {
		sb.WriteString("提示:\n")
		for _, warning := range result.Warnings {
			sb.WriteString(fmt.Sprintf("  - %s\n", warning))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("分项差异:\n")
	all := append([]MetricDifference{result.TotalDifference}, result.MetricDifferences...)
	for _, item := range all {
		sb.WriteString(fmt.Sprintf("  - %s: A=%.2f, B=%.2f, 差值=%+.2f (%+.2f%%), 胜出=%s\n",
			item.Name,
			item.ReportA,
			item.ReportB,
			item.Delta,
			item.DeltaPercent,
			item.Winner,
		))
	}
	return sb.String()
}

func FormatJSON(result *Result) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("序列化对比结果失败: %w", err)
	}
	return string(data), nil
}

func readReport(path string) (*models.Report, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var report models.Report
	if err := json.Unmarshal(content, &report); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w", err)
	}
	if report.Summary == nil {
		return nil, fmt.Errorf("报告缺少 summary")
	}
	return &report, nil
}

func diff(name string, a float64, b float64, higherIsBetter bool) MetricDifference {
	delta := b - a
	percent := 0.0
	if a != 0 {
		percent = delta / a * 100
	}

	winner := "平局"
	if delta != 0 {
		if (delta > 0 && higherIsBetter) || (delta < 0 && !higherIsBetter) {
			winner = string(ReportB)
		} else {
			winner = string(ReportA)
		}
	}

	return MetricDifference{
		Name:           name,
		ReportA:        a,
		ReportB:        b,
		Delta:          delta,
		DeltaPercent:   percent,
		Winner:         winner,
		HigherIsBetter: higherIsBetter,
	}
}

func score(report *models.Report, key string) float64 {
	if report == nil || report.Summary == nil {
		return 0
	}
	if value, ok := report.Summary[key].(float64); ok {
		return value
	}
	if overall, ok := report.Summary["overall_score"].(map[string]interface{}); ok {
		if value, ok := overall[key].(float64); ok {
			return value
		}
	}
	if breakdown, ok := report.Summary["score_breakdown"].(map[string]interface{}); ok {
		if key == "total_score" {
			if normalized, ok := breakdown["normalized_total"].(map[string]interface{}); ok {
				if value, ok := normalized["score"].(float64); ok {
					return value
				}
			}
		}
		if component := scoreComponent(key); component != "" {
			if item, ok := breakdown[component].(map[string]interface{}); ok {
				if value, ok := item["score"].(float64); ok {
					return value
				}
			}
		}
	}
	return 0
}

func scoreComponent(key string) string {
	switch key {
	case "cpu_score":
		return "cpu"
	case "memory_score":
		return "memory"
	case "disk_score":
		return "disk"
	case "network_score":
		return "network"
	default:
		return ""
	}
}

func scoreProfile(report *models.Report) string {
	if report == nil || report.Summary == nil {
		return ""
	}
	if value, ok := report.Summary["score_profile"].(string); ok {
		return value
	}
	if breakdown, ok := report.Summary["score_breakdown"].(map[string]interface{}); ok {
		if value, ok := breakdown["score_profile"].(string); ok {
			return value
		}
	}
	return ""
}

func summaryObject(report *models.Report, key string) interface{} {
	if report == nil || report.Summary == nil {
		return nil
	}
	return report.Summary[key]
}

func grade(report *models.Report) string {
	if report == nil || report.Summary == nil {
		return ""
	}
	if value, ok := report.Summary["grade"].(string); ok {
		return value
	}
	return ""
}

func boolText(value bool) string {
	if value {
		return "是"
	}
	return "否"
}

func emptyAsUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}
