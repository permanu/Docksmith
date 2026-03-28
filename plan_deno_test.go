package docksmith

import (
	"testing"
)

func makeFreshFramework() *Framework {
	return &Framework{
		Name:         "deno-fresh",
		StartCommand: "deno run -A main.ts",
		Port:         8000,
		DenoVersion:  "2",
	}
}

func mustPlanDeno(t *testing.T, fw *Framework) *BuildPlan {
	t.Helper()
	plan, err := planDeno(fw)
	if err != nil {
		t.Fatalf("planDeno: %v", err)
	}
	return plan
}

func TestPlanDeno_TwoStages(t *testing.T) {
	plan := mustPlanDeno(t, makeFreshFramework())
	if len(plan.Stages) != 2 {
		t.Fatalf("want 2 stages, got %d", len(plan.Stages))
	}
	if plan.Stages[0].Name != "build" {
		t.Errorf("stage 0: want %q, got %q", "build", plan.Stages[0].Name)
	}
	if plan.Stages[1].Name != "runtime" {
		t.Errorf("stage 1: want %q, got %q", "runtime", plan.Stages[1].Name)
	}
}

func TestPlanDeno_BaseImages(t *testing.T) {
	plan := mustPlanDeno(t, makeFreshFramework())
	denoImg := ResolveDockerTag("deno", "2")
	if plan.Stages[0].From != denoImg {
		t.Errorf("build from: want %q, got %q", denoImg, plan.Stages[0].From)
	}
	if plan.Stages[1].From != denoImg {
		t.Errorf("runtime from: want %q, got %q", denoImg, plan.Stages[1].From)
	}
}

func TestPlanDeno_BuildStage_HasCopy(t *testing.T) {
	plan := mustPlanDeno(t, makeFreshFramework())
	build := plan.Stages[0]
	var hasCopy bool
	for _, s := range build.Steps {
		if s.Type == StepCopy {
			hasCopy = true
			break
		}
	}
	if !hasCopy {
		t.Error("build stage: expected COPY step")
	}
}

func TestPlanDeno_RuntimeCopiesFromBuild(t *testing.T) {
	plan := mustPlanDeno(t, makeFreshFramework())
	runtime := plan.Stages[1]
	var found bool
	for _, s := range runtime.Steps {
		if s.Type == StepCopyFrom && s.CopyFrom != nil && s.CopyFrom.Stage == "build" {
			found = true
			break
		}
	}
	if !found {
		t.Error("runtime stage: expected COPY --from=build step")
	}
}

func TestPlanDeno_Validate(t *testing.T) {
	plan := mustPlanDeno(t, makeFreshFramework())
	if err := plan.Validate(); err != nil {
		t.Errorf("Validate() error: %v", err)
	}
}

func TestPlanDeno_Expose(t *testing.T) {
	plan := mustPlanDeno(t, makeFreshFramework())
	if plan.Expose != 8000 {
		t.Errorf("expose: want 8000, got %d", plan.Expose)
	}
}

func TestPlanDeno_Framework(t *testing.T) {
	plan := mustPlanDeno(t, makeFreshFramework())
	if plan.Framework != "deno-fresh" {
		t.Errorf("framework: want %q, got %q", "deno-fresh", plan.Framework)
	}
}

func TestPlanDeno_TaskEntrypoint_NoCacheOrFallback(t *testing.T) {
	fw := &Framework{
		Name:         "deno",
		StartCommand: "deno task start",
		Port:         8000,
		DenoVersion:  "2",
	}
	plan := mustPlanDeno(t, fw)
	if err := plan.Validate(); err != nil {
		t.Errorf("Validate() error: %v", err)
	}
}

func TestPlanDeno_CacheTarget_FileEntrypoint(t *testing.T) {
	if got := denoCacheTarget("deno run -A server.ts"); got != "server.ts" {
		t.Errorf("want server.ts, got %q", got)
	}
	if got := denoCacheTarget("deno task start"); got != "main.ts" {
		t.Errorf("task: want main.ts, got %q", got)
	}
	if got := denoCacheTarget("deno run -A main.tsx"); got != "main.tsx" {
		t.Errorf("want main.tsx, got %q", got)
	}
}

func TestPlanDeno_Runtime_HasDenoUser(t *testing.T) {
	plan := mustPlanDeno(t, makeFreshFramework())
	runtime := plan.Stages[1]
	for _, s := range runtime.Steps {
		if s.Type == StepUser && s.Args[0] == "deno" {
			return
		}
	}
	t.Error("deno runtime should have USER deno step")
}

func TestPlanDeno_Runtime_HasHealthcheck(t *testing.T) {
	plan := mustPlanDeno(t, makeFreshFramework())
	runtime := plan.Stages[1]
	for _, s := range runtime.Steps {
		if s.Type == StepHealthcheck {
			return
		}
	}
	t.Error("deno runtime should have a HEALTHCHECK step")
}
