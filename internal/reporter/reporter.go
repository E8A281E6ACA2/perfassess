// Package reporter 提供报告生成功能
package reporter

import (
	"fmt"
	"strings"
	"time"

	"performance-assessment-system/internal/models"
)

// ReportGenerator 报告生成器
// 负责生成完整的性能评估报告
type ReportGenerator struct {
	scoreCalculator *ScoreCalculator
}

// NewReportGenerator 创建新的报告生成器
func NewReportGenerator() *ReportGenerator {
	return &ReportGenerator{
		scoreCalculator: NewScoreCalculator(),
	}
}

// GenerateReport 生成完整的评估报告
// 包含系统信息、测试结果、评分和摘要
func (rg *ReportGenerator) GenerateReport(sessionID string, systemInfo *models.SystemInfo, testResults *models.TestResults) (*models.Report, error) {
	if systemInfo == nil || testResults == nil {
		return nil, fmt.Errorf("系统信息或测试结果不能为空")
	}

	// 计算综合评分
	overallScore := rg.scoreCalculator.CalculateOverallScore(testResults)

	// 创建报告
	report := &models.Report{
		SessionID:   sessionID,
		Timestamp:   time.Now(),
		SystemInfo:  systemInfo,
		TestResults: testResults,
		Summary:     make(map[string]interface{}),
	}

	// 添加摘要信息
	rg.AddSummary(report, overallScore)

	// 格式化报告内容
	report.FormattedContent = rg.FormatReport(report)

	return report, nil
}

// FormatSystemInfo 格式化系统信息
// 返回易读的系统信息文本
func (rg *ReportGenerator) FormatSystemInfo(systemInfo *models.SystemInfo) string {
	if systemInfo == nil {
		return "系统信息不可用"
	}

	var sb strings.Builder

	sb.WriteString("=== 系统信息 ===\n\n")

	// CPU信息
	if systemInfo.CPU != nil {
		sb.WriteString(fmt.Sprintf("CPU型号:        %s\n", systemInfo.CPU.Model))
		sb.WriteString(fmt.Sprintf("CPU核心数:      %d 核心\n", systemInfo.CPU.Cores))
		sb.WriteString(fmt.Sprintf("CPU线程数:      %d 线程\n", systemInfo.CPU.Threads))
		sb.WriteString(fmt.Sprintf("CPU频率:        %.2f MHz\n", systemInfo.CPU.FrequencyMHz))
	}

	sb.WriteString("\n")

	// 内存信息
	if systemInfo.Memory != nil {
		sb.WriteString(fmt.Sprintf("内存总量:       %d MB\n", systemInfo.Memory.TotalMB))
		sb.WriteString(fmt.Sprintf("可用内存:       %d MB\n", systemInfo.Memory.AvailableMB))
		if systemInfo.Memory.MemoryType != "" {
			sb.WriteString(fmt.Sprintf("内存类型:       %s\n", systemInfo.Memory.MemoryType))
		}
	}

	sb.WriteString("\n")

	// 磁盘信息
	if systemInfo.Disk != nil {
		sb.WriteString(fmt.Sprintf("磁盘总量:       %.2f GB\n", systemInfo.Disk.TotalGB))
		sb.WriteString(fmt.Sprintf("可用空间:       %.2f GB\n", systemInfo.Disk.AvailableGB))
		if systemInfo.Disk.DiskType != "" {
			sb.WriteString(fmt.Sprintf("磁盘类型:       %s\n", systemInfo.Disk.DiskType))
		}
	}

	sb.WriteString("\n")

	// 操作系统信息
	if systemInfo.OS != nil {
		sb.WriteString(fmt.Sprintf("操作系统:       %s\n", systemInfo.OS.Name))
		sb.WriteString(fmt.Sprintf("系统版本:       %s\n", systemInfo.OS.Version))
		sb.WriteString(fmt.Sprintf("系统架构:       %s\n", systemInfo.OS.Architecture))
	}

	// 虚拟化信息
	if systemInfo.Virtualization != nil {
		sb.WriteString("\n")
		if systemInfo.Virtualization.IsVirtualized {
			sb.WriteString(fmt.Sprintf("虚拟化类型:     %s\n", systemInfo.Virtualization.Type))
			if systemInfo.Virtualization.Vendor != "" {
				sb.WriteString(fmt.Sprintf("虚拟化厂商:     %s\n", systemInfo.Virtualization.Vendor))
			}
		} else {
			sb.WriteString("虚拟化类型:     物理机\n")
		}
	}

	// IP信息
	if systemInfo.IPInfo != nil {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("公网IP:         %s\n", systemInfo.IPInfo.PublicIP))
		if systemInfo.IPInfo.GeoLocation != nil {
			sb.WriteString(fmt.Sprintf("地理位置:       %s, %s\n",
				systemInfo.IPInfo.GeoLocation.Country,
				systemInfo.IPInfo.GeoLocation.City))
		}
		if systemInfo.IPInfo.ISP != "" {
			sb.WriteString(fmt.Sprintf("ISP:            %s\n", systemInfo.IPInfo.ISP))
		}
	}

	return sb.String()
}

// FormatTestResults 格式化测试结果
// 返回易读的测试结果文本
func (rg *ReportGenerator) FormatTestResults(testResults *models.TestResults) string {
	if testResults == nil {
		return "测试结果不可用"
	}

	var sb strings.Builder

	sb.WriteString("=== 性能测试结果 ===\n\n")

	// CPU测试结果
	sb.WriteString(rg.formatSingleTestResult("CPU性能测试", testResults.CPUResult))

	// 内存测试结果
	sb.WriteString(rg.formatSingleTestResult("内存性能测试", testResults.MemoryResult))

	// 磁盘测试结果
	sb.WriteString(rg.formatSingleTestResult("磁盘性能测试", testResults.DiskResult))

	// 网络测试结果
	sb.WriteString(rg.formatSingleTestResult("网络性能测试", testResults.NetworkResult))

	return sb.String()
}

// formatSingleTestResult 格式化单个测试结果
func (rg *ReportGenerator) formatSingleTestResult(testName string, result *models.TestResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("--- %s ---\n", testName))

	if result == nil {
		sb.WriteString("状态: 未执行\n\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("状态:           %s\n", result.Status))
	sb.WriteString(fmt.Sprintf("耗时:           %.2f 秒\n", result.DurationSeconds))

	if result.Status == "success" && result.Metrics != nil {
		sb.WriteString("测试指标:\n")

		// 根据不同测试类型格式化指标
		switch result.TestName {
		case "cpu", "CPU性能测试":
			if score, ok := metricFloat64(result.Metrics, "single_core_score"); ok {
				sb.WriteString(fmt.Sprintf("  单核评分:     %.2f\n", score))
			}
			if score, ok := metricFloat64(result.Metrics, "multi_core_score"); ok {
				sb.WriteString(fmt.Sprintf("  多核评分:     %.2f\n", score))
			}
			if score, ok := getCPUScore(result); ok {
				sb.WriteString(fmt.Sprintf("  总体评分:     %.2f\n", score))
			}
			if cores, ok := metricInt(result.Metrics, "cpu_cores"); ok {
				sb.WriteString(fmt.Sprintf("  CPU核心数:    %d\n", cores))
			}

		case "memory", "内存性能测试":
			if speed, ok := getMemoryReadSpeed(result); ok {
				sb.WriteString(fmt.Sprintf("  读取速度:     %.2f MB/s\n", speed))
			}
			if speed, ok := getMemoryWriteSpeed(result); ok {
				sb.WriteString(fmt.Sprintf("  写入速度:     %.2f MB/s\n", speed))
			}
			if score, ok := metricFloat64(result.Metrics, "score"); ok {
				sb.WriteString(fmt.Sprintf("  测试评分:     %.2f\n", score))
			}

		case "disk", "磁盘性能测试":
			if speed, ok := getDiskReadSpeed(result); ok {
				sb.WriteString(fmt.Sprintf("  顺序读取:     %.2f MB/s\n", speed))
			}
			if speed, ok := getDiskWriteSpeed(result); ok {
				sb.WriteString(fmt.Sprintf("  顺序写入:     %.2f MB/s\n", speed))
			}
			if iops, ok := getDiskRandomIOPS(result); ok {
				sb.WriteString(fmt.Sprintf("  随机IOPS:     %d\n", iops))
			}
			if score, ok := metricFloat64(result.Metrics, "score"); ok {
				sb.WriteString(fmt.Sprintf("  测试评分:     %.2f\n", score))
			}

		case "network", "网络性能测试":
			if message := getNetworkError(result); message != "" {
				sb.WriteString(fmt.Sprintf("  网络说明:     %s\n", message))
			}
			if latency, ok := getNetworkLatency(result); ok {
				source := getNetworkLatencySource(result)
				if source != "" {
					sb.WriteString(fmt.Sprintf("  平均延迟:     %.2f ms (%s)\n", latency, source))
				} else {
					sb.WriteString(fmt.Sprintf("  平均延迟:     %.2f ms\n", latency))
				}
			}
			if speed, ok := getNetworkDownloadSpeed(result); ok {
				source := getNetworkDownloadSource(result)
				if source != "" {
					sb.WriteString(fmt.Sprintf("  下载速度:     %.2f Mbps (%s)\n", speed, source))
				} else {
					sb.WriteString(fmt.Sprintf("  下载速度:     %.2f Mbps\n", speed))
				}
			}
			if speed, ok := getNetworkUploadSpeed(result); ok {
				source := getNetworkUploadSource(result)
				if source != "" {
					sb.WriteString(fmt.Sprintf("  上传速度:     %.2f Mbps (%s)\n", speed, source))
				} else {
					sb.WriteString(fmt.Sprintf("  上传速度:     %.2f Mbps\n", speed))
				}
				if isNetworkUploadEstimated(result) {
					sb.WriteString("  上传说明:     当前结果为估算值，不参与真实上传评分\n")
				}
			}
			if score, ok := metricFloat64(result.Metrics, "score"); ok {
				sb.WriteString(fmt.Sprintf("  测试评分:     %.2f\n", score))
			}
		}
	} else if result.Status == "failed" && result.ErrorMessage != "" {
		sb.WriteString(fmt.Sprintf("错误信息:       %s\n", result.ErrorMessage))
	} else if result.Status == "skipped" && result.ErrorMessage != "" {
		sb.WriteString(fmt.Sprintf("跳过原因:       %s\n", result.ErrorMessage))
	}

	sb.WriteString("\n")
	return sb.String()
}

// AddSummary 添加摘要信息到报告
// 包含综合评分、性能等级等关键信息
func (rg *ReportGenerator) AddSummary(report *models.Report, overallScore *models.OverallScore) {
	if report == nil || overallScore == nil {
		return
	}

	report.Summary["overall_score"] = overallScore
	report.Summary["total_score"] = overallScore.TotalScore
	report.Summary["grade"] = overallScore.Grade
	report.Summary["cpu_score"] = overallScore.CPUScore
	report.Summary["memory_score"] = overallScore.MemoryScore
	report.Summary["disk_score"] = overallScore.DiskScore
	report.Summary["network_score"] = overallScore.NetworkScore

	// 统计测试执行情况
	successCount := 0
	failedCount := 0
	skippedCount := 0

	if report.TestResults != nil {
		for _, result := range []*models.TestResult{
			report.TestResults.CPUResult,
			report.TestResults.MemoryResult,
			report.TestResults.DiskResult,
			report.TestResults.NetworkResult,
		} {
			if result != nil {
				switch result.Status {
				case "success":
					successCount++
				case "failed":
					failedCount++
				case "skipped":
					skippedCount++
				}
			}
		}
	}

	report.Summary["tests_success"] = successCount
	report.Summary["tests_failed"] = failedCount
	report.Summary["tests_skipped"] = skippedCount

	allCompleted := successCount == 4 && report.TestResults != nil &&
		report.TestResults.CPUResult != nil && report.TestResults.CPUResult.Status == "success" &&
		report.TestResults.MemoryResult != nil && report.TestResults.MemoryResult.Status == "success" &&
		report.TestResults.DiskResult != nil && report.TestResults.DiskResult.Status == "success" &&
		report.TestResults.NetworkResult != nil && report.TestResults.NetworkResult.Status == "success"

	if !allCompleted {
		note := "由于未执行所有性能测试，无法给出完整的性能结论。"
		report.Summary["performance_note"] = note
		report.Summary["grade"] = "未完成"
		overallScore.Grade = "未完成"
	}
}

// formatReport 格式化完整报告
func (rg *ReportGenerator) FormatReport(report *models.Report) string {
	var sb strings.Builder

	// 报告标题
	sb.WriteString("╔════════════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║          高性能多终端自动化性能评估系统 - 评估报告            ║\n")
	sb.WriteString("╚════════════════════════════════════════════════════════════════╝\n\n")

	// 会话信息
	sb.WriteString(fmt.Sprintf("会话ID:         %s\n", report.SessionID))
	sb.WriteString(fmt.Sprintf("报告时间:       %s\n\n", report.Timestamp.Format("2006-01-02 15:04:05")))

	// 系统信息
	sb.WriteString(rg.FormatSystemInfo(report.SystemInfo))
	sb.WriteString("\n")

	// 测试结果
	sb.WriteString(rg.FormatTestResults(report.TestResults))
	sb.WriteString("\n")

	// 综合评分
	sb.WriteString("=== 综合性能评分 ===\n\n")
	if overall, ok := report.Summary["overall_score"].(*models.OverallScore); ok && overall != nil {
		sb.WriteString(fmt.Sprintf("CPU评分:        %.2f / 100\n", overall.CPUScore))
		sb.WriteString(fmt.Sprintf("内存评分:       %.2f / 100\n", overall.MemoryScore))
		sb.WriteString(fmt.Sprintf("磁盘评分:       %.2f / 100\n", overall.DiskScore))
		sb.WriteString(fmt.Sprintf("网络评分:       %.2f / 100\n", overall.NetworkScore))
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("总体评分:       %.2f / 100\n", overall.TotalScore))
		sb.WriteString(fmt.Sprintf("性能等级:       %s\n", overall.Grade))
	}

	if note, ok := report.Summary["performance_note"].(string); ok && note != "" {
		sb.WriteString(fmt.Sprintf("说明:           %s\n", note))
	}

	sb.WriteString("\n")

	// 测试统计
	if report.Summary != nil {
		sb.WriteString("=== 测试统计 ===\n\n")
		sb.WriteString(fmt.Sprintf("成功测试:       %d\n", report.Summary["tests_success"]))
		sb.WriteString(fmt.Sprintf("失败测试:       %d\n", report.Summary["tests_failed"]))
		sb.WriteString(fmt.Sprintf("跳过测试:       %d\n", report.Summary["tests_skipped"]))

		// 路由追踪结果
		if routeResults, ok := report.Summary["route_trace_results"].([]*models.TraceResult); ok && len(routeResults) > 0 {
			sb.WriteString("\n=== 路由追踪结果 ===\n\n")
			for _, result := range routeResults {
				sb.WriteString(fmt.Sprintf("目标: %s\n", result.Target))
				if result.Success {
					sb.WriteString(fmt.Sprintf("总跳数: %d\n", result.TotalHops))
					for _, hop := range result.Hops {
						if hop.IP == "*" {
							sb.WriteString(fmt.Sprintf("  %2d  *  (超时)\n", hop.Number))
						} else {
							sb.WriteString(fmt.Sprintf("  %2d  %-15s  %.2f ms\n", hop.Number, hop.IP, float64(hop.Latency.Microseconds())/1000.0))
						}
					}
				} else {
					sb.WriteString(fmt.Sprintf("状态: 失败 - %s\n", result.ErrorMessage))
				}
				sb.WriteString("\n")
			}
		}

		// 流媒体检测结果
		if streamingResults, ok := report.Summary["streaming_results"].(map[string]*models.StreamingResult); ok && len(streamingResults) > 0 {
			sb.WriteString("=== 流媒体解锁检测 ===\n\n")
			for platform, result := range streamingResults {
				status := "❌ 不可用"
				if result.Available {
					status = "✅ 可用"
					if result.Region != "" && result.Region != "Unknown" {
						status += fmt.Sprintf(" (%s)", result.Region)
					}
				}
				sb.WriteString(fmt.Sprintf("%-15s  %s  - %s\n", platform, status, result.Message))
			}
			sb.WriteString("\n")
		}

		// AI 服务检测结果
		if aiResults, ok := report.Summary["ai_results"].(map[string]*models.AIServiceResult); ok && len(aiResults) > 0 {
			sb.WriteString("=== AI服务检测 ===\n\n")
			for service, result := range aiResults {
				status := "❌ 不可用"
				if result.Available {
					status = "✅ 可用"
				}
				sb.WriteString(fmt.Sprintf("%-15s  %s  - %s\n", service, status, result.Message))
			}
			sb.WriteString("\n")
		}

		// 压力测试结果
		if stressReport, ok := report.Summary["stress_report"].(*models.StressTestReport); ok && stressReport != nil {
			sb.WriteString("=== 长时间压力测试 ===\n\n")
			tempNote := "温度数据: 未提供"
			if stressReport.TemperatureAvailable {
				tempNote = "温度数据: 可获取"
			}
			sb.WriteString(fmt.Sprintf("%s\n总耗时: %.0f 秒\n\n", tempNote, stressReport.TotalDurationSeconds))
			for _, comp := range stressReport.Components {
				sb.WriteString(fmt.Sprintf("[%s] 状态: %s，持续 %.0f 秒\n", comp.Name, comp.Status, comp.DurationSeconds))
				if comp.AverageTemperature > 0 {
					sb.WriteString(fmt.Sprintf("  平均温度: %.1f°C，峰值: %.1f°C\n", comp.AverageTemperature, comp.PeakTemperature))
				}
				if len(comp.Metrics) > 0 {
					for key, val := range comp.Metrics {
						sb.WriteString(fmt.Sprintf("  %s: %v\n", key, val))
					}
				}
				if comp.Notes != "" {
					sb.WriteString(fmt.Sprintf("  备注: %s\n", comp.Notes))
				}
				sb.WriteString("\n")
			}
		}

		if securityReport, ok := report.Summary["security_report"].(*models.SecurityReport); ok && securityReport != nil {
			sb.WriteString("=== 安全体检 ===\n\n")
			if len(securityReport.Findings) == 0 {
				sb.WriteString("未发现安全提示。\n\n")
			} else {
				for _, finding := range securityReport.Findings {
					sb.WriteString(fmt.Sprintf("[%s][%s] %s\n", strings.ToUpper(finding.Category), strings.ToUpper(finding.Severity), finding.Title))
					if finding.Detail != "" {
						sb.WriteString(fmt.Sprintf("  详情: %s\n", finding.Detail))
					}
					if finding.Advice != "" {
						sb.WriteString(fmt.Sprintf("  建议: %s\n", finding.Advice))
					}
					sb.WriteString("\n")
				}
			}
		}
	}

	sb.WriteString("════════════════════════════════════════════════════════════════\n")
	sb.WriteString("                         报告结束\n")
	sb.WriteString("════════════════════════════════════════════════════════════════\n")

	return sb.String()
}
