package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
)

type SpeedtestNetworkBackend struct {
	timeout time.Duration
	runner  commandRunner
	result  *speedtestJSONResult
}

type SpeedtestConfig struct {
	Timeout time.Duration
	Runner  commandRunner
}

type speedtestJSONResult struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Ping      struct {
		Latency float64 `json:"latency"`
		Jitter  float64 `json:"jitter"`
	} `json:"ping"`
	Download struct {
		Bandwidth float64 `json:"bandwidth"`
		Bytes     int64   `json:"bytes"`
		Elapsed   int64   `json:"elapsed"`
	} `json:"download"`
	Upload struct {
		Bandwidth float64 `json:"bandwidth"`
		Bytes     int64   `json:"bytes"`
		Elapsed   int64   `json:"elapsed"`
	} `json:"upload"`
	Server struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Location string `json:"location"`
		Country  string `json:"country"`
		Host     string `json:"host"`
	} `json:"server"`
	Result struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"result"`
	Interface struct {
		InternalIP string `json:"internalIp"`
		Name       string `json:"name"`
		MACAddress string `json:"macAddr"`
		IsVPN      bool   `json:"isVpn"`
		ExternalIP string `json:"externalIp"`
	} `json:"interface"`
	ISP string `json:"isp"`
}

func NewSpeedtestNetworkBackend(cfg SpeedtestConfig) *SpeedtestNetworkBackend {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 90 * time.Second
	}
	runner := cfg.Runner
	if runner == nil {
		runner = &execCommandRunner{}
	}
	return &SpeedtestNetworkBackend{
		timeout: timeout,
		runner:  runner,
	}
}

func (b *SpeedtestNetworkBackend) Name() string {
	return models.NetworkBackendSpeedtest
}

func (b *SpeedtestNetworkBackend) Server() string {
	if b.result == nil || b.result.Server.Host == "" {
		return ""
	}
	return b.result.Server.Host
}

func (b *SpeedtestNetworkBackend) DownloadSource(result NetworkDownloadResult) string {
	return models.NetworkDownloadSourceSpeedtest
}

func (b *SpeedtestNetworkBackend) UploadSource(estimated bool) string {
	return models.NetworkUploadSourceSpeedtest
}

func (b *SpeedtestNetworkBackend) MeasureLatency(hosts []string) (float64, error) {
	result, err := b.measure()
	if err != nil {
		return 0, err
	}
	if result.Ping.Latency <= 0 {
		return 0, newBenchmarkError(BenchmarkErrorParseFailed, "speedtest_latency_parse", "speedtest JSON 中缺少 ping latency，请保留原始输出排查。", fmt.Errorf("speedtest result missing ping latency"))
	}
	return result.Ping.Latency, nil
}

func (b *SpeedtestNetworkBackend) MeasureDownload() (NetworkDownloadResult, error) {
	result, err := b.measure()
	if err != nil {
		return NetworkDownloadResult{}, err
	}
	speed := bandwidthBytesPerSecondToMbps(result.Download.Bandwidth)
	if speed <= 0 {
		return NetworkDownloadResult{}, newBenchmarkError(BenchmarkErrorParseFailed, "speedtest_download_parse", "speedtest JSON 中缺少 download bandwidth，请保留原始输出排查。", fmt.Errorf("speedtest result missing download bandwidth"))
	}
	return NetworkDownloadResult{SpeedMbps: speed, SourceURL: models.NetworkDownloadSourceSpeedtest}, nil
}

func (b *SpeedtestNetworkBackend) MeasureUpload(downloadSpeed float64) (float64, bool, error) {
	result, err := b.measure()
	if err != nil {
		return 0, false, err
	}
	speed := bandwidthBytesPerSecondToMbps(result.Upload.Bandwidth)
	if speed <= 0 {
		return 0, false, newBenchmarkError(BenchmarkErrorParseFailed, "speedtest_upload_parse", "speedtest JSON 中缺少 upload bandwidth，请保留原始输出排查。", fmt.Errorf("speedtest result missing upload bandwidth"))
	}
	return speed, false, nil
}

func (b *SpeedtestNetworkBackend) AppendMetrics(metrics map[string]interface{}) {
	if metrics == nil || b.result == nil {
		return
	}
	result := b.result
	metrics["speedtest_profile"] = "ookla_cli"
	if result.Server.ID > 0 {
		metrics["speedtest_server_id"] = result.Server.ID
	}
	addStringMetric(metrics, "speedtest_server_name", result.Server.Name)
	addStringMetric(metrics, "speedtest_server_location", result.Server.Location)
	addStringMetric(metrics, "speedtest_server_country", result.Server.Country)
	addStringMetric(metrics, "speedtest_server_host", result.Server.Host)
	addStringMetric(metrics, "speedtest_result_id", result.Result.ID)
	addStringMetric(metrics, "speedtest_result_url", result.Result.URL)
	addStringMetric(metrics, "speedtest_isp", result.ISP)
	addStringMetric(metrics, "speedtest_interface_name", result.Interface.Name)
	addStringMetric(metrics, "speedtest_interface_internal_ip", result.Interface.InternalIP)
	addStringMetric(metrics, "speedtest_interface_external_ip", result.Interface.ExternalIP)
	addStringMetric(metrics, "speedtest_interface_mac", result.Interface.MACAddress)
	metrics["speedtest_interface_is_vpn"] = result.Interface.IsVPN
	if result.Ping.Jitter > 0 {
		metrics["speedtest_ping_jitter_ms"] = result.Ping.Jitter
	}
}

func (b *SpeedtestNetworkBackend) measure() (*speedtestJSONResult, error) {
	if b.result != nil {
		return b.result, nil
	}
	if _, err := b.runner.LookPath("speedtest"); err != nil {
		return nil, newBenchmarkError(BenchmarkErrorMissingDependency, "speedtest_lookup", "安装 Ookla Speedtest CLI 后重试。Ubuntu/Debian 请参考 speedtest.net/apps/cli；macOS: brew install speedtest-cli", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	output, err := b.runner.Run(ctx, "speedtest", "--format=json", "--accept-license", "--accept-gdpr")
	if err != nil {
		return nil, classifyExternalCommandError("speedtest_run", output, err, "确认 speedtest CLI 可执行、许可参数可用，并检查当前网络是否允许测速。")
	}
	result, err := parseSpeedtestJSON(output)
	if err != nil {
		return nil, newBenchmarkError(BenchmarkErrorParseFailed, "speedtest_parse", "speedtest JSON 输出格式无法识别，请保留 stdout/stderr 用于排查。", err)
	}
	b.result = result
	return result, nil
}

func parseSpeedtestJSON(output []byte) (*speedtestJSONResult, error) {
	var result speedtestJSONResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("parse speedtest result: %w", err)
	}
	if result.Type != "" && result.Type != "result" {
		return nil, fmt.Errorf("speedtest returned non-result payload type %q", result.Type)
	}
	return &result, nil
}

func bandwidthBytesPerSecondToMbps(value float64) float64 {
	if value <= 0 {
		return 0
	}
	return value * 8 / 1000000
}

func addStringMetric(metrics map[string]interface{}, key, value string) {
	if value != "" {
		metrics[key] = value
	}
}
