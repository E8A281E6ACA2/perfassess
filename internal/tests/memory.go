// Package tests 提供内存性能测试功能
package tests

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/mem"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
	"github.com/E8A281E6ACA2/perfassess/pkg/logger"
	"github.com/E8A281E6ACA2/perfassess/pkg/utils"
)

const memorySampleRuns = 3

// MemoryTest 内存性能测试
// 测试内存的读写速度
type MemoryTest struct {
	*BaseTest
	testSize int // 测试数据大小（MB）
	backend  MemoryBenchmarkBackend
}

type MemoryBenchmarkBackend interface {
	Name() string
	ReadSource() string
	WriteSource() string
	MeasureRead(sizeMB int) (MemoryBackendResult, error)
	MeasureWrite(sizeMB int) (MemoryBackendResult, error)
}

type memoryBenchmarkPreparer interface {
	Prepare(sizeMB int) error
}

type memoryBenchmarkReleaser interface {
	Release()
}

type MemoryBackendResult struct {
	SpeedMBps float64
}

type BuiltinMemoryBackend struct {
	test   *MemoryTest
	buffer []byte
}

func (b *BuiltinMemoryBackend) Name() string {
	return models.MemoryBackendBuiltin
}

func (b *BuiltinMemoryBackend) ReadSource() string {
	return models.MemorySourceBuiltin
}

func (b *BuiltinMemoryBackend) WriteSource() string {
	return models.MemorySourceBuiltin
}

func (b *BuiltinMemoryBackend) MeasureRead(sizeMB int) (MemoryBackendResult, error) {
	if err := b.Prepare(sizeMB); err != nil {
		return MemoryBackendResult{}, err
	}
	speed, err := b.test.TestSequentialReadWithBuffer(sizeMB, b.buffer)
	return MemoryBackendResult{SpeedMBps: speed}, err
}

func (b *BuiltinMemoryBackend) MeasureWrite(sizeMB int) (MemoryBackendResult, error) {
	if err := b.Prepare(sizeMB); err != nil {
		return MemoryBackendResult{}, err
	}
	speed, err := b.test.TestSequentialWriteWithBuffer(sizeMB, b.buffer)
	return MemoryBackendResult{SpeedMBps: speed}, err
}

func (b *BuiltinMemoryBackend) Prepare(sizeMB int) error {
	size, err := memoryBufferSize(sizeMB)
	if err != nil {
		return err
	}
	if cap(b.buffer) < size {
		b.buffer = make([]byte, size)
	} else {
		b.buffer = b.buffer[:size]
	}
	return nil
}

func (b *BuiltinMemoryBackend) Release() {
	b.buffer = nil
}

// NewMemoryTest 创建内存性能测试
// 参数:
//   - logger: 日志记录器
//
// 返回:
//   - *MemoryTest: 内存测试实例
func NewMemoryTest(logger *logger.Logger) *MemoryTest {
	return NewMemoryTestWithBackend(logger, nil)
}

func NewMemoryTestWithBackend(logger *logger.Logger, backend MemoryBenchmarkBackend) *MemoryTest {
	test := &MemoryTest{
		BaseTest: NewBaseTest("内存性能测试", 60*time.Second, logger),
		testSize: 0, // 将在Setup中计算
	}
	if backend != nil {
		test.backend = backend
	} else {
		test.backend = &BuiltinMemoryBackend{test: test}
	}
	return test
}

// Setup 测试前的准备工作
// 检查可用内存并计算安全的测试大小
// 返回:
//   - error: 准备错误
func (mt *MemoryTest) Setup() error {
	// 获取安全的测试大小
	testSize, err := mt.GetSafeTestSize()
	if err != nil {
		return err
	}

	mt.testSize = testSize
	mt.GetLogger().Info(fmt.Sprintf("内存测试大小: %d MB", testSize))

	return nil
}

// Execute 执行内存性能测试
// 返回:
//   - *models.TestResult: 测试结果
//   - error: 测试错误
func (mt *MemoryTest) Execute() (*models.TestResult, error) {
	mt.MarkStart()
	status := "success"
	defer func() {
		mt.releaseBackendMemory()
		runtime.GC()
		mt.MarkEnd(status)
	}()

	metrics := make(map[string]interface{})
	metrics["backend"] = mt.backend.Name()
	if err := mt.prepareBackendMemory(); err != nil {
		status = "failed"
		addBenchmarkErrorMetrics(metrics, err)
		return mt.CreateResult("failed", metrics, fmt.Sprintf("准备内存测试失败: %v", err)), err
	}

	// 测试顺序读取速度
	mt.GetLogger().Info("开始内存顺序读取测试...")
	readSamples, err := mt.collectSamples(memorySampleRuns, func() (MemoryBackendResult, error) {
		return mt.backend.MeasureRead(mt.testSize)
	})
	if err != nil {
		status = "failed"
		addBenchmarkErrorMetrics(metrics, err)
		return mt.CreateResult("failed", metrics, fmt.Sprintf("读取测试失败: %v", err)), err
	}
	readStats := calculateMemorySpeedStats(readSamples)
	readSpeed := readStats.Median
	metrics["read_speed_mbps"] = readSpeed
	metrics["read_speed_source"] = mt.backend.ReadSource()
	addSampleStatsMetrics(metrics, "read_speed_mbps", readStats)
	mt.GetLogger().Info(fmt.Sprintf("读取速度: %.2f MB/s", readSpeed))

	runtime.GC()

	// 测试顺序写入速度
	mt.GetLogger().Info("开始内存顺序写入测试...")
	writeSamples, err := mt.collectSamples(memorySampleRuns, func() (MemoryBackendResult, error) {
		return mt.backend.MeasureWrite(mt.testSize)
	})
	if err != nil {
		status = "failed"
		addBenchmarkErrorMetrics(metrics, err)
		return mt.CreateResult("failed", metrics, fmt.Sprintf("写入测试失败: %v", err)), err
	}
	writeStats := calculateMemorySpeedStats(writeSamples)
	writeSpeed := writeStats.Median
	metrics["write_speed_mbps"] = writeSpeed
	metrics["write_speed_source"] = mt.backend.WriteSource()
	addSampleStatsMetrics(metrics, "write_speed_mbps", writeStats)
	mt.GetLogger().Info(fmt.Sprintf("写入速度: %.2f MB/s", writeSpeed))

	// 计算总体评分
	score := mt.calculateScore(readSpeed, writeSpeed)
	metrics["score"] = score
	metrics["test_size_mb"] = mt.testSize

	mt.GetLogger().Info(fmt.Sprintf("内存测试完成，评分: %.2f", score))

	return mt.CreateResult("success", metrics, ""), nil
}

func (mt *MemoryTest) prepareBackendMemory() error {
	if preparer, ok := mt.backend.(memoryBenchmarkPreparer); ok {
		return preparer.Prepare(mt.testSize)
	}
	return nil
}

func (mt *MemoryTest) releaseBackendMemory() {
	if releaser, ok := mt.backend.(memoryBenchmarkReleaser); ok {
		releaser.Release()
	}
}

func (mt *MemoryTest) collectSamples(runs int, measure func() (MemoryBackendResult, error)) ([]MemoryBackendResult, error) {
	samples := make([]MemoryBackendResult, 0, runs)
	for i := 0; i < runs; i++ {
		result, err := measure()
		if err != nil {
			return nil, err
		}
		samples = append(samples, result)
	}
	return samples, nil
}

func calculateMemorySpeedStats(samples []MemoryBackendResult) sampleStats {
	speeds := make([]float64, 0, len(samples))
	for _, sample := range samples {
		speeds = append(speeds, sample.SpeedMBps)
	}
	return calculateSampleStats(speeds)
}

// GetSafeTestSize 获取安全的测试大小
// 限制为可用内存的保守比例（默认15%，上限256MB），最小128MB
// 返回:
//   - int: 测试大小（MB）
//   - error: 获取错误
func (mt *MemoryTest) GetSafeTestSize() (int, error) {
	// 获取内存信息
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return 0, fmt.Errorf("获取内存信息失败: %w", err)
	}

	// 转换为MB
	availableMB := int(vmStat.Available / 1024 / 1024)

	// 如果处于容器环境，使用容器限制来估算可用内存
	if limitMB := getContainerMemoryLimitMB(); limitMB > 0 && limitMB < availableMB {
		availableMB = limitMB
	}

	// 检查最小内存要求（256MB）
	if availableMB < 256 {
		return 0, utils.WrapError(
			utils.ErrInsufficientMemory,
			fmt.Sprintf("可用内存不足: %d MB < 256 MB", availableMB),
		)
	}

	// 为系统其他进程和 Go runtime 保留一定的缓冲区
	reserveMB := int(float64(availableMB) * 0.2)
	if reserveMB < 256 {
		reserveMB = 256
	}
	if availableMB-reserveMB < 256 {
		return 0, utils.WrapError(
			utils.ErrInsufficientMemory,
			fmt.Sprintf("当前可用于测试的内存不足: 仅 %d MB", availableMB-reserveMB),
		)
	}

	// 使用保守比例避免默认一把梭在小内存 VPS 或并发场景下触发 OOM。
	testSize := int(float64(availableMB) * 0.15)
	if testSize > 256 {
		testSize = 256
	}
	if maxUsable := availableMB - reserveMB; testSize > maxUsable {
		testSize = maxUsable
	}

	// 至少使用128MB进行测试。
	if testSize < 128 {
		testSize = 128
	}

	return testSize, nil
}

// TestSequentialRead 测试顺序读取速度
// 参数:
//   - sizeMB: 测试数据大小（MB）
//
// 返回:
//   - float64: 读取速度（MB/s）
//   - error: 测试错误
func (mt *MemoryTest) TestSequentialRead(sizeMB int) (float64, error) {
	size, err := memoryBufferSize(sizeMB)
	if err != nil {
		return 0, err
	}
	return mt.TestSequentialReadWithBuffer(sizeMB, make([]byte, size))
}

func (mt *MemoryTest) TestSequentialReadWithBuffer(sizeMB int, data []byte) (float64, error) {
	size, err := memoryBufferSize(sizeMB)
	if err != nil {
		return 0, err
	}
	if len(data) < size {
		return 0, fmt.Errorf("内存测试缓冲区不足: %d < %d bytes", len(data), size)
	}
	data = data[:size]

	// 先写入数据
	for i := 0; i < size; i++ {
		data[i] = byte(i % 256)
	}

	// 测试读取速度
	startTime := time.Now()
	var sum int64

	// 顺序读取
	for i := 0; i < size; i++ {
		sum += int64(data[i])
	}

	duration := time.Since(startTime)

	// 计算速度（MB/s）
	speed := float64(sizeMB) / duration.Seconds()

	// 防止编译器优化掉sum变量
	if sum < 0 {
		return 0, fmt.Errorf("读取测试异常")
	}

	return speed, nil
}

// TestSequentialWrite 测试顺序写入速度
// 参数:
//   - sizeMB: 测试数据大小（MB）
//
// 返回:
//   - float64: 写入速度（MB/s）
//   - error: 测试错误
func (mt *MemoryTest) TestSequentialWrite(sizeMB int) (float64, error) {
	size, err := memoryBufferSize(sizeMB)
	if err != nil {
		return 0, err
	}
	return mt.TestSequentialWriteWithBuffer(sizeMB, make([]byte, size))
}

func (mt *MemoryTest) TestSequentialWriteWithBuffer(sizeMB int, data []byte) (float64, error) {
	size, err := memoryBufferSize(sizeMB)
	if err != nil {
		return 0, err
	}
	if len(data) < size {
		return 0, fmt.Errorf("内存测试缓冲区不足: %d < %d bytes", len(data), size)
	}
	data = data[:size]

	// 测试写入速度
	startTime := time.Now()

	// 顺序写入
	for i := 0; i < size; i++ {
		data[i] = byte(i % 256)
	}

	duration := time.Since(startTime)

	// 计算速度（MB/s）
	speed := float64(sizeMB) / duration.Seconds()

	return speed, nil
}

func memoryBufferSize(sizeMB int) (int, error) {
	if sizeMB <= 0 {
		return 0, fmt.Errorf("内存测试大小必须大于 0 MB，当前为 %d MB", sizeMB)
	}
	const bytesPerMB = 1024 * 1024
	if sizeMB > int(^uint(0)>>1)/bytesPerMB {
		return 0, fmt.Errorf("内存测试大小过大: %d MB", sizeMB)
	}
	return sizeMB * bytesPerMB, nil
}

// calculateScore 计算内存性能评分
// 参数:
//   - readSpeed: 读取速度（MB/s）
//   - writeSpeed: 写入速度（MB/s）
//
// 返回:
//   - float64: 评分（0-100）
func (mt *MemoryTest) calculateScore(readSpeed, writeSpeed float64) float64 {
	// 基准速度（MB/s）
	// 现代DDR4内存理论速度约为20000-30000 MB/s
	// 但实际测试中由于各种开销，通常在5000-15000 MB/s
	const baseReadSpeed = 10000.0
	const baseWriteSpeed = 8000.0

	// 计算读写评分
	readScore := (readSpeed / baseReadSpeed) * 100
	writeScore := (writeSpeed / baseWriteSpeed) * 100

	// 限制评分范围
	if readScore > 100 {
		readScore = 100
	}
	if writeScore > 100 {
		writeScore = 100
	}

	// 读写各占50%
	totalScore := (readScore + writeScore) / 2

	return totalScore
}

// getContainerMemoryLimitMB 读取容器内存限制（如果存在）
func getContainerMemoryLimitMB() int {
	cgroupPaths := []string{
		"/sys/fs/cgroup/memory.max",                   // cgroup v2
		"/sys/fs/cgroup/memory/memory.limit_in_bytes", // cgroup v1
	}

	for _, path := range cgroupPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		value := strings.TrimSpace(string(data))
		if value == "" || value == "max" {
			continue
		}

		limitBytes, err := strconv.ParseUint(value, 10, 64)
		if err != nil || limitBytes == 0 {
			continue
		}

		limitMB := int(limitBytes / 1024 / 1024)
		if limitMB > 0 {
			return limitMB
		}
	}

	return 0
}
