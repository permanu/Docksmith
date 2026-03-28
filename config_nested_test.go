package docksmith

import (
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// LoadConfig — nested TOML schema
// ---------------------------------------------------------------------------

func TestLoadConfig_Nested_TOML_BasicFields(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("testdata", "fixtures", "config-nested-toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("got nil config")
	}
	if cfg.Runtime != "go" {
		t.Errorf("Runtime = %q, want go", cfg.Runtime)
	}
	if cfg.Version != "1.22" {
		t.Errorf("Version = %q, want 1.22", cfg.Version)
	}
	if cfg.Build.Command != "go build -o server ." {
		t.Errorf("Build.Command = %q, want go build", cfg.Build.Command)
	}
	if cfg.Start.Command != "go run ." {
		t.Errorf("Start.Command = %q, want go run .", cfg.Start.Command)
	}
	if cfg.RuntimeConfig.Expose != 9090 {
		t.Errorf("RuntimeConfig.Expose = %d, want 9090", cfg.RuntimeConfig.Expose)
	}
}

func TestLoadConfig_Nested_TOML_UserString(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("testdata", "fixtures", "config-nested-toml-user"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	opts, err := cfg.ToPlanOptions()
	if err != nil {
		t.Fatalf("ToPlanOptions: %v", err)
	}
	resolved := resolvePlanConfig(opts)
	if resolved.user == nil {
		t.Fatal("user should be set")
	}
	if *resolved.user != "appuser" {
		t.Errorf("user = %q, want appuser", *resolved.user)
	}
}

func TestLoadConfig_Nested_TOML_UserFalse_DisablesUser(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("testdata", "fixtures", "config-nested-toml-user-false"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	opts, err := cfg.ToPlanOptions()
	if err != nil {
		t.Fatalf("ToPlanOptions: %v", err)
	}
	resolved := resolvePlanConfig(opts)
	if resolved.user == nil {
		t.Fatal("user pointer should be non-nil when set to false")
	}
	if *resolved.user != "" {
		t.Errorf("user = %q, want empty string (disabled)", *resolved.user)
	}
}

func TestLoadConfig_Nested_TOML_HealthcheckDisabled(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("testdata", "fixtures", "config-nested-toml-hc-false"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	opts, err := cfg.ToPlanOptions()
	if err != nil {
		t.Fatalf("ToPlanOptions: %v", err)
	}
	resolved := resolvePlanConfig(opts)
	if resolved.healthcheck == nil {
		t.Fatal("healthcheck pointer should be non-nil when set to false")
	}
	if *resolved.healthcheck != "" {
		t.Errorf("healthcheck = %q, want empty string (disabled)", *resolved.healthcheck)
	}
}

func TestLoadConfig_Nested_TOML_HealthcheckString(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("testdata", "fixtures", "config-nested-toml-hc-cmd"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	opts, err := cfg.ToPlanOptions()
	if err != nil {
		t.Fatalf("ToPlanOptions: %v", err)
	}
	resolved := resolvePlanConfig(opts)
	if resolved.healthcheck == nil {
		t.Fatal("healthcheck should be set")
	}
	if *resolved.healthcheck != "curl -f http://localhost:8080/health" {
		t.Errorf("healthcheck = %q, want curl command", *resolved.healthcheck)
	}
}

// ---------------------------------------------------------------------------
// LoadPlanOptions — convenience loader
// ---------------------------------------------------------------------------

func TestLoadPlanOptions_ReturnsOptions(t *testing.T) {
	opts, err := LoadPlanOptions(filepath.Join("testdata", "fixtures", "config-nested-toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resolved := resolvePlanConfig(opts)
	if resolved.expose == nil {
		t.Fatal("expose should be set from nested config")
	}
	if *resolved.expose != 9090 {
		t.Errorf("expose = %d, want 9090", *resolved.expose)
	}
}

func TestLoadPlanOptions_NoConfigFile_ReturnsEmptySlice(t *testing.T) {
	dir := t.TempDir()
	opts, err := LoadPlanOptions(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(opts) != 0 {
		t.Errorf("want empty opts when no config, got %d", len(opts))
	}
}

// ---------------------------------------------------------------------------
// ToPlanOptions — field mapping
// ---------------------------------------------------------------------------

func TestToPlanOptions_BuildCommand(t *testing.T) {
	cfg := &Config{
		Runtime: "go",
		Build:   BuildConfig{Command: "make build"},
		Start:   StartConfig{Command: "./server"},
	}
	opts, err := cfg.ToPlanOptions()
	if err != nil {
		t.Fatalf("ToPlanOptions: %v", err)
	}
	resolved := resolvePlanConfig(opts)
	if resolved.buildCmd == nil || *resolved.buildCmd != "make build" {
		t.Errorf("buildCmd = %v, want make build", resolved.buildCmd)
	}
}

func TestToPlanOptions_StartCommand(t *testing.T) {
	cfg := &Config{
		Runtime: "node",
		Start:   StartConfig{Command: "node dist/index.js"},
	}
	opts, err := cfg.ToPlanOptions()
	if err != nil {
		t.Fatalf("ToPlanOptions: %v", err)
	}
	resolved := resolvePlanConfig(opts)
	if resolved.startCmd == nil || *resolved.startCmd != "node dist/index.js" {
		t.Errorf("startCmd = %v, want node dist/index.js", resolved.startCmd)
	}
}

func TestToPlanOptions_ExtraEnv(t *testing.T) {
	cfg := &Config{
		Runtime: "python",
		Start:   StartConfig{Command: "gunicorn app:app"},
		Env:     map[string]string{"PORT": "8000"},
	}
	opts, err := cfg.ToPlanOptions()
	if err != nil {
		t.Fatalf("ToPlanOptions: %v", err)
	}
	resolved := resolvePlanConfig(opts)
	if resolved.extraEnv == nil || resolved.extraEnv["PORT"] != "8000" {
		t.Errorf("extraEnv = %v, want PORT=8000", resolved.extraEnv)
	}
}

func TestToPlanOptions_SystemDeps(t *testing.T) {
	cfg := &Config{
		Runtime: "python",
		Start:   StartConfig{Command: "gunicorn app:app"},
		Install: InstallConfig{SystemDeps: []string{"libpq-dev", "curl"}},
	}
	opts, err := cfg.ToPlanOptions()
	if err != nil {
		t.Fatalf("ToPlanOptions: %v", err)
	}
	resolved := resolvePlanConfig(opts)
	if len(resolved.systemDeps) != 2 || resolved.systemDeps[0] != "libpq-dev" {
		t.Errorf("systemDeps = %v, want [libpq-dev curl]", resolved.systemDeps)
	}
}

func TestToPlanOptions_NoBuildCache(t *testing.T) {
	cfg := &Config{
		Runtime: "go",
		Start:   StartConfig{Command: "./server"},
		Build:   BuildConfig{NoCache: true},
	}
	opts, err := cfg.ToPlanOptions()
	if err != nil {
		t.Fatalf("ToPlanOptions: %v", err)
	}
	resolved := resolvePlanConfig(opts)
	if !resolved.noBuildCache {
		t.Error("noBuildCache should be true")
	}
}

func TestToPlanOptions_RuntimeImage(t *testing.T) {
	cfg := &Config{
		Runtime: "go",
		Start:   StartConfig{Command: "./server"},
		RuntimeConfig: RuntimeCfg{
			Image: "gcr.io/distroless/static:nonroot",
		},
	}
	opts, err := cfg.ToPlanOptions()
	if err != nil {
		t.Fatalf("ToPlanOptions: %v", err)
	}
	resolved := resolvePlanConfig(opts)
	if resolved.runtimeImage == nil || *resolved.runtimeImage != "gcr.io/distroless/static:nonroot" {
		t.Errorf("runtimeImage = %v, want distroless", resolved.runtimeImage)
	}
}

// ---------------------------------------------------------------------------
// Unknown key detection
// ---------------------------------------------------------------------------

func TestLoadConfig_Nested_UnknownKey_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, dir+"/docksmith.toml", "runtime = \"go\"\nstart_cmd = \"./server\"\n[start]\ncommand = \"./server\"\n")
	_, err := LoadConfig(dir)
	if err == nil {
		t.Fatal("want error for unknown key start_cmd")
	}
}

func TestLoadConfig_NestedYAML_Parses(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("testdata", "fixtures", "config-yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil || cfg.Runtime != "node" {
		t.Errorf("nested yaml should parse, got %v", cfg)
	}
}

func TestLoadConfig_NestedJSON_Parses(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("testdata", "fixtures", "config-json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil || cfg.Runtime != "python" {
		t.Errorf("nested json should parse, got %v", cfg)
	}
}
