package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
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

func TestClassifyExternalCommandErrorDetectsTimeout(t *testing.T) {
	err := classifyExternalCommandError("cpu_sysbench_run", nil, context.DeadlineExceeded, "fallback")
	category, stage, hint, ok := benchmarkErrorFields(err)
	if !ok {
		t.Fatalf("expected benchmark error fields, got %T", err)
	}
	if category != BenchmarkErrorTimeout || stage != "cpu_sysbench_run" || !strings.Contains(hint, "超时") {
		t.Fatalf("unexpected timeout classification: %q %q %q", category, stage, hint)
	}
}

func TestSysbenchCPUBackendClassifiesPermissionFailure(t *testing.T) {
	backend := NewSysbenchCPUBackend(SysbenchCPUConfig{
		Runner: &fakeCommandRunner{
			output:     []byte("FATAL: permission denied"),
			err:        fmt.Errorf("exit status 1"),
			lookPathOK: true,
		},
	})

	_, err := backend.MeasureSingleCore()
	if err == nil {
		t.Fatal("expected sysbench command failure")
	}
	category, stage, hint, ok := benchmarkErrorFields(err)
	if !ok {
		t.Fatalf("expected benchmark error fields, got %T", err)
	}
	if category != BenchmarkErrorPermission || stage != "cpu_sysbench_run" || !strings.Contains(hint, "权限") {
		t.Fatalf("unexpected benchmark error fields: %q %q %q", category, stage, hint)
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
	category, stage, hint, ok := benchmarkErrorFields(err)
	if !ok {
		t.Fatalf("expected benchmark error fields, got %T", err)
	}
	if category != BenchmarkErrorMissingDependency || stage != "cpu_sysbench_lookup" || hint == "" {
		t.Fatalf("unexpected benchmark error fields: %q %q %q", category, stage, hint)
	}
}

func TestCPUTestAddsBenchmarkErrorMetrics(t *testing.T) {
	cpuTest := NewCPUTestWithBackend(newTestLogger(t), NewSysbenchCPUBackend(SysbenchCPUConfig{
		Runner: &fakeCommandRunner{lookPathErr: fmt.Errorf("not found")},
	}))

	result, err := cpuTest.Execute()
	if err == nil {
		t.Fatal("expected sysbench execute to fail")
	}
	assertMetricString(t, result.Metrics, "error_category", BenchmarkErrorMissingDependency)
	assertMetricString(t, result.Metrics, "error_stage", "cpu_sysbench_lookup")
	if result.Metrics["error_hint"] == "" {
		t.Fatalf("expected error_hint metric, got %#v", result.Metrics)
	}
}

func TestCPUTestUsesLongerTimeoutForSysbenchBackend(t *testing.T) {
	builtin := NewCPUTest(newTestLogger(t))
	if builtin.GetTimeout() != builtinCPUTestTimeout {
		t.Fatalf("expected builtin CPU timeout %s, got %s", builtinCPUTestTimeout, builtin.GetTimeout())
	}

	sysbench := NewCPUTestWithBackend(newTestLogger(t), NewSysbenchCPUBackend(SysbenchCPUConfig{
		Runner: &fakeCommandRunner{output: []byte("events per second:  1200.00"), lookPathOK: true},
	}))
	if sysbench.GetTimeout() < 2*time.Minute {
		t.Fatalf("expected sysbench CPU timeout to allow multi-sample runs, got %s", sysbench.GetTimeout())
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
