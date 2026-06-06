package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
)

func TestBindFlagsQuickPreset(t *testing.T) {
	app := NewCLI()
	cmd := app.GetRootCmd()
	cmd.SetArgs([]string{"--quick"})
	if err := cmd.ParseFlags([]string{"--quick"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := app.bindFlags(cmd); err != nil {
		t.Fatalf("expected quick bind to succeed, got %v", err)
	}

	assertStringSlice(t, app.config.Tests, []string{"cpu", "memory", "disk"})
	if app.config.DiskBackend != "builtin" {
		t.Fatalf("expected quick preset disk backend builtin, got %q", app.config.DiskBackend)
	}
	if app.config.CPUBackend != "builtin" {
		t.Fatalf("expected quick preset cpu backend builtin, got %q", app.config.CPUBackend)
	}
	if app.config.MemoryBackend != "builtin" {
		t.Fatalf("expected quick preset memory backend builtin, got %q", app.config.MemoryBackend)
	}
	if app.config.EnableRouteTrace {
		t.Fatal("expected quick preset to disable route trace")
	}
	if app.config.EnableIPQuality {
		t.Fatal("expected quick preset to disable ip quality checks")
	}
}

func TestVersionCommandUsesInjectedVersion(t *testing.T) {
	app := NewCLIWithVersion("v9.9.9-test")
	cmd := app.GetRootCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected version command to succeed, got %v", err)
	}
	if strings.TrimSpace(output.String()) != "perfassess v9.9.9-test" {
		t.Fatalf("unexpected version output: %q", output.String())
	}
}

func TestBindFlagsFullPresetWithIperf3Server(t *testing.T) {
	app := NewCLI()
	cmd := app.GetRootCmd()
	args := []string{"--full", "--iperf3-server", "127.0.0.1:5201"}
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := app.bindFlags(cmd); err != nil {
		t.Fatalf("expected full bind to succeed, got %v", err)
	}

	assertStringSlice(t, app.config.Tests, []string{"all"})
	if app.config.DiskBackend != "fio" {
		t.Fatalf("expected full preset disk backend fio, got %q", app.config.DiskBackend)
	}
	if app.config.CPUBackend != "sysbench" {
		t.Fatalf("expected full preset cpu backend sysbench, got %q", app.config.CPUBackend)
	}
	if app.config.MemoryBackend != "sysbench" {
		t.Fatalf("expected full preset memory backend sysbench, got %q", app.config.MemoryBackend)
	}
	if app.config.NetworkBackend != "iperf3" {
		t.Fatalf("expected full preset with server to use iperf3, got %q", app.config.NetworkBackend)
	}
	if !app.config.EnableRouteTrace || !app.config.EnableStreaming || !app.config.EnableAIServices || !app.config.EnableIPQuality || !app.config.EnableStressTest || !app.config.EnableSecurityScan {
		t.Fatal("expected full preset to enable optional checks")
	}
}

func TestBindFlagsFullPresetAllowsExplicitBackendOverride(t *testing.T) {
	app := NewCLI()
	cmd := app.GetRootCmd()
	args := []string{"--full", "--iperf3-server", "127.0.0.1:5201", "--network-backend", "builtin", "--disk-backend", "builtin", "--cpu-backend", "builtin", "--memory-backend", "builtin"}
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := app.bindFlags(cmd); err != nil {
		t.Fatalf("expected full bind to succeed, got %v", err)
	}

	if app.config.NetworkBackend != "builtin" {
		t.Fatalf("expected explicit network backend override, got %q", app.config.NetworkBackend)
	}
	if app.config.DiskBackend != "builtin" {
		t.Fatalf("expected explicit disk backend override, got %q", app.config.DiskBackend)
	}
	if app.config.CPUBackend != "builtin" {
		t.Fatalf("expected explicit cpu backend override, got %q", app.config.CPUBackend)
	}
	if app.config.MemoryBackend != "builtin" {
		t.Fatalf("expected explicit memory backend override, got %q", app.config.MemoryBackend)
	}
}

func TestBindFlagsVPSProfilePreset(t *testing.T) {
	app := NewCLI()
	cmd := app.GetRootCmd()
	args := []string{"--vps-profile"}
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := app.bindFlags(cmd); err != nil {
		t.Fatalf("expected vps profile bind to succeed, got %v", err)
	}

	assertStringSlice(t, app.config.Tests, []string{"all"})
	if app.config.ScoreProfile != "vps" {
		t.Fatalf("expected vps score profile, got %q", app.config.ScoreProfile)
	}
	if app.config.CPUBackend != "sysbench" {
		t.Fatalf("expected vps profile cpu backend sysbench, got %q", app.config.CPUBackend)
	}
	if app.config.MemoryBackend != "sysbench" {
		t.Fatalf("expected vps profile memory backend sysbench, got %q", app.config.MemoryBackend)
	}
	if app.config.DiskBackend != "fio" {
		t.Fatalf("expected vps profile disk backend fio, got %q", app.config.DiskBackend)
	}
	if app.config.NetworkBackend != "speedtest" {
		t.Fatalf("expected vps profile network backend speedtest, got %q", app.config.NetworkBackend)
	}
	if app.config.NetworkProfile != "standard" {
		t.Fatalf("expected vps profile network profile standard, got %q", app.config.NetworkProfile)
	}
	if !app.config.EnableRouteTrace {
		t.Fatal("expected vps profile to enable route trace")
	}
	if !app.config.EnableStreaming {
		t.Fatal("expected vps profile to enable streaming checks")
	}
	if app.config.EnableAIServices {
		t.Fatal("expected vps profile to keep ai service checks disabled")
	}
	if !app.config.EnableIPQuality {
		t.Fatal("expected vps profile to enable ip quality checks")
	}
	if app.config.EnableStressTest {
		t.Fatal("expected vps profile to keep stress test disabled")
	}
	if app.config.EnableSecurityScan {
		t.Fatal("expected vps profile to keep security scan disabled")
	}
}

func TestBindFlagsVPSProfileAllowsExplicitOverrides(t *testing.T) {
	app := NewCLI()
	cmd := app.GetRootCmd()
	args := []string{
		"--vps-profile",
		"--network-backend", "builtin",
		"--disk-backend", "builtin",
		"--cpu-backend", "builtin",
		"--memory-backend", "builtin",
		"--score-profile", "server",
		"--route-trace=false",
		"--streaming=false",
		"--ip-quality=false",
	}
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := app.bindFlags(cmd); err != nil {
		t.Fatalf("expected vps profile bind to succeed, got %v", err)
	}

	if app.config.NetworkBackend != "builtin" {
		t.Fatalf("expected explicit network backend override, got %q", app.config.NetworkBackend)
	}
	if app.config.DiskBackend != "builtin" {
		t.Fatalf("expected explicit disk backend override, got %q", app.config.DiskBackend)
	}
	if app.config.CPUBackend != "builtin" {
		t.Fatalf("expected explicit cpu backend override, got %q", app.config.CPUBackend)
	}
	if app.config.MemoryBackend != "builtin" {
		t.Fatalf("expected explicit memory backend override, got %q", app.config.MemoryBackend)
	}
	if app.config.ScoreProfile != "server" {
		t.Fatalf("expected explicit score profile override, got %q", app.config.ScoreProfile)
	}
	if app.config.EnableRouteTrace {
		t.Fatal("expected explicit route trace override")
	}
	if app.config.EnableStreaming {
		t.Fatal("expected explicit streaming override")
	}
	if app.config.EnableIPQuality {
		t.Fatal("expected explicit ip quality override")
	}
}

func TestBindFlagsAcceptsNetworkProfile(t *testing.T) {
	app := NewCLI()
	cmd := app.GetRootCmd()
	args := []string{"--network-profile", "standard"}
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := app.bindFlags(cmd); err != nil {
		t.Fatalf("expected network profile bind to succeed, got %v", err)
	}

	if app.config.NetworkProfile != "standard" {
		t.Fatalf("expected network profile standard, got %q", app.config.NetworkProfile)
	}
	if len(app.config.RouteTraceTargets) <= 3 {
		t.Fatalf("expected standard network profile to expand route targets, got %#v", app.config.RouteTraceTargets)
	}
}

func TestBindFlagsFullPresetUsesFullNetworkProfile(t *testing.T) {
	app := NewCLI()
	cmd := app.GetRootCmd()
	args := []string{"--full"}
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := app.bindFlags(cmd); err != nil {
		t.Fatalf("expected full preset bind to succeed, got %v", err)
	}

	if app.config.NetworkProfile != "full" {
		t.Fatalf("expected full preset network profile full, got %q", app.config.NetworkProfile)
	}
	if len(app.config.RouteTraceTargets) <= 6 {
		t.Fatalf("expected full network profile to expand route targets, got %#v", app.config.RouteTraceTargets)
	}
}

func TestBindFlagsVPSProfileWithIperf3ServerUsesIperf3(t *testing.T) {
	app := NewCLI()
	cmd := app.GetRootCmd()
	args := []string{"--vps-profile", "--iperf3-server", "127.0.0.1:5201"}
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := app.bindFlags(cmd); err != nil {
		t.Fatalf("expected vps profile bind to succeed, got %v", err)
	}

	if app.config.NetworkBackend != "iperf3" {
		t.Fatalf("expected vps profile with iperf3 server to use iperf3, got %q", app.config.NetworkBackend)
	}
}

func TestBindFlagsVPSProfileWithIperf3ServerFileUsesIperf3(t *testing.T) {
	app := NewCLI()
	cmd := app.GetRootCmd()
	args := []string{"--vps-profile", "--iperf3-server-file", "docs/examples/iperf3-servers.txt"}
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := app.bindFlags(cmd); err != nil {
		t.Fatalf("expected vps profile bind to succeed, got %v", err)
	}

	if app.config.NetworkBackend != "iperf3" {
		t.Fatalf("expected vps profile with iperf3 server file to use iperf3, got %q", app.config.NetworkBackend)
	}
	if app.config.Iperf3ServerFile != "docs/examples/iperf3-servers.txt" {
		t.Fatalf("expected iperf3 server file to bind, got %q", app.config.Iperf3ServerFile)
	}
}

func TestBindFlagsAcceptsSpeedtestNetworkBackend(t *testing.T) {
	app := NewCLI()
	cmd := app.GetRootCmd()
	args := []string{"--network-backend", "speedtest"}
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := app.bindFlags(cmd); err != nil {
		t.Fatalf("expected speedtest backend bind to succeed, got %v", err)
	}

	if app.config.NetworkBackend != "speedtest" {
		t.Fatalf("expected speedtest backend, got %q", app.config.NetworkBackend)
	}
}

func TestBindFlagsAcceptsIperf3Servers(t *testing.T) {
	app := NewCLI()
	cmd := app.GetRootCmd()
	args := []string{"--full", "--iperf3-servers", "node-a:5201,node-b:5201"}
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := app.bindFlags(cmd); err != nil {
		t.Fatalf("expected iperf3 servers bind to succeed, got %v", err)
	}

	if app.config.NetworkBackend != "iperf3" {
		t.Fatalf("expected full preset with iperf3 servers to use iperf3, got %q", app.config.NetworkBackend)
	}
	assertStringSlice(t, app.config.Iperf3Servers, []string{"node-a:5201", "node-b:5201"})
}

func TestBindFlagsAcceptsGeekbenchCPUBackend(t *testing.T) {
	app := NewCLI()
	cmd := app.GetRootCmd()
	args := []string{"--cpu-backend", "geekbench"}
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := app.bindFlags(cmd); err != nil {
		t.Fatalf("expected geekbench backend bind to succeed, got %v", err)
	}

	if app.config.CPUBackend != "geekbench" {
		t.Fatalf("expected geekbench backend, got %q", app.config.CPUBackend)
	}
}

func TestBindFlagsOutputFormat(t *testing.T) {
	app := NewCLI()
	cmd := app.GetRootCmd()
	if err := cmd.ParseFlags([]string{"--output-format", "json"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := app.bindFlags(cmd); err != nil {
		t.Fatalf("expected output format bind to succeed, got %v", err)
	}

	if app.config.OutputFormat != "json" {
		t.Fatalf("expected json output format, got %q", app.config.OutputFormat)
	}
}

func TestBindFlagsScoreWeights(t *testing.T) {
	app := NewCLI()
	cmd := app.GetRootCmd()
	args := []string{"--score-weights", "cpu=0.4,memory=0.1,disk=0.3,network=0.2"}
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := app.bindFlags(cmd); err != nil {
		t.Fatalf("expected score weights bind to succeed, got %v", err)
	}

	if app.config.ScoreWeights["cpu"] != 0.4 {
		t.Fatalf("expected cpu weight 0.4, got %.2f", app.config.ScoreWeights["cpu"])
	}
	if app.config.ScoreWeights["memory"] != 0.1 {
		t.Fatalf("expected memory weight 0.1, got %.2f", app.config.ScoreWeights["memory"])
	}
}

func TestBindFlagsScoreWeightsRejectsMissingComponent(t *testing.T) {
	app := NewCLI()
	cmd := app.GetRootCmd()
	args := []string{"--score-weights", "cpu=0.4,memory=0.1,disk=0.3"}
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := app.bindFlags(cmd); err == nil {
		t.Fatal("expected missing score weight to fail")
	}
}

func TestBindFlagsScoreProfile(t *testing.T) {
	app := NewCLI()
	cmd := app.GetRootCmd()
	args := []string{"--score-profile", "vps"}
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := app.bindFlags(cmd); err != nil {
		t.Fatalf("expected score profile bind to succeed, got %v", err)
	}

	if app.config.ScoreProfile != "vps" {
		t.Fatalf("expected score profile vps, got %q", app.config.ScoreProfile)
	}
}

func TestValidateFlagsRejectsUnknownScoreProfile(t *testing.T) {
	app := NewCLI()
	app.config.ScoreProfile = "unknown"

	if err := app.validateFlags(); err == nil {
		t.Fatal("expected invalid score profile to fail validation")
	}
}

func TestCompareCommandOutputsJSON(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, "a.json")
	pathB := filepath.Join(dir, "b.json")
	writeCLIReport(t, pathA, 80, "server")
	writeCLIReport(t, pathB, 90, "server")

	app := NewCLI()
	cmd := app.GetRootCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"compare", pathA, pathB, "--format", "json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected compare command to succeed, got %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("expected compare command to output valid json, got %v\n%s", err, output.String())
	}
	if decoded["winner"] != "B" {
		t.Fatalf("expected winner B, got %#v", decoded["winner"])
	}
}

func TestCompareCommandRejectsInvalidFormat(t *testing.T) {
	app := NewCLI()
	cmd := app.GetRootCmd()
	cmd.SetArgs([]string{"compare", "a.json", "b.json", "--format", "xml"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected invalid compare format to fail")
	}
	if !strings.Contains(err.Error(), "无效的对比输出格式") {
		t.Fatalf("expected invalid format error, got %v", err)
	}
}

func TestHistoryCommandsAddListAndTrend(t *testing.T) {
	dir := t.TempDir()
	store := filepath.Join(dir, "history.jsonl")
	pathA := filepath.Join(dir, "a.json")
	pathB := filepath.Join(dir, "b.json")
	writeCLIReport(t, pathA, 80, "server")
	writeCLIReport(t, pathB, 90, "server")

	app := NewCLI()
	addCmd := app.GetRootCmd()
	addCmd.SetArgs([]string{"history", "add", pathA, "--store", store})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("expected history add a to succeed, got %v", err)
	}

	app = NewCLI()
	addCmd = app.GetRootCmd()
	addCmd.SetArgs([]string{"history", "add", pathB, "--store", store})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("expected history add b to succeed, got %v", err)
	}

	app = NewCLI()
	listCmd := app.GetRootCmd()
	var listOutput bytes.Buffer
	listCmd.SetOut(&listOutput)
	listCmd.SetArgs([]string{"history", "list", "--store", store, "--format", "json"})
	if err := listCmd.Execute(); err != nil {
		t.Fatalf("expected history list to succeed, got %v", err)
	}
	var entries []map[string]interface{}
	if err := json.Unmarshal(listOutput.Bytes(), &entries); err != nil {
		t.Fatalf("expected history list json, got %v\n%s", err, listOutput.String())
	}
	if len(entries) != 2 {
		t.Fatalf("expected two history entries, got %d", len(entries))
	}

	app = NewCLI()
	trendCmd := app.GetRootCmd()
	var trendOutput bytes.Buffer
	trendCmd.SetOut(&trendOutput)
	trendCmd.SetArgs([]string{"history", "trend", "--store", store, "--format", "json"})
	if err := trendCmd.Execute(); err != nil {
		t.Fatalf("expected history trend to succeed, got %v", err)
	}
	var trend map[string]interface{}
	if err := json.Unmarshal(trendOutput.Bytes(), &trend); err != nil {
		t.Fatalf("expected history trend json, got %v\n%s", err, trendOutput.String())
	}
	if trend["count"] != float64(2) {
		t.Fatalf("expected trend count 2, got %#v", trend["count"])
	}
}

func TestCompareDirCommandOutputsRankedJSON(t *testing.T) {
	dir := t.TempDir()
	writeCLIReport(t, filepath.Join(dir, "a.json"), 80, "server")
	writeCLIReport(t, filepath.Join(dir, "b.json"), 90, "server")

	app := NewCLI()
	cmd := app.GetRootCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"compare-dir", dir, "--sort-by", "total", "--format", "json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected compare-dir to succeed, got %v", err)
	}

	var entries []map[string]interface{}
	if err := json.Unmarshal(output.Bytes(), &entries); err != nil {
		t.Fatalf("expected compare-dir json, got %v\n%s", err, output.String())
	}
	if len(entries) != 2 {
		t.Fatalf("expected two entries, got %d", len(entries))
	}
	if entries[0]["total_score"] != float64(90) {
		t.Fatalf("expected highest score first, got %#v", entries[0]["total_score"])
	}
}

func assertStringSlice(t *testing.T, actual []string, expected []string) {
	t.Helper()

	if len(actual) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
	for i := range expected {
		if actual[i] != expected[i] {
			t.Fatalf("expected %v, got %v", expected, actual)
		}
	}
}

func writeCLIReport(t *testing.T, path string, total float64, profile string) {
	t.Helper()
	report := &models.Report{
		SessionID: "cli_test",
		Timestamp: testTimestampFromScore(total),
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
			"cpu_score":     total,
			"memory_score":  total,
			"disk_score":    total,
			"network_score": total,
			"total_score":   total,
			"grade":         "良好",
			"score_profile": profile,
		},
	}
	content, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("failed to marshal report: %v", err)
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("failed to write report: %v", err)
	}
}

func testTimestampFromScore(score float64) time.Time {
	return time.Date(2026, 6, 1, int(score)-70, 0, 0, 0, time.UTC)
}
