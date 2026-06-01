package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"performance-assessment-system/internal/models"
)

type commandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
	LookPath(name string) (string, error)
}

type execCommandRunner struct{}

func (r *execCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

func (r *execCommandRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

type Iperf3NetworkBackend struct {
	server    string
	timeout   time.Duration
	runner    commandRunner
	latencyFn func([]string) (float64, error)
}

type Iperf3Config struct {
	Server    string
	Timeout   time.Duration
	Runner    commandRunner
	LatencyFn func([]string) (float64, error)
}

type iperf3Endpoint struct {
	Host string
	Port string
}

func NewIperf3NetworkBackend(cfg Iperf3Config) *Iperf3NetworkBackend {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 15 * time.Second
	}
	runner := cfg.Runner
	if runner == nil {
		runner = &execCommandRunner{}
	}

	return &Iperf3NetworkBackend{
		server:    cfg.Server,
		timeout:   timeout,
		runner:    runner,
		latencyFn: cfg.LatencyFn,
	}
}

func (b *Iperf3NetworkBackend) Name() string {
	return models.NetworkBackendIperf3
}

func (b *Iperf3NetworkBackend) Server() string {
	return b.server
}

func (b *Iperf3NetworkBackend) DownloadSource(result NetworkDownloadResult) string {
	return models.NetworkDownloadSourceIperf3
}

func (b *Iperf3NetworkBackend) UploadSource(estimated bool) string {
	return models.NetworkUploadSourceIperf3
}

func (b *Iperf3NetworkBackend) MeasureLatency(hosts []string) (float64, error) {
	if b.latencyFn == nil {
		return 0, fmt.Errorf("iperf3 latency fallback is not configured")
	}
	return b.latencyFn(hosts)
}

func (b *Iperf3NetworkBackend) MeasureDownload() (NetworkDownloadResult, error) {
	endpoint, err := b.validateReady()
	if err != nil {
		return NetworkDownloadResult{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	output, err := b.runner.Run(ctx, "iperf3", b.commandArgs(endpoint, false)...)
	if err != nil {
		return NetworkDownloadResult{}, fmt.Errorf("iperf3 download failed: %w", err)
	}

	speed, err := parseIperf3Mbps(output)
	if err != nil {
		return NetworkDownloadResult{}, fmt.Errorf("parse iperf3 download result: %w", err)
	}
	return NetworkDownloadResult{SpeedMbps: speed, SourceURL: models.NetworkDownloadSourceIperf3}, nil
}

func (b *Iperf3NetworkBackend) MeasureUpload(downloadSpeed float64) (float64, bool, error) {
	endpoint, err := b.validateReady()
	if err != nil {
		return 0, false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	output, err := b.runner.Run(ctx, "iperf3", b.commandArgs(endpoint, true)...)
	if err != nil {
		return 0, false, fmt.Errorf("iperf3 upload failed: %w", err)
	}

	speed, err := parseIperf3Mbps(output)
	if err != nil {
		return 0, false, fmt.Errorf("parse iperf3 upload result: %w", err)
	}
	return speed, false, nil
}

func (b *Iperf3NetworkBackend) validateReady() (iperf3Endpoint, error) {
	endpoint, err := parseIperf3Endpoint(b.server)
	if err != nil {
		return iperf3Endpoint{}, err
	}
	if _, err := b.runner.LookPath("iperf3"); err != nil {
		return iperf3Endpoint{}, fmt.Errorf("iperf3 is not installed; install it manually before using --network-backend iperf3. Ubuntu/Debian: sudo apt install iperf3; RHEL/CentOS: sudo yum install iperf3; macOS: brew install iperf3")
	}
	return endpoint, nil
}

func (b *Iperf3NetworkBackend) commandArgs(endpoint iperf3Endpoint, reverse bool) []string {
	args := []string{"-c", endpoint.Host}
	if endpoint.Port != "" {
		args = append(args, "-p", endpoint.Port)
	}
	if reverse {
		args = append(args, "--reverse")
	}
	return append(args, "--json")
}

func parseIperf3Endpoint(server string) (iperf3Endpoint, error) {
	server = strings.TrimSpace(server)
	if server == "" {
		return iperf3Endpoint{}, fmt.Errorf("iperf3 server is required; example: --network-backend iperf3 --iperf3-server 1.2.3.4:5201")
	}

	host, port, err := net.SplitHostPort(server)
	if err == nil {
		if host == "" {
			return iperf3Endpoint{}, fmt.Errorf("iperf3 server host is required")
		}
		if err := validateIperf3Port(port); err != nil {
			return iperf3Endpoint{}, err
		}
		return iperf3Endpoint{Host: host, Port: port}, nil
	}

	if strings.Count(server, ":") > 1 {
		return iperf3Endpoint{}, fmt.Errorf("invalid iperf3 server %q; use [ipv6]:port or host:port", server)
	}
	if strings.Contains(server, ":") {
		lastColon := strings.LastIndex(server, ":")
		host = strings.TrimSpace(server[:lastColon])
		port = strings.TrimSpace(server[lastColon+1:])
		if host == "" {
			return iperf3Endpoint{}, fmt.Errorf("iperf3 server host is required")
		}
		if err := validateIperf3Port(port); err != nil {
			return iperf3Endpoint{}, err
		}
		return iperf3Endpoint{Host: host, Port: port}, nil
	}

	return iperf3Endpoint{Host: server}, nil
}

func validateIperf3Port(port string) error {
	if port == "" {
		return fmt.Errorf("iperf3 server port is required when using host:port")
	}
	value, err := strconv.Atoi(port)
	if err != nil || value < 1 || value > 65535 {
		return fmt.Errorf("invalid iperf3 server port %q; expected 1-65535", port)
	}
	return nil
}

type iperf3JSONResult struct {
	End struct {
		SumReceived *struct {
			BitsPerSecond float64 `json:"bits_per_second"`
		} `json:"sum_received"`
		SumSent *struct {
			BitsPerSecond float64 `json:"bits_per_second"`
		} `json:"sum_sent"`
	} `json:"end"`
}

func parseIperf3Mbps(output []byte) (float64, error) {
	var result iperf3JSONResult
	if err := json.Unmarshal(output, &result); err != nil {
		return 0, err
	}

	bitsPerSecond := 0.0
	if result.End.SumReceived != nil && result.End.SumReceived.BitsPerSecond > 0 {
		bitsPerSecond = result.End.SumReceived.BitsPerSecond
	} else if result.End.SumSent != nil && result.End.SumSent.BitsPerSecond > 0 {
		bitsPerSecond = result.End.SumSent.BitsPerSecond
	}
	if bitsPerSecond <= 0 {
		return 0, fmt.Errorf("missing bits_per_second in iperf3 result")
	}

	return bitsPerSecond / 1000000.0, nil
}
