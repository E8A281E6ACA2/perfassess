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
		return rt.enrichResult(&models.TraceResult{
			Target:       target,
			Success:      false,
			ErrorMessage: message,
		}), nil
	}

	cmd = exec.CommandContext(ctx, bin, args...)

	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 检查是否是超时
		if ctx.Err() == context.DeadlineExceeded {
			rt.logger.Warn(fmt.Sprintf("路由追踪超时: %s", target))
			return rt.enrichResult(&models.TraceResult{
				Target:       target,
				Success:      false,
				ErrorMessage: "追踪超时",
			}), nil
		}

		// 其他错误
		rt.logger.Error(fmt.Sprintf("路由追踪失败: %s", target), err)
		return rt.enrichResult(&models.TraceResult{
			Target:       target,
			Success:      false,
			ErrorMessage: err.Error(),
		}), err
	}

	// 解析输出
	result := rt.parseOutput(target, string(output))
	result = rt.enrichResult(result)

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

func (rt *RouteTracer) enrichResult(result *models.TraceResult) *models.TraceResult {
	if result == nil {
		return result
	}
	result.DirectionGroup = routeDirectionGroup(result.Target)
	result.IsRealReturnRoute = false
	result.TotalHops = len(result.Hops)
	result.LastVisibleHop = routeLastVisibleHop(result)
	result.TimeoutHops = routeTimeoutHops(result)
	result.AverageLatencyMs = routeAverageLatencyMs(result)
	result.Quality = buildRouteQuality(result)
	result.Evidence = buildRouteEvidence(result)
	result.Recommendations = buildRouteRecommendations(result)
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

func buildRouteQuality(result *models.TraceResult) *models.RouteQuality {
	grade := routeQualityGrade(result)
	status := routeQualityStatus(result)
	parts := []string{
		routeDirectionLabel(result.DirectionGroup),
		fmt.Sprintf("评级 %s", grade),
	}
	if result.Success {
		parts = append(parts, fmt.Sprintf("%d 跳", result.TotalHops))
		if result.AverageLatencyMs > 0 {
			parts = append(parts, fmt.Sprintf("平均 %.2f ms", result.AverageLatencyMs))
		}
		if result.TimeoutHops > 0 {
			parts = append(parts, fmt.Sprintf("超时跳 %d", result.TimeoutHops))
		}
		if result.LastVisibleHop != "" {
			parts = append(parts, fmt.Sprintf("末跳 %s", result.LastVisibleHop))
		}
	} else if result.ErrorMessage != "" {
		parts = append(parts, result.ErrorMessage)
	}
	return &models.RouteQuality{
		Grade:          grade,
		Status:         status,
		Summary:        strings.Join(parts, "；"),
		DirectionLabel: routeDirectionLabel(result.DirectionGroup),
	}
}

func buildRouteEvidence(result *models.TraceResult) []*models.RouteEvidence {
	if result == nil {
		return nil
	}
	evidence := []*models.RouteEvidence{
		{Name: "方向", Value: routeDirectionLabel(result.DirectionGroup), Status: "info", Detail: "基于目标域名/IP 分组"},
		{Name: "真实回程", Value: yesNoRoute(result.IsRealReturnRoute), Status: "warning", Detail: "内置 traceroute 只能表示本机出站路径"},
		{Name: "状态", Value: successRouteLabel(result.Success), Status: routeSuccessStatus(result.Success), Detail: result.ErrorMessage},
		{Name: "跳数", Value: fmt.Sprintf("%d", result.TotalHops), Status: routeHopStatus(result.TotalHops)},
		{Name: "超时跳", Value: fmt.Sprintf("%d", result.TimeoutHops), Status: routeTimeoutStatus(result)},
	}
	if result.AverageLatencyMs > 0 {
		evidence = append(evidence, &models.RouteEvidence{
			Name:   "平均延迟",
			Value:  fmt.Sprintf("%.2f ms", result.AverageLatencyMs),
			Status: routeLatencyStatus(result.AverageLatencyMs),
		})
	}
	if result.LastVisibleHop != "" {
		evidence = append(evidence, &models.RouteEvidence{Name: "最后可见跳", Value: result.LastVisibleHop, Status: "info"})
	}
	return evidence
}

func buildRouteRecommendations(result *models.TraceResult) []string {
	if result == nil {
		return nil
	}
	recommendations := []string{}
	if !result.IsRealReturnRoute {
		recommendations = append(recommendations, "该结果是本机出站路径参考，不是真实回程；真实回程需要远端探针或第三方平台配合。")
	}
	if !result.Success {
		recommendations = append(recommendations, "该目标路由追踪失败，建议确认 traceroute/tracert 依赖、ICMP/UDP 策略或目标网络是否限制。")
	}
	if result.TimeoutHops > 0 && result.TotalHops > 0 {
		recommendations = append(recommendations, "存在不可见跳点，可能是中间路由屏蔽探测包，不一定代表链路不可用。")
	}
	if result.TotalHops > 20 {
		recommendations = append(recommendations, "跳数偏多，跨区域访问可能经过较长路径，建议结合实际业务地区评估。")
	}
	if result.AverageLatencyMs > 200 {
		recommendations = append(recommendations, "平均延迟偏高，建议结合 TCP 延迟、丢包和业务目标地区继续验证。")
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "当前路由追踪未发现明显异常，建议结合目标业务的实际访问体验确认。")
	}
	return recommendations
}

func routeDirectionGroup(target string) string {
	lowered := strings.ToLower(strings.TrimSpace(target))
	chinaTargets := []string{
		"189.cn", "10086.cn", "chinaunicom.com.cn", "ctyun.cn", "qq.com",
		"baidu.com", "aliyun.com", "tencent.com", "chinatelecom", "cmcc",
	}
	for _, marker := range chinaTargets {
		if strings.Contains(lowered, marker) {
			return "china_reference"
		}
	}
	return "public"
}

func routeDirectionLabel(group string) string {
	switch group {
	case "china_reference":
		return "国内方向参考"
	default:
		return "公共方向"
	}
}

func routeQualityGrade(result *models.TraceResult) string {
	if result == nil || !result.Success {
		return "D"
	}
	timeoutRatio := 0.0
	if result.TotalHops > 0 {
		timeoutRatio = float64(result.TimeoutHops) / float64(result.TotalHops)
	}
	switch {
	case result.TimeoutHops == 0 && result.TotalHops <= 12 && result.AverageLatencyMs > 0 && result.AverageLatencyMs <= 80:
		return "A"
	case timeoutRatio <= 0.2 && result.TotalHops <= 18 && (result.AverageLatencyMs == 0 || result.AverageLatencyMs <= 160):
		return "B"
	case timeoutRatio <= 0.4 && result.TotalHops <= 24 && (result.AverageLatencyMs == 0 || result.AverageLatencyMs <= 260):
		return "C"
	default:
		return "D"
	}
}

func routeQualityStatus(result *models.TraceResult) string {
	switch routeQualityGrade(result) {
	case "A", "B":
		return "success"
	case "C":
		return "warning"
	default:
		return "failed"
	}
}

func routeLastVisibleHop(result *models.TraceResult) string {
	if result == nil {
		return ""
	}
	for i := len(result.Hops) - 1; i >= 0; i-- {
		hop := result.Hops[i]
		if hop == nil || hop.IP == "" || hop.IP == "*" {
			continue
		}
		if hop.Hostname != "" {
			return fmt.Sprintf("%s (%s)", hop.IP, hop.Hostname)
		}
		return hop.IP
	}
	return ""
}

func routeTimeoutHops(result *models.TraceResult) int {
	if result == nil {
		return 0
	}
	count := 0
	for _, hop := range result.Hops {
		if hop == nil || hop.IP == "" || hop.IP == "*" {
			count++
		}
	}
	return count
}

func routeAverageLatencyMs(result *models.TraceResult) float64 {
	if result == nil {
		return 0
	}
	total := time.Duration(0)
	count := 0
	for _, hop := range result.Hops {
		if hop == nil || hop.Latency <= 0 {
			continue
		}
		total += hop.Latency
		count++
	}
	if count == 0 {
		return 0
	}
	return float64(total/time.Duration(count)) / float64(time.Millisecond)
}

func successRouteLabel(success bool) string {
	if success {
		return "成功"
	}
	return "失败"
}

func yesNoRoute(value bool) string {
	if value {
		return "是"
	}
	return "否"
}

func routeSuccessStatus(success bool) string {
	if success {
		return "success"
	}
	return "failed"
}

func routeHopStatus(hops int) string {
	switch {
	case hops == 0:
		return "unknown"
	case hops <= 12:
		return "good"
	case hops <= 20:
		return "medium"
	default:
		return "high"
	}
}

func routeTimeoutStatus(result *models.TraceResult) string {
	if result == nil || result.TotalHops == 0 {
		return "unknown"
	}
	ratio := float64(result.TimeoutHops) / float64(result.TotalHops)
	switch {
	case result.TimeoutHops == 0:
		return "clean"
	case ratio <= 0.2:
		return "partial"
	default:
		return "warning"
	}
}

func routeLatencyStatus(avgMs float64) string {
	switch {
	case avgMs <= 0:
		return "unknown"
	case avgMs <= 80:
		return "good"
	case avgMs <= 180:
		return "medium"
	default:
		return "high"
	}
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
