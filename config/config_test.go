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
