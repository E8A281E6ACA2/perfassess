// Package reporter 提供报告生成功能
package reporter

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/E8A281E6ACA2/perfassess/internal/config"
	"github.com/E8A281E6ACA2/perfassess/internal/models"
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

func NewReportGeneratorWithWeights(weights map[string]float64) *ReportGenerator {
	return NewReportGeneratorWithWeightsAndProfile(weights, config.DefaultScoreProfile)
}

func NewReportGeneratorWithWeightsAndProfile(weights map[string]float64, profile string) *ReportGenerator {
	return &ReportGenerator{
		scoreCalculator: NewScoreCalculatorWithWeightsAndProfile(weights, profile),
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
			if backend := getCPUBackend(result); backend != "" {
				sb.WriteString(fmt.Sprintf("  测试后端:     %s\n", backend))
			}
			if score, ok := metricFloat64(result.Metrics, "single_core_score"); ok {
				sb.WriteString(fmt.Sprintf("  单核评分:     %.2f\n", score))
			}
			if rawScore, ok := getCPUSingleCoreRawScore(result); ok {
				sb.WriteString(fmt.Sprintf("  单核原始分:   %.0f\n", rawScore))
			}
			if events, ok := getCPUSingleCoreEvents(result); ok {
				sb.WriteString(fmt.Sprintf("  单核吞吐:     %.2f events/s\n", events))
			}
			if score, ok := metricFloat64(result.Metrics, "multi_core_score"); ok {
				sb.WriteString(fmt.Sprintf("  多核评分:     %.2f\n", score))
			}
			if rawScore, ok := getCPUMultiCoreRawScore(result); ok {
				sb.WriteString(fmt.Sprintf("  多核原始分:   %.0f\n", rawScore))
			}
			if events, ok := getCPUMultiCoreEvents(result); ok {
				sb.WriteString(fmt.Sprintf("  多核吞吐:     %.2f events/s\n", events))
			}
			if score, ok := getCPUScore(result); ok {
				sb.WriteString(fmt.Sprintf("  总体评分:     %.2f\n", score))
			}
			if stddev, ok := getCPUScoreStdDev(result); ok {
				sb.WriteString(fmt.Sprintf("  评分波动:     %.2f stddev\n", stddev))
			}
			if cores, ok := metricInt(result.Metrics, "cpu_cores"); ok {
				sb.WriteString(fmt.Sprintf("  CPU核心数:    %d\n", cores))
			}

		case "memory", "内存性能测试":
			if backend := getMemoryBackend(result); backend != "" {
				sb.WriteString(fmt.Sprintf("  测试后端:     %s\n", backend))
			}
			if speed, ok := getMemoryReadSpeed(result); ok {
				if source := getMemoryReadSource(result); source != "" {
					sb.WriteString(fmt.Sprintf("  读取速度:     %.2f MB/s (%s)\n", speed, source))
				} else {
					sb.WriteString(fmt.Sprintf("  读取速度:     %.2f MB/s\n", speed))
				}
			}
			if stddev, ok := getMemoryReadStdDev(result); ok {
				sb.WriteString(fmt.Sprintf("  读取波动:     %.2f MB/s stddev\n", stddev))
			}
			if speed, ok := getMemoryWriteSpeed(result); ok {
				if source := getMemoryWriteSource(result); source != "" {
					sb.WriteString(fmt.Sprintf("  写入速度:     %.2f MB/s (%s)\n", speed, source))
				} else {
					sb.WriteString(fmt.Sprintf("  写入速度:     %.2f MB/s\n", speed))
				}
			}
			if stddev, ok := getMemoryWriteStdDev(result); ok {
				sb.WriteString(fmt.Sprintf("  写入波动:     %.2f MB/s stddev\n", stddev))
			}
			if _, readOK := getMemoryReadSpeed(result); readOK {
				if _, writeOK := getMemoryWriteSpeed(result); writeOK {
					score := rg.scoreCalculator.CalculateMemoryScore(result)
					sb.WriteString(fmt.Sprintf("  测试评分:     %.2f\n", score))
					break
				}
			}
			if score, ok := metricFloat64(result.Metrics, "score"); ok {
				sb.WriteString(fmt.Sprintf("  测试评分:     %.2f\n", score))
			}

		case "disk", "磁盘性能测试":
			if backend := getDiskBackend(result); backend != "" {
				sb.WriteString(fmt.Sprintf("  测试后端:     %s\n", backend))
			}
			if speed, ok := getDiskReadSpeed(result); ok {
				sb.WriteString(fmt.Sprintf("  顺序读取:     %.2f MB/s\n", speed))
			}
			if speed, ok := getDiskWriteSpeed(result); ok {
				sb.WriteString(fmt.Sprintf("  顺序写入:     %.2f MB/s\n", speed))
			}
			if iops, ok := getDiskRandomIOPS(result); ok {
				sb.WriteString(fmt.Sprintf("  随机IOPS:     %d\n", iops))
			}
			if readIOPS, ok := getDiskRandomReadIOPS(result); ok {
				sb.WriteString(fmt.Sprintf("  随机读IOPS:   %.2f\n", readIOPS))
			}
			if writeIOPS, ok := getDiskRandomWriteIOPS(result); ok {
				sb.WriteString(fmt.Sprintf("  随机写IOPS:   %.2f\n", writeIOPS))
			}
			if readP95, ok := getDiskRandomReadP95Latency(result); ok {
				sb.WriteString(fmt.Sprintf("  随机读P95:    %.2f ms\n", readP95))
			}
			if writeP95, ok := getDiskRandomWriteP95Latency(result); ok {
				sb.WriteString(fmt.Sprintf("  随机写P95:    %.2f ms\n", writeP95))
			}
			if rows := getDiskFioMixedRows(result); len(rows) > 0 {
				sb.WriteString("  fio混合矩阵:  block | read MB/s | write MB/s | total MB/s | total IOPS\n")
				for _, row := range rows {
					sb.WriteString(fmt.Sprintf("                 %4s | %9.2f | %10.2f | %10.2f | %10.2f\n",
						row.BlockSize, row.ReadMBps, row.WriteMBps, row.TotalMBps, row.TotalIOPS))
				}
			}
			if _, readOK := getDiskReadSpeed(result); readOK {
				if _, writeOK := getDiskWriteSpeed(result); writeOK {
					if _, iopsOK := getDiskRandomIOPS(result); iopsOK {
						score := rg.scoreCalculator.CalculateDiskScore(result)
						sb.WriteString(fmt.Sprintf("  测试评分:     %.2f\n", score))
						break
					}
				}
			}
			if score, ok := metricFloat64(result.Metrics, "score"); ok {
				sb.WriteString(fmt.Sprintf("  测试评分:     %.2f\n", score))
			}

		case "network", "网络性能测试":
			if backend := getNetworkBackend(result); backend != "" {
				sb.WriteString(fmt.Sprintf("  测试后端:     %s\n", backend))
			}
			if server := getNetworkBackendServer(result); server != "" {
				sb.WriteString(fmt.Sprintf("  后端服务端:   %s\n", server))
			}
			if message := getNetworkError(result); message != "" {
				sb.WriteString(fmt.Sprintf("  网络说明:     %s\n", message))
			}
			rg.writeBenchmarkErrorSummary(&sb, result, "  ")
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
					sb.WriteString("  上传说明:     当前结果为估算值，不参与真实上传评分，网络评分上限为 85\n")
				}
			}
			if node, ok := getNetworkSpeedtestNode(result); ok {
				sb.WriteString("  speedtest节点: server | location | country | host | isp\n")
				sb.WriteString(fmt.Sprintf("                 %s | %s | %s | %s | %s\n",
					networkSpeedtestServerLabel(node), node.Location, node.Country, node.Host, node.ISP))
				if node.PingJitter > 0 {
					sb.WriteString(fmt.Sprintf("  speedtest抖动: %.2f ms\n", node.PingJitter))
				}
				if node.ExternalIP != "" || node.Interface != "" {
					sb.WriteString(fmt.Sprintf("  speedtest出口: %s | interface %s | vpn %t\n",
						node.ExternalIP, node.Interface, node.IsVPN))
				}
				if node.ResultURL != "" {
					sb.WriteString(fmt.Sprintf("  speedtest结果: %s\n", node.ResultURL))
				}
			}
			if rows := getNetworkIperf3MatrixRows(result); len(rows) > 0 {
				sb.WriteString("  iperf3矩阵:   node | region | provider | auth | proto | latency ms | download Mbps | upload Mbps\n")
				for _, row := range rows {
					sb.WriteString(fmt.Sprintf("                 %s | %s | %s | %s | %s | %.2f | %.2f | %.2f\n",
						iperf3MatrixNodeLabel(row), fallbackText(row.Region, "-"), fallbackText(row.Provider, "-"), fallbackText(row.Authorization, "-"), row.Protocol, row.LatencyMs, row.DownloadMbps, row.UploadMbps))
					if row.Error != "" {
						sb.WriteString(fmt.Sprintf("                 节点错误: %s\n", row.Error))
					}
				}
			}
			if rows := getNetworkQualityRows(result); len(rows) > 0 {
				sb.WriteString("  网络质量矩阵: target | proto | available | avg ms | jitter ms | fail %\n")
				for _, row := range rows {
					sb.WriteString(fmt.Sprintf("                 %s | %s | %t | %.2f | %.2f | %.2f%%\n",
						row.Target, row.Protocol, row.Available, row.AvgLatencyMs, row.JitterMs, row.FailureRate*100))
				}
			}
			if _, latencyOK := getNetworkLatency(result); latencyOK {
				if _, downloadOK := getNetworkDownloadSpeed(result); downloadOK {
					score := rg.scoreCalculator.CalculateNetworkScore(result)
					sb.WriteString(fmt.Sprintf("  测试评分:     %.2f\n", score))
					break
				}
			}
			if score, ok := metricFloat64(result.Metrics, "score"); ok {
				sb.WriteString(fmt.Sprintf("  测试评分:     %.2f\n", score))
			}
		}
	} else if result.Status == "failed" && result.ErrorMessage != "" {
		sb.WriteString(fmt.Sprintf("错误信息:       %s\n", result.ErrorMessage))
		rg.writeBenchmarkErrorSummary(&sb, result, "")
	} else if result.Status == "skipped" && result.ErrorMessage != "" {
		sb.WriteString(fmt.Sprintf("跳过原因:       %s\n", result.ErrorMessage))
		rg.writeBenchmarkErrorSummary(&sb, result, "")
	}

	sb.WriteString("\n")
	return sb.String()
}

func (rg *ReportGenerator) writeBenchmarkErrorSummary(sb *strings.Builder, result *models.TestResult, prefix string) {
	summary := getBenchmarkErrorSummary(result)
	if summary.Category == "" && summary.Stage == "" && summary.Hint == "" {
		return
	}
	if summary.Category != "" {
		sb.WriteString(fmt.Sprintf("%s错误分类:     %s\n", prefix, benchmarkErrorCategoryText(summary.Category)))
	}
	if summary.Stage != "" {
		sb.WriteString(fmt.Sprintf("%s错误阶段:     %s\n", prefix, summary.Stage))
	}
	if summary.Hint != "" {
		sb.WriteString(fmt.Sprintf("%s处理建议:     %s\n", prefix, summary.Hint))
	}
}

func iperf3MatrixNodeLabel(row networkIperf3MatrixRow) string {
	if row.Name != "" {
		return row.Name + " (" + row.Server + ")"
	}
	return row.Server
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
	report.Summary["quality_notes"] = rg.buildQualityNotes(report.TestResults)
	report.Summary["benchmark_profile"] = rg.buildBenchmarkProfile(report.TestResults)
	report.Summary["confidence_level"] = rg.buildConfidenceLevel(report.TestResults)
	report.Summary["score_calibration"] = rg.scoreCalculator.BuildScoreCalibration()
	report.Summary["score_breakdown"] = rg.scoreCalculator.BuildScoreBreakdown(report.TestResults, overallScore)
	report.Summary["score_profile"] = rg.scoreCalculator.profile.Name

	// 统计测试执行情况
	successCount := 0
	failedCount := 0
	skippedCount := 0
	degradedCount := 0

	if report.TestResults != nil {
		for _, result := range []*models.TestResult{
			report.TestResults.CPUResult,
			report.TestResults.MemoryResult,
			report.TestResults.DiskResult,
			report.TestResults.NetworkResult,
		} {
			if result != nil {
				switch result.Status {
				case models.TestStatusSuccess:
					successCount++
				case models.TestStatusFailed:
					failedCount++
				case models.TestStatusSkipped:
					skippedCount++
				case models.TestStatusDegraded:
					degradedCount++
				}
			}
		}
	}

	report.Summary["tests_success"] = successCount
	report.Summary["tests_failed"] = failedCount
	report.Summary["tests_skipped"] = skippedCount
	report.Summary["tests_degraded"] = degradedCount

	allCompleted := successCount == 4 && report.TestResults != nil &&
		report.TestResults.CPUResult != nil && report.TestResults.CPUResult.Status == models.TestStatusSuccess &&
		report.TestResults.MemoryResult != nil && report.TestResults.MemoryResult.Status == models.TestStatusSuccess &&
		report.TestResults.DiskResult != nil && report.TestResults.DiskResult.Status == models.TestStatusSuccess &&
		report.TestResults.NetworkResult != nil && report.TestResults.NetworkResult.Status == models.TestStatusSuccess

	if !allCompleted {
		note := "由于未执行所有性能测试，无法给出完整的性能结论。"
		report.Summary["performance_note"] = note
		report.Summary["grade"] = "未完成"
		overallScore.Grade = "未完成"
	}
	report.Summary["vps_benchmark_summary"] = rg.buildVPSBenchmarkSummary(report, overallScore)
	report.Summary["assessment_conclusion"] = rg.buildAssessmentConclusion(report, overallScore)
	report.Summary["share_templates"] = rg.buildShareTemplates(report)
	rg.RefreshModuleAssessments(report)
}

func (rg *ReportGenerator) RefreshModuleAssessments(report *models.Report) {
	if report == nil || report.Summary == nil {
		return
	}
	report.Summary["module_assessments"] = rg.buildModuleAssessments(report)
}

func (rg *ReportGenerator) buildModuleAssessments(report *models.Report) map[string]interface{} {
	modules := map[string]interface{}{}
	modules["cpu"] = rg.buildCPUModuleAssessment(report)
	modules["memory"] = rg.buildMemoryModuleAssessment(report)
	modules["disk"] = rg.buildDiskModuleAssessment(report)
	modules["network"] = rg.buildNetworkModuleAssessment(report)
	modules["route"] = rg.buildRouteModuleAssessment(report)
	modules["ip_quality"] = rg.buildIPQualityModuleAssessment(report)
	modules["streaming"] = rg.buildStreamingModuleAssessment(report)
	modules["ai_services"] = rg.buildAIModuleAssessment(report)
	return modules
}

func (rg *ReportGenerator) buildCPUModuleAssessment(report *models.Report) map[string]interface{} {
	assessment := moduleAssessmentBase("cpu", "CPU 性能", "skipped", "low", "CPU 测试未执行。")
	if report == nil || report.TestResults == nil || report.TestResults.CPUResult == nil {
		assessment["recommendations"] = []string{"运行包含 CPU 的测评档位后再判断计算能力。"}
		return assessment
	}

	result := report.TestResults.CPUResult
	status := moduleStatusFromTestStatus(result.Status)
	backend := fallbackText(getCPUBackend(result), "-")
	confidence := confidenceFromCoreResult(result, backend != models.CPUBackendBuiltin)
	summary := fmt.Sprintf("CPU 后端 %s，状态 %s。", backend, statusText(result.Status))
	evidence := []map[string]interface{}{
		moduleEvidence("backend", "测试后端", backend, backendEvidenceStatus(backend, models.CPUBackendBuiltin), "sysbench 或 Geekbench 更适合公开横向比较；builtin 更适合快速验收。"),
	}
	if single, ok := metricFloat64(result.Metrics, "single_core_score"); ok {
		evidence = append(evidence, moduleEvidence("single_core", "单核评分", fmt.Sprintf("%.2f", single), evidenceStatus(single, 75, 60), "单核评分影响动态语言、控制面和低并发请求响应。"))
	}
	if multi, ok := metricFloat64(result.Metrics, "multi_core_score"); ok {
		evidence = append(evidence, moduleEvidence("multi_core", "多核评分", fmt.Sprintf("%.2f", multi), evidenceStatus(multi, 75, 60), "多核评分影响编译、批处理和多进程服务容量。"))
	}
	if score, ok := getCPUScore(result); ok {
		evidence = append(evidence, moduleEvidence("score", "CPU 总分", fmt.Sprintf("%.2f / 100", score), evidenceStatus(score, 75, 60), "CPU 总分由单核、多核和后端评分口径综合生成。"))
	}
	if stddev, ok := getCPUScoreStdDev(result); ok {
		status := "success"
		if stddev > 10 {
			status = "warning"
		}
		evidence = append(evidence, moduleEvidence("stability", "采样波动", fmt.Sprintf("%.2f stddev", stddev), status, "多轮采样波动过大时，应在低负载时段复测。"))
	}
	if result.ErrorMessage != "" {
		evidence = append(evidence, moduleEvidence("error", "执行问题", result.ErrorMessage, "failed", "失败或跳过会降低整份报告置信度。"))
	}

	recommendations := []string{}
	limitations := []string{}
	if backend == models.CPUBackendBuiltin {
		limitations = append(limitations, "CPU 使用内置轻量后端，适合快速参考，不适合作为公开基准排名。")
		recommendations = append(recommendations, "如需主流 CPU 基准，使用 --cpu-backend sysbench 或 --cpu-backend geekbench 复测。")
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "保留同一 CPU 后端和档位，用于不同 VPS 横向比较。")
	}
	assessment["status"] = status
	assessment["confidence"] = confidence
	assessment["summary"] = summary
	assessment["evidence"] = evidence
	assessment["limitations"] = limitations
	assessment["recommendations"] = recommendations
	return assessment
}

func (rg *ReportGenerator) buildMemoryModuleAssessment(report *models.Report) map[string]interface{} {
	assessment := moduleAssessmentBase("memory", "内存性能", "skipped", "low", "内存测试未执行。")
	if report == nil || report.TestResults == nil || report.TestResults.MemoryResult == nil {
		assessment["recommendations"] = []string{"运行包含内存的测评档位后再判断内存吞吐。"}
		return assessment
	}

	result := report.TestResults.MemoryResult
	status := moduleStatusFromTestStatus(result.Status)
	backend := fallbackText(getMemoryBackend(result), "-")
	confidence := confidenceFromCoreResult(result, backend != models.MemoryBackendBuiltin)
	summary := fmt.Sprintf("内存后端 %s，状态 %s。", backend, statusText(result.Status))
	evidence := []map[string]interface{}{
		moduleEvidence("backend", "测试后端", backend, backendEvidenceStatus(backend, models.MemoryBackendBuiltin), "sysbench 内存后端更接近主流报告口径；builtin 更适合快速验收。"),
	}
	if read, ok := getMemoryReadSpeed(result); ok {
		evidence = append(evidence, moduleEvidence("read", "读取速度", fmt.Sprintf("%.2f MB/s", read), evidenceStatus(read, 3000, 1000), "读取吞吐影响缓存、扫描和数据处理场景。"))
	}
	if write, ok := getMemoryWriteSpeed(result); ok {
		evidence = append(evidence, moduleEvidence("write", "写入速度", fmt.Sprintf("%.2f MB/s", write), evidenceStatus(write, 2500, 800), "写入吞吐影响缓存更新和内存密集型任务。"))
	}
	if score, ok := metricFloat64(result.Metrics, "score"); ok {
		evidence = append(evidence, moduleEvidence("score", "内存评分", fmt.Sprintf("%.2f / 100", score), evidenceStatus(score, 75, 60), "内存评分由读取、写入和评分基准综合生成。"))
	}
	if readStd, ok := getMemoryReadStdDev(result); ok && readStd > 0 {
		status := "success"
		if readStd > 1000 {
			status = "warning"
		}
		evidence = append(evidence, moduleEvidence("read_stability", "读取波动", fmt.Sprintf("%.2f MB/s stddev", readStd), status, "读取波动过大可能来自同宿主负载或测评期间资源争用。"))
	}
	if writeStd, ok := getMemoryWriteStdDev(result); ok && writeStd > 0 {
		status := "success"
		if writeStd > 1000 {
			status = "warning"
		}
		evidence = append(evidence, moduleEvidence("write_stability", "写入波动", fmt.Sprintf("%.2f MB/s stddev", writeStd), status, "写入波动过大时建议在低负载时段复测。"))
	}
	if result.ErrorMessage != "" {
		evidence = append(evidence, moduleEvidence("error", "执行问题", result.ErrorMessage, "failed", "失败或跳过会降低整份报告置信度。"))
	}

	recommendations := []string{}
	limitations := []string{}
	if backend == models.MemoryBackendBuiltin {
		limitations = append(limitations, "内存使用内置轻量后端，适合快速参考，不适合作为公开基准排名。")
		recommendations = append(recommendations, "如需主流内存基准，使用 --memory-backend sysbench 复测。")
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "保留同一内存后端和档位，用于不同 VPS 横向比较。")
	}
	assessment["status"] = status
	assessment["confidence"] = confidence
	assessment["summary"] = summary
	assessment["evidence"] = evidence
	assessment["limitations"] = limitations
	assessment["recommendations"] = recommendations
	return assessment
}

func (rg *ReportGenerator) buildDiskModuleAssessment(report *models.Report) map[string]interface{} {
	assessment := moduleAssessmentBase("disk", "磁盘性能", "skipped", "low", "磁盘测试未执行。")
	if report == nil || report.TestResults == nil || report.TestResults.DiskResult == nil {
		assessment["recommendations"] = []string{"运行包含磁盘的测评档位后再判断存储能力。"}
		return assessment
	}

	result := report.TestResults.DiskResult
	status := moduleStatusFromTestStatus(result.Status)
	backend := fallbackText(getDiskBackend(result), "-")
	confidence := confidenceFromCoreResult(result, backend == models.DiskBackendFio)
	summary := fmt.Sprintf("磁盘后端 %s，状态 %s。", backend, statusText(result.Status))
	evidence := []map[string]interface{}{
		moduleEvidence("backend", "测试后端", backend, backendEvidenceStatus(backend, models.DiskBackendBuiltin), "fio 更接近主流磁盘基准口径；builtin 更适合快速验收。"),
	}
	if read, ok := getDiskReadSpeed(result); ok {
		evidence = append(evidence, moduleEvidence("sequential_read", "顺序读取", fmt.Sprintf("%.2f MB/s", read), evidenceStatus(read, 1000, 300), "顺序读取影响镜像、备份和大文件读取场景。"))
	}
	if write, ok := getDiskWriteSpeed(result); ok {
		evidence = append(evidence, moduleEvidence("sequential_write", "顺序写入", fmt.Sprintf("%.2f MB/s", write), evidenceStatus(write, 800, 200), "顺序写入影响日志、备份和大文件写入场景。"))
	}
	if iops, ok := getDiskRandomIOPS(result); ok {
		evidence = append(evidence, moduleEvidence("random_iops", "随机 IOPS", fmt.Sprintf("%d", iops), evidenceStatus(float64(iops), 5000, 1000), "随机 IOPS 更影响数据库、小文件和高并发读写。"))
	}
	if rows := getDiskFioMixedRows(result); len(rows) > 0 {
		evidence = append(evidence, moduleEvidence("fio_matrix", "fio 混合矩阵", fmt.Sprintf("%d 组", len(rows)), "success", "混合矩阵可用于观察 4k/64k/1m 等块大小下的读写均衡性。"))
	}
	if score, ok := metricFloat64(result.Metrics, "score"); ok {
		evidence = append(evidence, moduleEvidence("score", "磁盘评分", fmt.Sprintf("%.2f / 100", score), evidenceStatus(score, 75, 60), "磁盘评分由顺序读写、随机 IOPS 和评分基准综合生成。"))
	}
	if result.ErrorMessage != "" {
		evidence = append(evidence, moduleEvidence("error", "执行问题", result.ErrorMessage, "failed", "失败或跳过会降低整份报告置信度。"))
	}

	recommendations := []string{}
	limitations := []string{}
	if backend == models.DiskBackendBuiltin {
		limitations = append(limitations, "磁盘使用内置轻量后端，适合快速参考，不适合作为公开基准排名。")
		recommendations = append(recommendations, "如需主流磁盘基准，使用 --disk-backend fio 复测。")
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "保留同一磁盘后端和档位，用于不同 VPS 横向比较。")
	}
	assessment["status"] = status
	assessment["confidence"] = confidence
	assessment["summary"] = summary
	assessment["evidence"] = evidence
	assessment["limitations"] = limitations
	assessment["recommendations"] = recommendations
	return assessment
}

func (rg *ReportGenerator) buildNetworkModuleAssessment(report *models.Report) map[string]interface{} {
	assessment := moduleAssessmentBase("network", "网络质量", "skipped", "low", "网络性能测试未执行。")
	if report == nil || report.TestResults == nil || report.TestResults.NetworkResult == nil {
		assessment["recommendations"] = []string{"运行包含 network 的测评档位后再判断网络质量。"}
		return assessment
	}

	result := report.TestResults.NetworkResult
	status := moduleStatusFromTestStatus(result.Status)
	confidence := "high"
	if result.Status != models.TestStatusSuccess {
		confidence = "low"
	} else if isNetworkUploadEstimated(result) || getNetworkBackend(result) == models.NetworkBackendBuiltin {
		confidence = "medium"
	}
	backend := fallbackText(getNetworkBackend(result), "-")
	summary := fmt.Sprintf("网络后端 %s，状态 %s。", backend, statusText(result.Status))
	evidence := []map[string]interface{}{
		moduleEvidence("backend", "网络后端", backend, status, fmt.Sprintf("下载来源 %s，上传来源 %s。", fallbackText(getNetworkDownloadSource(result), "-"), fallbackText(getNetworkUploadSource(result), "-"))),
	}
	if latency, ok := getNetworkLatency(result); ok {
		evidence = append(evidence, moduleEvidence("latency", "平均延迟", fmt.Sprintf("%.2f ms", latency), evidenceStatus(100-latency, 70, 40), "延迟来自网络测试后端或 TCP connect 近似测量。"))
	}
	if download, ok := getNetworkDownloadSpeed(result); ok {
		evidence = append(evidence, moduleEvidence("download", "下载速度", fmt.Sprintf("%.2f Mbps", download), evidenceStatus(download, 500, 100), "下载速度用于判断公网入站吞吐参考能力。"))
	}
	if upload, ok := getNetworkUploadSpeed(result); ok {
		uploadStatus := evidenceStatus(upload, 300, 50)
		detail := "上传速度来自真实测量路径。"
		if isNetworkUploadEstimated(result) {
			uploadStatus = "warning"
			detail = "上传速度为估算值，不能等同于真实上传吞吐。"
		}
		evidence = append(evidence, moduleEvidence("upload", "上传速度", fmt.Sprintf("%.2f Mbps", upload), uploadStatus, detail))
	}
	if errMsg := getNetworkError(result); errMsg != "" {
		evidence = append(evidence, moduleEvidence("partial_error", "部分失败", errMsg, "warning", "网络后端返回了部分失败信息。"))
	}

	recommendations := []string{}
	limitations := []string{}
	if isNetworkUploadEstimated(result) {
		limitations = append(limitations, "上传速度为估算值，网络上传敏感业务需要补充真实测速。")
		recommendations = append(recommendations, "配置 iperf3 服务端或 speedtest 后端后复测上传和多节点吞吐。")
	}
	if getNetworkBackend(result) == models.NetworkBackendBuiltin {
		limitations = append(limitations, "内置网络后端更适合快速验收，不适合作为公开带宽排名依据。")
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "保留同一网络档位结果，用于不同 VPS 横向对比。")
	}
	assessment["status"] = status
	assessment["confidence"] = confidence
	assessment["summary"] = summary
	assessment["evidence"] = evidence
	assessment["limitations"] = limitations
	assessment["recommendations"] = recommendations
	return assessment
}

func (rg *ReportGenerator) buildRouteModuleAssessment(report *models.Report) map[string]interface{} {
	assessment := moduleAssessmentBase("route", "路由追踪", "skipped", "low", "路由追踪未执行。")
	if report == nil || report.Summary == nil {
		assessment["recommendations"] = []string{"使用 --route-trace 或 standard/full 自动档位补充路由追踪。"}
		return assessment
	}
	results, ok := report.Summary["route_trace_results"].([]*models.TraceResult)
	if !ok || len(results) == 0 {
		assessment["recommendations"] = []string{"使用 --route-trace 或 standard/full 自动档位补充路由追踪。"}
		return assessment
	}

	successCount := 0
	timeoutHops := 0
	totalHops := 0
	latencyTotal := 0.0
	latencyCount := 0
	realReturnCount := 0
	failedTargets := []string{}
	recommendations := []string{}
	for _, result := range results {
		if result == nil {
			continue
		}
		if result.Success {
			successCount++
			totalHops += result.TotalHops
			timeoutHops += result.TimeoutHops
			if result.AverageLatencyMs > 0 {
				latencyTotal += result.AverageLatencyMs
				latencyCount++
			}
		} else {
			failedTargets = append(failedTargets, result.Target)
		}
		if result.IsRealReturnRoute {
			realReturnCount++
		}
		for _, recommendation := range result.Recommendations {
			if recommendation != "" && !stringSliceContains(recommendations, recommendation) {
				recommendations = append(recommendations, recommendation)
			}
		}
	}

	total := len(results)
	status := availabilityStatus(successCount, total)
	confidence := "medium"
	if successCount == 0 {
		confidence = "low"
	} else if successCount == total && total >= 3 {
		confidence = "high"
	}
	avgHops := 0.0
	if successCount > 0 {
		avgHops = float64(totalHops) / float64(successCount)
	}
	avgLatency := 0.0
	if latencyCount > 0 {
		avgLatency = latencyTotal / float64(latencyCount)
	}

	evidence := []map[string]interface{}{
		moduleEvidence("coverage", "追踪成功率", fmt.Sprintf("%d / %d", successCount, total), status, "成功率反映本机到目标的 traceroute/tracert 可见性。"),
		moduleEvidence("return_route_boundary", "真实回程", yesNoText(realReturnCount > 0), "warning", "当前内置路由追踪表示本机出站路径，不是真实回程。"),
	}
	if avgHops > 0 {
		evidence = append(evidence, moduleEvidence("avg_hops", "平均跳数", fmt.Sprintf("%.1f hops", avgHops), routeEvidenceStatusFromHops(avgHops), "跳数偏多时可能表示跨区域或绕路。"))
	}
	if timeoutHops > 0 {
		evidence = append(evidence, moduleEvidence("timeout_hops", "超时跳", fmt.Sprintf("%d hops", timeoutHops), "warning", "不可见跳点可能来自中间路由屏蔽探测包，不一定代表链路不可用。"))
	}
	if avgLatency > 0 {
		evidence = append(evidence, moduleEvidence("avg_latency", "平均延迟", fmt.Sprintf("%.2f ms", avgLatency), evidenceStatus(100-avgLatency/2, 70, 40), "延迟来自可见跳点均值，只能作为路径参考。"))
	}
	if len(failedTargets) > 0 {
		evidence = append(evidence, moduleEvidence("failed_targets", "失败目标", strings.Join(failedTargets, ", "), "failed", "失败目标可能由依赖缺失、ICMP/UDP 策略或目标网络限制导致。"))
	}
	for _, result := range results {
		if result == nil {
			continue
		}
		appendEvidenceSummaryRows(&evidence, result.Target, result.EvidenceSummary)
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "结合业务目标地区的真实访问延迟和丢包继续验证。")
	}
	limitations := []string{
		"内置 traceroute/tracert 只能表示本机到目标的出站路径。",
		"国内方向参考不等同于真实回程；真实回程需要远端探针或第三方平台配合。",
	}
	assessment["status"] = status
	assessment["confidence"] = confidence
	assessment["summary"] = fmt.Sprintf("追踪 %d 个目标，成功 %d 个；平均跳数 %.1f，超时跳 %d。", total, successCount, avgHops, timeoutHops)
	assessment["evidence"] = evidence
	assessment["limitations"] = limitations
	assessment["recommendations"] = recommendations
	return assessment
}

func (rg *ReportGenerator) buildIPQualityModuleAssessment(report *models.Report) map[string]interface{} {
	assessment := moduleAssessmentBase("ip_quality", "IP 质量", "skipped", "low", "IP 质量检测未执行。")
	if report == nil || report.Summary == nil {
		assessment["recommendations"] = []string{"使用 --ip-quality 或 standard/full 自动档位补充 IP 质量检测。"}
		return assessment
	}
	ipReport, ok := report.Summary["ip_quality_report"].(*models.IPQualityReport)
	if !ok || ipReport == nil {
		assessment["recommendations"] = []string{"使用 --ip-quality 或 standard/full 自动档位补充 IP 质量检测。"}
		return assessment
	}

	status := "success"
	confidence := "medium"
	switch ipReport.RiskLevel {
	case "high":
		status = "failed"
	case "medium":
		status = "warning"
	}
	if len(ipReport.Evidence) >= 3 && ipReport.BlacklistSummary != nil {
		confidence = "high"
	}
	if len(ipReport.Evidence) == 0 {
		confidence = "low"
	}

	evidence := []map[string]interface{}{
		moduleEvidence("risk", "风险评分", fmt.Sprintf("%d / 100", ipReport.RiskScore), status, "风险分来自 IP 类型、DNSBL、邮件连通性和启发式风险来源。"),
		moduleEvidence("identity", "IP 归属", ipQualityTextASNLabel(ipReport), "success", fmt.Sprintf("%s %s，%s。", fallbackText(ipReport.Country, "-"), fallbackText(ipReport.City, "-"), ipQualityTextNodeTypeSummary(ipReport))),
	}
	if ipReport.BlacklistSummary != nil {
		blacklistStatus := "success"
		if ipReport.BlacklistSummary.Listed > 0 {
			blacklistStatus = "warning"
		}
		evidence = append(evidence, moduleEvidence("dnsbl", "DNSBL", fmt.Sprintf("命中 %d / 总计 %d", ipReport.BlacklistSummary.Listed, ipReport.BlacklistSummary.Total), blacklistStatus, "DNSBL 命中会影响邮件和部分风控场景。"))
	}
	if ipReport.MailSummary != nil {
		mailStatus := "success"
		if ipReport.MailSummary.ProviderOpen == 0 {
			mailStatus = "warning"
		}
		evidence = append(evidence, moduleEvidence("mail", "邮件连通", fmt.Sprintf("服务商 %d / %d 可连", ipReport.MailSummary.ProviderOpen, ipReport.MailSummary.Providers), mailStatus, "邮件端口结果用于判断 SMTP 使用环境。"))
	}
	for _, item := range ipReport.Evidence {
		if item == nil {
			continue
		}
		evidence = append(evidence, moduleEvidence("source", item.Name, item.Value, normalizeEvidenceStatus(item.Status), item.Detail))
	}
	for _, item := range ipReport.EvidenceSummary {
		if item == nil {
			continue
		}
		evidence = append(evidence, moduleEvidence(
			item.Category,
			item.Label,
			fmt.Sprintf("%s / 置信度 %s", ipQualityEvidenceSummaryStatusText(item.Status), item.Confidence),
			normalizeEvidenceStatus(item.Status),
			strings.TrimSpace(item.Detail+" "+item.Limitation),
		))
	}

	recommendations := append([]string{}, ipReport.Recommendations...)
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "保留 IP 质量模块 JSON，用于后续和其他节点横向比较。")
	}
	assessment["status"] = status
	assessment["confidence"] = confidence
	assessment["summary"] = ipQualitySummaryText(ipReport)
	assessment["evidence"] = evidence
	assessment["limitations"] = ipReport.Notes
	assessment["recommendations"] = recommendations
	return assessment
}

func (rg *ReportGenerator) buildStreamingModuleAssessment(report *models.Report) map[string]interface{} {
	assessment := moduleAssessmentBase("streaming", "流媒体解锁", "skipped", "low", "流媒体检测未执行。")
	if report == nil || report.Summary == nil {
		assessment["recommendations"] = []string{"使用 --streaming 或 standard/full 自动档位补充流媒体检测。"}
		return assessment
	}
	results, ok := report.Summary["streaming_results"].(map[string]*models.StreamingResult)
	if !ok || len(results) == 0 {
		assessment["recommendations"] = []string{"使用 --streaming 或 standard/full 自动档位补充流媒体检测。"}
		return assessment
	}

	total, available, partial := streamingCounts(results)
	status := availabilityStatus(available, total)
	confidence := "medium"
	if total >= 8 {
		confidence = "high"
	}
	if available == 0 {
		confidence = "medium"
	}
	evidence := []map[string]interface{}{
		moduleEvidence("coverage", "平台覆盖", fmt.Sprintf("%d 个平台", total), "success", "平台数量由流媒体检测档位决定。"),
		moduleEvidence("availability", "可用平台", fmt.Sprintf("%d / %d", available, total), status, "可用数量反映当前 IP 对主流平台的访问能力。"),
	}
	if partial > 0 {
		evidence = append(evidence, moduleEvidence("partial", "部分解锁", fmt.Sprintf("%d 个平台", partial), "warning", "部分解锁通常表示区域或内容库受限。"))
	}
	for _, pair := range sortedStreamingResultsForText(results) {
		if pair.result == nil {
			continue
		}
		appendEvidenceSummaryRows(&evidence, pair.key, pair.result.EvidenceSummary)
	}
	assessment["status"] = status
	assessment["confidence"] = confidence
	assessment["summary"] = fmt.Sprintf("检测 %d 个流媒体平台，可用 %d 个，部分解锁 %d 个。", total, available, partial)
	assessment["evidence"] = evidence
	assessment["limitations"] = []string{"流媒体检测基于平台响应和区域提示，平台策略变化可能导致结果波动。"}
	assessment["recommendations"] = []string{"需要公开分享时，建议使用 full 流媒体档位并保留原始 JSON。"}
	return assessment
}

func (rg *ReportGenerator) buildAIModuleAssessment(report *models.Report) map[string]interface{} {
	assessment := moduleAssessmentBase("ai_services", "AI 服务", "skipped", "low", "AI 服务检测未执行。")
	if report == nil || report.Summary == nil {
		assessment["recommendations"] = []string{"使用 --ai-services 或 standard/full 自动档位补充 AI 服务检测。"}
		return assessment
	}
	results, ok := report.Summary["ai_results"].(map[string]*models.AIServiceResult)
	if !ok || len(results) == 0 {
		assessment["recommendations"] = []string{"使用 --ai-services 或 standard/full 自动档位补充 AI 服务检测。"}
		return assessment
	}

	total, available, restricted := aiCounts(results)
	status := availabilityStatus(available, total)
	confidence := "medium"
	if total >= 4 {
		confidence = "high"
	}
	evidence := []map[string]interface{}{
		moduleEvidence("coverage", "服务覆盖", fmt.Sprintf("%d 个服务", total), "success", "服务数量由 AI 检测列表决定。"),
		moduleEvidence("availability", "可访问服务", fmt.Sprintf("%d / %d", available, total), status, "可访问数量反映当前 IP 对主流 AI 服务的访问能力。"),
	}
	if restricted > 0 {
		evidence = append(evidence, moduleEvidence("restricted", "受限服务", fmt.Sprintf("%d 个服务", restricted), "warning", "受限、需验证或限流并不等同于完全不可用。"))
	}
	for _, pair := range sortedAIResultsForText(results) {
		if pair.result == nil {
			continue
		}
		appendEvidenceSummaryRows(&evidence, pair.key, pair.result.EvidenceSummary)
	}
	assessment["status"] = status
	assessment["confidence"] = confidence
	assessment["summary"] = fmt.Sprintf("检测 %d 个 AI 服务，可访问 %d 个，受限 %d 个。", total, available, restricted)
	assessment["evidence"] = evidence
	assessment["limitations"] = []string{"AI 服务检测基于访问性响应，不能替代账号、风控和长期稳定性验证。"}
	assessment["recommendations"] = []string{"业务依赖 AI 服务时，建议在目标账号和真实请求路径下复测。"}
	return assessment
}

func moduleAssessmentBase(id string, title string, status string, confidence string, summary string) map[string]interface{} {
	return map[string]interface{}{
		"id":              id,
		"title":           title,
		"status":          status,
		"confidence":      confidence,
		"summary":         summary,
		"evidence":        []map[string]interface{}{},
		"limitations":     []string{},
		"recommendations": []string{},
	}
}

func moduleEvidence(category string, label string, value string, status string, detail string) map[string]interface{} {
	return map[string]interface{}{
		"category": category,
		"label":    label,
		"value":    value,
		"status":   status,
		"detail":   detail,
	}
}

func appendEvidenceSummaryRows(evidence *[]map[string]interface{}, prefix string, summaries []*models.EvidenceSummary) {
	if evidence == nil {
		return
	}
	for _, item := range summaries {
		if item == nil {
			continue
		}
		*evidence = append(*evidence, moduleEvidence(
			item.Category,
			fmt.Sprintf("%s / %s", prefix, item.Label),
			fmt.Sprintf("%s / 置信度 %s", evidenceSummaryStatusText(item.Status), item.Confidence),
			normalizeEvidenceStatus(item.Status),
			strings.TrimSpace(item.Detail+" "+item.Limitation),
		))
	}
}

func moduleStatusFromTestStatus(status string) string {
	switch status {
	case models.TestStatusSuccess:
		return "success"
	case models.TestStatusDegraded:
		return "warning"
	case models.TestStatusSkipped:
		return "skipped"
	default:
		return "failed"
	}
}

func normalizeEvidenceStatus(status string) string {
	switch status {
	case "success", "clean", "available", "ok":
		return "success"
	case "partial", "warning", "medium":
		return "warning"
	case "failed", "high", "listed", "blocked":
		return "failed"
	default:
		return "partial"
	}
}

func availabilityStatus(available int, total int) string {
	if total <= 0 {
		return "skipped"
	}
	if available == total {
		return "success"
	}
	if available > 0 {
		return "warning"
	}
	return "failed"
}

func routeEvidenceStatusFromHops(hops float64) string {
	switch {
	case hops <= 12:
		return "success"
	case hops <= 20:
		return "warning"
	default:
		return "failed"
	}
}

func stringSliceContains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func backendEvidenceStatus(backend string, builtinBackend string) string {
	if backend == "" || backend == "-" {
		return "warning"
	}
	if backend == builtinBackend {
		return "warning"
	}
	return "success"
}

func confidenceFromCoreResult(result *models.TestResult, mainstream bool) string {
	if result == nil || result.Status != models.TestStatusSuccess {
		return "low"
	}
	if mainstream {
		return "high"
	}
	return "medium"
}

func streamingCounts(results map[string]*models.StreamingResult) (total int, available int, partial int) {
	for _, result := range results {
		if result == nil {
			continue
		}
		total++
		if result.Available {
			available++
		}
		if result.UnlockType == "partial" || result.UnlockType == "limited" || result.UnlockType == "login_required" {
			partial++
		}
	}
	return total, available, partial
}

func aiCounts(results map[string]*models.AIServiceResult) (total int, available int, restricted int) {
	for _, result := range results {
		if result == nil {
			continue
		}
		total++
		if result.Available {
			available++
		}
		switch result.AccessType {
		case "restricted", "rate_limited", "verification_required", "login_required":
			restricted++
		}
	}
	return total, available, restricted
}

func ipQualitySummaryText(report *models.IPQualityReport) string {
	if report == nil {
		return "IP 质量检测未执行。"
	}
	if report.Verdict != nil && report.Verdict.Summary != "" {
		return report.Verdict.Summary
	}
	return fmt.Sprintf("%s 风险等级 %s，风险分 %d/100。", fallbackText(report.PublicIP, "当前 IP"), fallbackText(report.RiskLevel, "unknown"), report.RiskScore)
}

func (rg *ReportGenerator) buildAssessmentConclusion(report *models.Report, overallScore *models.OverallScore) map[string]interface{} {
	conclusion := map[string]interface{}{
		"headline":        "测评结论不可用",
		"scenario":        "需要完整核心测试后判断适用场景。",
		"suitability":     []string{},
		"evidence":        []map[string]interface{}{},
		"limitations":     []string{},
		"recommendations": []string{},
		"bottlenecks":     []map[string]interface{}{},
	}
	if report == nil || overallScore == nil {
		return conclusion
	}

	confidence := "unknown"
	if value, ok := report.Summary["confidence_level"].(map[string]interface{}); ok {
		if level, ok := value["level"].(string); ok && level != "" {
			confidence = level
		}
	}
	grade := overallScore.Grade
	if value, ok := report.Summary["grade"].(string); ok && value != "" {
		grade = value
	}

	conclusion["headline"] = conclusionHeadline(overallScore.TotalScore, grade, confidence)
	conclusion["scenario"] = scenarioConclusion(overallScore, grade)
	conclusion["suitability"] = suitabilityList(overallScore, report.TestResults, grade)
	conclusion["evidence"] = conclusionEvidence(report, overallScore, confidence, grade)
	conclusion["limitations"] = conclusionLimitations(report)
	conclusion["recommendations"] = conclusionRecommendations(report, overallScore)
	conclusion["bottlenecks"] = scoreBottlenecks(overallScore)
	conclusion["confidence"] = confidence
	conclusion["grade"] = grade
	conclusion["total_score"] = overallScore.TotalScore
	return conclusion
}

func conclusionEvidence(report *models.Report, score *models.OverallScore, confidence string, grade string) []map[string]interface{} {
	items := []map[string]interface{}{}
	if score != nil {
		scoreStatus := evidenceStatus(score.TotalScore, 75, 60)
		scoreDetail := fmt.Sprintf("评分基准汇总 CPU %.2f、内存 %.2f、磁盘 %.2f、网络 %.2f。", score.CPUScore, score.MemoryScore, score.DiskScore, score.NetworkScore)
		if grade == "未完成" {
			scoreStatus = "failed"
			scoreDetail = "核心测试未全部成功，当前总分只能作为已完成分项的局部参考。"
		}
		items = append(items, conclusionEvidenceItem(
			"score",
			"综合评分",
			fmt.Sprintf("%.2f / 100，等级 %s", score.TotalScore, grade),
			scoreStatus,
			scoreDetail,
		))
		if bottlenecks := scoreBottlenecks(score); len(bottlenecks) > 0 {
			weakest := bottlenecks[0]
			items = append(items, conclusionEvidenceItem(
				"bottleneck",
				"主要短板",
				fmt.Sprintf("%s %.2f / 100", summaryValueString(weakest["label"]), weakest["score"].(float64)),
				evidenceStatus(weakest["score"].(float64), 75, 60),
				"短板排序按分项得分从低到高生成，用于判断优先复测或规避的业务场景。",
			))
		}
	}
	if report == nil || report.TestResults == nil {
		items = append(items, conclusionEvidenceItem("coverage", "核心测试覆盖", "0 / 4", "failed", "未获取核心测试结果，结论只能作为排查参考。"))
		return items
	}

	success, failed, skipped, degraded := coreTestCounts(report.TestResults)
	coverageStatus := "success"
	if failed > 0 || skipped > 0 {
		coverageStatus = "failed"
	} else if degraded > 0 {
		coverageStatus = "warning"
	}
	items = append(items, conclusionEvidenceItem(
		"coverage",
		"核心测试覆盖",
		fmt.Sprintf("%d / 4 成功，失败 %d，跳过 %d，降级 %d", success, failed, skipped, degraded),
		coverageStatus,
		"CPU、内存、磁盘、网络四个核心模块决定报告是否可作为完整性能结论。",
	))

	mainstream := countMainstreamBackends(report.TestResults)
	backendStatus := "success"
	if mainstream == 0 {
		backendStatus = "warning"
	} else if mainstream < 3 {
		backendStatus = "partial"
	}
	items = append(items, conclusionEvidenceItem(
		"backend",
		"主流后端覆盖",
		fmt.Sprintf("%d / 4", mainstream),
		backendStatus,
		fmt.Sprintf("CPU=%s，内存=%s，磁盘=%s，网络=%s。", fallbackText(getCPUBackend(report.TestResults.CPUResult), "-"), fallbackText(getMemoryBackend(report.TestResults.MemoryResult), "-"), fallbackText(getDiskBackend(report.TestResults.DiskResult), "-"), fallbackText(getNetworkBackend(report.TestResults.NetworkResult), "-")),
	))

	if report.TestResults.NetworkResult != nil {
		networkStatus := "success"
		networkDetail := "网络上传为真实测量路径。"
		if isNetworkUploadEstimated(report.TestResults.NetworkResult) {
			networkStatus = "warning"
			networkDetail = "网络上传为估算值，上传敏感业务需使用 iperf3 或 speedtest 复测。"
		}
		value := "上传真实测量"
		if isNetworkUploadEstimated(report.TestResults.NetworkResult) {
			value = "上传估算"
		}
		if latency, ok := getNetworkLatency(report.TestResults.NetworkResult); ok {
			value = fmt.Sprintf("%s，延迟 %.2f ms", value, latency)
		}
		items = append(items, conclusionEvidenceItem("network", "网络测量路径", value, networkStatus, networkDetail))
	}

	items = append(items, conclusionEvidenceItem(
		"confidence",
		"报告置信度",
		confidence,
		confidenceEvidenceStatus(confidence),
		"置信度会受核心测试失败、内置轻量后端、估算上传和外部依赖缺失影响。",
	))
	return items
}

func conclusionEvidenceItem(category string, label string, value string, status string, detail string) map[string]interface{} {
	return map[string]interface{}{
		"category": category,
		"label":    label,
		"value":    value,
		"status":   status,
		"detail":   detail,
	}
}

func coreTestCounts(testResults *models.TestResults) (success int, failed int, skipped int, degraded int) {
	if testResults == nil {
		return 0, 4, 0, 0
	}
	for _, result := range []*models.TestResult{testResults.CPUResult, testResults.MemoryResult, testResults.DiskResult, testResults.NetworkResult} {
		if result == nil {
			skipped++
			continue
		}
		switch result.Status {
		case models.TestStatusSuccess:
			success++
		case models.TestStatusSkipped:
			skipped++
		case models.TestStatusDegraded:
			degraded++
		default:
			failed++
		}
	}
	return success, failed, skipped, degraded
}

func evidenceStatus(value float64, goodThreshold float64, watchThreshold float64) string {
	switch {
	case value >= goodThreshold:
		return "success"
	case value >= watchThreshold:
		return "warning"
	default:
		return "failed"
	}
}

func confidenceEvidenceStatus(confidence string) string {
	switch confidence {
	case "high":
		return "success"
	case "medium":
		return "warning"
	default:
		return "failed"
	}
}

func conclusionHeadline(totalScore float64, grade string, confidence string) string {
	if grade == "未完成" {
		return fmt.Sprintf("核心测试未完成，当前结果仅适合作为排查参考，置信度 %s。", confidence)
	}
	switch {
	case totalScore >= 90:
		return fmt.Sprintf("综合性能优秀，适合高负载 VPS 场景，置信度 %s。", confidence)
	case totalScore >= 75:
		return fmt.Sprintf("综合性能良好，适合多数生产和建站场景，置信度 %s。", confidence)
	case totalScore >= 60:
		return fmt.Sprintf("综合性能一般，适合轻量服务和低并发任务，置信度 %s。", confidence)
	default:
		return fmt.Sprintf("综合性能偏弱，建议只承担轻量或备用任务，置信度 %s。", confidence)
	}
}

func scenarioConclusion(score *models.OverallScore, grade string) string {
	if score == nil || grade == "未完成" {
		return "未完成全部核心测试，不建议直接用于采购或迁移决策。"
	}
	weakest := weakestComponent(score)
	switch weakest {
	case "network":
		return "主要短板在网络侧，更适合计算、本地任务或对外带宽不敏感的服务。"
	case "disk":
		return "主要短板在磁盘侧，更适合静态服务、轻量 API 或低写入压力业务。"
	case "memory":
		return "主要短板在内存侧，更适合轻量 Web、代理、监控节点等低内存占用场景。"
	case "cpu":
		return "主要短板在 CPU 侧，更适合转发、静态内容和低计算密度任务。"
	default:
		return "分项表现较均衡，适合常规 Web、API、代理、监控和中轻量数据库场景。"
	}
}

func suitabilityList(score *models.OverallScore, testResults *models.TestResults, grade string) []string {
	if score == nil {
		return []string{}
	}
	if grade == "未完成" {
		return []string{"已完成分项可作为局部参考，不建议直接判定整体适用场景"}
	}
	items := []string{}
	if score.TotalScore >= 75 && score.CPUScore >= 70 && score.MemoryScore >= 60 {
		items = append(items, "常规 Web / API 服务")
	}
	if score.NetworkScore >= 75 {
		items = append(items, "网络转发、下载、代理或边缘节点")
	}
	if score.DiskScore >= 75 && score.MemoryScore >= 70 {
		items = append(items, "中轻量数据库或缓存服务")
	}
	if score.CPUScore >= 80 {
		items = append(items, "编译、压缩、批处理等 CPU 任务")
	}
	if score.TotalScore < 60 {
		items = append(items, "低负载管理面板、备用节点或临时测试")
	}
	if testResults != nil && isNetworkUploadEstimated(testResults.NetworkResult) {
		items = append(items, "网络上传敏感业务需补充真实 iperf3/speedtest 验证")
	}
	return uniqueStrings(items)
}

func conclusionLimitations(report *models.Report) []string {
	if report == nil || report.Summary == nil {
		return []string{"报告缺少摘要信息。"}
	}
	items := []string{}
	if note, ok := report.Summary["performance_note"].(string); ok && note != "" {
		items = append(items, note)
	}
	if notes, ok := report.Summary["quality_notes"].([]string); ok {
		for _, note := range notes {
			items = append(items, note)
		}
	}
	if len(items) == 0 {
		items = append(items, "核心测试均已完成，未发现明显降级或估算路径。")
	}
	return uniqueStrings(items)
}

func conclusionRecommendations(report *models.Report, score *models.OverallScore) []string {
	items := []string{}
	if report != nil && report.TestResults != nil {
		if missing := missingCoreTestNames(report.TestResults); len(missing) > 0 {
			items = append(items, "先补齐未完成核心测试："+strings.Join(missing, "、")+"。")
		}
		if getCPUBackend(report.TestResults.CPUResult) == models.CPUBackendBuiltin {
			items = append(items, "如需公开对比 CPU，请使用 --cpu-backend sysbench 或 --cpu-backend geekbench 复测。")
		}
		if getMemoryBackend(report.TestResults.MemoryResult) == models.MemoryBackendBuiltin {
			items = append(items, "如需更接近主流口径的内存结果，请使用 --memory-backend sysbench。")
		}
		if getDiskBackend(report.TestResults.DiskResult) == models.DiskBackendBuiltin {
			items = append(items, "如需真实磁盘基准，请使用 --disk-backend fio。")
		}
		if isNetworkUploadEstimated(report.TestResults.NetworkResult) {
			items = append(items, "网络上传为估算值，建议配置 iperf3 节点或 speedtest 后端复测。")
		}
	}
	if score != nil {
		switch weakestComponent(score) {
		case "network":
			items = append(items, "网络分数偏低时，优先复测不同地区节点并检查 IPv6、丢包和路由绕行。")
		case "disk":
			items = append(items, "磁盘分数偏低时，避免高写入数据库和日志密集型业务。")
		case "memory":
			items = append(items, "内存分数偏低时，控制服务数量并设置合理 swap。")
		case "cpu":
			items = append(items, "CPU 分数偏低时，避免编译、视频处理和高并发动态计算。")
		}
	}
	if len(items) == 0 {
		items = append(items, "可以保留本次 JSON 报告，用同一档位对其他 VPS 做横向比较。")
	}
	return uniqueStrings(items)
}

func missingCoreTestNames(testResults *models.TestResults) []string {
	if testResults == nil {
		return []string{"CPU", "内存", "磁盘", "网络"}
	}
	items := []string{}
	for _, item := range []struct {
		name   string
		result *models.TestResult
	}{
		{name: "CPU", result: testResults.CPUResult},
		{name: "内存", result: testResults.MemoryResult},
		{name: "磁盘", result: testResults.DiskResult},
		{name: "网络", result: testResults.NetworkResult},
	} {
		if item.result == nil || item.result.Status != models.TestStatusSuccess {
			items = append(items, item.name)
		}
	}
	return items
}

func scoreBottlenecks(score *models.OverallScore) []map[string]interface{} {
	if score == nil {
		return []map[string]interface{}{}
	}
	items := []struct {
		key   string
		label string
		score float64
	}{
		{key: "cpu", label: "CPU", score: score.CPUScore},
		{key: "memory", label: "内存", score: score.MemoryScore},
		{key: "disk", label: "磁盘", score: score.DiskScore},
		{key: "network", label: "网络", score: score.NetworkScore},
	}
	ordered := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		ordered = append(ordered, map[string]interface{}{
			"component": item.key,
			"label":     item.label,
			"score":     item.score,
			"severity":  bottleneckSeverity(item.score),
		})
	}
	for i := 0; i < len(ordered)-1; i++ {
		for j := i + 1; j < len(ordered); j++ {
			if ordered[j]["score"].(float64) < ordered[i]["score"].(float64) {
				ordered[i], ordered[j] = ordered[j], ordered[i]
			}
		}
	}
	return ordered
}

func weakestComponent(score *models.OverallScore) string {
	if score == nil {
		return ""
	}
	weakest := "cpu"
	minScore := score.CPUScore
	for key, value := range map[string]float64{
		"memory":  score.MemoryScore,
		"disk":    score.DiskScore,
		"network": score.NetworkScore,
	} {
		if value < minScore {
			weakest = key
			minScore = value
		}
	}
	if minScore >= 70 {
		return ""
	}
	return weakest
}

func bottleneckSeverity(score float64) string {
	switch {
	case score >= 75:
		return "good"
	case score >= 60:
		return "watch"
	default:
		return "weak"
	}
}

func uniqueStrings(items []string) []string {
	seen := map[string]bool{}
	unique := []string{}
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		unique = append(unique, item)
	}
	return unique
}

func (rg *ReportGenerator) buildQualityNotes(testResults *models.TestResults) []string {
	if testResults == nil {
		return []string{"未获取测试结果，报告置信度不足。"}
	}

	notes := []string{}
	for _, item := range []struct {
		name   string
		result *models.TestResult
	}{
		{name: "CPU", result: testResults.CPUResult},
		{name: "内存", result: testResults.MemoryResult},
		{name: "磁盘", result: testResults.DiskResult},
		{name: "网络", result: testResults.NetworkResult},
	} {
		if item.result == nil {
			notes = append(notes, fmt.Sprintf("%s测试未执行。", item.name))
			continue
		}
		if item.result.Status != models.TestStatusSuccess {
			if item.result.ErrorMessage != "" {
				notes = append(notes, fmt.Sprintf("%s测试%s：%s。", item.name, statusText(item.result.Status), item.result.ErrorMessage))
			} else {
				notes = append(notes, fmt.Sprintf("%s测试%s。", item.name, statusText(item.result.Status)))
			}
		}
	}

	if testResults.CPUResult != nil {
		if getCPUBackend(testResults.CPUResult) == "builtin" {
			notes = append(notes, "CPU 测试使用内置后端，结果适合快速参考；如需主流 CPU 基准建议使用 --cpu-backend sysbench。")
		}
	}
	if testResults.CPUResult != nil && testResults.CPUResult.Status == models.TestStatusSuccess {
		if stddev, ok := getCPUScoreStdDev(testResults.CPUResult); ok && stddev > 10 {
			notes = append(notes, fmt.Sprintf("CPU 多轮采样波动较大（stddev %.2f），建议复测。", stddev))
		}
	}
	if testResults.MemoryResult != nil {
		if getMemoryBackend(testResults.MemoryResult) == "builtin" {
			notes = append(notes, "内存测试使用内置后端，结果适合快速参考；如需主流内存基准建议使用 --memory-backend sysbench。")
		}
	}
	if testResults.MemoryResult != nil && testResults.MemoryResult.Status == models.TestStatusSuccess {
		if readStd, ok := getMemoryReadStdDev(testResults.MemoryResult); ok && readStd > 1000 {
			notes = append(notes, fmt.Sprintf("内存读取波动较大（stddev %.2f MB/s），建议复测。", readStd))
		}
		if writeStd, ok := getMemoryWriteStdDev(testResults.MemoryResult); ok && writeStd > 1000 {
			notes = append(notes, fmt.Sprintf("内存写入波动较大（stddev %.2f MB/s），建议复测。", writeStd))
		}
	}
	if testResults.DiskResult != nil {
		switch getDiskBackend(testResults.DiskResult) {
		case "":
			notes = append(notes, "磁盘测试未标记后端来源。")
		case "builtin":
			notes = append(notes, "磁盘测试使用内置后端，结果适合快速参考；如需主流基准建议使用 --disk-backend fio。")
		}
	}
	if testResults.NetworkResult != nil {
		if isNetworkUploadEstimated(testResults.NetworkResult) {
			notes = append(notes, "网络上传速度为估算值，网络评分上限为 85；如需真实上传建议使用 --network-backend iperf3。")
		}
		if message := getNetworkError(testResults.NetworkResult); message != "" {
			notes = append(notes, "网络测试存在部分失败："+message)
		}
	}

	if len(notes) == 0 {
		notes = append(notes, "核心测试均已完成，未发现明显降级或估算路径。")
	}
	return notes
}

func (rg *ReportGenerator) buildBenchmarkProfile(testResults *models.TestResults) map[string]interface{} {
	profile := map[string]interface{}{
		"name":             "custom",
		"cpu_backend":      resultBackend(testResults, "cpu"),
		"memory_backend":   resultBackend(testResults, "memory"),
		"disk_backend":     resultBackend(testResults, "disk"),
		"network_backend":  resultBackend(testResults, "network"),
		"mainstream_count": countMainstreamBackends(testResults),
	}

	if testResults != nil &&
		getCPUBackend(testResults.CPUResult) == models.CPUBackendBuiltin &&
		getMemoryBackend(testResults.MemoryResult) == models.MemoryBackendBuiltin &&
		getDiskBackend(testResults.DiskResult) == models.DiskBackendBuiltin &&
		testResults.NetworkResult == nil {
		profile["name"] = "quick"
	}
	if testResults != nil &&
		getCPUBackend(testResults.CPUResult) == models.CPUBackendBuiltin &&
		getMemoryBackend(testResults.MemoryResult) == models.MemoryBackendBuiltin &&
		getDiskBackend(testResults.DiskResult) == models.DiskBackendBuiltin &&
		getNetworkBackend(testResults.NetworkResult) == models.NetworkBackendBuiltin {
		profile["name"] = "default"
	}
	if testResults != nil &&
		getCPUBackend(testResults.CPUResult) == models.CPUBackendSysbench &&
		getMemoryBackend(testResults.MemoryResult) == models.MemoryBackendSysbench &&
		getDiskBackend(testResults.DiskResult) == models.DiskBackendFio {
		if testResults.NetworkResult == nil || getNetworkBackend(testResults.NetworkResult) == models.NetworkBackendBuiltin {
			profile["name"] = "full"
		}
		if getNetworkBackend(testResults.NetworkResult) == models.NetworkBackendIperf3 {
			profile["name"] = "full_iperf3"
		}
		if getNetworkBackend(testResults.NetworkResult) == models.NetworkBackendSpeedtest {
			profile["name"] = "full_speedtest"
		}
	}
	return profile
}

func (rg *ReportGenerator) buildConfidenceLevel(testResults *models.TestResults) map[string]interface{} {
	level := "high"
	reasons := []string{}

	if testResults == nil {
		return map[string]interface{}{
			"level":   "low",
			"reasons": []string{"未获取测试结果。"},
		}
	}

	for _, item := range []struct {
		name   string
		result *models.TestResult
	}{
		{name: "CPU", result: testResults.CPUResult},
		{name: "内存", result: testResults.MemoryResult},
		{name: "磁盘", result: testResults.DiskResult},
		{name: "网络", result: testResults.NetworkResult},
	} {
		if item.result == nil {
			reasons = append(reasons, item.name+"测试未执行")
			level = "low"
			continue
		}
		if item.result.Status != models.TestStatusSuccess {
			reasons = append(reasons, item.name+"测试未成功")
			level = "low"
		}
	}

	if getCPUBackend(testResults.CPUResult) == models.CPUBackendBuiltin {
		reasons = append(reasons, "CPU 使用内置后端")
		level = lowerConfidence(level, "medium")
	}
	if getMemoryBackend(testResults.MemoryResult) == models.MemoryBackendBuiltin {
		reasons = append(reasons, "内存使用内置后端")
		level = lowerConfidence(level, "medium")
	}
	if getDiskBackend(testResults.DiskResult) == models.DiskBackendBuiltin {
		reasons = append(reasons, "磁盘使用内置后端")
		level = lowerConfidence(level, "medium")
	}
	if isNetworkUploadEstimated(testResults.NetworkResult) {
		reasons = append(reasons, "网络上传为估算值")
		level = lowerConfidence(level, "medium")
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "核心测试均使用主流或真实测量路径。")
	}
	return map[string]interface{}{
		"level":   level,
		"reasons": reasons,
	}
}

func resultBackend(testResults *models.TestResults, name string) string {
	if testResults == nil {
		return ""
	}
	switch name {
	case "cpu":
		return getCPUBackend(testResults.CPUResult)
	case "memory":
		return getMemoryBackend(testResults.MemoryResult)
	case "disk":
		return getDiskBackend(testResults.DiskResult)
	case "network":
		return getNetworkBackend(testResults.NetworkResult)
	default:
		return ""
	}
}

func countMainstreamBackends(testResults *models.TestResults) int {
	count := 0
	if testResults == nil {
		return count
	}
	if getCPUBackend(testResults.CPUResult) == models.CPUBackendSysbench {
		count++
	}
	if getMemoryBackend(testResults.MemoryResult) == models.MemoryBackendSysbench {
		count++
	}
	if getDiskBackend(testResults.DiskResult) == models.DiskBackendFio {
		count++
	}
	if getNetworkBackend(testResults.NetworkResult) == models.NetworkBackendIperf3 ||
		getNetworkBackend(testResults.NetworkResult) == models.NetworkBackendSpeedtest {
		count++
	}
	return count
}

func lowerConfidence(current string, candidate string) string {
	rank := map[string]int{
		"low":    0,
		"medium": 1,
		"high":   2,
	}
	if rank[candidate] < rank[current] {
		return candidate
	}
	return current
}

func statusText(status string) string {
	switch status {
	case models.TestStatusSuccess:
		return "成功"
	case models.TestStatusFailed:
		return "失败"
	case models.TestStatusSkipped:
		return "跳过"
	case models.TestStatusDegraded:
		return "降级"
	default:
		return status
	}
}

// formatReport 格式化完整报告
func (rg *ReportGenerator) FormatReport(report *models.Report) string {
	var sb strings.Builder

	// 报告标题
	sb.WriteString("╔════════════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║          Perfassess - 评估报告            ║\n")
	sb.WriteString("╚════════════════════════════════════════════════════════════════╝\n\n")

	// 会话信息
	sb.WriteString(fmt.Sprintf("会话ID:         %s\n", report.SessionID))
	sb.WriteString(fmt.Sprintf("报告时间:       %s\n\n", report.Timestamp.Format("2006-01-02 15:04:05")))

	if vpsSummary := rg.FormatVPSBenchmarkSummary(report); vpsSummary != "" {
		sb.WriteString(vpsSummary)
	}
	if conclusion := rg.FormatAssessmentConclusion(report); conclusion != "" {
		sb.WriteString(conclusion)
	}

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
	if profile, ok := report.Summary["benchmark_profile"].(map[string]interface{}); ok {
		sb.WriteString(fmt.Sprintf("评测档位:       %v\n", profile["name"]))
		sb.WriteString(fmt.Sprintf("后端组合:       CPU=%v, 内存=%v, 磁盘=%v, 网络=%v\n",
			profile["cpu_backend"],
			profile["memory_backend"],
			profile["disk_backend"],
			profile["network_backend"],
		))
	}
	if confidence, ok := report.Summary["confidence_level"].(map[string]interface{}); ok {
		sb.WriteString(fmt.Sprintf("置信等级:       %v\n", confidence["level"]))
	}
	if profile, ok := report.Summary["score_profile"].(string); ok && profile != "" {
		sb.WriteString(fmt.Sprintf("评分基准:       %s\n", profile))
	}
	if breakdown, ok := report.Summary["score_breakdown"].(map[string]interface{}); ok {
		sb.WriteString("评分说明:\n")
		for _, key := range []string{"cpu", "memory", "disk", "network"} {
			if item, ok := breakdown[key].(map[string]interface{}); ok {
				sb.WriteString(fmt.Sprintf("  - %s: 分数 %v，权重 %v，%s\n",
					scoreComponentLabel(key),
					formatBreakdownNumber(item["score"]),
					formatBreakdownNumber(item["weight"]),
					item["formula"],
				))
			}
		}
		if normalized, ok := breakdown["normalized_total"].(map[string]interface{}); ok {
			sb.WriteString(fmt.Sprintf("  - 总分: %v，参与权重 %v\n",
				formatBreakdownNumber(normalized["score"]),
				formatBreakdownNumber(normalized["active_weight"]),
			))
		}
	}
	if notes, ok := report.Summary["quality_notes"].([]string); ok && len(notes) > 0 {
		sb.WriteString("质量提示:\n")
		for _, note := range notes {
			sb.WriteString(fmt.Sprintf("  - %s\n", note))
		}
	}

	sb.WriteString("\n")
	if moduleText := rg.FormatModuleAssessments(report); moduleText != "" {
		sb.WriteString(moduleText)
	}

	if share := rg.FormatSharePlainText(report); share != "" {
		sb.WriteString("=== 分享模板 ===\n\n")
		sb.WriteString(share)
		sb.WriteString("\n\n")
	}

	// 测试统计
	if report.Summary != nil {
		sb.WriteString("=== 测试统计 ===\n\n")
		sb.WriteString(fmt.Sprintf("成功测试:       %d\n", report.Summary["tests_success"]))
		sb.WriteString(fmt.Sprintf("失败测试:       %d\n", report.Summary["tests_failed"]))
		sb.WriteString(fmt.Sprintf("跳过测试:       %d\n", report.Summary["tests_skipped"]))
		if degraded, ok := report.Summary["tests_degraded"]; ok {
			sb.WriteString(fmt.Sprintf("降级测试:       %d\n", degraded))
		}

		// 路由追踪结果
		if routeResults, ok := report.Summary["route_trace_results"].([]*models.TraceResult); ok && len(routeResults) > 0 {
			sb.WriteString("\n=== 路由追踪结果 ===\n\n")
			if note, ok := report.Summary["route_trace_note"].(string); ok && note != "" {
				sb.WriteString(fmt.Sprintf("说明: %s\n\n", note))
			}
			for _, result := range routeResults {
				sb.WriteString(fmt.Sprintf("目标: %s\n", result.Target))
				sb.WriteString(fmt.Sprintf("方向: %s\n", routeDirectionReportLabel(result)))
				sb.WriteString(fmt.Sprintf("评级: %s\n", routeQualityReportLabel(result)))
				if result.Success {
					sb.WriteString(fmt.Sprintf("总跳数: %d\n", result.TotalHops))
					sb.WriteString(fmt.Sprintf("超时跳: %d\n", result.TimeoutHops))
					if result.AverageLatencyMs > 0 {
						sb.WriteString(fmt.Sprintf("平均延迟: %.2f ms\n", result.AverageLatencyMs))
					}
					if result.LastVisibleHop != "" {
						sb.WriteString(fmt.Sprintf("最后可见跳: %s\n", result.LastVisibleHop))
					}
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
				if len(result.EvidenceSummary) > 0 {
					sb.WriteString("证据可信度:\n")
					for _, item := range result.EvidenceSummary {
						if item == nil {
							continue
						}
						sb.WriteString(fmt.Sprintf("  - %s / %s / 置信度 %s；%s\n",
							item.Label,
							evidenceSummaryStatusText(item.Status),
							item.Confidence,
							item.Limitation,
						))
					}
				}
				if len(result.Recommendations) > 0 {
					sb.WriteString("建议:\n")
					for _, recommendation := range result.Recommendations {
						sb.WriteString(fmt.Sprintf("  - %s\n", recommendation))
					}
				}
				sb.WriteString("\n")
			}
		}

		// 流媒体检测结果
		if streamingResults, ok := report.Summary["streaming_results"].(map[string]*models.StreamingResult); ok && len(streamingResults) > 0 {
			sb.WriteString("=== 流媒体解锁检测 ===\n\n")
			for _, pair := range sortedStreamingResultsForText(streamingResults) {
				result := pair.result
				status := "❌ 不可用"
				if result != nil && result.Available {
					status = "✅ 可用"
					if result.Region != "" && result.Region != "Unknown" {
						status += fmt.Sprintf(" (%s)", result.Region)
					}
				}
				sb.WriteString(fmt.Sprintf("%-15s  %-8s  %-10s  %-14s  %s\n",
					pair.key,
					streamingTextCategory(result),
					status,
					streamingTextUnlockType(result),
					streamingTextMessage(result),
				))
				for _, item := range result.EvidenceSummary {
					if item == nil {
						continue
					}
					sb.WriteString(fmt.Sprintf("    证据: %s / %s / 置信度 %s；%s\n",
						item.Label,
						evidenceSummaryStatusText(item.Status),
						item.Confidence,
						item.Limitation,
					))
				}
			}
			sb.WriteString("\n")
		}

		// AI 服务检测结果
		if aiResults, ok := report.Summary["ai_results"].(map[string]*models.AIServiceResult); ok && len(aiResults) > 0 {
			sb.WriteString("=== AI服务检测 ===\n\n")
			for _, pair := range sortedAIResultsForText(aiResults) {
				result := pair.result
				status := "❌ 不可用"
				if result != nil && result.Available {
					status = "✅ 可用"
				}
				sb.WriteString(fmt.Sprintf("%-15s  %-8s  %-10s  %-14s  %s\n",
					pair.key,
					aiTextCategory(result),
					status,
					aiTextAccessType(result),
					aiTextMessage(result),
				))
				for _, item := range result.EvidenceSummary {
					if item == nil {
						continue
					}
					sb.WriteString(fmt.Sprintf("    证据: %s / %s / 置信度 %s；%s\n",
						item.Label,
						evidenceSummaryStatusText(item.Status),
						item.Confidence,
						item.Limitation,
					))
				}
			}
			sb.WriteString("\n")
		}

		if ipQualityReport, ok := report.Summary["ip_quality_report"].(*models.IPQualityReport); ok && ipQualityReport != nil {
			sb.WriteString("=== IP 节点分析报告 ===\n\n")
			sb.WriteString(fmt.Sprintf("IP 地址:        %s (%s)\n", fallbackText(ipQualityReport.PublicIP, "-"), fallbackText(ipQualityReport.IPVersion, "-")))
			location := strings.Trim(strings.Join([]string{ipQualityReport.Country, ipQualityReport.City}, " "), " ")
			if location != "" {
				sb.WriteString(fmt.Sprintf("国家/地区:      %s\n", location))
			}
			sb.WriteString(fmt.Sprintf("运营商/ASN:     %s\n", ipQualityTextASNLabel(ipQualityReport)))
			if len(ipQualityReport.ReverseDNS) > 0 {
				sb.WriteString(fmt.Sprintf("反向 DNS:       %s\n", strings.Join(ipQualityReport.ReverseDNS, ", ")))
			}
			sb.WriteString("\n")
			sb.WriteString(fmt.Sprintf("IP 类型:        %s\n", ipQualityTextNodeTypeSummary(ipQualityReport)))
			sb.WriteString(fmt.Sprintf("代理/VPN 标记:  %s\n", yesNoText(ipQualityTextRiskFactorDetected(ipQualityReport, "proxy") || ipQualityTextRiskFactorDetected(ipQualityReport, "vpn"))))
			sb.WriteString(fmt.Sprintf("机房/托管标记:  %s\n\n", yesNoText(ipQualityTextRiskFactorDetected(ipQualityReport, "datacenter"))))
			sb.WriteString(fmt.Sprintf("欺诈风险分:     %s %d/100\n", ipQualityRiskBar(ipQualityReport.RiskScore), ipQualityReport.RiskScore))
			sb.WriteString(fmt.Sprintf("风险等级:       %s\n", ipRiskLabel(ipQualityReport.RiskLevel)))
			if ipQualityReport.BlacklistSummary != nil {
				summary := ipQualityReport.BlacklistSummary
				sb.WriteString(fmt.Sprintf("DNSBL 检查:     命中 %d / 正常 %d / 超时 %d / 跳过 %d / 总计 %d\n",
					summary.Listed, summary.Clean, summary.Timeout, summary.Skipped, summary.Total))
			}
			sb.WriteString(fmt.Sprintf("邮件端口可连:   %d/%d\n", countReachableMailChecks(ipQualityReport.MailChecks), len(ipQualityReport.MailChecks)))
			sb.WriteString(fmt.Sprintf("综合评级:       [ %s ] %s\n", ipQualityTextCompositeGrade(ipQualityReport), ipQualityTextGradeDescription(ipQualityReport)))
			sb.WriteString(fmt.Sprintf("评级依据:       %s\n\n", ipQualityTextBasis(ipQualityReport)))

			if ipQualityReport.NetworkStack != nil {
				sb.WriteString(fmt.Sprintf("网络栈:         %s\n", ipQualityTextNetworkStackLabel(ipQualityReport.NetworkStack)))
				if ipQualityReport.NetworkStack.Note != "" {
					sb.WriteString(fmt.Sprintf("网络栈说明:     %s\n", ipQualityReport.NetworkStack.Note))
				}
				sb.WriteString("\n")
			}
			if len(ipQualityReport.RiskSources) > 0 {
				sb.WriteString("风险来源:\n")
				for _, source := range ipQualityReport.RiskSources {
					if source == nil {
						continue
					}
					sb.WriteString(fmt.Sprintf("  - %-22s %-10s %s", source.Name, ipRiskSourceStatusLabel(source), source.Detail))
					if source.Signal != "" {
						sb.WriteString(fmt.Sprintf(" - %s", source.Signal))
					}
					sb.WriteString("\n")
				}
				sb.WriteString("\n")
			}
			if len(ipQualityReport.EvidenceSummary) > 0 {
				sb.WriteString("证据可信度:\n")
				for _, item := range ipQualityReport.EvidenceSummary {
					if item == nil {
						continue
					}
					sb.WriteString(fmt.Sprintf("  - %-12s %-6s 置信度 %-6s %s\n",
						item.Label,
						ipQualityEvidenceSummaryStatusText(item.Status),
						item.Confidence,
						item.Detail,
					))
					if item.Limitation != "" {
						sb.WriteString(fmt.Sprintf("    限制: %s\n", item.Limitation))
					}
				}
				sb.WriteString("\n")
			}
			if len(ipQualityReport.RiskFactors) > 0 {
				sb.WriteString("IP 类型与风险因子:\n")
				for _, factor := range ipQualityReport.RiskFactors {
					sb.WriteString(fmt.Sprintf("  - %-10s %s", riskFactorLabel(factor), riskFactorStatusLabel(factor)))
					if factor != nil && factor.Detail != "" {
						sb.WriteString(fmt.Sprintf(" - %s", factor.Detail))
					}
					sb.WriteString("\n")
				}
				sb.WriteString("\n")
			}
			if len(ipQualityReport.BlacklistChecks) > 0 {
				sb.WriteString("欺诈与黑名单检查:\n")
				for _, check := range ipQualityReport.BlacklistChecks {
					sb.WriteString(fmt.Sprintf("  - %-24s %s", check.Zone, ipBlacklistStatusLabel(check)))
					if check.Detail != "" {
						sb.WriteString(fmt.Sprintf(" - %s", check.Detail))
					}
					sb.WriteString("\n")
				}
				sb.WriteString("\n")
			}
			if len(ipQualityReport.MailChecks) > 0 {
				sb.WriteString("邮件端口连通性:\n")
				if ipQualityReport.MailSummary != nil {
					summary := ipQualityReport.MailSummary
					sb.WriteString(fmt.Sprintf("  汇总: 服务商 %d 个 / 可连服务商 %d 个 / 可连端口 %d / 阻断 %d / 超时 %d\n",
						summary.Providers, summary.ProviderOpen, summary.Reachable, summary.Blocked, summary.Timeout))
				}
				for _, check := range ipQualityReport.MailChecks {
					sb.WriteString(fmt.Sprintf("  - %-10s %-18s %-5d %s", fallbackText(check.Provider, "-"), check.Target, check.Port, mailCheckStatusLabel(check)))
					if check.Detail != "" {
						sb.WriteString(fmt.Sprintf(" - %s", check.Detail))
					}
					sb.WriteString("\n")
				}
				sb.WriteString("\n")
			}
			if len(ipQualityReport.Notes) > 0 {
				sb.WriteString("说明:\n")
				for _, note := range ipQualityReport.Notes {
					sb.WriteString(fmt.Sprintf("  - %s\n", note))
				}
				sb.WriteString("\n")
			}
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

func (rg *ReportGenerator) FormatAssessmentConclusion(report *models.Report) string {
	if report == nil || report.Summary == nil {
		return ""
	}
	conclusion, ok := report.Summary["assessment_conclusion"].(map[string]interface{})
	if !ok || len(conclusion) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("=== 测评结论 ===\n\n")
	sb.WriteString(fmt.Sprintf("结论:           %s\n", summaryValueString(conclusion["headline"])))
	sb.WriteString(fmt.Sprintf("适用判断:       %s\n", summaryValueString(conclusion["scenario"])))
	if items := summaryStringSlice(conclusion["suitability"]); len(items) > 0 {
		sb.WriteString("适合场景:\n")
		for _, item := range items {
			sb.WriteString(fmt.Sprintf("  - %s\n", item))
		}
	}
	if evidence := summaryEvidence(conclusion["evidence"]); len(evidence) > 0 {
		sb.WriteString("关键证据:\n")
		for _, item := range evidence {
			sb.WriteString(fmt.Sprintf("  - [%s] %s: %s；%s\n", evidenceStatusText(item.Status), item.Label, item.Value, item.Detail))
		}
	}
	if bottlenecks := summaryBottlenecks(conclusion["bottlenecks"]); len(bottlenecks) > 0 {
		sb.WriteString("短板排序:\n")
		for _, item := range bottlenecks {
			sb.WriteString(fmt.Sprintf("  - %s: %.2f / 100，%s\n", item.Label, item.Score, bottleneckSeverityText(item.Severity)))
		}
	}
	if items := summaryStringSlice(conclusion["recommendations"]); len(items) > 0 {
		sb.WriteString("建议:\n")
		for _, item := range items {
			sb.WriteString(fmt.Sprintf("  - %s\n", item))
		}
	}
	sb.WriteString("\n")
	return sb.String()
}

func (rg *ReportGenerator) FormatModuleAssessments(report *models.Report) string {
	if report == nil || report.Summary == nil {
		return ""
	}
	modules, ok := report.Summary["module_assessments"].(map[string]interface{})
	if !ok || len(modules) == 0 {
		return ""
	}

	var sb strings.Builder
	wroteHeader := false
	for _, key := range []string{"cpu", "memory", "disk", "network", "route", "ip_quality", "streaming", "ai_services"} {
		module, ok := modules[key].(map[string]interface{})
		if !ok || module["status"] == "skipped" {
			continue
		}
		if !wroteHeader {
			sb.WriteString("=== 模块可信度 ===\n\n")
			wroteHeader = true
		}
		sb.WriteString(fmt.Sprintf("%s: %s | 置信度 %s\n", summaryValueString(module["title"]), moduleStatusText(summaryValueString(module["status"])), summaryValueString(module["confidence"])))
		sb.WriteString(fmt.Sprintf("  结论: %s\n", summaryValueString(module["summary"])))
		if evidence := summaryEvidence(module["evidence"]); len(evidence) > 0 {
			for _, item := range evidence[:minInt(len(evidence), 3)] {
				sb.WriteString(fmt.Sprintf("  - [%s] %s: %s\n", evidenceStatusText(item.Status), item.Label, item.Value))
			}
		}
		if recommendations := summaryStringSlice(module["recommendations"]); len(recommendations) > 0 {
			sb.WriteString(fmt.Sprintf("  建议: %s\n", recommendations[0]))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

type conclusionBottleneck struct {
	Label    string
	Score    float64
	Severity string
}

type conclusionEvidenceRow struct {
	Label  string
	Value  string
	Status string
	Detail string
}

func summaryValueString(value interface{}) string {
	if value == nil {
		return "-"
	}
	if text, ok := value.(string); ok && text != "" {
		return text
	}
	return fmt.Sprintf("%v", value)
}

func summaryStringSlice(value interface{}) []string {
	switch items := value.(type) {
	case []string:
		return items
	case []interface{}:
		out := make([]string, 0, len(items))
		for _, item := range items {
			if text := summaryValueString(item); text != "-" {
				out = append(out, text)
			}
		}
		return out
	default:
		return []string{}
	}
}

func summaryBottlenecks(value interface{}) []conclusionBottleneck {
	items, ok := value.([]map[string]interface{})
	if !ok {
		if generic, ok := value.([]interface{}); ok {
			items = make([]map[string]interface{}, 0, len(generic))
			for _, item := range generic {
				if mapped, ok := item.(map[string]interface{}); ok {
					items = append(items, mapped)
				}
			}
		}
	}
	out := make([]conclusionBottleneck, 0, len(items))
	for _, item := range items {
		score, _ := metricFloat64(item, "score")
		out = append(out, conclusionBottleneck{
			Label:    summaryValueString(item["label"]),
			Score:    score,
			Severity: summaryValueString(item["severity"]),
		})
	}
	return out
}

func summaryEvidence(value interface{}) []conclusionEvidenceRow {
	items, ok := value.([]map[string]interface{})
	if !ok {
		interfaceItems, ok := value.([]interface{})
		if !ok {
			return nil
		}
		items = make([]map[string]interface{}, 0, len(interfaceItems))
		for _, item := range interfaceItems {
			if itemMap, ok := item.(map[string]interface{}); ok {
				items = append(items, itemMap)
			}
		}
	}

	out := make([]conclusionEvidenceRow, 0, len(items))
	for _, item := range items {
		out = append(out, conclusionEvidenceRow{
			Label:  summaryValueString(item["label"]),
			Value:  summaryValueString(item["value"]),
			Status: summaryValueString(item["status"]),
			Detail: summaryValueString(item["detail"]),
		})
	}
	return out
}

func evidenceStatusText(status string) string {
	switch status {
	case "success":
		return "通过"
	case "partial":
		return "部分"
	case "warning":
		return "注意"
	case "failed":
		return "风险"
	default:
		return "未知"
	}
}

func moduleStatusText(status string) string {
	switch status {
	case "success":
		return "通过"
	case "warning":
		return "注意"
	case "failed":
		return "风险"
	case "skipped":
		return "未执行"
	default:
		return status
	}
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func bottleneckSeverityText(severity string) string {
	switch severity {
	case "good":
		return "表现正常"
	case "watch":
		return "需要关注"
	case "weak":
		return "明显短板"
	default:
		return severity
	}
}

func ipRiskSourceStatusLabel(source *models.IPRiskSource) string {
	if source == nil {
		return "未知"
	}
	switch source.Status {
	case "available":
		return "可用"
	case "missing":
		return "缺失"
	case "clean":
		return "正常"
	case "listed":
		return "命中"
	case "partial":
		return "部分"
	case "disabled":
		return "未启用"
	default:
		return source.Status
	}
}

func evidenceSummaryStatusText(status string) string {
	switch status {
	case "success":
		return "可用"
	case "partial":
		return "部分"
	case "warning":
		return "注意"
	case "skipped":
		return "未执行"
	default:
		return status
	}
}

func ipQualityEvidenceSummaryStatusText(status string) string {
	return evidenceSummaryStatusText(status)
}

type streamingTextResultPair struct {
	key    string
	result *models.StreamingResult
}

func sortedStreamingResultsForText(results map[string]*models.StreamingResult) []streamingTextResultPair {
	pairs := make([]streamingTextResultPair, 0, len(results))
	for key, result := range results {
		pairs = append(pairs, streamingTextResultPair{key: key, result: result})
	}
	sort.Slice(pairs, func(i, j int) bool {
		leftCategory := streamingTextCategory(pairs[i].result)
		rightCategory := streamingTextCategory(pairs[j].result)
		if leftCategory != rightCategory {
			return leftCategory < rightCategory
		}
		return strings.ToLower(pairs[i].key) < strings.ToLower(pairs[j].key)
	})
	return pairs
}

func streamingTextCategory(result *models.StreamingResult) string {
	if result == nil {
		return "-"
	}
	switch result.Category {
	case "global":
		return "全球"
	case "us":
		return "美国"
	case "jp":
		return "日本"
	case "cn":
		return "中国"
	case "hk":
		return "香港"
	case "kr":
		return "韩国"
	case "eu":
		return "欧洲"
	case "asia":
		return "亚洲"
	case "music":
		return "音乐"
	case "sports":
		return "体育"
	default:
		return fallbackText(result.Category, "-")
	}
}

func streamingTextUnlockType(result *models.StreamingResult) string {
	if result == nil {
		return "-"
	}
	switch result.UnlockType {
	case "full":
		return "完整解锁"
	case "partial":
		return "部分解锁"
	case "limited":
		return "受限"
	case "blocked":
		return "不可用"
	case "login_required":
		return "需要登录"
	case "available":
		return "可访问"
	default:
		return fallbackText(result.UnlockType, "-")
	}
}

func streamingTextMessage(result *models.StreamingResult) string {
	if result == nil {
		return "-"
	}
	if result.RegionSource != "" && result.RegionSource != "unknown" {
		return fmt.Sprintf("%s；区域来源: %s", fallbackText(result.Message, "-"), streamingTextRegionSource(result.RegionSource))
	}
	return fallbackText(result.Message, "-")
}

func streamingTextRegionSource(source string) string {
	switch source {
	case "response":
		return "响应"
	case "platform_hint":
		return "平台提示"
	default:
		return source
	}
}

type aiTextResultPair struct {
	key    string
	result *models.AIServiceResult
}

func sortedAIResultsForText(results map[string]*models.AIServiceResult) []aiTextResultPair {
	pairs := make([]aiTextResultPair, 0, len(results))
	for key, result := range results {
		pairs = append(pairs, aiTextResultPair{key: key, result: result})
	}
	sort.Slice(pairs, func(i, j int) bool {
		leftCategory := aiTextCategory(pairs[i].result)
		rightCategory := aiTextCategory(pairs[j].result)
		if leftCategory != rightCategory {
			return leftCategory < rightCategory
		}
		return strings.ToLower(pairs[i].key) < strings.ToLower(pairs[j].key)
	})
	return pairs
}

func aiTextCategory(result *models.AIServiceResult) string {
	if result == nil {
		return "-"
	}
	switch result.Category {
	case "chatbot":
		return "对话"
	case "assistant":
		return "助手"
	case "search":
		return "搜索"
	case "coding":
		return "编程"
	default:
		return fallbackText(result.Category, "-")
	}
}

func aiTextAccessType(result *models.AIServiceResult) string {
	if result == nil {
		return "-"
	}
	switch result.AccessType {
	case "full":
		return "可访问"
	case "login_required":
		return "需要登录"
	case "verification_required":
		return "需验证"
	case "rate_limited":
		return "限流"
	case "restricted":
		return "受限"
	case "available":
		return "可用"
	default:
		return fallbackText(result.AccessType, "-")
	}
}

func aiTextMessage(result *models.AIServiceResult) string {
	if result == nil {
		return "-"
	}
	if result.RegionHint != "" {
		return fmt.Sprintf("%s；区域提示: %s", fallbackText(result.Message, "-"), result.RegionHint)
	}
	return fallbackText(result.Message, "-")
}

func fallbackText(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func ipRiskLabel(level string) string {
	switch level {
	case "low":
		return "低风险"
	case "medium":
		return "中风险"
	case "high":
		return "高风险"
	default:
		return "未知"
	}
}

func ipQualityLabel(value string) string {
	switch value {
	case "datacenter_likely":
		return "疑似机房 / VPS"
	case "residential_or_isp_likely":
		return "疑似住宅或运营商网络"
	default:
		return "未知"
	}
}

func ipBlacklistStatusLabel(check *models.IPBlacklistCheck) string {
	if check == nil {
		return "未知"
	}
	switch check.Status {
	case "listed":
		return "命中"
	case "clean":
		return "未命中"
	case "timeout":
		return "超时"
	case "skipped":
		return "跳过"
	default:
		return check.Status
	}
}

func mailCheckStatusLabel(check *models.MailPortCheck) string {
	if check == nil {
		return "未知"
	}
	switch check.Status {
	case "reachable":
		return "可连接"
	case "blocked":
		return "不可连接"
	case "timeout":
		return "超时"
	default:
		return check.Status
	}
}

func yesNoText(value bool) string {
	if value {
		return "是"
	}
	return "否"
}

func ipQualityTextRiskFactorDetected(report *models.IPQualityReport, name string) bool {
	if report == nil {
		return false
	}
	for _, factor := range report.RiskFactors {
		if factor != nil && factor.Name == name && factor.Detected {
			return true
		}
	}
	return false
}

func ipQualityTextASNLabel(report *models.IPQualityReport) string {
	if report == nil {
		return "-"
	}
	parts := []string{}
	if report.ISP != "" {
		parts = append(parts, report.ISP)
	}
	if report.Organization != "" {
		parts = append(parts, report.Organization)
	}
	if report.ASN != "" {
		parts = append(parts, "AS"+report.ASN)
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, " / ")
}

func ipQualityTextNodeTypeSummary(report *models.IPQualityReport) string {
	if report == nil {
		return "未知"
	}
	switch report.IPType {
	case "datacenter_likely":
		return "机房 IP (Datacenter/IDC)"
	case "residential_or_isp_likely":
		return "住宅或运营商 IP"
	default:
		return "未知"
	}
}

func ipQualityTextCompositeGrade(report *models.IPQualityReport) string {
	if report == nil {
		return "N/A"
	}
	listed := 0
	for _, check := range report.BlacklistChecks {
		if check != nil && check.Listed {
			listed++
		}
	}
	switch {
	case report.RiskLevel == "low" && listed == 0:
		return "A"
	case report.RiskLevel == "low":
		return "B"
	case report.RiskLevel == "medium":
		return "C"
	default:
		return "D"
	}
}

func ipQualityTextGradeDescription(report *models.IPQualityReport) string {
	switch ipQualityTextCompositeGrade(report) {
	case "A":
		return "低风险，适合大多数用途"
	case "B":
		return "中等，部分平台可能敏感"
	case "C":
		return "偏高风险，建议谨慎使用"
	case "D":
		return "高风险，可能影响解锁或投递"
	default:
		return "未知"
	}
}

func ipQualityTextBasis(report *models.IPQualityReport) string {
	if report == nil {
		return "-"
	}
	parts := []string{ipQualityLabel(report.IPType), fmt.Sprintf("欺诈风险分 %d/100", report.RiskScore)}
	if report.BlacklistSummary != nil {
		parts = append(parts, fmt.Sprintf("DNSBL 命中 %d/%d", report.BlacklistSummary.Listed, report.BlacklistSummary.Total))
	}
	if len(report.MailChecks) > 0 {
		if report.MailSummary != nil {
			parts = append(parts, fmt.Sprintf("邮件端口可连 %d/%d，服务商 %d/%d", report.MailSummary.Reachable, report.MailSummary.Total, report.MailSummary.ProviderOpen, report.MailSummary.Providers))
		} else {
			parts = append(parts, fmt.Sprintf("邮件端口可连 %d/%d", countReachableMailChecks(report.MailChecks), len(report.MailChecks)))
		}
	}
	return strings.Join(parts, " + ")
}

func ipQualityTextNetworkStackLabel(stack *models.IPNetworkStack) string {
	if stack == nil {
		return "-"
	}
	if stack.DualStack {
		return "IPv4/IPv6 双栈"
	}
	return fallbackText(stack.DetectedVersion, "-")
}

func ipQualityRiskBar(score int) string {
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	filled := score / 10
	if score > 0 && filled == 0 {
		filled = 1
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", 10-filled) + "]"
}

func riskFactorLabel(factor *models.IPRiskFactor) string {
	if factor == nil {
		return "未知"
	}
	switch factor.Name {
	case "proxy":
		return "代理"
	case "vpn":
		return "VPN"
	case "tor":
		return "Tor"
	case "datacenter":
		return "机房/托管"
	case "abuse":
		return "滥用"
	default:
		return fallbackText(factor.Name, "未知")
	}
}

func riskFactorStatusLabel(factor *models.IPRiskFactor) string {
	if factor == nil {
		return "未知"
	}
	if factor.Detected {
		return "命中"
	}
	return "未发现"
}

func scoreComponentLabel(key string) string {
	switch key {
	case "cpu":
		return "CPU"
	case "memory":
		return "内存"
	case "disk":
		return "磁盘"
	case "network":
		return "网络"
	default:
		return key
	}
}

func routeDirectionReportLabel(result *models.TraceResult) string {
	if result == nil {
		return "-"
	}
	switch result.DirectionGroup {
	case "china_reference":
		return "国内方向参考"
	case "public":
		return "公共方向"
	}
	target := strings.ToLower(result.Target)
	for _, marker := range []string{"189.cn", "10086.cn", "chinaunicom", "ctyun", "qq.com"} {
		if strings.Contains(target, marker) {
			return "国内方向参考"
		}
	}
	return "公共方向"
}

func routeQualityReportLabel(result *models.TraceResult) string {
	if result == nil {
		return "-"
	}
	if result.Quality != nil {
		if result.Quality.Summary != "" {
			return result.Quality.Summary
		}
		if result.Quality.Grade != "" {
			return result.Quality.Grade
		}
	}
	if result.Success {
		return "完成"
	}
	return "失败"
}

func formatBreakdownNumber(value interface{}) string {
	switch v := value.(type) {
	case float64:
		return fmt.Sprintf("%.2f", v)
	case float32:
		return fmt.Sprintf("%.2f", v)
	case int:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", value)
	}
}
