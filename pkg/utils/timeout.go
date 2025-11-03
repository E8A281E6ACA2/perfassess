// Package utils 提供超时控制相关的工具函数
package utils

import (
	"context"
	"time"
)

// RunWithTimeout 在指定超时时间内执行函数
// 如果函数执行时间超过timeout，则返回超时错误
// 参数:
//   - ctx: 上下文对象，用于传递取消信号
//   - timeout: 超时时间
//   - fn: 要执行的函数
// 返回:
//   - error: 执行错误或超时错误
func RunWithTimeout(ctx context.Context, timeout time.Duration, fn func() error) error {
	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	// 创建错误通道
	errChan := make(chan error, 1)
	
	// 在goroutine中执行函数
	go func() {
		errChan <- fn()
	}()
	
	// 等待函数完成或超时
	select {
	case err := <-errChan:
		// 函数正常完成
		return err
	case <-ctx.Done():
		// 超时或取消
		if ctx.Err() == context.DeadlineExceeded {
			return ErrTimeout
		}
		return ctx.Err()
	}
}

// RunWithTimeoutAndResult 在指定超时时间内执行函数并返回结果
// 这是RunWithTimeout的泛型版本，可以返回函数的执行结果
// 参数:
//   - ctx: 上下文对象
//   - timeout: 超时时间
//   - fn: 要执行的函数，返回结果和错误
// 返回:
//   - interface{}: 函数执行结果
//   - error: 执行错误或超时错误
func RunWithTimeoutAndResult(ctx context.Context, timeout time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	// 创建结果通道
	type result struct {
		value interface{}
		err   error
	}
	resultChan := make(chan result, 1)
	
	// 在goroutine中执行函数
	go func() {
		value, err := fn()
		resultChan <- result{value: value, err: err}
	}()
	
	// 等待函数完成或超时
	select {
	case res := <-resultChan:
		// 函数正常完成
		return res.value, res.err
	case <-ctx.Done():
		// 超时或取消
		if ctx.Err() == context.DeadlineExceeded {
			return nil, ErrTimeout
		}
		return nil, ctx.Err()
	}
}

// WithTimeout 创建带超时的上下文
// 这是一个便捷函数，用于创建带超时的上下文
// 参数:
//   - parent: 父上下文
//   - timeout: 超时时间
// 返回:
//   - context.Context: 带超时的上下文
//   - context.CancelFunc: 取消函数
func WithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}

// WithDeadline 创建带截止时间的上下文
// 参数:
//   - parent: 父上下文
//   - deadline: 截止时间
// 返回:
//   - context.Context: 带截止时间的上下文
//   - context.CancelFunc: 取消函数
func WithDeadline(parent context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(parent, deadline)
}

// IsTimeout 判断错误是否为超时错误
// 参数:
//   - err: 错误对象
// 返回:
//   - bool: true表示超时错误
func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	return err == ErrTimeout || err == context.DeadlineExceeded
}
