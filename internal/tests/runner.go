// Package tests 提供测试运行器功能
package tests

import (
	"context"
	"fmt"
	"time"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
	"github.com/E8A281E6ACA2/perfassess/pkg/logger"
	"github.com/E8A281E6ACA2/perfassess/pkg/utils"
)

// TestRunner 测试运行器
// 负责管理和执行性能测试
type TestRunner struct {
	// logger 日志记录器
	logger *logger.Logger

	// errorHandler 错误处理器
	errorHandler *utils.ErrorHandler
}

// NewTestRunner 创建测试运行器
// 参数:
//   - logger: 日志记录器
//
// 返回:
//   - *TestRunner: 测试运行器实例
func NewTestRunner(logger *logger.Logger) *TestRunner {
	return &TestRunner{
		logger:       logger,
		errorHandler: utils.NewErrorHandler(logger),
	}
}

// RunTests 运行多个测试
// 参数:
//   - tests: 测试列表
//
// 返回:
//   - *models.TestResults: 所有测试的结果
//   - error: 运行错误
func (tr *TestRunner) RunTests(tests []PerformanceTest) (*models.TestResults, error) {
	results := &models.TestResults{}

	for _, test := range tests {
		// 检查是否应该跳过测试
		shouldSkip, reason := tr.ShouldSkipTest(test)
		if shouldSkip {
			tr.logger.Warn(fmt.Sprintf("跳过测试: %s, 原因: %s", test.GetName(), reason))

			// 创建跳过状态的结果
			result := &models.TestResult{
				TestName:        test.GetName(),
				Status:          "skipped",
				StartTime:       time.Now(),
				EndTime:         time.Now(),
				DurationSeconds: 0,
				ErrorMessage:    reason,
			}

			tr.assignResult(results, test.GetName(), result)
			continue
		}

		// 运行单个测试
		result, err := tr.RunSingleTest(test)
		if err != nil {
			// 处理错误
			action := tr.errorHandler.HandleError(err, test.GetName())

			if action == utils.ActionTerminate {
				if result == nil {
					result = &models.TestResult{
						TestName:        test.GetName(),
						Status:          "failed",
						StartTime:       time.Now(),
						EndTime:         time.Now(),
						DurationSeconds: 0,
						ErrorMessage:    err.Error(),
					}
				}
				tr.assignResult(results, test.GetName(), result)
				return results, fmt.Errorf("测试 %s 遇到致命错误: %w", test.GetName(), err)
			}

			// 非致命错误，继续执行其他测试
			if result == nil {
				result = &models.TestResult{
					TestName:        test.GetName(),
					Status:          "failed",
					StartTime:       time.Now(),
					EndTime:         time.Now(),
					DurationSeconds: 0,
					ErrorMessage:    err.Error(),
				}
			}
		}

		// 保存结果
		tr.assignResult(results, test.GetName(), result)
	}

	return results, nil
}

// RunSingleTest 运行单个测试
// 参数:
//   - test: 性能测试实例
//
// 返回:
//   - *models.TestResult: 测试结果
//   - error: 运行错误
func (tr *TestRunner) RunSingleTest(test PerformanceTest) (*models.TestResult, error) {
	testName := test.GetName()
	tr.logger.Info(fmt.Sprintf("开始测试: %s", testName))

	// Setup阶段
	if err := test.Setup(); err != nil {
		tr.logger.Error(fmt.Sprintf("测试 %s Setup失败", testName), err)
		now := time.Now()
		return &models.TestResult{
			TestName:        testName,
			Status:          "failed",
			StartTime:       now,
			EndTime:         now,
			DurationSeconds: 0,
			ErrorMessage:    fmt.Sprintf("Setup失败: %v", err),
		}, err
	}

	// 确保Teardown被执行
	defer func() {
		if err := test.Teardown(); err != nil {
			tr.logger.Error(fmt.Sprintf("测试 %s Teardown失败", testName), err)
		}
	}()

	// 使用超时控制执行测试
	ctx := context.Background()
	timeout := test.GetTimeout()

	resultInterface, err := utils.RunWithTimeoutAndResult(ctx, timeout, func() (interface{}, error) {
		return test.Execute()
	})

	if err != nil {
		if resultInterface != nil {
			if result, ok := resultInterface.(*models.TestResult); ok && result != nil {
				if result.Status == "" {
					result.Status = "failed"
				}
				if result.ErrorMessage == "" {
					result.ErrorMessage = err.Error()
				}
				tr.logger.Error(fmt.Sprintf("测试 %s 执行失败", testName), err)
				return result, err
			}
		}
		if utils.IsTimeout(err) {
			tr.logger.Error(fmt.Sprintf("测试 %s 超时", testName), err)
			now := time.Now()
			return &models.TestResult{
				TestName:        testName,
				Status:          "failed",
				StartTime:       now,
				EndTime:         now,
				DurationSeconds: 0,
				ErrorMessage:    fmt.Sprintf("测试超时（%v）", timeout),
			}, err
		}

		tr.logger.Error(fmt.Sprintf("测试 %s 执行失败", testName), err)
		now := time.Now()
		return &models.TestResult{
			TestName:        testName,
			Status:          "failed",
			StartTime:       now,
			EndTime:         now,
			DurationSeconds: 0,
			ErrorMessage:    err.Error(),
		}, err
	}

	result := resultInterface.(*models.TestResult)
	tr.logger.Info(fmt.Sprintf("测试 %s 完成，状态: %s", testName, result.Status))

	return result, nil
}

// ShouldSkipTest 判断是否应该跳过测试
// 检查系统资源是否满足测试要求
// 参数:
//   - test: 性能测试实例
//
// 返回:
//   - bool: 是否跳过
//   - string: 跳过原因
func (tr *TestRunner) ShouldSkipTest(test PerformanceTest) (bool, string) {
	// 这里可以添加更多的跳过条件检查
	// 例如：检查可用内存、磁盘空间、网络连接等

	// 目前返回false，表示不跳过
	// 具体的跳过逻辑将在各个测试的Setup方法中实现
	return false, ""
}

// assignResult 将测试结果分配到对应的字段
// 参数:
//   - results: 测试结果集合
//   - testName: 测试名称
//   - result: 测试结果
func (tr *TestRunner) assignResult(results *models.TestResults, testName string, result *models.TestResult) {
	switch testName {
	case "CPU性能测试", "CPU Performance Test":
		results.CPUResult = result
	case "内存性能测试", "Memory Performance Test":
		results.MemoryResult = result
	case "磁盘性能测试", "Disk Performance Test":
		results.DiskResult = result
	case "网络性能测试", "Network Performance Test":
		results.NetworkResult = result
	default:
		// 对于未知的测试名称，记录警告
		tr.logger.Warn(fmt.Sprintf("未知的测试名称: %s", testName))
	}
}
