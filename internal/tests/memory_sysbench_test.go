package tests

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseSysbenchMemoryMBps(t *testing.T) {
	output := []byte(`
Total operations: 1024 (2048.00 per second)

1024.00 MiB transferred (2048.00 MiB/sec)
`)

	speed, err := parseSysbenchMemoryMBps(output)
	if err != nil {
		t.Fatalf("expected sysbench memory output to parse, got %v", err)
	}
	if speed != 2048.0 {
		t.Fatalf("expected 2048 MiB/sec, got %.2f", speed)
	}
}

func TestSysbenchMemoryBackendReportsMissingBinary(t *testing.T) {
	backend := NewSysbenchMemoryBackend(SysbenchMemoryConfig{
		Runner: &fakeCommandRunner{
			lookPathErr: fmt.Errorf("not found"),
		},
	})

	_, err := backend.MeasureRead(512)
	if err == nil {
		t.Fatal("expected missing sysbench binary to fail")
	}
	if !strings.Contains(err.Error(), "sudo apt install sysbench") {
		t.Fatalf("expected install hint in error, got %v", err)
	}
}

func TestSysbenchMemoryBackendClassifiesResourceFailure(t *testing.T) {
	backend := NewSysbenchMemoryBackend(SysbenchMemoryConfig{
		Runner: &fakeCommandRunner{
			output:     []byte("FATAL: cannot allocate memory"),
			err:        fmt.Errorf("exit status 1"),
			lookPathOK: true,
		},
	})

	_, err := backend.MeasureRead(512)
	if err == nil {
		t.Fatal("expected sysbench memory command failure")
	}
	category, stage, hint, ok := benchmarkErrorFields(err)
	if !ok {
		t.Fatalf("expected benchmark error fields, got %T", err)
	}
	if category != BenchmarkErrorResource || stage != "memory_sysbench_run" || !strings.Contains(hint, "资源不足") {
		t.Fatalf("unexpected benchmark error fields: %q %q %q", category, stage, hint)
	}
}

func TestSysbenchMemoryBackendParsesReadResult(t *testing.T) {
	runner := &fakeCommandRunner{
		output:     []byte(`1024.00 MiB transferred (1536.50 MiB/sec)`),
		lookPathOK: true,
	}
	backend := NewSysbenchMemoryBackend(SysbenchMemoryConfig{Runner: runner})

	result, err := backend.MeasureRead(256)
	if err != nil {
		t.Fatalf("expected sysbench memory result to parse, got %v", err)
	}
	if result.SpeedMBps != 1536.5 {
		t.Fatalf("expected 1536.5 MB/s, got %.2f", result.SpeedMBps)
	}
	if runner.name != "sysbench" {
		t.Fatalf("expected sysbench command, got %q", runner.name)
	}
	if !containsArgPrefix(runner.args, "--memory-oper=read") {
		t.Fatalf("expected read operation arg, got %v", runner.args)
	}
	if !containsArgPrefix(runner.args, "--memory-total-size=256M") {
		t.Fatalf("expected memory size arg, got %v", runner.args)
	}
}
