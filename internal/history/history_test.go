package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
)

func TestEntryFromReportExtractsScoresAndProfiles(t *testing.T) {
	report := testReport("session-a", 80, 70, 90, 60, 75, "server", time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC))

	entry := EntryFromReport(report, "a.json")

	if entry.ReportPath != "a.json" {
		t.Fatalf("expected report path, got %q", entry.ReportPath)
	}
	if entry.TotalScore != 75 || entry.CPUScore != 80 {
		t.Fatalf("expected scores to be extracted, got %#v", entry)
	}
	if entry.ScoreProfile != "server" {
		t.Fatalf("expected score profile server, got %q", entry.ScoreProfile)
	}
	if entry.BenchmarkProfile["name"] != "full_iperf3" {
		t.Fatalf("expected benchmark profile, got %#v", entry.BenchmarkProfile)
	}
	if entry.HostID == "" || entry.HostID == "unknown" {
		t.Fatalf("expected host id to be derived, got %q", entry.HostID)
	}
}

func TestAddReportAndLoadEntries(t *testing.T) {
	dir := t.TempDir()
	store := filepath.Join(dir, "history.jsonl")
	reportPath := filepath.Join(dir, "report.json")
	writeReport(t, reportPath, testReport("session-a", 80, 70, 90, 60, 75, "server", time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)))

	added, err := AddReport(store, reportPath)
	if err != nil {
		t.Fatalf("expected add report to succeed, got %v", err)
	}
	if added.SessionID != "session-a" {
		t.Fatalf("expected added entry session id, got %q", added.SessionID)
	}

	entries, err := LoadEntries(store)
	if err != nil {
		t.Fatalf("expected load entries to succeed, got %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one history entry, got %d", len(entries))
	}
	if entries[0].TotalScore != 75 {
		t.Fatalf("expected total score 75, got %.2f", entries[0].TotalScore)
	}
}

func TestBuildTrendCalculatesDeltas(t *testing.T) {
	host := "linux|amd64|Example CPU|8|8192"
	entries := []Entry{
		{HostID: host, Timestamp: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC), TotalScore: 80, CPUScore: 70, MemoryScore: 75, DiskScore: 85, NetworkScore: 90},
		{HostID: host, Timestamp: time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC), TotalScore: 90, CPUScore: 80, MemoryScore: 80, DiskScore: 95, NetworkScore: 90},
	}

	trend, err := BuildTrend(entries, host)
	if err != nil {
		t.Fatalf("expected trend to succeed, got %v", err)
	}
	if trend.Count != 2 {
		t.Fatalf("expected count 2, got %d", trend.Count)
	}
	if trend.TotalDelta.Delta != 10 {
		t.Fatalf("expected total delta 10, got %.2f", trend.TotalDelta.Delta)
	}
	if trend.CPUDelta.DeltaPercent <= 0 {
		t.Fatalf("expected positive cpu percent delta, got %.2f", trend.CPUDelta.DeltaPercent)
	}
}

func TestBuildTrendRequiresHostWhenMultipleHostsExist(t *testing.T) {
	entries := []Entry{
		{HostID: "host-a", Timestamp: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC), TotalScore: 80},
		{HostID: "host-b", Timestamp: time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC), TotalScore: 90},
	}

	_, err := BuildTrend(entries, "")
	if err == nil {
		t.Fatal("expected multiple hosts without filter to fail")
	}
	if !strings.Contains(err.Error(), "--host") {
		t.Fatalf("expected host filter hint, got %v", err)
	}
}

func TestEntriesFromDirAndSortEntries(t *testing.T) {
	dir := t.TempDir()
	writeReport(t, filepath.Join(dir, "a.json"), testReport("a", 70, 70, 70, 70, 70, "server", time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)))
	writeReport(t, filepath.Join(dir, "b.json"), testReport("b", 90, 90, 90, 90, 90, "server", time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)))
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{"), 0644); err != nil {
		t.Fatalf("failed to write bad json: %v", err)
	}

	entries, err := EntriesFromDir(dir)
	if err != nil {
		t.Fatalf("expected entries from dir to succeed, got %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected two usable entries, got %d", len(entries))
	}
	if err := SortEntries(entries, "total", true); err != nil {
		t.Fatalf("expected sort to succeed, got %v", err)
	}
	if entries[0].TotalScore != 90 {
		t.Fatalf("expected highest score first, got %.2f", entries[0].TotalScore)
	}
}

func TestFormatTextOutputsUsefulFields(t *testing.T) {
	entry := Entry{
		ReportPath:       "report.json",
		Timestamp:        time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		HostID:           "host-a",
		ScoreProfile:     "server",
		BenchmarkProfile: map[string]interface{}{"name": "full_iperf3"},
		TotalScore:       88,
		CPUScore:         90,
	}

	text := FormatEntriesText([]Entry{entry})
	for _, snippet := range []string{"历史报告列表", "host=host-a", "total=88.00", "report.json"} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("expected text to contain %q, got:\n%s", snippet, text)
		}
	}
}

func testReport(sessionID string, cpu, memory, disk, network, total float64, profile string, timestamp time.Time) *models.Report {
	return &models.Report{
		SessionID: sessionID,
		Timestamp: timestamp,
		SystemInfo: &models.SystemInfo{
			CPU: &models.CPUInfo{
				Model:   "Example CPU",
				Cores:   4,
				Threads: 8,
			},
			Memory: &models.MemoryInfo{
				TotalMB: 8192,
			},
			OS: &models.OSInfo{
				Name:         "linux",
				Version:      "example",
				Architecture: "amd64",
			},
		},
		Summary: map[string]interface{}{
			"cpu_score":     cpu,
			"memory_score":  memory,
			"disk_score":    disk,
			"network_score": network,
			"total_score":   total,
			"grade":         "良好",
			"score_profile": profile,
			"benchmark_profile": map[string]interface{}{
				"name": "full_iperf3",
			},
		},
	}
}

func writeReport(t *testing.T, path string, report *models.Report) {
	t.Helper()
	content, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("failed to marshal report: %v", err)
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("failed to write report: %v", err)
	}
}
