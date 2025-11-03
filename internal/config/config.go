// Package config 提供配置管理功能
// 负责加载、解析和管理应用程序的配置参数
package config

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
	
	// Verbose 是否启用详细输出模式
	// 启用后会显示更多调试信息
	Verbose bool `mapstructure:"verbose"`
	
	// EnableRouteTrace 是否启用路由追踪功能
	EnableRouteTrace bool `mapstructure:"enable_route_trace"`
	
	// EnableStreaming 是否启用流媒体检测功能
	EnableStreaming bool `mapstructure:"enable_streaming"`
	
	// RouteTraceTargets 路由追踪的目标地址列表
	// 默认包含常用的公共服务器地址
	RouteTraceTargets []string `mapstructure:"route_trace_targets"`
	
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
		Tests:             []string{"all"},
		Output:            "",
		Verbose:           false,
		EnableRouteTrace:  false,
		EnableStreaming:   false,
		RouteTraceTargets: []string{"8.8.8.8", "1.1.1.1", "cloudflare.com"},
		LogLevel:          "info",
		GeoIPDBPath:       "",
		EnableWeb:         false,
		WebPort:           8080,
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
