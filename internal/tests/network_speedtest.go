package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"performance-assessment-system/internal/models"
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
		return 0, fmt.Errorf("speedtest result missing ping latency")
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
		return NetworkDownloadResult{}, fmt.Errorf("speedtest result missing download bandwidth")
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
		return 0, false, fmt.Errorf("speedtest result missing upload bandwidth")
	}
	return speed, false, nil
}

func (b *SpeedtestNetworkBackend) measure() (*speedtestJSONResult, error) {
	if b.result != nil {
		return b.result, nil
	}
	if _, err := b.runner.LookPath("speedtest"); err != nil {
		return nil, fmt.Errorf("speedtest CLI is not installed; install Ookla speedtest before using --network-backend speedtest. Ubuntu/Debian: install from https://www.speedtest.net/apps/cli; macOS: brew install speedtest-cli")
	}

	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	output, err := b.runner.Run(ctx, "speedtest", "--format=json", "--accept-license", "--accept-gdpr")
	if err != nil {
		return nil, fmt.Errorf("speedtest failed: %w", err)
	}
	result, err := parseSpeedtestJSON(output)
	if err != nil {
		return nil, err
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
