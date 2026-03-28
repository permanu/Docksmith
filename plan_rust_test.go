package docksmith

import (
	"strings"
	"testing"
)

func rustActixFramework() *Framework {
	return &Framework{
		Name:         "rust-actix",
		BuildCommand: "cargo build --release",
		StartCommand: "./target/release/app",
		Port:         8080,
	}
}

func mustPlanRust(t *testing.T, fw *Framework) *BuildPlan {
	t.Helper()
	plan, err := planRust(fw)
	if err != nil {
		t.Fatalf("planRust: %v", err)
	}
	return plan
}

func TestPlanRust_TwoStages(t *testing.T) {
	plan := mustPlanRust(t, rustActixFramework())
	if len(plan.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(plan.Stages))
	}
}

func TestPlanRust_StageNames(t *testing.T) {
	plan := mustPlanRust(t, rustActixFramework())
	if plan.Stages[0].Name != "builder" {
		t.Errorf("stage 0: got %q, want %q", plan.Stages[0].Name, "builder")
	}
	if plan.Stages[1].Name != "runtime" {
		t.Errorf("stage 1: got %q, want %q", plan.Stages[1].Name, "runtime")
	}
}

func TestPlanRust_BuilderUsesRustAlpine(t *testing.T) {
	plan := mustPlanRust(t, rustActixFramework())
	want := ResolveDockerTag("rust", "")
	if plan.Stages[0].From != want {
		t.Errorf("builder from: got %q, want %q", plan.Stages[0].From, want)
	}
}

func TestPlanRust_RuntimeUsesDistroless(t *testing.T) {
	plan := mustPlanRust(t, rustActixFramework())
	from := plan.Stages[1].From
	if from != "gcr.io/distroless/cc-debian12:nonroot" {
		t.Errorf("runtime should use distroless/cc nonroot, got %q", from)
	}
}

func TestPlanRust_RuntimeHasNonRootUser(t *testing.T) {
	plan := mustPlanRust(t, rustActixFramework())
	runtime := plan.Stages[1]
	for _, step := range runtime.Steps {
		if step.Type == StepUser && step.Args[0] == "nonroot" {
			return
		}
	}
	t.Error("runtime should have USER nonroot step")
}

func TestPlanRust_RuntimeNoHealthcheck(t *testing.T) {
	plan := mustPlanRust(t, rustActixFramework())
	runtime := plan.Stages[1]
	for _, step := range runtime.Steps {
		if step.Type == StepHealthcheck {
			t.Error("rust distroless runtime should not have HEALTHCHECK")
		}
	}
}

func TestPlanRust_BuilderRunsCargoRelease(t *testing.T) {
	plan := mustPlanRust(t, rustActixFramework())
	builder := plan.Stages[0]
	found := false
	for _, step := range builder.Steps {
		if step.Type == StepRun {
			for _, arg := range step.Args {
				if strings.Contains(arg, "cargo build --release") {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("builder should run cargo build --release")
	}
}

func TestPlanRust_BuilderCopiesCargoToml(t *testing.T) {
	plan := mustPlanRust(t, rustActixFramework())
	builder := plan.Stages[0]
	found := false
	for _, step := range builder.Steps {
		if step.Type == StepCopy {
			for _, arg := range step.Args {
				if strings.Contains(arg, "Cargo.toml") {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("builder should copy Cargo.toml")
	}
}

func TestPlanRust_BuilderHasCacheMount(t *testing.T) {
	plan := mustPlanRust(t, rustActixFramework())
	builder := plan.Stages[0]
	found := false
	for _, step := range builder.Steps {
		if step.Type == StepRun && step.CacheMount != nil {
			found = true
		}
	}
	if !found {
		t.Error("builder should have a cargo registry cache mount")
	}
}

func TestPlanRust_RuntimeCopiesFromBuilder(t *testing.T) {
	plan := mustPlanRust(t, rustActixFramework())
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

func TestPlanRust_RuntimeHasCmd(t *testing.T) {
	plan := mustPlanRust(t, rustActixFramework())
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

func TestPlanRust_ExposedPort(t *testing.T) {
	plan := mustPlanRust(t, rustActixFramework())
	if plan.Expose != 8080 {
		t.Errorf("expose: got %d, want 8080", plan.Expose)
	}
}

func TestPlanRust_DefaultPort(t *testing.T) {
	fw := &Framework{Name: "rust-axum", BuildCommand: "cargo build --release", StartCommand: "./app"}
	plan := mustPlanRust(t, fw)
	if plan.Expose != 8080 {
		t.Errorf("default port: got %d, want 8080", plan.Expose)
	}
}

func TestPlanRust_ValidatesOK(t *testing.T) {
	plan := mustPlanRust(t, rustActixFramework())
	if err := plan.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestPlanRust_FrameworkName(t *testing.T) {
	plan := mustPlanRust(t, rustActixFramework())
	if plan.Framework != "rust-actix" {
		t.Errorf("framework: got %q, want %q", plan.Framework, "rust-actix")
	}
}
