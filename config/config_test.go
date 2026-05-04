package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_MissingFile_ReturnsNilNil(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("want nil error, got: %v", err)
	}
	if cfg != nil {
		t.Errorf("want nil config for empty dir, got %+v", cfg)
	}
}

func TestLoad_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	content := "runtime: node\nstart:\n  command: node index.js\n"
	if err := os.WriteFile(filepath.Join(dir, "docksmith.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("got nil config, want non-nil")
	}
	if cfg.Runtime != "node" {
		t.Errorf("Runtime = %q, want %q", cfg.Runtime, "node")
	}
}

func TestLoadWithNames_CustomName(t *testing.T) {
	dir := t.TempDir()
	content := "runtime: ruby\nstart:\n  command: bundle exec puma\n"
	if err := os.WriteFile(filepath.Join(dir, "deploy.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadWithNames(dir, []string{"deploy.yaml"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("got nil config, want non-nil")
	}
	if cfg.Runtime != "ruby" {
		t.Errorf("Runtime = %q, want %q", cfg.Runtime, "ruby")
	}
}

func TestValidate_MissingRuntime(t *testing.T) {
	cfg := &Config{Start: StartConfig{Command: "node index.js"}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("want error for missing runtime, got nil")
	}
}

func TestHealthcheckOpts_ValidateOK(t *testing.T) {
	valid := []HealthcheckOpts{
		{},
		{Interval: "30s", Timeout: "3s", StartPeriod: "10s", Retries: 3},
		{Interval: "1m"},
		{Timeout: "500ms"},
		{Retries: 0},
		{Retries: 10},
	}
	for _, h := range valid {
		if err := h.validate(); err != nil {
			t.Errorf("validate(%+v) unexpected error: %v", h, err)
		}
	}
}

func TestHealthcheckOpts_ValidateErrors(t *testing.T) {
	cases := []struct {
		name string
		opts HealthcheckOpts
	}{
		{"bad interval", HealthcheckOpts{Interval: "30"}},
		{"bad timeout", HealthcheckOpts{Timeout: "xyz"}},
		{"bad start_period", HealthcheckOpts{StartPeriod: "10 s"}},
		{"retries too high", HealthcheckOpts{Retries: 11}},
		{"retries negative", HealthcheckOpts{Retries: -1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.opts.validate(); err == nil {
				t.Errorf("validate(%+v): expected error, got nil", tc.opts)
			}
		})
	}
}
