// Package tests 提供路由追踪测试功能
package tests

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
	"github.com/E8A281E6ACA2/perfassess/pkg/logger"
)

// RouteTracer 路由追踪器
// 负责执行路由追踪测试
type RouteTracer struct {
	// logger 日志记录器
	logger *logger.Logger

	// maxHops 最大跳数
	maxHops int

	// timeout 单次追踪超时时间
	timeout time.Duration
}

// NewRouteTracer 创建新的路由追踪器
func NewRouteTracer(logger *logger.Logger) *RouteTracer {
	return &RouteTracer{
		logger:  logger,
		maxHops: 30,
		timeout: 30 * time.Second,
	}
}

// Trace 执行单个目标的路由追踪
// 参数:
//   - target: 目标地址（IP或域名）
//
// 返回:
//   - *models.TraceResult: 追踪结果
//   - error: 追踪错误
func (rt *RouteTracer) Trace(target string) (*models.TraceResult, error) {
	rt.logger.Info(fmt.Sprintf("开始路由追踪: %s", target))

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), rt.timeout)
	defer cancel()

	// 根据操作系统选择命令
	var cmd *exec.Cmd
	var bin string
	var args []string
	switch runtime.GOOS {
	case "windows":
		bin = "tracert"
		args = []string{"-h", strconv.Itoa(rt.maxHops), target}
	case "darwin", "linux":
		bin = "traceroute"
		args = []string{"-m", strconv.Itoa(rt.maxHops), "-w", "2", target}
	default:
		return nil, fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}

	if _, err := exec.LookPath(bin); err != nil {
		instruction := dependencyHint(bin)
		message := fmt.Sprintf("%s 未安装。%s", bin, instruction)
		rt.logger.Warn(message)
		return &models.TraceResult{
			Target:       target,
			Success:      false,
			ErrorMessage: message,
		}, nil
	}

	cmd = exec.CommandContext(ctx, bin, args...)

	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 检查是否是超时
		if ctx.Err() == context.DeadlineExceeded {
			rt.logger.Warn(fmt.Sprintf("路由追踪超时: %s", target))
			return &models.TraceResult{
				Target:       target,
				Success:      false,
				ErrorMessage: "追踪超时",
			}, nil
		}

		// 其他错误
		rt.logger.Error(fmt.Sprintf("路由追踪失败: %s", target), err)
		return &models.TraceResult{
			Target:       target,
			Success:      false,
			ErrorMessage: err.Error(),
		}, err
	}

	// 解析输出
	result := rt.parseOutput(target, string(output))

	rt.logger.Info(fmt.Sprintf("路由追踪完成: %s, 总跳数: %d", target, result.TotalHops))

	return result, nil
}

// TraceMultiple 并发追踪多个目标
// 参数:
//   - targets: 目标地址列表
//
// 返回:
//   - []*models.TraceResult: 所有追踪结果
//   - error: 追踪错误
func (rt *RouteTracer) TraceMultiple(targets []string) ([]*models.TraceResult, error) {
	if len(targets) == 0 {
		return nil, fmt.Errorf("目标列表为空")
	}

	rt.logger.Info(fmt.Sprintf("开始并发路由追踪，目标数量: %d", len(targets)))

	// 创建结果通道
	resultChan := make(chan *models.TraceResult, len(targets))
	errorChan := make(chan error, len(targets))

	// 并发执行追踪
	for _, target := range targets {
		go func(t string) {
			result, err := rt.Trace(t)
			if err != nil {
				errorChan <- err
			} else {
				resultChan <- result
			}
		}(target)
	}

	// 收集结果
	results := make([]*models.TraceResult, 0, len(targets))
	successCount := 0
	for i := 0; i < len(targets); i++ {
		select {
		case result := <-resultChan:
			results = append(results, result)
			if result != nil && result.Success {
				successCount++
			}
		case err := <-errorChan:
			rt.logger.Error("路由追踪错误", err)
			// 继续收集其他结果
		}
	}

	rt.logger.Info(fmt.Sprintf("并发路由追踪完成，成功: %d/%d", successCount, len(targets)))

	return results, nil
}

// parseOutput 解析traceroute/tracert输出
func (rt *RouteTracer) parseOutput(target string, output string) *models.TraceResult {
	result := &models.TraceResult{
		Target:  target,
		Hops:    make([]*models.Hop, 0),
		Success: true,
	}

	lines := strings.Split(output, "\n")

	// 根据操作系统使用不同的解析逻辑
	if runtime.GOOS == "windows" {
		result = rt.parseWindowsOutput(target, lines)
	} else {
		result = rt.parseUnixOutput(target, lines)
	}

	result.TotalHops = len(result.Hops)

	return result
}

// parseWindowsOutput 解析Windows tracert输出
func (rt *RouteTracer) parseWindowsOutput(target string, lines []string) *models.TraceResult {
	result := &models.TraceResult{
		Target:  target,
		Hops:    make([]*models.Hop, 0),
		Success: true,
	}

	// Windows tracert 输出格式:
	// 1    <1 ms    <1 ms    <1 ms  192.168.1.1
	// 2     5 ms     4 ms     5 ms  10.0.0.1

	hopRegex := regexp.MustCompile(`^\s*(\d+)\s+(?:(<?\d+)\s*ms|\*)\s+(?:(<?\d+)\s*ms|\*)\s+(?:(<?\d+)\s*ms|\*)\s+(.+)$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := hopRegex.FindStringSubmatch(line)
		if len(matches) >= 6 {
			hopNum, _ := strconv.Atoi(matches[1])

			// 提取IP地址
			ipPart := strings.TrimSpace(matches[5])
			ip := rt.extractIP(ipPart)

			// 计算平均延迟
			var latency time.Duration
			latencyCount := 0
			for i := 2; i <= 4; i++ {
				if matches[i] != "" && matches[i] != "*" {
					ms, err := strconv.Atoi(strings.TrimPrefix(matches[i], "<"))
					if err == nil {
						latency += time.Duration(ms) * time.Millisecond
						latencyCount++
					}
				}
			}
			if latencyCount > 0 {
				latency = latency / time.Duration(latencyCount)
			}

			hop := &models.Hop{
				Number:  hopNum,
				IP:      ip,
				Latency: latency,
			}

			result.Hops = append(result.Hops, hop)
		}
	}

	return result
}

// parseUnixOutput 解析Unix/Linux/macOS traceroute输出
func (rt *RouteTracer) parseUnixOutput(target string, lines []string) *models.TraceResult {
	result := &models.TraceResult{
		Target:  target,
		Hops:    make([]*models.Hop, 0),
		Success: true,
	}

	// Unix traceroute 输出格式:
	// 1  192.168.1.1 (192.168.1.1)  0.345 ms  0.298 ms  0.287 ms
	// 2  10.0.0.1 (10.0.0.1)  5.123 ms  4.987 ms  5.234 ms

	hopRegex := regexp.MustCompile(`^\s*(\d+)\s+(.+?)(?:\s+\(([^\)]+)\))?\s+([\d\.]+)\s*ms`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "traceroute") {
			continue
		}

		matches := hopRegex.FindStringSubmatch(line)
		if len(matches) >= 5 {
			hopNum, _ := strconv.Atoi(matches[1])

			// 提取IP地址
			ip := matches[3]
			if ip == "" {
				ip = matches[2]
			}
			ip = rt.extractIP(ip)

			// 提取延迟
			latencyMs, _ := strconv.ParseFloat(matches[4], 64)
			latency := time.Duration(latencyMs * float64(time.Millisecond))

			hop := &models.Hop{
				Number:  hopNum,
				IP:      ip,
				Latency: latency,
			}

			result.Hops = append(result.Hops, hop)
		} else if strings.Contains(line, "*") {
			// 处理超时的跳
			hopRegex2 := regexp.MustCompile(`^\s*(\d+)\s+\*`)
			matches2 := hopRegex2.FindStringSubmatch(line)
			if len(matches2) >= 2 {
				hopNum, _ := strconv.Atoi(matches2[1])
				hop := &models.Hop{
					Number:  hopNum,
					IP:      "*",
					Latency: 0,
				}
				result.Hops = append(result.Hops, hop)
			}
		}
	}

	return result
}

func dependencyHint(bin string) string {
	switch runtime.GOOS {
	case "linux":
		return "请使用 sudo apt install traceroute 或 sudo yum install traceroute 安装该工具。"
	case "darwin":
		return "macOS 自带 traceroute，如不可用请确认系统 PATH。"
	case "windows":
		if bin == "tracert" {
			return "Windows 自带 tracert，请确认系统 PATH 或管理员策略未禁用。"
		}
	}
	return "请安装系统自带的路由追踪工具后重试。"
}

// extractIP 从字符串中提取IP地址
func (rt *RouteTracer) extractIP(s string) string {
	// 尝试解析为IP地址
	if ip := net.ParseIP(s); ip != nil {
		return ip.String()
	}

	// 尝试从括号中提取IP
	if strings.Contains(s, "(") && strings.Contains(s, ")") {
		start := strings.Index(s, "(")
		end := strings.Index(s, ")")
		if start < end {
			ipStr := s[start+1 : end]
			if ip := net.ParseIP(ipStr); ip != nil {
				return ip.String()
			}
		}
	}

	// 尝试提取第一个看起来像IP的部分
	ipRegex := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
	if match := ipRegex.FindString(s); match != "" {
		return match
	}

	return s
}

// ParseHop 解析单个跳的数据（用于自定义解析）
// 参数:
//   - hopData: 跳数据字符串
//
// 返回:
//   - *models.Hop: 解析后的跳信息
//   - error: 解析错误
func (rt *RouteTracer) ParseHop(hopData string) (*models.Hop, error) {
	// 这是一个辅助方法，用于解析自定义格式的跳数据
	// 格式: "序号 IP 延迟ms"
	parts := strings.Fields(hopData)
	if len(parts) < 3 {
		return nil, fmt.Errorf("无效的跳数据格式: %s", hopData)
	}

	number, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("无效的跳序号: %s", parts[0])
	}

	ip := parts[1]

	latencyMs, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return nil, fmt.Errorf("无效的延迟值: %s", parts[2])
	}

	return &models.Hop{
		Number:  number,
		IP:      ip,
		Latency: time.Duration(latencyMs * float64(time.Millisecond)),
	}, nil
}
