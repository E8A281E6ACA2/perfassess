package tests

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/host"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
	"github.com/E8A281E6ACA2/perfassess/pkg/logger"
)

// SecurityScanner 执行基础安全体检
type SecurityScanner struct {
	logger      *logger.Logger
	targetHost  string
	portList    []int
	sshConfPath string
}

// NewSecurityScanner 创建安全扫描器
func NewSecurityScanner(logger *logger.Logger, systemInfo *models.SystemInfo) *SecurityScanner {
	target := "127.0.0.1"
	if systemInfo != nil && systemInfo.IPInfo != nil && systemInfo.IPInfo.PublicIP != "" {
		target = systemInfo.IPInfo.PublicIP
	}

	return &SecurityScanner{
		logger:      logger,
		targetHost:  target,
		portList:    []int{21, 22, 25, 53, 80, 110, 143, 443, 465, 587, 8080, 8443, 873, 1433, 1521, 2049, 2375, 3306, 3389, 5000, 5432, 5672, 5900, 6379, 7001, 8081, 8888, 9000, 9200, 9300, 11211, 27017},
		sshConfPath: "/etc/ssh/sshd_config",
	}
}

// Run 执行安全扫描
func (ss *SecurityScanner) Run() *models.SecurityReport {
	findings := []*models.SecurityFinding{}

	findings = append(findings, ss.scanPorts()...)
	findings = append(findings, ss.checkSSHConfig())
	findings = append(findings, ss.checkKernelVersion())
	findings = append(findings, ss.checkPasswordPolicy())
	findings = append(findings, ss.checkFirewallStatus())

	return &models.SecurityReport{Findings: findings}
}

func (ss *SecurityScanner) scanPorts() []*models.SecurityFinding {
	ss.logger.Info("安全体检: 开始端口扫描")
	findings := []*models.SecurityFinding{}
	timeout := 2 * time.Second

	for _, port := range ss.portList {
		address := fmt.Sprintf("%s:%d", ss.targetHost, port)
		conn, err := net.DialTimeout("tcp", address, timeout)
		if err == nil {
			conn.Close()
			findings = append(findings, &models.SecurityFinding{
				Category: "端口暴露",
				Severity: ss.portSeverity(port),
				Title:    fmt.Sprintf("端口 %d 对外开放", port),
				Detail:   fmt.Sprintf("检测到 %s 可访问，建议确认是否需要对公网开放。", address),
				Advice:   ss.portAdvice(port),
			})
		}
	}

	if len(findings) == 0 {
		findings = append(findings, &models.SecurityFinding{
			Category: "端口暴露",
			Severity: "info",
			Title:    "未发现常见服务端口暴露",
		})
	}
	return findings
}

func (ss *SecurityScanner) checkSSHConfig() *models.SecurityFinding {
	file, err := os.Open(ss.sshConfPath)
	if err != nil {
		return &models.SecurityFinding{
			Category: "SSH 配置",
			Severity: "info",
			Title:    "无法读取 sshd_config",
			Detail:   err.Error(),
		}
	}
	defer file.Close()

	opts := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			opts[strings.ToLower(fields[0])] = strings.ToLower(fields[1])
		}
	}

	findings := []string{}
	if val, ok := opts["permitrootlogin"]; ok && (val == "yes" || val == "prohibit-password") {
		findings = append(findings, "允许 root 登录")
	}
	if val, ok := opts["passwordauthentication"]; ok && val == "yes" {
		findings = append(findings, "开启密码认证")
	}
	if val, ok := opts["permitemptypasswords"]; ok && val == "yes" {
		findings = append(findings, "允许空密码")
	}

	if len(findings) == 0 {
		return &models.SecurityFinding{
			Category: "SSH 配置",
			Severity: "info",
			Title:    "SSH 配置未发现明显风险",
		}
	}

	return &models.SecurityFinding{
		Category: "SSH 配置",
		Severity: "medium",
		Title:    "检测到潜在风险配置",
		Detail:   strings.Join(findings, "；"),
		Advice:   "建议禁用 root 远程登录、关闭密码登录或启用密钥认证。",
	}
}

func (ss *SecurityScanner) checkKernelVersion() *models.SecurityFinding {
	info, err := host.Info()
	if err != nil {
		return &models.SecurityFinding{
			Category: "系统信息",
			Severity: "info",
			Title:    "无法获取内核信息",
			Detail:   err.Error(),
		}
	}

	title := fmt.Sprintf("内核版本: %s", info.KernelVersion)
	advice := "建议保持系统更新，定期执行安全补丁。"

	severity := "info"
	if isKernelOutdated(info.KernelVersion) {
		severity = "medium"
		advice = "检测到较旧内核，建议计划升级到厂商支持的 LTS 版本。"
	}

	return &models.SecurityFinding{
		Category: "系统信息",
		Severity: severity,
		Title:    title,
		Detail:   fmt.Sprintf("平台: %s %s (%s)", info.Platform, info.PlatformVersion, info.KernelArch),
		Advice:   advice,
	}
}

func (ss *SecurityScanner) checkPasswordPolicy() *models.SecurityFinding {
	file, err := os.Open("/etc/login.defs")
	if err != nil {
		return &models.SecurityFinding{
			Category: "密码策略",
			Severity: "info",
			Title:    "无法读取 /etc/login.defs",
			Detail:   err.Error(),
		}
	}
	defer file.Close()

	opts := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			opts[strings.ToUpper(fields[0])] = fields[1]
		}
	}

	issues := []string{}
	if val, ok := opts["PASS_MAX_DAYS"]; ok && (val == "" || val == "99999") {
		issues = append(issues, "未限制密码最大有效期")
	}
	if val, ok := opts["PASS_MIN_DAYS"]; ok && val == "0" {
		issues = append(issues, "密码修改间隔为0")
	}
	if val, ok := opts["PASS_WARN_AGE"]; ok && val == "0" {
		issues = append(issues, "未设置密码过期提醒")
	}

	if len(issues) == 0 {
		return &models.SecurityFinding{
			Category: "密码策略",
			Severity: "info",
			Title:    "未发现密码策略问题",
		}
	}

	return &models.SecurityFinding{
		Category: "密码策略",
		Severity: "medium",
		Title:    "密码策略存在改进空间",
		Detail:   strings.Join(issues, "；"),
		Advice:   "建议设置合理的密码有效期、修改间隔和提前提醒。",
	}
}

func (ss *SecurityScanner) checkFirewallStatus() *models.SecurityFinding {
	if output, err := runCommand("ufw", "status"); err == nil {
		if strings.Contains(strings.ToLower(output), "inactive") {
			return &models.SecurityFinding{
				Category: "防火墙",
				Severity: "medium",
				Title:    "UFW 未启用",
				Advice:   "建议启用 UFW 或其他防火墙限制入站流量。",
			}
		}
		return &models.SecurityFinding{
			Category: "防火墙",
			Severity: "info",
			Title:    "UFW 已启用",
		}
	}

	if output, err := runCommand("firewall-cmd", "--state"); err == nil {
		if strings.TrimSpace(output) != "running" {
			return &models.SecurityFinding{
				Category: "防火墙",
				Severity: "medium",
				Title:    "firewalld 未运行",
				Advice:   "建议启动 firewalld 或配置其他防火墙策略。",
			}
		}
		return &models.SecurityFinding{
			Category: "防火墙",
			Severity: "info",
			Title:    "firewalld 运行中",
		}
	}

	return &models.SecurityFinding{
		Category: "防火墙",
		Severity: "info",
		Title:    "无法确定防火墙状态",
		Detail:   "未检测到 UFW 或 firewalld",
	}
}

func (ss *SecurityScanner) portSeverity(port int) string {
	switch port {
	case 22, 3306, 5432, 6379, 27017:
		return "medium"
	case 3389, 5900:
		return "high"
	default:
		return "info"
	}
}

func (ss *SecurityScanner) portAdvice(port int) string {
	switch port {
	case 22:
		return "如需暴露 SSH，建议开启防火墙限制和密钥登录。"
	case 3306, 5432:
		return "数据库服务不建议直接对公网开放，至少应用白名单。"
	case 6379, 27017:
		return "Redis/MongoDB 默认无认证，务必限制来源并设置密码。"
	case 3389:
		return "远程桌面端口容易被暴力破解，建议启用 VPN 或改用安全网关。"
	default:
		return "确认该端口确有业务需求，未使用的端口请关闭。"
	}
}

func isKernelOutdated(version string) bool {
	var major, minor int
	if _, err := fmt.Sscanf(version, "%d.%d", &major, &minor); err != nil {
		return false
	}
	if major < 5 {
		return true
	}
	if major == 5 && minor < 10 {
		return true
	}
	return false
}

func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}
