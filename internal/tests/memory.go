// Package tests 提供内存性能测试功能
package tests

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/mem"

	"performance-assessment-system/internal/models"
	"performance-assessment-system/pkg/logger"
	"performance-assessment-system/pkg/utils"
)

// MemoryTest 内存性能测试
// 测试内存的读写速度
type MemoryTest struct {
	*BaseTest
	testSize int // 测试数据大小（MB）
}

// NewMemoryTest 创建内存性能测试
// 参数:
//   - logger: 日志记录器
// 返回:
//   - *MemoryTest: 内存测试实例
func NewMemoryTest(logger *logger.Logger) *MemoryTest {
	return &MemoryTest{
		BaseTest: NewBaseTest("内存性能测试", 60*time.Second, logger),
		testSize: 0, // 将在Setup中计算
	}
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
	defer func() {
		mt.MarkEnd("success")
	}()
	
	metrics := make(map[string]interface{})
	
	// 测试顺序读取速度
	mt.GetLogger().Info("开始内存顺序读取测试...")
	readSpeed, err := mt.TestSequentialRead(mt.testSize)
	if err != nil {
		return mt.CreateResult("failed", nil, fmt.Sprintf("读取测试失败: %v", err)), err
	}
	metrics["read_speed_mbps"] = readSpeed
	mt.GetLogger().Info(fmt.Sprintf("读取速度: %.2f MB/s", readSpeed))
	
	// 测试顺序写入速度
	mt.GetLogger().Info("开始内存顺序写入测试...")
	writeSpeed, err := mt.TestSequentialWrite(mt.testSize)
	if err != nil {
		return mt.CreateResult("failed", nil, fmt.Sprintf("写入测试失败: %v", err)), err
	}
	metrics["write_speed_mbps"] = writeSpeed
	mt.GetLogger().Info(fmt.Sprintf("写入速度: %.2f MB/s", writeSpeed))
	
	// 计算总体评分
	score := mt.calculateScore(readSpeed, writeSpeed)
	metrics["score"] = score
	metrics["test_size_mb"] = mt.testSize
	
	mt.GetLogger().Info(fmt.Sprintf("内存测试完成，评分: %.2f", score))
	
	return mt.CreateResult("success", metrics, ""), nil
}

// GetSafeTestSize 获取安全的测试大小
// 限制为可用内存的80%，最小512MB
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
	
	// 检查最小内存要求（512MB）
	if availableMB < 512 {
		return 0, utils.WrapError(
			utils.ErrInsufficientMemory,
			fmt.Sprintf("可用内存不足: %d MB < 512 MB", availableMB),
		)
	}
	
	// 使用可用内存的80%，但不超过2GB
	testSize := int(float64(availableMB) * 0.8)
	if testSize > 2048 {
		testSize = 2048
	}
	
	// 至少使用256MB进行测试
	if testSize < 256 {
		testSize = 256
	}
	
	return testSize, nil
}

// TestSequentialRead 测试顺序读取速度
// 参数:
//   - sizeMB: 测试数据大小（MB）
// 返回:
//   - float64: 读取速度（MB/s）
//   - error: 测试错误
func (mt *MemoryTest) TestSequentialRead(sizeMB int) (float64, error) {
	// 分配内存
	size := sizeMB * 1024 * 1024 // 转换为字节
	data := make([]byte, size)
	
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
// 返回:
//   - float64: 写入速度（MB/s）
//   - error: 测试错误
func (mt *MemoryTest) TestSequentialWrite(sizeMB int) (float64, error) {
	// 分配内存
	size := sizeMB * 1024 * 1024 // 转换为字节
	data := make([]byte, size)
	
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

// calculateScore 计算内存性能评分
// 参数:
//   - readSpeed: 读取速度（MB/s）
//   - writeSpeed: 写入速度（MB/s）
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
