package tests

import (
	"fmt"
	"strings"
	"testing"
)

func TestFioDiskBackendReportsMissingBinary(t *testing.T) {
	backend := NewFioDiskBackend(FioConfig{
		Runner: &fakeCommandRunner{
			lookPathErr: fmt.Errorf("not found"),
		},
	})

	_, err := backend.MeasureSequentialRead(100)
	if err == nil {
		t.Fatal("expected missing fio binary to fail")
	}
	if !strings.Contains(err.Error(), "sudo apt install fio") {
		t.Fatalf("expected install hint in error, got %v", err)
	}
}

func TestFioDiskBackendParsesSequentialReadMBps(t *testing.T) {
	runner := &fakeCommandRunner{
		output:     []byte(`{"jobs":[{"read":{"bw_bytes":104857600},"write":{"bw_bytes":0}}]}`),
		lookPathOK: true,
	}
	backend := NewFioDiskBackend(FioConfig{Runner: runner})

	speed, err := backend.MeasureSequentialRead(100)
	if err != nil {
		t.Fatalf("expected fio read result to parse, got error: %v", err)
	}
	if speed != 100.0 {
		t.Fatalf("expected 100 MB/s, got %.2f", speed)
	}
	if runner.name != "fio" {
		t.Fatalf("expected fio command, got %q", runner.name)
	}
	if !containsArgPrefix(runner.args, "--rw=read") {
		t.Fatalf("expected fio read args, got %v", runner.args)
	}
	if !containsArgPrefix(runner.args, "--bs=1M") {
		t.Fatalf("expected fio sequential block size arg, got %v", runner.args)
	}
	if !containsArgPrefix(runner.args, "--output-format=json") {
		t.Fatalf("expected fio json output arg, got %v", runner.args)
	}
}

func TestFioDiskBackendParsesSequentialWriteMBps(t *testing.T) {
	runner := &fakeCommandRunner{
		output:     []byte(`{"jobs":[{"read":{"bw_bytes":0},"write":{"bw_bytes":52428800}}]}`),
		lookPathOK: true,
	}
	backend := NewFioDiskBackend(FioConfig{Runner: runner})

	speed, err := backend.MeasureSequentialWrite(100)
	if err != nil {
		t.Fatalf("expected fio write result to parse, got error: %v", err)
	}
	if speed != 50.0 {
		t.Fatalf("expected 50 MB/s, got %.2f", speed)
	}
	if !containsArgPrefix(runner.args, "--rw=write") {
		t.Fatalf("expected fio write args, got %v", runner.args)
	}
}

func TestFioDiskBackendParsesRandomIOPS(t *testing.T) {
	runner := &fakeCommandRunner{
		output:     []byte(`{"jobs":[{"read":{"iops":1234.4},"write":{"iops":765.6}}]}`),
		lookPathOK: true,
	}
	backend := NewFioDiskBackend(FioConfig{Runner: runner})

	iops, err := backend.MeasureRandomIOPS(5)
	if err != nil {
		t.Fatalf("expected fio random iops result to parse, got error: %v", err)
	}
	if iops != 2000 {
		t.Fatalf("expected 2000 IOPS, got %d", iops)
	}
	if !containsArgPrefix(runner.args, "--rw=randrw") {
		t.Fatalf("expected fio randrw args, got %v", runner.args)
	}
	if !containsArgPrefix(runner.args, "--bs=4k") {
		t.Fatalf("expected fio random block size arg, got %v", runner.args)
	}
	if !containsArgPrefix(runner.args, "--runtime=5") {
		t.Fatalf("expected fio runtime arg, got %v", runner.args)
	}
}

func TestFioParsesDetailedLatencyMetrics(t *testing.T) {
	output := []byte(`{
		"jobs": [{
			"read": {
				"bw_bytes": 104857600,
				"iops": 101.5,
				"clat_ns": {
					"mean": 1200000,
					"percentile": {"95.000000": 2400000}
				}
			},
			"write": {
				"bw_bytes": 52428800,
				"iops": 202.5,
				"clat_ns": {
					"mean": 2200000,
					"percentile": {"95.000000": 4400000}
				}
			}
		}]
	}`)

	readStats, err := parseFioDirectionStats(output, "read")
	if err != nil {
		t.Fatalf("expected read stats to parse, got %v", err)
	}
	if readStats.BandwidthMBps != 100.0 {
		t.Fatalf("expected read bandwidth 100, got %.2f", readStats.BandwidthMBps)
	}
	if readStats.IOPS != 101.5 {
		t.Fatalf("expected read iops 101.5, got %.2f", readStats.IOPS)
	}
	if readStats.LatencyMeanMs != 1.2 {
		t.Fatalf("expected read latency mean 1.2ms, got %.2f", readStats.LatencyMeanMs)
	}
	if readStats.LatencyP95Ms != 2.4 {
		t.Fatalf("expected read p95 2.4ms, got %.2f", readStats.LatencyP95Ms)
	}

	randomStats, err := parseFioRandomStats(output)
	if err != nil {
		t.Fatalf("expected random stats to parse, got %v", err)
	}
	if randomStats.ReadIOPS != 101.5 || randomStats.WriteIOPS != 202.5 {
		t.Fatalf("expected detailed random iops, got read %.2f write %.2f", randomStats.ReadIOPS, randomStats.WriteIOPS)
	}
	if randomStats.WriteLatencyP95Ms != 4.4 {
		t.Fatalf("expected write p95 4.4ms, got %.2f", randomStats.WriteLatencyP95Ms)
	}
}

func TestFioDiskBackendAppendsDetailedMetrics(t *testing.T) {
	backend := &FioDiskBackend{
		seqReadStats: fioDirectionStats{
			BandwidthMBps: 100,
			IOPS:          101.5,
			LatencyMeanMs: 1.2,
			LatencyP95Ms:  2.4,
		},
		seqWriteStats: fioDirectionStats{
			BandwidthMBps: 50,
			IOPS:          50.5,
			LatencyMeanMs: 2.2,
			LatencyP95Ms:  4.4,
		},
		randomStats: fioRandomStats{
			ReadIOPS:           300,
			WriteIOPS:          200,
			ReadLatencyMeanMs:  1.1,
			WriteLatencyMeanMs: 1.5,
			ReadLatencyP95Ms:   2.1,
			WriteLatencyP95Ms:  2.5,
		},
	}

	metrics := map[string]interface{}{}
	backend.AppendMetrics(metrics)

	assertMetricFloat(t, metrics, "sequential_read_iops", 101.5)
	assertMetricFloat(t, metrics, "sequential_write_latency_p95_ms", 4.4)
	assertMetricFloat(t, metrics, "random_read_iops", 300)
	assertMetricFloat(t, metrics, "random_write_latency_ms", 1.5)
}

func TestParseFioRejectsMissingJobs(t *testing.T) {
	if _, err := parseFioBandwidthMBps([]byte(`{"jobs":[]}`), "read"); err == nil {
		t.Fatal("expected missing fio jobs to fail")
	}
}

func TestDiskExecutePopulatesBackendAndSources(t *testing.T) {
	diskTest := NewDiskTestWithBackend(newTestLogger(t), &fakeDiskBackend{
		read:  500,
		write: 300,
		iops:  10000,
	})

	result, err := diskTest.Execute()
	if err != nil {
		t.Fatalf("expected disk execute to succeed, got error: %v", err)
	}

	assertMetricString(t, result.Metrics, "backend", "fake")
	assertMetricFloat(t, result.Metrics, "sequential_read_mbps", 500)
	assertMetricFloat(t, result.Metrics, "sequential_write_mbps", 300)
	assertMetricString(t, result.Metrics, "sequential_read_source", "fake_read")
	assertMetricString(t, result.Metrics, "sequential_write_source", "fake_write")
	assertMetricString(t, result.Metrics, "random_iops_source", "fake_iops")
}

type fakeDiskBackend struct {
	read  float64
	write float64
	iops  int
}

func (b *fakeDiskBackend) Name() string {
	return "fake"
}

func (b *fakeDiskBackend) SequentialReadSource() string {
	return "fake_read"
}

func (b *fakeDiskBackend) SequentialWriteSource() string {
	return "fake_write"
}

func (b *fakeDiskBackend) RandomIOPSSource() string {
	return "fake_iops"
}

func (b *fakeDiskBackend) MeasureSequentialWrite(fileSizeMB int) (float64, error) {
	return b.write, nil
}

func (b *fakeDiskBackend) MeasureSequentialRead(fileSizeMB int) (float64, error) {
	return b.read, nil
}

func (b *fakeDiskBackend) MeasureRandomIOPS(durationSec int) (int, error) {
	return b.iops, nil
}

func (b *fakeDiskBackend) AppendMetrics(metrics map[string]interface{}) {}

func (b *fakeDiskBackend) Cleanup() error {
	return nil
}

func containsArgPrefix(args []string, expected string) bool {
	for _, arg := range args {
		if strings.HasPrefix(arg, expected) {
			return true
		}
	}
	return false
}
