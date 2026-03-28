package plan

import (
	"reflect"
	"testing"
)

func ptr[T any](v T) *T { return &v }

func TestWithUser_SetsPointer(t *testing.T) {
	cfg := ResolvePlanConfig([]PlanOption{WithUser("appuser")})
	if cfg.User == nil {
		t.Fatal("user should be set")
	}
	if *cfg.User != "appuser" {
		t.Errorf("user = %q, want %q", *cfg.User, "appuser")
	}
}

func TestWithUser_EmptyString_DisablesUser(t *testing.T) {
	cfg := ResolvePlanConfig([]PlanOption{WithUser("")})
	if cfg.User == nil {
		t.Fatal("user pointer should be non-nil (even for empty string)")
	}
	if *cfg.User != "" {
		t.Errorf("user = %q, want empty string", *cfg.User)
	}
}

func TestWithUser_NotSet_NilPointer(t *testing.T) {
	cfg := ResolvePlanConfig(nil)
	if cfg.User != nil {
		t.Errorf("user should be nil when not set, got %q", *cfg.User)
	}
}

func TestWithHealthcheck_SetsPointer(t *testing.T) {
	cfg := ResolvePlanConfig([]PlanOption{WithHealthcheck("curl -f http://localhost/health")})
	if cfg.Healthcheck == nil {
		t.Fatal("healthcheck should be set")
	}
	if *cfg.Healthcheck != "curl -f http://localhost/health" {
		t.Errorf("healthcheck = %q, want curl command", *cfg.Healthcheck)
	}
}

func TestWithHealthcheckDisabled_SetsEmptyPointer(t *testing.T) {
	cfg := ResolvePlanConfig([]PlanOption{WithHealthcheckDisabled()})
	if cfg.Healthcheck == nil {
		t.Fatal("healthcheck pointer should be non-nil when disabled")
	}
	if *cfg.Healthcheck != "" {
		t.Errorf("healthcheck = %q, want empty string", *cfg.Healthcheck)
	}
}

func TestWithRuntimeImage_Overrides(t *testing.T) {
	cfg := ResolvePlanConfig([]PlanOption{WithRuntimeImage("gcr.io/distroless/static:nonroot")})
	if cfg.RuntimeImage == nil {
		t.Fatal("runtimeImage should be set")
	}
	if *cfg.RuntimeImage != "gcr.io/distroless/static:nonroot" {
		t.Errorf("runtimeImage = %q, want distroless", *cfg.RuntimeImage)
	}
}

func TestWithBaseImage_Overrides(t *testing.T) {
	cfg := ResolvePlanConfig([]PlanOption{WithBaseImage("node:20-bookworm")})
	if cfg.BaseImage == nil {
		t.Fatal("baseImage should be set")
	}
	if *cfg.BaseImage != "node:20-bookworm" {
		t.Errorf("baseImage = %q, want node:20-bookworm", *cfg.BaseImage)
	}
}

func TestWithEntrypoint_SetsSlice(t *testing.T) {
	cfg := ResolvePlanConfig([]PlanOption{WithEntrypoint("/bin/sh", "-c")})
	if !reflect.DeepEqual(cfg.Entrypoint, []string{"/bin/sh", "-c"}) {
		t.Errorf("entrypoint = %v, want [/bin/sh -c]", cfg.Entrypoint)
	}
}

func TestWithEntrypoint_NotSet_NilSlice(t *testing.T) {
	cfg := ResolvePlanConfig(nil)
	if cfg.Entrypoint != nil {
		t.Errorf("entrypoint should be nil when not set, got %v", cfg.Entrypoint)
	}
}

func TestWithExtraEnv_SetsMap(t *testing.T) {
	env := map[string]string{"FOO": "bar", "BAZ": "qux"}
	cfg := ResolvePlanConfig([]PlanOption{WithExtraEnv(env)})
	if !reflect.DeepEqual(cfg.ExtraEnv, env) {
		t.Errorf("extraEnv = %v, want %v", cfg.ExtraEnv, env)
	}
}

func TestWithExpose_SetsPort(t *testing.T) {
	cfg := ResolvePlanConfig([]PlanOption{WithExpose(9000)})
	if cfg.Expose == nil {
		t.Fatal("expose should be set")
	}
	if *cfg.Expose != 9000 {
		t.Errorf("expose = %d, want 9000", *cfg.Expose)
	}
}

func TestWithInstallCommand_SetsPointer(t *testing.T) {
	cfg := ResolvePlanConfig([]PlanOption{WithInstallCommand("npm ci --prefer-offline")})
	if cfg.InstallCmd == nil {
		t.Fatal("installCmd should be set")
	}
	if *cfg.InstallCmd != "npm ci --prefer-offline" {
		t.Errorf("installCmd = %q, want npm ci", *cfg.InstallCmd)
	}
}

func TestWithBuildCommand_SetsPointer(t *testing.T) {
	cfg := ResolvePlanConfig([]PlanOption{WithBuildCommand("make build")})
	if cfg.BuildCmd == nil {
		t.Fatal("buildCmd should be set")
	}
	if *cfg.BuildCmd != "make build" {
		t.Errorf("buildCmd = %q, want make build", *cfg.BuildCmd)
	}
}

func TestWithStartCommand_SetsPointer(t *testing.T) {
	cfg := ResolvePlanConfig([]PlanOption{WithStartCommand("node dist/index.js")})
	if cfg.StartCmd == nil {
		t.Fatal("startCmd should be set")
	}
	if *cfg.StartCmd != "node dist/index.js" {
		t.Errorf("startCmd = %q, want node dist/index.js", *cfg.StartCmd)
	}
}

func TestWithSystemDeps_SetsSlice(t *testing.T) {
	cfg := ResolvePlanConfig([]PlanOption{WithSystemDeps("libpq-dev", "curl")})
	if !reflect.DeepEqual(cfg.SystemDeps, []string{"libpq-dev", "curl"}) {
		t.Errorf("systemDeps = %v, want [libpq-dev curl]", cfg.SystemDeps)
	}
}

func TestWithBuildCacheDisabled_SetsFlag(t *testing.T) {
	cfg := ResolvePlanConfig([]PlanOption{WithBuildCacheDisabled()})
	if !cfg.NoBuildCache {
		t.Error("noBuildCache should be true")
	}
}

func TestWithBuildCacheDisabled_DefaultFalse(t *testing.T) {
	cfg := ResolvePlanConfig(nil)
	if cfg.NoBuildCache {
		t.Error("noBuildCache should default to false")
	}
}

func TestResolvePlanConfig_MultipleOptions_LastWins(t *testing.T) {
	cfg := ResolvePlanConfig([]PlanOption{
		WithUser("first"),
		WithUser("second"),
	})
	if cfg.User == nil || *cfg.User != "second" {
		t.Errorf("last WithUser should win, got %v", cfg.User)
	}
}

func TestResolvePlanConfig_Empty_AllNil(t *testing.T) {
	cfg := ResolvePlanConfig([]PlanOption{})
	if cfg.User != nil || cfg.Healthcheck != nil || cfg.RuntimeImage != nil ||
		cfg.BaseImage != nil || cfg.Entrypoint != nil || cfg.ExtraEnv != nil ||
		cfg.Expose != nil || cfg.InstallCmd != nil || cfg.BuildCmd != nil ||
		cfg.StartCmd != nil || cfg.SystemDeps != nil || cfg.NoBuildCache {
		t.Error("empty option slice should produce zero planConfig")
	}
}
