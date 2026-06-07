package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStreamingProfilesSelectExpectedPlatforms(t *testing.T) {
	detector := NewStreamingDetectorWithProfile(newTestLogger(t), "quick")
	quick := detector.platformsForProfile()
	if len(quick) != 8 {
		t.Fatalf("expected quick streaming profile to use 8 platforms, got %#v", quick)
	}
	if quick[0] != "Netflix" || quick[1] != "YouTube" {
		t.Fatalf("expected stable platform order, got %#v", quick[:2])
	}

	detector.SetProfile("standard")
	standard := detector.platformsForProfile()
	if len(standard) <= len(quick) {
		t.Fatalf("expected standard streaming profile to add platforms, quick=%d standard=%d", len(quick), len(standard))
	}

	detector.SetProfile("full")
	full := detector.platformsForProfile()
	if len(full) <= len(standard) {
		t.Fatalf("expected full streaming profile to add platforms, standard=%d full=%d", len(standard), len(full))
	}
	if full[len(full)-1] != "Viu" {
		t.Fatalf("expected full profile to keep stable configured order, got last=%q", full[len(full)-1])
	}
}

func TestStreamingProfileFallbacksToQuick(t *testing.T) {
	detector := NewStreamingDetectorWithProfile(newTestLogger(t), "unknown")

	if detector.profile != "quick" {
		t.Fatalf("expected unknown streaming profile to fallback to quick, got %q", detector.profile)
	}
}

func TestStreamingResultIncludesReportMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	detector := NewStreamingDetectorWithProfile(newTestLogger(t), "quick")
	detector.AddCustomPlatform(StreamingPlatform{
		Name:       "Example Video",
		TestURL:    server.URL,
		Category:   "jp",
		RegionHint: "JP",
		Protocol:   "ipv4",
		CheckFunc:  basicStatusCheck("Unknown"),
	})

	result, err := detector.CheckPlatform("Example Video")
	if err != nil {
		t.Fatalf("expected custom streaming check to succeed, got %v", err)
	}
	if result.Category != "jp" {
		t.Fatalf("expected category metadata, got %q", result.Category)
	}
	if result.Region != "JP" {
		t.Fatalf("expected region to fallback to hint, got %q", result.Region)
	}
	if result.RegionSource != "platform_hint" {
		t.Fatalf("expected platform_hint region source, got %q", result.RegionSource)
	}
	if result.UnlockType != "full" {
		t.Fatalf("expected full unlock type, got %q", result.UnlockType)
	}
	if result.Protocol != "ipv4" {
		t.Fatalf("expected protocol metadata, got %q", result.Protocol)
	}
}

func TestStreamingUnlockTypeLabels(t *testing.T) {
	cases := []struct {
		available bool
		message   string
		expected  string
	}{
		{available: true, message: "完全解锁", expected: "full"},
		{available: true, message: "仅限自制内容", expected: "partial"},
		{available: true, message: "需要登录", expected: "login_required"},
		{available: false, message: "地区限制", expected: "blocked"},
	}

	for _, tt := range cases {
		if got := streamingUnlockType(tt.available, tt.message); got != tt.expected {
			t.Fatalf("expected unlock type %q for %q, got %q", tt.expected, tt.message, got)
		}
	}
}
