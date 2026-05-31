package config

import "testing"

func TestDefaultConfigUsesBuiltinNetworkBackend(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.NetworkBackend != "builtin" {
		t.Fatalf("expected default network backend builtin, got %q", cfg.NetworkBackend)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected default config to validate, got %v", err)
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
