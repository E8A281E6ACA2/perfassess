// Package models 定义报告相关的数据结构
package models

import (
	"time"
	
	"performance-assessment-system/internal/config"
)

// Report 表示完整的性能评估报告
// 包含系统信息、测试结果、摘要和格式化内容
type Report struct {
	// SessionID 评估会话的唯一标识符
	SessionID string `json:"session_id"`
	
	// Timestamp 报告生成时间
	Timestamp time.Time `json:"timestamp"`
	
	// SystemInfo 系统硬件和软件信息
	SystemInfo *SystemInfo `json:"system_info"`
	
	// TestResults 所有性能测试的结果
	TestResults *TestResults `json:"test_results"`
	
	// Summary 评估摘要信息
	// 包含总体评分、性能等级等关键信息
	Summary map[string]interface{} `json:"summary"`
	
	// FormattedContent 格式化后的报告内容（用于文本输出）
	FormattedContent string `json:"-"`
}

// AssessmentSession 表示一次评估会话
// 记录会话的配置和状态信息
type AssessmentSession struct {
	// SessionID 会话的唯一标识符，通常使用时间戳生成
	SessionID string `json:"session_id"`
	
	// StartTime 会话开始时间
	StartTime time.Time `json:"start_time"`
	
	// Config 会话配置参数
	Config *config.Config `json:"config"`
	
	// LogFile 会话日志文件路径
	LogFile string `json:"log_file"`
}
