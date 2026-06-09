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

func TestBuiltinMemoryBackendReusesAndReleasesBuffer(t *testing.T) {
	memoryTest := NewMemoryTest(newTestLogger(t))
	backend, ok := memoryTest.backend.(*BuiltinMemoryBackend)
	if !ok {
		t.Fatalf("expected builtin memory backend, got %T", memoryTest.backend)
	}

	if err := backend.Prepare(1); err != nil {
		t.Fatalf("expected buffer preparation to succeed, got %v", err)
	}
	firstBuffer := &backend.buffer[0]
	if _, err := backend.MeasureRead(1); err != nil {
		t.Fatalf("expected read sample to succeed, got %v", err)
	}
	if _, err := backend.MeasureWrite(1); err != nil {
		t.Fatalf("expected write sample to succeed, got %v", err)
	}
	if &backend.buffer[0] != firstBuffer {
		t.Fatal("expected builtin memory backend to reuse the prepared buffer")
	}

	backend.Release()
	if backend.buffer != nil {
		t.Fatal("expected builtin memory backend buffer to be released")
	}
}

func TestMemoryBufferSizeRejectsInvalidInput(t *testing.T) {
	if _, err := memoryBufferSize(0); err == nil {
		t.Fatal("expected zero memory buffer size to fail")
	}
	if _, err := memoryBufferSize(-1); err == nil {
		t.Fatal("expected negative memory buffer size to fail")
	}
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
