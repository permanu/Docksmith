package docksmith

import (
	"strings"
	"testing"
)

func staticFramework() *Framework {
	return &Framework{
		Name: "static",
	}
}

func mustPlanStatic(t *testing.T, fw *Framework) *BuildPlan {
	t.Helper()
	plan, err := planStatic(fw)
	if err != nil {
		t.Fatalf("planStatic: %v", err)
	}
	return plan
}

func TestPlanStatic_SingleStage(t *testing.T) {
	plan := mustPlanStatic(t, staticFramework())
	if len(plan.Stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(plan.Stages))
	}
}

func TestPlanStatic_StageName(t *testing.T) {
	plan := mustPlanStatic(t, staticFramework())
	if plan.Stages[0].Name != "runtime" {
		t.Errorf("stage name: got %q, want %q", plan.Stages[0].Name, "runtime")
	}
}

func TestPlanStatic_BaseImage(t *testing.T) {
	plan := mustPlanStatic(t, staticFramework())
	from := plan.Stages[0].From
	if !strings.HasPrefix(from, "nginx:") {
		t.Errorf("base image: got %q, want nginx:... prefix", from)
	}
}

func TestPlanStatic_CopiesOutputDir(t *testing.T) {
	plan := mustPlanStatic(t, staticFramework())
	stage := plan.Stages[0]
	found := false
	for _, step := range stage.Steps {
		if step.Type == StepCopy {
			for _, arg := range step.Args {
				if strings.Contains(arg, "/usr/share/nginx/html") {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("stage should copy files to /usr/share/nginx/html")
	}
}

func TestPlanStatic_CustomOutputDir(t *testing.T) {
	fw := &Framework{Name: "static", OutputDir: "dist"}
	plan := mustPlanStatic(t, fw)
	stage := plan.Stages[0]
	found := false
	for _, step := range stage.Steps {
		if step.Type == StepCopy {
			if len(step.Args) > 0 && step.Args[0] == "dist" {
				found = true
			}
		}
	}
	if !found {
		t.Error("stage should copy from OutputDir 'dist'")
	}
}

func TestPlanStatic_DefaultOutputDir(t *testing.T) {
	plan := mustPlanStatic(t, staticFramework())
	stage := plan.Stages[0]
	for _, step := range stage.Steps {
		if step.Type == StepCopy && len(step.Args) > 0 {
			if step.Args[0] != "." {
				t.Errorf("default output dir: got %q, want \".\"", step.Args[0])
			}
			return
		}
	}
	t.Error("no COPY step found")
}

func TestPlanStatic_HasCmd(t *testing.T) {
	plan := mustPlanStatic(t, staticFramework())
	stage := plan.Stages[0]
	found := false
	for _, step := range stage.Steps {
		if step.Type == StepCmd {
			for _, arg := range step.Args {
				if arg == "nginx" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("stage should have CMD [nginx ...]")
	}
}

func TestPlanStatic_ExposeIsZero(t *testing.T) {
	plan := mustPlanStatic(t, staticFramework())
	if plan.Expose != 0 {
		t.Errorf("expose: got %d, want 0 for static sites", plan.Expose)
	}
}

func TestPlanStatic_ValidatesOK(t *testing.T) {
	plan := mustPlanStatic(t, staticFramework())
	if err := plan.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestPlanStatic_FrameworkName(t *testing.T) {
	plan := mustPlanStatic(t, staticFramework())
	if plan.Framework != "static" {
		t.Errorf("framework: got %q, want %q", plan.Framework, "static")
	}
}

func TestPlanStatic_HasNginxCacheDirs(t *testing.T) {
	plan := mustPlanStatic(t, staticFramework())
	stage := plan.Stages[0]
	found := false
	for _, s := range stage.Steps {
		if s.Type == StepRun && len(s.Args) > 0 &&
			strings.Contains(s.Args[0], "/var/cache/nginx/client_temp") &&
			strings.Contains(s.Args[0], "chown") {
			found = true
		}
	}
	if !found {
		t.Error("static runtime should create and chown nginx cache dirs before USER nginx")
	}
}

func TestPlanStatic_HasNginxUser(t *testing.T) {
	plan := mustPlanStatic(t, staticFramework())
	stage := plan.Stages[0]
	for _, s := range stage.Steps {
		if s.Type == StepUser && s.Args[0] == "nginx" {
			return
		}
	}
	t.Error("static runtime should have USER nginx step")
}

func TestPlanStatic_HasHealthcheck(t *testing.T) {
	plan := mustPlanStatic(t, staticFramework())
	stage := plan.Stages[0]
	for _, s := range stage.Steps {
		if s.Type == StepHealthcheck {
			return
		}
	}
	t.Error("static runtime should have a HEALTHCHECK step")
}
