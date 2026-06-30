package reporter

import (
	"fmt"
	"strconv"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
)

func metricFloat64(metrics map[string]interface{}, key string) (float64, bool) {
	if metrics == nil {
		return 0, false
	}

	value, ok := metrics[key]
	if !ok || value == nil {
		return 0, false
	}

	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint64:
		return float64(v), true
	case uint32:
		return float64(v), true
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		if err == nil {
			return parsed, true
		}
	}

	return 0, false
}

func metricInt(metrics map[string]interface{}, key string) (int, bool) {
	if metrics == nil {
		return 0, false
	}

	value, ok := metrics[key]
	if !ok || value == nil {
		return 0, false
	}

	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case int32:
		return int(v), true
	case float64:
		return int(v), true
	case float32:
		return int(v), true
	case uint:
		return int(v), true
	case uint64:
		return int(v), true
	case uint32:
		return int(v), true
	case string:
		parsed, err := strconv.Atoi(v)
		if err == nil {
			return parsed, true
		}
	}

	return 0, false
}

func metricBool(metrics map[string]interface{}, key string) (bool, bool) {
	if metrics == nil {
		return false, false
	}

	value, ok := metrics[key]
	if !ok || value == nil {
		return false, false
	}

	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		parsed, err := strconv.ParseBool(v)
		if err == nil {
			return parsed, true
		}
	}

	return false, false
}

func metricString(metrics map[string]interface{}, key string) (string, bool) {
	if metrics == nil {
		return "", false
	}

	value, ok := metrics[key]
	if !ok || value == nil {
		return "", false
	}

	switch v := value.(type) {
	case string:
		return v, true
	}

	return fmt.Sprintf("%v", value), true
}

func formatMetric(value float64, unit string) string {
	if unit == "" {
		return fmt.Sprintf("%.2f", value)
	}
	return fmt.Sprintf("%.2f %s", value, unit)
}

func getCPUScore(result *models.TestResult) (float64, bool) {
	if result == nil {
		return 0, false
	}
	return metricFloat64(result.Metrics, "total_score")
}

func getCPUBackend(result *models.TestResult) string {
	if result == nil {
		return ""
	}
	if backend, ok := metricString(result.Metrics, "backend"); ok {
		return backend
	}
	return ""
}

func getCPUSingleCoreEvents(result *models.TestResult) (float64, bool) {
	if result == nil {
		return 0, false
	}
	return metricFloat64(result.Metrics, "single_core_events_per_sec")
}

func getCPUSingleCoreRawScore(result *models.TestResult) (float64, bool) {
	if result == nil {
		return 0, false
	}
	return metricFloat64(result.Metrics, "single_core_raw_score")
}

func getCPUMultiCoreEvents(result *models.TestResult) (float64, bool) {
	if result == nil {
		return 0, false
	}
	return metricFloat64(result.Metrics, "multi_core_events_per_sec")
}

func getCPUMultiCoreRawScore(result *models.TestResult) (float64, bool) {
	if result == nil {
		return 0, false
	}
	return metricFloat64(result.Metrics, "multi_core_raw_score")
}

func getCPUScoreStdDev(result *models.TestResult) (float64, bool) {
	if result == nil {
		return 0, false
	}
	singleStdDev, singleOK := metricFloat64(result.Metrics, "single_core_score_stddev")
	multiStdDev, multiOK := metricFloat64(result.Metrics, "multi_core_score_stddev")
	if !singleOK || !multiOK {
		return 0, false
	}
	return singleStdDev*0.4 + multiStdDev*0.6, true
}

func getMemoryReadSpeed(result *models.TestResult) (float64, bool) {
	if result == nil {
		return 0, false
	}
	return metricFloat64(result.Metrics, "read_speed_mbps")
}

func getMemoryBackend(result *models.TestResult) string {
	if result == nil {
		return ""
	}
	if backend, ok := metricString(result.Metrics, "backend"); ok {
		return backend
	}
	return ""
}

func getMemoryReadSource(result *models.TestResult) string {
	if result == nil {
		return ""
	}
	if source, ok := metricString(result.Metrics, "read_speed_source"); ok {
		return source
	}
	return ""
}

func getMemoryWriteSource(result *models.TestResult) string {
	if result == nil {
		return ""
	}
	if source, ok := metricString(result.Metrics, "write_speed_source"); ok {
		return source
	}
	return ""
}

func getMemoryWriteSpeed(result *models.TestResult) (float64, bool) {
	if result == nil {
		return 0, false
	}
	return metricFloat64(result.Metrics, "write_speed_mbps")
}

func getMemoryReadStdDev(result *models.TestResult) (float64, bool) {
	if result == nil {
		return 0, false
	}
	return metricFloat64(result.Metrics, "read_speed_mbps_stddev")
}

func getMemoryWriteStdDev(result *models.TestResult) (float64, bool) {
	if result == nil {
		return 0, false
	}
	return metricFloat64(result.Metrics, "write_speed_mbps_stddev")
}

func getDiskReadSpeed(result *models.TestResult) (float64, bool) {
	if result == nil {
		return 0, false
	}
	if speed, ok := metricFloat64(result.Metrics, "sequential_read_mbps"); ok {
		return speed, true
	}
	return metricFloat64(result.Metrics, "read_speed_mbps")
}

func getDiskWriteSpeed(result *models.TestResult) (float64, bool) {
	if result == nil {
		return 0, false
	}
	if speed, ok := metricFloat64(result.Metrics, "sequential_write_mbps"); ok {
		return speed, true
	}
	return metricFloat64(result.Metrics, "write_speed_mbps")
}

func getDiskRandomIOPS(result *models.TestResult) (int, bool) {
	if result == nil {
		return 0, false
	}
	return metricInt(result.Metrics, "random_iops")
}

func getDiskRandomReadIOPS(result *models.TestResult) (float64, bool) {
	if result == nil {
		return 0, false
	}
	return metricFloat64(result.Metrics, "random_read_iops")
}

func getDiskRandomWriteIOPS(result *models.TestResult) (float64, bool) {
	if result == nil {
		return 0, false
	}
	return metricFloat64(result.Metrics, "random_write_iops")
}

func getDiskRandomReadP95Latency(result *models.TestResult) (float64, bool) {
	if result == nil {
		return 0, false
	}
	return metricFloat64(result.Metrics, "random_read_latency_p95_ms")
}

func getDiskRandomWriteP95Latency(result *models.TestResult) (float64, bool) {
	if result == nil {
		return 0, false
	}
	return metricFloat64(result.Metrics, "random_write_latency_p95_ms")
}

type diskFioMixedRow struct {
	BlockSize string
	ReadMBps  float64
	WriteMBps float64
	TotalMBps float64
	ReadIOPS  float64
	WriteIOPS float64
	TotalIOPS float64
}

func getDiskFioMixedRows(result *models.TestResult) []diskFioMixedRow {
	if result == nil || result.Metrics == nil {
		return nil
	}

	blockSizes := []string{"4k", "64k", "512k", "1m"}
	rows := make([]diskFioMixedRow, 0, len(blockSizes))
	for _, blockSize := range blockSizes {
		prefix := "fio_mixed_" + blockSize
		totalIOPS, iopsOK := metricFloat64(result.Metrics, prefix+"_total_iops")
		totalMBps, mbpsOK := metricFloat64(result.Metrics, prefix+"_total_mbps")
		if !iopsOK && !mbpsOK {
			continue
		}

		readMBps, _ := metricFloat64(result.Metrics, prefix+"_read_mbps")
		writeMBps, _ := metricFloat64(result.Metrics, prefix+"_write_mbps")
		readIOPS, _ := metricFloat64(result.Metrics, prefix+"_read_iops")
		writeIOPS, _ := metricFloat64(result.Metrics, prefix+"_write_iops")
		rows = append(rows, diskFioMixedRow{
			BlockSize: blockSize,
			ReadMBps:  readMBps,
			WriteMBps: writeMBps,
			TotalMBps: totalMBps,
			ReadIOPS:  readIOPS,
			WriteIOPS: writeIOPS,
			TotalIOPS: totalIOPS,
		})
	}
	return rows
}

func getDiskBackend(result *models.TestResult) string {
	if result == nil {
		return ""
	}
	if backend, ok := metricString(result.Metrics, "backend"); ok {
		return backend
	}
	return ""
}

func getNetworkLatency(result *models.TestResult) (float64, bool) {
	if result == nil {
		return 0, false
	}
	if latency, ok := metricFloat64(result.Metrics, "average_latency_ms"); ok {
		return latency, true
	}
	return metricFloat64(result.Metrics, "latency_ms")
}

func getNetworkBackend(result *models.TestResult) string {
	if result == nil {
		return ""
	}
	if backend, ok := metricString(result.Metrics, "backend"); ok {
		return backend
	}
	return ""
}

func getNetworkBackendServer(result *models.TestResult) string {
	if result == nil {
		return ""
	}
	if server, ok := metricString(result.Metrics, "backend_server"); ok {
		return server
	}
	return ""
}

type networkIperf3MatrixRow struct {
	Index         int
	Server        string
	Name          string
	Region        string
	Provider      string
	Authorization string
	Protocol      string
	DownloadMbps  float64
	UploadMbps    float64
	LatencyMs     float64
	Error         string
}

type networkQualityRow struct {
	Index        int
	Target       string
	Address      string
	Protocol     string
	Available    bool
	SuccessCount int
	FailureCount int
	FailureRate  float64
	AvgLatencyMs float64
	MinLatencyMs float64
	MaxLatencyMs float64
	JitterMs     float64
}

type networkSpeedtestNode struct {
	ServerID   int
	Name       string
	Location   string
	Country    string
	Host       string
	ResultURL  string
	ISP        string
	ExternalIP string
	Interface  string
	IsVPN      bool
	PingJitter float64
}

func getNetworkIperf3MatrixRows(result *models.TestResult) []networkIperf3MatrixRow {
	if result == nil || result.Metrics == nil {
		return nil
	}

	count, ok := metricInt(result.Metrics, "iperf3_matrix_server_count")
	if !ok || count <= 0 {
		return nil
	}
	rows := make([]networkIperf3MatrixRow, 0, count)
	for i := 1; i <= count; i++ {
		prefix := fmt.Sprintf("iperf3_matrix_%d", i)
		server, _ := metricString(result.Metrics, prefix+"_server")
		name, _ := metricString(result.Metrics, prefix+"_name")
		region, _ := metricString(result.Metrics, prefix+"_region")
		provider, _ := metricString(result.Metrics, prefix+"_provider")
		authorization, _ := metricString(result.Metrics, prefix+"_authorization")
		protocol, _ := metricString(result.Metrics, prefix+"_protocol")
		download, _ := metricFloat64(result.Metrics, prefix+"_download_mbps")
		upload, _ := metricFloat64(result.Metrics, prefix+"_upload_mbps")
		latency, _ := metricFloat64(result.Metrics, prefix+"_latency_ms")
		errorMessage, _ := metricString(result.Metrics, prefix+"_error")
		rows = append(rows, networkIperf3MatrixRow{
			Index:         i,
			Server:        server,
			Name:          name,
			Region:        region,
			Provider:      provider,
			Authorization: authorization,
			Protocol:      protocol,
			DownloadMbps:  download,
			UploadMbps:    upload,
			LatencyMs:     latency,
			Error:         errorMessage,
		})
	}
	return rows
}

func getNetworkSpeedtestNode(result *models.TestResult) (networkSpeedtestNode, bool) {
	if result == nil || result.Metrics == nil {
		return networkSpeedtestNode{}, false
	}
	if profile, ok := metricString(result.Metrics, "speedtest_profile"); ok && profile != "" {
		node := networkSpeedtestNode{}
		node.ServerID, _ = metricInt(result.Metrics, "speedtest_server_id")
		node.Name, _ = metricString(result.Metrics, "speedtest_server_name")
		node.Location, _ = metricString(result.Metrics, "speedtest_server_location")
		node.Country, _ = metricString(result.Metrics, "speedtest_server_country")
		node.Host, _ = metricString(result.Metrics, "speedtest_server_host")
		node.ResultURL, _ = metricString(result.Metrics, "speedtest_result_url")
		node.ISP, _ = metricString(result.Metrics, "speedtest_isp")
		node.ExternalIP, _ = metricString(result.Metrics, "speedtest_interface_external_ip")
		node.Interface, _ = metricString(result.Metrics, "speedtest_interface_name")
		node.IsVPN, _ = metricBool(result.Metrics, "speedtest_interface_is_vpn")
		node.PingJitter, _ = metricFloat64(result.Metrics, "speedtest_ping_jitter_ms")
		return node, true
	}
	return networkSpeedtestNode{}, false
}

func networkSpeedtestServerLabel(node networkSpeedtestNode) string {
	if node.ServerID > 0 && node.Name != "" {
		return fmt.Sprintf("%d/%s", node.ServerID, node.Name)
	}
	if node.Name != "" {
		return node.Name
	}
	if node.ServerID > 0 {
		return fmt.Sprintf("%d", node.ServerID)
	}
	return "-"
}

func getNetworkQualityRows(result *models.TestResult) []networkQualityRow {
	if result == nil || result.Metrics == nil {
		return nil
	}

	count, ok := metricInt(result.Metrics, "network_quality_target_count")
	if !ok || count <= 0 {
		return nil
	}
	rows := make([]networkQualityRow, 0, count)
	for i := 1; i <= count; i++ {
		prefix := fmt.Sprintf("network_quality_%d", i)
		target, _ := metricString(result.Metrics, prefix+"_target")
		address, _ := metricString(result.Metrics, prefix+"_address")
		protocol, _ := metricString(result.Metrics, prefix+"_protocol")
		available, _ := metricBool(result.Metrics, prefix+"_available")
		successCount, _ := metricInt(result.Metrics, prefix+"_success_count")
		failureCount, _ := metricInt(result.Metrics, prefix+"_failure_count")
		failureRate, _ := metricFloat64(result.Metrics, prefix+"_failure_rate")
		avgLatency, _ := metricFloat64(result.Metrics, prefix+"_avg_latency_ms")
		minLatency, _ := metricFloat64(result.Metrics, prefix+"_min_latency_ms")
		maxLatency, _ := metricFloat64(result.Metrics, prefix+"_max_latency_ms")
		jitter, _ := metricFloat64(result.Metrics, prefix+"_jitter_ms")
		rows = append(rows, networkQualityRow{
			Index:        i,
			Target:       target,
			Address:      address,
			Protocol:     protocol,
			Available:    available,
			SuccessCount: successCount,
			FailureCount: failureCount,
			FailureRate:  failureRate,
			AvgLatencyMs: avgLatency,
			MinLatencyMs: minLatency,
			MaxLatencyMs: maxLatency,
			JitterMs:     jitter,
		})
	}
	return rows
}

func getNetworkDownloadSpeed(result *models.TestResult) (float64, bool) {
	if result == nil {
		return 0, false
	}
	return metricFloat64(result.Metrics, "download_speed_mbps")
}

func getNetworkUploadSpeed(result *models.TestResult) (float64, bool) {
	if result == nil {
		return 0, false
	}
	return metricFloat64(result.Metrics, "upload_speed_mbps")
}

func getNetworkLatencySource(result *models.TestResult) string {
	if result == nil {
		return ""
	}
	if source, ok := metricString(result.Metrics, "latency_source"); ok {
		return source
	}
	return ""
}

func getNetworkDownloadSource(result *models.TestResult) string {
	if result == nil {
		return ""
	}
	if source, ok := metricString(result.Metrics, "download_speed_source"); ok {
		return source
	}
	return ""
}

func getNetworkUploadSource(result *models.TestResult) string {
	if result == nil {
		return ""
	}
	if source, ok := metricString(result.Metrics, "upload_speed_source"); ok {
		return source
	}
	return ""
}

func getNetworkError(result *models.TestResult) string {
	if result == nil {
		return ""
	}
	if message, ok := metricString(result.Metrics, "network_error"); ok {
		return message
	}
	return ""
}

type benchmarkErrorSummary struct {
	Category string
	Stage    string
	Hint     string
}

func getBenchmarkErrorSummary(result *models.TestResult) benchmarkErrorSummary {
	if result == nil || result.Metrics == nil {
		return benchmarkErrorSummary{}
	}
	summary := benchmarkErrorSummary{}
	summary.Category, _ = metricString(result.Metrics, "error_category")
	summary.Stage, _ = metricString(result.Metrics, "error_stage")
	summary.Hint, _ = metricString(result.Metrics, "error_hint")
	return summary
}

func benchmarkErrorCategoryText(category string) string {
	switch category {
	case "missing_dependency":
		return "依赖缺失"
	case "invalid_config":
		return "配置错误"
	case "command_failed":
		return "命令执行失败"
	case "permission_denied":
		return "权限不足"
	case "resource_limited":
		return "资源不足"
	case "network_unavailable":
		return "网络不可达"
	case "parse_failed":
		return "结果解析失败"
	case "timeout":
		return "执行超时"
	case "runtime_error":
		return "运行异常"
	default:
		return category
	}
}

func isNetworkUploadEstimated(result *models.TestResult) bool {
	if result == nil {
		return false
	}
	if estimated, ok := metricBool(result.Metrics, "upload_speed_estimated"); ok {
		return estimated
	}
	return false
}
