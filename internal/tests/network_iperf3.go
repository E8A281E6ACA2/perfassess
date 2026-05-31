package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
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
	server  string
	timeout time.Duration
	runner  commandRunner
}

type Iperf3Config struct {
	Server  string
	Timeout time.Duration
	Runner  commandRunner
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
		server:  cfg.Server,
		timeout: timeout,
		runner:  runner,
	}
}

func (b *Iperf3NetworkBackend) Name() string {
	return models.NetworkBackendIperf3
}

func (b *Iperf3NetworkBackend) Server() string {
	return b.server
}

func (b *Iperf3NetworkBackend) DownloadSource() string {
	return models.NetworkDownloadSourceIperf3
}

func (b *Iperf3NetworkBackend) UploadSource(estimated bool) string {
	return models.NetworkUploadSourceIperf3
}

func (b *Iperf3NetworkBackend) MeasureLatency(hosts []string) (float64, error) {
	return 0, fmt.Errorf("iperf3 latency measurement is not implemented")
}

func (b *Iperf3NetworkBackend) MeasureDownload() (float64, error) {
	if err := b.validateReady(); err != nil {
		return 0, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	output, err := b.runner.Run(ctx, "iperf3", "-c", b.server, "--json")
	if err != nil {
		return 0, fmt.Errorf("iperf3 download failed: %w", err)
	}

	speed, err := parseIperf3Mbps(output)
	if err != nil {
		return 0, fmt.Errorf("parse iperf3 download result: %w", err)
	}
	return speed, nil
}

func (b *Iperf3NetworkBackend) MeasureUpload(downloadSpeed float64) (float64, bool, error) {
	if err := b.validateReady(); err != nil {
		return 0, false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	output, err := b.runner.Run(ctx, "iperf3", "-c", b.server, "--reverse", "--json")
	if err != nil {
		return 0, false, fmt.Errorf("iperf3 upload failed: %w", err)
	}

	speed, err := parseIperf3Mbps(output)
	if err != nil {
		return 0, false, fmt.Errorf("parse iperf3 upload result: %w", err)
	}
	return speed, false, nil
}

func (b *Iperf3NetworkBackend) validateReady() error {
	if b.server == "" {
		return fmt.Errorf("iperf3 server is required; example: --network-backend iperf3 --iperf3-server 1.2.3.4:5201")
	}
	if _, err := b.runner.LookPath("iperf3"); err != nil {
		return fmt.Errorf("iperf3 is not installed; install it manually before using --network-backend iperf3. Ubuntu/Debian: sudo apt install iperf3; RHEL/CentOS: sudo yum install iperf3; macOS: brew install iperf3")
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
