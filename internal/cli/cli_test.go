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
	args := []string{"--full", "--iperf3-server", "127.0.0.1:5201", "--network-backend", "builtin", "--disk-backend", "builtin", "--cpu-backend", "builtin"}
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
