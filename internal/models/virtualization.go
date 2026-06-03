// Package models 定义虚拟化相关的数据结构
package models

// VirtualizationInfo 表示虚拟化环境信息
// 用于检测和记录服务器的虚拟化类型
type VirtualizationInfo struct {
	// IsVirtualized 是否运行在虚拟化环境中
	// true表示虚拟机，false表示物理机
	IsVirtualized bool `json:"is_virtualized"`

	// Type 虚拟化类型
	// 可选值: "KVM", "VMware", "Xen", "OpenVZ", "Hyper-V", "VirtualBox", "Physical", "Unknown"
	Type string `json:"type"`

	// Vendor 虚拟化厂商名称
	// 如 "QEMU", "VMware, Inc.", "Microsoft Corporation"
	Vendor string `json:"vendor,omitempty"`
}
