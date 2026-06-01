package tests

import "testing"

func TestMemoryExecutePopulatesBackendAndSources(t *testing.T) {
	memoryTest := NewMemoryTestWithBackend(newTestLogger(t), &fakeMemoryBackend{
		read:  1200,
		write: 900,
	})
	memoryTest.testSize = 256

	result, err := memoryTest.Execute()
	if err != nil {
		t.Fatalf("expected memory execute to succeed, got error: %v", err)
	}

	assertMetricString(t, result.Metrics, "backend", "fake")
	assertMetricFloat(t, result.Metrics, "read_speed_mbps", 1200)
	assertMetricString(t, result.Metrics, "read_speed_source", "fake_read")
	assertMetricFloat(t, result.Metrics, "write_speed_mbps", 900)
	assertMetricString(t, result.Metrics, "write_speed_source", "fake_write")
	assertMetricInt(t, result.Metrics, "test_size_mb", 256)
}

type fakeMemoryBackend struct {
	read  float64
	write float64
}

func (b *fakeMemoryBackend) Name() string {
	return "fake"
}

func (b *fakeMemoryBackend) ReadSource() string {
	return "fake_read"
}

func (b *fakeMemoryBackend) WriteSource() string {
	return "fake_write"
}

func (b *fakeMemoryBackend) MeasureRead(sizeMB int) (MemoryBackendResult, error) {
	return MemoryBackendResult{SpeedMBps: b.read}, nil
}

func (b *fakeMemoryBackend) MeasureWrite(sizeMB int) (MemoryBackendResult, error) {
	return MemoryBackendResult{SpeedMBps: b.write}, nil
}
