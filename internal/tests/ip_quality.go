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
	Provider string
	Service  string
	Host     string
	Port     int
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
			"all.spamrats.com",
			"auth.spamrats.com",
			"dyna.spamrats.com",
			"noptr.spamrats.com",
			"psbl.surriel.com",
			"rbl.interserver.net",
			"spamrbl.imp.ch",
			"ubl.unsubscore.com",
			"truncate.gbudb.net",
			"bl.blocklist.de",
			"hostkarma.junkemailfilter.com",
			"mail-abuse.blacklist.jippg.org",
		},
		mailTargets: []ipQualityMailTarget{
			{Provider: "Gmail", Service: "MX", Host: "gmail-smtp-in.l.google.com", Port: 25},
			{Provider: "Gmail", Service: "SMTP SSL", Host: "smtp.gmail.com", Port: 465},
			{Provider: "Gmail", Service: "Submission", Host: "smtp.gmail.com", Port: 587},
			{Provider: "Outlook", Service: "Submission", Host: "smtp-mail.outlook.com", Port: 587},
			{Provider: "Yahoo", Service: "Submission", Host: "smtp.mail.yahoo.com", Port: 587},
			{Provider: "Apple", Service: "Submission", Host: "smtp.mail.me.com", Port: 587},
			{Provider: "QQ", Service: "SMTP SSL", Host: "smtp.qq.com", Port: 465},
			{Provider: "Mail.ru", Service: "SMTP SSL", Host: "smtp.mail.ru", Port: 465},
			{Provider: "AOL", Service: "Submission", Host: "smtp.aol.com", Port: 587},
			{Provider: "GMX", Service: "Submission", Host: "mail.gmx.com", Port: 587},
			{Provider: "Mail.com", Service: "Submission", Host: "smtp.mail.com", Port: 587},
			{Provider: "163", Service: "SMTP SSL", Host: "smtp.163.com", Port: 465},
			{Provider: "Sohu", Service: "SMTP SSL", Host: "smtp.sohu.com", Port: 465},
			{Provider: "Sina", Service: "SMTP SSL", Host: "smtp.sina.com", Port: 465},
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
	report.MailSummary = summarizeMailPorts(report.MailChecks)
	report.NetworkStack = buildIPNetworkStack(report.IPVersion)
	report.RiskSources = buildIPRiskSources(report)
	report.RiskScore, report.RiskLevel = calculateIPRisk(report)
	report.Verdict = buildIPQualityVerdict(report)
	report.Evidence = buildIPQualityEvidence(report)
	report.EvidenceSummary = buildIPQualityEvidenceSummary(report)
	report.Recommendations = buildIPQualityRecommendations(report)

	if report.IPVersion == "IPv6" {
		report.Notes = append(report.Notes, "当前版本仅对 IPv4 执行 DNSBL 黑名单查询，IPv6 已跳过该项。")
	}
	if report.IPType == "datacenter_likely" {
		report.Notes = append(report.Notes, "IP 类型基于 ISP、ASN 组织和反向 DNS 关键词推断，仅作为参考。")
	}
	report.Notes = append(report.Notes, "ASN 信息通过 Team Cymru DNS 查询获取；若网络或 DNS 限制导致失败，会自动留空。")
	report.Notes = append(report.Notes, "风险来源当前使用 DNSBL、Team Cymru ASN、反向 DNS 和关键词启发式；未使用需要 API Key 的商业数据库。")
	report.Notes = append(report.Notes, "邮件连通性只测试出站 TCP 连接，不代表收信信誉或真实投递率。")

	return report
}

func buildIPQualityVerdict(report *models.IPQualityReport) *models.IPQualityVerdict {
	grade := ipQualityGrade(report)
	mailUsable := countReachableIPQualityMail(report.MailChecks) > 0
	hostingHint := ipQualityRiskFactorDetected(report, "datacenter") || report.IPType == "datacenter_likely"
	proxyHint := ipQualityRiskFactorDetected(report, "proxy") || ipQualityRiskFactorDetected(report, "vpn") || ipQualityRiskFactorDetected(report, "tor")
	return &models.IPQualityVerdict{
		Grade:       grade,
		Summary:     ipQualityVerdictSummary(report, grade, mailUsable, hostingHint, proxyHint),
		IPTypeLabel: ipQualityTypeLabel(report.IPType),
		RiskLabel:   ipQualityRiskLabel(report.RiskLevel),
		MailUsable:  mailUsable,
		HostingHint: hostingHint,
		ProxyHint:   proxyHint,
	}
}

func buildIPQualityEvidence(report *models.IPQualityReport) []*models.IPQualityEvidence {
	evidence := []*models.IPQualityEvidence{
		{Name: "公网 IP", Value: report.PublicIP, Status: "info", Detail: strings.TrimSpace(strings.Join([]string{report.Country, report.City}, " "))},
		{Name: "ASN/组织", Value: ipQualityASNValue(report), Status: "info", Detail: report.ISP},
		{Name: "IP 类型", Value: ipQualityTypeLabel(report.IPType), Status: ipQualityTypeStatus(report.IPType), Detail: "基于 ISP、ASN 组织和反向 DNS 关键词推断"},
		{Name: "风险分", Value: fmt.Sprintf("%d/100", report.RiskScore), Status: report.RiskLevel, Detail: ipQualityRiskLabel(report.RiskLevel)},
	}
	if report.BlacklistSummary != nil {
		status := "clean"
		if report.BlacklistSummary.Listed > 0 {
			status = "listed"
		} else if report.BlacklistSummary.Timeout > 0 || report.BlacklistSummary.Other > 0 {
			status = "partial"
		}
		evidence = append(evidence, &models.IPQualityEvidence{
			Name:   "DNSBL",
			Value:  fmt.Sprintf("命中 %d/%d", report.BlacklistSummary.Listed, report.BlacklistSummary.Total),
			Status: status,
			Detail: fmt.Sprintf("干净 %d，超时 %d，跳过 %d，其他 %d", report.BlacklistSummary.Clean, report.BlacklistSummary.Timeout, report.BlacklistSummary.Skipped, report.BlacklistSummary.Other),
		})
	}
	mailReachable := countReachableIPQualityMail(report.MailChecks)
	mailDetail := "仅表示 TCP 出站连通性，不代表真实投递率"
	if report.MailSummary != nil {
		mailDetail = fmt.Sprintf("服务商 %d 个，可连服务商 %d 个；%s", report.MailSummary.Providers, report.MailSummary.ProviderOpen, mailDetail)
	}
	evidence = append(evidence, &models.IPQualityEvidence{
		Name:   "邮件端口",
		Value:  fmt.Sprintf("可连 %d/%d", mailReachable, len(report.MailChecks)),
		Status: ipQualityMailStatus(report.MailChecks),
		Detail: mailDetail,
	})
	if report.NetworkStack != nil {
		evidence = append(evidence, &models.IPQualityEvidence{
			Name:   "网络栈",
			Value:  ipQualityNetworkStackLabel(report.NetworkStack),
			Status: "info",
			Detail: report.NetworkStack.Note,
		})
	}
	for _, factor := range report.RiskFactors {
		if factor == nil || !factor.Detected {
			continue
		}
		evidence = append(evidence, &models.IPQualityEvidence{
			Name:   "风险因子",
			Value:  factor.Name,
			Status: factor.Confidence,
			Detail: factor.Detail,
		})
	}
	return evidence
}

func buildIPQualityEvidenceSummary(report *models.IPQualityReport) []*models.EvidenceSummary {
	if report == nil {
		return nil
	}
	items := []*models.EvidenceSummary{}

	identityStatus := "partial"
	identityConfidence := "low"
	identityDetail := "公网 IP 已采集，ASN 或反向 DNS 信息不完整。"
	if strings.TrimSpace(report.ASN) != "" && strings.TrimSpace(report.Organization) != "" {
		identityStatus = "success"
		identityConfidence = "medium"
		identityDetail = "ASN 和组织名称可用，可辅助判断云厂商、运营商或托管线索。"
	}
	if len(report.ReverseDNS) > 0 {
		identityConfidence = "medium"
		if identityStatus == "partial" {
			identityDetail = "反向 DNS 可用，但 ASN 信息不完整。"
		}
	}
	items = append(items, &models.EvidenceSummary{
		Category:   "identity",
		Label:      "IP 身份信息",
		Status:     identityStatus,
		Confidence: identityConfidence,
		Impact:     "影响 IP 类型推断和云服务器/住宅运营商判断。",
		Detail:     identityDetail,
		Limitation: "ASN、ISP 和 rDNS 只能说明网络归属线索，不能单独证明真实用户属性。",
	})

	dnsblStatus := "skipped"
	dnsblConfidence := "low"
	dnsblDetail := "DNSBL 未执行或没有可用结果。"
	if report.BlacklistSummary != nil {
		dnsblStatus = "success"
		dnsblConfidence = "high"
		dnsblDetail = fmt.Sprintf("查询 %d 个 DNSBL，命中 %d，正常 %d，超时 %d，跳过 %d。",
			report.BlacklistSummary.Total,
			report.BlacklistSummary.Listed,
			report.BlacklistSummary.Clean,
			report.BlacklistSummary.Timeout,
			report.BlacklistSummary.Skipped,
		)
		switch {
		case report.BlacklistSummary.Listed > 0:
			dnsblStatus = "warning"
		case report.BlacklistSummary.Total == report.BlacklistSummary.Skipped:
			dnsblStatus = "skipped"
			dnsblConfidence = "low"
		case report.BlacklistSummary.Timeout > 0 || report.BlacklistSummary.Other > 0:
			dnsblStatus = "partial"
			dnsblConfidence = "medium"
		}
	}
	items = append(items, &models.EvidenceSummary{
		Category:   "dnsbl",
		Label:      "DNSBL 黑名单",
		Status:     dnsblStatus,
		Confidence: dnsblConfidence,
		Impact:     "影响邮件投递、反滥用和部分风控场景判断。",
		Detail:     dnsblDetail,
		Limitation: "公开 DNSBL 覆盖范围有限，未命中不代表没有商业风控风险。",
	})

	mailStatus := "skipped"
	mailConfidence := "low"
	mailDetail := "邮件端口连通性未执行。"
	if report.MailSummary != nil && report.MailSummary.Total > 0 {
		mailStatus = "success"
		mailConfidence = "medium"
		if report.MailSummary.ProviderOpen == 0 {
			mailStatus = "warning"
		} else if report.MailSummary.ProviderOpen < report.MailSummary.Providers {
			mailStatus = "partial"
		}
		mailDetail = fmt.Sprintf("检测 %d 个服务商、%d 个端口，可连服务商 %d，可连端口 %d，阻断 %d，超时 %d。",
			report.MailSummary.Providers,
			report.MailSummary.Total,
			report.MailSummary.ProviderOpen,
			report.MailSummary.Reachable,
			report.MailSummary.Blocked,
			report.MailSummary.Timeout,
		)
	}
	items = append(items, &models.EvidenceSummary{
		Category:   "mail",
		Label:      "邮件出站连通",
		Status:     mailStatus,
		Confidence: mailConfidence,
		Impact:     "影响是否适合直接运行 SMTP 发信或邮件相关服务。",
		Detail:     mailDetail,
		Limitation: "这里只测试 TCP 出站连通，不代表收信信誉、SPF/DKIM/DMARC 或真实投递率。",
	})

	heuristicStatus := "success"
	heuristicConfidence := "low"
	heuristicDetail := "未发现代理、VPN、Tor、滥用或机房关键词。"
	if hasDetectedIPRiskFactor(report.RiskFactors) {
		heuristicStatus = "warning"
		heuristicConfidence = "medium"
		heuristicDetail = detectedIPRiskFactorSummary(report.RiskFactors)
	}
	items = append(items, &models.EvidenceSummary{
		Category:   "heuristic",
		Label:      "本地启发式线索",
		Status:     heuristicStatus,
		Confidence: heuristicConfidence,
		Impact:     "影响代理/VPN/Tor/机房等标签和风险解释。",
		Detail:     heuristicDetail,
		Limitation: "关键词启发式只能作为弱证据，不能替代商业风险库或目标业务实测。",
	})

	items = append(items, &models.EvidenceSummary{
		Category:   "external_api",
		Label:      "商业风险 API",
		Status:     "skipped",
		Confidence: "low",
		Impact:     "默认不影响本次评分，只作为隐私边界说明。",
		Detail:     "未调用需要 API Key 的第三方商业风险数据库。",
		Limitation: "报告不会给出商业风控数据库级别的强结论；如需此类结论，应显式接入自有授权来源。",
	})

	return items
}

func detectedIPRiskFactorSummary(factors []*models.IPRiskFactor) string {
	names := []string{}
	for _, factor := range factors {
		if factor == nil || !factor.Detected {
			continue
		}
		names = append(names, factor.Name)
	}
	if len(names) == 0 {
		return "未发现代理、VPN、Tor、滥用或机房关键词。"
	}
	return "命中风险线索: " + strings.Join(names, ", ")
}

func buildIPQualityRecommendations(report *models.IPQualityReport) []string {
	recommendations := []string{}
	if report.BlacklistSummary != nil && report.BlacklistSummary.Listed > 0 {
		recommendations = append(recommendations, "DNSBL 存在命中，不建议直接用于邮件发送或高信誉业务。")
	}
	if countReachableIPQualityMail(report.MailChecks) == 0 && len(report.MailChecks) > 0 {
		recommendations = append(recommendations, "邮件端口全部不可连，如需发信建议使用第三方 SMTP 或工单确认端口策略。")
	}
	if ipQualityRiskFactorDetected(report, "proxy") || ipQualityRiskFactorDetected(report, "vpn") || ipQualityRiskFactorDetected(report, "tor") {
		recommendations = append(recommendations, "检测到代理/VPN/Tor 相关线索，可能影响风控敏感服务访问。")
	}
	if report.IPType == "datacenter_likely" {
		recommendations = append(recommendations, "该 IP 更像机房/云服务器节点，适合常规建站、代理入口或测试环境，不应按住宅 IP 预期使用。")
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "当前未发现明显高风险线索，仍建议结合目标业务进行实际访问和投递测试。")
	}
	return recommendations
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

func summarizeMailPorts(checks []*models.MailPortCheck) *models.MailPortSummary {
	summary := &models.MailPortSummary{Total: len(checks)}
	providers := map[string]bool{}
	openProviders := map[string]bool{}
	for _, check := range checks {
		if check == nil {
			continue
		}
		if check.Provider != "" {
			providers[check.Provider] = true
		}
		switch check.Status {
		case "reachable":
			summary.Reachable++
			if check.Provider != "" {
				openProviders[check.Provider] = true
			}
		case "timeout":
			summary.Timeout++
		default:
			summary.Blocked++
		}
	}
	summary.Providers = len(providers)
	summary.ProviderOpen = len(openProviders)
	return summary
}

func (s *IPQualityScanner) checkMailPorts() []*models.MailPortCheck {
	checks := make([]*models.MailPortCheck, 0, len(s.mailTargets))
	for _, target := range s.mailTargets {
		address := net.JoinHostPort(target.Host, fmt.Sprintf("%d", target.Port))
		conn, err := net.DialTimeout("tcp", address, s.connectTimeout)
		check := &models.MailPortCheck{
			Service:  target.Service,
			Provider: target.Provider,
			Target:   fmt.Sprintf("%s %s (%s)", target.Provider, target.Service, target.Host),
			Port:     target.Port,
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

func buildIPRiskSources(report *models.IPQualityReport) []*models.IPRiskSource {
	sources := []*models.IPRiskSource{
		{
			Name:    "Team Cymru ASN",
			Type:    "dns",
			Status:  enabledStatus(report.ASN != "" || report.Organization != ""),
			Signal:  ipQualityASNValue(report),
			Detail:  "ASN 和组织名称，用于判断云厂商、托管和运营商线索",
			Weight:  15,
			Enabled: true,
		},
		{
			Name:    "Reverse DNS",
			Type:    "dns",
			Status:  enabledStatus(len(report.ReverseDNS) > 0),
			Signal:  strings.Join(report.ReverseDNS, ", "),
			Detail:  "反向 DNS 关键词可辅助判断代理、VPN、Tor 或托管线索",
			Weight:  10,
			Enabled: true,
		},
		{
			Name:    "DNSBL",
			Type:    "dnsbl",
			Status:  dnsblSourceStatus(report.BlacklistSummary),
			Signal:  dnsblSourceSignal(report.BlacklistSummary),
			Detail:  "多组公开 DNSBL 黑名单查询",
			Weight:  35,
			Enabled: true,
		},
		{
			Name:    "Heuristic Keywords",
			Type:    "heuristic",
			Status:  enabledStatus(hasDetectedIPRiskFactor(report.RiskFactors)),
			Detail:  "基于 ISP、ASN 组织和 rDNS 的本地关键词判断",
			Weight:  20,
			Enabled: true,
		},
		{
			Name:    "Commercial Risk APIs",
			Type:    "external_api",
			Status:  "disabled",
			Detail:  "未调用需要 API Key 的商业风险数据库，避免默认流程依赖第三方账号",
			Enabled: false,
		},
	}
	return sources
}

func dnsblSourceStatus(summary *models.IPBlacklistSummary) string {
	if summary == nil || summary.Total == 0 {
		return "missing"
	}
	if summary.Listed > 0 {
		return "listed"
	}
	if summary.Timeout > 0 || summary.Other > 0 {
		return "partial"
	}
	return "clean"
}

func dnsblSourceSignal(summary *models.IPBlacklistSummary) string {
	if summary == nil {
		return ""
	}
	return fmt.Sprintf("命中 %d/%d，正常 %d，超时 %d", summary.Listed, summary.Total, summary.Clean, summary.Timeout)
}

func buildIPNetworkStack(version string) *models.IPNetworkStack {
	stack := &models.IPNetworkStack{DetectedVersion: version}
	switch version {
	case "IPv4":
		stack.IPv4Available = true
	case "IPv6":
		stack.IPv6Available = true
	}
	stack.DualStack = stack.IPv4Available && stack.IPv6Available
	stack.Note = "IP 质量模块基于当前采集到的公网 IP 检测；完整双栈对比建议结合网络质量矩阵查看。"
	return stack
}

func enabledStatus(value bool) string {
	if value {
		return "available"
	}
	return "missing"
}

func hasDetectedIPRiskFactor(factors []*models.IPRiskFactor) bool {
	for _, factor := range factors {
		if factor != nil && factor.Detected {
			return true
		}
	}
	return false
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
	asn := parseTeamCymruOriginASN(records[0])
	if asn == "" {
		return "", ""
	}
	return asn, s.lookupASNName(asn)
}

func (s *IPQualityScanner) lookupASNName(asn string) string {
	ctx, cancel := context.WithTimeout(context.Background(), s.lookupTimeout)
	defer cancel()
	records, err := s.resolver.LookupTXT(ctx, "AS"+asn+".asn.cymru.com")
	if err != nil || len(records) == 0 {
		return ""
	}
	return parseTeamCymruASNName(records[0])
}

func parseTeamCymruOriginASN(record string) string {
	fields := strings.Split(record, "|")
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimSpace(fields[0])
}

func parseTeamCymruASNName(record string) string {
	fields := strings.Split(record, "|")
	if len(fields) < 5 {
		return ""
	}
	return strings.TrimSpace(fields[4])
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

func ipQualityGrade(report *models.IPQualityReport) string {
	listed := 0
	if report.BlacklistSummary != nil {
		listed = report.BlacklistSummary.Listed
	}
	switch {
	case report.RiskLevel == "low" && listed == 0:
		return "A"
	case report.RiskLevel == "low":
		return "B"
	case report.RiskLevel == "medium":
		return "C"
	default:
		return "D"
	}
}

func ipQualityVerdictSummary(report *models.IPQualityReport, grade string, mailUsable bool, hostingHint bool, proxyHint bool) string {
	parts := []string{fmt.Sprintf("综合评级 %s", grade), ipQualityTypeLabel(report.IPType), ipQualityRiskLabel(report.RiskLevel)}
	if mailUsable {
		parts = append(parts, "邮件端口部分可连")
	} else if len(report.MailChecks) > 0 {
		parts = append(parts, "邮件端口不可连")
	}
	if proxyHint {
		parts = append(parts, "存在代理/VPN/Tor 线索")
	} else if hostingHint {
		parts = append(parts, "存在机房/托管线索")
	}
	return strings.Join(parts, "；")
}

func ipQualityTypeLabel(value string) string {
	switch value {
	case "datacenter_likely":
		return "机房/云服务器 IP"
	case "residential_or_isp_likely":
		return "住宅或运营商 IP"
	default:
		return "类型未知"
	}
}

func ipQualityTypeStatus(value string) string {
	switch value {
	case "datacenter_likely":
		return "hosting"
	case "residential_or_isp_likely":
		return "clean"
	default:
		return "unknown"
	}
}

func ipQualityRiskLabel(value string) string {
	switch value {
	case "low":
		return "低风险"
	case "medium":
		return "中风险"
	case "high":
		return "高风险"
	default:
		return "未知风险"
	}
}

func ipQualityMailStatus(checks []*models.MailPortCheck) string {
	reachable := countReachableIPQualityMail(checks)
	switch {
	case len(checks) == 0:
		return "unknown"
	case reachable == len(checks):
		return "open"
	case reachable > 0:
		return "partial"
	default:
		return "blocked"
	}
}

func ipQualityNetworkStackLabel(stack *models.IPNetworkStack) string {
	if stack == nil {
		return "-"
	}
	if stack.DualStack {
		return "IPv4/IPv6 双栈"
	}
	return stack.DetectedVersion
}

func countReachableIPQualityMail(checks []*models.MailPortCheck) int {
	count := 0
	for _, check := range checks {
		if check != nil && check.Reachable {
			count++
		}
	}
	return count
}

func ipQualityRiskFactorDetected(report *models.IPQualityReport, name string) bool {
	for _, factor := range report.RiskFactors {
		if factor != nil && factor.Name == name && factor.Detected {
			return true
		}
	}
	return false
}

func ipQualityASNValue(report *models.IPQualityReport) string {
	parts := []string{}
	if strings.TrimSpace(report.ASN) != "" {
		parts = append(parts, "AS"+strings.TrimSpace(report.ASN))
	}
	if strings.TrimSpace(report.Organization) != "" {
		parts = append(parts, strings.TrimSpace(report.Organization))
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, " / ")
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
