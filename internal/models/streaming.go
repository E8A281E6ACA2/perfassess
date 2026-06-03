// Package models 定义流媒体检测相关的数据结构
package models

// StreamingResult 表示单个流媒体平台的检测结果
// 记录平台的可访问性和区域信息
type StreamingResult struct {
	// Platform 流媒体平台名称
	// 如 "Netflix", "YouTube", "Disney+"
	Platform string `json:"platform"`

	// Available 平台是否可访问
	// true表示可以访问，false表示被限制或不可用
	Available bool `json:"available"`

	// Region 可访问的区域
	// 如 "US", "JP", "Global"，仅在Available为true时有意义
	Region string `json:"region,omitempty"`

	// Message 检测消息
	// 提供额外的状态信息，如 "完全解锁", "仅限自制内容", "不可用"
	Message string `json:"message"`
}
