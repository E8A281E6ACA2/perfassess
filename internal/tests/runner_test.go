package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
)

func TestRunSingleTestFailureHasTimestamps(t *testing.T) {
	runner := NewTestRunner(newTestLogger(t))
	result, err := runner.RunSingleTest(&failingPerformanceTest{
		name: "内存性能测试",
		err:  fmt.Errorf("missing dependency"),
	})
	if err == nil {
		t.Fatal("expected failing test to return error")
	}
	if result == nil {
		t.Fatal("expected failed result")
	}
	if result.StartTime.IsZero() {
		t.Fatal("expected start time to be populated")
	}
	if result.EndTime.IsZero() {
		t.Fatal("expected end time to be populated")
	}
}

func TestRunSingleTestPreservesPartialFailureResult(t *testing.T) {
	runner := NewTestRunner(newTestLogger(t))
	result, err := runner.RunSingleTest(&partialFailingPerformanceTest{
		name: "内存性能测试",
		err:  fmt.Errorf("missing dependency"),
	})
	if err == nil {
		t.Fatal("expected failing test to return error")
	}
	if result == nil {
		t.Fatal("expected failed result")
	}
	assertMetricString(t, result.Metrics, "backend", "sysbench")
}

func TestRunTestsPreservesFatalFailureResult(t *testing.T) {
	runner := NewTestRunner(newTestLogger(t))
	results, err := runner.RunTests([]PerformanceTest{
		&failingPerformanceTest{
			name: "内存性能测试",
			err:  fmt.Errorf("out of memory"),
		},
	})
	if err == nil {
		t.Fatal("expected fatal test failure to return error")
	}
	if results == nil {
		t.Fatal("expected partial test results")
	}
	if results.MemoryResult == nil {
		t.Fatal("expected failed memory result to be preserved")
	}
	if results.MemoryResult.Status != "failed" {
		t.Fatalf("expected failed status, got %q", results.MemoryResult.Status)
	}
	if results.MemoryResult.ErrorMessage == "" {
		t.Fatal("expected error message to be preserved")
	}
}

type failingPerformanceTest struct {
	name string
	err  error
}

type partialFailingPerformanceTest struct {
	name string
	err  error
}

func (t *partialFailingPerformanceTest) Setup() error {
	return nil
}

func (t *partialFailingPerformanceTest) Execute() (*models.TestResult, error) {
	now := time.Now()
	return &models.TestResult{
		TestName:        t.name,
		Status:          "failed",
		StartTime:       now,
		EndTime:         now,
		DurationSeconds: 0,
		Metrics: map[string]interface{}{
			"backend": "sysbench",
		},
		ErrorMessage: t.err.Error(),
	}, t.err
}

func (t *partialFailingPerformanceTest) Teardown() error {
	return nil
}

func (t *partialFailingPerformanceTest) GetTimeout() time.Duration {
	return time.Second
}

func (t *partialFailingPerformanceTest) GetName() string {
	return t.name
}

func (t *failingPerformanceTest) Setup() error {
	return nil
}

func (t *failingPerformanceTest) Execute() (*models.TestResult, error) {
	return nil, t.err
}

func (t *failingPerformanceTest) Teardown() error {
	return nil
}

func (t *failingPerformanceTest) GetTimeout() time.Duration {
	return time.Second
}

func (t *failingPerformanceTest) GetName() string {
	return t.name
}
