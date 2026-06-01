// Package models 定义评分相关的数据结构
package models

// OverallScore 表示综合性能评分
// 包含各项测试的单独评分和总体评分
type OverallScore struct {
	// CPUScore CPU性能评分（0-100）
	CPUScore float64 `json:"cpu_score"`

	// MemoryScore 内存性能评分（0-100）
	MemoryScore float64 `json:"memory_score"`

	// DiskScore 磁盘性能评分（0-100）
	DiskScore float64 `json:"disk_score"`

	// NetworkScore 网络性能评分（0-100）
	NetworkScore float64 `json:"network_score"`

	// TotalScore 总体性能评分（0-100）
	// 基于各项评分的加权平均计算
	TotalScore float64 `json:"total_score"`

	// Grade 性能等级
	// 可选值: "优秀"(90+), "良好"(75-89), "一般"(60-74), "较差"(<60)
	Grade string `json:"grade"`

	// Weights 各项测试的权重配置
	// 用于记录评分计算时使用的权重
	Weights map[string]float64 `json:"weights,omitempty"`
}
