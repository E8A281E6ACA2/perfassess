// Package models 定义 AI 服务检测相关的数据结构
package models

// AIServiceResult 表示单个 AI 服务的检测结果
// 用于描述是否可访问以及附加信息
type AIServiceResult struct {
	// Service AI 服务名称，如 "ChatGPT", "Claude"
	Service string `json:"service"`

	// Available 是否可访问
	Available bool `json:"available"`

	// Message 附加信息（如区域限制、需要登录等）
	Message string `json:"message"`

	// Category 服务分组，如 chatbot、search、coding
	Category string `json:"category,omitempty"`

	// AccessType 访问类型，如 full、login_required、restricted、rate_limited
	AccessType string `json:"access_type,omitempty"`

	// RegionHint 服务主要区域或区域策略提示
	RegionHint string `json:"region_hint,omitempty"`

	// EvidenceSummary 说明本次服务判断的依据、置信度和限制
	EvidenceSummary []*EvidenceSummary `json:"evidence_summary,omitempty"`
}
