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

	// DirectionGroup 路由方向分组，例如 public 或 china_reference
	DirectionGroup string `json:"direction_group,omitempty"`

	// IsRealReturnRoute 是否为真实回程检测。当前内置 traceroute 为本机出站路径，因此默认为 false
	IsRealReturnRoute bool `json:"is_real_return_route"`

	// LastVisibleHop 最后一跳可见 IP 或主机名
	LastVisibleHop string `json:"last_visible_hop,omitempty"`

	// TimeoutHops 超时或不可见跳点数量
	TimeoutHops int `json:"timeout_hops"`

	// AverageLatencyMs 可见跳点平均延迟，单位毫秒
	AverageLatencyMs float64 `json:"average_latency_ms,omitempty"`

	// Quality 面向人工阅读的路由质量结论
	Quality *RouteQuality `json:"quality,omitempty"`

	// Evidence 支持路由质量结论的证据项
	Evidence []*RouteEvidence `json:"evidence,omitempty"`

	// EvidenceSummary 说明路由结论的证据可信度和边界
	EvidenceSummary []*EvidenceSummary `json:"evidence_summary,omitempty"`

	// Recommendations 路由相关建议
	Recommendations []string `json:"recommendations,omitempty"`
}

// RouteQuality 表示单条路由追踪的质量结论
type RouteQuality struct {
	Grade          string `json:"grade"`
	Status         string `json:"status"`
	Summary        string `json:"summary"`
	DirectionLabel string `json:"direction_label"`
}

// RouteEvidence 表示路由追踪证据项
type RouteEvidence struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}
