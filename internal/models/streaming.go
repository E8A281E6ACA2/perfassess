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

	// Category 平台分类，如 global、us、jp、cn、hk、kr、eu、asia、music
	Category string `json:"category,omitempty"`

	// UnlockType 解锁类型，如 full、partial、limited、blocked、login_required
	UnlockType string `json:"unlock_type,omitempty"`

	// Protocol 本次检测使用的网络协议视角，当前为 default
	Protocol string `json:"protocol,omitempty"`

	// RegionSource 区域判定来源说明
	RegionSource string `json:"region_source,omitempty"`
}
