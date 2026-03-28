package docksmith

import (
	"reflect"
	"testing"
)

func ptr[T any](v T) *T { return &v }

func TestWithUser_SetsPointer(t *testing.T) {
	cfg := resolvePlanConfig([]PlanOption{WithUser("appuser")})
	if cfg.user == nil {
		t.Fatal("user should be set")
	}
	if *cfg.user != "appuser" {
		t.Errorf("user = %q, want %q", *cfg.user, "appuser")
	}
}

func TestWithUser_EmptyString_DisablesUser(t *testing.T) {
	cfg := resolvePlanConfig([]PlanOption{WithUser("")})
	if cfg.user == nil {
		t.Fatal("user pointer should be non-nil (even for empty string)")
	}
	if *cfg.user != "" {
		t.Errorf("user = %q, want empty string", *cfg.user)
	}
}

func TestWithUser_NotSet_NilPointer(t *testing.T) {
	cfg := resolvePlanConfig(nil)
	if cfg.user != nil {
		t.Errorf("user should be nil when not set, got %q", *cfg.user)
	}
}

func TestWithHealthcheck_SetsPointer(t *testing.T) {
	cfg := resolvePlanConfig([]PlanOption{WithHealthcheck("curl -f http://localhost/health")})
	if cfg.healthcheck == nil {
		t.Fatal("healthcheck should be set")
	}
	if *cfg.healthcheck != "curl -f http://localhost/health" {
		t.Errorf("healthcheck = %q, want curl command", *cfg.healthcheck)
	}
}

func TestWithHealthcheckDisabled_SetsEmptyPointer(t *testing.T) {
	cfg := resolvePlanConfig([]PlanOption{WithHealthcheckDisabled()})
	if cfg.healthcheck == nil {
		t.Fatal("healthcheck pointer should be non-nil when disabled")
	}
	if *cfg.healthcheck != "" {
		t.Errorf("healthcheck = %q, want empty string", *cfg.healthcheck)
	}
}

func TestWithRuntimeImage_Overrides(t *testing.T) {
	cfg := resolvePlanConfig([]PlanOption{WithRuntimeImage("gcr.io/distroless/static:nonroot")})
	if cfg.runtimeImage == nil {
		t.Fatal("runtimeImage should be set")
	}
	if *cfg.runtimeImage != "gcr.io/distroless/static:nonroot" {
		t.Errorf("runtimeImage = %q, want distroless", *cfg.runtimeImage)
	}
}

func TestWithBaseImage_Overrides(t *testing.T) {
	cfg := resolvePlanConfig([]PlanOption{WithBaseImage("node:20-bookworm")})
	if cfg.baseImage == nil {
		t.Fatal("baseImage should be set")
	}
	if *cfg.baseImage != "node:20-bookworm" {
		t.Errorf("baseImage = %q, want node:20-bookworm", *cfg.baseImage)
	}
}

func TestWithEntrypoint_SetsSlice(t *testing.T) {
	cfg := resolvePlanConfig([]PlanOption{WithEntrypoint("/bin/sh", "-c")})
	if !reflect.DeepEqual(cfg.entrypoint, []string{"/bin/sh", "-c"}) {
		t.Errorf("entrypoint = %v, want [/bin/sh -c]", cfg.entrypoint)
	}
}

func TestWithEntrypoint_NotSet_NilSlice(t *testing.T) {
	cfg := resolvePlanConfig(nil)
	if cfg.entrypoint != nil {
		t.Errorf("entrypoint should be nil when not set, got %v", cfg.entrypoint)
	}
}

func TestWithExtraEnv_SetsMap(t *testing.T) {
	env := map[string]string{"FOO": "bar", "BAZ": "qux"}
	cfg := resolvePlanConfig([]PlanOption{WithExtraEnv(env)})
	if !reflect.DeepEqual(cfg.extraEnv, env) {
		t.Errorf("extraEnv = %v, want %v", cfg.extraEnv, env)
	}
}

func TestWithExpose_SetsPort(t *testing.T) {
	cfg := resolvePlanConfig([]PlanOption{WithExpose(9000)})
	if cfg.expose == nil {
		t.Fatal("expose should be set")
	}
	if *cfg.expose != 9000 {
		t.Errorf("expose = %d, want 9000", *cfg.expose)
	}
}

func TestWithInstallCommand_SetsPointer(t *testing.T) {
	cfg := resolvePlanConfig([]PlanOption{WithInstallCommand("npm ci --prefer-offline")})
	if cfg.installCmd == nil {
		t.Fatal("installCmd should be set")
	}
	if *cfg.installCmd != "npm ci --prefer-offline" {
		t.Errorf("installCmd = %q, want npm ci", *cfg.installCmd)
	}
}

func TestWithBuildCommand_SetsPointer(t *testing.T) {
	cfg := resolvePlanConfig([]PlanOption{WithBuildCommand("make build")})
	if cfg.buildCmd == nil {
		t.Fatal("buildCmd should be set")
	}
	if *cfg.buildCmd != "make build" {
		t.Errorf("buildCmd = %q, want make build", *cfg.buildCmd)
	}
}

func TestWithStartCommand_SetsPointer(t *testing.T) {
	cfg := resolvePlanConfig([]PlanOption{WithStartCommand("node dist/index.js")})
	if cfg.startCmd == nil {
		t.Fatal("startCmd should be set")
	}
	if *cfg.startCmd != "node dist/index.js" {
		t.Errorf("startCmd = %q, want node dist/index.js", *cfg.startCmd)
	}
}

func TestWithSystemDeps_SetsSlice(t *testing.T) {
	cfg := resolvePlanConfig([]PlanOption{WithSystemDeps("libpq-dev", "curl")})
	if !reflect.DeepEqual(cfg.systemDeps, []string{"libpq-dev", "curl"}) {
		t.Errorf("systemDeps = %v, want [libpq-dev curl]", cfg.systemDeps)
	}
}

func TestWithBuildCacheDisabled_SetsFlag(t *testing.T) {
	cfg := resolvePlanConfig([]PlanOption{WithBuildCacheDisabled()})
	if !cfg.noBuildCache {
		t.Error("noBuildCache should be true")
	}
}

func TestWithBuildCacheDisabled_DefaultFalse(t *testing.T) {
	cfg := resolvePlanConfig(nil)
	if cfg.noBuildCache {
		t.Error("noBuildCache should default to false")
	}
}

func TestResolvePlanConfig_MultipleOptions_LastWins(t *testing.T) {
	cfg := resolvePlanConfig([]PlanOption{
		WithUser("first"),
		WithUser("second"),
	})
	if cfg.user == nil || *cfg.user != "second" {
		t.Errorf("last WithUser should win, got %v", cfg.user)
	}
}

func TestResolvePlanConfig_Empty_AllNil(t *testing.T) {
	cfg := resolvePlanConfig([]PlanOption{})
	if cfg.user != nil || cfg.healthcheck != nil || cfg.runtimeImage != nil ||
		cfg.baseImage != nil || cfg.entrypoint != nil || cfg.extraEnv != nil ||
		cfg.expose != nil || cfg.installCmd != nil || cfg.buildCmd != nil ||
		cfg.startCmd != nil || cfg.systemDeps != nil || cfg.noBuildCache {
		t.Error("empty option slice should produce zero planConfig")
	}
}
