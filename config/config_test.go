package config

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	testSHA256Amd64 = "c12b889e3349f0e5610aec32fe327e5a6911a0e472754a0c381c30c7c0630e88"
	testSHA256Arm64 = "b76308f558d50d006add507f3ab86afc1147644519dd327f7f5fac6d02d4f595"
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

func TestRuntimeConfig_SystemDeps_ParseTOML(t *testing.T) {
	data := []byte(`
runtime = "go"
[start]
command = "./server"
[runtime_config]
image = "alpine:3.21"
system_deps = ["ca-certificates", "tzdata"]
`)
	cfg, err := ParseConfig("docksmith.toml", data)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	want := []string{"ca-certificates", "tzdata"}
	if len(cfg.RuntimeConfig.SystemDeps) != len(want) {
		t.Fatalf("SystemDeps = %v, want %v", cfg.RuntimeConfig.SystemDeps, want)
	}
	for i, dep := range want {
		if cfg.RuntimeConfig.SystemDeps[i] != dep {
			t.Errorf("SystemDeps[%d] = %q, want %q", i, cfg.RuntimeConfig.SystemDeps[i], dep)
		}
	}
}

func TestRuntimeConfig_SystemDeps_ParseYAML(t *testing.T) {
	data := []byte("runtime: go\nstart:\n  command: ./server\nruntime_config:\n  image: alpine:3.21\n  system_deps:\n    - ca-certificates\n    - tzdata\n")
	cfg, err := ParseConfig("docksmith.yaml", data)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if len(cfg.RuntimeConfig.SystemDeps) != 2 || cfg.RuntimeConfig.SystemDeps[0] != "ca-certificates" {
		t.Errorf("SystemDeps = %v, want [ca-certificates tzdata]", cfg.RuntimeConfig.SystemDeps)
	}
}

func TestRuntimeConfig_SystemDeps_InvalidPackageName(t *testing.T) {
	data := []byte(`
runtime = "go"
[start]
command = "./server"
[runtime_config]
system_deps = ["ca-certificates; rm -rf /"]
`)
	_, err := ParseConfig("docksmith.toml", data)
	if err == nil {
		t.Fatal("expected error for invalid package name with shell metacharacter, got nil")
	}
}

// TestParseConfig_ExternalToolSHA256String verifies that a TOML config with a
// single-string sha256 decodes into ExternalTool.SHA256.
func TestParseConfig_ExternalToolSHA256String(t *testing.T) {
	data := []byte(`
runtime = "go"
[start]
command = "./app"
[[external_tools]]
name = "atlas"
url = "https://release.ariga.io/atlas/atlas-linux-${arch}-v1.2.0"
sha256 = "` + testSHA256Amd64 + `"
install_path = "/app/bin"
format = "binary"
`)
	cfg, err := ParseConfig("docksmith.toml", data)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if len(cfg.ExternalTools) != 1 {
		t.Fatalf("want 1 external tool, got %d", len(cfg.ExternalTools))
	}
	tool := cfg.ExternalTools[0]
	if tool.SHA256 != testSHA256Amd64 {
		t.Errorf("SHA256 = %q, want %q", tool.SHA256, testSHA256Amd64)
	}
	if len(tool.SHA256Map) != 0 {
		t.Errorf("SHA256Map should be empty for string sha256, got %v", tool.SHA256Map)
	}
}

// TestParseConfig_ExternalToolSHA256Map_TOML verifies that a TOML config with an
// inline table sha256 decodes into ExternalTool.SHA256Map.
func TestParseConfig_ExternalToolSHA256Map_TOML(t *testing.T) {
	data := []byte(`
runtime = "go"
[start]
command = "./app"
[[external_tools]]
name = "atlas"
url = "https://release.ariga.io/atlas/atlas-linux-${arch}-v1.2.0"
sha256 = { amd64 = "` + testSHA256Amd64 + `", arm64 = "` + testSHA256Arm64 + `" }
install_path = "/app/bin"
format = "binary"
`)
	cfg, err := ParseConfig("docksmith.toml", data)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if len(cfg.ExternalTools) != 1 {
		t.Fatalf("want 1 external tool, got %d", len(cfg.ExternalTools))
	}
	tool := cfg.ExternalTools[0]
	if tool.SHA256 != "" {
		t.Errorf("SHA256 should be empty for map sha256, got %q", tool.SHA256)
	}
	if tool.SHA256Map["amd64"] != testSHA256Amd64 {
		t.Errorf("SHA256Map[amd64] = %q, want %q", tool.SHA256Map["amd64"], testSHA256Amd64)
	}
	if tool.SHA256Map["arm64"] != testSHA256Arm64 {
		t.Errorf("SHA256Map[arm64] = %q, want %q", tool.SHA256Map["arm64"], testSHA256Arm64)
	}
}

// TestParseConfig_ExternalToolSHA256Map_YAML verifies YAML map-form sha256.
func TestParseConfig_ExternalToolSHA256Map_YAML(t *testing.T) {
	data := []byte(`
runtime: go
start:
  command: ./app
external_tools:
  - name: atlas
    url: "https://release.ariga.io/atlas/atlas-linux-${arch}-v1.2.0"
    sha256:
      amd64: "` + testSHA256Amd64 + `"
      arm64: "` + testSHA256Arm64 + `"
    install_path: /app/bin
    format: binary
`)
	cfg, err := ParseConfig("docksmith.yaml", data)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if len(cfg.ExternalTools) != 1 {
		t.Fatalf("want 1 external tool, got %d", len(cfg.ExternalTools))
	}
	tool := cfg.ExternalTools[0]
	if tool.SHA256 != "" {
		t.Errorf("SHA256 should be empty for map sha256, got %q", tool.SHA256)
	}
	if tool.SHA256Map["amd64"] != testSHA256Amd64 {
		t.Errorf("SHA256Map[amd64] = %q, want %q", tool.SHA256Map["amd64"], testSHA256Amd64)
	}
	if tool.SHA256Map["arm64"] != testSHA256Arm64 {
		t.Errorf("SHA256Map[arm64] = %q, want %q", tool.SHA256Map["arm64"], testSHA256Arm64)
	}
}
