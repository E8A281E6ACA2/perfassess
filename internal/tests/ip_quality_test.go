package tests

import (
	"net"
	"testing"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
)

func TestCalculateIPRiskLevels(t *testing.T) {
	report := &models.IPQualityReport{
		IPType: "datacenter_likely",
		RiskFactors: []*models.IPRiskFactor{
			{Name: "vpn", Detected: true},
		},
		BlacklistChecks: []*models.IPBlacklistCheck{
			{Zone: "zen.spamhaus.org", Listed: true, Status: "listed"},
			{Zone: "bl.spamcop.net", Status: "clean"},
		},
		MailChecks: []*models.MailPortCheck{
			{Target: "gmail-smtp-in.l.google.com", Port: 25, Status: "blocked"},
			{Target: "smtp.gmail.com", Port: 465, Status: "blocked"},
			{Target: "smtp.gmail.com", Port: 587, Status: "blocked"},
			{Target: "smtp-mail.outlook.com", Port: 587, Status: "blocked"},
		},
	}

	score, level := calculateIPRisk(report)
	if level != "high" {
		t.Fatalf("expected high risk, got %q", level)
	}
	if score < 60 {
		t.Fatalf("expected high risk score, got %d", score)
	}
}

func TestClassifyIPTypeUsesISPHints(t *testing.T) {
	scanner := NewIPQualityScanner(nil)

	if got := scanner.classifyIPType("Hetzner Online GmbH"); got != "datacenter_likely" {
		t.Fatalf("expected datacenter hint, got %q", got)
	}
	if got := scanner.classifyIPType("Local Fiber ISP"); got != "residential_or_isp_likely" {
		t.Fatalf("expected isp likely, got %q", got)
	}
	if got := scanner.classifyIPType(""); got != "unknown" {
		t.Fatalf("expected unknown without ISP, got %q", got)
	}
}

func TestTeamCymruQueryFormatsIPv4AndIPv6(t *testing.T) {
	if got := teamCymruQuery(netParseIP(t, "192.0.2.1")); got != "1.2.0.192.origin.asn.cymru.com" {
		t.Fatalf("unexpected IPv4 cymru query: %q", got)
	}
	if got := teamCymruQuery(netParseIP(t, "2001:db8::1")); got != "1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.origin6.asn.cymru.com" {
		t.Fatalf("unexpected IPv6 cymru query: %q", got)
	}
}

func TestParseTeamCymruRecords(t *testing.T) {
	origin := "197540 | 159.195.40.0/22 | DE | ripencc | 1992-05-26"
	if got := parseTeamCymruOriginASN(origin); got != "197540" {
		t.Fatalf("expected origin ASN, got %q", got)
	}

	asnName := "197540 | DE | ripencc | 2011-07-21 | NETCUP-AS netcup GmbH, DE"
	if got := parseTeamCymruASNName(asnName); got != "NETCUP-AS netcup GmbH, DE" {
		t.Fatalf("expected ASN name, got %q", got)
	}
}

func TestSummarizeBlacklists(t *testing.T) {
	summary := summarizeBlacklists([]*models.IPBlacklistCheck{
		{Status: "clean"},
		{Status: "listed"},
		{Status: "timeout"},
		{Status: "skipped"},
		{Status: "other"},
	})

	if summary.Total != 5 || summary.Clean != 1 || summary.Listed != 1 || summary.Timeout != 1 || summary.Skipped != 1 || summary.Other != 1 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
}

func TestDetectRiskFactorsUsesEvidence(t *testing.T) {
	scanner := NewIPQualityScanner(nil)
	factors := scanner.detectRiskFactors(&models.IPQualityReport{
		ISP:          "Example Fiber",
		Organization: "Example VPN Hosting",
		ReverseDNS:   []string{"tor-exit.example.net"},
	})

	detected := map[string]bool{}
	for _, factor := range factors {
		detected[factor.Name] = factor.Detected
	}
	if !detected["vpn"] || !detected["tor"] || !detected["datacenter"] {
		t.Fatalf("expected vpn, tor and datacenter factors, got %#v", detected)
	}
}

func TestBuildIPQualityVerdictEvidenceAndRecommendations(t *testing.T) {
	report := &models.IPQualityReport{
		PublicIP:     "192.0.2.10",
		IPType:       "datacenter_likely",
		RiskLevel:    "medium",
		RiskScore:    45,
		ASN:          "64500",
		Organization: "Example Hosting",
		ISP:          "Example ISP",
		BlacklistSummary: &models.IPBlacklistSummary{
			Total:  2,
			Clean:  1,
			Listed: 1,
		},
		RiskFactors: []*models.IPRiskFactor{
			{Name: "datacenter", Detected: true, Confidence: "medium", Detail: "hosting keyword"},
		},
		MailChecks: []*models.MailPortCheck{
			{Provider: "Example", Target: "smtp.example", Port: 25, Status: "blocked"},
		},
		MailSummary:  &models.MailPortSummary{Total: 1, Providers: 1, Blocked: 1},
		NetworkStack: &models.IPNetworkStack{DetectedVersion: "IPv4", IPv4Available: true},
	}

	verdict := buildIPQualityVerdict(report)
	if verdict.Grade != "C" {
		t.Fatalf("expected grade C, got %#v", verdict)
	}
	if !verdict.HostingHint || verdict.MailUsable {
		t.Fatalf("unexpected verdict flags: %#v", verdict)
	}

	evidence := buildIPQualityEvidence(report)
	if len(evidence) < 6 {
		t.Fatalf("expected evidence rows, got %#v", evidence)
	}

	recommendations := buildIPQualityRecommendations(report)
	if len(recommendations) == 0 {
		t.Fatal("expected recommendations")
	}

	summary := buildIPQualityEvidenceSummary(report)
	if len(summary) < 5 {
		t.Fatalf("expected evidence summary rows, got %#v", summary)
	}
	byCategory := map[string]*models.EvidenceSummary{}
	for _, item := range summary {
		byCategory[item.Category] = item
	}
	if byCategory["dnsbl"].Status != "warning" || byCategory["dnsbl"].Confidence != "high" {
		t.Fatalf("expected DNSBL warning/high evidence summary, got %#v", byCategory["dnsbl"])
	}
	if byCategory["mail"].Status != "warning" {
		t.Fatalf("expected mail warning evidence summary, got %#v", byCategory["mail"])
	}
	if byCategory["external_api"].Status != "skipped" {
		t.Fatalf("expected external API skipped summary, got %#v", byCategory["external_api"])
	}
}

func TestIPQualityDefaultsIncludeExpandedChecks(t *testing.T) {
	scanner := NewIPQualityScanner(nil)

	if len(scanner.blacklistZones) < 20 {
		t.Fatalf("expected expanded DNSBL list, got %d", len(scanner.blacklistZones))
	}
	if len(scanner.mailTargets) < 12 {
		t.Fatalf("expected expanded mail target list, got %d", len(scanner.mailTargets))
	}
	providers := map[string]bool{}
	for _, target := range scanner.mailTargets {
		providers[target.Provider] = true
	}
	for _, provider := range []string{"Gmail", "Outlook", "Yahoo", "Apple", "QQ", "Mail.ru", "AOL", "GMX", "Mail.com", "163", "Sohu", "Sina"} {
		if !providers[provider] {
			t.Fatalf("expected provider %s in mail matrix", provider)
		}
	}
}

func TestSummarizeMailPorts(t *testing.T) {
	summary := summarizeMailPorts([]*models.MailPortCheck{
		{Provider: "Gmail", Status: "reachable", Reachable: true},
		{Provider: "Gmail", Status: "blocked"},
		{Provider: "Outlook", Status: "timeout"},
		{Provider: "Yahoo", Status: "blocked"},
	})

	if summary.Total != 4 || summary.Reachable != 1 || summary.Blocked != 2 || summary.Timeout != 1 {
		t.Fatalf("unexpected mail summary counts: %#v", summary)
	}
	if summary.Providers != 3 || summary.ProviderOpen != 1 {
		t.Fatalf("unexpected mail provider counts: %#v", summary)
	}
}

func TestBuildIPRiskSources(t *testing.T) {
	report := &models.IPQualityReport{
		ASN:          "64500",
		Organization: "Example Hosting",
		ReverseDNS:   []string{"host.example"},
		BlacklistSummary: &models.IPBlacklistSummary{
			Total:  3,
			Clean:  2,
			Listed: 1,
		},
		RiskFactors: []*models.IPRiskFactor{
			{Name: "datacenter", Detected: true},
		},
	}

	sources := buildIPRiskSources(report)
	statusByName := map[string]string{}
	for _, source := range sources {
		statusByName[source.Name] = source.Status
	}
	if statusByName["Team Cymru ASN"] != "available" {
		t.Fatalf("expected Team Cymru source available, got %#v", statusByName)
	}
	if statusByName["DNSBL"] != "listed" {
		t.Fatalf("expected DNSBL listed, got %#v", statusByName)
	}
	if statusByName["Commercial Risk APIs"] != "disabled" {
		t.Fatalf("expected commercial APIs disabled, got %#v", statusByName)
	}
}

func netParseIP(t *testing.T, value string) net.IP {
	t.Helper()
	ip := net.ParseIP(value)
	if ip == nil {
		t.Fatalf("invalid IP fixture: %s", value)
	}
	return ip
}
