package cli

import "testing"

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
	if !app.config.EnableRouteTrace || !app.config.EnableStreaming || !app.config.EnableAIServices || !app.config.EnableSecurityScan {
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
