// Package platform 提供平台检测和适配功能
// 负责识别操作系统类型并提供平台特定的实现
package platform

import (
	"runtime"

	"github.com/E8A281E6ACA2/perfassess/pkg/utils"
)

// PlatformType 定义平台类型常量
type PlatformType string

const (
	// PlatformLinux Linux操作系统
	PlatformLinux PlatformType = "linux"

	// PlatformWindows Windows操作系统
	PlatformWindows PlatformType = "windows"

	// PlatformMacOS macOS操作系统
	PlatformMacOS PlatformType = "darwin"

	// PlatformUnsupported 不支持的操作系统
	PlatformUnsupported PlatformType = "unsupported"
)

// PlatformDetector 平台检测器结构体
// 负责检测当前运行的操作系统平台
type PlatformDetector struct {
	// platform 当前检测到的平台类型
	platform PlatformType

	// arch 系统架构（amd64, arm64等）
	arch string
}

// NewPlatformDetector 创建新的平台检测器
// 返回:
//   - *PlatformDetector: 平台检测器实例
func NewPlatformDetector() *PlatformDetector {
	return &PlatformDetector{
		platform: PlatformType(runtime.GOOS),
		arch:     runtime.GOARCH,
	}
}

// DetectPlatform 检测当前操作系统平台
// 返回:
//   - PlatformType: 平台类型（linux/windows/darwin）
func (pd *PlatformDetector) DetectPlatform() PlatformType {
	return pd.platform
}

// GetArchitecture 获取系统架构
// 返回:
//   - string: 系统架构（amd64, arm64等）
func (pd *PlatformDetector) GetArchitecture() string {
	return pd.arch
}

// IsSupported 检查当前平台是否被支持
// 返回:
//   - bool: true表示支持，false表示不支持
func (pd *PlatformDetector) IsSupported() bool {
	switch pd.platform {
	case PlatformLinux, PlatformWindows, PlatformMacOS:
		return true
	default:
		return false
	}
}

// GetPlatformName 获取平台的友好名称
// 返回:
//   - string: 平台名称
func (pd *PlatformDetector) GetPlatformName() string {
	switch pd.platform {
	case PlatformLinux:
		return "Linux"
	case PlatformWindows:
		return "Windows"
	case PlatformMacOS:
		return "macOS"
	default:
		return "Unknown"
	}
}

// ValidatePlatform 验证平台支持性
// 如果平台不支持，返回错误
// 返回:
//   - error: 平台不支持时返回错误
func (pd *PlatformDetector) ValidatePlatform() error {
	if !pd.IsSupported() {
		return utils.WrapError(
			utils.ErrPlatformNotSupported,
			"当前平台 "+string(pd.platform)+" 不被支持",
		)
	}
	return nil
}

// GetPlatformAdapter 获取当前平台的适配器
// 根据检测到的平台类型返回相应的适配器实现
// 返回:
//   - PlatformAdapter: 平台适配器接口
//   - error: 平台不支持时返回错误
func (pd *PlatformDetector) GetPlatformAdapter() (PlatformAdapter, error) {
	if err := pd.ValidatePlatform(); err != nil {
		return nil, err
	}

	switch pd.platform {
	case PlatformLinux:
		return NewLinuxAdapter(), nil
	case PlatformWindows:
		return NewWindowsAdapter(), nil
	case PlatformMacOS:
		return NewMacOSAdapter(), nil
	default:
		return nil, utils.ErrPlatformNotSupported
	}
}
