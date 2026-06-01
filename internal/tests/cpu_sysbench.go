package tests

import (
	"context"
	"fmt"
	"regexp"
	"runtime"
	"strconv"
	"time"

	"performance-assessment-system/internal/models"
)

type SysbenchCPUBackend struct {
	timeout time.Duration
	runner  commandRunner
}

type SysbenchCPUConfig struct {
	Timeout time.Duration
	Runner  commandRunner
}

func NewSysbenchCPUBackend(cfg SysbenchCPUConfig) *SysbenchCPUBackend {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 20 * time.Second
	}
	runner := cfg.Runner
	if runner == nil {
		runner = &execCommandRunner{}
	}
	return &SysbenchCPUBackend{
		timeout: timeout,
		runner:  runner,
	}
}

func (b *SysbenchCPUBackend) Name() string {
	return models.CPUBackendSysbench
}

func (b *SysbenchCPUBackend) SingleCoreSource() string {
	return models.CPUSourceSysbench
}

func (b *SysbenchCPUBackend) MultiCoreSource() string {
	return models.CPUSourceSysbench
}

func (b *SysbenchCPUBackend) MeasureSingleCore() (CPUBackendResult, error) {
	return b.runSysbench(1)
}

func (b *SysbenchCPUBackend) MeasureMultiCore() (CPUBackendResult, error) {
	return b.runSysbench(runtime.NumCPU())
}

func (b *SysbenchCPUBackend) runSysbench(threads int) (CPUBackendResult, error) {
	if _, err := b.runner.LookPath("sysbench"); err != nil {
		return CPUBackendResult{}, fmt.Errorf("sysbench is not installed; install it manually before using --cpu-backend sysbench. Ubuntu/Debian: sudo apt install sysbench; RHEL/CentOS: sudo yum install sysbench; macOS: brew install sysbench")
	}

	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	output, err := b.runner.Run(ctx, "sysbench", "cpu", fmt.Sprintf("--threads=%d", threads), "--time=10", "run")
	if err != nil {
		return CPUBackendResult{}, fmt.Errorf("sysbench cpu failed: %w", err)
	}

	eventsPerSecond, err := parseSysbenchEventsPerSecond(output)
	if err != nil {
		return CPUBackendResult{}, fmt.Errorf("parse sysbench cpu result: %w", err)
	}

	score := scoreSysbenchCPU(eventsPerSecond, threads)
	return CPUBackendResult{
		Score:        score,
		EventsPerSec: eventsPerSecond,
	}, nil
}

func scoreSysbenchCPU(eventsPerSecond float64, threads int) float64 {
	if eventsPerSecond <= 0 {
		return 0
	}

	basePerThread := 1200.0
	base := basePerThread * float64(threads)
	score := eventsPerSecond / base * 100
	if score > 100 {
		return 100
	}
	return score
}

var sysbenchEventsRegexp = regexp.MustCompile(`events per second:\s*([0-9]+(?:\.[0-9]+)?)`)

func parseSysbenchEventsPerSecond(output []byte) (float64, error) {
	matches := sysbenchEventsRegexp.FindSubmatch(output)
	if len(matches) != 2 {
		return 0, fmt.Errorf("missing events per second")
	}
	value, err := strconv.ParseFloat(string(matches[1]), 64)
	if err != nil {
		return 0, err
	}
	if value <= 0 {
		return 0, fmt.Errorf("invalid events per second %.2f", value)
	}
	return value, nil
}
