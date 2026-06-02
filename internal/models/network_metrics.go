package models

import "strings"

const (
	NetworkBackendBuiltin   = "builtin"
	NetworkBackendIperf3    = "iperf3"
	NetworkBackendSpeedtest = "speedtest"
)

const (
	NetworkLatencySourceTCPConnect = "tcp_connect"
)

const (
	NetworkDownloadSourceHTTP       = "http_download"
	NetworkDownloadSourceHTTPFailed = "http_download_failed"
	NetworkDownloadSourceIperf3     = "iperf3_download"
	NetworkDownloadSourceSpeedtest  = "speedtest_download"
)

const (
	NetworkUploadSourceHTTP        = "http_upload"
	NetworkUploadSourceUnavailable = "unavailable"
	NetworkUploadSourceEstimated   = "estimated_from_download"
	NetworkUploadSourceIperf3      = "iperf3_upload"
	NetworkUploadSourceSpeedtest   = "speedtest_upload"
)

// NetworkMetrics 表示网络测试的结构化结果
// 作为弱类型 Metrics map 的上游数据结构，便于后续接入 iperf3 等后端。
type NetworkMetrics struct {
	Backend           string  `json:"backend,omitempty"`
	BackendServer     string  `json:"backend_server,omitempty"`
	LatencyMs         float64 `json:"latency_ms"`
	AverageLatencyMs  float64 `json:"average_latency_ms"`
	LatencySource     string  `json:"latency_source,omitempty"`
	DownloadSpeedMbps float64 `json:"download_speed_mbps"`
	DownloadSource    string  `json:"download_speed_source,omitempty"`
	UploadSpeedMbps   float64 `json:"upload_speed_mbps"`
	UploadSource      string  `json:"upload_speed_source,omitempty"`
	UploadEstimated   bool    `json:"upload_speed_estimated"`
	ErrorMessage      string  `json:"network_error,omitempty"`
	Score             float64 `json:"score"`
}

// AppendError adds a user-facing network error while preserving prior failures.
func (nm *NetworkMetrics) AppendError(message string) {
	if nm == nil || message == "" {
		return
	}
	if nm.ErrorMessage == "" {
		nm.ErrorMessage = message
		return
	}
	nm.ErrorMessage = strings.Join([]string{nm.ErrorMessage, message}, "; ")
}

// ToMetricsMap 将结构化网络结果转换为当前报告层使用的 Metrics map
func (nm *NetworkMetrics) ToMetricsMap() map[string]interface{} {
	if nm == nil {
		return nil
	}

	return map[string]interface{}{
		"backend":                nm.Backend,
		"backend_server":         nm.BackendServer,
		"latency_ms":             nm.LatencyMs,
		"average_latency_ms":     nm.AverageLatencyMs,
		"latency_source":         nm.LatencySource,
		"download_speed_mbps":    nm.DownloadSpeedMbps,
		"download_speed_source":  nm.DownloadSource,
		"upload_speed_mbps":      nm.UploadSpeedMbps,
		"upload_speed_source":    nm.UploadSource,
		"upload_speed_estimated": nm.UploadEstimated,
		"network_error":          nm.ErrorMessage,
		"score":                  nm.Score,
	}
}
