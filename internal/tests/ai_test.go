package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAIServiceResultIncludesReportMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	detector := NewAIServiceDetector(newTestLogger(t))
	detector.services["ExampleAI"] = AIService{
		Name:       "ExampleAI",
		TestURL:    server.URL,
		Category:   "coding",
		RegionHint: "Global",
		CheckFunc: func(resp *http.Response, _ string) (bool, string) {
			return resp.StatusCode == http.StatusOK, "可访问"
		},
	}

	result, err := detector.CheckService("ExampleAI")
	if err != nil {
		t.Fatalf("expected AI service check to succeed, got %v", err)
	}
	if result.Category != "coding" {
		t.Fatalf("expected category metadata, got %q", result.Category)
	}
	if result.AccessType != "full" {
		t.Fatalf("expected full access type, got %q", result.AccessType)
	}
	if result.RegionHint != "Global" {
		t.Fatalf("expected region hint metadata, got %q", result.RegionHint)
	}
	if len(result.EvidenceSummary) != 3 {
		t.Fatalf("expected AI evidence summary, got %#v", result.EvidenceSummary)
	}
	if result.EvidenceSummary[0].Category != "access" || result.EvidenceSummary[0].Status != "success" {
		t.Fatalf("unexpected access evidence summary: %#v", result.EvidenceSummary[0])
	}
	if result.EvidenceSummary[2].Category != "account" || result.EvidenceSummary[2].Status != "skipped" {
		t.Fatalf("unexpected account evidence summary: %#v", result.EvidenceSummary[2])
	}
}

func TestAIAccessTypeLabels(t *testing.T) {
	cases := []struct {
		available bool
		message   string
		expected  string
	}{
		{available: true, message: "可访问", expected: "full"},
		{available: true, message: "需要登录", expected: "login_required"},
		{available: true, message: "需要额外验证", expected: "verification_required"},
		{available: true, message: "请求过于频繁或地区限制", expected: "rate_limited"},
		{available: false, message: "地区限制", expected: "restricted"},
	}

	for _, tt := range cases {
		if got := aiAccessType(tt.available, tt.message); got != tt.expected {
			t.Fatalf("expected access type %q for %q, got %q", tt.expected, tt.message, got)
		}
	}
}
