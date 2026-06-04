package reporter

import (
	"fmt"
	"strings"

	"performance-assessment-system/internal/models"
)

func (rg *ReportGenerator) buildVPSBenchmarkSummary(report *models.Report, overallScore *models.OverallScore) map[string]interface{} {
	summary := map[string]interface{}{
		"system":     buildVPSSystemSummary(report.SystemInfo),
		"cpu":        buildVPSCPUSummary(report.TestResults.CPUResult),
		"memory":     buildVPSMemorySummary(report.TestResults.MemoryResult),
		"disk":       buildVPSDiskSummary(report.TestResults.DiskResult),
		"network":    buildVPSNetworkSummary(report.TestResults.NetworkResult),
		"scores":     buildVPSScoresSummary(report.Summary, overallScore),
		"confidence": buildVPSConfidenceSummary(report.Summary),
	}

	if profile, ok := report.Summary["benchmark_profile"].(map[string]interface{}); ok {
		summary["benchmark_profile"] = profile["name"]
		summary["mainstream_backend_count"] = profile["mainstream_count"]
	}
	return summary
}

func buildVPSSystemSummary(systemInfo *models.SystemInfo) map[string]interface{} {
	summary := map[string]interface{}{}
	if systemInfo == nil {
		return summary
	}
	if systemInfo.CPU != nil {
		summary["cpu_model"] = systemInfo.CPU.Model
		summary["cpu_cores"] = systemInfo.CPU.Cores
		summary["cpu_threads"] = systemInfo.CPU.Threads
		summary["cpu_frequency_mhz"] = systemInfo.CPU.FrequencyMHz
	}
	if systemInfo.Memory != nil {
		summary["memory_total_mb"] = systemInfo.Memory.TotalMB
		summary["memory_available_mb"] = systemInfo.Memory.AvailableMB
	}
	if systemInfo.Disk != nil {
		summary["disk_total_gb"] = systemInfo.Disk.TotalGB
		summary["disk_available_gb"] = systemInfo.Disk.AvailableGB
		summary["disk_type"] = systemInfo.Disk.DiskType
	}
	if systemInfo.OS != nil {
		summary["os"] = systemInfo.OS.Name
		summary["os_version"] = systemInfo.OS.Version
		summary["architecture"] = systemInfo.OS.Architecture
	}
	if systemInfo.Virtualization != nil {
		if systemInfo.Virtualization.IsVirtualized {
			summary["virtualization"] = systemInfo.Virtualization.Type
			if systemInfo.Virtualization.Vendor != "" {
				summary["virtualization_vendor"] = systemInfo.Virtualization.Vendor
			}
		} else {
			summary["virtualization"] = "Physical"
		}
	}
	if systemInfo.IPInfo != nil {
		summary["public_ip"] = systemInfo.IPInfo.PublicIP
		summary["isp"] = systemInfo.IPInfo.ISP
		if systemInfo.IPInfo.GeoLocation != nil {
			summary["location"] = strings.Trim(strings.Join([]string{
				systemInfo.IPInfo.GeoLocation.Country,
				systemInfo.IPInfo.GeoLocation.City,
			}, ", "), ", ")
			summary["country_code"] = systemInfo.IPInfo.GeoLocation.CountryCode
		}
	}
	return summary
}

func buildVPSCPUSummary(result *models.TestResult) map[string]interface{} {
	summary := map[string]interface{}{
		"backend": getCPUBackend(result),
		"status":  resultStatus(result),
	}
	addMetricFloat(summary, "single_core_score", result, "single_core_score")
	addMetricFloat(summary, "multi_core_score", result, "multi_core_score")
	addMetricFloat(summary, "total_score", result, "total_score")
	addMetricFloat(summary, "single_core_events_per_sec", result, "single_core_events_per_sec")
	addMetricFloat(summary, "multi_core_events_per_sec", result, "multi_core_events_per_sec")
	addMetricFloat(summary, "single_core_raw_score", result, "single_core_raw_score")
	addMetricFloat(summary, "multi_core_raw_score", result, "multi_core_raw_score")
	if cores, ok := metricInt(resultMetrics(result), "cpu_cores"); ok {
		summary["cpu_cores"] = cores
	}
	return summary
}

func buildVPSMemorySummary(result *models.TestResult) map[string]interface{} {
	summary := map[string]interface{}{
		"backend": getMemoryBackend(result),
		"status":  resultStatus(result),
	}
	if read, ok := getMemoryReadSpeed(result); ok {
		summary["read_mbps"] = read
	}
	if write, ok := getMemoryWriteSpeed(result); ok {
		summary["write_mbps"] = write
	}
	addMetricFloat(summary, "score", result, "score")
	return summary
}

func buildVPSDiskSummary(result *models.TestResult) map[string]interface{} {
	summary := map[string]interface{}{
		"backend": getDiskBackend(result),
		"status":  resultStatus(result),
	}
	if read, ok := getDiskReadSpeed(result); ok {
		summary["sequential_read_mbps"] = read
	}
	if write, ok := getDiskWriteSpeed(result); ok {
		summary["sequential_write_mbps"] = write
	}
	if iops, ok := getDiskRandomIOPS(result); ok {
		summary["random_iops"] = iops
	}
	if readIOPS, ok := getDiskRandomReadIOPS(result); ok {
		summary["random_read_iops"] = readIOPS
	}
	if writeIOPS, ok := getDiskRandomWriteIOPS(result); ok {
		summary["random_write_iops"] = writeIOPS
	}
	if readP95, ok := getDiskRandomReadP95Latency(result); ok {
		summary["random_read_p95_ms"] = readP95
	}
	if writeP95, ok := getDiskRandomWriteP95Latency(result); ok {
		summary["random_write_p95_ms"] = writeP95
	}
	for _, row := range getDiskFioMixedRows(result) {
		if row.BlockSize == "4k" {
			summary["fio_mixed_4k_total_iops"] = row.TotalIOPS
		}
		if row.BlockSize == "1m" {
			summary["fio_mixed_1m_total_mbps"] = row.TotalMBps
		}
	}
	addMetricFloat(summary, "score", result, "score")
	return summary
}

func buildVPSNetworkSummary(result *models.TestResult) map[string]interface{} {
	summary := map[string]interface{}{
		"backend": getNetworkBackend(result),
		"status":  resultStatus(result),
	}
	if server := getNetworkBackendServer(result); server != "" {
		summary["server"] = server
	}
	if latency, ok := getNetworkLatency(result); ok {
		summary["latency_ms"] = latency
	}
	if download, ok := getNetworkDownloadSpeed(result); ok {
		summary["download_mbps"] = download
	}
	if upload, ok := getNetworkUploadSpeed(result); ok {
		summary["upload_mbps"] = upload
	}
	summary["upload_estimated"] = isNetworkUploadEstimated(result)
	if node, ok := getNetworkSpeedtestNode(result); ok {
		if node.ServerID > 0 {
			summary["speedtest_server_id"] = node.ServerID
		}
		if node.Name != "" {
			summary["speedtest_server_name"] = node.Name
		}
		if node.Location != "" {
			summary["speedtest_server_location"] = node.Location
		}
		if node.Country != "" {
			summary["speedtest_server_country"] = node.Country
		}
		if node.Host != "" {
			summary["speedtest_server_host"] = node.Host
		}
		if node.ISP != "" {
			summary["speedtest_isp"] = node.ISP
		}
		if node.ExternalIP != "" {
			summary["speedtest_external_ip"] = node.ExternalIP
		}
		if node.ResultURL != "" {
			summary["speedtest_result_url"] = node.ResultURL
		}
		if node.PingJitter > 0 {
			summary["speedtest_ping_jitter_ms"] = node.PingJitter
		}
	}
	if rows := getNetworkIperf3MatrixRows(result); len(rows) > 0 {
		summary["iperf3_node_count"] = len(rows)
		addMetricFloat(summary, "iperf3_avg_download_mbps", result, "iperf3_matrix_avg_download_mbps")
		addMetricFloat(summary, "iperf3_avg_upload_mbps", result, "iperf3_matrix_avg_upload_mbps")
		addMetricFloat(summary, "iperf3_best_download_mbps", result, "iperf3_matrix_best_download_mbps")
		addMetricFloat(summary, "iperf3_best_upload_mbps", result, "iperf3_matrix_best_upload_mbps")
	}
	if rows := getNetworkQualityRows(result); len(rows) > 0 {
		ipv4, _ := metricBool(resultMetrics(result), "network_quality_ipv4_available")
		ipv6, _ := metricBool(resultMetrics(result), "network_quality_ipv6_available")
		summary["ipv4_available"] = ipv4
		summary["ipv6_available"] = ipv6
		addMetricFloat(summary, "quality_failure_rate", result, "network_quality_failure_rate")
		addMetricFloat(summary, "quality_avg_latency_ms", result, "network_quality_avg_latency_ms")
		addMetricFloat(summary, "quality_jitter_ms", result, "network_quality_jitter_ms")
	}
	if message := getNetworkError(result); message != "" {
		summary["error"] = message
	}
	addMetricFloat(summary, "score", result, "score")
	return summary
}

func buildVPSScoresSummary(reportSummary map[string]interface{}, overallScore *models.OverallScore) map[string]interface{} {
	summary := map[string]interface{}{}
	if overallScore != nil {
		summary["total_score"] = overallScore.TotalScore
		summary["grade"] = overallScore.Grade
		summary["cpu_score"] = overallScore.CPUScore
		summary["memory_score"] = overallScore.MemoryScore
		summary["disk_score"] = overallScore.DiskScore
		summary["network_score"] = overallScore.NetworkScore
	}
	if profile, ok := reportSummary["score_profile"].(string); ok {
		summary["score_profile"] = profile
	}
	return summary
}

func buildVPSConfidenceSummary(reportSummary map[string]interface{}) map[string]interface{} {
	summary := map[string]interface{}{}
	if confidence, ok := reportSummary["confidence_level"].(map[string]interface{}); ok {
		summary["level"] = confidence["level"]
		summary["reasons"] = confidence["reasons"]
	}
	if notes, ok := reportSummary["quality_notes"].([]string); ok {
		summary["quality_notes"] = notes
	}
	return summary
}

func (rg *ReportGenerator) FormatVPSBenchmarkSummary(report *models.Report) string {
	if report == nil || report.Summary == nil {
		return ""
	}
	summary, ok := report.Summary["vps_benchmark_summary"].(map[string]interface{})
	if !ok || len(summary) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("=== VPS测评摘要 ===\n\n")
	sb.WriteString(fmt.Sprintf("系统:           %s | %s | %s\n",
		summaryPathString(summary, "system", "cpu_model"),
		formatCoreThread(summary),
		formatMemoryDisk(summary)))
	sb.WriteString(fmt.Sprintf("环境:           %s %s | %s | %s\n",
		summaryPathString(summary, "system", "os"),
		summaryPathString(summary, "system", "architecture"),
		summaryPathString(summary, "system", "virtualization"),
		summaryPathString(summary, "system", "location")))
	sb.WriteString(fmt.Sprintf("CPU:            %s | 单核 %s | 多核 %s | 总分 %s\n",
		summaryPathString(summary, "cpu", "backend"),
		summaryPathNumber(summary, "cpu", "single_core_score"),
		summaryPathNumber(summary, "cpu", "multi_core_score"),
		summaryPathNumber(summary, "cpu", "total_score")))
	sb.WriteString(fmt.Sprintf("内存:           %s | 读 %s MB/s | 写 %s MB/s\n",
		summaryPathString(summary, "memory", "backend"),
		summaryPathNumber(summary, "memory", "read_mbps"),
		summaryPathNumber(summary, "memory", "write_mbps")))
	sb.WriteString(fmt.Sprintf("磁盘:           %s | 读 %s MB/s | 写 %s MB/s | 随机 %s IOPS\n",
		summaryPathString(summary, "disk", "backend"),
		summaryPathNumber(summary, "disk", "sequential_read_mbps"),
		summaryPathNumber(summary, "disk", "sequential_write_mbps"),
		summaryPathNumber(summary, "disk", "random_iops")))
	sb.WriteString(fmt.Sprintf("网络:           %s | 延迟 %s ms | 下载 %s Mbps | 上传 %s Mbps%s\n",
		summaryPathString(summary, "network", "backend"),
		summaryPathNumber(summary, "network", "latency_ms"),
		summaryPathNumber(summary, "network", "download_mbps"),
		summaryPathNumber(summary, "network", "upload_mbps"),
		estimatedSuffix(summary)))
	sb.WriteString(fmt.Sprintf("网络质量:       IPv4 %s | IPv6 %s | 抖动 %s ms | 失败率 %s%%\n",
		availableSummary(summary, "ipv4_available"),
		availableSummary(summary, "ipv6_available"),
		summaryPathNumber(summary, "network", "quality_jitter_ms"),
		summaryPathPercent(summary, "network", "quality_failure_rate")))
	sb.WriteString(fmt.Sprintf("评分:           总分 %s | 等级 %s | 置信 %s | 基准 %s\n\n",
		summaryPathNumber(summary, "scores", "total_score"),
		summaryPathString(summary, "scores", "grade"),
		summaryPathString(summary, "confidence", "level"),
		summaryPathString(summary, "scores", "score_profile")))
	return sb.String()
}

func resultMetrics(result *models.TestResult) map[string]interface{} {
	if result == nil {
		return nil
	}
	return result.Metrics
}

func resultStatus(result *models.TestResult) string {
	if result == nil {
		return "not_run"
	}
	return result.Status
}

func addMetricFloat(target map[string]interface{}, outputKey string, result *models.TestResult, metricKey string) {
	if value, ok := metricFloat64(resultMetrics(result), metricKey); ok {
		target[outputKey] = value
	}
}

func summarySection(summary map[string]interface{}, section string) map[string]interface{} {
	value, ok := summary[section].(map[string]interface{})
	if !ok {
		return nil
	}
	return value
}

func summaryPathString(summary map[string]interface{}, section string, key string) string {
	value, ok := summarySection(summary, section)[key]
	if !ok || value == nil || fmt.Sprintf("%v", value) == "" {
		return "-"
	}
	return fmt.Sprintf("%v", value)
}

func summaryPathNumber(summary map[string]interface{}, section string, key string) string {
	value, ok := summarySection(summary, section)[key]
	if !ok || value == nil {
		return "-"
	}
	switch v := value.(type) {
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%.2f", v)
	case float32:
		return fmt.Sprintf("%.2f", v)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func summaryPathPercent(summary map[string]interface{}, section string, key string) string {
	value, ok := summarySection(summary, section)[key]
	if !ok || value == nil {
		return "-"
	}
	switch v := value.(type) {
	case float64:
		return fmt.Sprintf("%.2f", v*100)
	case float32:
		return fmt.Sprintf("%.2f", v*100)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func formatCoreThread(summary map[string]interface{}) string {
	system := summarySection(summary, "system")
	if system == nil {
		return "-"
	}
	cores, coresOK := system["cpu_cores"]
	threads, threadsOK := system["cpu_threads"]
	if !coresOK || !threadsOK {
		return "-"
	}
	return fmt.Sprintf("%vC/%vT", cores, threads)
}

func formatMemoryDisk(summary map[string]interface{}) string {
	memory := summaryPathNumber(summary, "system", "memory_total_mb")
	disk := summaryPathNumber(summary, "system", "disk_total_gb")
	return fmt.Sprintf("%s MB RAM / %s GB Disk", memory, disk)
}

func estimatedSuffix(summary map[string]interface{}) string {
	value, ok := summarySection(summary, "network")["upload_estimated"].(bool)
	if ok && value {
		return " (估算)"
	}
	return ""
}

func availableSummary(summary map[string]interface{}, key string) string {
	value, ok := summarySection(summary, "network")[key].(bool)
	if !ok {
		return "-"
	}
	if value {
		return "可用"
	}
	return "不可用"
}
