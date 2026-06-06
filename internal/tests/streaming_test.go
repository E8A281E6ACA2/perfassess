package tests

import "testing"

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
	if full[len(full)-1] != "TVB Anywhere" {
		t.Fatalf("expected full profile to keep stable configured order, got last=%q", full[len(full)-1])
	}
}

func TestStreamingProfileFallbacksToQuick(t *testing.T) {
	detector := NewStreamingDetectorWithProfile(newTestLogger(t), "unknown")

	if detector.profile != "quick" {
		t.Fatalf("expected unknown streaming profile to fallback to quick, got %q", detector.profile)
	}
}
