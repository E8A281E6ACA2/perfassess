package config

import "testing"

func TestDefaultConfigUsesBuiltinNetworkBackend(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.NetworkBackend != "builtin" {
		t.Fatalf("expected default network backend builtin, got %q", cfg.NetworkBackend)
	}
	if cfg.DiskBackend != "builtin" {
		t.Fatalf("expected default disk backend builtin, got %q", cfg.DiskBackend)
	}
	if cfg.OutputFormat != "text" {
		t.Fatalf("expected default output format text, got %q", cfg.OutputFormat)
	}
	if cfg.ScoreWeights["cpu"] != DefaultCPUWeight || cfg.ScoreWeights["network"] != DefaultNetworkWeight {
		t.Fatalf("expected default score weights, got %#v", cfg.ScoreWeights)
	}
	if cfg.ScoreProfile != DefaultScoreProfile {
		t.Fatalf("expected default score profile %q, got %q", DefaultScoreProfile, cfg.ScoreProfile)
	}
	if cfg.EnableIPQuality {
		t.Fatal("expected default ip quality checks disabled")
	}
	if cfg.CPUBackend != "builtin" {
		t.Fatalf("expected default cpu backend builtin, got %q", cfg.CPUBackend)
	}
	if cfg.MemoryBackend != "builtin" {
		t.Fatalf("expected default memory backend builtin, got %q", cfg.MemoryBackend)
	}
	if len(cfg.Iperf3Servers) != 0 {
		t.Fatalf("expected default iperf3 servers empty, got %#v", cfg.Iperf3Servers)
	}
	if cfg.Iperf3ServerFile != "" {
		t.Fatalf("expected default iperf3 server file empty, got %q", cfg.Iperf3ServerFile)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected default config to validate, got %v", err)
	}
}

func TestValidateRejectsMissingScoreWeight(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ScoreWeights = map[string]float64{
		"cpu":    0.3,
		"memory": 0.2,
		"disk":   0.5,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected missing score weight to fail validation")
	}

	configErr, ok := err.(*ConfigError)
	if !ok {
		t.Fatalf("expected ConfigError, got %T", err)
	}
	if configErr.Field != "score_weights" {
		t.Fatalf("expected field score_weights, got %q", configErr.Field)
	}
}

func TestValidateRejectsNegativeScoreWeight(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ScoreWeights["cpu"] = -0.1

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected negative score weight to fail validation")
	}

	configErr, ok := err.(*ConfigError)
	if !ok {
		t.Fatalf("expected ConfigError, got %T", err)
	}
	if configErr.Field != "score_weights" {
		t.Fatalf("expected field score_weights, got %q", configErr.Field)
	}
}

func TestValidateRejectsUnknownScoreProfile(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ScoreProfile = "unknown"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected invalid score profile to fail validation")
	}

	configErr, ok := err.(*ConfigError)
	if !ok {
		t.Fatalf("expected ConfigError, got %T", err)
	}
	if configErr.Field != "score_profile" {
		t.Fatalf("expected field score_profile, got %q", configErr.Field)
	}
}

func TestValidateAcceptsKnownScoreProfiles(t *testing.T) {
	for _, profile := range []string{"vps", "server", "workstation"} {
		cfg := DefaultConfig()
		cfg.ScoreProfile = profile

		if err := cfg.Validate(); err != nil {
			t.Fatalf("expected profile %q to validate, got %v", profile, err)
		}
	}
}

func TestValidateRejectsUnknownOutputFormat(t *testing.T) {
	cfg := DefaultConfig()
	cfg.OutputFormat = "xml"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected invalid output format to fail validation")
	}

	configErr, ok := err.(*ConfigError)
	if !ok {
		t.Fatalf("expected ConfigError, got %T", err)
	}
	if configErr.Field != "output_format" {
		t.Fatalf("expected field output_format, got %q", configErr.Field)
	}
}

func TestValidateAcceptsJSONOutputFormat(t *testing.T) {
	cfg := DefaultConfig()
	cfg.OutputFormat = "json"

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected json output format to validate, got %v", err)
	}
}

func TestValidateRejectsUnknownNetworkBackend(t *testing.T) {
	cfg := DefaultConfig()
	cfg.NetworkBackend = "unknown"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected invalid network backend to fail validation")
	}

	configErr, ok := err.(*ConfigError)
	if !ok {
		t.Fatalf("expected ConfigError, got %T", err)
	}
	if configErr.Field != "network_backend" {
		t.Fatalf("expected field network_backend, got %q", configErr.Field)
	}
}

func TestValidateAcceptsSpeedtestNetworkBackend(t *testing.T) {
	cfg := DefaultConfig()
	cfg.NetworkBackend = "speedtest"

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected speedtest network backend to validate, got %v", err)
	}
}

func TestValidateAcceptsGeekbenchCPUBackend(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CPUBackend = "geekbench"

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected geekbench cpu backend to validate, got %v", err)
	}
}

func TestValidateRejectsUnknownCPUBackend(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CPUBackend = "unknown"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected invalid cpu backend to fail validation")
	}

	configErr, ok := err.(*ConfigError)
	if !ok {
		t.Fatalf("expected ConfigError, got %T", err)
	}
	if configErr.Field != "cpu_backend" {
		t.Fatalf("expected field cpu_backend, got %q", configErr.Field)
	}
}

func TestValidateRejectsUnknownMemoryBackend(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MemoryBackend = "unknown"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected invalid memory backend to fail validation")
	}

	configErr, ok := err.(*ConfigError)
	if !ok {
		t.Fatalf("expected ConfigError, got %T", err)
	}
	if configErr.Field != "memory_backend" {
		t.Fatalf("expected field memory_backend, got %q", configErr.Field)
	}
}

func TestValidateRejectsUnknownDiskBackend(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DiskBackend = "unknown"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected invalid disk backend to fail validation")
	}

	configErr, ok := err.(*ConfigError)
	if !ok {
		t.Fatalf("expected ConfigError, got %T", err)
	}
	if configErr.Field != "disk_backend" {
		t.Fatalf("expected field disk_backend, got %q", configErr.Field)
	}
}
