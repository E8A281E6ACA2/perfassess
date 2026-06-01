package tests

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseSysbenchEventsPerSecond(t *testing.T) {
	output := []byte(`
CPU speed:
    events per second:  2345.67
`)

	value, err := parseSysbenchEventsPerSecond(output)
	if err != nil {
		t.Fatalf("expected sysbench output to parse, got %v", err)
	}
	if value != 2345.67 {
		t.Fatalf("expected events per second 2345.67, got %.2f", value)
	}
}

func TestParseSysbenchEventsPerSecondRejectsMissingValue(t *testing.T) {
	if _, err := parseSysbenchEventsPerSecond([]byte("no events here")); err == nil {
		t.Fatal("expected missing events per second to fail")
	}
}

func TestSysbenchCPUBackendReportsMissingBinary(t *testing.T) {
	backend := NewSysbenchCPUBackend(SysbenchCPUConfig{
		Runner: &fakeCommandRunner{
			lookPathErr: fmt.Errorf("not found"),
		},
	})

	_, err := backend.MeasureSingleCore()
	if err == nil {
		t.Fatal("expected missing sysbench binary to fail")
	}
	if !strings.Contains(err.Error(), "sudo apt install sysbench") {
		t.Fatalf("expected install hint in error, got %v", err)
	}
}

func TestSysbenchCPUBackendParsesSingleCoreResult(t *testing.T) {
	runner := &fakeCommandRunner{
		output:     []byte("events per second:  1200.00"),
		lookPathOK: true,
	}
	backend := NewSysbenchCPUBackend(SysbenchCPUConfig{Runner: runner})

	result, err := backend.MeasureSingleCore()
	if err != nil {
		t.Fatalf("expected sysbench result to parse, got %v", err)
	}
	if result.EventsPerSec != 1200.0 {
		t.Fatalf("expected events per second 1200, got %.2f", result.EventsPerSec)
	}
	if result.Score != 100.0 {
		t.Fatalf("expected score 100, got %.2f", result.Score)
	}
	if runner.name != "sysbench" {
		t.Fatalf("expected sysbench command, got %q", runner.name)
	}
	if !containsArgPrefix(runner.args, "cpu") {
		t.Fatalf("expected sysbench cpu args, got %v", runner.args)
	}
	if !containsArgPrefix(runner.args, "--threads=1") {
		t.Fatalf("expected single-thread sysbench args, got %v", runner.args)
	}
}
