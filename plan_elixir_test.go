package docksmith

import (
	"strings"
	"testing"
)

func phoenixFramework() *Framework {
	return &Framework{
		Name:         "elixir-phoenix",
		BuildCommand: "mix deps.get && mix compile",
		StartCommand: "mix phx.server",
		Port:         4000,
	}
}

func mustPlanElixir(t *testing.T, fw *Framework) *BuildPlan {
	t.Helper()
	plan, err := planElixir(fw)
	if err != nil {
		t.Fatalf("planElixir: %v", err)
	}
	return plan
}

func TestPlanElixir_TwoStages(t *testing.T) {
	plan := mustPlanElixir(t, phoenixFramework())
	if len(plan.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(plan.Stages))
	}
}

func TestPlanElixir_StageNames(t *testing.T) {
	plan := mustPlanElixir(t, phoenixFramework())
	if plan.Stages[0].Name != "builder" {
		t.Errorf("stage 0: got %q, want %q", plan.Stages[0].Name, "builder")
	}
	if plan.Stages[1].Name != "runtime" {
		t.Errorf("stage 1: got %q, want %q", plan.Stages[1].Name, "runtime")
	}
}

func TestPlanElixir_BuilderUsesElixirAlpine(t *testing.T) {
	plan := mustPlanElixir(t, phoenixFramework())
	want := ResolveDockerTag("elixir", "")
	if plan.Stages[0].From != want {
		t.Errorf("builder from: got %q, want %q", plan.Stages[0].From, want)
	}
}

func TestPlanElixir_RuntimeUsesAlpine(t *testing.T) {
	plan := mustPlanElixir(t, phoenixFramework())
	from := plan.Stages[1].From
	if !strings.HasPrefix(from, "alpine:") {
		t.Errorf("runtime should use alpine, got %q", from)
	}
}

func TestPlanElixir_BuilderSetsMixEnvProd(t *testing.T) {
	plan := mustPlanElixir(t, phoenixFramework())
	builder := plan.Stages[0]
	found := false
	for _, step := range builder.Steps {
		if step.Type == StepEnv && len(step.Args) >= 2 &&
			step.Args[0] == "MIX_ENV" && step.Args[1] == "prod" {
			found = true
		}
	}
	if !found {
		t.Error("builder should set MIX_ENV=prod")
	}
}

func TestPlanElixir_BuilderFetchesDeps(t *testing.T) {
	plan := mustPlanElixir(t, phoenixFramework())
	builder := plan.Stages[0]
	found := false
	for _, step := range builder.Steps {
		if step.Type == StepRun {
			for _, arg := range step.Args {
				if strings.Contains(arg, "mix deps.get") {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("builder should run mix deps.get")
	}
}

func TestPlanElixir_BuilderHasCacheMount(t *testing.T) {
	plan := mustPlanElixir(t, phoenixFramework())
	builder := plan.Stages[0]
	found := false
	for _, step := range builder.Steps {
		if step.Type == StepRun && step.CacheMount != nil {
			found = true
		}
	}
	if !found {
		t.Error("builder should have a mix cache mount")
	}
}

func TestPlanElixir_BuilderRunsMixRelease(t *testing.T) {
	plan := mustPlanElixir(t, phoenixFramework())
	builder := plan.Stages[0]
	found := false
	for _, step := range builder.Steps {
		if step.Type == StepRun {
			for _, arg := range step.Args {
				if strings.Contains(arg, "mix release") {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("builder should run mix release")
	}
}

func TestPlanElixir_RuntimeCopiesRelease(t *testing.T) {
	plan := mustPlanElixir(t, phoenixFramework())
	runtime := plan.Stages[1]
	found := false
	for _, step := range runtime.Steps {
		if step.Type == StepCopyFrom && step.CopyFrom != nil && step.CopyFrom.Stage == "builder" {
			found = true
		}
	}
	if !found {
		t.Error("runtime should copy from builder stage")
	}
}

func TestPlanElixir_RuntimeSetsPortEnv(t *testing.T) {
	plan := mustPlanElixir(t, phoenixFramework())
	runtime := plan.Stages[1]
	found := false
	for _, step := range runtime.Steps {
		if step.Type == StepEnv && len(step.Args) >= 1 && step.Args[0] == "PORT" {
			found = true
		}
	}
	if !found {
		t.Error("runtime should set PORT env var")
	}
}

func TestPlanElixir_RuntimeHasCmd(t *testing.T) {
	plan := mustPlanElixir(t, phoenixFramework())
	runtime := plan.Stages[1]
	found := false
	for _, step := range runtime.Steps {
		if step.Type == StepCmd {
			found = true
		}
	}
	if !found {
		t.Error("runtime must have a CMD step")
	}
}

func TestPlanElixir_ExposedPort(t *testing.T) {
	plan := mustPlanElixir(t, phoenixFramework())
	if plan.Expose != 4000 {
		t.Errorf("expose: got %d, want 4000", plan.Expose)
	}
}

func TestPlanElixir_DefaultPort(t *testing.T) {
	fw := &Framework{Name: "elixir-phoenix", BuildCommand: "mix compile", StartCommand: "mix phx.server"}
	plan := mustPlanElixir(t, fw)
	if plan.Expose != 4000 {
		t.Errorf("default port: got %d, want 4000", plan.Expose)
	}
}

func TestPlanElixir_ValidatesOK(t *testing.T) {
	plan := mustPlanElixir(t, phoenixFramework())
	if err := plan.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestPlanElixir_FrameworkName(t *testing.T) {
	plan := mustPlanElixir(t, phoenixFramework())
	if plan.Framework != "elixir-phoenix" {
		t.Errorf("framework: got %q, want %q", plan.Framework, "elixir-phoenix")
	}
}
