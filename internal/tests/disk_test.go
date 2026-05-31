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
