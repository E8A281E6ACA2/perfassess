// Package collector 提供虚拟化检测功能
package collector

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
	"github.com/E8A281E6ACA2/perfassess/internal/platform"
	"github.com/E8A281E6ACA2/perfassess/pkg/utils"
)

// VirtualizationDetector 虚拟化检测器
// 负责检测服务器是否运行在虚拟化环境中
type VirtualizationDetector struct {
	// adapter 平台适配器
	adapter platform.PlatformAdapter

	// timeout 检测超时时间
	timeout time.Duration
}

// NewVirtualizationDetector 创建虚拟化检测器
// 参数:
//   - adapter: 平台适配器
//
// 返回:
//   - *VirtualizationDetector: 虚拟化检测器实例
func NewVirtualizationDetector(adapter platform.PlatformAdapter) *VirtualizationDetector {
	return &VirtualizationDetector{
		adapter: adapter,
		timeout: 2 * time.Second, // 默认2秒超时
	}
}

// Detect 检测虚拟化环境
// 根据不同平台使用不同的检测方法
// 返回:
//   - *models.VirtualizationInfo: 虚拟化信息
//   - error: 检测错误
func (vd *VirtualizationDetector) Detect() (*models.VirtualizationInfo, error) {
	ctx := context.Background()

	result, err := utils.RunWithTimeoutAndResult(ctx, vd.timeout, func() (interface{}, error) {
		switch runtime.GOOS {
		case "linux":
			return vd.detectLinux()
		case "windows":
			return vd.detectWindows()
		case "darwin":
			return vd.detectMacOS()
		default:
			return &models.VirtualizationInfo{
				IsVirtualized: false,
				Type:          "Unknown",
				Vendor:        "",
			}, nil
		}
	})

	if err != nil {
		// 检测失败时返回未知状态
		return &models.VirtualizationInfo{
			IsVirtualized: false,
			Type:          "Unknown",
			Vendor:        "",
		}, nil
	}

	return result.(*models.VirtualizationInfo), nil
}

// detectLinux 检测Linux平台的虚拟化环境
// 检查 /sys/class/dmi/id/product_name 等文件
// 或使用 systemd-detect-virt 命令
// 返回:
//   - *models.VirtualizationInfo: 虚拟化信息
//   - error: 检测错误
func (vd *VirtualizationDetector) detectLinux() (*models.VirtualizationInfo, error) {
	// 方法1: 使用 systemd-detect-virt 命令
	cmd := exec.Command("systemd-detect-virt")
	output, err := cmd.Output()
	if err == nil {
		virtType := strings.TrimSpace(string(output))
		if virtType != "none" && virtType != "" {
			return &models.VirtualizationInfo{
				IsVirtualized: true,
				Type:          vd.normalizeVirtType(virtType),
				Vendor:        virtType,
			}, nil
		}
	}

	// 方法2: 检查 DMI 信息
	productName, _ := os.ReadFile("/sys/class/dmi/id/product_name")
	productNameStr := strings.ToLower(strings.TrimSpace(string(productName)))

	sysVendor, _ := os.ReadFile("/sys/class/dmi/id/sys_vendor")
	sysVendorStr := strings.ToLower(strings.TrimSpace(string(sysVendor)))

	// 检测常见虚拟化标识
	if strings.Contains(productNameStr, "kvm") || strings.Contains(sysVendorStr, "qemu") {
		return &models.VirtualizationInfo{
			IsVirtualized: true,
			Type:          "KVM",
			Vendor:        "QEMU/KVM",
		}, nil
	}

	if strings.Contains(productNameStr, "vmware") || strings.Contains(sysVendorStr, "vmware") {
		return &models.VirtualizationInfo{
			IsVirtualized: true,
			Type:          "VMware",
			Vendor:        "VMware, Inc.",
		}, nil
	}

	if strings.Contains(productNameStr, "virtualbox") || strings.Contains(sysVendorStr, "virtualbox") {
		return &models.VirtualizationInfo{
			IsVirtualized: true,
			Type:          "VirtualBox",
			Vendor:        "Oracle Corporation",
		}, nil
	}

	if strings.Contains(productNameStr, "xen") || strings.Contains(sysVendorStr, "xen") {
		return &models.VirtualizationInfo{
			IsVirtualized: true,
			Type:          "Xen",
			Vendor:        "Xen",
		}, nil
	}

	if strings.Contains(productNameStr, "microsoft") || strings.Contains(sysVendorStr, "microsoft") {
		return &models.VirtualizationInfo{
			IsVirtualized: true,
			Type:          "Hyper-V",
			Vendor:        "Microsoft Corporation",
		}, nil
	}

	// 方法3: 检查 /proc/cpuinfo
	cpuinfo, err := os.ReadFile("/proc/cpuinfo")
	if err == nil {
		cpuinfoStr := strings.ToLower(string(cpuinfo))
		if strings.Contains(cpuinfoStr, "hypervisor") {
			return &models.VirtualizationInfo{
				IsVirtualized: true,
				Type:          "Unknown",
				Vendor:        "Unknown Hypervisor",
			}, nil
		}
	}

	// 未检测到虚拟化
	return &models.VirtualizationInfo{
		IsVirtualized: false,
		Type:          "Physical",
		Vendor:        "",
	}, nil
}

// detectWindows 检测Windows平台的虚拟化环境
// 使用 WMI 查询检测虚拟化
// 返回:
//   - *models.VirtualizationInfo: 虚拟化信息
//   - error: 检测错误
func (vd *VirtualizationDetector) detectWindows() (*models.VirtualizationInfo, error) {
	// 使用 wmic 命令查询计算机系统信息
	cmd := exec.Command("wmic", "computersystem", "get", "Model,Manufacturer")
	output, err := cmd.Output()
	if err != nil {
		return &models.VirtualizationInfo{
			IsVirtualized: false,
			Type:          "Unknown",
			Vendor:        "",
		}, nil
	}

	outputStr := strings.ToLower(string(output))

	// 检测常见虚拟化标识
	if strings.Contains(outputStr, "vmware") {
		return &models.VirtualizationInfo{
			IsVirtualized: true,
			Type:          "VMware",
			Vendor:        "VMware, Inc.",
		}, nil
	}

	if strings.Contains(outputStr, "virtualbox") {
		return &models.VirtualizationInfo{
			IsVirtualized: true,
			Type:          "VirtualBox",
			Vendor:        "Oracle Corporation",
		}, nil
	}

	if strings.Contains(outputStr, "microsoft") && strings.Contains(outputStr, "virtual") {
		return &models.VirtualizationInfo{
			IsVirtualized: true,
			Type:          "Hyper-V",
			Vendor:        "Microsoft Corporation",
		}, nil
	}

	if strings.Contains(outputStr, "qemu") || strings.Contains(outputStr, "kvm") {
		return &models.VirtualizationInfo{
			IsVirtualized: true,
			Type:          "KVM",
			Vendor:        "QEMU/KVM",
		}, nil
	}

	if strings.Contains(outputStr, "xen") {
		return &models.VirtualizationInfo{
			IsVirtualized: true,
			Type:          "Xen",
			Vendor:        "Xen",
		}, nil
	}

	// 未检测到虚拟化
	return &models.VirtualizationInfo{
		IsVirtualized: false,
		Type:          "Physical",
		Vendor:        "",
	}, nil
}

// detectMacOS 检测macOS平台的虚拟化环境
// 使用 sysctl 命令检测虚拟化
// 返回:
//   - *models.VirtualizationInfo: 虚拟化信息
//   - error: 检测错误
func (vd *VirtualizationDetector) detectMacOS() (*models.VirtualizationInfo, error) {
	// 检查是否支持虚拟化（表示可能在虚拟机中）
	cmd := exec.Command("sysctl", "-n", "kern.hv_support")
	output, err := cmd.Output()
	if err == nil {
		hvSupport := strings.TrimSpace(string(output))
		if hvSupport == "0" {
			// 不支持虚拟化可能表示在虚拟机中
			return &models.VirtualizationInfo{
				IsVirtualized: true,
				Type:          "Unknown",
				Vendor:        "Unknown",
			}, nil
		}
	}

	// 检查硬件型号
	cmd = exec.Command("sysctl", "-n", "hw.model")
	output, err = cmd.Output()
	if err == nil {
		model := strings.ToLower(strings.TrimSpace(string(output)))

		if strings.Contains(model, "vmware") {
			return &models.VirtualizationInfo{
				IsVirtualized: true,
				Type:          "VMware",
				Vendor:        "VMware, Inc.",
			}, nil
		}

		if strings.Contains(model, "virtualbox") {
			return &models.VirtualizationInfo{
				IsVirtualized: true,
				Type:          "VirtualBox",
				Vendor:        "Oracle Corporation",
			}, nil
		}

		if strings.Contains(model, "parallels") {
			return &models.VirtualizationInfo{
				IsVirtualized: true,
				Type:          "Parallels",
				Vendor:        "Parallels",
			}, nil
		}
	}

	// 未检测到虚拟化
	return &models.VirtualizationInfo{
		IsVirtualized: false,
		Type:          "Physical",
		Vendor:        "",
	}, nil
}

// normalizeVirtType 标准化虚拟化类型名称
// 参数:
//   - virtType: 原始虚拟化类型字符串
//
// 返回:
//   - string: 标准化后的类型名称
func (vd *VirtualizationDetector) normalizeVirtType(virtType string) string {
	virtType = strings.ToLower(virtType)

	switch {
	case strings.Contains(virtType, "kvm"):
		return "KVM"
	case strings.Contains(virtType, "qemu"):
		return "KVM"
	case strings.Contains(virtType, "vmware"):
		return "VMware"
	case strings.Contains(virtType, "virtualbox"):
		return "VirtualBox"
	case strings.Contains(virtType, "xen"):
		return "Xen"
	case strings.Contains(virtType, "hyperv") || strings.Contains(virtType, "microsoft"):
		return "Hyper-V"
	case strings.Contains(virtType, "openvz"):
		return "OpenVZ"
	case strings.Contains(virtType, "lxc"):
		return "LXC"
	case strings.Contains(virtType, "docker"):
		return "Docker"
	default:
		return strings.Title(virtType)
	}
}

// IsVirtualized 判断是否运行在虚拟化环境中
// 返回:
//   - bool: true表示虚拟化环境
//   - error: 检测错误
func (vd *VirtualizationDetector) IsVirtualized() (bool, error) {
	info, err := vd.Detect()
	if err != nil {
		return false, err
	}
	return info.IsVirtualized, nil
}

// GetVirtualizationType 获取虚拟化类型
// 返回:
//   - string: 虚拟化类型
//   - error: 检测错误
func (vd *VirtualizationDetector) GetVirtualizationType() (string, error) {
	info, err := vd.Detect()
	if err != nil {
		return "Unknown", err
	}
	return info.Type, nil
}
