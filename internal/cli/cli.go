// Package cli 提供命令行界面功能
// 使用 cobra 框架实现命令行参数解析和命令处理
package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"performance-assessment-system/internal/compare"
	"performance-assessment-system/internal/config"
	"performance-assessment-system/internal/controller"
	"performance-assessment-system/internal/doctor"
	"performance-assessment-system/internal/history"
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
	c.rootCmd.AddCommand(c.newCompareCommand())
	c.rootCmd.AddCommand(c.newCompareDirCommand())
	c.rootCmd.AddCommand(c.newHistoryCommand())

	// 添加命令行参数
	flags := c.rootCmd.Flags()

	// --interactive 参数：启用交互式模式
	flags.BoolP("interactive", "i", false,
		"启用交互式菜单模式")

	// --benchmarks 参数：指定要运行的检测项目
	flags.StringSliceP("benchmarks", "b", []string{"all"},
		"指定要运行的检测项目 (cpu,memory,disk,network,all)")

	// --quick 参数：快速预设
	flags.Bool("quick", false,
		"快速预设：只运行 CPU、内存、磁盘基础测试")

	// --full 参数：完整预设
	flags.Bool("full", false,
		"完整预设：运行基础测试并启用可选检查；提供 iperf3 服务端时使用 iperf3")

	// 保留 --tests 作为别名，向后兼容
	flags.StringSliceP("tests", "t", []string{},
		"(已弃用，请使用 --benchmarks) 指定要运行的测试类型")

	// --output 参数：指定输出文件路径
	flags.StringP("output", "o", "",
		"指定输出文件路径（不指定则只输出到控制台）")

	// --output-format 参数：指定输出格式
	flags.String("output-format", "text",
		"指定输出格式 (text,json)")

	// --score-weights 参数：指定综合评分权重
	flags.String("score-weights", "",
		"综合评分权重，如 cpu=0.3,memory=0.2,disk=0.25,network=0.25")

	// --score-profile 参数：指定评分基准档位
	flags.String("score-profile", config.DefaultScoreProfile,
		"评分基准档位 (vps,server,workstation)")

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
		"网络测试后端 (builtin,iperf3,speedtest)")

	// --iperf3-server 参数：iperf3 服务端地址
	flags.String("iperf3-server", "",
		"iperf3 服务端地址（仅 network-backend=iperf3 时使用）")

	// --disk-backend 参数：选择磁盘测试后端
	flags.String("disk-backend", "builtin",
		"磁盘测试后端 (builtin,fio)")

	// --cpu-backend 参数：选择 CPU 测试后端
	flags.String("cpu-backend", "builtin",
		"CPU 测试后端 (builtin,sysbench,geekbench)")

	// --memory-backend 参数：选择内存测试后端
	flags.String("memory-backend", "builtin",
		"内存测试后端 (builtin,sysbench)")

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

func (c *CLI) newCompareCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compare <report-a.json> <report-b.json>",
		Short: "对比两份 JSON 评估报告",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			format, _ := cmd.Flags().GetString("format")
			if format != "text" && format != "json" {
				return fmt.Errorf("无效的对比输出格式: %s\n有效的输出格式: text, json", format)
			}

			result, err := compare.CompareFiles(args[0], args[1])
			if err != nil {
				return err
			}

			if format == "json" {
				content, err := compare.FormatJSON(result)
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), content)
				return nil
			}

			fmt.Fprint(cmd.OutOrStdout(), compare.FormatText(result))
			return nil
		},
	}
	cmd.Flags().String("format", "text", "对比输出格式 (text,json)")
	return cmd
}

func (c *CLI) newCompareDirCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compare-dir <reports-dir>",
		Short: "批量排序目录中的 JSON 评估报告",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			format, _ := cmd.Flags().GetString("format")
			if err := validateOutputFormat(format); err != nil {
				return err
			}
			sortBy, _ := cmd.Flags().GetString("sort-by")
			entries, err := history.EntriesFromDir(args[0])
			if err != nil {
				return err
			}
			if err := history.SortEntries(entries, sortBy, true); err != nil {
				return err
			}
			if format == "json" {
				content, err := history.FormatJSON(entries)
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), content)
				return nil
			}
			fmt.Fprint(cmd.OutOrStdout(), history.FormatRankText(entries, sortBy))
			return nil
		},
	}
	cmd.Flags().String("format", "text", "输出格式 (text,json)")
	cmd.Flags().String("sort-by", "total", "排序字段 (total,cpu,memory,disk,network)")
	return cmd
}

func (c *CLI) newHistoryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "管理本地 JSON 报告历史库",
	}
	cmd.AddCommand(c.newHistoryAddCommand())
	cmd.AddCommand(c.newHistoryListCommand())
	cmd.AddCommand(c.newHistoryTrendCommand())
	return cmd
}

func (c *CLI) newHistoryAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <report.json>",
		Short: "加入一份 JSON 报告到本地历史库",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, _ := cmd.Flags().GetString("store")
			entry, err := history.AddReport(store, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "已加入历史库: %s | total=%.2f | host=%s\n", entry.ReportPath, entry.TotalScore, entry.HostID)
			return nil
		},
	}
	cmd.Flags().String("store", history.DefaultStorePath(), "历史库 JSONL 路径")
	return cmd
}

func (c *CLI) newHistoryListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出本地历史库中的报告",
		RunE: func(cmd *cobra.Command, args []string) error {
			format, _ := cmd.Flags().GetString("format")
			if err := validateOutputFormat(format); err != nil {
				return err
			}
			store, _ := cmd.Flags().GetString("store")
			entries, err := history.LoadEntries(store)
			if err != nil {
				return err
			}
			if format == "json" {
				content, err := history.FormatJSON(entries)
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), content)
				return nil
			}
			fmt.Fprint(cmd.OutOrStdout(), history.FormatEntriesText(entries))
			return nil
		},
	}
	cmd.Flags().String("store", history.DefaultStorePath(), "历史库 JSONL 路径")
	cmd.Flags().String("format", "text", "输出格式 (text,json)")
	return cmd
}

func (c *CLI) newHistoryTrendCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trend",
		Short: "查看本地历史库中的评分趋势",
		RunE: func(cmd *cobra.Command, args []string) error {
			format, _ := cmd.Flags().GetString("format")
			if err := validateOutputFormat(format); err != nil {
				return err
			}
			store, _ := cmd.Flags().GetString("store")
			hostID, _ := cmd.Flags().GetString("host")
			entries, err := history.LoadEntries(store)
			if err != nil {
				return err
			}
			trend, err := history.BuildTrend(entries, hostID)
			if err != nil {
				return err
			}
			if format == "json" {
				content, err := history.FormatJSON(trend)
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), content)
				return nil
			}
			fmt.Fprint(cmd.OutOrStdout(), history.FormatTrendText(trend))
			return nil
		},
	}
	cmd.Flags().String("store", history.DefaultStorePath(), "历史库 JSONL 路径")
	cmd.Flags().String("host", "", "按 host_id 过滤；为空时使用全部历史")
	cmd.Flags().String("format", "text", "输出格式 (text,json)")
	return cmd
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
				requiredText := "可选"
				if status.Required {
					requiredText = "必需"
				}
				if status.Found {
					fmt.Printf("[OK] [%s] %s - %s (%s)\n", requiredText, status.Name, status.Purpose, status.Path)
					continue
				}

				missing++
				fmt.Printf("[缺失] [%s] %s - %s\n", requiredText, status.Name, status.Purpose)
				fmt.Printf("       %s\n", status.Hint)
			}

			if missing > 0 {
				fmt.Println()
				fmt.Printf("发现 %d 个缺失依赖。程序不会自动安装，请按提示手动安装后重试。\n", missing)
				fmt.Println("说明：缺失可选依赖不会影响默认一把梭，但会影响 --cpu-backend sysbench、--cpu-backend geekbench、--memory-backend sysbench、--disk-backend fio、--network-backend iperf3、--network-backend speedtest 或 --route-trace。")
			}
			return nil
		},
	}
}

// bindFlags 绑定命令行参数到配置
func (c *CLI) bindFlags(cmd *cobra.Command) error {
	flags := cmd.Flags()

	quick, _ := flags.GetBool("quick")
	full, _ := flags.GetBool("full")
	if quick {
		c.applyQuickPreset()
	}
	if full {
		c.applyFullPreset()
	}

	// 绑定 benchmarks 参数（优先）
	if benchmarks, err := flags.GetStringSlice("benchmarks"); err == nil && flags.Changed("benchmarks") && len(benchmarks) > 0 {
		c.config.Tests = benchmarks
	} else if tests, err := flags.GetStringSlice("tests"); err == nil && flags.Changed("tests") && len(tests) > 0 {
		// 向后兼容 --tests 参数
		c.config.Tests = tests
	}

	// 绑定 output 参数
	if output, err := flags.GetString("output"); err == nil {
		c.config.Output = output
	}

	// 绑定 output-format 参数
	if outputFormat, err := flags.GetString("output-format"); err == nil {
		c.config.OutputFormat = outputFormat
	}

	// 绑定 score-weights 参数
	if scoreWeights, err := flags.GetString("score-weights"); err == nil && flags.Changed("score-weights") {
		weights, err := parseScoreWeights(scoreWeights)
		if err != nil {
			return err
		}
		c.config.ScoreWeights = weights
	}

	// 绑定 score-profile 参数
	if scoreProfile, err := flags.GetString("score-profile"); err == nil {
		c.config.ScoreProfile = scoreProfile
	}

	// 绑定 verbose 参数
	if verbose, err := flags.GetBool("verbose"); err == nil && flags.Changed("verbose") {
		c.config.Verbose = verbose
		if verbose {
			c.config.LogLevel = "debug"
		}
	}

	// 绑定 route-trace 参数
	if routeTrace, err := flags.GetBool("route-trace"); err == nil && flags.Changed("route-trace") {
		c.config.EnableRouteTrace = routeTrace
	}

	// 绑定 streaming 参数
	if streaming, err := flags.GetBool("streaming"); err == nil && flags.Changed("streaming") {
		c.config.EnableStreaming = streaming
	}

	// 绑定 ai-services 参数
	if aiServices, err := flags.GetBool("ai-services"); err == nil && flags.Changed("ai-services") {
		c.config.EnableAIServices = aiServices
	}

	// 绑定 stress 参数
	if stress, err := flags.GetBool("stress"); err == nil && flags.Changed("stress") {
		c.config.EnableStressTest = stress
	}

	// 绑定 security 参数
	if security, err := flags.GetBool("security"); err == nil && flags.Changed("security") {
		c.config.EnableSecurityScan = security
	}

	// 绑定 network-backend 参数
	if networkBackend, err := flags.GetString("network-backend"); err == nil && flags.Changed("network-backend") {
		c.config.NetworkBackend = networkBackend
	}

	// 绑定 iperf3-server 参数
	if iperf3Server, err := flags.GetString("iperf3-server"); err == nil {
		c.config.Iperf3Server = iperf3Server
	}
	if full && c.config.Iperf3Server != "" && !flags.Changed("network-backend") {
		c.config.NetworkBackend = "iperf3"
	}

	// 绑定 disk-backend 参数
	if diskBackend, err := flags.GetString("disk-backend"); err == nil && flags.Changed("disk-backend") {
		c.config.DiskBackend = diskBackend
	}

	// 绑定 cpu-backend 参数
	if cpuBackend, err := flags.GetString("cpu-backend"); err == nil && flags.Changed("cpu-backend") {
		c.config.CPUBackend = cpuBackend
	}

	// 绑定 memory-backend 参数
	if memoryBackend, err := flags.GetString("memory-backend"); err == nil && flags.Changed("memory-backend") {
		c.config.MemoryBackend = memoryBackend
	}

	// 绑定 log-level 参数
	if logLevel, err := flags.GetString("log-level"); err == nil {
		c.config.LogLevel = logLevel
	}

	// 绑定 web 参数
	if web, err := flags.GetBool("web"); err == nil && flags.Changed("web") {
		c.config.EnableWeb = web
	}

	// 绑定 port 参数
	if port, err := flags.GetInt("port"); err == nil && flags.Changed("port") {
		c.config.WebPort = port
	}

	return nil
}

func (c *CLI) applyQuickPreset() {
	c.config.Tests = []string{"cpu", "memory", "disk"}
	c.config.EnableRouteTrace = false
	c.config.EnableStreaming = false
	c.config.EnableAIServices = false
	c.config.EnableStressTest = false
	c.config.EnableSecurityScan = false
	c.config.DiskBackend = "builtin"
	c.config.CPUBackend = "builtin"
	c.config.MemoryBackend = "builtin"
	c.config.NetworkBackend = "builtin"
}

func (c *CLI) applyFullPreset() {
	c.config.Tests = []string{"all"}
	c.config.EnableRouteTrace = true
	c.config.EnableStreaming = true
	c.config.EnableAIServices = true
	c.config.EnableSecurityScan = true
	c.config.CPUBackend = "sysbench"
	c.config.MemoryBackend = "sysbench"
	c.config.DiskBackend = "fio"
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

	validOutputFormats := map[string]bool{
		"text": true,
		"json": true,
	}
	if !validOutputFormats[c.config.OutputFormat] {
		return fmt.Errorf("无效的输出格式: %s\n有效的输出格式: text, json", c.config.OutputFormat)
	}

	if err := config.ValidateScoreWeights(c.config.ScoreWeights); err != nil {
		return err
	}
	if !config.IsValidScoreProfile(c.config.ScoreProfile) {
		return fmt.Errorf("无效的评分基准档位: %s\n有效的评分基准档位: vps, server, workstation", c.config.ScoreProfile)
	}

	validCPUBackends := map[string]bool{
		"builtin":   true,
		"sysbench":  true,
		"geekbench": true,
	}
	if !validCPUBackends[c.config.CPUBackend] {
		return fmt.Errorf("无效的 CPU 测试后端: %s\n有效的 CPU 测试后端: builtin, sysbench, geekbench", c.config.CPUBackend)
	}

	validMemoryBackends := map[string]bool{
		"builtin":  true,
		"sysbench": true,
	}
	if !validMemoryBackends[c.config.MemoryBackend] {
		return fmt.Errorf("无效的内存测试后端: %s\n有效的内存测试后端: builtin, sysbench", c.config.MemoryBackend)
	}

	validDiskBackends := map[string]bool{
		"builtin": true,
		"fio":     true,
	}
	if !validDiskBackends[c.config.DiskBackend] {
		return fmt.Errorf("无效的磁盘测试后端: %s\n有效的磁盘测试后端: builtin, fio", c.config.DiskBackend)
	}

	validNetworkBackends := map[string]bool{
		"builtin":   true,
		"iperf3":    true,
		"speedtest": true,
	}
	if !validNetworkBackends[c.config.NetworkBackend] {
		return fmt.Errorf("无效的网络测试后端: %s\n有效的网络测试后端: builtin, iperf3, speedtest", c.config.NetworkBackend)
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
		fmt.Printf("输出格式: %s\n", c.config.OutputFormat)
		fmt.Printf("评分权重: CPU=%.2f, 内存=%.2f, 磁盘=%.2f, 网络=%.2f\n",
			c.config.ScoreWeights["cpu"],
			c.config.ScoreWeights["memory"],
			c.config.ScoreWeights["disk"],
			c.config.ScoreWeights["network"],
		)
		fmt.Printf("评分基准: %s\n", c.config.ScoreProfile)
		fmt.Printf("CPU测试后端: %s\n", c.config.CPUBackend)
		fmt.Printf("内存测试后端: %s\n", c.config.MemoryBackend)
		fmt.Printf("磁盘测试后端: %s\n", c.config.DiskBackend)
		fmt.Printf("网络测试后端: %s\n", c.config.NetworkBackend)
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

func parseScoreWeights(value string) (map[string]float64, error) {
	weights := config.DefaultScoreWeights()
	if strings.TrimSpace(value) == "" {
		return weights, nil
	}

	seen := map[string]bool{}
	for _, part := range strings.Split(value, ",") {
		pair := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(pair) != 2 {
			return nil, fmt.Errorf("无效的评分权重格式: %s", part)
		}
		key := strings.TrimSpace(pair[0])
		raw := strings.TrimSpace(pair[1])
		parsed, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("无效的评分权重数值: %s=%s", key, raw)
		}
		weights[key] = parsed
		seen[key] = true
	}

	required := []string{"cpu", "memory", "disk", "network"}
	for _, key := range required {
		if !seen[key] {
			return nil, fmt.Errorf("评分权重必须显式包含 %s", key)
		}
	}
	if err := config.ValidateScoreWeights(weights); err != nil {
		return nil, err
	}
	return weights, nil
}

func validateOutputFormat(format string) error {
	if format != "text" && format != "json" {
		return fmt.Errorf("无效的输出格式: %s\n有效的输出格式: text, json", format)
	}
	return nil
}

// Execute 执行命令行界面
func (c *CLI) Execute() error {
	return c.rootCmd.Execute()
}

// GetRootCmd 获取根命令（用于测试）
func (c *CLI) GetRootCmd() *cobra.Command {
	return c.rootCmd
}
