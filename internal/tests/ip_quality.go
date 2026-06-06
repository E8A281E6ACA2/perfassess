package tests

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
	"github.com/E8A281E6ACA2/perfassess/pkg/logger"
)

type ipQualityMailTarget struct {
	Service string
	Host    string
	Port    int
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
		logger:   logger,
		resolver: net.DefaultResolver,
		blacklistZones: []string{
			"zen.spamhaus.org",
			"bl.spamcop.net",
			"dnsbl.sorbs.net",
			"b.barracudacentral.org",
			"cbl.abuseat.org",
			"dnsbl.spfbl.net",
			"bl.spameatingmonkey.net",
			"combined.abuse.ch",
			"dnsbl.dronebl.org",
			"spam.dnsbl.sorbs.net",
		},
		mailTargets: []ipQualityMailTarget{
			{Service: "Gmail MX", Host: "gmail-smtp-in.l.google.com", Port: 25},
			{Service: "Gmail SMTP SSL", Host: "smtp.gmail.com", Port: 465},
			{Service: "Gmail SMTP Submission", Host: "smtp.gmail.com", Port: 587},
			{Service: "Outlook SMTP", Host: "smtp-mail.outlook.com", Port: 587},
			{Service: "Yahoo SMTP", Host: "smtp.mail.yahoo.com", Port: 587},
			{Service: "QQ SMTP", Host: "smtp.qq.com", Port: 465},
			{Service: "163 SMTP", Host: "smtp.163.com", Port: 465},
		},
		lookupTimeout:  3 * time.Second,
		connectTimeout: 3 * time.Second,
		datacenterHints: []string{
			"amazon", "aws", "azure", "cloud", "cloudflare", "contabo", "datacenter", "data center",
			"digitalocean", "google", "hetzner", "hosting", "linode", "microsoft", "netcup", "ovh",
			"server", "vps", "colo", "colocation", "leaseweb", "oracle", "tencent", "alibaba",
			"akamai", "choopa", "vultr", "equinix", "rackspace", "softlayer", "m247", "datahouse",
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
	report.ASN, report.Organization = s.lookupASN(parsedIP)
	report.ReverseDNS = s.lookupReverseDNS(parsedIP)
	report.IPType = s.classifyIPType(strings.Join([]string{report.ISP, report.Organization, strings.Join(report.ReverseDNS, " ")}, " "))
	report.RiskFactors = s.detectRiskFactors(report)
	report.BlacklistChecks = s.checkBlacklists(parsedIP)
	report.BlacklistSummary = summarizeBlacklists(report.BlacklistChecks)
	report.MailChecks = s.checkMailPorts()
	report.RiskScore, report.RiskLevel = calculateIPRisk(report)

	if report.IPVersion == "IPv6" {
		report.Notes = append(report.Notes, "当前版本仅对 IPv4 执行 DNSBL 黑名单查询，IPv6 已跳过该项。")
	}
	if report.IPType == "datacenter_likely" {
		report.Notes = append(report.Notes, "IP 类型基于 ISP、ASN 组织和反向 DNS 关键词推断，仅作为参考。")
	}
	report.Notes = append(report.Notes, "ASN 信息通过 Team Cymru DNS 查询获取；若网络或 DNS 限制导致失败，会自动留空。")
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
		case isDNSNotFound(err):
			check.Status = "clean"
		default:
			check.Status = "error"
			if err != nil {
				check.Detail = err.Error()
			}
		}
		checks = append(checks, check)
	}

	return checks
}

func summarizeBlacklists(checks []*models.IPBlacklistCheck) *models.IPBlacklistSummary {
	summary := &models.IPBlacklistSummary{Total: len(checks)}
	for _, check := range checks {
		if check == nil {
			summary.Other++
			continue
		}
		switch check.Status {
		case "clean":
			summary.Clean++
		case "listed":
			summary.Listed++
		case "timeout":
			summary.Timeout++
		case "skipped":
			summary.Skipped++
		default:
			summary.Other++
		}
	}
	return summary
}

func (s *IPQualityScanner) checkMailPorts() []*models.MailPortCheck {
	checks := make([]*models.MailPortCheck, 0, len(s.mailTargets))
	for _, target := range s.mailTargets {
		address := net.JoinHostPort(target.Host, fmt.Sprintf("%d", target.Port))
		conn, err := net.DialTimeout("tcp", address, s.connectTimeout)
		check := &models.MailPortCheck{
			Target: fmt.Sprintf("%s (%s)", target.Service, target.Host),
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

func (s *IPQualityScanner) lookupReverseDNS(ip net.IP) []string {
	ctx, cancel := context.WithTimeout(context.Background(), s.lookupTimeout)
	defer cancel()
	names, err := s.resolver.LookupAddr(ctx, ip.String())
	if err != nil {
		return nil
	}
	for i := range names {
		names[i] = strings.TrimSuffix(names[i], ".")
	}
	return names
}

func (s *IPQualityScanner) lookupASN(ip net.IP) (string, string) {
	query := teamCymruQuery(ip)
	if query == "" {
		return "", ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.lookupTimeout)
	defer cancel()
	records, err := s.resolver.LookupTXT(ctx, query)
	if err != nil || len(records) == 0 {
		return "", ""
	}
	fields := strings.Split(records[0], "|")
	if len(fields) < 5 {
		return strings.TrimSpace(fields[0]), ""
	}
	return strings.TrimSpace(fields[0]), strings.TrimSpace(fields[4])
}

func teamCymruQuery(ip net.IP) string {
	if ip4 := ip.To4(); ip4 != nil {
		return fmt.Sprintf("%d.%d.%d.%d.origin.asn.cymru.com", ip4[3], ip4[2], ip4[1], ip4[0])
	}
	ip16 := ip.To16()
	if ip16 == nil {
		return ""
	}
	encoded := hex.EncodeToString(ip16)
	parts := make([]string, 0, len(encoded)+4)
	for i := len(encoded) - 1; i >= 0; i-- {
		parts = append(parts, string(encoded[i]))
	}
	return strings.Join(parts, ".") + ".origin6.asn.cymru.com"
}

func (s *IPQualityScanner) classifyIPType(text string) string {
	normalized := strings.ToLower(strings.TrimSpace(text))
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

func (s *IPQualityScanner) detectRiskFactors(report *models.IPQualityReport) []*models.IPRiskFactor {
	evidence := strings.ToLower(strings.Join([]string{
		report.ISP,
		report.Organization,
		strings.Join(report.ReverseDNS, " "),
	}, " "))

	return []*models.IPRiskFactor{
		keywordRiskFactor("proxy", "代理", evidence, []string{"proxy", "squid", "socks", "http-proxy"}),
		keywordRiskFactor("vpn", "VPN", evidence, []string{"vpn", "openvpn", "wireguard", "ipsec"}),
		keywordRiskFactor("tor", "Tor", evidence, []string{"tor", "exitnode", "exit-node"}),
		keywordRiskFactor("datacenter", "机房/托管", evidence, s.datacenterHints),
		keywordRiskFactor("abuse", "滥用线索", evidence, []string{"abuse", "spam", "blacklist", "malware", "botnet"}),
	}
}

func keywordRiskFactor(name string, label string, text string, keywords []string) *models.IPRiskFactor {
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return &models.IPRiskFactor{
				Name:       name,
				Detected:   true,
				Confidence: "medium",
				Source:     "heuristic",
				Detail:     fmt.Sprintf("%s 关键词命中: %s", label, keyword),
			}
		}
	}
	return &models.IPRiskFactor{
		Name:       name,
		Detected:   false,
		Confidence: "low",
		Source:     "heuristic",
		Detail:     fmt.Sprintf("未从 ISP、ASN 组织或反向 DNS 中发现%s关键词", label),
	}
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
	for _, factor := range report.RiskFactors {
		if factor == nil || !factor.Detected {
			continue
		}
		switch factor.Name {
		case "proxy", "vpn", "tor":
			score += 25
		case "abuse":
			score += 20
		case "datacenter":
			score += 5
		}
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

func isDNSNotFound(err error) bool {
	if err == nil {
		return false
	}
	if dnsErr, ok := err.(*net.DNSError); ok {
		return dnsErr.IsNotFound
	}
	return false
}
