package plan

import (
	"strings"
	"testing"

	"github.com/permanu/docksmith/core"
)

func TestApplyContextRoot_RewritesCopyDot(t *testing.T) {
	plan := &core.BuildPlan{
		Framework: "nextjs",
		Expose:    3000,
		Stages: []core.Stage{
			{
				Name: "build",
				From: "node:22-alpine",
				Steps: []core.Step{
					{Type: core.StepWorkdir, Args: []string{"/app"}},
					{Type: core.StepCopy, Args: []string{".", "."}},
					{Type: core.StepRun, Args: []string{"npm run build"}},
				},
			},
		},
	}

	applyContextRoot(plan, "apps/frontend")

	copyStep := plan.Stages[0].Steps[1]
	if copyStep.Args[0] != "./apps/frontend" {
		t.Errorf("src = %q, want %q", copyStep.Args[0], "./apps/frontend")
	}
	if copyStep.Args[1] != "." {
		t.Errorf("dst = %q, want %q (unchanged)", copyStep.Args[1], ".")
	}
}

func TestApplyContextRoot_RewritesLockfileCopy(t *testing.T) {
	plan := &core.BuildPlan{
		Framework: "nextjs",
		Expose:    3000,
		Stages: []core.Stage{
			{
				Name: "deps",
				From: "node:22-alpine",
				Steps: []core.Step{
					{Type: core.StepWorkdir, Args: []string{"/app"}},
					{Type: core.StepCopy, Args: []string{"package.json", "package-lock.json*", "./"}},
				},
			},
		},
	}

	applyContextRoot(plan, "apps/frontend")

	copyStep := plan.Stages[0].Steps[1]
	if copyStep.Args[0] != "./apps/frontend/package.json" {
		t.Errorf("arg[0] = %q, want %q", copyStep.Args[0], "./apps/frontend/package.json")
	}
	if copyStep.Args[1] != "./apps/frontend/package-lock.json*" {
		t.Errorf("arg[1] = %q, want %q", copyStep.Args[1], "./apps/frontend/package-lock.json*")
	}
	// Destination unchanged.
	if copyStep.Args[2] != "./" {
		t.Errorf("dst = %q, want %q", copyStep.Args[2], "./")
	}
}

func TestApplyContextRoot_IgnoresAbsoluteSrc(t *testing.T) {
	plan := &core.BuildPlan{
		Framework: "go",
		Expose:    8080,
		Stages: []core.Stage{
			{
				Name: "builder",
				From: "golang:1.26-alpine",
				Steps: []core.Step{
					{Type: core.StepCopy, Args: []string{"/app/dist", "/usr/share/nginx/html"}},
				},
			},
		},
	}

	applyContextRoot(plan, "apps/api")

	copyStep := plan.Stages[0].Steps[0]
	if copyStep.Args[0] != "/app/dist" {
		t.Errorf("absolute src was rewritten: got %q", copyStep.Args[0])
	}
}

func TestApplyContextRoot_EmptySubdir_NoOp(t *testing.T) {
	plan := &core.BuildPlan{
		Framework: "go",
		Expose:    8080,
		Stages: []core.Stage{
			{
				Name: "builder",
				From: "golang:1.26-alpine",
				Steps: []core.Step{
					{Type: core.StepCopy, Args: []string{".", "."}},
				},
			},
		},
	}

	applyContextRoot(plan, "")

	if plan.Stages[0].Steps[0].Args[0] != "." {
		t.Errorf("empty subdir should not rewrite, got %q", plan.Stages[0].Steps[0].Args[0])
	}
}

func TestApplyContextRoot_LeavesNonCopySteps(t *testing.T) {
	plan := &core.BuildPlan{
		Framework: "go",
		Expose:    8080,
		Stages: []core.Stage{
			{
				Name: "builder",
				From: "golang:1.26-alpine",
				Steps: []core.Step{
					{Type: core.StepWorkdir, Args: []string{"/app"}},
					{Type: core.StepRun, Args: []string{"go build"}},
					{Type: core.StepEnv, Args: []string{"CGO_ENABLED", "0"}},
				},
			},
		},
	}

	applyContextRoot(plan, "apps/api")

	if plan.Stages[0].Steps[0].Args[0] != "/app" {
		t.Error("WORKDIR was unexpectedly modified")
	}
	if plan.Stages[0].Steps[1].Args[0] != "go build" {
		t.Error("RUN was unexpectedly modified")
	}
}

func TestApplyContextRoot_MultiStageNode(t *testing.T) {
	fw := &core.Framework{
		Name:           "nextjs",
		BuildCommand:   "npm run build",
		StartCommand:   "npm start",
		Port:           3000,
		NodeVersion:    "22",
		PackageManager: "npm",
	}
	plan, err := Plan(fw, WithContextRoot("apps/web"))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	// deps stage: lockfile copy should be prefixed
	deps := plan.Stages[0]
	for _, s := range deps.Steps {
		if s.Type == core.StepCopy {
			for i := 0; i < len(s.Args)-1; i++ {
				if !strings.HasPrefix(s.Args[i], "./apps/web/") && s.Args[i] != "./apps/web" {
					t.Errorf("deps COPY src %q missing apps/web prefix", s.Args[i])
				}
			}
		}
	}

	// build stage: "COPY . ." should become "COPY ./apps/web ."
	build := plan.Stages[1]
	for _, s := range build.Steps {
		if s.Type == core.StepCopy {
			if s.Args[0] != "./apps/web" {
				t.Errorf("build COPY src = %q, want %q", s.Args[0], "./apps/web")
			}
		}
	}
}

func TestWithContextRoot_BackwardCompat_NoContextRoot(t *testing.T) {
	fw := &core.Framework{
		Name:           "nextjs",
		BuildCommand:   "npm run build",
		StartCommand:   "npm start",
		Port:           3000,
		NodeVersion:    "22",
		PackageManager: "npm",
	}
	plan, err := Plan(fw)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	// build stage should still have "COPY . ." without prefix
	build := plan.Stages[1]
	for _, s := range build.Steps {
		if s.Type == core.StepCopy && len(s.Args) == 2 {
			if s.Args[0] != "." {
				t.Errorf("without context root, COPY src = %q, want %q", s.Args[0], ".")
			}
		}
	}
}
