// Package platform 定义平台适配器接口
package platform

// PlatformAdapter 平台适配器接口
// 定义了各平台需要实现的系统命令和操作
type PlatformAdapter interface {
	// GetCPUInfoCommand 获取CPU信息的命令
	// 返回用于获取CPU信息的系统命令
	GetCPUInfoCommand() string

	// GetMemoryInfoCommand 获取内存信息的命令
	// 返回用于获取内存信息的系统命令
	GetMemoryInfoCommand() string

	// GetDiskInfoCommand 获取磁盘信息的命令
	// 返回用于获取磁盘信息的系统命令
	GetDiskInfoCommand() string

	// GetOSInfoCommand 获取操作系统信息的命令
	// 返回用于获取OS信息的系统命令
	GetOSInfoCommand() string

	// GetVirtualizationCheckCommand 获取虚拟化检测命令
	// 返回用于检测虚拟化环境的系统命令
	GetVirtualizationCheckCommand() string

	// GetNetworkInterfaceCommand 获取网络接口信息的命令
	// 返回用于获取网络接口信息的系统命令
	GetNetworkInterfaceCommand() string

	// GetPlatformName 获取平台名称
	// 返回平台的友好名称
	GetPlatformName() string
}
