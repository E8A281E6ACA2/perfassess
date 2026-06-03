// Package config 提供配置管理功能
// 负责加载、解析和管理应用程序的配置参数
package config

import "fmt"

const (
	DefaultCPUWeight     = 0.30
	DefaultMemoryWeight  = 0.20
	DefaultDiskWeight    = 0.25
	DefaultNetworkWeight = 0.25
	DefaultScoreProfile  = "server"
)

// Config 表示应用程序的配置结构
// 包含所有可配置的参数和选项，对应命令行参数和配置文件
type Config struct {
	// Tests 要执行的测试类型列表
	// 可选值: "cpu", "memory", "disk", "network", "all"
	// 多个测试类型用逗号分隔，如 "cpu,memory"
	Tests []string `mapstructure:"tests"`

	// Output 输出文件路径
	// 如果为空，则只输出到控制台
	Output string `mapstructure:"output"`

	// OutputFormat 输出格式，可选值: text, json
	OutputFormat string `mapstructure:"output_format"`

	// ScoreWeights 综合评分权重，键为 cpu/memory/disk/network
	ScoreWeights map[string]float64 `mapstructure:"score_weights"`

	// ScoreProfile 评分基准档位，可选值: vps, server, workstation
	ScoreProfile string `mapstructure:"score_profile"`

	// Verbose 是否启用详细输出模式
	// 启用后会显示更多调试信息
	Verbose bool `mapstructure:"verbose"`

	// EnableRouteTrace 是否启用路由追踪功能
	EnableRouteTrace bool `mapstructure:"enable_route_trace"`

	// EnableStreaming 是否启用流媒体检测功能
	EnableStreaming bool `mapstructure:"enable_streaming"`

	// EnableAIServices 是否启用AI服务检测功能
	EnableAIServices bool `mapstructure:"enable_ai_services"`

	// EnableStressTest 是否启用长时间压力测试
	EnableStressTest bool `mapstructure:"enable_stress_test"`

	// EnableSecurityScan 是否启用安全体检
	EnableSecurityScan bool `mapstructure:"enable_security_scan"`

	// RouteTraceTargets 路由追踪的目标地址列表
	// 默认包含常用的公共服务器地址
	RouteTraceTargets []string `mapstructure:"route_trace_targets"`

	// CPUBackend CPU 测试后端，可选值: builtin, sysbench, geekbench
	CPUBackend string `mapstructure:"cpu_backend"`

	// MemoryBackend 内存测试后端，可选值: builtin, sysbench
	MemoryBackend string `mapstructure:"memory_backend"`

	// NetworkBackend 网络测试后端，可选值: builtin, iperf3, speedtest
	NetworkBackend string `mapstructure:"network_backend"`

	// Iperf3Server iperf3 服务端地址，仅在 network_backend=iperf3 时使用
	Iperf3Server string `mapstructure:"iperf3_server"`

	// Iperf3Servers iperf3 多服务端地址列表，仅在 network_backend=iperf3 时使用
	Iperf3Servers []string `mapstructure:"iperf3_servers"`

	// DiskBackend 磁盘测试后端，可选值: builtin, fio
	DiskBackend string `mapstructure:"disk_backend"`

	// LogLevel 日志级别（debug/info/warn/error）
	LogLevel string `mapstructure:"log_level"`

	// GeoIPDBPath GeoIP 数据库文件路径（可选）
	// 用于IP地理位置查询
	GeoIPDBPath string `mapstructure:"geoip_db_path"`

	// EnableWeb 是否启用 Web 报告服务器
	EnableWeb bool `mapstructure:"enable_web"`

	// WebPort Web 服务器端口
	WebPort int `mapstructure:"web_port"`
}

// DefaultConfig 返回默认配置
// 当配置文件不存在或某些参数未设置时使用
func DefaultConfig() *Config {
	return &Config{
		Tests:              []string{"all"},
		Output:             "",
		OutputFormat:       "text",
		ScoreWeights:       DefaultScoreWeights(),
		ScoreProfile:       DefaultScoreProfile,
		Verbose:            false,
		EnableRouteTrace:   false,
		EnableStreaming:    false,
		EnableAIServices:   false,
		EnableStressTest:   false,
		EnableSecurityScan: false,
		RouteTraceTargets:  []string{"8.8.8.8", "1.1.1.1", "cloudflare.com"},
		CPUBackend:         "builtin",
		MemoryBackend:      "builtin",
		NetworkBackend:     "builtin",
		Iperf3Server:       "",
		Iperf3Servers:      []string{},
		DiskBackend:        "builtin",
		LogLevel:           "info",
		GeoIPDBPath:        "",
		EnableWeb:          false,
		WebPort:            8080,
	}
}

func DefaultScoreWeights() map[string]float64 {
	return map[string]float64{
		"cpu":     DefaultCPUWeight,
		"memory":  DefaultMemoryWeight,
		"disk":    DefaultDiskWeight,
		"network": DefaultNetworkWeight,
	}
}

// Validate 验证配置的有效性
// 检查配置参数是否合法，返回错误信息
func (c *Config) Validate() error {
	// 验证测试类型
	validTests := map[string]bool{
		"cpu":     true,
		"memory":  true,
		"disk":    true,
		"network": true,
		"all":     true,
	}

	for _, test := range c.Tests {
		if !validTests[test] {
			return &ConfigError{
				Field:   "tests",
				Message: "无效的测试类型: " + test,
			}
		}
	}

	// 验证日志级别
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLogLevels[c.LogLevel] {
		return &ConfigError{
			Field:   "log_level",
			Message: "无效的日志级别: " + c.LogLevel,
		}
	}

	validOutputFormats := map[string]bool{
		"text": true,
		"json": true,
	}
	if !validOutputFormats[c.OutputFormat] {
		return &ConfigError{
			Field:   "output_format",
			Message: "无效的输出格式: " + c.OutputFormat,
		}
	}

	if err := ValidateScoreWeights(c.ScoreWeights); err != nil {
		return err
	}

	if !IsValidScoreProfile(c.ScoreProfile) {
		return &ConfigError{
			Field:   "score_profile",
			Message: "无效的评分基准档位: " + c.ScoreProfile,
		}
	}

	validCPUBackends := map[string]bool{
		"builtin":   true,
		"sysbench":  true,
		"geekbench": true,
	}
	if !validCPUBackends[c.CPUBackend] {
		return &ConfigError{
			Field:   "cpu_backend",
			Message: "无效的 CPU 测试后端: " + c.CPUBackend,
		}
	}

	validMemoryBackends := map[string]bool{
		"builtin":  true,
		"sysbench": true,
	}
	if !validMemoryBackends[c.MemoryBackend] {
		return &ConfigError{
			Field:   "memory_backend",
			Message: "无效的内存测试后端: " + c.MemoryBackend,
		}
	}

	validNetworkBackends := map[string]bool{
		"builtin":   true,
		"iperf3":    true,
		"speedtest": true,
	}
	if !validNetworkBackends[c.NetworkBackend] {
		return &ConfigError{
			Field:   "network_backend",
			Message: "无效的网络测试后端: " + c.NetworkBackend,
		}
	}

	validDiskBackends := map[string]bool{
		"builtin": true,
		"fio":     true,
	}
	if !validDiskBackends[c.DiskBackend] {
		return &ConfigError{
			Field:   "disk_backend",
			Message: "无效的磁盘测试后端: " + c.DiskBackend,
		}
	}

	return nil
}

func IsValidScoreProfile(profile string) bool {
	switch profile {
	case "vps", "server", "workstation":
		return true
	default:
		return false
	}
}

func ValidateScoreWeights(weights map[string]float64) error {
	if len(weights) == 0 {
		return &ConfigError{
			Field:   "score_weights",
			Message: "评分权重不能为空",
		}
	}

	required := []string{"cpu", "memory", "disk", "network"}
	total := 0.0
	for _, key := range required {
		value, ok := weights[key]
		if !ok {
			return &ConfigError{
				Field:   "score_weights",
				Message: "缺少评分权重: " + key,
			}
		}
		if value < 0 {
			return &ConfigError{
				Field:   "score_weights",
				Message: fmt.Sprintf("评分权重不能为负数: %s=%.4f", key, value),
			}
		}
		total += value
	}

	for key := range weights {
		known := false
		for _, requiredKey := range required {
			if key == requiredKey {
				known = true
				break
			}
		}
		if !known {
			return &ConfigError{
				Field:   "score_weights",
				Message: "未知评分权重: " + key,
			}
		}
	}

	if total <= 0 {
		return &ConfigError{
			Field:   "score_weights",
			Message: "评分权重总和必须大于 0",
		}
	}

	return nil
}

// ConfigError 表示配置错误
type ConfigError struct {
	// Field 出错的配置字段名
	Field string

	// Message 错误消息
	Message string
}

// Error 实现error接口
func (e *ConfigError) Error() string {
	return "配置错误 [" + e.Field + "]: " + e.Message
}
