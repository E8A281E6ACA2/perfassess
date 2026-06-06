package tests

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
	"github.com/E8A281E6ACA2/perfassess/pkg/logger"
)

type ipQualityMailTarget struct {
	Host string
	Port int
}

// IPQualityScanner 执行公网 IP 质量检测
type IPQualityScanner struct {
	logger          *logger.Logger
	resolver        *net.Resolver
	blacklistZones  []string
	mailTargets     []ipQualityMailTarget
	lookupTimeout   time.Duration
	connectTimeout  time.Duration
	datacenterHints []string
}

// NewIPQualityScanner 创建 IP 质量检测器
func NewIPQualityScanner(logger *logger.Logger) *IPQualityScanner {
	return &IPQualityScanner{
		logger:         logger,
		resolver:       net.DefaultResolver,
		blacklistZones: []string{"zen.spamhaus.org", "bl.spamcop.net", "dnsbl.sorbs.net"},
		mailTargets: []ipQualityMailTarget{
			{Host: "gmail-smtp-in.l.google.com", Port: 25},
			{Host: "smtp.gmail.com", Port: 465},
			{Host: "smtp.gmail.com", Port: 587},
			{Host: "smtp-mail.outlook.com", Port: 587},
		},
		lookupTimeout:  3 * time.Second,
		connectTimeout: 3 * time.Second,
		datacenterHints: []string{
			"amazon", "aws", "azure", "cloud", "cloudflare", "contabo", "datacenter", "data center",
			"digitalocean", "google", "hetzner", "hosting", "linode", "microsoft", "netcup", "ovh",
			"server", "vps", "colo", "colocation", "leaseweb", "oracle", "tencent", "alibaba",
		},
	}
}

// Run 执行完整 IP 质量检测
func (s *IPQualityScanner) Run(systemInfo *models.SystemInfo) *models.IPQualityReport {
	report := &models.IPQualityReport{
		IPType:    "unknown",
		RiskLevel: "unknown",
		Notes:     []string{},
	}

	if systemInfo != nil && systemInfo.IPInfo != nil {
		ipInfo := systemInfo.IPInfo
		report.PublicIP = strings.TrimSpace(ipInfo.PublicIP)
		report.ISP = ipInfo.ISP
		if ipInfo.GeoLocation != nil {
			report.Country = ipInfo.GeoLocation.Country
			report.City = ipInfo.GeoLocation.City
		}
	}

	parsedIP := net.ParseIP(report.PublicIP)
	if parsedIP == nil {
		report.RiskLevel = "unknown"
		report.Notes = append(report.Notes, "未获取到有效公网 IP，无法执行 IP 质量检测。")
		return report
	}

	report.IPVersion = ipVersion(parsedIP)
	report.IPType = s.classifyIPType(report.ISP)
	report.BlacklistChecks = s.checkBlacklists(parsedIP)
	report.MailChecks = s.checkMailPorts()
	report.RiskScore, report.RiskLevel = calculateIPRisk(report)

	if report.IPVersion == "IPv6" {
		report.Notes = append(report.Notes, "当前版本仅对 IPv4 执行 DNSBL 黑名单查询，IPv6 已跳过该项。")
	}
	if report.IPType == "datacenter_likely" {
		report.Notes = append(report.Notes, "IP 类型基于 ISP 关键词推断，仅作为参考。")
	}
	report.Notes = append(report.Notes, "邮件连通性只测试出站 TCP 连接，不代表收信信誉或真实投递率。")

	return report
}

func (s *IPQualityScanner) checkBlacklists(ip net.IP) []*models.IPBlacklistCheck {
	ip4 := ip.To4()
	checks := []*models.IPBlacklistCheck{}
	if ip4 == nil {
		for _, zone := range s.blacklistZones {
			checks = append(checks, &models.IPBlacklistCheck{
				Zone:   zone,
				Status: "skipped",
				Detail: "当前版本仅支持 IPv4 DNSBL 查询",
			})
		}
		return checks
	}

	reversed := fmt.Sprintf("%d.%d.%d.%d", ip4[3], ip4[2], ip4[1], ip4[0])
	for _, zone := range s.blacklistZones {
		query := reversed + "." + zone
		ctx, cancel := context.WithTimeout(context.Background(), s.lookupTimeout)
		records, err := s.resolver.LookupHost(ctx, query)
		cancel()

		check := &models.IPBlacklistCheck{Zone: zone}
		switch {
		case err == nil && len(records) > 0:
			check.Listed = true
			check.Status = "listed"
			check.Detail = strings.Join(records, ", ")
		case isDNSTimeout(err):
			check.Status = "timeout"
			check.Detail = err.Error()
		default:
			check.Status = "clean"
		}
		checks = append(checks, check)
	}

	return checks
}

func (s *IPQualityScanner) checkMailPorts() []*models.MailPortCheck {
	checks := make([]*models.MailPortCheck, 0, len(s.mailTargets))
	for _, target := range s.mailTargets {
		address := net.JoinHostPort(target.Host, fmt.Sprintf("%d", target.Port))
		conn, err := net.DialTimeout("tcp", address, s.connectTimeout)
		check := &models.MailPortCheck{
			Target: target.Host,
			Port:   target.Port,
		}
		if err == nil {
			conn.Close()
			check.Reachable = true
			check.Status = "reachable"
		} else if isDNSTimeout(err) {
			check.Status = "timeout"
			check.Detail = err.Error()
		} else {
			check.Status = "blocked"
			check.Detail = err.Error()
		}
		checks = append(checks, check)
	}
	return checks
}

func (s *IPQualityScanner) classifyIPType(isp string) string {
	normalized := strings.ToLower(strings.TrimSpace(isp))
	if normalized == "" {
		return "unknown"
	}
	for _, hint := range s.datacenterHints {
		if strings.Contains(normalized, hint) {
			return "datacenter_likely"
		}
	}
	return "residential_or_isp_likely"
}

func calculateIPRisk(report *models.IPQualityReport) (int, string) {
	score := 0
	for _, check := range report.BlacklistChecks {
		switch check.Status {
		case "listed":
			score += 35
		case "timeout":
			score += 5
		}
	}

	blockedMail := 0
	for _, check := range report.MailChecks {
		if !check.Reachable {
			blockedMail++
		}
	}
	if len(report.MailChecks) > 0 && blockedMail == len(report.MailChecks) {
		score += 25
	} else {
		score += blockedMail * 5
	}

	if report.IPType == "datacenter_likely" {
		score += 10
	}
	if score > 100 {
		score = 100
	}

	switch {
	case score >= 60:
		return score, "high"
	case score >= 30:
		return score, "medium"
	default:
		return score, "low"
	}
}

func ipVersion(ip net.IP) string {
	if ip.To4() != nil {
		return "IPv4"
	}
	return "IPv6"
}

func isDNSTimeout(err error) bool {
	if err == nil {
		return false
	}
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "timeout")
}
