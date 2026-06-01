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
	if cfg.CPUBackend != "builtin" {
		t.Fatalf("expected default cpu backend builtin, got %q", cfg.CPUBackend)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected default config to validate, got %v", err)
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
