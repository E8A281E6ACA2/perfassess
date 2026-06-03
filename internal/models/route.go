// Package models 定义路由追踪相关的数据结构
package models

import "time"

// Hop 表示路由追踪中的一跳
// 记录每一跳的序号、IP地址和延迟
type Hop struct {
	// Number 跳数序号，从1开始
	Number int `json:"number"`

	// IP 该跳的IP地址
	// 如果无法获取则为空字符串或"*"
	IP string `json:"ip"`

	// Latency 到达该跳的延迟时间
	Latency time.Duration `json:"latency"`

	// Hostname 主机名（如果可解析）
	Hostname string `json:"hostname,omitempty"`
}

// TraceResult 表示一次路由追踪的完整结果
// 包含目标地址、所有跳数和追踪状态
type TraceResult struct {
	// Target 追踪的目标地址
	// 可以是IP地址或域名
	Target string `json:"target"`

	// Hops 所有跳的详细信息列表
	Hops []*Hop `json:"hops"`

	// TotalHops 总跳数
	TotalHops int `json:"total_hops"`

	// Success 追踪是否成功完成
	// false表示超时或其他错误
	Success bool `json:"success"`

	// ErrorMessage 错误消息（如果追踪失败）
	ErrorMessage string `json:"error_message,omitempty"`
}
