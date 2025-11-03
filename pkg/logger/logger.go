// Package logger 提供统一的日志记录功能
// 封装 zap 日志库，提供简单易用的日志接口
package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 日志管理器结构体
// 封装了 zap.Logger 并提供会话日志管理功能
type Logger struct {
	// zapLogger zap日志记录器实例
	zapLogger *zap.Logger
	
	// logDir 日志文件存储目录
	logDir string
	
	// sessionLogFile 当前会话的日志文件路径
	sessionLogFile string
	
	// level 日志级别
	level zapcore.Level
}

// NewLogger 创建新的日志管理器
// 参数:
//   - logDir: 日志文件存储目录路径
//   - level: 日志级别（debug/info/warn/error）
// 返回:
//   - *Logger: 日志管理器实例
//   - error: 初始化错误
func NewLogger(logDir string, level string) (*Logger, error) {
	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %w", err)
	}
	
	// 解析日志级别
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}
	
	// 配置编码器（日志格式）
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	
	// 创建核心配置
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		zapLevel,
	)
	
	// 创建 zap logger
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	
	return &Logger{
		zapLogger: zapLogger,
		logDir:    logDir,
		level:     zapLevel,
	}, nil
}

// CreateSessionLog 创建会话日志文件
// 为当前评估会话创建一个带时间戳的日志文件
// 返回:
//   - string: 日志文件路径
//   - error: 创建错误
func (l *Logger) CreateSessionLog() (string, error) {
	// 生成带时间戳的日志文件名
	timestamp := time.Now().Format("20060102_150405")
	logFileName := fmt.Sprintf("session_%s.log", timestamp)
	logFilePath := filepath.Join(l.logDir, logFileName)
	
	// 创建日志文件
	logFile, err := os.Create(logFilePath)
	if err != nil {
		return "", fmt.Errorf("创建会话日志文件失败: %w", err)
	}
	logFile.Close()
	
	// 更新 logger 配置，同时输出到控制台和文件
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	
	// 重新打开文件用于写入
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("打开会话日志文件失败: %w", err)
	}
	
	// 创建多输出核心（控制台 + 文件）
	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		l.level,
	)
	
	fileCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(file),
		l.level,
	)
	
	core := zapcore.NewTee(consoleCore, fileCore)
	l.zapLogger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	l.sessionLogFile = logFilePath
	
	return logFilePath, nil
}

// Info 记录信息级别的日志
// 参数:
//   - message: 日志消息
//   - fields: 附加字段（可选）
func (l *Logger) Info(message string, fields ...zap.Field) {
	l.zapLogger.Info(message, fields...)
}

// Error 记录错误级别的日志
// 参数:
//   - message: 日志消息
//   - err: 错误对象
//   - fields: 附加字段（可选）
func (l *Logger) Error(message string, err error, fields ...zap.Field) {
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	l.zapLogger.Error(message, fields...)
}

// Debug 记录调试级别的日志
// 参数:
//   - message: 日志消息
//   - fields: 附加字段（可选）
func (l *Logger) Debug(message string, fields ...zap.Field) {
	l.zapLogger.Debug(message, fields...)
}

// Warn 记录警告级别的日志
// 参数:
//   - message: 日志消息
//   - fields: 附加字段（可选）
func (l *Logger) Warn(message string, fields ...zap.Field) {
	l.zapLogger.Warn(message, fields...)
}

// LogTestStart 记录测试开始
// 参数:
//   - testName: 测试名称
func (l *Logger) LogTestStart(testName string) {
	l.Info("测试开始", zap.String("test", testName))
}

// LogTestEnd 记录测试结束
// 参数:
//   - testName: 测试名称
//   - status: 测试状态（success/failed/skipped）
//   - duration: 测试持续时间
func (l *Logger) LogTestEnd(testName string, status string, duration time.Duration) {
	l.Info("测试结束",
		zap.String("test", testName),
		zap.String("status", status),
		zap.Duration("duration", duration),
	)
}

// LogSessionSummary 记录会话摘要信息
// 参数:
//   - summary: 摘要信息映射
func (l *Logger) LogSessionSummary(summary map[string]interface{}) {
	fields := make([]zap.Field, 0, len(summary))
	for key, value := range summary {
		fields = append(fields, zap.Any(key, value))
	}
	l.Info("评估会话摘要", fields...)
}

// Sync 同步日志缓冲区
// 确保所有日志都被写入到文件
// 返回:
//   - error: 同步错误
func (l *Logger) Sync() error {
	// 忽略 stdout/stderr 的同步错误（在某些系统上是正常的）
	err := l.zapLogger.Sync()
	if err != nil {
		errMsg := err.Error()
		// 忽略这些常见的无害错误
		if errMsg == "sync /dev/stdout: inappropriate ioctl for device" ||
			errMsg == "sync /dev/stderr: inappropriate ioctl for device" ||
			errMsg == "sync /dev/stdout: bad file descriptor" ||
			errMsg == "sync /dev/stderr: bad file descriptor" {
			return nil
		}
		return err
	}
	return nil
}

// GetSessionLogFile 获取当前会话日志文件路径
// 返回:
//   - string: 日志文件路径
func (l *Logger) GetSessionLogFile() string {
	return l.sessionLogFile
}
