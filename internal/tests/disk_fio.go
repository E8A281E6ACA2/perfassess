package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"performance-assessment-system/internal/models"
)

type FioDiskBackend struct {
	workDir string
	timeout time.Duration
	runner  commandRunner
}

type FioConfig struct {
	WorkDir string
	Timeout time.Duration
	Runner  commandRunner
}

func NewFioDiskBackend(cfg FioConfig) *FioDiskBackend {
	workDir := cfg.WorkDir
	if workDir == "" {
		workDir = os.TempDir()
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 90 * time.Second
	}
	runner := cfg.Runner
	if runner == nil {
		runner = &execCommandRunner{}
	}

	return &FioDiskBackend{
		workDir: workDir,
		timeout: timeout,
		runner:  runner,
	}
}

func (b *FioDiskBackend) Name() string {
	return models.DiskBackendFio
}

func (b *FioDiskBackend) SequentialReadSource() string {
	return models.DiskSourceFio
}

func (b *FioDiskBackend) SequentialWriteSource() string {
	return models.DiskSourceFio
}

func (b *FioDiskBackend) RandomIOPSSource() string {
	return models.DiskSourceFio
}

func (b *FioDiskBackend) MeasureSequentialWrite(fileSizeMB int) (float64, error) {
	output, err := b.runFio("perfassess-seqwrite", "write", "1M", fileSizeMB, 0)
	if err != nil {
		return 0, err
	}
	return parseFioBandwidthMBps(output, "write")
}

func (b *FioDiskBackend) MeasureSequentialRead(fileSizeMB int) (float64, error) {
	output, err := b.runFio("perfassess-seqread", "read", "1M", fileSizeMB, 0)
	if err != nil {
		return 0, err
	}
	return parseFioBandwidthMBps(output, "read")
}

func (b *FioDiskBackend) MeasureRandomIOPS(durationSec int) (int, error) {
	output, err := b.runFio("perfassess-randrw", "randrw", "4k", 128, durationSec)
	if err != nil {
		return 0, err
	}
	return parseFioRandomIOPS(output)
}

func (b *FioDiskBackend) Cleanup() error {
	pattern := filepath.Join(b.workDir, "perfassess-fio-*")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	for _, file := range files {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (b *FioDiskBackend) runFio(name string, rw string, blockSize string, sizeMB int, runtimeSec int) ([]byte, error) {
	if _, err := b.runner.LookPath("fio"); err != nil {
		return nil, fmt.Errorf("fio is not installed; install it manually before using --disk-backend fio. Ubuntu/Debian: sudo apt install fio; RHEL/CentOS: sudo yum install fio; macOS: brew install fio")
	}

	filename := filepath.Join(b.workDir, "perfassess-fio-"+name+".dat")
	args := []string{
		"--name=" + name,
		"--filename=" + filename,
		"--rw=" + rw,
		"--bs=" + blockSize,
		fmt.Sprintf("--size=%dM", sizeMB),
		"--ioengine=sync",
		"--direct=1",
		"--numjobs=1",
		"--group_reporting",
		"--output-format=json",
	}
	if runtimeSec > 0 {
		args = append(args, "--time_based", fmt.Sprintf("--runtime=%d", runtimeSec))
	}

	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	output, err := b.runner.Run(ctx, "fio", args...)
	if err != nil {
		return nil, fmt.Errorf("fio %s failed: %w", rw, err)
	}
	return output, nil
}

type fioJSONResult struct {
	Jobs []struct {
		Read  fioJobStats `json:"read"`
		Write fioJobStats `json:"write"`
	} `json:"jobs"`
}

type fioJobStats struct {
	BwBytes int64   `json:"bw_bytes"`
	IOPS    float64 `json:"iops"`
}

func parseFioBandwidthMBps(output []byte, direction string) (float64, error) {
	result, err := parseFioJSON(output)
	if err != nil {
		return 0, err
	}

	var totalBytesPerSecond int64
	for _, job := range result.Jobs {
		switch direction {
		case "read":
			totalBytesPerSecond += job.Read.BwBytes
		case "write":
			totalBytesPerSecond += job.Write.BwBytes
		default:
			return 0, fmt.Errorf("unsupported fio direction: %s", direction)
		}
	}
	if totalBytesPerSecond <= 0 {
		return 0, fmt.Errorf("missing fio %s bw_bytes", direction)
	}
	return float64(totalBytesPerSecond) / 1024.0 / 1024.0, nil
}

func parseFioRandomIOPS(output []byte) (int, error) {
	result, err := parseFioJSON(output)
	if err != nil {
		return 0, err
	}

	totalIOPS := 0.0
	for _, job := range result.Jobs {
		totalIOPS += job.Read.IOPS + job.Write.IOPS
	}
	if totalIOPS <= 0 {
		return 0, fmt.Errorf("missing fio random iops")
	}
	return int(math.Round(totalIOPS)), nil
}

func parseFioJSON(output []byte) (*fioJSONResult, error) {
	var result fioJSONResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}
	if len(result.Jobs) == 0 {
		return nil, fmt.Errorf("missing fio jobs")
	}
	return &result, nil
}
