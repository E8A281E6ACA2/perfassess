package reporter

import (
	"fmt"
	"strconv"

	"performance-assessment-system/internal/models"
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
	return metricFloat64(result.Metrics, "total_score")
}

func getMemoryReadSpeed(result *models.TestResult) (float64, bool) {
	return metricFloat64(result.Metrics, "read_speed_mbps")
}

func getMemoryWriteSpeed(result *models.TestResult) (float64, bool) {
	return metricFloat64(result.Metrics, "write_speed_mbps")
}

func getDiskReadSpeed(result *models.TestResult) (float64, bool) {
	if speed, ok := metricFloat64(result.Metrics, "sequential_read_mbps"); ok {
		return speed, true
	}
	return metricFloat64(result.Metrics, "read_speed_mbps")
}

func getDiskWriteSpeed(result *models.TestResult) (float64, bool) {
	if speed, ok := metricFloat64(result.Metrics, "sequential_write_mbps"); ok {
		return speed, true
	}
	return metricFloat64(result.Metrics, "write_speed_mbps")
}

func getDiskRandomIOPS(result *models.TestResult) (int, bool) {
	return metricInt(result.Metrics, "random_iops")
}

func getNetworkLatency(result *models.TestResult) (float64, bool) {
	if latency, ok := metricFloat64(result.Metrics, "average_latency_ms"); ok {
		return latency, true
	}
	return metricFloat64(result.Metrics, "latency_ms")
}

func getNetworkDownloadSpeed(result *models.TestResult) (float64, bool) {
	return metricFloat64(result.Metrics, "download_speed_mbps")
}

func getNetworkUploadSpeed(result *models.TestResult) (float64, bool) {
	return metricFloat64(result.Metrics, "upload_speed_mbps")
}

func getNetworkLatencySource(result *models.TestResult) string {
	if source, ok := metricString(result.Metrics, "latency_source"); ok {
		return source
	}
	return ""
}

func getNetworkDownloadSource(result *models.TestResult) string {
	if source, ok := metricString(result.Metrics, "download_speed_source"); ok {
		return source
	}
	return ""
}

func getNetworkUploadSource(result *models.TestResult) string {
	if source, ok := metricString(result.Metrics, "upload_speed_source"); ok {
		return source
	}
	return ""
}

func isNetworkUploadEstimated(result *models.TestResult) bool {
	if estimated, ok := metricBool(result.Metrics, "upload_speed_estimated"); ok {
		return estimated
	}
	return false
}
