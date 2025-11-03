// Package utils 提供错误处理相关的工具函数
package utils

import (
	"errors"
	"fmt"
	"strings"

	"performance-assessment-system/pkg/logger"
)

// ErrorAction 定义错误处理动作类型
type ErrorAction int

const (
	// ActionTerminate 终止程序执行
	ActionTerminate ErrorAction = iota
	
	// ActionSkipTest 跳过当前测试
	ActionSkipTest
	
	// ActionRetry 重试操作
	ActionRetry
	
	// ActionContinue 继续执行
	ActionContinue
)

// ErrorHandler 错误处理器结构体
// 负责分类和处理各种错误情况
type ErrorHandler struct {
	// logger 日志记录器
	logger *logger.Logger
}

// NewErrorHandler 创建新的错误处理器
// 参数:
//   - logger: 日志记录器实例
// 返回:
//   - *ErrorHandler: 错误处理器实例
func NewErrorHandler(logger *logger.Logger) *ErrorHandler {
	return &ErrorHandler{
		logger: logger,
	}
}

// HandleError 处理错误并返回相应的处理动作
// 参数:
//   - err: 错误对象
//   - context: 错误发生的上下文信息
// 返回:
//   - ErrorAction: 建议的错误处理动作
func (eh *ErrorHandler) HandleError(err error, context string) ErrorAction {
	if err == nil {
		return ActionContinue
	}
	
	// 判断是否为致命错误
	if eh.IsFatal(err) {
		eh.logger.Error(fmt.Sprintf("致命错误 [%s]", context), err)
		return ActionTerminate
	}
	
	// 判断是否需要跳过测试
	if eh.shouldSkipTest(err) {
		eh.logger.Warn(fmt.Sprintf("跳过测试 [%s]", context))
		return ActionSkipTest
	}
	
	// 判断是否可以重试
	if eh.canRetry(err) {
		eh.logger.Warn(fmt.Sprintf("可重试错误 [%s]", context))
		return ActionRetry
	}
	
	// 默认继续执行
	eh.LogAndContinue(err, context)
	return ActionContinue
}

// IsFatal 判断错误是否为致命错误
// 致命错误会导致程序终止
// 参数:
//   - err: 错误对象
// 返回:
//   - bool: true表示致命错误，false表示非致命错误
func (eh *ErrorHandler) IsFatal(err error) bool {
	if err == nil {
		return false
	}
	
	errMsg := strings.ToLower(err.Error())
	
	// 致命错误关键词列表
	fatalKeywords := []string{
		"platform not supported",     // 平台不支持
		"平台不支持",
		"permission denied",          // 权限不足
		"权限不足",
		"access denied",
		"访问被拒绝",
		"out of memory",              // 内存不足
		"内存不足",
		"cannot allocate memory",
		"无法分配内存",
		"disk full",                  // 磁盘已满
		"磁盘已满",
		"no space left",
		"没有剩余空间",
	}
	
	// 检查错误消息是否包含致命错误关键词
	for _, keyword := range fatalKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}
	
	return false
}

// shouldSkipTest 判断是否应该跳过测试
// 参数:
//   - err: 错误对象
// 返回:
//   - bool: true表示应该跳过测试
func (eh *ErrorHandler) shouldSkipTest(err error) bool {
	if err == nil {
		return false
	}
	
	errMsg := strings.ToLower(err.Error())
	
	// 需要跳过测试的错误关键词
	skipKeywords := []string{
		"insufficient",               // 资源不足
		"资源不足",
		"not enough",
		"network unavailable",        // 网络不可用
		"网络不可用",
		"no network",
		"没有网络",
		"disk space",                 // 磁盘空间不足
		"磁盘空间",
		"available memory",           // 可用内存不足
		"可用内存",
	}
	
	for _, keyword := range skipKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}
	
	return false
}

// canRetry 判断错误是否可以重试
// 参数:
//   - err: 错误对象
// 返回:
//   - bool: true表示可以重试
func (eh *ErrorHandler) canRetry(err error) bool {
	if err == nil {
		return false
	}
	
	errMsg := strings.ToLower(err.Error())
	
	// 可重试的错误关键词
	retryKeywords := []string{
		"timeout",                    // 超时
		"超时",
		"temporary",                  // 临时错误
		"临时",
		"connection refused",         // 连接被拒绝
		"连接被拒绝",
		"connection reset",           // 连接重置
		"连接重置",
	}
	
	for _, keyword := range retryKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}
	
	return false
}

// LogAndContinue 记录错误日志并继续执行
// 参数:
//   - err: 错误对象
//   - context: 错误上下文信息
func (eh *ErrorHandler) LogAndContinue(err error, context string) {
	if err != nil {
		eh.logger.Warn(fmt.Sprintf("非致命错误 [%s]，继续执行", context))
	}
}

// 预定义的错误类型

var (
	// ErrPlatformNotSupported 平台不支持错误
	ErrPlatformNotSupported = errors.New("平台不支持")
	
	// ErrPermissionDenied 权限不足错误
	ErrPermissionDenied = errors.New("权限不足")
	
	// ErrInsufficientMemory 内存不足错误
	ErrInsufficientMemory = errors.New("可用内存不足")
	
	// ErrInsufficientDiskSpace 磁盘空间不足错误
	ErrInsufficientDiskSpace = errors.New("磁盘空间不足")
	
	// ErrNetworkUnavailable 网络不可用错误
	ErrNetworkUnavailable = errors.New("网络不可用")
	
	// ErrTimeout 超时错误
	ErrTimeout = errors.New("操作超时")
	
	// ErrTestFailed 测试失败错误
	ErrTestFailed = errors.New("测试执行失败")
)

// WrapError 包装错误并添加上下文信息
// 参数:
//   - err: 原始错误
//   - context: 上下文信息
// 返回:
//   - error: 包装后的错误
func WrapError(err error, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", context, err)
}
