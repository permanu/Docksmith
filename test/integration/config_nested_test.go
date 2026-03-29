package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/permanu/docksmith"
	"github.com/permanu/docksmith/plan"
)

// ---------------------------------------------------------------------------
// LoadConfig — nested TOML schema
// ---------------------------------------------------------------------------

func TestLoadConfig_Nested_TOML_BasicFields(t *testing.T) {
	cfg, err := docksmith.LoadConfig(filepath.Join("../../testdata", "fixtures", "config-nested-toml"))
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
	cfg, err := docksmith.LoadConfig(filepath.Join("../../testdata", "fixtures", "config-nested-toml-user"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	opts, err := docksmith.ConfigToPlanOptions(cfg)
	if err != nil {
		t.Fatalf("ToPlanOptions: %v", err)
	}
	resolved := plan.ResolvePlanConfig(opts)
	if resolved.User == nil {
		t.Fatal("user should be set")
	}
	if *resolved.User != "appuser" {
		t.Errorf("user = %q, want appuser", *resolved.User)
	}
}

func TestLoadConfig_Nested_TOML_UserFalse_DisablesUser(t *testing.T) {
	cfg, err := docksmith.LoadConfig(filepath.Join("../../testdata", "fixtures", "config-nested-toml-user-false"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	opts, err := docksmith.ConfigToPlanOptions(cfg)
	if err != nil {
		t.Fatalf("ToPlanOptions: %v", err)
	}
	resolved := plan.ResolvePlanConfig(opts)
	if resolved.User == nil {
		t.Fatal("user pointer should be non-nil when set to false")
	}
	if *resolved.User != "" {
		t.Errorf("user = %q, want empty string (disabled)", *resolved.User)
	}
}

func TestLoadConfig_Nested_TOML_HealthcheckDisabled(t *testing.T) {
	cfg, err := docksmith.LoadConfig(filepath.Join("../../testdata", "fixtures", "config-nested-toml-hc-false"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	opts, err := docksmith.ConfigToPlanOptions(cfg)
	if err != nil {
		t.Fatalf("ToPlanOptions: %v", err)
	}
	resolved := plan.ResolvePlanConfig(opts)
	if resolved.Healthcheck == nil {
		t.Fatal("healthcheck pointer should be non-nil when set to false")
	}
	if *resolved.Healthcheck != "" {
		t.Errorf("healthcheck = %q, want empty string (disabled)", *resolved.Healthcheck)
	}
}

func TestLoadConfig_Nested_TOML_HealthcheckString(t *testing.T) {
	cfg, err := docksmith.LoadConfig(filepath.Join("../../testdata", "fixtures", "config-nested-toml-hc-cmd"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	opts, err := docksmith.ConfigToPlanOptions(cfg)
	if err != nil {
		t.Fatalf("ToPlanOptions: %v", err)
	}
	resolved := plan.ResolvePlanConfig(opts)
	if resolved.Healthcheck == nil {
		t.Fatal("healthcheck should be set")
	}
	if *resolved.Healthcheck != "curl -f http://localhost:8080/health" {
		t.Errorf("healthcheck = %q, want curl command", *resolved.Healthcheck)
	}
}

// ---------------------------------------------------------------------------
// LoadPlanOptions
// ---------------------------------------------------------------------------

func TestLoadPlanOptions_ReturnsOptions(t *testing.T) {
	opts, err := docksmith.LoadPlanOptions(filepath.Join("../../testdata", "fixtures", "config-nested-toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resolved := plan.ResolvePlanConfig(opts)
	if resolved.Expose == nil {
		t.Fatal("expose should be set from nested config")
	}
	if *resolved.Expose != 9090 {
		t.Errorf("expose = %d, want 9090", *resolved.Expose)
	}
}

func TestLoadPlanOptions_NoConfigFile_ReturnsEmptySlice(t *testing.T) {
	dir := t.TempDir()
	opts, err := docksmith.LoadPlanOptions(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(opts) != 0 {
		t.Errorf("want empty opts when no config, got %d", len(opts))
	}
}

// ---------------------------------------------------------------------------
// ToPlanOptions
// ---------------------------------------------------------------------------

func TestToPlanOptions_BuildCommand(t *testing.T) {
	cfg := &docksmith.Config{
		Runtime: "go",
		Build:   docksmith.BuildConfig{Command: "make build"},
		Start:   docksmith.StartConfig{Command: "./server"},
	}
	opts, err := docksmith.ConfigToPlanOptions(cfg)
	if err != nil {
		t.Fatalf("ToPlanOptions: %v", err)
	}
	resolved := plan.ResolvePlanConfig(opts)
	if resolved.BuildCmd == nil || *resolved.BuildCmd != "make build" {
		t.Errorf("buildCmd = %v, want make build", resolved.BuildCmd)
	}
}

func TestToPlanOptions_StartCommand(t *testing.T) {
	cfg := &docksmith.Config{
		Runtime: "node",
		Start:   docksmith.StartConfig{Command: "node dist/index.js"},
	}
	opts, err := docksmith.ConfigToPlanOptions(cfg)
	if err != nil {
		t.Fatalf("ToPlanOptions: %v", err)
	}
	resolved := plan.ResolvePlanConfig(opts)
	if resolved.StartCmd == nil || *resolved.StartCmd != "node dist/index.js" {
		t.Errorf("startCmd = %v, want node dist/index.js", resolved.StartCmd)
	}
}

func TestToPlanOptions_ExtraEnv(t *testing.T) {
	cfg := &docksmith.Config{
		Runtime: "python",
		Start:   docksmith.StartConfig{Command: "gunicorn app:app"},
		Env:     map[string]string{"PORT": "8000"},
	}
	opts, err := docksmith.ConfigToPlanOptions(cfg)
	if err != nil {
		t.Fatalf("ToPlanOptions: %v", err)
	}
	resolved := plan.ResolvePlanConfig(opts)
	if resolved.ExtraEnv == nil || resolved.ExtraEnv["PORT"] != "8000" {
		t.Errorf("extraEnv = %v, want PORT=8000", resolved.ExtraEnv)
	}
}

func TestToPlanOptions_SystemDeps(t *testing.T) {
	cfg := &docksmith.Config{
		Runtime: "python",
		Start:   docksmith.StartConfig{Command: "gunicorn app:app"},
		Install: docksmith.InstallConfig{SystemDeps: []string{"libpq-dev", "curl"}},
	}
	opts, err := docksmith.ConfigToPlanOptions(cfg)
	if err != nil {
		t.Fatalf("ToPlanOptions: %v", err)
	}
	resolved := plan.ResolvePlanConfig(opts)
	if len(resolved.SystemDeps) != 2 || resolved.SystemDeps[0] != "libpq-dev" {
		t.Errorf("systemDeps = %v, want [libpq-dev curl]", resolved.SystemDeps)
	}
}

func TestToPlanOptions_NoBuildCache(t *testing.T) {
	cfg := &docksmith.Config{
		Runtime: "go", Start: docksmith.StartConfig{Command: "./server"},
		Build: docksmith.BuildConfig{NoCache: true},
	}
	opts, err := docksmith.ConfigToPlanOptions(cfg)
	if err != nil {
		t.Fatalf("ToPlanOptions: %v", err)
	}
	resolved := plan.ResolvePlanConfig(opts)
	if !resolved.NoBuildCache {
		t.Error("noBuildCache should be true")
	}
}

func TestToPlanOptions_RuntimeImage(t *testing.T) {
	cfg := &docksmith.Config{
		Runtime: "go", Start: docksmith.StartConfig{Command: "./server"},
		RuntimeConfig: docksmith.RuntimeCfg{Image: "gcr.io/distroless/static:nonroot"},
	}
	opts, err := docksmith.ConfigToPlanOptions(cfg)
	if err != nil {
		t.Fatalf("ToPlanOptions: %v", err)
	}
	resolved := plan.ResolvePlanConfig(opts)
	if resolved.RuntimeImage == nil || *resolved.RuntimeImage != "gcr.io/distroless/static:nonroot" {
		t.Errorf("runtimeImage = %v, want distroless", resolved.RuntimeImage)
	}
}

// ---------------------------------------------------------------------------
// Unknown key detection
// ---------------------------------------------------------------------------

func TestLoadConfig_Nested_UnknownKey_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, dir+"/docksmith.toml", "runtime = \"go\"\nstart_cmd = \"./server\"\n[start]\ncommand = \"./server\"\n")
	_, err := docksmith.LoadConfig(dir)
	if err == nil {
		t.Fatal("want error for unknown key start_cmd")
	}
}

func TestLoadConfig_NestedYAML_Parses(t *testing.T) {
	cfg, err := docksmith.LoadConfig(filepath.Join("../../testdata", "fixtures", "config-yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil || cfg.Runtime != "node" {
		t.Errorf("nested yaml should parse, got %v", cfg)
	}
}

func TestLoadConfig_NestedJSON_Parses(t *testing.T) {
	cfg, err := docksmith.LoadConfig(filepath.Join("../../testdata", "fixtures", "config-json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil || cfg.Runtime != "python" {
		t.Errorf("nested json should parse, got %v", cfg)
	}
}
