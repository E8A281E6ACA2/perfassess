// Package collector 提供系统信息收集功能
package collector

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"

	"performance-assessment-system/internal/models"
	"performance-assessment-system/internal/platform"
	"performance-assessment-system/pkg/utils"
)

// SystemInfoCollector 系统信息收集器
// 负责收集CPU、内存、磁盘和操作系统信息
type SystemInfoCollector struct {
	// adapter 平台适配器，提供平台特定的命令
	adapter platform.PlatformAdapter

	// timeout 收集超时时间
	timeout time.Duration
}

// NewSystemInfoCollector 创建系统信息收集器
// 参数:
//   - adapter: 平台适配器
//
// 返回:
//   - *SystemInfoCollector: 系统信息收集器实例
func NewSystemInfoCollector(adapter platform.PlatformAdapter) *SystemInfoCollector {
	return &SystemInfoCollector{
		adapter: adapter,
		timeout: 5 * time.Second, // 默认5秒超时
	}
}

// CollectAll 收集所有系统信息
// 在5秒内完成所有信息的收集
// 返回:
//   - *models.SystemInfo: 系统信息
//   - error: 收集错误
func (sic *SystemInfoCollector) CollectAll() (*models.SystemInfo, error) {
	ctx := context.Background()

	// 使用超时控制确保在5秒内完成
	result, err := utils.RunWithTimeoutAndResult(ctx, sic.timeout, func() (interface{}, error) {
		systemInfo := &models.SystemInfo{
			CollectionTime: time.Now(),
		}

		// 收集CPU信息
		cpuInfo, err := sic.CollectCPUInfo()
		if err != nil {
			return nil, utils.WrapError(err, "收集CPU信息失败")
		}
		systemInfo.CPU = cpuInfo

		// 收集内存信息
		memInfo, err := sic.CollectMemoryInfo()
		if err != nil {
			return nil, utils.WrapError(err, "收集内存信息失败")
		}
		systemInfo.Memory = memInfo

		// 收集磁盘信息
		diskInfo, err := sic.CollectDiskInfo()
		if err != nil {
			return nil, utils.WrapError(err, "收集磁盘信息失败")
		}
		systemInfo.Disk = diskInfo

		// 收集操作系统信息
		osInfo, err := sic.CollectOSInfo()
		if err != nil {
			return nil, utils.WrapError(err, "收集操作系统信息失败")
		}
		systemInfo.OS = osInfo

		return systemInfo, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(*models.SystemInfo), nil
}

// CollectCPUInfo 收集CPU信息
// 使用gopsutil库获取CPU型号、核心数、线程数和频率
// 返回:
//   - *models.CPUInfo: CPU信息
//   - error: 收集错误
func (sic *SystemInfoCollector) CollectCPUInfo() (*models.CPUInfo, error) {
	// 获取CPU信息
	cpuInfos, err := cpu.Info()
	if err != nil {
		return nil, fmt.Errorf("获取CPU信息失败: %w", err)
	}

	if len(cpuInfos) == 0 {
		return nil, fmt.Errorf("未找到CPU信息")
	}

	// 获取物理核心数
	physicalCores, err := cpu.Counts(false)
	if err != nil {
		physicalCores = runtime.NumCPU() // 降级使用runtime
	}

	// 获取逻辑核心数（线程数）
	logicalCores, err := cpu.Counts(true)
	if err != nil {
		logicalCores = runtime.NumCPU()
	}

	// 使用第一个CPU的信息
	firstCPU := cpuInfos[0]

	return &models.CPUInfo{
		Model:        firstCPU.ModelName,
		Cores:        physicalCores,
		Threads:      logicalCores,
		FrequencyMHz: firstCPU.Mhz,
	}, nil
}

// CollectMemoryInfo 收集内存信息
// 使用gopsutil库获取内存总量和可用量
// 返回:
//   - *models.MemoryInfo: 内存信息
//   - error: 收集错误
func (sic *SystemInfoCollector) CollectMemoryInfo() (*models.MemoryInfo, error) {
	// 获取虚拟内存信息
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("获取内存信息失败: %w", err)
	}

	// 转换为MB
	totalMB := int64(vmStat.Total / 1024 / 1024)
	availableMB := int64(vmStat.Available / 1024 / 1024)

	// 内存类型检测（简化版本，实际可能需要更复杂的检测）
	memType := "Unknown"
	// 注意：gopsutil v3 可能不直接提供内存类型
	// 这里使用占位符，后续可以通过平台特定命令获取

	return &models.MemoryInfo{
		TotalMB:     totalMB,
		AvailableMB: availableMB,
		MemoryType:  memType,
	}, nil
}

// CollectDiskInfo 收集磁盘信息
// 使用gopsutil库获取磁盘容量和可用空间
// 返回:
//   - *models.DiskInfo: 磁盘信息
//   - error: 收集错误
func (sic *SystemInfoCollector) CollectDiskInfo() (*models.DiskInfo, error) {
	// 获取根分区的使用情况
	// Windows: C:\, Linux/macOS: /
	var path string
	if runtime.GOOS == "windows" {
		path = "C:\\"
	} else {
		path = "/"
	}

	usage, err := disk.Usage(path)
	if err != nil {
		return nil, fmt.Errorf("获取磁盘信息失败: %w", err)
	}

	// 转换为GB
	totalGB := float64(usage.Total) / 1024 / 1024 / 1024
	availableGB := float64(usage.Free) / 1024 / 1024 / 1024

	// 磁盘类型检测（简化版本）
	diskType := "Unknown"
	// 注意：准确的磁盘类型检测需要平台特定的实现
	// 这里使用占位符

	return &models.DiskInfo{
		TotalGB:     totalGB,
		AvailableGB: availableGB,
		DiskType:    diskType,
	}, nil
}

// CollectOSInfo 收集操作系统信息
// 使用gopsutil库获取操作系统名称、版本和架构
// 返回:
//   - *models.OSInfo: 操作系统信息
//   - error: 收集错误
func (sic *SystemInfoCollector) CollectOSInfo() (*models.OSInfo, error) {
	// 获取主机信息
	hostInfo, err := host.Info()
	if err != nil {
		return nil, fmt.Errorf("获取操作系统信息失败: %w", err)
	}

	// 构建操作系统名称
	osName := hostInfo.Platform
	if hostInfo.PlatformFamily != "" {
		osName = hostInfo.PlatformFamily
	}

	// 构建版本信息
	version := hostInfo.PlatformVersion
	if hostInfo.KernelVersion != "" {
		version = fmt.Sprintf("%s (Kernel: %s)", version, hostInfo.KernelVersion)
	}

	return &models.OSInfo{
		Name:         osName,
		Version:      version,
		Architecture: hostInfo.KernelArch,
	}, nil
}

// SetTimeout 设置收集超时时间
// 参数:
//   - timeout: 超时时间
func (sic *SystemInfoCollector) SetTimeout(timeout time.Duration) {
	sic.timeout = timeout
}
