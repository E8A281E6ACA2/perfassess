// Package cli 提供命令行界面功能
// 使用 cobra 框架实现命令行参数解析和命令处理
package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"performance-assessment-system/internal/config"
	"performance-assessment-system/internal/controller"
	"performance-assessment-system/internal/doctor"
)

// CLI 命令行界面结构体
type CLI struct {
	rootCmd *cobra.Command
	config  *config.Config
}

// NewCLI 创建新的命令行界面
func NewCLI() *CLI {
	cfg := config.DefaultConfig()

	cli := &CLI{
		config: cfg,
	}

	cli.setupCommands()

	return cli
}

// RunInteractive 运行交互式模式
func (c *CLI) RunInteractive() error {
	menu := NewInteractiveMenu()
	cfg, err := menu.Show()
	if err != nil {
		return err
	}

	c.config = cfg

	// 创建评估控制器
	controller, err := controller.NewAssessmentController(c.config)
	if err != nil {
		return fmt.Errorf("创建评估控制器失败: %w", err)
	}

	// 确保资源清理
	defer func() {
		if err := controller.Cleanup(); err != nil {
			fmt.Fprintf(os.Stderr, "清理资源失败: %v\n", err)
		}
	}()

	// 启动评估会话
	session, err := controller.StartSession()
	if err != nil {
		return fmt.Errorf("启动评估会话失败: %w", err)
	}

	// 运行评估
	_, err = controller.RunAssessment(session)
	if err != nil {
		return fmt.Errorf("运行评估失败: %w", err)
	}

	return nil
}

// setupCommands 配置命令和参数
func (c *CLI) setupCommands() {
	c.rootCmd = &cobra.Command{
		Use:   "perfassess",
		Short: "高性能多终端自动化性能评估系统",
		Long: `高性能多终端自动化性能评估系统
		
一款用于自动化评估计算机和服务器性能的工具。
支持 CPU、内存、磁盘、网络等多项性能测试。
支持 Linux、Windows、macOS 多平台。`,
		RunE: c.run,
	}
	c.rootCmd.AddCommand(c.newCheckDepsCommand())

	// 添加命令行参数
	flags := c.rootCmd.Flags()

	// --interactive 参数：启用交互式模式
	flags.BoolP("interactive", "i", false,
		"启用交互式菜单模式（推荐新手使用）")

	// --benchmarks 参数：指定要运行的检测项目
	flags.StringSliceP("benchmarks", "b", []string{"all"},
		"指定要运行的检测项目 (cpu,memory,disk,network,all)")

	// 保留 --tests 作为别名，向后兼容
	flags.StringSliceP("tests", "t", []string{},
		"(已弃用，请使用 --benchmarks) 指定要运行的测试类型")

	// --output 参数：指定输出文件路径
	flags.StringP("output", "o", "",
		"指定输出文件路径（不指定则只输出到控制台）")

	// --verbose 参数：启用详细输出模式
	flags.BoolP("verbose", "v", false,
		"启用详细输出模式")

	// --route-trace 参数：启用路由追踪
	flags.Bool("route-trace", false,
		"启用路由追踪功能")

	// --streaming 参数：启用流媒体检测
	flags.Bool("streaming", false,
		"启用流媒体解锁检测功能")

	// --ai-services 参数：启用 AI 服务检测
	flags.Bool("ai-services", false,
		"启用 AI 服务可用性检测功能")

	// --stress 参数：启用长时间压力测试
	flags.Bool("stress", false,
		"启用长时间压力测试（耗时较长）")

	// --security 参数：启用安全体检
	flags.Bool("security", false,
		"启用安全/加固体检（端口扫描、SSH 配置等）")

	// --network-backend 参数：选择网络测试后端
	flags.String("network-backend", "builtin",
		"网络测试后端 (builtin,iperf3)")

	// --iperf3-server 参数：iperf3 服务端地址
	flags.String("iperf3-server", "",
		"iperf3 服务端地址（仅 network-backend=iperf3 时使用）")

	// --web 参数：启用 Web 报告
	flags.Bool("web", false,
		"启用 Web 报告服务器")

	// --port 参数：Web 服务器端口
	flags.Int("port", 8080,
		"Web 服务器端口 (默认: 8080)")

	// --log-level 参数：设置日志级别
	flags.String("log-level", "info",
		"设置日志级别 (debug,info,warn,error)")

	// 绑定参数到配置
	c.rootCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		return c.bindFlags(cmd)
	}
}

func (c *CLI) newCheckDepsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "check-deps",
		Short: "检查外部测试工具依赖",
		RunE: func(cmd *cobra.Command, args []string) error {
			statuses := doctor.CheckDependencies()
			fmt.Println("外部依赖检查:")
			fmt.Println()

			missing := 0
			for _, status := range statuses {
				if status.Found {
					fmt.Printf("[OK] %s - %s (%s)\n", status.Name, status.Purpose, status.Path)
					continue
				}

				missing++
				fmt.Printf("[缺失] %s - %s\n", status.Name, status.Purpose)
				fmt.Printf("       %s\n", status.Hint)
			}

			if missing > 0 {
				fmt.Println()
				fmt.Printf("发现 %d 个缺失依赖。程序不会自动安装，请按提示手动安装后重试。\n", missing)
			}
			return nil
		},
	}
}

// bindFlags 绑定命令行参数到配置
func (c *CLI) bindFlags(cmd *cobra.Command) error {
	flags := cmd.Flags()

	// 绑定 benchmarks 参数（优先）
	if benchmarks, err := flags.GetStringSlice("benchmarks"); err == nil && len(benchmarks) > 0 {
		c.config.Tests = benchmarks
	} else if tests, err := flags.GetStringSlice("tests"); err == nil && len(tests) > 0 {
		// 向后兼容 --tests 参数
		c.config.Tests = tests
	}

	// 绑定 output 参数
	if output, err := flags.GetString("output"); err == nil {
		c.config.Output = output
	}

	// 绑定 verbose 参数
	if verbose, err := flags.GetBool("verbose"); err == nil {
		c.config.Verbose = verbose
		if verbose {
			c.config.LogLevel = "debug"
		}
	}

	// 绑定 route-trace 参数
	if routeTrace, err := flags.GetBool("route-trace"); err == nil {
		c.config.EnableRouteTrace = routeTrace
	}

	// 绑定 streaming 参数
	if streaming, err := flags.GetBool("streaming"); err == nil {
		c.config.EnableStreaming = streaming
	}

	// 绑定 ai-services 参数
	if aiServices, err := flags.GetBool("ai-services"); err == nil {
		c.config.EnableAIServices = aiServices
	}

	// 绑定 stress 参数
	if stress, err := flags.GetBool("stress"); err == nil {
		c.config.EnableStressTest = stress
	}

	// 绑定 security 参数
	if security, err := flags.GetBool("security"); err == nil {
		c.config.EnableSecurityScan = security
	}

	// 绑定 network-backend 参数
	if networkBackend, err := flags.GetString("network-backend"); err == nil {
		c.config.NetworkBackend = networkBackend
	}

	// 绑定 iperf3-server 参数
	if iperf3Server, err := flags.GetString("iperf3-server"); err == nil {
		c.config.Iperf3Server = iperf3Server
	}

	// 绑定 log-level 参数
	if logLevel, err := flags.GetString("log-level"); err == nil {
		c.config.LogLevel = logLevel
	}

	// 绑定 web 参数
	if web, err := flags.GetBool("web"); err == nil {
		c.config.EnableWeb = web
	}

	// 绑定 port 参数
	if port, err := flags.GetInt("port"); err == nil {
		c.config.WebPort = port
	}

	return nil
}

// run 执行主命令
func (c *CLI) run(cmd *cobra.Command, args []string) error {
	// 检查是否使用交互式模式
	interactive, _ := cmd.Flags().GetBool("interactive")
	if interactive {
		return c.RunInteractive()
	}

	// 验证参数
	if err := c.validateFlags(); err != nil {
		return err
	}

	// 显示欢迎信息
	c.printWelcome()

	// 创建评估控制器
	controller, err := controller.NewAssessmentController(c.config)
	if err != nil {
		return fmt.Errorf("创建评估控制器失败: %w", err)
	}

	// 确保资源清理
	defer func() {
		if err := controller.Cleanup(); err != nil {
			fmt.Fprintf(os.Stderr, "清理资源失败: %v\n", err)
		}
	}()

	// 启动评估会话
	session, err := controller.StartSession()
	if err != nil {
		return fmt.Errorf("启动评估会话失败: %w", err)
	}

	// 运行评估
	_, err = controller.RunAssessment(session)
	if err != nil {
		return fmt.Errorf("运行评估失败: %w", err)
	}

	return nil
}

// validateFlags 验证命令行参数
func (c *CLI) validateFlags() error {
	// 验证测试类型
	validTests := map[string]bool{
		"cpu":     true,
		"memory":  true,
		"disk":    true,
		"network": true,
		"all":     true,
	}

	for _, test := range c.config.Tests {
		if !validTests[test] {
			return fmt.Errorf("无效的测试类型: %s\n有效的测试类型: cpu, memory, disk, network, all", test)
		}
	}

	// 验证日志级别
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLogLevels[c.config.LogLevel] {
		return fmt.Errorf("无效的日志级别: %s\n有效的日志级别: debug, info, warn, error", c.config.LogLevel)
	}

	return nil
}

// printWelcome 打印欢迎信息
func (c *CLI) printWelcome() {
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║          高性能多终端自动化性能评估系统                       ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 显示配置信息
	if c.config.Verbose {
		fmt.Printf("检测项目: %s\n", strings.Join(c.config.Tests, ", "))
		if c.config.Output != "" {
			fmt.Printf("输出文件: %s\n", c.config.Output)
		}
		if c.config.EnableRouteTrace {
			fmt.Println("路由追踪: 已启用")
		}
		if c.config.EnableStreaming {
			fmt.Println("流媒体检测: 已启用")
		}
		if c.config.EnableAIServices {
			fmt.Println("AI 服务检测: 已启用")
		}
		if c.config.EnableStressTest {
			fmt.Println("压力测试: 已启用")
		}
		if c.config.EnableSecurityScan {
			fmt.Println("安全体检: 已启用")
		}
		fmt.Println()
	}
}

// Execute 执行命令行界面
func (c *CLI) Execute() error {
	return c.rootCmd.Execute()
}

// GetRootCmd 获取根命令（用于测试）
func (c *CLI) GetRootCmd() *cobra.Command {
	return c.rootCmd
}
