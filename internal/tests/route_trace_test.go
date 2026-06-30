package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
)

func TestEnrichRouteTraceResultBuildsQualityEvidence(t *testing.T) {
	tracer := &RouteTracer{}
	result := tracer.enrichResult(&models.TraceResult{
		Target:  "www.189.cn",
		Success: true,
		Hops: []*models.Hop{
			{Number: 1, IP: "192.0.2.1", Latency: 2 * time.Millisecond},
			{Number: 2, IP: "*"},
			{Number: 3, IP: "203.0.113.1", Hostname: "edge.example", Latency: 50 * time.Millisecond},
		},
	})

	if result.DirectionGroup != "china_reference" {
		t.Fatalf("expected china direction group, got %q", result.DirectionGroup)
	}
	if result.IsRealReturnRoute {
		t.Fatal("expected built-in traceroute to be marked as non-return-route")
	}
	if result.TimeoutHops != 1 {
		t.Fatalf("expected one timeout hop, got %d", result.TimeoutHops)
	}
	if result.LastVisibleHop != "203.0.113.1 (edge.example)" {
		t.Fatalf("unexpected last visible hop: %q", result.LastVisibleHop)
	}
	if result.Quality == nil || result.Quality.DirectionLabel != "国内方向参考" || result.Quality.Grade == "" {
		t.Fatalf("unexpected route quality: %#v", result.Quality)
	}
	if len(result.Evidence) == 0 {
		t.Fatal("expected route evidence")
	}
	if len(result.EvidenceSummary) < 4 {
		t.Fatalf("expected route evidence summary, got %#v", result.EvidenceSummary)
	}
	summaryByCategory := map[string]*models.EvidenceSummary{}
	for _, item := range result.EvidenceSummary {
		summaryByCategory[item.Category] = item
	}
	if summaryByCategory["return_route_boundary"].Status != "warning" || summaryByCategory["return_route_boundary"].Confidence != "high" {
		t.Fatalf("expected return route boundary warning/high summary, got %#v", summaryByCategory["return_route_boundary"])
	}
	if summaryByCategory["visibility"].Status != "partial" {
		t.Fatalf("expected visibility partial summary with timeout hop, got %#v", summaryByCategory["visibility"])
	}
	if !routeRecommendationsContain(result.Recommendations, "不是真实回程") {
		t.Fatalf("expected return-route boundary recommendation, got %#v", result.Recommendations)
	}
}

func TestRouteQualityGradeDetectsFailedTrace(t *testing.T) {
	tracer := &RouteTracer{}
	result := tracer.enrichResult(&models.TraceResult{
		Target:       "cloudflare.com",
		Success:      false,
		ErrorMessage: "追踪超时",
	})

	if result.Quality == nil || result.Quality.Grade != "D" || result.Quality.Status != "failed" {
		t.Fatalf("expected failed grade D, got %#v", result.Quality)
	}
	if !routeRecommendationsContain(result.Recommendations, "路由追踪失败") {
		t.Fatalf("expected failure recommendation, got %#v", result.Recommendations)
	}
}

func routeRecommendationsContain(items []string, needle string) bool {
	for _, item := range items {
		if strings.Contains(item, needle) {
			return true
		}
	}
	return false
}
