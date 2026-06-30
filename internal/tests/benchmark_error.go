package tests

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const (
	BenchmarkErrorMissingDependency = "missing_dependency"
	BenchmarkErrorInvalidConfig     = "invalid_config"
	BenchmarkErrorCommandFailed     = "command_failed"
	BenchmarkErrorPermission        = "permission_denied"
	BenchmarkErrorResource          = "resource_limited"
	BenchmarkErrorNetwork           = "network_unavailable"
	BenchmarkErrorParseFailed       = "parse_failed"
	BenchmarkErrorTimeout           = "timeout"
	BenchmarkErrorRuntime           = "runtime_error"
)

type BenchmarkError struct {
	Category string
	Stage    string
	Hint     string
	Err      error
}

func (e *BenchmarkError) Error() string {
	if e == nil {
		return ""
	}
	parts := []string{}
	if e.Stage != "" {
		parts = append(parts, e.Stage)
	}
	if e.Category != "" {
		parts = append(parts, e.Category)
	}
	prefix := strings.Join(parts, ": ")
	if e.Err == nil {
		if e.Hint != "" && prefix != "" {
			return prefix + ": " + e.Hint
		}
		return prefix
	}
	message := e.Err.Error()
	if e.Hint != "" {
		message += "; " + e.Hint
	}
	if prefix == "" {
		return message
	}
	return fmt.Sprintf("%s: %s", prefix, message)
}

func (e *BenchmarkError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func newBenchmarkError(category string, stage string, hint string, err error) error {
	return &BenchmarkError{
		Category: category,
		Stage:    stage,
		Hint:     hint,
		Err:      err,
	}
}

func classifyExternalCommandError(stage string, output []byte, err error, defaultHint string) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return newBenchmarkError(BenchmarkErrorTimeout, stage, "外部命令超时；建议降低测评档位、缩短测试时间，或检查机器是否负载过高。", err)
	}

	text := strings.ToLower(strings.TrimSpace(string(output) + "\n" + errorString(err)))
	switch {
	case strings.Contains(text, "executable file not found") ||
		strings.Contains(text, "no such file or directory"):
		return newBenchmarkError(BenchmarkErrorMissingDependency, stage, "外部命令不可执行或路径不存在；请确认依赖已安装并在 PATH 中。", err)
	case strings.Contains(text, "accept the license") ||
		strings.Contains(text, "license agreement") ||
		strings.Contains(text, "accept-license") ||
		strings.Contains(text, "accept-gdpr") ||
		strings.Contains(text, "terms of use") ||
		strings.Contains(text, "eula"):
		return newBenchmarkError(BenchmarkErrorInvalidConfig, stage, "外部工具需要许可/条款确认；请确认命令参数、版本和首次运行授权状态。", err)
	case strings.Contains(text, "invalid option") ||
		strings.Contains(text, "unrecognized option") ||
		strings.Contains(text, "unknown option") ||
		strings.Contains(text, "invalid argument") ||
		strings.Contains(text, "unsupported") ||
		strings.Contains(text, "not supported"):
		return newBenchmarkError(BenchmarkErrorInvalidConfig, stage, "外部工具参数、版本或当前系统能力不兼容；请检查工具版本、测试参数和平台支持情况。", err)
	case strings.Contains(text, "permission denied") ||
		strings.Contains(text, "operation not permitted") ||
		strings.Contains(text, "not permitted") ||
		strings.Contains(text, "access denied"):
		return newBenchmarkError(BenchmarkErrorPermission, stage, "权限不足；请确认测试目录权限、direct I/O 权限、二进制执行权限或安全策略。", err)
	case strings.Contains(text, "no space left") ||
		strings.Contains(text, "disk full") ||
		strings.Contains(text, "not enough space") ||
		strings.Contains(text, "cannot allocate memory") ||
		strings.Contains(text, "unable to allocate memory") ||
		strings.Contains(text, "out of memory") ||
		strings.Contains(text, "killed"):
		return newBenchmarkError(BenchmarkErrorResource, stage, "资源不足；请检查可用磁盘、内存和 swap，低配机器建议使用 basic/builtin 档位。", err)
	case strings.Contains(text, "network is unreachable") ||
		strings.Contains(text, "connection refused") ||
		strings.Contains(text, "connection timed out") ||
		strings.Contains(text, "temporary failure in name resolution") ||
		strings.Contains(text, "name or service not known") ||
		strings.Contains(text, "could not resolve") ||
		strings.Contains(text, "couldn't resolve") ||
		strings.Contains(text, "failed to connect") ||
		strings.Contains(text, "connection reset") ||
		strings.Contains(text, "broken pipe") ||
		strings.Contains(text, "tls handshake timeout") ||
		strings.Contains(text, "i/o timeout") ||
		strings.Contains(text, "timeout") ||
		strings.Contains(text, "too many requests") ||
		strings.Contains(text, "rate limit") ||
		strings.Contains(text, "no servers found"):
		return newBenchmarkError(BenchmarkErrorNetwork, stage, "外部测速网络不可达；请检查 DNS、出口网络、防火墙、安全组或测速服务端状态。", err)
	default:
		return newBenchmarkError(BenchmarkErrorCommandFailed, stage, defaultHint, err)
	}
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func benchmarkErrorFields(err error) (category string, stage string, hint string, ok bool) {
	var benchmarkErr *BenchmarkError
	if errors.As(err, &benchmarkErr) && benchmarkErr != nil {
		return benchmarkErr.Category, benchmarkErr.Stage, benchmarkErr.Hint, true
	}
	return "", "", "", false
}

func addBenchmarkErrorMetrics(metrics map[string]interface{}, err error) {
	if metrics == nil || err == nil {
		return
	}
	category, stage, hint, ok := benchmarkErrorFields(err)
	if !ok {
		return
	}
	if category != "" {
		metrics["error_category"] = category
	}
	if stage != "" {
		metrics["error_stage"] = stage
	}
	if hint != "" {
		metrics["error_hint"] = hint
	}
}

func addNetworkBenchmarkErrorMetrics(metrics map[string]interface{}, stage string, err error) {
	if metrics == nil || err == nil {
		return
	}
	category, errorStage, hint, ok := benchmarkErrorFields(err)
	if !ok {
		return
	}
	if errorStage == "" {
		errorStage = stage
	}
	if category != "" && metrics["error_category"] == nil {
		metrics["error_category"] = category
	}
	if errorStage != "" && metrics["error_stage"] == nil {
		metrics["error_stage"] = errorStage
	}
	if hint != "" && metrics["error_hint"] == nil {
		metrics["error_hint"] = hint
	}
	prefix := "network_error_" + stage
	if category != "" {
		metrics[prefix+"_category"] = category
	}
	if errorStage != "" {
		metrics[prefix+"_stage"] = errorStage
	}
	if hint != "" {
		metrics[prefix+"_hint"] = hint
	}
}
