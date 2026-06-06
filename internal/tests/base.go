// Package tests 提供性能测试功能
package tests

import (
	"time"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
	"github.com/E8A281E6ACA2/perfassess/pkg/logger"
)

// PerformanceTest 性能测试接口
// 定义了所有性能测试必须实现的方法
type PerformanceTest interface {
	// Setup 测试前的准备工作
	// 返回错误时测试将被跳过
	Setup() error

	// Execute 执行测试
	// 返回测试结果和可能的错误
	Execute() (*models.TestResult, error)

	// Teardown 测试后的清理工作
	// 无论测试成功或失败都会被调用
	Teardown() error

	// GetTimeout 获取测试超时时间
	// 返回测试的最大执行时间
	GetTimeout() time.Duration

	// GetName 获取测试名称
	// 返回测试的显示名称
	GetName() string
}

// BaseTest 测试基类
// 提供所有测试的通用功能
type BaseTest struct {
	// name 测试名称
	name string

	// timeout 测试超时时间
	timeout time.Duration

	// logger 日志记录器
	logger *logger.Logger

	// startTime 测试开始时间
	startTime time.Time

	// endTime 测试结束时间
	endTime time.Time
}

// NewBaseTest 创建测试基类实例
// 参数:
//   - name: 测试名称
//   - timeout: 超时时间
//   - logger: 日志记录器
//
// 返回:
//   - *BaseTest: 基类实例
func NewBaseTest(name string, timeout time.Duration, logger *logger.Logger) *BaseTest {
	return &BaseTest{
		name:    name,
		timeout: timeout,
		logger:  logger,
	}
}

// GetName 获取测试名称
func (bt *BaseTest) GetName() string {
	return bt.name
}

// GetTimeout 获取测试超时时间
func (bt *BaseTest) GetTimeout() time.Duration {
	return bt.timeout
}

// GetLogger 获取日志记录器
func (bt *BaseTest) GetLogger() *logger.Logger {
	return bt.logger
}

// MarkStart 标记测试开始
func (bt *BaseTest) MarkStart() {
	bt.startTime = time.Now()
	if bt.logger != nil {
		bt.logger.LogTestStart(bt.name)
	}
}

// MarkEnd 标记测试结束
// 参数:
//   - status: 测试状态
func (bt *BaseTest) MarkEnd(status string) {
	bt.endTime = time.Now()
	duration := bt.endTime.Sub(bt.startTime)
	if bt.logger != nil {
		bt.logger.LogTestEnd(bt.name, status, duration)
	}
}

// GetDuration 获取测试持续时间
// 返回:
//   - time.Duration: 持续时间
func (bt *BaseTest) GetDuration() time.Duration {
	if bt.endTime.IsZero() {
		return time.Since(bt.startTime)
	}
	return bt.endTime.Sub(bt.startTime)
}

// CreateResult 创建测试结果
// 参数:
//   - status: 测试状态
//   - metrics: 测试指标
//   - errorMsg: 错误消息
//
// 返回:
//   - *models.TestResult: 测试结果
func (bt *BaseTest) CreateResult(status string, metrics map[string]interface{}, errorMsg string) *models.TestResult {
	startTime := bt.startTime
	if startTime.IsZero() {
		startTime = time.Now()
	}
	endTime := bt.endTime
	if endTime.IsZero() {
		endTime = time.Now()
	}

	return &models.TestResult{
		TestName:        bt.name,
		Status:          status,
		StartTime:       startTime,
		EndTime:         endTime,
		DurationSeconds: endTime.Sub(startTime).Seconds(),
		Metrics:         metrics,
		ErrorMessage:    errorMsg,
	}
}

// Setup 默认的Setup实现（空操作）
func (bt *BaseTest) Setup() error {
	return nil
}

// Teardown 默认的Teardown实现（空操作）
func (bt *BaseTest) Teardown() error {
	return nil
}
