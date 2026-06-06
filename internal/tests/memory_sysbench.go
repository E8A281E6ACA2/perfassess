package tests

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
)

type SysbenchMemoryBackend struct {
	timeout time.Duration
	runner  commandRunner
}

type SysbenchMemoryConfig struct {
	Timeout time.Duration
	Runner  commandRunner
}

func NewSysbenchMemoryBackend(cfg SysbenchMemoryConfig) *SysbenchMemoryBackend {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 20 * time.Second
	}
	runner := cfg.Runner
	if runner == nil {
		runner = &execCommandRunner{}
	}
	return &SysbenchMemoryBackend{
		timeout: timeout,
		runner:  runner,
	}
}

func (b *SysbenchMemoryBackend) Name() string {
	return models.MemoryBackendSysbench
}

func (b *SysbenchMemoryBackend) ReadSource() string {
	return models.MemorySourceSysbench
}

func (b *SysbenchMemoryBackend) WriteSource() string {
	return models.MemorySourceSysbench
}

func (b *SysbenchMemoryBackend) MeasureRead(sizeMB int) (MemoryBackendResult, error) {
	return b.runSysbench(sizeMB, "read")
}

func (b *SysbenchMemoryBackend) MeasureWrite(sizeMB int) (MemoryBackendResult, error) {
	return b.runSysbench(sizeMB, "write")
}

func (b *SysbenchMemoryBackend) runSysbench(sizeMB int, operation string) (MemoryBackendResult, error) {
	if _, err := b.runner.LookPath("sysbench"); err != nil {
		return MemoryBackendResult{}, fmt.Errorf("sysbench is not installed; install it manually before using --memory-backend sysbench. Ubuntu/Debian: sudo apt install sysbench; RHEL/CentOS: sudo yum install sysbench; macOS: brew install sysbench")
	}
	if sizeMB <= 0 {
		sizeMB = 512
	}

	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	output, err := b.runner.Run(
		ctx,
		"sysbench",
		"memory",
		fmt.Sprintf("--memory-oper=%s", operation),
		"--memory-block-size=1M",
		fmt.Sprintf("--memory-total-size=%dM", sizeMB),
		"run",
	)
	if err != nil {
		return MemoryBackendResult{}, fmt.Errorf("sysbench memory %s failed: %w", operation, err)
	}

	speed, err := parseSysbenchMemoryMBps(output)
	if err != nil {
		return MemoryBackendResult{}, fmt.Errorf("parse sysbench memory %s result: %w", operation, err)
	}
	return MemoryBackendResult{SpeedMBps: speed}, nil
}

var sysbenchMemoryTransferredRegexp = regexp.MustCompile(`\(([0-9]+(?:\.[0-9]+)?)\s+MiB/sec\)`)

func parseSysbenchMemoryMBps(output []byte) (float64, error) {
	matches := sysbenchMemoryTransferredRegexp.FindSubmatch(output)
	if len(matches) != 2 {
		return 0, fmt.Errorf("missing transferred MiB/sec")
	}
	value, err := strconv.ParseFloat(string(matches[1]), 64)
	if err != nil {
		return 0, err
	}
	if value <= 0 {
		return 0, fmt.Errorf("invalid memory throughput %.2f", value)
	}
	return value, nil
}
