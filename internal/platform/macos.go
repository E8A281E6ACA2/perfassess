// Package platform 提供macOS平台的适配器实现
package platform

// MacOSAdapter macOS平台适配器
// 实现了macOS系统特定的命令和操作
type MacOSAdapter struct{}

// NewMacOSAdapter 创建macOS平台适配器
// 返回:
//   - *MacOSAdapter: macOS适配器实例
func NewMacOSAdapter() *MacOSAdapter {
	return &MacOSAdapter{}
}

// GetCPUInfoCommand 获取CPU信息的命令
// macOS系统使用 sysctl 命令
// 返回:
//   - string: CPU信息命令
func (ma *MacOSAdapter) GetCPUInfoCommand() string {
	return "sysctl -n machdep.cpu.brand_string"
}

// GetMemoryInfoCommand 获取内存信息的命令
// macOS系统使用 sysctl 命令或 vm_stat
// 返回:
//   - string: 内存信息命令
func (ma *MacOSAdapter) GetMemoryInfoCommand() string {
	return "sysctl hw.memsize"
}

// GetDiskInfoCommand 获取磁盘信息的命令
// macOS系统使用 df 命令或 diskutil
// 返回:
//   - string: 磁盘信息命令
func (ma *MacOSAdapter) GetDiskInfoCommand() string {
	return "df -h"
}

// GetOSInfoCommand 获取操作系统信息的命令
// macOS系统使用 sw_vers 命令
// 返回:
//   - string: OS信息命令
func (ma *MacOSAdapter) GetOSInfoCommand() string {
	return "sw_vers"
}

// GetVirtualizationCheckCommand 获取虚拟化检测命令
// macOS系统使用 sysctl 命令检测虚拟化
// 返回:
//   - string: 虚拟化检测命令
func (ma *MacOSAdapter) GetVirtualizationCheckCommand() string {
	return "sysctl kern.hv_support"
}

// GetNetworkInterfaceCommand 获取网络接口信息的命令
// macOS系统使用 ifconfig 命令
// 返回:
//   - string: 网络接口信息命令
func (ma *MacOSAdapter) GetNetworkInterfaceCommand() string {
	return "ifconfig"
}

// GetPlatformName 获取平台名称
// 返回:
//   - string: 平台名称 "macOS"
func (ma *MacOSAdapter) GetPlatformName() string {
	return "macOS"
}
