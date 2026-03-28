package docksmith

import (
	"testing"
)

func makeElysiaFramework() *Framework {
	return &Framework{
		Name:         "bun-elysia",
		BuildCommand: "bun install --frozen-lockfile",
		StartCommand: "bun run src/index.ts",
		Port:         3000,
		BunVersion:   "1",
	}
}

func mustPlanBun(t *testing.T, fw *Framework) *BuildPlan {
	t.Helper()
	plan, err := planBun(fw)
	if err != nil {
		t.Fatalf("planBun: %v", err)
	}
	return plan
}

func TestPlanBun_TwoStages(t *testing.T) {
	plan := mustPlanBun(t, makeElysiaFramework())
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

func TestPlanBun_BaseImages(t *testing.T) {
	plan := mustPlanBun(t, makeElysiaFramework())
	bunImg := ResolveDockerTag("bun", "1")
	if plan.Stages[0].From != bunImg {
		t.Errorf("build from: want %q, got %q", bunImg, plan.Stages[0].From)
	}
	if plan.Stages[1].From != bunImg {
		t.Errorf("runtime from: want %q, got %q", bunImg, plan.Stages[1].From)
	}
}

func TestPlanBun_RuntimeCopiesFromBuild(t *testing.T) {
	plan := mustPlanBun(t, makeElysiaFramework())
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

func TestPlanBun_NoBuildCommand_SkipsBuildRun(t *testing.T) {
	fw := &Framework{
		Name:         "bun",
		BuildCommand: "",
		StartCommand: "bun run index.ts",
		Port:         3000,
		BunVersion:   "1",
	}
	plan := mustPlanBun(t, fw)
	build := plan.Stages[0]
	for _, s := range build.Steps {
		if s.Type == StepRun && len(s.Args) == 1 && s.Args[0] == "" {
			t.Error("build stage: RUN step with empty command")
		}
	}
}

func TestPlanBun_Validate(t *testing.T) {
	plan := mustPlanBun(t, makeElysiaFramework())
	if err := plan.Validate(); err != nil {
		t.Errorf("Validate() error: %v", err)
	}
}

func TestPlanBun_Expose(t *testing.T) {
	plan := mustPlanBun(t, makeElysiaFramework())
	if plan.Expose != 3000 {
		t.Errorf("expose: want 3000, got %d", plan.Expose)
	}
}

func TestPlanBun_Framework(t *testing.T) {
	plan := mustPlanBun(t, makeElysiaFramework())
	if plan.Framework != "bun-elysia" {
		t.Errorf("framework: want %q, got %q", "bun-elysia", plan.Framework)
	}
}
