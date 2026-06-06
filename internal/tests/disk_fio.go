package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
)

var fioYABSMixedBlocks = []string{"4k", "64k", "512k", "1m"}

type FioDiskBackend struct {
	workDir       string
	timeout       time.Duration
	runner        commandRunner
	seqReadStats  fioDirectionStats
	seqWriteStats fioDirectionStats
	randomStats   fioRandomStats
	mixedStats    map[string]fioRandomStats
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
	stats, err := parseFioDirectionStats(output, "write")
	if err != nil {
		return 0, err
	}
	b.seqWriteStats = stats
	return stats.BandwidthMBps, nil
}

func (b *FioDiskBackend) MeasureSequentialRead(fileSizeMB int) (float64, error) {
	output, err := b.runFio("perfassess-seqread", "read", "1M", fileSizeMB, 0)
	if err != nil {
		return 0, err
	}
	stats, err := parseFioDirectionStats(output, "read")
	if err != nil {
		return 0, err
	}
	b.seqReadStats = stats
	return stats.BandwidthMBps, nil
}

func (b *FioDiskBackend) MeasureRandomIOPS(durationSec int) (int, error) {
	mixedStats, err := b.runYABSMixedMatrix(durationSec)
	if err != nil {
		return 0, err
	}
	b.mixedStats = mixedStats
	b.randomStats = mixedStats["4k"]
	return int(math.Round(b.randomStats.TotalIOPS())), nil
}

func (b *FioDiskBackend) AppendMetrics(metrics map[string]interface{}) {
	if b.seqReadStats.BandwidthMBps > 0 {
		metrics["sequential_read_iops"] = b.seqReadStats.IOPS
		metrics["sequential_read_latency_ms"] = b.seqReadStats.LatencyMeanMs
		metrics["sequential_read_latency_p95_ms"] = b.seqReadStats.LatencyP95Ms
	}
	if b.seqWriteStats.BandwidthMBps > 0 {
		metrics["sequential_write_iops"] = b.seqWriteStats.IOPS
		metrics["sequential_write_latency_ms"] = b.seqWriteStats.LatencyMeanMs
		metrics["sequential_write_latency_p95_ms"] = b.seqWriteStats.LatencyP95Ms
	}
	if b.randomStats.ReadIOPS > 0 || b.randomStats.WriteIOPS > 0 {
		metrics["random_read_mbps"] = b.randomStats.ReadBandwidthMBps
		metrics["random_write_mbps"] = b.randomStats.WriteBandwidthMBps
		metrics["random_read_iops"] = b.randomStats.ReadIOPS
		metrics["random_write_iops"] = b.randomStats.WriteIOPS
		metrics["random_read_latency_ms"] = b.randomStats.ReadLatencyMeanMs
		metrics["random_write_latency_ms"] = b.randomStats.WriteLatencyMeanMs
		metrics["random_read_latency_p95_ms"] = b.randomStats.ReadLatencyP95Ms
		metrics["random_write_latency_p95_ms"] = b.randomStats.WriteLatencyP95Ms
	}
	if len(b.mixedStats) > 0 {
		metrics["fio_mixed_profile"] = "yabs_randrw_50_50"
		metrics["fio_mixed_block_sizes"] = append([]string(nil), fioYABSMixedBlocks...)
		for _, blockSize := range fioYABSMixedBlocks {
			stats, ok := b.mixedStats[blockSize]
			if !ok || stats.TotalIOPS() <= 0 {
				continue
			}
			prefix := "fio_mixed_" + blockSize
			metrics[prefix+"_read_mbps"] = stats.ReadBandwidthMBps
			metrics[prefix+"_write_mbps"] = stats.WriteBandwidthMBps
			metrics[prefix+"_total_mbps"] = stats.TotalBandwidthMBps()
			metrics[prefix+"_read_iops"] = stats.ReadIOPS
			metrics[prefix+"_write_iops"] = stats.WriteIOPS
			metrics[prefix+"_total_iops"] = stats.TotalIOPS()
		}
	}
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

func (b *FioDiskBackend) runYABSMixedMatrix(durationSec int) (map[string]fioRandomStats, error) {
	if durationSec <= 0 {
		durationSec = 5
	}

	results := make(map[string]fioRandomStats, len(fioYABSMixedBlocks))
	for _, blockSize := range fioYABSMixedBlocks {
		output, err := b.runFioMixed("perfassess-yabs-"+blockSize, blockSize, 128, durationSec)
		if err != nil {
			return nil, err
		}
		stats, err := parseFioRandomStats(output)
		if err != nil {
			return nil, fmt.Errorf("parse fio mixed %s result: %w", blockSize, err)
		}
		results[blockSize] = stats
	}
	return results, nil
}

func (b *FioDiskBackend) runFioMixed(name string, blockSize string, sizeMB int, runtimeSec int) ([]byte, error) {
	if _, err := b.runner.LookPath("fio"); err != nil {
		return nil, fmt.Errorf("fio is not installed; install it manually before using --disk-backend fio. Ubuntu/Debian: sudo apt install fio; RHEL/CentOS: sudo yum install fio; macOS: brew install fio")
	}

	filename := filepath.Join(b.workDir, "perfassess-fio-"+name+".dat")
	args := []string{
		"--name=" + name,
		"--filename=" + filename,
		"--rw=randrw",
		"--rwmixread=50",
		"--bs=" + blockSize,
		fmt.Sprintf("--size=%dM", sizeMB),
		"--direct=1",
		"--numjobs=2",
		"--time_based",
		fmt.Sprintf("--runtime=%d", runtimeSec),
		"--group_reporting",
		"--output-format=json",
	}
	if runtime.GOOS == "linux" {
		args = append(args, "--ioengine=libaio", "--iodepth=64")
	} else {
		args = append(args, "--ioengine=sync")
	}

	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	output, err := b.runner.Run(ctx, "fio", args...)
	if err != nil {
		return nil, fmt.Errorf("fio mixed %s failed: %w", blockSize, err)
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
	BwBytes int64      `json:"bw_bytes"`
	IOPS    float64    `json:"iops"`
	Latency fioLatency `json:"lat_ns"`
	Clat    fioLatency `json:"clat_ns"`
}

type fioLatency struct {
	Mean       float64            `json:"mean"`
	Percentile map[string]float64 `json:"percentile"`
}

type fioDirectionStats struct {
	BandwidthMBps float64
	IOPS          float64
	LatencyMeanMs float64
	LatencyP95Ms  float64
}

type fioRandomStats struct {
	ReadBandwidthMBps  float64
	WriteBandwidthMBps float64
	ReadIOPS           float64
	WriteIOPS          float64
	ReadLatencyMeanMs  float64
	WriteLatencyMeanMs float64
	ReadLatencyP95Ms   float64
	WriteLatencyP95Ms  float64
}

func (s fioRandomStats) TotalBandwidthMBps() float64 {
	return s.ReadBandwidthMBps + s.WriteBandwidthMBps
}

func (s fioRandomStats) TotalIOPS() float64 {
	return s.ReadIOPS + s.WriteIOPS
}

func parseFioBandwidthMBps(output []byte, direction string) (float64, error) {
	stats, err := parseFioDirectionStats(output, direction)
	if err != nil {
		return 0, err
	}
	return stats.BandwidthMBps, nil
}

func parseFioDirectionStats(output []byte, direction string) (fioDirectionStats, error) {
	result, err := parseFioJSON(output)
	if err != nil {
		return fioDirectionStats{}, err
	}

	var totalBytesPerSecond int64
	totalIOPS := 0.0
	latencyMeanWeighted := 0.0
	latencyWeight := 0.0
	p95 := 0.0
	for _, job := range result.Jobs {
		var stats fioJobStats
		switch direction {
		case "read":
			stats = job.Read
		case "write":
			stats = job.Write
		default:
			return fioDirectionStats{}, fmt.Errorf("unsupported fio direction: %s", direction)
		}
		totalBytesPerSecond += stats.BwBytes
		totalIOPS += stats.IOPS
		meanNs := fioMeanLatencyNs(stats)
		if meanNs > 0 && stats.IOPS > 0 {
			latencyMeanWeighted += meanNs * stats.IOPS
			latencyWeight += stats.IOPS
		}
		if percentile := fioP95LatencyNs(stats); percentile > p95 {
			p95 = percentile
		}
	}
	if totalBytesPerSecond <= 0 {
		return fioDirectionStats{}, fmt.Errorf("missing fio %s bw_bytes", direction)
	}
	meanMs := 0.0
	if latencyWeight > 0 {
		meanMs = latencyMeanWeighted / latencyWeight / 1000000.0
	}
	return fioDirectionStats{
		BandwidthMBps: float64(totalBytesPerSecond) / 1024.0 / 1024.0,
		IOPS:          totalIOPS,
		LatencyMeanMs: meanMs,
		LatencyP95Ms:  p95 / 1000000.0,
	}, nil
}

func parseFioRandomIOPS(output []byte) (int, error) {
	stats, err := parseFioRandomStats(output)
	if err != nil {
		return 0, err
	}
	return int(math.Round(stats.ReadIOPS + stats.WriteIOPS)), nil
}

func parseFioRandomStats(output []byte) (fioRandomStats, error) {
	result, err := parseFioJSON(output)
	if err != nil {
		return fioRandomStats{}, err
	}

	stats := fioRandomStats{}
	for _, job := range result.Jobs {
		stats.ReadBandwidthMBps += float64(job.Read.BwBytes) / 1024.0 / 1024.0
		stats.WriteBandwidthMBps += float64(job.Write.BwBytes) / 1024.0 / 1024.0
		stats.ReadIOPS += job.Read.IOPS
		stats.WriteIOPS += job.Write.IOPS
		stats.ReadLatencyMeanMs = maxFloat(stats.ReadLatencyMeanMs, fioMeanLatencyNs(job.Read)/1000000.0)
		stats.WriteLatencyMeanMs = maxFloat(stats.WriteLatencyMeanMs, fioMeanLatencyNs(job.Write)/1000000.0)
		stats.ReadLatencyP95Ms = maxFloat(stats.ReadLatencyP95Ms, fioP95LatencyNs(job.Read)/1000000.0)
		stats.WriteLatencyP95Ms = maxFloat(stats.WriteLatencyP95Ms, fioP95LatencyNs(job.Write)/1000000.0)
	}
	if stats.ReadIOPS+stats.WriteIOPS <= 0 {
		return fioRandomStats{}, fmt.Errorf("missing fio random iops")
	}
	return stats, nil
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

func fioMeanLatencyNs(stats fioJobStats) float64 {
	if stats.Latency.Mean > 0 {
		return stats.Latency.Mean
	}
	return stats.Clat.Mean
}

func fioP95LatencyNs(stats fioJobStats) float64 {
	if value := stats.Latency.Percentile["95.000000"]; value > 0 {
		return value
	}
	if value := stats.Latency.Percentile["95.00"]; value > 0 {
		return value
	}
	if value := stats.Clat.Percentile["95.000000"]; value > 0 {
		return value
	}
	return stats.Clat.Percentile["95.00"]
}

func maxFloat(a float64, b float64) float64 {
	if b > a {
		return b
	}
	return a
}
