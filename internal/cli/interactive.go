// Package cli 提供交互式菜单功能
package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"performance-assessment-system/internal/config"
)

// InteractiveMenu 交互式菜单
type InteractiveMenu struct {
	reader *bufio.Reader
	config *config.Config
}

// NewInteractiveMenu 创建交互式菜单
func NewInteractiveMenu() *InteractiveMenu {
	return &InteractiveMenu{
		reader: bufio.NewReader(os.Stdin),
		config: config.DefaultConfig(),
	}
}

// Show 显示交互式菜单并获取用户选择
func (im *InteractiveMenu) Show() (*config.Config, error) {
	im.printWelcome()

	// 选择测试类型
	if err := im.selectTests(); err != nil {
		return nil, err
	}

	// 选择可选功能
	if err := im.selectOptionalFeatures(); err != nil {
		return nil, err
	}

	// 选择输出选项
	if err := im.selectOutputOptions(); err != nil {
		return nil, err
	}

	// 确认配置
	if !im.confirmConfiguration() {
		fmt.Println("\n已取消测试。")
		os.Exit(0)
	}

	return im.config, nil
}

// printWelcome 打印欢迎信息
func (im *InteractiveMenu) printWelcome() {
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║          高性能多终端自动化性能评估系统                       ║")
	fmt.Println("║          Performance Assessment System v1.0                    ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("欢迎使用性能评估系统！")
	fmt.Println("本工具将帮助您全面评估系统性能。")
	fmt.Println()
}

// selectTests 选择要运行的测试
func (im *InteractiveMenu) selectTests() error {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("【步骤 1/2】选择检测项目")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("请选择要运行的检测项目：")
	fmt.Println()
	fmt.Println("  1. 完整检测（推荐）  - 运行所有性能测试")
	fmt.Println("  2. CPU 性能测试      - 测试处理器单核和多核性能")
	fmt.Println("  3. 内存性能测试      - 测试内存读写速度")
	fmt.Println("  4. 磁盘性能测试      - 测试磁盘I/O性能")
	fmt.Println("  5. 网络性能测试      - 测试网络延迟和带宽")
	fmt.Println("  6. 路由追踪测试      - 测试到主要地区的网络路由")
	fmt.Println("  7. 流媒体解锁检测    - 检测流媒体平台访问情况")
	fmt.Println("  8. AI 服务检测       - 检测主流 AI 服务访问情况")
	fmt.Println("  9. 长时间压力测试    - 持续高压运行，观察稳定性")
	fmt.Println(" 10. 安全体检          - 端口扫描与 SSH 配置检查")
	fmt.Println(" 11. 自定义组合        - 自由选择多个检测项目")
	fmt.Println()
	fmt.Print("请输入选项 [1-11] (默认: 1): ")

	choice, err := im.readLine()
	if err != nil {
		return err
	}

	choice = strings.TrimSpace(choice)
	if choice == "" {
		choice = "1"
	}

	switch choice {
	case "1":
		im.config.Tests = []string{"all"}
		fmt.Println("✓ 已选择：完整检测（包含所有性能测试）")
	case "2":
		im.config.Tests = []string{"cpu"}
		fmt.Println("✓ 已选择：CPU 性能测试")
	case "3":
		im.config.Tests = []string{"memory"}
		fmt.Println("✓ 已选择：内存性能测试")
	case "4":
		im.config.Tests = []string{"disk"}
		fmt.Println("✓ 已选择：磁盘性能测试")
	case "5":
		im.config.Tests = []string{"network"}
		fmt.Println("✓ 已选择：网络性能测试")
	case "6":
		im.config.EnableRouteTrace = true
		im.config.Tests = []string{} // 只做路由追踪
		fmt.Println("✓ 已选择：路由追踪测试")
	case "7":
		im.config.EnableStreaming = true
		im.config.Tests = []string{} // 只做流媒体检测
		fmt.Println("✓ 已选择：流媒体解锁检测")
	case "8":
		im.config.EnableAIServices = true
		im.config.Tests = []string{} // 只做AI检测
		fmt.Println("✓ 已选择：AI 服务检测")
	case "9":
		im.config.EnableStressTest = true
		im.config.Tests = []string{}
		fmt.Println("✓ 已选择：长时间压力测试")
	case "10":
		im.config.EnableSecurityScan = true
		im.config.Tests = []string{}
		fmt.Println("✓ 已选择：安全体检")
	case "11":
		return im.selectCustomTests()
	default:
		fmt.Println("⚠ 无效选项，使用默认：完整检测")
		im.config.Tests = []string{"all"}
	}

	fmt.Println()
	return nil
}

// selectCustomTests 自定义选择测试
func (im *InteractiveMenu) selectCustomTests() error {
	fmt.Println()
	fmt.Println("请选择要运行的检测项目（可多选，用空格分隔数字）：")
	fmt.Println()
	fmt.Println("  1 - CPU 性能测试")
	fmt.Println("  2 - 内存性能测试")
	fmt.Println("  3 - 磁盘性能测试")
	fmt.Println("  4 - 网络性能测试")
	fmt.Println("  5 - 路由追踪测试")
	fmt.Println("  6 - 流媒体解锁检测")
	fmt.Println("  7 - AI 服务检测")
	fmt.Println("  8 - 长时间压力测试")
	fmt.Println("  9 - 安全体检")
	fmt.Println()
	fmt.Print("请输入选项 (例如: 1 2 3 或 1 2 6 7 8 9): ")

	input, err := im.readLine()
	if err != nil {
		return err
	}

	input = strings.TrimSpace(input)
	if input == "" {
		im.config.Tests = []string{"all"}
		fmt.Println("✓ 未选择，使用默认：完整检测")
		return nil
	}

	tests := []string{}
	parts := strings.Fields(input)

	for _, part := range parts {
		switch part {
		case "1":
			tests = append(tests, "cpu")
		case "2":
			tests = append(tests, "memory")
		case "3":
			tests = append(tests, "disk")
		case "4":
			tests = append(tests, "network")
		case "5":
			im.config.EnableRouteTrace = true
		case "6":
			im.config.EnableStreaming = true
		case "7":
			im.config.EnableAIServices = true
		case "8":
			im.config.EnableStressTest = true
		case "9":
			im.config.EnableSecurityScan = true
		}
	}

	if len(tests) == 0 && !im.config.EnableRouteTrace && !im.config.EnableStreaming && !im.config.EnableAIServices && !im.config.EnableStressTest && !im.config.EnableSecurityScan {
		im.config.Tests = []string{"all"}
		fmt.Println("✓ 无效选择，使用默认：完整检测")
	} else {
		im.config.Tests = tests

		// 显示选择的项目
		selectedItems := []string{}
		if len(tests) > 0 {
			selectedItems = append(selectedItems, tests...)
		}
		if im.config.EnableRouteTrace {
			selectedItems = append(selectedItems, "路由追踪")
		}
		if im.config.EnableStreaming {
			selectedItems = append(selectedItems, "流媒体检测")
		}
		if im.config.EnableAIServices {
			selectedItems = append(selectedItems, "AI 服务检测")
		}
		if im.config.EnableStressTest {
			selectedItems = append(selectedItems, "长时间压力测试")
		}
		if im.config.EnableSecurityScan {
			selectedItems = append(selectedItems, "安全体检")
		}
		fmt.Printf("✓ 已选择：%s\n", strings.Join(selectedItems, ", "))
	}

	fmt.Println()
	return nil
}

// selectOptionalFeatures 选择可选功能（已整合到步骤1，此方法保留为空）
func (im *InteractiveMenu) selectOptionalFeatures() error {
	// 路由追踪和流媒体检测已经在步骤1中选择
	// 这个方法保留是为了保持代码结构，但不再需要用户交互
	return nil
}

// selectOutputOptions 选择输出选项
func (im *InteractiveMenu) selectOutputOptions() error {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("【步骤 2/2】输出选项")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// 详细输出
	fmt.Println("是否启用详细输出模式？")
	fmt.Println("  1. 是 - 显示更多调试信息")
	fmt.Println("  2. 否（默认）- 标准输出")
	fmt.Print("请选择 [1-2]: ")
	verbose, err := im.readLine()
	if err != nil {
		return err
	}
	verbose = strings.TrimSpace(verbose)
	im.config.Verbose = verbose == "1"
	if im.config.Verbose {
		im.config.LogLevel = "debug"
		fmt.Println("✓ 已启用：详细输出模式")
	} else {
		fmt.Println("✗ 使用标准输出模式")
	}
	fmt.Println()

	// 保存到文件
	fmt.Println("是否保存报告到文件？")
	fmt.Println("  1. 是 - 保存到文件")
	fmt.Println("  2. 否（默认）- 仅显示在终端")
	fmt.Print("请选择 [1-2]: ")
	saveFile, err := im.readLine()
	if err != nil {
		return err
	}
	saveFile = strings.TrimSpace(saveFile)

	if saveFile == "1" {
		fmt.Print("请输入文件名 (直接回车使用默认: report.txt): ")
		filename, err := im.readLine()
		if err != nil {
			return err
		}
		filename = strings.TrimSpace(filename)
		if filename == "" {
			filename = "report.txt"
		}
		im.config.Output = filename
		fmt.Printf("✓ 报告将保存到：%s\n", filename)
	} else {
		fmt.Println("✗ 报告仅显示在终端")
	}
	fmt.Println()

	// Web 报告
	fmt.Println("是否启用 Web 报告服务器？")
	fmt.Println("  1. 是 - 启动 Web 服务器查看报告")
	fmt.Println("  2. 否（默认）")
	fmt.Print("请选择 [1-2]: ")
	webChoice, err := im.readLine()
	if err != nil {
		return err
	}
	webChoice = strings.TrimSpace(webChoice)

	if webChoice == "1" {
		im.config.EnableWeb = true
		fmt.Print("请输入端口号 (直接回车使用默认: 8080): ")
		portStr, err := im.readLine()
		if err != nil {
			return err
		}
		portStr = strings.TrimSpace(portStr)
		if portStr != "" {
			var port int
			if _, err := fmt.Sscanf(portStr, "%d", &port); err == nil && port > 0 && port < 65536 {
				im.config.WebPort = port
			}
		}
		fmt.Printf("✓ Web 服务器将启动在端口：%d\n", im.config.WebPort)
	} else {
		fmt.Println("✗ 不启用 Web 报告")
	}

	fmt.Println()
	return nil
}

// confirmConfiguration 确认配置
func (im *InteractiveMenu) confirmConfiguration() bool {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("【配置确认】")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("您的测试配置：")
	fmt.Printf("  检测项目:     %s\n", strings.Join(im.config.Tests, ", "))
	fmt.Printf("  路由追踪:     %s\n", im.boolToString(im.config.EnableRouteTrace))
	fmt.Printf("  流媒体检测:   %s\n", im.boolToString(im.config.EnableStreaming))
	fmt.Printf("  AI 服务检测:  %s\n", im.boolToString(im.config.EnableAIServices))
	fmt.Printf("  压力测试:     %s\n", im.boolToString(im.config.EnableStressTest))
	fmt.Printf("  安全体检:     %s\n", im.boolToString(im.config.EnableSecurityScan))
	fmt.Printf("  详细输出:     %s\n", im.boolToString(im.config.Verbose))
	if im.config.Output != "" {
		fmt.Printf("  输出文件:     %s\n", im.config.Output)
	}
	if im.config.EnableWeb {
		fmt.Printf("  Web 服务器:   端口 %d\n", im.config.WebPort)
	}
	fmt.Println()
	fmt.Println("确认开始测试？")
	fmt.Println("  1. 确认开始（默认）")
	fmt.Println("  2. 取消")
	fmt.Print("请选择 [1-2]: ")

	confirm, err := im.readLine()
	if err != nil {
		return false
	}

	confirm = strings.TrimSpace(confirm)
	return confirm == "" || confirm == "1"
}

// readLine 读取一行输入
func (im *InteractiveMenu) readLine() (string, error) {
	line, err := im.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// boolToString 将布尔值转换为中文字符串
func (im *InteractiveMenu) boolToString(b bool) string {
	if b {
		return "是"
	}
	return "否"
}
