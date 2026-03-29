package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecrets_TOML_ParsesCorrectly(t *testing.T) {
	data := `
runtime = "node"
[start]
command = "node index.js"
[secrets]
[secrets.npm]
target = "/root/.npmrc"
[secrets.pip]
target = "/root/.pip/pip.conf"
env = "PIP_INDEX_URL"
[secrets.license]
env = "LICENSE_KEY"
`
	cfg, err := ParseConfig("docksmith.toml", []byte(data))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(cfg.Secrets) != 3 {
		t.Fatalf("want 3 secrets, got %d", len(cfg.Secrets))
	}
	npm := cfg.Secrets["npm"]
	if npm.Target != "/root/.npmrc" {
		t.Errorf("npm.Target = %q, want %q", npm.Target, "/root/.npmrc")
	}
	pip := cfg.Secrets["pip"]
	if pip.Env != "PIP_INDEX_URL" {
		t.Errorf("pip.Env = %q, want %q", pip.Env, "PIP_INDEX_URL")
	}
	lic := cfg.Secrets["license"]
	if lic.Env != "LICENSE_KEY" {
		t.Errorf("license.Env = %q, want %q", lic.Env, "LICENSE_KEY")
	}
	if lic.Target != "" {
		t.Errorf("license.Target = %q, want empty", lic.Target)
	}
}

func TestSecrets_YAML_ParsesCorrectly(t *testing.T) {
	data := `
runtime: node
start:
  command: node index.js
secrets:
  npm:
    target: /root/.npmrc
  custom_key:
    env: LICENSE_KEY
`
	cfg, err := ParseConfig("docksmith.yaml", []byte(data))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(cfg.Secrets) != 2 {
		t.Fatalf("want 2 secrets, got %d", len(cfg.Secrets))
	}
}

func TestSecrets_Empty_NoError(t *testing.T) {
	data := `
runtime = "node"
[start]
command = "node index.js"
`
	cfg, err := ParseConfig("docksmith.toml", []byte(data))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(cfg.Secrets) != 0 {
		t.Errorf("want 0 secrets, got %d", len(cfg.Secrets))
	}
}

func TestSecrets_PathTraversal_Rejected(t *testing.T) {
	data := `
runtime = "node"
[start]
command = "node index.js"
[secrets]
[secrets.evil]
target = "/root/../etc/passwd"
`
	cfg, err := ParseConfig("docksmith.toml", []byte(data))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	err = cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "..") {
		t.Errorf("error should mention '..', got: %v", err)
	}
}

func TestSecrets_NeitherTargetNorEnv_Rejected(t *testing.T) {
	data := `
runtime = "node"
[start]
command = "node index.js"
[secrets]
[secrets.empty_secret]
`
	cfg, err := ParseConfig("docksmith.toml", []byte(data))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	err = cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for secret with neither target nor env")
	}
	if !strings.Contains(err.Error(), "at least one") {
		t.Errorf("error should mention 'at least one', got: %v", err)
	}
}

func TestSecrets_Load_FromFile(t *testing.T) {
	dir := t.TempDir()
	content := `
runtime = "node"
[start]
command = "node index.js"
[secrets]
[secrets.npm]
target = "/root/.npmrc"
`
	if err := os.WriteFile(filepath.Join(dir, "docksmith.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg == nil {
		t.Fatal("got nil config")
	}
	if len(cfg.Secrets) != 1 {
		t.Fatalf("want 1 secret, got %d", len(cfg.Secrets))
	}
	if cfg.Secrets["npm"].Target != "/root/.npmrc" {
		t.Errorf("npm target = %q, want %q", cfg.Secrets["npm"].Target, "/root/.npmrc")
	}
}
