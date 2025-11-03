// Package platform 提供Windows平台的适配器实现
package platform

// WindowsAdapter Windows平台适配器
// 实现了Windows系统特定的命令和操作
type WindowsAdapter struct{}

// NewWindowsAdapter 创建Windows平台适配器
// 返回:
//   - *WindowsAdapter: Windows适配器实例
func NewWindowsAdapter() *WindowsAdapter {
	return &WindowsAdapter{}
}

// GetCPUInfoCommand 获取CPU信息的命令
// Windows系统使用 wmic 命令或 PowerShell
// 返回:
//   - string: CPU信息命令
func (wa *WindowsAdapter) GetCPUInfoCommand() string {
	return "wmic cpu get Name,NumberOfCores,NumberOfLogicalProcessors,MaxClockSpeed"
}

// GetMemoryInfoCommand 获取内存信息的命令
// Windows系统使用 wmic 命令或 systeminfo
// 返回:
//   - string: 内存信息命令
func (wa *WindowsAdapter) GetMemoryInfoCommand() string {
	return "wmic OS get TotalVisibleMemorySize,FreePhysicalMemory"
}

// GetDiskInfoCommand 获取磁盘信息的命令
// Windows系统使用 wmic 命令
// 返回:
//   - string: 磁盘信息命令
func (wa *WindowsAdapter) GetDiskInfoCommand() string {
	return "wmic logicaldisk get Size,FreeSpace,Caption"
}

// GetOSInfoCommand 获取操作系统信息的命令
// Windows系统使用 ver 命令或 systeminfo
// 返回:
//   - string: OS信息命令
func (wa *WindowsAdapter) GetOSInfoCommand() string {
	return "wmic os get Caption,Version,OSArchitecture"
}

// GetVirtualizationCheckCommand 获取虚拟化检测命令
// Windows系统使用 WMI 查询检测虚拟化
// 返回:
//   - string: 虚拟化检测命令
func (wa *WindowsAdapter) GetVirtualizationCheckCommand() string {
	return "wmic computersystem get Model,Manufacturer"
}

// GetNetworkInterfaceCommand 获取网络接口信息的命令
// Windows系统使用 ipconfig 命令
// 返回:
//   - string: 网络接口信息命令
func (wa *WindowsAdapter) GetNetworkInterfaceCommand() string {
	return "ipconfig /all"
}

// GetPlatformName 获取平台名称
// 返回:
//   - string: 平台名称 "Windows"
func (wa *WindowsAdapter) GetPlatformName() string {
	return "Windows"
}
