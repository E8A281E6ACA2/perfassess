package tests

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestParseGeekbenchCPUScoresTopLevel(t *testing.T) {
	content := []byte(`{
		"single_core_score": 2450,
		"multi_core_score": 9800
	}`)

	score, err := parseGeekbenchCPUScores(content)
	if err != nil {
		t.Fatalf("expected geekbench scores to parse, got %v", err)
	}
	if score.SingleCoreScore != 2450 {
		t.Fatalf("expected single score 2450, got %.2f", score.SingleCoreScore)
	}
	if score.MultiCoreScore != 9800 {
		t.Fatalf("expected multi score 9800, got %.2f", score.MultiCoreScore)
	}
}

func TestParseGeekbenchCPUScoresNested(t *testing.T) {
	content := []byte(`{
		"results": {
			"single_core": {"score": 2501},
			"multi_core": {"score": 12005}
		}
	}`)

	score, err := parseGeekbenchCPUScores(content)
	if err != nil {
		t.Fatalf("expected nested geekbench scores to parse, got %v", err)
	}
	if score.SingleCoreScore != 2501 {
		t.Fatalf("expected single score 2501, got %.2f", score.SingleCoreScore)
	}
	if score.MultiCoreScore != 12005 {
		t.Fatalf("expected multi score 12005, got %.2f", score.MultiCoreScore)
	}
}

func TestParseGeekbenchCPUScoresRejectsMissingValue(t *testing.T) {
	if _, err := parseGeekbenchCPUScores([]byte(`{"score": 100}`)); err == nil {
		t.Fatal("expected missing geekbench scores to fail")
	}
}

func TestGeekbenchCPUBackendReportsMissingBinary(t *testing.T) {
	backend := NewGeekbenchCPUBackend(GeekbenchCPUConfig{
		Runner: &fakeCommandRunner{
			lookPathErr: fmt.Errorf("not found"),
		},
	})

	_, err := backend.MeasureSingleCore()
	if err == nil {
		t.Fatal("expected missing geekbench6 binary to fail")
	}
	if !strings.Contains(err.Error(), "geekbench6 is not installed") {
		t.Fatalf("expected install hint in error, got %v", err)
	}
}

func TestGeekbenchCPUBackendRunsOnceAndCachesResult(t *testing.T) {
	runner := &geekbenchFakeRunner{
		content: []byte(`{
			"single_core_score": 2500,
			"multi_core_score": 10000
		}`),
	}
	backend := NewGeekbenchCPUBackend(GeekbenchCPUConfig{Runner: runner})

	single, err := backend.MeasureSingleCore()
	if err != nil {
		t.Fatalf("expected single core geekbench result, got %v", err)
	}
	multi, err := backend.MeasureMultiCore()
	if err != nil {
		t.Fatalf("expected multi core geekbench result, got %v", err)
	}

	if single.RawScore != 2500 {
		t.Fatalf("expected raw single score 2500, got %.2f", single.RawScore)
	}
	if single.Score != 100 {
		t.Fatalf("expected normalized single score 100, got %.2f", single.Score)
	}
	if multi.RawScore != 10000 {
		t.Fatalf("expected raw multi score 10000, got %.2f", multi.RawScore)
	}
	if runner.calls != 1 {
		t.Fatalf("expected geekbench command to run once, got %d", runner.calls)
	}
	if runner.name != geekbenchBinary {
		t.Fatalf("expected geekbench6 command, got %q", runner.name)
	}
	if !containsArgPrefix(runner.args, "--no-upload") {
		t.Fatalf("expected --no-upload arg, got %v", runner.args)
	}
	if !containsArgPrefix(runner.args, "--export-json") {
		t.Fatalf("expected --export-json arg, got %v", runner.args)
	}
}

func TestGeekbenchCPUBackendNamesSources(t *testing.T) {
	backend := NewGeekbenchCPUBackend(GeekbenchCPUConfig{Runner: &fakeCommandRunner{lookPathOK: true}})

	if backend.Name() != "geekbench" {
		t.Fatalf("expected backend geekbench, got %q", backend.Name())
	}
	if backend.SingleCoreSource() != "geekbench6" {
		t.Fatalf("expected single core source geekbench6, got %q", backend.SingleCoreSource())
	}
	if backend.MultiCoreSource() != "geekbench6" {
		t.Fatalf("expected multi core source geekbench6, got %q", backend.MultiCoreSource())
	}
}

type geekbenchFakeRunner struct {
	content []byte
	name    string
	args    []string
	calls   int
}

func (r *geekbenchFakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.name = name
	r.args = append([]string(nil), args...)
	r.calls++
	for i, arg := range args {
		if arg == "--export-json" && i+1 < len(args) {
			return nil, os.WriteFile(args[i+1], r.content, 0600)
		}
	}
	return nil, fmt.Errorf("missing --export-json path")
}

func (r *geekbenchFakeRunner) LookPath(name string) (string, error) {
	return "/usr/bin/" + name, nil
}
