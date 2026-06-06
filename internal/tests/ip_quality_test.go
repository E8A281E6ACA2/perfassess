package tests

import (
	"testing"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
)

func TestCalculateIPRiskLevels(t *testing.T) {
	report := &models.IPQualityReport{
		IPType: "datacenter_likely",
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
