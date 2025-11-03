// Package models 定义测试结果相关的数据结构
package models

import "time"

// TestResult 表示单个性能测试的结果
// 包含测试的状态、时间、指标和错误信息
type TestResult struct {
	// TestName 测试名称，如 "CPU性能测试", "内存性能测试"
	TestName string `json:"test_name"`
	
	// Status 测试状态，可选值: "success"(成功), "failed"(失败), "skipped"(跳过)
	Status string `json:"status"`
	
	// StartTime 测试开始时间
	StartTime time.Time `json:"start_time"`
	
	// EndTime 测试结束时间
	EndTime time.Time `json:"end_time"`
	
	// DurationSeconds 测试持续时间，单位为秒
	DurationSeconds float64 `json:"duration_seconds"`
	
	// Metrics 测试特定的指标数据
	// 不同类型的测试会有不同的指标，如CPU测试包含单核/多核评分，
	// 内存测试包含读写速度等
	Metrics map[string]interface{} `json:"metrics"`
	
	// ErrorMessage 错误消息，当测试失败时记录错误详情
	ErrorMessage string `json:"error_message,omitempty"`
}

// TestResults 表示所有性能测试的结果集合
// 包含CPU、内存、磁盘和网络测试的结果
type TestResults struct {
	// CPUResult CPU性能测试结果
	CPUResult *TestResult `json:"cpu_result,omitempty"`
	
	// MemoryResult 内存性能测试结果
	MemoryResult *TestResult `json:"memory_result,omitempty"`
	
	// DiskResult 磁盘性能测试结果
	DiskResult *TestResult `json:"disk_result,omitempty"`
	
	// NetworkResult 网络性能测试结果
	NetworkResult *TestResult `json:"network_result,omitempty"`
}
