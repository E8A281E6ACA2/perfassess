package tests

import (
	"testing"
	"time"
)

func TestCreateResultPopulatesEndTimeBeforeDeferredMarkEnd(t *testing.T) {
	test := NewBaseTest("base", time.Second, nil)
	test.MarkStart()

	result := test.CreateResult("success", nil, "")

	if result.EndTime.IsZero() {
		t.Fatal("expected result end time to be populated")
	}
	if result.DurationSeconds <= 0 {
		t.Fatalf("expected positive duration, got %.6f", result.DurationSeconds)
	}
	if result.EndTime.Before(result.StartTime) {
		t.Fatalf("expected end time after start time, got start=%s end=%s", result.StartTime, result.EndTime)
	}
}

func TestCreateResultPopulatesStartTimeWithoutMarkStart(t *testing.T) {
	test := NewBaseTest("base", time.Second, nil)

	result := test.CreateResult("failed", nil, "boom")

	if result.StartTime.IsZero() {
		t.Fatal("expected result start time to be populated")
	}
	if result.EndTime.IsZero() {
		t.Fatal("expected result end time to be populated")
	}
	if result.DurationSeconds < 0 {
		t.Fatalf("expected non-negative duration, got %.6f", result.DurationSeconds)
	}
}
