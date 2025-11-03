// Package platform 提供Linux平台的适配器实现
package platform

// LinuxAdapter Linux平台适配器
// 实现了Linux系统特定的命令和操作
type LinuxAdapter struct{}

// NewLinuxAdapter 创建Linux平台适配器
// 返回:
//   - *LinuxAdapter: Linux适配器实例
func NewLinuxAdapter() *LinuxAdapter {
	return &LinuxAdapter{}
}

// GetCPUInfoCommand 获取CPU信息的命令
// Linux系统使用 /proc/cpuinfo 文件或 lscpu 命令
// 返回:
//   - string: CPU信息命令
func (la *LinuxAdapter) GetCPUInfoCommand() string {
	return "lscpu"
}

// GetMemoryInfoCommand 获取内存信息的命令
// Linux系统使用 /proc/meminfo 文件或 free 命令
// 返回:
//   - string: 内存信息命令
func (la *LinuxAdapter) GetMemoryInfoCommand() string {
	return "free -m"
}

// GetDiskInfoCommand 获取磁盘信息的命令
// Linux系统使用 df 命令
// 返回:
//   - string: 磁盘信息命令
func (la *LinuxAdapter) GetDiskInfoCommand() string {
	return "df -h"
}

// GetOSInfoCommand 获取操作系统信息的命令
// Linux系统使用 /etc/os-release 文件或 uname 命令
// 返回:
//   - string: OS信息命令
func (la *LinuxAdapter) GetOSInfoCommand() string {
	return "cat /etc/os-release"
}

// GetVirtualizationCheckCommand 获取虚拟化检测命令
// Linux系统检查 /sys/class/dmi/id/product_name 等文件
// 或使用 systemd-detect-virt 命令
// 返回:
//   - string: 虚拟化检测命令
func (la *LinuxAdapter) GetVirtualizationCheckCommand() string {
	return "systemd-detect-virt"
}

// GetNetworkInterfaceCommand 获取网络接口信息的命令
// Linux系统使用 ip 命令或 ifconfig 命令
// 返回:
//   - string: 网络接口信息命令
func (la *LinuxAdapter) GetNetworkInterfaceCommand() string {
	return "ip addr show"
}

// GetPlatformName 获取平台名称
// 返回:
//   - string: 平台名称 "Linux"
func (la *LinuxAdapter) GetPlatformName() string {
	return "Linux"
}
