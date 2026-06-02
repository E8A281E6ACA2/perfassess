// Package reporter 提供报告生成功能
package reporter

import (
	"fmt"
	"strings"
	"time"

	"performance-assessment-system/internal/config"
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
	report.Summary["quality_notes"] = rg.buildQualityNotes(report.TestResults)
	report.Summary["benchmark_profile"] = rg.buildBenchmarkProfile(report.TestResults)
	report.Summary["confidence_level"] = rg.buildConfidenceLevel(report.TestResults)
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
