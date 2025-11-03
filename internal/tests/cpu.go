// Package tests 提供CPU性能测试功能
package tests

import (
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"performance-assessment-system/internal/models"
	"performance-assessment-system/pkg/logger"
)

// CPUTest CPU性能测试
// 测试单核和多核CPU的计算性能
type CPUTest struct {
	*BaseTest
}

// NewCPUTest 创建CPU性能测试
// 参数:
//   - logger: 日志记录器
// 返回:
//   - *CPUTest: CPU测试实例
func NewCPUTest(logger *logger.Logger) *CPUTest {
	return &CPUTest{
		BaseTest: NewBaseTest("CPU性能测试", 60*time.Second, logger),
	}
}

// Execute 执行CPU性能测试
// 返回:
//   - *models.TestResult: 测试结果
//   - error: 测试错误
func (ct *CPUTest) Execute() (*models.TestResult, error) {
	ct.MarkStart()
	defer func() {
		ct.MarkEnd("success")
	}()
	
	metrics := make(map[string]interface{})
	
	// 测试单核性能
	ct.GetLogger().Info("开始单核CPU测试...")
	singleCoreScore, err := ct.TestSingleCore()
	if err != nil {
		return ct.CreateResult("failed", nil, fmt.Sprintf("单核测试失败: %v", err)), err
	}
	metrics["single_core_score"] = singleCoreScore
	ct.GetLogger().Info(fmt.Sprintf("单核测试完成，评分: %.2f", singleCoreScore))
	
	// 测试多核性能
	ct.GetLogger().Info("开始多核CPU测试...")
	multiCoreScore, err := ct.TestMultiCore()
	if err != nil {
		return ct.CreateResult("failed", nil, fmt.Sprintf("多核测试失败: %v", err)), err
	}
	metrics["multi_core_score"] = multiCoreScore
	ct.GetLogger().Info(fmt.Sprintf("多核测试完成，评分: %.2f", multiCoreScore))
	
	// 计算总体评分
	totalScore := ct.CalculateScore(singleCoreScore, multiCoreScore)
	metrics["total_score"] = totalScore
	metrics["cpu_cores"] = runtime.NumCPU()
	
	ct.GetLogger().Info(fmt.Sprintf("CPU测试完成，总评分: %.2f", totalScore))
	
	return ct.CreateResult("success", metrics, ""), nil
}

// TestSingleCore 测试单核性能
// 使用质数计算作为基准测试
// 返回:
//   - float64: 单核性能评分
//   - error: 测试错误
func (ct *CPUTest) TestSingleCore() (float64, error) {
	startTime := time.Now()
	
	// 计算一定范围内的质数
	const maxNumber = 100000
	primeCount := ct.countPrimes(maxNumber)
	
	duration := time.Since(startTime)
	
	// 基准：在1秒内计算完成得100分
	// 实际评分 = (1秒 / 实际用时) * 100
	baseTime := 1.0 // 秒
	score := (baseTime / duration.Seconds()) * 100
	
	// 限制评分范围在0-100之间
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	
	ct.GetLogger().Debug(fmt.Sprintf("单核测试: 计算了 %d 个质数，用时 %.3f 秒", primeCount, duration.Seconds()))
	
	return score, nil
}

// TestMultiCore 测试多核性能
// 使用并行计算测试多核性能
// 返回:
//   - float64: 多核性能评分
//   - error: 测试错误
func (ct *CPUTest) TestMultiCore() (float64, error) {
	numCPU := runtime.NumCPU()
	startTime := time.Now()
	
	// 使用所有CPU核心并行计算
	var wg sync.WaitGroup
	var totalOps int64
	
	// 每个核心执行相同的计算任务
	for i := 0; i < numCPU; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			// 执行密集计算（矩阵运算）
			ops := ct.performMatrixOperations(1000)
			atomic.AddInt64(&totalOps, ops)
		}()
	}
	
	wg.Wait()
	duration := time.Since(startTime)
	
	// 基准：在2秒内完成得100分
	baseTime := 2.0 // 秒
	score := (baseTime / duration.Seconds()) * 100
	
	// 限制评分范围
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	
	ct.GetLogger().Debug(fmt.Sprintf("多核测试: %d 个核心，完成 %d 次操作，用时 %.3f 秒", 
		numCPU, totalOps, duration.Seconds()))
	
	return score, nil
}

// countPrimes 计算指定范围内的质数个数
// 参数:
//   - max: 最大数值
// 返回:
//   - int: 质数个数
func (ct *CPUTest) countPrimes(max int) int {
	count := 0
	for n := 2; n <= max; n++ {
		if ct.isPrime(n) {
			count++
		}
	}
	return count
}

// isPrime 判断一个数是否为质数
// 参数:
//   - n: 待判断的数
// 返回:
//   - bool: 是否为质数
func (ct *CPUTest) isPrime(n int) bool {
	if n < 2 {
		return false
	}
	if n == 2 {
		return true
	}
	if n%2 == 0 {
		return false
	}
	
	sqrtN := int(math.Sqrt(float64(n)))
	for i := 3; i <= sqrtN; i += 2 {
		if n%i == 0 {
			return false
		}
	}
	return true
}

// performMatrixOperations 执行矩阵运算
// 参数:
//   - iterations: 迭代次数
// 返回:
//   - int64: 完成的操作数
func (ct *CPUTest) performMatrixOperations(iterations int) int64 {
	const size = 50
	var ops int64
	
	// 创建两个矩阵
	a := make([][]float64, size)
	b := make([][]float64, size)
	c := make([][]float64, size)
	
	for i := 0; i < size; i++ {
		a[i] = make([]float64, size)
		b[i] = make([]float64, size)
		c[i] = make([]float64, size)
		
		for j := 0; j < size; j++ {
			a[i][j] = float64(i + j)
			b[i][j] = float64(i - j)
		}
	}
	
	// 执行矩阵乘法
	for iter := 0; iter < iterations; iter++ {
		for i := 0; i < size; i++ {
			for j := 0; j < size; j++ {
				sum := 0.0
				for k := 0; k < size; k++ {
					sum += a[i][k] * b[k][j]
					ops++
				}
				c[i][j] = sum
			}
		}
	}
	
	return ops
}

// CalculateScore 计算CPU总体评分
// 参数:
//   - singleCoreScore: 单核评分
//   - multiCoreScore: 多核评分
// 返回:
//   - float64: 总体评分
func (ct *CPUTest) CalculateScore(singleCoreScore, multiCoreScore float64) float64 {
	// 单核占40%，多核占60%
	return singleCoreScore*0.4 + multiCoreScore*0.6
}
