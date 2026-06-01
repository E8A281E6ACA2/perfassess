// Package tests 提供磁盘性能测试功能
package tests

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/disk"

	"performance-assessment-system/internal/models"
	"performance-assessment-system/pkg/logger"
	"performance-assessment-system/pkg/utils"
)

// DiskTest 磁盘性能测试
// 测试磁盘的读写速度和IOPS
type DiskTest struct {
	*BaseTest
	testFilePath string // 测试文件路径
	testDir      string // 测试目录
	backend      DiskBenchmarkBackend
}

type DiskBenchmarkBackend interface {
	Name() string
	SequentialReadSource() string
	SequentialWriteSource() string
	RandomIOPSSource() string
	MeasureSequentialWrite(fileSizeMB int) (float64, error)
	MeasureSequentialRead(fileSizeMB int) (float64, error)
	MeasureRandomIOPS(durationSec int) (int, error)
	AppendMetrics(metrics map[string]interface{})
	Cleanup() error
}

type BuiltinDiskBackend struct {
	test *DiskTest
}

func (b *BuiltinDiskBackend) Name() string {
	return models.DiskBackendBuiltin
}

func (b *BuiltinDiskBackend) SequentialReadSource() string {
	return models.DiskSourceBuiltinSequential
}

func (b *BuiltinDiskBackend) SequentialWriteSource() string {
	return models.DiskSourceBuiltinSequential
}

func (b *BuiltinDiskBackend) RandomIOPSSource() string {
	return models.DiskSourceBuiltinRandom
}

func (b *BuiltinDiskBackend) MeasureSequentialWrite(fileSizeMB int) (float64, error) {
	return b.test.TestSequentialWrite(fileSizeMB)
}

func (b *BuiltinDiskBackend) MeasureSequentialRead(fileSizeMB int) (float64, error) {
	return b.test.TestSequentialRead(fileSizeMB)
}

func (b *BuiltinDiskBackend) MeasureRandomIOPS(durationSec int) (int, error) {
	return b.test.TestRandomIOPS(durationSec)
}

func (b *BuiltinDiskBackend) AppendMetrics(metrics map[string]interface{}) {}

func (b *BuiltinDiskBackend) Cleanup() error {
	return b.test.CleanupTestFiles()
}

// NewDiskTest 创建磁盘性能测试
// 参数:
//   - logger: 日志记录器
//
// 返回:
//   - *DiskTest: 磁盘测试实例
func NewDiskTest(logger *logger.Logger) *DiskTest {
	return NewDiskTestWithBackend(logger, nil)
}

func NewDiskTestWithBackend(logger *logger.Logger, backend DiskBenchmarkBackend) *DiskTest {
	// 根据操作系统选择测试目录
	testDir := "/tmp"
	if runtime.GOOS == "windows" {
		testDir = os.TempDir()
	}

	test := &DiskTest{
		BaseTest:     NewBaseTest("磁盘性能测试", 60*time.Second, logger),
		testDir:      testDir,
		testFilePath: filepath.Join(testDir, "perf_test_disk.dat"),
	}
	if backend != nil {
		test.backend = backend
	} else {
		test.backend = &BuiltinDiskBackend{test: test}
	}

	return test
}

// Setup 测试前的准备工作
// 检查磁盘空间是否足够
// 返回:
//   - error: 准备错误
func (dt *DiskTest) Setup() error {
	// 检查磁盘空间
	usage, err := disk.Usage(dt.testDir)
	if err != nil {
		return fmt.Errorf("获取磁盘信息失败: %w", err)
	}

	// 转换为GB
	freeGB := float64(usage.Free) / 1024 / 1024 / 1024

	// 检查最小空间要求（1GB）
	if freeGB < 1.0 {
		return utils.WrapError(
			utils.ErrInsufficientDiskSpace,
			fmt.Sprintf("磁盘可用空间不足: %.2f GB < 1 GB", freeGB),
		)
	}

	dt.GetLogger().Info(fmt.Sprintf("磁盘可用空间: %.2f GB", freeGB))

	return nil
}

// Execute 执行磁盘性能测试
// 返回:
//   - *models.TestResult: 测试结果
//   - error: 测试错误
func (dt *DiskTest) Execute() (*models.TestResult, error) {
	dt.MarkStart()
	defer func() {
		dt.MarkEnd("success")
	}()

	metrics := make(map[string]interface{})
	metrics["backend"] = dt.backend.Name()

	// 测试顺序写入速度
	dt.GetLogger().Info("开始磁盘顺序写入测试...")
	writeSpeed, err := dt.backend.MeasureSequentialWrite(100) // 100MB
	if err != nil {
		return dt.CreateResult("failed", nil, fmt.Sprintf("写入测试失败: %v", err)), err
	}
	metrics["write_speed_mbps"] = writeSpeed
	metrics["sequential_write_mbps"] = writeSpeed
	metrics["sequential_write_source"] = dt.backend.SequentialWriteSource()
	dt.GetLogger().Info(fmt.Sprintf("写入速度: %.2f MB/s", writeSpeed))

	// 测试顺序读取速度
	dt.GetLogger().Info("开始磁盘顺序读取测试...")
	readSpeed, err := dt.backend.MeasureSequentialRead(100) // 100MB
	if err != nil {
		return dt.CreateResult("failed", nil, fmt.Sprintf("读取测试失败: %v", err)), err
	}
	metrics["read_speed_mbps"] = readSpeed
	metrics["sequential_read_mbps"] = readSpeed
	metrics["sequential_read_source"] = dt.backend.SequentialReadSource()
	dt.GetLogger().Info(fmt.Sprintf("读取速度: %.2f MB/s", readSpeed))

	// 测试随机IOPS
	dt.GetLogger().Info("开始磁盘随机IOPS测试...")
	iops, err := dt.backend.MeasureRandomIOPS(5) // 5秒测试
	if err != nil {
		return dt.CreateResult("failed", nil, fmt.Sprintf("IOPS测试失败: %v", err)), err
	}
	metrics["random_iops"] = iops
	metrics["random_iops_source"] = dt.backend.RandomIOPSSource()
	dt.GetLogger().Info(fmt.Sprintf("随机IOPS: %d", iops))

	// 计算总体评分
	score := dt.calculateScore(readSpeed, writeSpeed, float64(iops))
	metrics["score"] = score
	dt.backend.AppendMetrics(metrics)

	dt.GetLogger().Info(fmt.Sprintf("磁盘测试完成，评分: %.2f", score))

	return dt.CreateResult("success", metrics, ""), nil
}

// Teardown 测试后的清理工作
// 删除测试文件
// 返回:
//   - error: 清理错误
func (dt *DiskTest) Teardown() error {
	if dt.backend != nil {
		return dt.backend.Cleanup()
	}
	return dt.CleanupTestFiles()
}

// TestSequentialWrite 测试顺序写入速度
// 参数:
//   - fileSizeMB: 文件大小（MB）
//
// 返回:
//   - float64: 写入速度（MB/s）
//   - error: 测试错误
func (dt *DiskTest) TestSequentialWrite(fileSizeMB int) (float64, error) {
	// 生成随机数据
	size := fileSizeMB * 1024 * 1024
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return 0, fmt.Errorf("生成测试数据失败: %w", err)
	}

	// 写入文件
	startTime := time.Now()

	file, err := os.Create(dt.testFilePath)
	if err != nil {
		return 0, fmt.Errorf("创建测试文件失败: %w", err)
	}
	defer file.Close()

	written, err := file.Write(data)
	if err != nil {
		return 0, fmt.Errorf("写入文件失败: %w", err)
	}

	// 确保数据写入磁盘
	if err := file.Sync(); err != nil {
		return 0, fmt.Errorf("同步文件失败: %w", err)
	}

	duration := time.Since(startTime)

	// 计算速度（MB/s）
	speed := float64(written) / 1024 / 1024 / duration.Seconds()

	return speed, nil
}

// TestSequentialRead 测试顺序读取速度
// 参数:
//   - fileSizeMB: 文件大小（MB）
//
// 返回:
//   - float64: 读取速度（MB/s）
//   - error: 测试错误
func (dt *DiskTest) TestSequentialRead(fileSizeMB int) (float64, error) {
	// 检查文件是否存在
	fileInfo, err := os.Stat(dt.testFilePath)
	if err != nil {
		return 0, fmt.Errorf("测试文件不存在: %w", err)
	}

	// 读取文件
	startTime := time.Now()

	file, err := os.Open(dt.testFilePath)
	if err != nil {
		return 0, fmt.Errorf("打开测试文件失败: %w", err)
	}
	defer file.Close()

	// 读取所有数据
	buffer := make([]byte, fileInfo.Size())
	bytesRead, err := file.Read(buffer)
	if err != nil {
		return 0, fmt.Errorf("读取文件失败: %w", err)
	}

	duration := time.Since(startTime)

	// 计算速度（MB/s）
	speed := float64(bytesRead) / 1024 / 1024 / duration.Seconds()

	return speed, nil
}

// TestRandomIOPS 测试随机读写IOPS
// 参数:
//   - durationSec: 测试持续时间（秒）
//
// 返回:
//   - int: IOPS值
//   - error: 测试错误
func (dt *DiskTest) TestRandomIOPS(durationSec int) (int, error) {
	// 创建测试文件
	file, err := os.Create(dt.testFilePath + ".iops")
	if err != nil {
		return 0, fmt.Errorf("创建IOPS测试文件失败: %w", err)
	}
	defer func() {
		file.Close()
		os.Remove(dt.testFilePath + ".iops")
	}()

	// 准备4KB数据块（标准IOPS测试块大小）
	blockSize := 4096
	data := make([]byte, blockSize)
	rand.Read(data)

	// 执行随机写入测试
	startTime := time.Now()
	operations := 0

	for time.Since(startTime) < time.Duration(durationSec)*time.Second {
		// 随机写入
		if _, err := file.WriteAt(data, int64(operations*blockSize)); err != nil {
			return 0, fmt.Errorf("随机写入失败: %w", err)
		}
		operations++

		// 每100次操作同步一次
		if operations%100 == 0 {
			file.Sync()
		}
	}

	duration := time.Since(startTime)

	// 计算IOPS
	iops := int(float64(operations) / duration.Seconds())

	return iops, nil
}

// CleanupTestFiles 清理测试文件
// 返回:
//   - error: 清理错误
func (dt *DiskTest) CleanupTestFiles() error {
	// 删除主测试文件
	if err := os.Remove(dt.testFilePath); err != nil && !os.IsNotExist(err) {
		dt.GetLogger().Warn(fmt.Sprintf("删除测试文件失败: %v", err))
	}

	// 删除IOPS测试文件
	iopsFile := dt.testFilePath + ".iops"
	if err := os.Remove(iopsFile); err != nil && !os.IsNotExist(err) {
		dt.GetLogger().Warn(fmt.Sprintf("删除IOPS测试文件失败: %v", err))
	}

	return nil
}

// calculateScore 计算磁盘性能评分
// 参数:
//   - readSpeed: 读取速度（MB/s）
//   - writeSpeed: 写入速度（MB/s）
//   - iops: 随机IOPS
//
// 返回:
//   - float64: 评分（0-100）
func (dt *DiskTest) calculateScore(readSpeed, writeSpeed, iops float64) float64 {
	// 基准值
	// SSD: 读取~500MB/s, 写入~400MB/s, IOPS~50000
	// HDD: 读取~150MB/s, 写入~120MB/s, IOPS~100
	const baseReadSpeed = 300.0  // MB/s
	const baseWriteSpeed = 250.0 // MB/s
	const baseIOPS = 10000.0     // IOPS

	// 计算各项评分
	readScore := (readSpeed / baseReadSpeed) * 100
	writeScore := (writeSpeed / baseWriteSpeed) * 100
	iopsScore := (iops / baseIOPS) * 100

	// 限制评分范围
	if readScore > 100 {
		readScore = 100
	}
	if writeScore > 100 {
		writeScore = 100
	}
	if iopsScore > 100 {
		iopsScore = 100
	}

	// 读取30%，写入30%，IOPS 40%
	totalScore := readScore*0.3 + writeScore*0.3 + iopsScore*0.4

	return totalScore
}
