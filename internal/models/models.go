// Package models 定义系统中使用的核心数据结构
// 包含性能指标、评估结果等数据模型
package models

import "time"

// CPUInfo 表示CPU硬件信息
// 包含CPU的型号、核心数、线程数和频率等关键参数
type CPUInfo struct {
	// Model CPU型号名称，如 "Intel Core i7-9700K"
	Model string `json:"model"`

	// Cores 物理核心数
	Cores int `json:"cores"`

	// Threads 逻辑线程数（支持超线程时大于核心数）
	Threads int `json:"threads"`

	// FrequencyMHz CPU主频，单位为MHz
	FrequencyMHz float64 `json:"frequency_mhz"`
}

// MemoryInfo 表示内存硬件信息
// 包含内存容量和类型等信息
type MemoryInfo struct {
	// TotalMB 总内存容量，单位为MB
	TotalMB int64 `json:"total_mb"`

	// AvailableMB 可用内存容量，单位为MB
	AvailableMB int64 `json:"available_mb"`

	// MemoryType 内存类型，如 "DDR4", "DDR5"
	MemoryType string `json:"memory_type"`
}

// DiskInfo 表示磁盘硬件信息
// 包含磁盘容量和类型等信息
type DiskInfo struct {
	// TotalGB 总磁盘容量，单位为GB
	TotalGB float64 `json:"total_gb"`

	// AvailableGB 可用磁盘容量，单位为GB
	AvailableGB float64 `json:"available_gb"`

	// DiskType 磁盘类型，如 "SSD", "HDD", "NVMe"
	DiskType string `json:"disk_type"`
}

// OSInfo 表示操作系统信息
// 包含操作系统名称、版本和架构等信息
type OSInfo struct {
	// Name 操作系统名称，如 "Linux", "Windows", "macOS"
	Name string `json:"name"`

	// Version 操作系统版本号，如 "Ubuntu 22.04", "Windows 11"
	Version string `json:"version"`

	// Architecture 系统架构，如 "amd64", "arm64"
	Architecture string `json:"architecture"`
}

// SystemInfo 表示完整的系统信息
// 整合了CPU、内存、磁盘和操作系统的所有信息
type SystemInfo struct {
	// CPU CPU硬件信息
	CPU *CPUInfo `json:"cpu"`

	// Memory 内存硬件信息
	Memory *MemoryInfo `json:"memory"`

	// Disk 磁盘硬件信息
	Disk *DiskInfo `json:"disk"`

	// OS 操作系统信息
	OS *OSInfo `json:"os"`

	// Virtualization 虚拟化信息
	Virtualization *VirtualizationInfo `json:"virtualization,omitempty"`

	// IPInfo IP地址和地理位置信息
	IPInfo *IPInfo `json:"ip_info,omitempty"`

	// CollectionTime 信息收集的时间戳
	CollectionTime time.Time `json:"collection_time"`
}
