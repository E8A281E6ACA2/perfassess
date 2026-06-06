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

func netParseIP(t *testing.T, value string) net.IP {
	t.Helper()
	ip := net.ParseIP(value)
	if ip == nil {
		t.Fatalf("invalid IP fixture: %s", value)
	}
	return ip
}
